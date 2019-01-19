package l2bridge

import (
	"errors"
	"fmt"
	"github.com/docker/libnetwork/iptables"
)

// TODO(nategraf) Look into creating a new chain to avoid clobbering the host environment.
func (n *bridgeNetwork) setupIPTables(config *networkConfiguration, i *bridgeInterface) error {
	d := n.driver
	d.Lock()
	driverConfig := d.config
	d.Unlock()

	// Sanity check.
	if driverConfig.EnableIPTables == false {
		return errors.New("Cannot program chains, EnableIPTable is disabled")
	}

	if err := setIcc(config.BridgeName, true); err != nil {
		return fmt.Errorf("Failed to Setup IP tables: %s", err.Error())
	}
	n.registerIptCleanFunc(func() error {
		return setIcc(config.BridgeName, false)
	})

	return nil
}

// setIcc add or removes a rule to allow traffic to pass through the bridge locally depending
// on whether enable is true or false respectivly.
func setIcc(bridgeIface string, insert bool) error {
	var (
		table = iptables.Filter
		chain = "FORWARD"
		rule  = []string{"-i", bridgeIface, "-o", bridgeIface, "-j", "ACCEPT"}
	)

	if insert {
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
