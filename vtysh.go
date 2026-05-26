package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strconv"
)

type vtyshRoute struct {
	Distance int            `json:"distance"`
	Nexthops []vtyshNexthop `json:"nexthops"`
}

type vtyshNexthop struct {
	Ip  string `json:"ip"`
	Afi string `json:"afi"`
}

type vtyshSelfOriginate struct {
	Routes vtyshRoutes `json:"routes"`
}

type vtyshRoutes map[string][]*vtyshRoute

type vtyshAction interface {
	command() []string
}

type announceAction struct {
	router string
	remove bool
	afi    string
	prefix *Net
}

func (a *announceAction) command() []string {
	neg := ""
	if a.remove {
		neg = "no "
	}

	return []string{"-c", "config", "-c", "router bgp " + a.router, "-c", "address-family " + a.afi + " unicast", "-c", neg + "network " + a.prefix.String()}
}

type routeAction struct {
	remove   bool
	afi      string
	prefix   *Net
	distance int
	via      *IP
}

func (r *routeAction) command() []string {
	neg := ""
	if r.remove {
		neg = "no "
	}

	proto := "ip"
	if r.afi == "ipv6" {
		proto = r.afi
	}

	dist := ""
	if !r.remove && r.distance > 1 {
		dist = " " + strconv.Itoa(r.distance)
	}

	return []string{"-c", "config", "-c", neg + proto + " route " + r.prefix.String() + " " + r.via.String() + dist}
}

func vtyshShowSelfOriginate(ctx context.Context, afi string, filter []*net.IPNet) (vtyshRoutes, error) {
	l := WithLogger(ctx)

	proto := "ip"
	if afi == "ipv6" {
		proto = afi
	}

	command := []string{"vtysh", "-c", "show " + proto + " bgp self-originate json"}
	l.Debug().Strs("command", command).Msgf("executing vtysh command")

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	soRoutes := vtyshSelfOriginate{}
	if err := json.Unmarshal(out, &soRoutes); err != nil {
		return nil, err
	}

	if len(filter) == 0 {
		return soRoutes.Routes, nil
	}

	for prefix := range soRoutes.Routes {
		netIP, _, err := net.ParseCIDR(prefix)
		if err != nil {
			return nil, err
		}

		contained := false
		for _, f := range filter {
			if f.Contains(netIP) {
				contained = true
				break
			}
		}

		if !contained {
			delete(soRoutes.Routes, prefix)
		}
	}

	return soRoutes.Routes, nil
}

func vtyshShowRoutes(ctx context.Context, afi string, filter []*net.IPNet) (vtyshRoutes, error) {
	l := WithLogger(ctx)

	proto := "ip"
	if afi == "ipv6" {
		proto = afi
	}

	command := []string{"vtysh", "-c", "show " + proto + " route static json"}
	l.Debug().Strs("command", command).Msgf("executing vtysh command")

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	routes := vtyshRoutes{}
	if err := json.Unmarshal(out, &routes); err != nil {
		return nil, err
	}

	if len(filter) == 0 {
		return routes, nil
	}

	for prefix := range routes {
		netIP, _, err := net.ParseCIDR(prefix)
		if err != nil {
			return nil, err
		}

		contained := false
		for _, f := range filter {
			if f.Contains(netIP) {
				contained = true
				break
			}
		}

		if !contained {
			delete(routes, prefix)
		}
	}

	return routes, nil
}

func vtyshExecute(ctx context.Context, actions []vtyshAction) error {
	l := WithLogger(ctx)

	if len(actions) == 0 {
		return nil
	}

	var args []string
	for _, a := range actions {
		args = append(args, a.command()...)
		args = append(args, "-c", "end")
	}

	args = append(args, "-c", "write mem")

	command := append([]string{"vtysh"}, args...)
	l.Debug().Strs("command", command).Msgf("executing vtysh command")

	cmd := exec.CommandContext(ctx, "vtysh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("vtysh: %w: %s", err, out)
	}

	return nil
}
