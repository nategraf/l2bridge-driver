package label

const (
	// DockerBridgeName label to specify a network's bridge name, understood by l2bridge.
	DockerBridgeName = "com.docker.network.bridge.name"

	// BridgeName label to specify a networks bridge name.
	BridgeName = "l2bridge.name"

	// GatewayIPv4 label to specify a network's default gateway.
	GatewayIPv4 = "l2bridge.gateway"

	// GatewayIPv6 label to specify a network's IPv6 default gateway.
	GatewayIPv6 = "l2bridge.ipv6.gateway"
)
