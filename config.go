package anyk

import (
	"fmt"
	"net"
)

func (c *AnykConfig) Validate() error {
	if c.ASN <= 0 {
		return fmt.Errorf("asn must be a positive integer")
	}
	for i, svc := range c.Services {
		if svc.Name == "" {
			return fmt.Errorf("services[%d]: name is required", i)
		}
		if len(svc.AnycastIPs) == 0 {
			return fmt.Errorf("service %q: anycast_ips is required", svc.Name)
		}
		for _, cidr := range svc.AnycastIPs {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				return fmt.Errorf("service %q: anycast_ip %q is not a valid CIDR (use e.g. 10.0.0.1/32)", svc.Name, cidr)
			}
		}
		for j, ep := range svc.Endpoints {
			if net.ParseIP(ep.IP) == nil {
				return fmt.Errorf("service %q endpoint[%d]: %q is not a valid IP address", svc.Name, j, ep.IP)
			}
			if ep.DNSCheck == nil && ep.HTTPCheck == nil {
				return fmt.Errorf("service %q endpoint %q: at least one check (http_check or dns_check) is required", svc.Name, ep.IP)
			}
			if ep.DNSCheck != nil {
				t := ep.DNSCheck.Type
				if t != "A" && t != "AAAA" && t != "TXT" {
					return fmt.Errorf("service %q endpoint %q: unsupported dns_check type %q (supported: A, AAAA, TXT)", svc.Name, ep.IP, t)
				}
			}
		}
	}
	return nil
}

type AnykConfig struct {
	ASN      int           `yaml:"asn" json:"asn,omitempty"`
	Services []AnykService `yaml:"services,omitempty" json:"services,omitempty"`
}

type AnykService struct {
	Name       string          `yaml:"name"                  json:"name"`
	Active     bool            `yaml:"active,omitempty"      json:"active,omitempty"`
	AnycastIPs []string        `yaml:"anycast_ips,omitempty" json:"anycast_ips,omitempty"`
	Endpoints  []AnykEndpoint  `yaml:"endpoints,omitempty"   json:"endpoints,omitempty"`
	Callbacks  []*AnykCallback `yaml:"callbacks" json:"callbacks,omitempty"`
}

type AnykCallback struct {
	Healthy bool     `yaml:"healthy" json:"healthy"`
	Command []string `yaml:"command" json:"command"`
}

type AnykEndpoint struct {
	IP        string         `yaml:"ip"                   json:"ip"`
	Distance  int            `yaml:"distance,omitempty"   json:"distance,omitempty"`
	HTTPCheck *AnykHTTPCheck `yaml:"http_check,omitempty" json:"http_check,omitempty"`
	DNSCheck  *AnykDNSCheck  `yaml:"dns_check,omitempty"  json:"dns_check,omitempty"`
}

type AnykHTTPCheck struct {
	Verb               string            `yaml:"verb,omitempty"                 json:"verb,omitempty"`
	URL                string            `yaml:"url,omitempty"                  json:"url,omitempty"`
	ExpectedCode       int               `yaml:"expected_code,omitempty"        json:"expected_code,omitempty"`
	Headers            map[string]string `yaml:"headers,omitempty"              json:"headers,omitempty"`
	Body               string            `yaml:"body,omitempty"                 json:"body,omitempty"`
	Timeout            int               `yaml:"timeout,omitempty"              json:"timeout,omitempty"`
	InsecureSkipVerify bool              `yaml:"insecure_skip_verify,omitempty" json:"insecure_skip_verify,omitempty"`
}

type AnykDNSCheck struct {
	Resolver    string `yaml:"resolver,omitempty" json:"resolver,omitempty"`
	Type        string `yaml:"type,omitempty"     json:"type,omitempty"`
	Query       string `yaml:"query,omitempty"    json:"query,omitempty"`
	Expected    string `yaml:"expected,omitempty" json:"expected,omitempty"`
	Timeout     int    `yaml:"timeout,omitempty"  json:"timeout,omitempty"`
	FollowCNAME bool   `yaml:"follow_cname"       json:"follow_cname"`
}
