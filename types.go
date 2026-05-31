package anyk

import "net"

type IP struct {
	net.IP
	Afi      string
	IPString string
}

type Net struct {
	*net.IPNet
	CidrString string
	Afi        string
}
