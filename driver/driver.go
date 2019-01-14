package driver

import (
        "fmt"
        "net"

	"github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/driverapi"
	"github.com/docker/libnetwork/types"
	"github.com/nategraf/simplenet/bridge"
	"github.com/sirupsen/logrus"
)

type Driver struct{
    Bridge *bridge.Driver
}

func New() *Driver {
    return &Driver{
        Bridge: bridge.NewDriver(),
    }
}

func unimplemented(name string) error {
	return types.NotImplementedErrorf("%s is not implemented", name)
}

var capabilities = &network.CapabilitiesResponse{
	Scope:             network.LocalScope,
	ConnectivityScope: network.LocalScope,
}

func logRequest(fname string, req interface{}, res interface{}, err error) {
	if err == nil {
		logrus.Infof("%s(%v): %v", fname, req, res)
	} else {
		logrus.Infof("[FAILED] %s(%v): %v", fname, req, err)
	}
}

func (d *Driver) GetCapabilities() (res *network.CapabilitiesResponse, err error) {
	defer logRequest("GetCapabilities", nil, res, err)
	return capabilities, nil
}

func (d *Driver) CreateNetwork(req *network.CreateNetworkRequest) (err error) {
	defer logRequest("CreateNetwork", req, nil, err)

        // Convert string IP addresses in the request to net.IPNet.
        ipv4, err := convertIPAMSlice(req.IPv4Data)
        if err != nil {
            return types.BadRequestErrorf("invalid IPv4 information: %v", err)
        }
        ipv6, err := convertIPAMSlice(req.IPv6Data)
        if err != nil {
            return types.BadRequestErrorf("invalid IPv6 information: %v", err)
        }

        // Call into the real bridge driver.
        return d.Bridge.CreateNetwork(req.NetworkID, req.Options, nil, ipv4, ipv6)
}

func (d *Driver) AllocateNetwork(req *network.AllocateNetworkRequest) (res *network.AllocateNetworkResponse, err error) {
	defer logRequest("AllocateNetwork", req, res, err)
	return nil, unimplemented("AllocateNetwork")
}

func (d *Driver) DeleteNetwork(req *network.DeleteNetworkRequest) (err error) {
	defer logRequest("DeleteNetwork", req, nil, err)
	return d.Bridge.DeleteNetwork(req.NetworkID)
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
	return nil
}

func (d *Driver) DiscoverDelete(notif *network.DiscoveryNotification) (err error) {
	defer logRequest("DiscoverDelete", notif, nil, err)
	return nil
}

func (d *Driver) ProgramExternalConnectivity(req *network.ProgramExternalConnectivityRequest) (err error) {
	defer logRequest("ProgramExternalConnectivity", req, nil, err)
	return unimplemented("ProgramExternalConnectivity")
}

func (d *Driver) RevokeExternalConnectivity(req *network.RevokeExternalConnectivityRequest) (err error) {
	defer logRequest("RevokeExternalConnectivity", req, nil, err)
	return unimplemented("RevokeExternalConnectivity")
}

func convertIPAMSlice(in []*network.IPAMData) ([]driverapi.IPAMData, error) {
    var out []driverapi.IPAMData
    for _, data := range in {
        converted, err := convertIPAMData(data)
        if err != nil {
            return nil, types.BadRequestErrorf("invalid ipv4 information: %v", err)
        }
        out = append(out, converted)
    }
    return out, nil
}

func convertIPAMData(in *network.IPAMData) (driverapi.IPAMData, error) {
    out := driverapi.IPAMData{
        AddressSpace: in.AddressSpace,
    }

    var err error
    if _, out.Pool, err = net.ParseCIDR(in.Pool); err != nil {
        return out, fmt.Errorf("bad pool address: %v ", err)
    }
    if _, out.Gateway, err = net.ParseCIDR(in.Gateway); err != nil {
        return out, fmt.Errorf("bad gateway address: %v ", err)
    }

    if len(in.AuxAddresses) != 0 {
        out.AuxAddresses = make(map[string]*net.IPNet)
    }
    for key, val := range in.AuxAddresses {
        switch addr := val.(type) {
        case *net.IPNet:
            out.AuxAddresses[key] = addr
        case string:
            if _, out.AuxAddresses[key], err = net.ParseCIDR(addr); err != nil {
                return out, fmt.Errorf("bad aux address %s: %v", key, err)
            }
        default:
            return out, fmt.Errorf("invalid aux address %s: %T is an unrecognized type", key, addr)
        }
    }
    return out, nil
}
