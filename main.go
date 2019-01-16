package main

import (
	"github.com/docker/go-plugins-helpers/network"
	"github.com/nategraf/l2bridge-driver/l2bridge"
)

const (
	socketAddress = "/run/docker/plugins/l2bridge.sock"
)

func main() {
	d := l2bridge.NewDriver()
	h := network.NewHandler(d)
	h.ServeUnix(socketAddress, 0)
}
