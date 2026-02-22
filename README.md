# Anyk

Anyk is a little utility that track health of a services using DNS or HTTP query and announce anycasts IP by manipulating FRR config through vtysh.

## Build

Everything you need to build Anyk is `make` and the `Go toolchain`.

```
$ make
```

## Usage

Write a anyk yaml configuration file :

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

Run anyk and specify your config file :

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
