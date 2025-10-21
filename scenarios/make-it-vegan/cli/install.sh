#!/usr/bin/env bash
set -euo pipefail

# Resolve APP_ROOT with fallback
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
if [ -z "${APP_ROOT}" ] || [ ! -d "${APP_ROOT}" ]; then
    echo "Error: APP_ROOT is not set or is not a valid directory" >&2
    exit 1
fi

# Construct CLI_DIR and validate
CLI_DIR="${APP_ROOT}/scenarios/make-it-vegan/cli"
if [ ! -d "${CLI_DIR}" ]; then
    echo "Error: CLI_DIR '${CLI_DIR}' is not a valid directory" >&2
    exit 1
fi

# Source installation utility
if [ ! -f "${APP_ROOT}/scripts/lib/utils/cli-install.sh" ]; then
    echo "Error: cli-install.sh not found at ${APP_ROOT}/scripts/lib/utils/cli-install.sh" >&2
    exit 1
fi
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

install_cli "$CLI_DIR/make-it-vegan" "make-it-vegan"
