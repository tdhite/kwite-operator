#!/bin/bash
# static.sh
#
# Copyright (c) 2019-2020 VMware, Inc.
#
# SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
#

set -e -x

# Save current directory
TOP="$(pwd)"

# Make the output area if it does not exist
mkdir -p ${TOP}/go-error-files

# change directories to the code
cd sources

# Test the code for formatting
find . -name \*.go | xargs gofmt -l | tee ${TOP}/go-error-files/gofmt.lst

# Check the results
if [ $(cat ${TOP}/go-error-files/gofmt.lst | wc -l) -ne 0 ]; then
    cat ${TOP}/go-error-files/gofmt.lst
    exit 1
fi

# check the ci pipeline scripts
for s in ci/scripts/*; do
	bash -n $s
done
