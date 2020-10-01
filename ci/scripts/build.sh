#!/bin/bash
# build.sh
#
# Copyright (c) 2019-2020 VMware, Inc.
#
# SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
#

set -e -x

# Save current directory
TOP="$(pwd)"
OUTPUT=${TOP}/build

# prepare for private repos
. ${TOP}/sources/ci/scripts/setup_private_repo.sh

# Show current setup
echo "GOPATH is: " $GOPATH
echo "TOP is: " $TOP
echo ""

# Assure output exists
mkdir -p ${OUTPUT}

# Build the beast
cd sources
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a --installsuffix cgo -o ${OUTPUT}/manager main.go

# Check static linked binary
echo "Check static link status:"
if ldd ${OUTPUT}/manager; then
    echo "The kwite opreator binary is dynamically linked, cannot use it."
    exit 1
fi
