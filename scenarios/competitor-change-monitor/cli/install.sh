#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/competitor-change-monitor/cli"
CLI_SCRIPT="$CLI_DIR/competitor-monitor"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

if [[ ! -f "$CLI_SCRIPT" ]]; then
    echo "⚠️  CLI script not found at $CLI_SCRIPT; skipping install" >&2
    exit 0
fi

install_cli "$CLI_SCRIPT" "competitor-monitor"
install_cli "$CLI_SCRIPT" "competitor-change-monitor"
