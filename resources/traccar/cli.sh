#!/bin/bash
# Traccar Fleet & Telematics Platform CLI Interface
# Provides management for Traccar GPS tracking server

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/lib"

# Source the core library
source "${LIB_DIR}/core.sh"

# Main command dispatcher
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help|--help|-h)
            cmd_help
            ;;
        info)
            cmd_info "$@"
            ;;
        manage)
            cmd_manage "$@"
            ;;
        test)
            cmd_test "$@"
            ;;
        content)
            cmd_content "$@"
            ;;
        status)
            cmd_status "$@"
            ;;
        logs)
            cmd_logs "$@"
            ;;
        credentials)
            cmd_credentials "$@"
            ;;
        device)
            cmd_device "$@"
            ;;
        track)
            cmd_track "$@"
            ;;
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run 'resource-traccar help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Run main function
main "$@"