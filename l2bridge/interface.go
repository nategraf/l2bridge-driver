package l2bridge

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// Interface models the bridge network device.
type bridgeInterface struct {
	Link netlink.Link
	nlh  *netlink.Handle
}

// newInterface creates a new bridge interface structure. It attempts to find
// an already existing device identified by the configuration BridgeName field,
// or the default bridge name when unspecified, but doesn't attempt to create
// one when missing
func newInterface(nlh *netlink.Handle, config *networkConfiguration) (*bridgeInterface, error) {
	var err error
	i := &bridgeInterface{nlh: nlh}

	// Attempt to find an existing bridge named with the specified name.
	i.Link, err = nlh.LinkByName(config.BridgeName)
	if err != nil {
		logrus.Debugf("Did not find any interface with name %s: %v", config.BridgeName, err)
	} else if _, ok := i.Link.(*netlink.Bridge); !ok {
		return nil, fmt.Errorf("existing interface %s is not a bridge", i.Link.Attrs().Name)
	}
	return i, nil
}

// exists indicates if the existing bridge interface exists on the system.
func (i *bridgeInterface) exists() bool {
	return i.Link != nil
}

// addresses returns all IPv4 addresses and all IPv6 addresses for the bridge interface.
func (i *bridgeInterface) addresses() ([]netlink.Addr, []netlink.Addr, error) {
	v4addr, err := i.nlh.AddrList(i.Link, netlink.FAMILY_V4)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to retrieve V4 addresses: %v", err)
	}

	v6addr, err := i.nlh.AddrList(i.Link, netlink.FAMILY_V6)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to retrieve V6 addresses: %v", err)
	}

	if len(v4addr) == 0 {
		return nil, v6addr, nil
	}
	return v4addr, v6addr, nil
}
