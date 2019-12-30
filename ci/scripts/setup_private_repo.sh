#!/bin/bash
# setup_private_repo.sh
#
# Copyright (c) 2019-2020 VMware, Inc.
#
# SPDX-License-Identifier: https://spdx.org/licenses/MIT.html
#

if [ -n "${github_private_repo}" ]; then
    if [ -z "${github_private_repo_user}" ]; then
        echo "No private user provided for private repo ${github_private_repo}. Aboring."
        exit 1
    fi

    export GONOSUMDB="${github_private_repo}"

    if [ -n "${github_private_api_key}" ]; then
        echo "machine github.com login ${github_private_repo_user} password ${github_private_api_key}" >~/.netrc
        chmod 600 ~/.netrc
    fi
fi
