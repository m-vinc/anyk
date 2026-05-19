package main

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
	Verb         string            `yaml:"verb,omitempty"          json:"verb,omitempty"`
	URL          string            `yaml:"url,omitempty"           json:"url,omitempty"`
	ExpectedCode int               `yaml:"expected_code,omitempty" json:"expected_code,omitempty"`
	Headers      map[string]string `yaml:"headers,omitempty"       json:"headers,omitempty"`
	Body         string            `yaml:"body,omitempty"          json:"body,omitempty"`
	Timeout      int               `yaml:"timeout,omitempty"       json:"timeout,omitempty"`
}

type AnykDNSCheck struct {
	Resolver    string `yaml:"resolver,omitempty" json:"resolver,omitempty"`
	Type        string `yaml:"type,omitempty"     json:"type,omitempty"`
	Query       string `yaml:"query,omitempty"    json:"query,omitempty"`
	Expected    string `yaml:"expected,omitempty" json:"expected,omitempty"`
	Timeout     int    `yaml:"timeout,omitempty"  json:"timeout,omitempty"`
	FollowCNAME bool   `yaml:"follow_cname" json:"follow_cname"`
}
