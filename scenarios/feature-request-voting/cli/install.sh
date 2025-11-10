#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/feature-request-voting/cli"
CLI_SCRIPT="$CLI_DIR/feature-voting"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

if [[ ! -f "$CLI_SCRIPT" ]]; then
    echo "⚠️  CLI script not found at $CLI_SCRIPT; skipping install" >&2
    exit 0
fi

install_cli "$CLI_SCRIPT" "feature-request-voting"
install_cli "$CLI_SCRIPT" "feature-voting"
