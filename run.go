package main

import (
	"context"
	"fmt"
	"github.com/goccy/go-yaml"
	"log"
	"net"
	"os"
	"strings"
)

type RunCmd struct {
	Config string `arg:"" name:"config" type:"path" help:"Path to configuration file." required:""`
}

func (r *RunCmd) Run() error {
	content, err := os.ReadFile(r.Config)
	if err != nil {
		return fmt.Errorf("cannot read config file: %w", err)
	}

	cfg := &AnykConfig{}
	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		return fmt.Errorf("cannot unmarshal config: %w", err)
	}

	log.Println("Running with config:", r.Config, cfg)
	for _, svc := range cfg.Services {
		if len(svc.AnycastIPs) == 0 {
			continue
		}

		anycastNets := []*AnykNet{}
		for _, ip := range svc.AnycastIPs {
			var ipnet *net.IPNet
			sip := ip.String()
			afi := "ipv4"

			if strings.Contains(sip, ":") {
				afi = "ipv6"
				_, ipnet, err = net.ParseCIDR(sip + "/128")
				if err != nil {
					return err
				}
			} else {
				_, ipnet, err = net.ParseCIDR(sip + "/32")
				if err != nil {
					return err
				}
			}
			anycastNets = append(anycastNets, &AnykNet{IPNet: ipnet, Afi: afi, CidrString: ipnet.String()})
		}

		// Healthchecks and stack every endpoint we need to handle
		routeActions := map[string][]*VtyshRouteAction{}
		announceActions := map[string]*VtyshAnnounceAction{}
		routeTargets := []*AnykIP{}

		for _, endpoint := range svc.Endpoints {
			if endpoint.Distance <= 0 {
				endpoint.Distance = 1
			}

			endpoint.Afi = "ipv4"
			if strings.Contains(endpoint.IP.String(), ":") {
				endpoint.Afi = "ipv6"
			}

			var health bool

			if svc.Active {
				health, err = healthcheck(context.Background(), endpoint)
				if !health || err != nil {
					log.Printf("healthcheck failed: %+v %+v", health, err)
				}
			}

			for _, aip := range anycastNets {
				if aip.Afi != endpoint.Afi {
					continue
				}

				if _, ok := routeActions[aip.CidrString]; !ok {
					routeActions[aip.CidrString] = []*VtyshRouteAction{}
				}

				ip := &AnykIP{IP: endpoint.IP, IPString: endpoint.IP.String(), Afi: endpoint.Afi}
				routeActions[aip.CidrString] = append(routeActions[aip.CidrString], &VtyshRouteAction{
					Remove:   !health,
					Distance: endpoint.Distance,
					Afi:      endpoint.Afi,
					Prefix:   aip,
					Via:      ip,
				})
				routeTargets = append(routeTargets, ip)

				if _, ok := announceActions[aip.CidrString]; !ok {
					announceActions[aip.CidrString] = &VtyshAnnounceAction{Router: cfg.Router, Remove: !health, Afi: endpoint.Afi, Prefix: aip}
				}

				if health && announceActions[aip.CidrString].Remove {
					announceActions[aip.CidrString].Remove = false
				}
			}
		}

		actions := []VtyshAction{}

		for _, ras := range routeActions {
			for _, ra := range ras {
				routes, err := VtyshShowIPRoute(context.Background(), ra.Afi, "static", []*net.IPNet{ra.Prefix.IPNet})
				if err != nil {
					return err
				}

				exist := false
				for _, r := range routes[ra.Prefix.CidrString] {
					for _, nh := range r.Nexthops {
						if nh.Afi != ra.Afi {
							continue
						}

						nhIP := net.ParseIP(nh.Ip)
						nhAnykIP := &AnykIP{IP: nhIP, IPString: nhIP.String(), Afi: nh.Afi}

						allowed := false
						for _, rt := range routeTargets {
							if rt.Afi == nh.Afi && rt.IPString == nh.Ip {
								allowed = true
								break
							}
						}

						if !allowed {
							actions = append(actions, &VtyshRouteAction{Remove: true, Prefix: ra.Prefix, Via: nhAnykIP})
						}

						if nh.Ip == ra.Via.IPString {
							if r.Distance != ra.Distance && !ra.Remove {
								actions = append(actions, &VtyshRouteAction{Remove: true, Prefix: ra.Prefix, Via: nhAnykIP, Distance: r.Distance})
								continue
							}

							exist = true
							continue
						}
					}

					if exist {
						break
					}
				}

				if exist == ra.Remove {
					actions = append(actions, ra)
				}
			}
		}

		for _, aas := range announceActions {
			ok, err := VtyshHasLocalRoute(context.TODO(), aas.Prefix)
			if err != nil {
				return err
			}

			if ok && aas.Remove || !ok && !aas.Remove {
				actions = append(actions, aas)
			}
		}

		err = VtyshExecute(context.TODO(), actions)
		if err != nil {
			return err
		}
	}

	return nil
}
