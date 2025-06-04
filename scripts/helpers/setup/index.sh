#!/usr/bin/env bash
# Sources all .sh scripts in this directory for easy single-line inclusion.
set -euo pipefail

CURRENT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CURRENT_SCRIPT="$(basename "${BASH_SOURCE[0]}")"

# Source each .sh file in this directory, excluding this index file
for curr_file in "${CURRENT_DIR}"/*.sh; do
    if [[ "$(basename "${curr_file}")" != "${CURRENT_SCRIPT}" ]]; then
        # shellcheck source=/dev/null
        source "${curr_file}"
    fi
done 