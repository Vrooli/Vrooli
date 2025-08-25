#!/usr/bin/env bash
# Neo4j Resource Management Script

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/neo4j"

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