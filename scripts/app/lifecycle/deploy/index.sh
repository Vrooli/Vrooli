#!/usr/bin/env bash
# Sources all .sh scripts in this directory for easy single-line inclusion.
set -euo pipefail

APP_LIFECYCLE_DEPLOY_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_DIR}/../../../lib/utils/var.sh"

CURRENT_SCRIPT="$(basename "${BASH_SOURCE[0]}")"

# Source each .sh file in this directory, excluding this index file
for curr_file in "${APP_LIFECYCLE_DEPLOY_DIR}"/*.sh; do
    if [[ "$(basename "${curr_file}")" != "${CURRENT_SCRIPT}" ]]; then
        # shellcheck disable=SC1091
        source "${curr_file}"
    fi
done 