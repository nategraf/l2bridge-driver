package l2bridge

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/docker/libnetwork/netlabel"
	"github.com/docker/libnetwork/netutils"
	"github.com/docker/libnetwork/ns"
	"github.com/docker/libnetwork/options"
	"github.com/docker/libnetwork/osl"
	"github.com/docker/libnetwork/types"
	"github.com/nategraf/l2bridge-driver/label"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

const (
	networkType                = "l2bridge"
	vethPrefix                 = "veth"
	vethLen                    = 7
	defaultContainerVethPrefix = "eth"
	maxAllocatePortAttempts    = 10
)

const (
	// DefaultGatewayV4AuxKey represents the default-gateway configured by the user
	DefaultGatewayV4AuxKey = "DefaultGatewayIPv4"
	// DefaultGatewayV6AuxKey represents the ipv6 default-gateway configured by the user
	DefaultGatewayV6AuxKey = "DefaultGatewayIPv6"
)

type iptableCleanFunc func() error
type iptablesCleanFuncs []iptableCleanFunc

// Configuration info for the "bridge" driver.
type Configuration struct {
	EnableIPForwarding bool
	EnableIPTables     bool
}

// networkConfiguration for network specific configuration
type networkConfiguration struct {
	ID                   string
	BridgeName           string
	EnableIPv6           bool
	Mtu                  int
	ContainerIfacePrefix string
	// Internal fields set after ipam data parsing
	PoolIPv4           *net.IPNet
	PoolIPv6           *net.IPNet
	DefaultGatewayIPv4 net.IP
	DefaultGatewayIPv6 net.IP
	dbIndex            uint64
	dbExists           bool
}

// ifaceCreator represents how the bridge interface was created
type ifaceCreator int8

const (
	ifaceCreatorUnknown ifaceCreator = iota
	ifaceCreatorSelf
	ifaceCreatorExternal
)

// endpointConfiguration represents the user specified configuration for the sandbox endpoint
type endpointConfiguration struct {
	MacAddress net.HardwareAddr
}

type bridgeEndpoint struct {
	id           string
	nid          string
	srcName      string
	addr         *net.IPNet
	addrv6       *net.IPNet
	gatewayv4    net.IP
	gatewayv6    net.IP
	macAddress   net.HardwareAddr
	config       *endpointConfiguration // User specified parameters
	exposedPorts []types.TransportPort
	dbIndex      uint64
	dbExists     bool
}

type bridgeNetwork struct {
	id            string
	bridge        *bridgeInterface // The bridge's L3 interface
	config        *networkConfiguration
	endpoints     map[string]*bridgeEndpoint // key: endpoint id
	driver        *bridgeDriver              // The network's driver
	iptCleanFuncs iptablesCleanFuncs
	sync.Mutex
}

// TODO(nategraf) Consolidate this driver code (ripped from libnetwork/drivers) with the remote driver code.
type bridgeDriver struct {
	config        *Configuration
	network       *bridgeNetwork
	networks      map[string]*bridgeNetwork
	nlh           *netlink.Handle
	configNetwork sync.Mutex
	sync.Mutex
}

// NewBridgeDriver constructs a new bridge driver
func NewBridgeDriver(config *Configuration) *bridgeDriver {
	if config == nil {
		config = &Configuration{
			EnableIPForwarding: true,
			EnableIPTables:     true,
		}
	}
	return &bridgeDriver{networks: map[string]*bridgeNetwork{}, config: config}
}

// Validate performs a static validation on the network configuration parameters.
// Whatever can be assessed a priori before attempting any programming.
func (c *networkConfiguration) Validate() error {
	if c.Mtu < 0 {
		return ErrInvalidMtu(c.Mtu)
	}

	// If bridge v4 subnet is specified
	if c.PoolIPv4 != nil {
		// If default gw is specified, it must be part of bridge subnet
		if c.DefaultGatewayIPv4 != nil {
			if !c.PoolIPv4.Contains(c.DefaultGatewayIPv4) {
				return &ErrInvalidGateway{}
			}
		}
	}

	// If default v6 gw is specified, PoolIPv6 must be specified and gw must belong to PoolIPv6 subnet
	if c.EnableIPv6 && c.DefaultGatewayIPv6 != nil {
		if c.PoolIPv6 == nil || !c.PoolIPv6.Contains(c.DefaultGatewayIPv6) {
			return &ErrInvalidGateway{}
		}
	}
	return nil
}

