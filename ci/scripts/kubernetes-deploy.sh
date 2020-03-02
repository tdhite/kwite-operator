#!/bin/bash
# kubernetes-deploy.sh
#
# Copyright (c) 2019-2020 VMware, Inc.
#
# SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
#

set -x

# Save current directory
TOP="$(pwd)"
echo "PATH: ${PATH}"
PATH=${PATH:/usr/local/bin}
export PATH=${PATH}:${TOP}

# install kubectl
EXE=$(which kubectl)
if [ $? -ne 0 ]; then
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

# get the tag for the docker container
tag=$(cat version/version)

# validate parameters
ret=0
if [ -z "${cluster_ca}" ]; then
    echo "ERROR: cluster_ca not supplied. Aborting!"
    ret=1
fi
if [ -z "${cluster_url}" ]; then
    echo "ERROR: cluster_url not supplied. Aborting!"
    ret=1
fi
if [ -z "${namespace}" ]; then
    echo "ERROR: namespace not supplied. Aborting!"
    ret=1
fi
if [ -z "${container}" ]; then
    echo "ERROR: container not supplied. Aborting!"
    ret=1
fi
if [ -z "${admin_token}" ]; then
    echo "INFO: admin_token not supplied. Using certs for auth."
    if [ -z "${admin_key}" ]; then
        echo "ERROR: admin_key not supplied. Aborting!"
        ret=1
    fi
    if [ -z "${admin_cert}" ]; then
        echo "ERROR: admin_cert not supplied. Aborting!"
        ret=1
    fi
fi
if [ -z "${tag}" ]; then
    echo "ERROR: tag (version) not supplied. Aborting!"
    ret=1
fi
if [ $ret -ne 0 ]; then
    exit $ret
fi

echo "build credentials"

echo "$cluster_ca" | base64 -d > ca.pem

# if using a bearer token or minikube, admin keys are useless
if [ -z "$admin_token" -o "${admin_token}" == "MINIKUBE" ]; then
    echo "$admin_key" | base64 -d > key.pem
    echo "$admin_cert" | base64 -d > cert.pem
fi

# list the directory now for debugging purposes
ls -lat

# setup kube config
kubectl config set-cluster kwite --server=${cluster_url} --certificate-authority=${TOP}/ca.pem

# set kube user
kubectl config set-credentials kwite --client-key=${TOP}/key.pem --client-certificate=${TOP}/cert.pem

# enable the context
kubectl config set-context kwite --user=kwite --cluster=kwite
kubectl config use-context kwite

# check kubectl for validity
kubectl get all --all-namespaces
if [ $? -ne 0 ]; then
    echo "kubectl failed to connect to Kubernetes API server."
    exit 1
fi

# Kustomize the manifestst and deploy (echo first for debugging)
kubectl apply -f ${TOP}/kubernetes/kwite-op.yaml
