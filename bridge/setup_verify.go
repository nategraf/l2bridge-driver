package bridge

import (
	"fmt"
	"strings"

	"github.com/docker/libnetwork/ns"
)

func bridgeInterfaceExists(name string) (bool, error) {
	nlh := ns.NlHandle()
	link, err := nlh.LinkByName(name)
	if err != nil {
		if strings.Contains(err.Error(), "Link not found") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check bridge interface existence: %v", err)
	}

	if link.Type() == "bridge" {
		return true, nil
	}
	return false, fmt.Errorf("existing interface %s is not a bridge", name)
}
