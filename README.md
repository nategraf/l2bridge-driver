# l2bridge Driver

Docker libnetwork remote driver for layer two aware networks, for when `bridge` is too smart for its own good.

The majority of the code in this code is appropriated from the [libnetwork bridge] driver itself, and the modifications
are largely reductive.

Features, compared to the standard bridge driver:
    * Overlapping ip subnets are permitted.
    * Bridge interface is assigned no IP addresses, keeping it at layer 2 and increasing security.
    * External interfaces may be attached without trouble.

This driver is written in support of my larger project [Naumachia]. Check it out!

[libnetwork bridge]: https://github.com/docker/libnetwork/tree/master/drivers/bridge
[Naumachia]: https://github.com/nategraf/Naumachia
