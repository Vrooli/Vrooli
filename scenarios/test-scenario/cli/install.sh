#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
REPO_ROOT="$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)"
CLI_PATH="${REPO_ROOT}/scenarios/test-scenario/cli/test-scenario"
source "${REPO_ROOT}/scripts/lib/utils/cli-install.sh"

if [[ ! -f "$CLI_PATH" ]]; then
    echo "⚠️  CLI script not found at $CLI_PATH; skipping install" >&2
    exit 0
fi

install_cli "$CLI_PATH" "test-scenario"