func (c *networkConfiguration) fromLabels(labels map[string]interface{}) error {
	var err error
	for key, value := range labels {
		switch key {
		case label.DockerBridgeName, label.BridgeName:
			switch name := value.(type) {
			case string:
				c.BridgeName = name
			default:
				return fmt.Errorf("unrecognized type for %s: %T", key, name)
			}
		case label.GatewayIPv4:
			logrus.Infof("GOT %s = %v", key, value)
			switch gateway := value.(type) {
			case string:
				c.DefaultGatewayIPv4 = net.ParseIP(gateway)
				if c.DefaultGatewayIPv4 == nil {
					return fmt.Errorf("failed to parse %s: %v is not a valid IPv4 address", label.GatewayIPv4, gateway)
				}
			case net.IP:
				c.DefaultGatewayIPv4 = gateway
			default:
				return fmt.Errorf("unrecognized type for %s: %T", key, gateway)
			}
		case label.GatewayIPv6:
			switch gateway := value.(type) {
			case string:
				c.DefaultGatewayIPv6 = net.ParseIP(gateway)
				if c.DefaultGatewayIPv6 == nil {
					return fmt.Errorf("failed to parse %s: %v is not a valid IPv6 address", label.GatewayIPv6, gateway)
				}
			case net.IP:
				c.DefaultGatewayIPv6 = gateway
			default:
				return fmt.Errorf("unrecognized type for %s: %T", key, gateway)
			}
		case netlabel.DriverMTU:
			switch mtu := value.(type) {
			case int:
				c.Mtu = mtu
			case string:
				if c.Mtu, err = strconv.Atoi(mtu); err != nil {
					return parseErr(key, mtu, err.Error())
				}
			default:
				return fmt.Errorf("unrecognized type for %s: %T", key, mtu)
			}
		case netlabel.EnableIPv6:
			switch enable := value.(type) {
			case bool:
				c.EnableIPv6 = enable
			case string:
				if c.EnableIPv6, err = strconv.ParseBool(enable); err != nil {
					return parseErr(key, enable, err.Error())
				}
			default:
				return fmt.Errorf("unrecognized type for %s: %T", key, enable)
			}
		case netlabel.ContainerIfacePrefix:
			switch prefix := value.(type) {
			case string:
				c.ContainerIfacePrefix = prefix
			default:
				return fmt.Errorf("unrecognized type for %s: %T", key, prefix)
			}
		default:
			logrus.Warnf("Ignoring unrecognized configuration option %s: %v", key, value)
		}
	}

	return nil
}

func parseErr(key, value, errString string) error {
	return types.BadRequestErrorf("failed to parse %s value: %v (%s)", key, value, errString)
}

func (n *bridgeNetwork) registerIptCleanFunc(clean iptableCleanFunc) {
	n.iptCleanFuncs = append(n.iptCleanFuncs, clean)
}

func (n *bridgeNetwork) getNetworkBridgeName() string {
	n.Lock()
	config := n.config
	n.Unlock()

	return config.BridgeName
}

func (n *bridgeNetwork) getEndpoint(eid string) (*bridgeEndpoint, error) {
	n.Lock()
	defer n.Unlock()

	if eid == "" {
		return nil, InvalidEndpointIDError(eid)
	}

	if ep, ok := n.endpoints[eid]; ok {
		return ep, nil
	}
	return nil, nil
}

