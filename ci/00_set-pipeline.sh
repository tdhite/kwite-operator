#!/bin/bash
# 00_set_pipeline.sh
#
# Copyright (c) 2019-2020 VMware, Inc.
#
# SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
#

FLYTARGET=${FLYTARGET:=kwite}

PARAMS="${PARAMS:=params.yaml}"
PIPELINE="${PIPELINE:=pipeline.yaml}"
PIPELINE_NAME="${PIPELINE_NAME:=kwite-operator}"

if [ "$1" == "-force" ]; then
	ccargs="--non-interactive"
fi

# correct the params file for current cluster creds
if [ -x ../hack/append-creds-to-params.sh ]; then
	../hack/append-creds-to-params.sh >/dev/null
else
	echo "WARN: append-creds-to-params.sh not found, assuming params.yml is correct for auth."
fi

fly -t ${FLYTARGET} set-pipeline $ccargs --pipeline="${PIPELINE_NAME}" --load-vars-from="${PARAMS}" --config="${PIPELINE}"

echo 'Pipeline added -- you can unpause it (and start a run) with the following command:'
echo "	fly -t ${FLYTARGET} unpause-pipeline --pipeline ${PIPELINE_NAME}"
