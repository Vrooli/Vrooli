#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_PATH="${APP_ROOT}/scenarios/code-tidiness-manager/cli/code-tidiness-manager"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

if [[ ! -f "$CLI_PATH" ]]; then
    echo "⚠️  CLI script not found at $CLI_PATH; skipping install" >&2
    exit 0
fi

install_cli "$CLI_PATH" "code-tidiness-manager"
