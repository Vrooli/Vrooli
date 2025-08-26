#!/usr/bin/env bash

# ⚠️ DEPRECATION NOTICE: This script is deprecated as of v2.0 (January 2025)
# Please use cli.sh instead: resource-neo4j <command>
# This file will be removed in v3.0 (target: December 2025)
#
# Migration guide:
#   OLD: ./manage.sh --action <command>
#   NEW: ./cli.sh <command>  OR  resource-neo4j <command>

# Show deprecation warning
if [[ "${VROOLI_SUPPRESS_DEPRECATION:-}" != "true" ]]; then
    echo "⚠️  WARNING: manage.sh is deprecated. Please use cli.sh instead." >&2
    echo "   This script will be removed in v3.0 (December 2025)" >&2
    echo "   To suppress this warning: export VROOLI_SUPPRESS_DEPRECATION=true" >&2
    echo "" >&2
fi

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