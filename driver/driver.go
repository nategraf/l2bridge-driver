package driver

import (
    "log"

    "github.com/docker/go-plugins-helpers/network"
    "github.com/docker/libnetwork/types"
    "github.com/nategraf/simplenet/bridge"
)

type Driver struct {}

func unimplemented(name string) error {
    return types.NotImplementedErrorf("%s is not implemented", name)
}

var capabilities = &network.CapabilitiesResponse{
    Scope: network.LocalScope,
    ConnectivityScope: network.LocalScope,
}

func logRequest(fname string, req interface{}, res interface{}, err error) {
    if err == nil {
        log.Printf("%s(%v): %v", fname, req, res)
    } else {
        log.Printf("[FAILED] %s(%v): %v", fname, req, err)
    }
}

func (d *Driver) GetCapabilities() (res *network.CapabilitiesResponse, err error) {
    defer logRequest("GetCapabilities", nil, res, err)
    return capabilities, nil
}

func (d *Driver) CreateNetwork(req *network.CreateNetworkRequest) (err error) {
    defer logRequest("CreateNetwork", req, nil, err)
    return unimplemented("CreateNetwork")
}

func (d *Driver) AllocateNetwork(req *network.AllocateNetworkRequest) (res *network.AllocateNetworkResponse, err error) {
    defer logRequest("AllocateNetwork", req, res, err)
    return nil, unimplemented("AllocateNetwork")
}

func (d *Driver) DeleteNetwork(req *network.DeleteNetworkRequest) (err error) {
    defer logRequest("DeleteNetwork", req, nil, err)
    return unimplemented("DeleteNetwork")
}

func (d *Driver) FreeNetwork(req *network.FreeNetworkRequest) (err error) {
    defer logRequest("FreeNetwork", req, nil, err)
    return unimplemented("FreeNetwork")
}

func (d *Driver) CreateEndpoint(req *network.CreateEndpointRequest) (res *network.CreateEndpointResponse, err error) {
    defer logRequest("CreateEndpoint", req, res, err)
    return nil, unimplemented("CreateEndpoint")
}

func (d *Driver) DeleteEndpoint(req *network.DeleteEndpointRequest) (err error) {
    defer logRequest("DeleteEndpoint", req, nil, err)
    return unimplemented("DeleteEndpoint")
}

func (d *Driver) EndpointInfo(req *network.InfoRequest) (res *network.InfoResponse, err error) {
    defer logRequest("EndpointInfo", req, res, err)
    return nil, unimplemented("EndpointInfo")
}

func (d *Driver) Join(req *network.JoinRequest) (res *network.JoinResponse, err error) {
    defer logRequest("Join", req, res, err)
    return nil, unimplemented("Join")
}

func (d *Driver) Leave(req *network.LeaveRequest) (err error) {
    defer logRequest("Leave", req, nil, err)
    return unimplemented("Leave")
}

func (d *Driver) DiscoverNew(notif *network.DiscoveryNotification) (err error) {
    defer logRequest("DiscoverNew", notif, nil, err)
    return unimplemented("DiscoverNew")
}

func (d *Driver) DiscoverDelete(notif *network.DiscoveryNotification) (err error) {
    defer logRequest("DiscoverDelete", notif, nil, err)
    return unimplemented("DiscoverDelete")
}

func (d *Driver) ProgramExternalConnectivity(req *network.ProgramExternalConnectivityRequest) (err error) {
    defer logRequest("ProgramExternalConnectivity", req, nil, err)
    return unimplemented("ProgramExternalConnectivity")
}

func (d *Driver) RevokeExternalConnectivity(req *network.RevokeExternalConnectivityRequest) (err error) {
    defer logRequest("RevokeExternalConnectivity", req, nil, err)
    return unimplemented("RevokeExternalConnectivity")
}
