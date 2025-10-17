#!/usr/bin/env bash
# OpenRocket Launch Vehicle Design Studio CLI
# Provides rocket design, simulation, and trajectory analysis capabilities

set -euo pipefail

RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${RESOURCE_DIR}/lib/core.sh"

# Default operation
OPERATION="${1:-help}"
shift 2>/dev/null || true

# Dispatch commands
case "${OPERATION}" in
    help)
        openrocket_help "$@"
        ;;
    info)
        openrocket_info "$@"
        ;;
    manage)
        openrocket_manage "$@"
        ;;
    test)
        openrocket_test "$@"
        ;;
    content)
        openrocket_content "$@"
        ;;
    status)
        openrocket_status "$@"
        ;;
    logs)
        openrocket_logs "$@"
        ;;
    *)
        echo "Error: Unknown command '${OPERATION}'"
        echo "Run '$(basename "$0") help' for usage information"
        exit 1
        ;;
esac