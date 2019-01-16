package l2bridge

import (
	"github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/types"
	"github.com/sirupsen/logrus"
)

type Driver struct {
	bridge *bridgeDriver
}

func NewDriver() *Driver {
	return &Driver{
		bridge: newBridgeDriver(),
	}
}

var capabilities = &network.CapabilitiesResponse{
	Scope:             network.LocalScope,
	ConnectivityScope: network.LocalScope,
}

func logRequest(fname string, req interface{}, res interface{}, err error) {
	if err == nil {
		logrus.Infof("%s(%v): %v", fname, req, res)
		return
	}
	switch err.(type) {
	case types.MaskableError:
		logrus.Infof("[MaskableError] %s(%v): %v", fname, req, err)
	case types.RetryError:
		logrus.Infof("[RetryError] %s(%v): %v", fname, req, err)
	case types.BadRequestError:
		logrus.Warnf("[BadRequestError] %s(%v): %v", fname, req, err)
	case types.NotFoundError:
		logrus.Warnf("[NotFoundError] %s(%v): %v", fname, req, err)
	case types.ForbiddenError:
		logrus.Warnf("[ForbiddenError] %s(%v): %v", fname, req, err)
	case types.NoServiceError:
		logrus.Warnf("[NoServiceError] %s(%v): %v", fname, req, err)
	case types.NotImplementedError:
		logrus.Warnf("[NotImplementedError] %s(%v): %v", fname, req, err)
	case types.TimeoutError:
		logrus.Errorf("[TimeoutError] %s(%v): %v", fname, req, err)
	case types.InternalError:
		logrus.Errorf("[InternalError] %s(%v): %v", fname, req, err)
	default:
		// Unclassified errors should be treated as bad.
		logrus.Errorf("[UNKNOWN] %s(%v): %v", fname, req, err)
	}
}

func (d *Driver) GetCapabilities() (res *network.CapabilitiesResponse, err error) {
	defer func() { logRequest("GetCapabilities", nil, res, err) }()
	return capabilities, nil
}

func (d *Driver) CreateNetwork(req *network.CreateNetworkRequest) (err error) {
	defer func() { logRequest("CreateNetwork", req, nil, err) }()

	// Convert string IP addresses in the request to net.IPNet.
	ipv4, err := ParseIPAMDataSlice(req.IPv4Data)
	if err != nil {
		return types.BadRequestErrorf("invalid IPv4 information: %v", err)
	}
	ipv6, err := ParseIPAMDataSlice(req.IPv6Data)
	if err != nil {
		return types.BadRequestErrorf("invalid IPv6 information: %v", err)
	}

	// Call into the real bridge driver.
	return d.bridge.CreateNetwork(req.NetworkID, req.Options, ipv4, ipv6)
}

func (d *Driver) AllocateNetwork(req *network.AllocateNetworkRequest) (res *network.AllocateNetworkResponse, err error) {
	defer func() { logRequest("AllocateNetwork", req, res, err) }()
	return nil, types.NotImplementedErrorf("not implemented")
}

func (d *Driver) DeleteNetwork(req *network.DeleteNetworkRequest) (err error) {
	defer func() { logRequest("DeleteNetwork", req, nil, err) }()
	return d.bridge.DeleteNetwork(req.NetworkID)
}

func (d *Driver) FreeNetwork(req *network.FreeNetworkRequest) (err error) {
	defer func() { logRequest("FreeNetwork", req, nil, err) }()
	return types.NotImplementedErrorf("not implemented")
}

func (d *Driver) CreateEndpoint(req *network.CreateEndpointRequest) (res *network.CreateEndpointResponse, err error) {
	defer func() { logRequest("CreateEndpoint", req, res, err) }()

	ei, err := ParseEndpointInterface(req.Interface)
	if err != nil {
		return nil, types.BadRequestErrorf("invalid endpoint info: %v", err)
	}
	ei, err = d.bridge.CreateEndpoint(req.NetworkID, req.EndpointID, ei, req.Options)
	if err != nil {
		return nil, err
	}
	return &network.CreateEndpointResponse{Interface: ei.Marshal()}, nil
}

func (d *Driver) DeleteEndpoint(req *network.DeleteEndpointRequest) (err error) {
	defer func() { logRequest("DeleteEndpoint", req, nil, err) }()
	return d.bridge.DeleteEndpoint(req.NetworkID, req.EndpointID)
}

func (d *Driver) EndpointInfo(req *network.InfoRequest) (res *network.InfoResponse, err error) {
	defer func() { logRequest("EndpointInfo", req, res, err) }()
	return nil, types.NotImplementedErrorf("not implemented")
}

func (d *Driver) Join(req *network.JoinRequest) (res *network.JoinResponse, err error) {
	defer func() { logRequest("Join", req, res, err) }()
	info, err := d.bridge.Join(req.NetworkID, req.EndpointID, req.SandboxKey, req.Options)
	if err != nil {
		return nil, err
	}
	return info.Marshal(), nil
}

func (d *Driver) Leave(req *network.LeaveRequest) (err error) {
	defer func() { logRequest("Leave", req, nil, err) }()
	return d.bridge.Leave(req.NetworkID, req.EndpointID)
}

func (d *Driver) DiscoverNew(notif *network.DiscoveryNotification) (err error) {
	defer func() { logRequest("DiscoverNew", notif, nil, err) }()
	return nil
}

func (d *Driver) DiscoverDelete(notif *network.DiscoveryNotification) (err error) {
	defer func() { logRequest("DiscoverDelete", notif, nil, err) }()
	return nil
}

func (d *Driver) ProgramExternalConnectivity(req *network.ProgramExternalConnectivityRequest) (err error) {
	defer func() { logRequest("ProgramExternalConnectivity", req, nil, err) }()
	return types.NotImplementedErrorf("not implemented")
}

func (d *Driver) RevokeExternalConnectivity(req *network.RevokeExternalConnectivityRequest) (err error) {
	defer func() { logRequest("RevokeExternalConnectivity", req, nil, err) }()
	return types.NotImplementedErrorf("not implemented")
}
