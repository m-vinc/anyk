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

func httpCheck(ctx context.Context, endpoint *AnykServiceEndpoint) (bool, error) {
	u, err := url.Parse(endpoint.HTTPCheck.URL)
	if err != nil {
		return false, err
	}

	if u.Scheme == "" {
		u.Scheme = "http"
	}

	if u.Host == "" {
		u.Host = endpoint.IP.String()
		if endpoint.Afi == "ipv6" {
			u.Host = "[" + u.Host + "]"
		}
	}

	var body io.Reader = nil
	if endpoint.HTTPCheck.Body != "" {
		body = strings.NewReader(endpoint.HTTPCheck.Body)
	}

	req, err := http.NewRequestWithContext(ctx, endpoint.HTTPCheck.Verb, u.String(), body)
	if err != nil {
		return false, err
	}

	if len(endpoint.HTTPCheck.Headers) > 0 {
		for key, value := range endpoint.HTTPCheck.Headers {
			req.Header.Add(key, value)
		}
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}

	if res.StatusCode != endpoint.HTTPCheck.ExpectedCode {
		return false, nil
	}

	return true, nil
}

func dnsCheck(ctx context.Context, endpoint *AnykServiceEndpoint) (bool, error) {
	if endpoint.DNSCheck.Type != "A" && endpoint.DNSCheck.Type != "TXT" {
		return false, fmt.Errorf("unsupported dns check query type: %+v", endpoint.DNSCheck.Type)
	}

	client := new(dns.Client)
	client.Transport = dns.NewTransport()
	client.Transport.ReadTimeout = time.Second * 1
	client.Transport.WriteTimeout = time.Second * 1
	var msg *dns.Msg

	switch endpoint.DNSCheck.Type {
	case "A":
		msg = dns.NewMsg(endpoint.DNSCheck.Query, dns.TypeA)
	case "TXT":
		msg = dns.NewMsg(endpoint.DNSCheck.Query, dns.TypeTXT)
	}

	resolver := endpoint.DNSCheck.Resolver

	if endpoint.DNSCheck.Resolver == "" {
		resolver = endpoint.IP.String()
		if strings.Contains(resolver, ":") {
			resolver = "[" + resolver + "]"
		}
		resolver += ":53"
	}

	in, _, err := client.Exchange(ctx, msg, "udp", resolver)
	if err != nil {
		return false, err
	}

	for _, answer := range in.Answer {
		if endpoint.DNSCheck.Type == "A" {
			a, ok := answer.(*dns.A)
			if !ok {
				continue
			}

			if a.A.String() == endpoint.DNSCheck.Expected {
				return true, nil
			}
		}

		if endpoint.DNSCheck.Type == "TXT" {
			txt, ok := answer.(*dns.TXT)
			if !ok {
				continue
			}

			text, _ := strconv.Unquote(txt.TXT.String())
			if text == endpoint.DNSCheck.Expected {
				return true, nil
			}
		}
	}

	return false, nil
}

func healthcheck(ctx context.Context, endpoint *AnykServiceEndpoint) (bool, error) {
	if endpoint.DNSCheck != nil {
		ok, err := dnsCheck(ctx, endpoint)
		if err != nil {
			return false, err
		}

		if !ok {
			return false, nil
		}
	}

	if endpoint.HTTPCheck != nil {
		ok, err := httpCheck(ctx, endpoint)
		if err != nil {
			return false, err
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}
