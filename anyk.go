package main

import "net"

type AnykIP struct {
	net.IP

	Afi      string
	IPString string
}

type AnykNet struct {
	*net.IPNet

	CidrString string
	Afi        string
}
