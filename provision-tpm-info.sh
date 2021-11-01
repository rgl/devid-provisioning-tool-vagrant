#!/bin/bash
source /vagrant/lib.sh

# echo the executed commands to stderr.
set -x

# install dependencies.
apt-get install -y libtspi-dev

# build.
cd /vagrant/tpm-info
CGO_ENABLED=0 go build -ldflags="-s" -o /usr/local/bin/tpm-info

# execute.
tpm-info
