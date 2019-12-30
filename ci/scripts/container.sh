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

# Copy build results and static content to the output
echo "Copy/create build artifacts on output"
cp -a ${TOP}/build/manager ${TOP}/container/

# Create the docker build file
cat >${TOP}/container/Dockerfile <<-EOF
	FROM gcr.io/distroless/static:nonroot

	LABEL MAINTAINER="thite@vmware.com" \
	      VERSION="${TAG}"

	WORKDIR /
	COPY manager .
	USER nonroot:nonroot

	ENTRYPOINT ["/manager"]
EOF

# List what got laid down
echo "List out the container directory"
ls -laRt ${TOP}/container
echo ""
