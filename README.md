# l2bridge Driver

Docker libnetwork remote driver for layer two aware networks, for when `bridge` is too smart for its own good.

The majority of the code in this code is appropriated from the [libnetwork bridge] driver itself, and the modifications
are largely reductive.

Features, compared to the standard bridge driver:
  * Overlapping ip subnets are permitted.
  * Bridge interface is assigned no IP addresses, keeping it at layer 2 and increasing security.
  * External interfaces may be attached without trouble.

This driver is written in support of my larger project [Naumachia]. Check it out!

## Installation as a service with SysV (Debian/Ubuntu)
```bash
# Download the service script and install it to init.d
sudo curl -L https://raw.githubusercontent.com/nategraf/l2bridge-driver/master/sysv.sh -o /etc/init.d/l2bridge
sudo chmod +x /etc/init.d/l2bridge

# Download the driver to usr/local/bin
sudo curl -L https://github.com/nategraf/l2bridge-driver/releases/latest/download/l2bridge-driver.linux.amd64 -o /usr/local/bin/l2bridge
sudo chmod +x /usr/local/bin/l2bridge

# Activate the service
sudo update-rc.d l2bridge defaults
sudo service l2bridge start

# Verify that it is running
sudo stat /run/docker/plugins/l2bridge.sock
#  File: /run/docker/plugins/l2bridge.sock
#  Size: 0               Blocks: 0          IO Block: 4096   socket
```

[libnetwork bridge]: https://github.com/docker/libnetwork/tree/master/drivers/bridge
[Naumachia]: https://github.com/nategraf/Naumachia
