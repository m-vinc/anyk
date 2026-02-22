# Anyk

Anyk is a small utility that tracks the health of services using DNS or HTTP queries and announces anycast IPs by manipulating the FRR configuration through vtysh.

## Build

Everything you need to build Anyk is `make` and the `Go toolchain`.

```
$ make
Building anyk with version 1.0.0...
GOOS=linux GOARCH=amd64 go build  -trimpath -ldflags "-X main.version=1.0.0" -o ./build/anyk
```

## Usage

Write an Anyk YAML configuration file :

```yaml
router: 65534

services:
  - name: a_simple_nginx_instance
    active: true
    anycast_ips: [10.0.0.1]
    endpoints:
      - ip: 10.0.0.5
        http_check:
          type: GET
          expected_code: 200

  - name: dc
    active: true
    anycast_ips: [fd00:0dc9:e421::1, 10.0.1.1]
    endpoints:
      - ip: 10.0.1.5
        dns_check:
          type: TXT
          query: _kerberos.my.lan.
          expected: "MY.LAN"

      - ip: fd00:0dc9:e421::5
        dns_check:
          type: TXT
          query: _kerberos.my.lan.
          expected: "MY.LAN"
```

Run Anyk and specify your config file. If you need to run it periodically, you can use your favorite cron daemon:

```
$ anyk run anyk.yml
2026/02/22 21:48:03 Running with config: /root/anyk/anyk.yml &{65534 [0x35d828174b40 0x35d828174fa0]}
2026/02/22 21:48:03 executing vtysh command: [-c show ip bgp 10.0.0.1 json]
2026/02/22 21:48:03 executing vtysh command: [-c show bgp fd00:0dc9:e421::1 json]
2026/02/22 21:48:03 executing vtysh command: [-c config -c ip route 10.0.0.1/32 10.0.0.5]
2026/02/22 21:48:03 executing vtysh command: [-c config -c ipv6 route fd00:0dc9:e421::1 fd00:0dc9:e421::5]
2026/02/22 21:48:03 executing vtysh command: [-c config -c router bgp 65534 -c address-family ipv4 unicast -c network 10.0.0.1]
2026/02/22 21:48:03 executing vtysh command: [-c config -c router bgp 65534 -c address-family ipv6 unicast -c network fd00:0dc9:e421::1/128]
```

