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

# change directories to the code and run tests
cd sources
echo "Can't \"make test\" quite yet  -- need to build up a Concourse task image to support."
