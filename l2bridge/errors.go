package l2bridge

import (
	"fmt"
)

// ErrInvalidDriverConfig error is returned when Bridge Driver is passed an invalid config
type ErrInvalidDriverConfig struct{}

func (eidc *ErrInvalidDriverConfig) Error() string {
	return "Invalid configuration passed to Bridge Driver"
}

// BadRequest denotes the type of this error
func (eidc *ErrInvalidDriverConfig) BadRequest() {}

// ErrInvalidEndpointConfig error is returned when an endpoint create is attempted with an invalid endpoint configuration.
type ErrInvalidEndpointConfig struct{}

func (eiec *ErrInvalidEndpointConfig) Error() string {
	return "trying to create an endpoint with an invalid endpoint configuration"
}

// BadRequest denotes the type of this error
func (eiec *ErrInvalidEndpointConfig) BadRequest() {}

// ErrInvalidTransportPortsOption is returned when the driver recieves a request with a TransportPorts key that could not be decoded.
type ErrInvalidTransportPortsOption struct{}

func (eitp *ErrInvalidTransportPortsOption) Error() string {
	return "specified transport ports could not be decoded"
}

// BadRequest denotes the type of this error
func (eitp *ErrInvalidTransportPortsOption) BadRequest() {}

// ErrInvalidGateway is returned when the user provided default gateway (v4/v6) is not not valid.
type ErrInvalidGateway struct{}

func (eig *ErrInvalidGateway) Error() string {
	return "default gateway ip must be part of the network"
}

// BadRequest denotes the type of this error
func (eig *ErrInvalidGateway) BadRequest() {}

// ErrInvalidMtu is returned when the user provided MTU is not valid.
type ErrInvalidMtu int

func (eim ErrInvalidMtu) Error() string {
	return fmt.Sprintf("invalid MTU number: %d", int(eim))
}

// BadRequest denotes the type of this error
func (eim ErrInvalidMtu) BadRequest() {}

// InvalidNetworkIDError is returned when the passed
// network id for an existing network is not a known id.
type InvalidNetworkIDError string

func (inie InvalidNetworkIDError) Error() string {
	return fmt.Sprintf("invalid network id %s", string(inie))
}

// NotFound denotes the type of this error
func (inie InvalidNetworkIDError) NotFound() {}

// InvalidEndpointIDError is returned when the passed
// endpoint id is not valid.
type InvalidEndpointIDError string

func (ieie InvalidEndpointIDError) Error() string {
	return fmt.Sprintf("invalid endpoint id: %s", string(ieie))
}

// BadRequest denotes the type of this error
func (ieie InvalidEndpointIDError) BadRequest() {}

// EndpointNotFoundError is returned when the no endpoint
// with the passed endpoint id is found.
type EndpointNotFoundError string

func (enfe EndpointNotFoundError) Error() string {
	return fmt.Sprintf("endpoint not found: %s", string(enfe))
}

// NotFound denotes the type of this error
func (enfe EndpointNotFoundError) NotFound() {}

// IPTableCfgError is returned when an unexpected ip tables configuration is entered
type IPTableCfgError string

func (name IPTableCfgError) Error() string {
	return fmt.Sprintf("unexpected request to set IP tables for interface: %s", string(name))
}

// BadRequest denotes the type of this error
func (name IPTableCfgError) BadRequest() {}

// ErrNoNetwork is returned if no network with the specified id exists
type ErrNoNetwork string

func (enn ErrNoNetwork) Error() string {
	return fmt.Sprintf("No network (%s) exists", string(enn))
}

// NotFound denotes the type of this error
func (enn ErrNoNetwork) NotFound() {}

// ErrEndpointExists is returned if more than one endpoint is added to the network
type ErrEndpointExists string

func (ee ErrEndpointExists) Error() string {
	return fmt.Sprintf("Endpoint (%s) already exists (Only one endpoint allowed)", string(ee))
}

// Forbidden denotes the type of this error
func (ee ErrEndpointExists) Forbidden() {}
