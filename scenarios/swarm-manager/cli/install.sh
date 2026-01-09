#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/swarm-manager/cli"
CLI_SCRIPT="$CLI_DIR/../cli.sh"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

if [[ ! -f "$CLI_SCRIPT" ]]; then
    echo "❌ CLI script not found at $CLI_SCRIPT" >&2
    exit 1
fi

install_cli "$CLI_SCRIPT" "swarm-manager"
