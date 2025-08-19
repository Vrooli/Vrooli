#!/usr/bin/env bash
# Codex resource management script

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Handle legacy --action format
if [[ "${1:-}" == "--action" ]]; then
    shift  # Remove --action
fi

# Forward to resource CLI
exec "$SCRIPT_DIR/resource-codex" "$@"