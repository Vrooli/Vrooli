#!/usr/bin/env bash
# Neo4j Resource Management Script

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Parse --action parameter if present
if [[ "$1" == "--action" ]]; then
    shift  # Remove --action
    ACTION="$1"
    shift  # Remove the action value
    # Execute CLI with the action and remaining parameters
    exec "$SCRIPT_DIR/cli.sh" "$ACTION" "$@"
else
    # Execute CLI with all parameters as is
    exec "$SCRIPT_DIR/cli.sh" "$@"
fi