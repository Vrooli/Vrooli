#!/usr/bin/env bash
# mcrcon CLI - Main entry point for Minecraft Remote Console resource
# Implements v2.0 universal resource contract

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Source configuration and libraries
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"

# Resource information
readonly RESOURCE_NAME="mcrcon"
readonly RESOURCE_VERSION="1.0.0"

# Help function
show_help() {
    cat << EOF
mcrcon - Minecraft Remote Console Resource

USAGE:
    resource-mcrcon <command> [options]

COMMANDS:
    help                    Show this help message
    info                    Show resource runtime information
    manage <subcommand>     Lifecycle management
        install             Install mcrcon and dependencies
        uninstall           Remove mcrcon
        start               Start mcrcon service
        stop                Stop mcrcon service
        restart             Restart mcrcon service
    test <type>             Run tests
        smoke               Quick health check (<30s)
        integration         Full functionality test (<120s)
        unit                Library function tests (<60s)
        all                 Run all test suites
    content <subcommand>    Execute Minecraft commands
        execute <cmd>       Execute RCON command
        list                List available servers
        add <server>        Add server configuration
        remove <server>     Remove server configuration
    status                  Show resource status
    logs                    View resource logs
    credentials             Display connection credentials

EXAMPLES:
    # Install and start mcrcon
    resource-mcrcon manage install
    resource-mcrcon manage start --wait

    # Execute Minecraft commands
    resource-mcrcon content execute "list"
    resource-mcrcon content execute "say Hello from Vrooli!"

    # Run tests
    resource-mcrcon test smoke
    resource-mcrcon test all

DEFAULT CONFIGURATION:
    Server Host: ${MCRCON_HOST:-localhost}
    Server Port: ${MCRCON_PORT:-25575}
    Timeout: ${MCRCON_TIMEOUT:-30}s

For more information, see the README.md file.
EOF
}

# Info function - displays runtime configuration
show_info() {
    local json_flag="${1:-}"
    local runtime_file="${SCRIPT_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        echo "Error: runtime.json not found" >&2
        return 1
    fi
    
    if [[ "$json_flag" == "--json" ]]; then
        cat "$runtime_file"
    else
        echo "mcrcon Resource Information:"
        echo "----------------------------"
        jq -r '
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Startup Time: \(.startup_time_estimate)",
            "Startup Timeout: \(.startup_timeout)s",
            "Recovery Attempts: \(.recovery_attempts)",
            "Priority: \(.priority)"
        ' "$runtime_file"
    fi
}

# Main command router
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        help|--help|-h|"")
            show_help
            ;;
        info)
            show_info "$@"
            ;;
        manage)
            "${SCRIPT_DIR}/lib/core.sh" manage "$@"
            ;;
        test)
            "${SCRIPT_DIR}/lib/test.sh" "$@"
            ;;
        content)
            "${SCRIPT_DIR}/lib/core.sh" content "$@"
            ;;
        status)
            "${SCRIPT_DIR}/lib/core.sh" status
            ;;
        logs)
            "${SCRIPT_DIR}/lib/core.sh" logs
            ;;
        credentials)
            "${SCRIPT_DIR}/lib/core.sh" credentials
            ;;
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run 'resource-mcrcon help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"