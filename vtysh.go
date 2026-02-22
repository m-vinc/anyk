package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
)

type VtyshBGP struct {
	Prefix string          `json:"prefix"`
	Paths  []*VtyshBGPPath `json:"paths"`
}

type VtyshBGPPath struct {
	ASPath *VtyshBGPASPath `json:"aspath"`
	Origin string          `json:"origin"`
	Valid  bool            `json:"valid"`
	Local  bool            `json:"local"`
}

type VtyshBGPASPath struct {
	String string `json:"string"`
}

type VtyshRoute struct {
	Prefix                   string         `json:"prefix"`
	PrefixLen                int            `json:"prefixLen"`
	Protocol                 string         `json:"protocol"`
	VrfId                    int            `json:"vrfId"`
	VrfName                  string         `json:"vrfName"`
	Selected                 bool           `json:"selected"`
	DestSelected             bool           `json:"destSelected"`
	Distance                 int            `json:"distance"`
	Metric                   int            `json:"metric"`
	Installed                bool           `json:"installed"`
	Table                    int            `json:"table"`
	InternalStatus           int            `json:"internalStatus"`
	InternalFlags            int            `json:"internalFlags"`
	InternalNextHopNum       int            `json:"internalNextHopNum"`
	InternalNextHopActiveNum int            `json:"internalNextHopActiveNum"`
	NexthopGroupId           int            `json:"nexthopGroupId"`
	InstalledNexthopGroupId  int            `json:"installedNexthopGroupId"`
	Uptime                   string         `json:"uptime"`
	Nexthops                 []VtyshNexthop `json:"nexthops"`
}

type VtyshNexthop struct {
	Flags          int    `json:"flags"`
	Fib            bool   `json:"fib"`
	Ip             string `json:"ip"`
	Afi            string `json:"afi"`
	InterfaceIndex int    `json:"interfaceIndex"`
	InterfaceName  string `json:"interfaceName"`
	Active         bool   `json:"active"`
	Weight         int    `json:"weight"`
}

type VtyshRoutes map[string][]*VtyshRoute

func VtyshHasLocalRoute(ctx context.Context, prefix *AnykNet) (bool, error) {
	afi := ""
	if prefix.Afi == "ipv4" {
		afi = "ip "
	}

	args := []string{"-c", "show " + afi + "bgp " + prefix.CidrString + " json"}
	log.Printf("executing vtysh command: %+v", args)
	cmd := exec.CommandContext(ctx, "vtysh", args...)

	out := new(bytes.Buffer)
	outerr := new(bytes.Buffer)

	cmd.Stdout = out
	cmd.Stderr = outerr

	err := cmd.Run()
	if err != nil {
		return false, err
	}

	bgpInfo := VtyshBGP{}
	err = json.Unmarshal(out.Bytes(), &bgpInfo)
	if err != nil {
		return false, err
	}

	if len(bgpInfo.Paths) == 0 {
		return false, nil
	}

	for _, p := range bgpInfo.Paths {
		if p.Origin == "IGP" && p.Valid && p.Local {
			return true, nil
		}
	}

	return false, nil
}

func VtyshShowIPRoute(ctx context.Context, afi string, routeType string, filter []*net.IPNet) (VtyshRoutes, error) {
	if afi != "ipv4" && afi != "ipv6" {
		return nil, fmt.Errorf("supported protocol: ip ipv6")
	}

	if routeType != "static" {
		return nil, fmt.Errorf("supported routeType: static")
	}

	proto := "ip"
	if afi == "ipv6" {
		proto = afi
	}

	cmd := exec.CommandContext(ctx, "vtysh", "-c", "show "+proto+" route "+routeType+" json")

	out := new(bytes.Buffer)
	outerr := new(bytes.Buffer)

	cmd.Stdout = out
	cmd.Stderr = outerr

	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	routes := VtyshRoutes{}
	err = json.Unmarshal(out.Bytes(), &routes)
	if err != nil {
		return nil, err
	}

	if len(filter) > 0 {
		toRemove := []string{}
		for prefix := range routes {
			netIP, _, err := net.ParseCIDR(prefix)
			if err != nil {
				return nil, err
			}

			contained := false
			for _, prefixFilter := range filter {
				if prefixFilter.Contains(netIP) {
					contained = true
					break
				}
			}

			if !contained {
				toRemove = append(toRemove, prefix)
			}

		}

		for _, prefix := range toRemove {
			delete(routes, prefix)
		}
	}

	return routes, nil
}

func VtyshExecute(ctx context.Context, actions []VtyshAction) error {
	for _, action := range actions {
		args := action.Command()
		log.Printf("executing vtysh command: %+v", args)
		cmd := exec.CommandContext(ctx, "vtysh", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("vtysh action execution failed %w: %+v", err, string(out))
		}
	}

	return VtyshWriteMem(ctx)
}

func VtyshWriteMem(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "vtysh", "-c", "write mem")
	_, err := cmd.CombinedOutput()
	return err
}

type VtyshAnnounceAction struct {
	Router string
	Remove bool

	Afi    string
	Prefix *AnykNet
}

func (aa *VtyshAnnounceAction) Command() []string {
	negation := ""
	if aa.Remove {
		negation = "no "
	}

	return []string{"-c", "config", "-c", "router bgp " + aa.Router, "-c", "address-family " + aa.Afi + " unicast", "-c", negation + "network " + aa.Prefix.String()}
}

type VtyshRouteAction struct {
	Remove bool

	Afi      string
	Prefix   *AnykNet
	Distance int
	Via      *AnykIP
}

func (ra *VtyshRouteAction) Command() []string {
	negation := ""
	if ra.Remove {
		negation = "no "
	}

	afi := "ip"
	if ra.Afi == "ipv6" {
		afi = ra.Afi
	}

	distance := ""
	if ra.Distance > 1 {
		distance = " " + strconv.FormatInt(int64(ra.Distance), 10)
	}

	return []string{"-c", "config", "-c", negation + afi + " route " + ra.Prefix.String() + " " + ra.Via.String() + distance}
}

type VtyshAction interface {
	Command() []string
}
