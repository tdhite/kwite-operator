#!/bin/bash

#tdhite/kwite-operator:0.0.102
REGISTRY_USER=${REGISTRY_USER:=tdhite}
#TAG="${TAG:=0.0.$(git rev-parse --verify --short HEAD)}"
TAG="${TAG:=0.0.102}"
STARTDIR=$(pwd)

# won't run in the hack directory
if [ -f ../.gitignore ]; then
	cd ..
fi

TOP=$(pwd)

git checkout -- config/manager/kustomization.yaml
make manifests
if [ ! -d "${TOP}/release" ]; then
	mkdir ${TOP}/release
fi
pushd config/manager >/dev/null 2>&1
kustomize edit set image controller=${REGISTRY_USER}/kwite-operator:${TAG}
popd >/dev/null 2>&1
kustomize build config/default >${TOP}/release/kwite-operator.yaml

# cleanup
git checkout -- config/manager/kustomization.yaml
