#!/bin/bash

# PyBullet Physics Simulation Resource CLI
# Implements v2.0 universal contract

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/core.sh"
source "$SCRIPT_DIR/lib/test.sh"

# Command routing
case "${1:-help}" in
    help)
        show_help
        ;;
    info)
        show_info "${2:-}"
        ;;
    manage)
        handle_manage "${2:-}" "${@:3}"
        ;;
    test)
        handle_test "${2:-}" "${@:3}"
        ;;
    content)
        handle_content "${2:-}" "${@:3}"
        ;;
    status)
        show_status "${2:-}"
        ;;
    logs)
        show_logs "${2:-}" "${@:3}"
        ;;
    credentials)
        show_credentials
        ;;
    *)
        echo "Error: Unknown command '$1'"
        echo "Run 'resource-pybullet help' for usage"
        exit 1
        ;;
esac