func (d *bridgeDriver) configure(option map[string]interface{}) error {
	genericData, ok := option[netlabel.GenericData]
	if !ok || genericData == nil {
		return nil
	}

	var config *Configuration
	switch opt := genericData.(type) {
	case options.Generic:
		opaqueConfig, err := options.GenerateFromModel(opt, &Configuration{})
		if err != nil {
			return err
		}
		config = opaqueConfig.(*Configuration)
	case *Configuration:
		config = opt
	default:
		return &ErrInvalidDriverConfig{}
	}

	if config.EnableIPTables {
		if _, err := os.Stat("/proc/sys/net/bridge"); err != nil {
			if out, err := exec.Command("modprobe", "-va", "bridge", "br_netfilter").CombinedOutput(); err != nil {
				logrus.WithError(err).Warnf("Running modprobe bridge br_netfilter failed with message: %s, error: %v", out, err)
			}
		}
	}

	if config.EnableIPForwarding {
		if err := setupIPForwarding(config.EnableIPTables); err != nil {
			logrus.WithError(err).Warnf("Failed to setup IP forwarding: ", err)
			return err
		}
	}

	d.Lock()
	d.config = config
	d.Unlock()

	// TODO(nategraf) Implement storage.
	//if err := d.initStore(option); err != nil {
	//	return err
	//}

	return nil
}

func (d *bridgeDriver) getNetwork(id string) (*bridgeNetwork, error) {
	if id == "" {
		return nil, types.BadRequestErrorf("invalid network id: %s", id)
	}

	d.Lock()
	n, ok := d.networks[id]
	d.Unlock()

	if !ok {
		return nil, types.NotFoundErrorf("network %s does not exist", id)
	}
	if n == nil {
		return nil, ErrNoNetwork(id)
	}

	// Sanity check
	n.Lock()
	eq := n.id == id
	n.Unlock()
	if !eq {
		return nil, InvalidNetworkIDError(id)
	}
	return n, nil
}

func parseNetworkGenericOptions(data interface{}) (*networkConfiguration, error) {
	var (
		err    error
		config *networkConfiguration
	)

	switch opt := data.(type) {
	case *networkConfiguration:
		config = opt
	case map[string]interface{}:
		config = &networkConfiguration{}
		err = config.fromLabels(opt)
	case options.Generic:
		var opaqueConfig interface{}
		if opaqueConfig, err = options.GenerateFromModel(opt, config); err == nil {
			config = opaqueConfig.(*networkConfiguration)
		}
	default:
		err = types.BadRequestErrorf("do not recognize network configuration format: %T", opt)
	}

	return config, err
}

func (c *networkConfiguration) processIPAM(id string, ipamV4Data, ipamV6Data []*IPAMData) error {
	if len(ipamV4Data) > 1 || len(ipamV6Data) > 1 {
		return types.ForbiddenErrorf("l2bridge driver doesn't support multiple subnets")
	}

	if len(ipamV4Data) == 0 || ipamV4Data[0].Pool == nil {
		return types.BadRequestErrorf("l2bridge network %s requires ipv4 configuration", id)
	}

	c.PoolIPv4 = types.GetIPNetCopy(ipamV4Data[0].Pool)
	if gw, ok := ipamV4Data[0].AuxAddresses[DefaultGatewayV4AuxKey]; ok {
		c.DefaultGatewayIPv4 = gw.IP
	}

	if len(ipamV6Data) > 0 {
		c.PoolIPv6 = ipamV6Data[0].Pool
		if gw, ok := ipamV6Data[0].AuxAddresses[DefaultGatewayV6AuxKey]; ok {
			c.DefaultGatewayIPv6 = gw.IP
		}
	}

	return nil
}

