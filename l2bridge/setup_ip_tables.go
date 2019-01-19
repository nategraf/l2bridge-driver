package l2bridge

import (
	"errors"
	"fmt"
	"github.com/docker/libnetwork/iptables"
)

func (n *bridgeNetwork) setupIPTables(config *networkConfiguration, i *bridgeInterface) error {
	d := n.driver
	d.Lock()
	driverConfig := d.config
	d.Unlock()

	// Sanity check.
	if driverConfig.EnableIPTables == false {
		return errors.New("cannot program chains, EnableIPTable is disabled")
	}

	if err := setLocalForwarding(config.BridgeName, true); err != nil {
		return fmt.Errorf("failed to setup IP tables: %v", err)
	}
	n.registerIptCleanFunc(func() error {
		return setLocalForwarding(config.BridgeName, false)
	})

	return nil
}

// setLocalForwarding add or removes a rule to allow traffic to pass through the bridge locally depending
// on whether enable is true or false respectivly.
func setLocalForwarding(bridgeIface string, enable bool) error {
	var (
		table = iptables.Filter
		chain = "FORWARD"
		rule  = []string{"-i", bridgeIface, "-o", bridgeIface, "-j", "ACCEPT"}
	)

	if enable {
		if err := iptables.ProgramRule(table, chain, iptables.Append, rule); err != nil {
			return fmt.Errorf("unable to setup bridge forwarding rule: %v", err)
		}
	} else {
		if err := iptables.ProgramRule(table, chain, iptables.Delete, rule); err != nil {
			return fmt.Errorf("unable to cleanup bridge forwarding rule: %v", err)
		}
	}
	return nil
}
