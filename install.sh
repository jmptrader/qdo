#!/bin/bash
set -e

# Require root user.
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root"
   exit 1
fi

# Build file must exist.
if [ ! -f "qdo" ]; then
    echo "Qdo not found, compile with: go build"
    exit 1
fi

# Copy binary.
if [ -f "/etc/init/qdo.conf" ]; then
    service qdo stop
fi
cp qdo /usr/bin/qdo

# Setup templates and static files.
if [ -d "/opt/qdo" ]; then
    rm -r /opt/qdo
fi
mkdir /opt/qdo
cp -r web/template /opt/qdo/
cp -r web/static /opt/qdo/

# Setup upstart script.
cp qdo.conf /etc/init/
initctl reload-configuration
