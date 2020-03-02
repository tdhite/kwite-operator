#!/bin/bash
# deployment-files.sh
#
# Copyright (c) 2019-2020 VMware, Inc.
#
# SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
#

set -e -x

echo "------Environment Variables------"
set

# Save current directory
TOP="$(pwd)"
OUTPUT=${TOP}/kubernetes
PATH=${PATH:/usr/local/bin}
export PATH=${PATH}:${TOP}

ret=0
if [ -z "${container}" ]; then
    echo "ERROR: container not supplied. Aborting!"
    ret=1
fi
if [ -z "${deployenv}" ]; then
    echo "ERROR: deploy (kustomize) environmnt not supplied. Aborting!"
    ret=1
fi
if [ $ret -ne 0 ]; then
    exit $ret
fi

tag="$(cat version/version)"

# Assure the output area exists
mkdir -p ${OUTPUT}

# install kubectl
set +e
which kubectl
ret=$?
set -e
if [ $ret -ne 0 ]; then
    LATEST="$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)"
    if [ $? -ne 0 ]; then
        echo "Failed to obtain latest kubectl release. Aborting!"
        exit 1
    fi

    curl -LO https://storage.googleapis.com/kubernetes-release/release/${LATEST}/bin/linux/amd64/kubectl
    if [ $? -ne 0 ]; then
        echo "Failed to download kubectl. Aborting!"
        exit 1
    fi
    chmod +x kubectl
fi

# install kustomize
set +e
which kustomize
ret=$?
set -e
if [ $ret -ne 0 ]; then
    #
    # see : https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh
    #
    opsys=linux
    curl -s https://api.github.com/repos/kubernetes-sigs/kustomize/releases |\
    grep browser_download |\
    grep $opsys |\
    cut -d '"' -f 4 |\
    grep /kustomize/v |\
    sort | tail -n 1 |\
    xargs curl -s -O -L
    tar xzf ./kustomize_v*_${opsys}_amd64.tar.gz
    chmod +x kustomize
fi

# prepare for private repos
. ${TOP}/sources/ci/scripts/setup_private_repo.sh

# create the kubernetes deployment manifest
cd ${TOP}/sources && make manifests
cd ${TOP}/sources/config/manager && kustomize edit set image controller=${container}:${tag}
cd ${TOP}/sources && kubectl kustomize config/${deployenv} >${OUTPUT}/kwite-op.yaml

# Check what's here
echo "List out the output directory:"
ls -laRt ${OUTPUT}
echo ""
