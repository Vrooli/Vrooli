#!/bin/bash
# Pushover resource management script

# Get script directory
PUSHOVER_MANAGE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Execute CLI with all arguments
"${PUSHOVER_MANAGE_DIR}/cli.sh" "$@"