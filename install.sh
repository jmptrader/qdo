#!/bin/bash

# Require root user.
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root"
   exit 1
fi

# Build file must exist.
if [ ! -f "cmd/qdo" ]; then
    echo "QDo not found, compile with: go build"
    exit 1
fi

# Copy binary.
if [ -f "/etc/init/qdo.conf" ]; then
    service qdo stop
fi
cp cmd/qdo /usr/bin/qdo

# Setup upstart script.
cp qdo.conf /etc/init/
initctl reload-configuration
