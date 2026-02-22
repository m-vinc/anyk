package main

import (
	"net"
)

type AnykConfig struct {
	Router string `yaml:"router"`

	Services []*AnykAnycastService `yaml:"services"`
}

type AnykAnycastService struct {
	Name       string                 `yaml:"name"`
	Active     bool                   `yaml:"active"`
	Endpoints  []*AnykServiceEndpoint `yaml:"endpoints"`
	AnycastIPs []net.IP               `yaml:"anycast_ips"`
}

type AnykServiceEndpoint struct {
	IP       net.IP `yaml:"ip"`
	Afi      string `yaml:"-"`
	Distance int    `yaml:"distance"`

	DNSCheck  *AnykEndpointDNSCheck  `yaml:"dns_check"`
	HTTPCheck *AnykEndpointHTTPCheck `yaml:"http_check"`
}

type AnykEndpointDNSCheck struct {
	Resolver string `yaml:"resolver"`
	Type     string `yaml:"type"`
	Query    string `yaml:"query"`
	Expected string `yaml:"expected"`
}

type AnykEndpointHTTPCheck struct {
	Verb         string            `yaml:"verb"`
	URL          string            `yaml:"url"`
	Headers      map[string]string `yaml:"headers"`
	Body         string            `yaml:"body"`
	ExpectedCode int               `yaml:"expected_code"`
}
