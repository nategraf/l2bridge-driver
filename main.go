package main

import (
    "github.com/docker/go-plugins-helpers/network"
    "github.com/nategraf/simplenet/driver"
)

const (
    socketAddress = "/run/docker/plugins/simple.sock"
)

func main() {
    d := &driver.Driver{}
    h := network.NewHandler(d)
    h.ServeUnix(socketAddress, 0)
}
