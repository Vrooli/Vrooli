#!/usr/bin/env bash
# Codex resource management script

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/codex"

# Handle legacy --action format
if [[ "${1:-}" == "--action" ]]; then
    shift  # Remove --action
fi

# Forward to resource CLI
exec "$SCRIPT_DIR/resource-codex" "$@"