#!/bin/sh

ip a
ip route
echo "Running ping $*"
exec ping "$@"