func parseNetworkOptions(id string, option options.Generic) (*networkConfiguration, error) {
	var (
		err    error
		config = &networkConfiguration{}
	)

	// Parse generic label first, config will be re-assigned
	if genData, ok := option[netlabel.GenericData]; ok && genData != nil {
		if config, err = parseNetworkGenericOptions(genData); err != nil {
			return nil, err
		}
	}

	// Process well-known labels next
	if val, ok := option[netlabel.EnableIPv6]; ok {
		config.EnableIPv6 = val.(bool)
	}

	// Finally validate the configuration
	if err = config.Validate(); err != nil {
		return nil, err
	}

	if config.BridgeName == "" {
		config.BridgeName = "br-" + id[:12]
	}

	exists, err := bridgeInterfaceExists(config.BridgeName)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, types.ForbiddenErrorf("interface with name %s exists", config.BridgeName)
	}

	config.ID = id
	return config, nil
}

// Return a slice of networks over which caller can iterate safely
func (d *bridgeDriver) getNetworks() []*bridgeNetwork {
	d.Lock()
	defer d.Unlock()

	ls := make([]*bridgeNetwork, 0, len(d.networks))
	for _, nw := range d.networks {
		ls = append(ls, nw)
	}
	return ls
}

// Create a new L2 Bridge network, including creating and performing inital setup on the bridge interface.
func (d *bridgeDriver) CreateNetwork(id string, option map[string]interface{}, ipV4Data, ipV6Data []*IPAMData) error {
	if len(ipV4Data) == 0 || ipV4Data[0].Pool.String() == "0.0.0.0/0" {
		return types.BadRequestErrorf("ipv4 pool is empty")
	}
	// Sanity checks
	d.Lock()
	if _, ok := d.networks[id]; ok {
		d.Unlock()
		return types.ForbiddenErrorf("network %s exists", id)
	}
	d.Unlock()

	// Parse and validate the config. It should not be conflict with existing networks' config
	config, err := parseNetworkOptions(id, option)
	if err != nil {
		return err
	}

	if err = config.processIPAM(id, ipV4Data, ipV6Data); err != nil {
		return err
	}

	// start the critical section, from this point onward we are dealing with the list of networks
	// so to be consistent we cannot allow that the list changes
	d.configNetwork.Lock()
	defer d.configNetwork.Unlock()
	if err = d.createNetwork(config); err != nil {
		return err
	}

	return nil //d.storeUpdate(config)
}

func (d *bridgeDriver) createNetwork(config *networkConfiguration) (err error) {
	defer osl.InitOSContext()()

	// Initialize handle when needed
	d.Lock()
	if d.nlh == nil {
		d.nlh = ns.NlHandle()
	}
	d.Unlock()

	// Create or retrieve the bridge L3 interface
	bridgeIface, err := newInterface(d.nlh, config)
	if err != nil {
		return err
	}

	// Create and set network handler in driver
	network := &bridgeNetwork{
		id:        config.ID,
		endpoints: make(map[string]*bridgeEndpoint),
		config:    config,
		bridge:    bridgeIface,
		driver:    d,
	}

	d.Lock()
	d.networks[config.ID] = network
	d.Unlock()

	// On failure make sure to reset driver network handler to nil
	defer func() {
		if err != nil {
			d.Lock()
			delete(d.networks, config.ID)
			d.Unlock()
		}
	}()

	// Prepare the bridge setup configuration
	bridgeSetup := newBridgeSetup(config, bridgeIface)

	// If the bridge interface doesn't exist, create a new device.
	if !bridgeIface.exists() {
		bridgeSetup.queueStep(setupDevice)
	}

	// Prevent the bridge from obtaining an IPv6 address.
	bridgeSetup.queueStep(setupDisableIPv6)

	if d.config.EnableIPTables {
		// Setup IPTables.
		bridgeSetup.queueStep(network.setupIPTables)

		//We want to track firewalld configuration so that
		//if it is started/reloaded, the rules can be applied correctly
		bridgeSetup.queueStep(network.setupFirewalld)
	}

	// Apply the prepared list of steps, and abort at the first error.
	bridgeSetup.queueStep(setupDeviceUp)
	return bridgeSetup.apply()
}

func (d *bridgeDriver) DeleteNetwork(nid string) error {

	d.configNetwork.Lock()
	defer d.configNetwork.Unlock()

	return d.deleteNetwork(nid)
}

