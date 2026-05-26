package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"codeberg.org/miekg/dns"
)

const defaultHTTPTimeout = 5 * time.Second
const defaultDNSTimeout = 1 * time.Second

func httpCheck(ctx context.Context, endpoint *AnykEndpoint) (bool, error) {
	check := endpoint.HTTPCheck

	verb := check.Verb
	if verb == "" {
		verb = "GET"
	}
	expectedCode := check.ExpectedCode
	if expectedCode == 0 {
		expectedCode = 200
	}

	u, err := url.Parse(check.URL)
	if err != nil {
		return false, err
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	if u.Host == "" {
		u.Host = endpoint.IP
		if strings.Contains(endpoint.IP, ":") {
			u.Host = "[" + endpoint.IP + "]"
		}
	}

	var body io.Reader
	if check.Body != "" {
		body = strings.NewReader(check.Body)
	}

	timeout := defaultHTTPTimeout
	if check.Timeout > 0 {
		timeout = time.Duration(check.Timeout) * time.Second
	}
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, verb, u.String(), body)
	if err != nil {
		return false, err
	}
	for k, v := range check.Headers {
		req.Header.Add(k, v)
	}

	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)

	return res.StatusCode == expectedCode, nil
}

const maxCNAMEDepth = 10

func resolveCNAME(ctx context.Context, client *dns.Client, resolver, name string) (string, error) {
	for range maxCNAMEDepth {
		msg := dns.NewMsg(name, dns.TypeCNAME)
		in, _, err := client.Exchange(ctx, msg, "udp", resolver)
		if err != nil {
			return "", err
		}
		next := ""
		for _, rr := range in.Answer {
			if cname, ok := rr.(*dns.CNAME); ok {
				next = cname.Target
				break
			}
		}
		if next == "" {
			return name, nil
		}
		name = next
	}
	return "", fmt.Errorf("CNAME chain exceeds maximum depth (%d)", maxCNAMEDepth)
}

func dnsCheck(ctx context.Context, endpoint *AnykEndpoint) (bool, error) {
	check := endpoint.DNSCheck
	if check.Type != "A" && check.Type != "AAAA" && check.Type != "TXT" {
		return false, fmt.Errorf("unsupported dns check type: %s", check.Type)
	}

	timeout := defaultDNSTimeout
	if check.Timeout > 0 {
		timeout = time.Duration(check.Timeout) * time.Second
	}

	client := new(dns.Client)
	client.Transport = dns.NewTransport()
	client.Transport.ReadTimeout = timeout
	client.Transport.WriteTimeout = timeout

	resolver := check.Resolver
	if resolver == "" {
		resolver = endpoint.IP
		if strings.Contains(resolver, ":") {
			resolver = "[" + resolver + "]"
		}
		resolver += ":53"
	}

	query := check.Query
	if check.FollowCNAME {
		var err error
		query, err = resolveCNAME(ctx, client, resolver, query)
		if err != nil {
			return false, err
		}
	}

	var msg *dns.Msg
	switch check.Type {
	case "A":
		msg = dns.NewMsg(query, dns.TypeA)
	case "AAAA":
		msg = dns.NewMsg(query, dns.TypeAAAA)
	case "TXT":
		msg = dns.NewMsg(query, dns.TypeTXT)
	}

	in, _, err := client.Exchange(ctx, msg, "udp", resolver)
	if err != nil {
		return false, err
	}

	for _, answer := range in.Answer {
		switch check.Type {
		case "A":
			if a, ok := answer.(*dns.A); ok && a.A.String() == check.Expected {
				return true, nil
			}
		case "AAAA":
			if aaaa, ok := answer.(*dns.AAAA); ok && aaaa.AAAA.String() == check.Expected {
				return true, nil
			}
		case "TXT":
			if txt, ok := answer.(*dns.TXT); ok {
				s := txt.TXT.String()
				if text, err := strconv.Unquote(s); err == nil {
					if text == check.Expected {
						return true, nil
					}
				} else if s == check.Expected {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func healthcheck(ctx context.Context, endpoint *AnykEndpoint) (bool, error) {
	if endpoint.DNSCheck == nil && endpoint.HTTPCheck == nil {
		return false, fmt.Errorf("endpoint %q has no checks configured", endpoint.IP)
	}
	if endpoint.DNSCheck != nil {
		ok, err := dnsCheck(ctx, endpoint)
		if err != nil || !ok {
			return false, err
		}
	}
	if endpoint.HTTPCheck != nil {
		ok, err := httpCheck(ctx, endpoint)
		if err != nil || !ok {
			return false, err
		}
	}
	return true, nil
}
