package l2bridge

import "github.com/docker/libnetwork/iptables"

func (n *bridgeNetwork) setupFirewalld(config *networkConfiguration, i *bridgeInterface) error {
	d := n.driver
	d.Lock()
	driverConfig := d.config
	d.Unlock()

	// Sanity check.
	if !driverConfig.EnableIPTables {
		return IPTableCfgError(config.BridgeName)
	}

	iptables.OnReloaded(func() { n.setupIPTables(config, i) })

	return nil
}