func (d *bridgeDriver) deleteNetwork(nid string) error {
	var err error

	defer osl.InitOSContext()()
	// Get network handler and remove it from driver
	d.Lock()
	n, ok := d.networks[nid]
	d.Unlock()

	if !ok {
		return types.InternalMaskableErrorf("network %s does not exist", nid)
	}

	n.Lock()
	config := n.config
	n.Unlock()

	// delete endpoints belong to this network
	for _, ep := range n.endpoints {
		if link, err := d.nlh.LinkByName(ep.srcName); err == nil {
			if err := d.nlh.LinkDel(link); err != nil {
				logrus.WithError(err).Errorf("Failed to delete interface (%s)'s link on endpoint (%s) delete", ep.srcName, ep.id)
			}
		}

		// TODO(nategraf) Implement storage.
		//if err := d.storeDelete(ep); err != nil {
		//	logrus.Warnf("Failed to remove bridge endpoint %.7s from store: %v", ep.id, err)
		//}
	}

	d.Lock()
	delete(d.networks, nid)
	d.Unlock()

	// On failure set network handler back in driver, but
	// only if is not already taken over by some other thread
	defer func() {
		if err != nil {
			d.Lock()
			if _, ok := d.networks[nid]; !ok {
				d.networks[nid] = n
			}
			d.Unlock()
		}
	}()

	if err := d.nlh.LinkDel(n.bridge.Link); err != nil {
		logrus.WithError(err).Warnf("Failed to remove bridge interface %s on network %s delete: %v", config.BridgeName, nid, err)
	}

	for _, cleanFunc := range n.iptCleanFuncs {
		if err := cleanFunc(); err != nil {
			logrus.WithError(err).Warnf("Failed to clean iptables rules for bridge network: %v", err)
		}
	}

	// TODO(nategraf) Implement storage.
	return nil // d.storeDelete(config)
}

func addToBridge(nlh *netlink.Handle, ifaceName, bridgeName string) error {
	link, err := nlh.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("could not find interface %s: %v", ifaceName, err)
	}
	bridge := &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: bridgeName}}
	if err = nlh.LinkSetMaster(link, bridge); err != nil {
		return err
	}
	return nil
}

func setHairpinMode(nlh *netlink.Handle, link netlink.Link, enable bool) error {
	err := nlh.LinkSetHairpin(link, enable)
	if err != nil && err != syscall.EINVAL {
		// If error is not EINVAL something else went wrong, bail out right away
		return fmt.Errorf("unable to set hairpin mode on %s via netlink: %v",
			link.Attrs().Name, err)
	}

	// Hairpin mode successfully set up
	if err == nil {
		return nil
	}

	// The netlink method failed with EINVAL which is probably because of an older
	// kernel. Try one more time via the sysfs method.
	path := filepath.Join("/sys/class/net", link.Attrs().Name, "brport/hairpin_mode")

	var val []byte
	if enable {
		val = []byte{'1', '\n'}
	} else {
		val = []byte{'0', '\n'}
	}

	if err := ioutil.WriteFile(path, val, 0644); err != nil {
		return fmt.Errorf("unable to set hairpin mode on %s via sysfs: %v", link.Attrs().Name, err)
	}

	return nil
}

