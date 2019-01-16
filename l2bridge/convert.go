package l2bridge

import (
	"fmt"
	"net"

	"github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/types"
)

// IPAMData contains IPv4 or IPv6 addressing information.
type IPAMData struct {
	AddressSpace string
	Pool         *net.IPNet
	Gateway      *net.IPNet
	AuxAddresses map[string]*net.IPNet
}

func ParseIPAMDataSlice(in []*network.IPAMData) ([]*IPAMData, error) {
	var out []*IPAMData
	for _, data := range in {
		parsed, err := ParseIPAMData(data)
		if err != nil {
			return nil, types.BadRequestErrorf("invalid ipv4 information: %v", err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func ParseIPAMData(in *network.IPAMData) (*IPAMData, error) {
	out := &IPAMData{
		AddressSpace: in.AddressSpace,
	}

	var err error
	if in.Pool != "" {
		if _, out.Pool, err = net.ParseCIDR(in.Pool); err != nil {
			return out, fmt.Errorf("bad pool address: %v ", err)
		}
	}
	if in.Gateway != "" {
		if _, out.Gateway, err = net.ParseCIDR(in.Gateway); err != nil {
			return out, fmt.Errorf("bad gateway address: %v ", err)
		}
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

// EndpointInterface contains information about an endpoint spanning a container boundary.
type EndpointInterface struct {
	MacAddress  net.HardwareAddr
	Address     *net.IPNet
	AddressIPv6 *net.IPNet
}

func ParseEndpointInterface(in *network.EndpointInterface) (*EndpointInterface, error) {
	out := &EndpointInterface{}

	var err error
	if in.MacAddress != "" {
		if out.MacAddress, err = net.ParseMAC(in.MacAddress); err != nil {
			return nil, fmt.Errorf("bad MAC address: %v ", err)
		}
	}
	if in.Address != "" {
		if _, out.Address, err = net.ParseCIDR(in.Address); err != nil {
			return nil, fmt.Errorf("bad IPv4 address: %v ", err)
		}
	}
	if in.AddressIPv6 != "" {
		if _, out.AddressIPv6, err = net.ParseCIDR(in.AddressIPv6); err != nil {
			return nil, fmt.Errorf("bad ipv6 address: %v ", err)
		}
	}
	return out, nil
}

func (e *EndpointInterface) Marshal() *network.EndpointInterface {
	if e == nil {
		return nil
	}
	out := &network.EndpointInterface{}
	if e.MacAddress != nil {
		out.MacAddress = e.MacAddress.String()
	}
	if e.Address != nil {
		out.Address = e.Address.String()
	}
	if e.AddressIPv6 != nil {
		out.AddressIPv6 = e.AddressIPv6.String()
	}
	return out
}

// StaticRoute contains static route information set during a Join.
type StaticRoute struct {
	Destination *net.IPNet
	RouteType   int
	NextHop     net.IP
}

func (s *StaticRoute) Marshal() *network.StaticRoute {
	if s == nil {
		return nil
	}
	out := &network.StaticRoute{
		RouteType: s.RouteType,
	}
	if s.Destination != nil {
		out.Destination = s.Destination.String()
	}
	if s.NextHop != nil {
		out.NextHop = s.NextHop.String()
	}
	return out
}

// Interface name specifies naming for an endpoint in and out of the container.
type InterfaceName network.InterfaceName

// Join response contains settins that can be established by the driver when a container joins a network.
type JoinResponse struct {
	InterfaceName         InterfaceName
	Gateway               net.IP
	GatewayIPv6           net.IP
	StaticRoutes          []*StaticRoute
	DisableGatewayService bool
}

func (j *JoinResponse) Marshal() *network.JoinResponse {
	if j == nil {
		return nil
	}
	out := &network.JoinResponse{
		InterfaceName:         network.InterfaceName(j.InterfaceName),
		DisableGatewayService: j.DisableGatewayService,
	}
	if j.Gateway != nil {
		out.Gateway = j.Gateway.String()
	}
	if j.GatewayIPv6 != nil {
		out.GatewayIPv6 = j.GatewayIPv6.String()
	}
	for _, route := range j.StaticRoutes {
		out.StaticRoutes = append(out.StaticRoutes, route.Marshal())
	}
	return out
}
