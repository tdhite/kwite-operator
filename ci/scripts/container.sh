#!/bin/bash
# container.sh
#
# Copyright (c) 2019-2020 VMware, Inc.
#
# SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
#

set -e -x

# Save current directory
TOP="$(pwd)"

# get the tag for the container
TAG=$(cat version/version)

# Show current setup
echo "TOP is: " $TOP
echo ""

# Create the output directory tree if necessary
mkdir -p ${TOP}/container/etc

# Copy build results and static content to the output
echo "Copy/create build artifacts on output"
cp -a ${TOP}/build/manager ${TOP}/container/

# Create necessary files for scratch based container
cat >${TOP}/container/etc/hosts <<-EOF
	127.0.0.1	localhost
	::1		localhost
EOF

echo "hosts: files dns" >${TOP}/container/etc/nsswitch

echo "nobody:x:65534:65534:Nobody:/:/noshellexists" >${TOP}/container/etc/passwd

mkdir -p ${TOP}/container/tmp && chmod 777 ${TOP}/container/tmp && chmod +t ${TOP}/container/tmp

tar -C ${TOP}/container -cvzf ${TOP}/container/rootfs.tgz etc tmp

# Create the docker build file
cat >${TOP}/container/Dockerfile <<-EOF
	FROM scratch
	LABEL MAINTAINER="thite@vmware.com" \\
	      VERSION="${TAG}"
	WORKDIR /
	COPY --chown=65534:65534 manager /manager
	ADD rootfs.tgz /
	USER 65534:65534
	ENTRYPOINT ["/manager"]
EOF

# List what got laid down
echo "List out the container directory"
ls -laRt ${TOP}/container
echo ""
