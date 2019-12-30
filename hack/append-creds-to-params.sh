#!/usr/bin/env bash
# append-creds-to-params.sh
#
# Copyright (c) 2019-2020 VMware, Inc.
#
# SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
#
# This script may help adding credentials to a params.yml file modified for
# use in setting concourse pipelines for the go-reminders project.
#
# The script makes attempts to pull cred info out of ~/.kube/config and set
# appropriate values (appends to) the file params.yml. It also sets a market
# in place to let the caller know what to delete from params.yml if it is
# executed again.
#
# What that tells all is this is by no means an idempotent process, rather it's
# just enough script to try to help setting variables for pipeline setup, but
# your mileage may vary.
#
DEBUG=1

# From where this script is executing
RUNDIR="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"

# Set this to anything hat follows variables like "certificate-authority"
# For instance many time the kube config will use "certificate-authority-data".
# Minikube leaves no such postfix.
#CDATAPOSTFIX="-data"
CDATAPOSTFIX=${CDATAPOSTFIX:=""}

# Set this to the name of the user you want to perform k8s deployments.
K8SUSER="${K8SUSER:=minikube}"

# Set this to the cluster into which you want to deploy.
K8SCLUSTER="${K8SCLUSTER:=minikube}"

# set the configuration file to draw on for credentials
KUBECONFIG="${KUBECONFIG:=${HOME}/.kube/config}"

if [ -z "$(which base64)" ]; then
    echo "ERROR: base64 must be available on the PATH, but is not."
    exit 2
fi

echo get cert data...
CADATA=$(kubectl config view --raw -o jsonpath="{.clusters[?(@.name == \"${K8SCLUSTER}\")].cluster.certificate-authority${CDATAPOSTFIX}}")
CCDATA=$(kubectl config view --raw -o jsonpath="{.users[?(@.name == \"${K8SUSER}\")].user.client-certificate${CDATAPOSTFIX}}")
CKDATA=$(kubectl config view --raw -o jsonpath="{.users[?(@.name == \"${K8SUSER}\")].user.client-key${CDATAPOSTFIX}}")

echo get k8s host...
if [ -z "${K8SHOST}" ]; then
    K8SHOST=$(kubectl config view -o jsonpath="{.clusters[?(@.name == \"${K8SCLUSTER}\")].cluster.server}")
fi

# cleanup old certs
awk -f ${RUNDIR}/killcerts.awk params.yaml >newparams.yaml
mv newparams.yaml params.yaml

# Append all the things to params.yaml
echo "#### Added by ${0}. Delete this line and below to rerun the script." >>params.yaml
echo "####" >>params.yaml
echo "appending k8s-cluster-url..."
echo -n "k8s-cluster-url: " >>params.yaml
echo "${K8SHOST}" >>params.yaml
echo

if [ "$K8SCLUSTER" == "minikube" ]; then
    echo "appending k8s-cluster-ca..."
    echo -n "k8s-cluster-ca: " >>params.yaml
    cat "${CADATA}" | base64 >>params.yaml
    echo >>params.yaml

    echo "appending k8s-admin-cert..."
    echo -n "k8s-admin-cert: " >>params.yaml
    cat ${CCDATA} | base64 >>params.yaml
    echo >>params.yaml

    echo "appending k8s-admin-key..."
    echo -n "k8s-admin-key: " >>params.yaml
    cat ${CKDATA} | base64 >>params.yaml
    echo >>params.yaml
else
    echo "appending k8s-cluster-ca..."
    echo -n "k8s-cluster-ca: " >>params.yaml
    echo -n ${CADATA} >>params.yaml
    echo >>params.yaml

    echo "appending k8s-admin-cert..."
    echo -n "k8s-admin-cert: " >>params.yaml
    echo -n ${CCDATA} >>params.yaml
    echo >>params.yaml

    echo "appending k8s-admin-key..."
    echo -n "k8s-admin-key: " >>params.yaml
    echo -n ${CKDATA} >>params.yaml
    echo >>params.yaml
fi

# Finally: replace bogus tokens
sed -i -e 's/k8s-admin-token: null/k8s-admin-token: MINIKUBE/g' params.yaml
if [ -n "${K8SHOSTCNAME}" ]; then
	sed -i -e "s/127.0.0.1/${K8SHOSTCNAME}/g" params.yaml
fi
