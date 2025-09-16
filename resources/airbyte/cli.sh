#!/bin/bash
# Airbyte Resource CLI - v2.0 Contract Compliant
# ELT data integration platform with 600+ connectors

set -euo pipefail

# Resolve symlinks to get actual resource directory
SCRIPT_PATH="${BASH_SOURCE[0]}"
while [ -h "$SCRIPT_PATH" ]; do
    SCRIPT_DIR="$(cd -P "$(dirname "$SCRIPT_PATH")" && pwd)"
    SCRIPT_PATH="$(readlink "$SCRIPT_PATH")"
    [[ $SCRIPT_PATH != /* ]] && SCRIPT_PATH="$SCRIPT_DIR/$SCRIPT_PATH"
done
SCRIPT_DIR="$(cd -P "$(dirname "$SCRIPT_PATH")" && pwd)"

source "${SCRIPT_DIR}/lib/core.sh"

# Display help information
show_help() {
    cat << EOF
Airbyte - Open-source ELT data integration platform

USAGE:
    resource-airbyte <command> [options]

COMMANDS:
    help                          Show this help message
    info                          Show resource information
    
    manage install                Install Airbyte and dependencies
    manage start                  Start Airbyte services
    manage stop                   Stop Airbyte services  
    manage restart                Restart Airbyte services
    manage uninstall              Remove Airbyte completely
    
    test smoke                    Quick health check (<30s)
    test integration              Full integration tests (<120s)
    test unit                     Test library functions (<60s)
    test all                      Run all test suites
    
    content list                  List sources, destinations, or connections
    content add                   Add a new connector or connection
    content get                   Get connector/connection details
    content remove                Remove a connector or connection
    content execute               Trigger a sync job
    
    status                        Show service status
    logs                          View service logs
    credentials                   Display API credentials

EXAMPLES:
    # Start Airbyte
    resource-airbyte manage start
    
    # List available source connectors
    resource-airbyte content list --type sources
    
    # Create a new connection
    resource-airbyte content add --type connection --config connection.json
    
    # Trigger a sync
    resource-airbyte content execute --connection-id my-connection
    
    # View server logs
    resource-airbyte logs --service server

DEFAULT CONFIGURATION:
    Webapp Port: 8002
    API Port: 8003
    Temporal Port: 8006
    Data Directory: ./data

For more information, see: /home/matthalloran8/Vrooli/resources/airbyte/README.md
EOF
}

# Main command dispatcher
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        help|--help|-h|"")
            show_help
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
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run 'resource-airbyte help' for usage information" >&2
            exit 1
            ;;
    esac
}

main "$@"