// CreateEndpoint makes a new link to be added to a container.
// Any fields set in the returned EndpointInterface will be understood as change requests by the Docker daemon.
func (d *bridgeDriver) CreateEndpoint(nid, eid string, ei *EndpointInterface, epOptions map[string]interface{}) (*EndpointInterface, error) {
	defer osl.InitOSContext()()

	if ei == nil {
		return nil, errors.New("invalid interface info")
	}

	n, err := d.getNetwork(nid)
	if err != nil {
		return nil, err
	}

	// Check if endpoint id is good and retrieve correspondent endpoint
	ep, err := n.getEndpoint(eid)
	if err != nil {
		return nil, err
	}
	// Endpoint with that id exists either on desired or other sandbox
	if ep != nil {
		return nil, ErrEndpointExists(eid)
	}

	// Try to convert the options to endpoint configuration
	epConfig, err := parseEndpointOptions(epOptions)
	if err != nil {
		return nil, err
	}

	// Create and add the endpoint
	n.Lock()
	endpoint := &bridgeEndpoint{id: eid, nid: nid, config: epConfig}
	n.endpoints[eid] = endpoint
	n.Unlock()

	// On failure make sure to remove the endpoint
	defer func() {
		if err != nil {
			n.Lock()
			delete(n.endpoints, eid)
			n.Unlock()
		}
	}()

	// Generate a name for what will be the host side pipe interface
	hostIfName, err := netutils.GenerateIfaceName(d.nlh, vethPrefix, vethLen)
	if err != nil {
		return nil, err
	}

	// Generate a name for what will be the sandbox side pipe interface
	containerIfName, err := netutils.GenerateIfaceName(d.nlh, vethPrefix, vethLen)
	if err != nil {
		return nil, err
	}

	// Generate and add the interface pipe host <-> sandbox
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: hostIfName, TxQLen: 0},
		PeerName:  containerIfName}
	if err = d.nlh.LinkAdd(veth); err != nil {
		return nil, types.InternalErrorf("failed to add the host (%s) <=> sandbox (%s) pair interfaces: %v", hostIfName, containerIfName, err)
	}

	// Get the host side pipe interface handler
	host, err := d.nlh.LinkByName(hostIfName)
	if err != nil {
		return nil, types.InternalErrorf("failed to find host side interface %s: %v", hostIfName, err)
	}
	defer func() {
		if err != nil {
			if err := d.nlh.LinkDel(host); err != nil {
				logrus.WithError(err).Warnf("Failed to delete host side interface (%s)'s link", hostIfName)
			}
		}
	}()

	// Get the sandbox side pipe interface handler
	sbox, err := d.nlh.LinkByName(containerIfName)
	if err != nil {
		return nil, types.InternalErrorf("failed to find sandbox side interface %s: %v", containerIfName, err)
	}
	defer func() {
		if err != nil {
			if err := d.nlh.LinkDel(sbox); err != nil {
				logrus.WithError(err).Warnf("Failed to delete sandbox side interface (%s)'s link", containerIfName)
			}
		}
	}()

	n.Lock()
	config := n.config
	n.Unlock()

	// Add bridge inherited attributes to pipe interfaces
	if config.Mtu != 0 {
		err = d.nlh.LinkSetMTU(host, config.Mtu)
		if err != nil {
			return nil, types.InternalErrorf("failed to set MTU on host interface %s: %v", hostIfName, err)
		}
		err = d.nlh.LinkSetMTU(sbox, config.Mtu)
		if err != nil {
			return nil, types.InternalErrorf("failed to set MTU on sandbox interface %s: %v", containerIfName, err)
		}
	}

	// Attach host side pipe interface into the bridge
	if err = addToBridge(d.nlh, hostIfName, config.BridgeName); err != nil {
		return nil, fmt.Errorf("adding interface %s to bridge %s failed: %v", hostIfName, config.BridgeName, err)
	}

	// Allow packets to enter and leave the same (bridge) interface.
	err = setHairpinMode(d.nlh, host, true)
	if err != nil {
		return nil, err
	}

	// Store the sandbox side pipe interface parameters
	endpoint.srcName = containerIfName
	endpoint.macAddress = ei.MacAddress
	endpoint.addr = ei.Address
	endpoint.addrv6 = ei.AddressIPv6

	// Set default gateway info if this endpoint is not the networks gatway.
	if gw := n.config.DefaultGatewayIPv4; gw != nil && !gw.Equal(endpoint.addr.IP) {
		endpoint.gatewayv4 = gw
	}
	if gw := n.config.DefaultGatewayIPv6; gw != nil && !gw.Equal(endpoint.addr.IP) {
		endpoint.gatewayv6 = gw
	}

	// Set the sbox's MAC if not provided. If specified, use the one configured by user, otherwise generate one based on IP.
	eiOut := &EndpointInterface{}
	if endpoint.macAddress == nil {
		endpoint.macAddress = netutils.GenerateRandomMAC()
		eiOut.MacAddress = endpoint.macAddress
	}

	// Up the host interface after finishing all netlink configuration
	if err = d.nlh.LinkSetUp(host); err != nil {
		return nil, fmt.Errorf("could not set link up for host interface %s: %v", hostIfName, err)
	}

	if endpoint.addrv6 == nil && config.EnableIPv6 {
		var ip6 net.IP
		network := n.config.PoolIPv6
		if config.PoolIPv6 != nil {
			network = config.PoolIPv6
		}

		ones, _ := network.Mask.Size()
		if ones > 80 {
			err = types.ForbiddenErrorf("Cannot self generate an IPv6 address on network %v: At least 48 host bits are needed.", network)
			return nil, err
		}

		ip6 = make(net.IP, len(network.IP))
		copy(ip6, network.IP)
		for i, h := range endpoint.macAddress {
			ip6[i+10] = h
		}

		endpoint.addrv6 = &net.IPNet{IP: ip6, Mask: network.Mask}
		eiOut.AddressIPv6 = endpoint.addrv6
	}

	// TODO(nategraf) Implement storage.
	//if err = d.storeUpdate(endpoint); err != nil {
	//	return nil, fmt.Errorf("failed to save bridge endpoint %.7s to store: %v", endpoint.id, err)
	//}

	return eiOut, nil
}

