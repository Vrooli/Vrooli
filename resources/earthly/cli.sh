#!/bin/bash
# Earthly Resource CLI - v2.0 Contract Compliant
# Provides containerized build automation with Dockerfile+Makefile concepts

set -euo pipefail

# Get the directory of this script (handle symlinks)
if [ -L "${BASH_SOURCE[0]}" ]; then
    SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
else
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi
RESOURCE_NAME="earthly"

# Source configuration and libraries
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Display help information
show_help() {
    cat << EOF
🏗️ EARTHLY Build Automation Resource

📋 USAGE:
    resource-${RESOURCE_NAME} <command> [subcommand] [options]

📖 DESCRIPTION:
    Containerized build system with automatic parallelization and intelligent caching

🎯 COMMAND GROUPS:
    content              📄 Build and artifact management
    manage               ⚙️  Resource lifecycle management  
    test                 🧪 Testing and validation

    💡 Use 'resource-${RESOURCE_NAME} <group> --help' for subcommands

ℹ️  INFORMATION COMMANDS:
    help                 Show this help message
    info                 Show resource configuration
    logs                 Show Earthly daemon logs
    status               Show detailed status

🔧 OTHER COMMANDS:
    credentials          Show Earthly configuration
    benchmark            Run performance benchmarks
    
⚙️  OPTIONS:
    --dry-run            Show what would be done without making changes
    --help, -h           Show help message

💡 EXAMPLES:
    # Resource lifecycle
    resource-${RESOURCE_NAME} manage install
    resource-${RESOURCE_NAME} manage start
    resource-${RESOURCE_NAME} manage stop

    # Build execution
    resource-${RESOURCE_NAME} content execute --target=+build
    resource-${RESOURCE_NAME} content list artifacts

    # Testing
    resource-${RESOURCE_NAME} test smoke
    resource-${RESOURCE_NAME} test all

📚 For more help on a specific command:
    resource-${RESOURCE_NAME} <command> --help
EOF
}

# Main command router
main() {
    local command="${1:-}"
    shift || true

    case "${command}" in
        help|--help|-h|"")
            show_help
            exit 0
            ;;
        info)
            show_info "$@"
            ;;
        manage)
            handle_manage "$@"
            ;;
        test)
            handle_test "$@"
            ;;
        content)
            handle_content "$@"
            ;;
        status)
            show_status "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
        credentials)
            show_credentials "$@"
            ;;
        benchmark)
            run_benchmark "$@"
            ;;
        *)
            echo "[ERROR] Unknown command: ${command}"
            echo "Run 'resource-${RESOURCE_NAME} help' for usage information"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"