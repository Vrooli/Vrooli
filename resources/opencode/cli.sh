#!/bin/bash

# OpenCode CLI - Thin wrapper around library functions
set -euo pipefail

# Get script directory
OPENCODE_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source library functions
source "${OPENCODE_CLI_DIR}/lib/common.sh"
source "${OPENCODE_CLI_DIR}/lib/status.sh"
source "${OPENCODE_CLI_DIR}/lib/install.sh"
source "${OPENCODE_CLI_DIR}/lib/start.sh"
source "${OPENCODE_CLI_DIR}/lib/stop.sh"
source "${OPENCODE_CLI_DIR}/lib/inject.sh"
source "${OPENCODE_CLI_DIR}/lib/configure.sh"
source "${OPENCODE_CLI_DIR}/lib/test.sh"

# Main command handler
main() {
    local command="${1:-}"
    shift || true
    
    case "${command}" in
        status)
            opencode_status "$@"
            ;;
        install)
            opencode_install "$@"
            ;;
        start)
            opencode_start "$@"
            ;;
        stop)
            opencode_stop "$@"
            ;;
        inject)
            opencode_inject "$@"
            ;;
        configure)
            opencode_configure "$@"
            ;;
        test|run-tests)
            opencode_test "$@"
            ;;
        help|--help|-h|"")
            cat <<EOF
OpenCode Resource CLI

Usage: $0 <command> [options]

Commands:
    status          Check OpenCode status
    install         Install OpenCode extension
    start           Start OpenCode service
    stop            Stop OpenCode service
    inject          Inject configuration into OpenCode
    configure       Configure API keys and settings
    test            Run integration tests
    help            Show this help message

Examples:
    $0 status
    $0 install
    $0 configure --model ollama/llama3.2:3b
    $0 test
EOF
            ;;
        *)
            echo "Unknown command: ${command}"
            echo "Run '$0 help' for usage information"
            exit 1
            ;;
    esac
}

main "$@"