#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
REPO_ROOT="$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)"
# Construct CLI_DIR and validate
CLI_DIR="${REPO_ROOT}/scenarios/local-info-scout/cli"
if [ ! -d "${CLI_DIR}" ]; then
    echo "Error: CLI_DIR '${CLI_DIR}' is not a valid directory" >&2
    exit 1
fi

# Source installation utility
if [ ! -f "${REPO_ROOT}/scripts/lib/utils/cli-install.sh" ]; then
    echo "Error: cli-install.sh not found at ${REPO_ROOT}/scripts/lib/utils/cli-install.sh" >&2
    exit 1
fi
source "${REPO_ROOT}/scripts/lib/utils/cli-install.sh"

install_cli "$CLI_DIR/local-info-scout" "local-info-scout"
