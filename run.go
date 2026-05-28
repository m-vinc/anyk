package main

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
)

type endpointResult struct {
	healthy bool
	err     error
}

func Run(ctx context.Context, asn int, services []AnykService) error {
	log := WithLogger(ctx)

	router := fmt.Sprintf("%d", asn)

	for _, svc := range services {
		if len(svc.AnycastIPs) == 0 {
			continue
		}

		anycastNets := []*Net{}
		afis := map[string]bool{}

		for _, cidr := range svc.AnycastIPs {
			_, ipnet, err := net.ParseCIDR(cidr)
			if err != nil {
				return fmt.Errorf("anycast_ip %q: %w", cidr, err)
			}
			afi := "ipv4"
			if strings.Contains(cidr, ":") {
				afi = "ipv6"
			}
			afis[afi] = true
			anycastNets = append(anycastNets, &Net{IPNet: ipnet, Afi: afi, CidrString: ipnet.String()})
		}

		results := make([]endpointResult, len(svc.Endpoints))
		if svc.Active {
			log.Debug().Str("service", svc.Name).Msg("started checking service")
			var wg sync.WaitGroup
			for i := range svc.Endpoints {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					results[i].healthy, results[i].err = healthcheck(ctx, &svc.Endpoints[i])
				}(i)
			}
			wg.Wait()
		}

		routeActions := map[string][]*routeAction{}
		announceActions := map[string]*announceAction{}
		serviceHealthy := false

		for i := range svc.Endpoints {
			ep := &svc.Endpoints[i]
			if ep.Distance <= 0 {
				ep.Distance = 1
			}
			afi := "ipv4"
			if strings.Contains(ep.IP, ":") {
				afi = "ipv6"
			}

			healthy := false
			if svc.Active {
				if results[i].err != nil {
					log.Warn().Str("service", svc.Name).Str("endpoint", ep.IP).Err(results[i].err).Msg("healthcheck error")
				}
				healthy = results[i].healthy
				if healthy {
					serviceHealthy = true
				}
			}

			log.Debug().Str("service", svc.Name).Str("endpoint", ep.IP).Bool("healthy", healthy).Msg("healthcheck")

			epIP := &IP{IP: net.ParseIP(ep.IP), IPString: ep.IP, Afi: afi}

			for _, anet := range anycastNets {
				if anet.Afi != afi {
					continue
				}

				if _, ok := routeActions[anet.CidrString]; !ok {
					routeActions[anet.CidrString] = nil
				}

				routeActions[anet.CidrString] = append(routeActions[anet.CidrString], &routeAction{
					remove:   !healthy,
					distance: ep.Distance,
					afi:      afi,
					prefix:   anet,
					via:      epIP,
				})

				if _, ok := announceActions[anet.CidrString]; !ok {
					announceActions[anet.CidrString] = &announceAction{router: router, remove: true, afi: afi, prefix: anet}
				}

				if healthy {
					announceActions[anet.CidrString].remove = false
				}
			}
		}

		routeCache := map[string]vtyshRoutes{}
		soCache := map[string]vtyshRoutes{}
		for afi := range afis {
			routes, err := vtyshShowRoutes(ctx, afi, nil)
			if err != nil {
				return err
			}
			routeCache[afi] = routes

			soRoutes, err := vtyshShowSelfOriginate(ctx, afi, nil)
			if err != nil {
				return err
			}
			soCache[afi] = soRoutes
		}

		var actions []vtyshAction

		for _, ras := range routeActions {
			if len(ras) == 0 {
				continue
			}

			routes := routeCache[ras[0].afi]

			for _, ra := range ras {
				exist := false
				for _, r := range routes[ra.prefix.CidrString] {
					for _, nh := range r.Nexthops {
						if nh.Afi != ra.afi {
							continue
						}

						if nh.Ip == ra.via.IPString {
							if r.Distance != ra.distance && !ra.remove {
								nhIP := &IP{IP: net.ParseIP(nh.Ip), IPString: nh.Ip, Afi: nh.Afi}
								actions = append(actions, &routeAction{remove: true, afi: ra.afi, prefix: ra.prefix, via: nhIP, distance: r.Distance})
								continue
							}
							exist = true
						}
					}
				}

				if exist == ra.remove {
					actions = append(actions, ra)
				}
			}
		}

		for _, aa := range announceActions {
			announced := false
			for prefix := range soCache[aa.afi] {
				if prefix == aa.prefix.CidrString {
					announced = true
					break
				}
			}

			if announced && aa.remove || !announced && !aa.remove {
				actions = append(actions, aa)
			}
		}

		if err := vtyshExecute(ctx, actions); err != nil {
			return fmt.Errorf("service %s: %w", svc.Name, err)
		}

		for _, callback := range svc.Callbacks {
			if callback.Healthy != serviceHealthy || len(callback.Command) == 0 {
				continue
			}

			log.Debug().
				Str("service", svc.Name).
				Strs("command", callback.Command).
				Msgf("executing callback")

			cmd := exec.CommandContext(ctx, callback.Command[0], callback.Command[1:]...)
			out, err := cmd.Output()
			if err != nil {
				log.Info().Str("service", svc.Name).Err(err).Str("stdout", string(out)).Msg("error while executing callback")
			}
		}

		log.Info().Str("service", svc.Name).Int("actions", len(actions)).Msg("anycast: applied")
	}

	return nil
}
