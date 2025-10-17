#!/usr/bin/env bash
# Apache Superset CLI - Enterprise Analytics Platform
set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Source configurations and libraries
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Main command handler
main() {
    local cmd="${1:-help}"
    shift || true

    case "$cmd" in
        help)
            superset::show_help
            ;;
        info)
            superset::show_info "$@"
            ;;
        manage)
            superset::manage "$@"
            ;;
        test)
            superset::test "$@"
            ;;
        content)
            superset::content "$@"
            ;;
        status)
            superset::status "$@"
            ;;
        logs)
            superset::logs "$@"
            ;;
        credentials)
            superset::credentials "$@"
            ;;
        *)
            echo "Error: Unknown command '$cmd'"
            echo "Run 'resource-apache-superset help' for usage information"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"