func (d *bridgeDriver) DeleteEndpoint(nid, eid string) error {
	var err error

	defer osl.InitOSContext()()

	n, err := d.getNetwork(nid)
	if err != nil {
		return err
	}

	// Check endpoint id and if an endpoint is actually there
	ep, err := n.getEndpoint(eid)
	if err != nil {
		return err
	}
	if ep == nil {
		return EndpointNotFoundError(eid)
	}

	// Remove it
	n.Lock()
	delete(n.endpoints, eid)
	n.Unlock()

	// On failure make sure to set back ep in n.endpoints, but only
	// if it hasn't been taken over already by some other thread.
	defer func() {
		if err != nil {
			n.Lock()
			if _, ok := n.endpoints[eid]; !ok {
				n.endpoints[eid] = ep
			}
			n.Unlock()
		}
	}()

	// Try removal of link. Discard error: it is a best effort.
	// Also make sure defer does not see this error either.
	if link, err := d.nlh.LinkByName(ep.srcName); err == nil {
		if err := d.nlh.LinkDel(link); err != nil {
			logrus.WithError(err).Errorf("Failed to delete interface (%s)'s link on endpoint (%s) delete", ep.srcName, ep.id)
		}
	}

	// TODO(nategraf) Implement storage.
	//if err := d.storeDelete(ep); err != nil {
	//	logrus.Warnf("Failed to remove bridge endpoint %.7s from store: %v", ep.id, err)
	//}

	return nil
}

// EndpointInfo returns useful data about an endpoint such as mac address and exposed ports.
func (d *bridgeDriver) EndpointInfo(nid, eid string) (map[string]string, error) {
	n, err := d.getNetwork(nid)
	if err != nil {
		return nil, err
	}

	// Check if endpoint id is good and retrieve correspondent endpoint
	ep, err := n.getEndpoint(eid)
	if err != nil {
		return nil, err
	}
	if ep == nil {
		return nil, EndpointNotFoundError(eid)
	}

	m := make(map[string]string)

	if ep.exposedPorts != nil {
		// Return a copy of the config data
		strs := make([]string, 0, len(ep.exposedPorts))
		for _, tp := range ep.exposedPorts {
			strs = append(strs, tp.String())
		}
		m[netlabel.ExposedPorts] = strings.Join(strs, ",")
	}

	if ep.macAddress != nil {
		m[netlabel.MacAddress] = ep.macAddress.String()
	}

	if ep.gatewayv4 != nil {
		m[netlabel.Gateway] = ep.gatewayv4.String()
	} else if ep.gatewayv6 != nil {
		m[netlabel.Gateway] = ep.gatewayv6.String()
	}

	return m, nil
}

// Join method is invoked when a Sandbox is attached to an endpoint.
func (d *bridgeDriver) Join(nid, eid, sboxKey string, opts map[string]interface{}) (*JoinResponse, error) {
	defer osl.InitOSContext()()

	network, err := d.getNetwork(nid)
	if err != nil {
		return nil, err
	}
	endpoint, err := network.getEndpoint(eid)
	if err != nil {
		return nil, err
	}
	if endpoint == nil {
		return nil, EndpointNotFoundError(eid)
	}

	containerVethPrefix := defaultContainerVethPrefix
	if network.config.ContainerIfacePrefix != "" {
		containerVethPrefix = network.config.ContainerIfacePrefix
	}

	if value, ok := opts[netlabel.ExposedPorts]; ok {
		ports, err := parseTransportPorts(value)
		if err == nil {
			endpoint.exposedPorts = ports
		} else {
			logrus.WithError(err).Warnf("parsing of %s failed: %v", netlabel.ExposedPorts, err)
		}
	}

	return &JoinResponse{
		InterfaceName: InterfaceName{
			SrcName:   endpoint.srcName,
			DstPrefix: containerVethPrefix,
		},
		Gateway:     endpoint.gatewayv4,
		GatewayIPv6: endpoint.gatewayv6,
		// Prevent Docker from creating a default gateway for us.
		DisableGatewayService: true,
	}, nil
}

// Leave method is invoked when a Sandbox detaches from an endpoint.
// Currently this is just a couple sanity checks to better report errors.
func (d *bridgeDriver) Leave(nid, eid string) error {
	defer osl.InitOSContext()()

	network, err := d.getNetwork(nid)
	if err != nil {
		return types.InternalMaskableErrorf("%s", err)
	}

	endpoint, err := network.getEndpoint(eid)
	if err != nil {
		return err
	}
	if endpoint == nil {
		return EndpointNotFoundError(eid)
	}

	return nil
}

func parseEndpointOptions(epOptions map[string]interface{}) (*endpointConfiguration, error) {
	if epOptions == nil {
		return nil, nil
	}

	ec := &endpointConfiguration{}

	if opt, ok := epOptions[netlabel.MacAddress]; ok {
		if mac, ok := opt.(net.HardwareAddr); ok {
			ec.MacAddress = mac
		} else {
			return nil, &ErrInvalidEndpointConfig{}
		}
	}

	return ec, nil
}

// parseTransportPorts unpacks the opaque transport ports array passed by libnetwork.
func parseTransportPorts(in interface{}) ([]types.TransportPort, error) {
	slice, ok := in.([]interface{})
	if !ok {
		return nil, &ErrInvalidTransportPortsOption{}
	}

	var out []types.TransportPort
	for _, value := range slice {
		dict, ok := value.(map[string]interface{})
		if !ok {
			return nil, &ErrInvalidTransportPortsOption{}
		}

		var tp types.TransportPort
		if proto, ok := dict["Proto"]; ok {
			switch x := proto.(type) {
			case float64:
				tp.Proto = types.Protocol(x)
			default:
				return nil, &ErrInvalidTransportPortsOption{}
			}
		} else {
			return nil, &ErrInvalidTransportPortsOption{}
		}

		if port, ok := dict["Port"]; ok {
			switch x := port.(type) {
			case float64:
				tp.Port = uint16(x)
			default:
				return nil, &ErrInvalidTransportPortsOption{}
			}
		} else {
			return nil, &ErrInvalidTransportPortsOption{}
		}
		out = append(out, tp)
	}
	return out, nil
}
