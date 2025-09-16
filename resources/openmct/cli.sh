#!/usr/bin/env bash
# Open MCT Resource CLI - v2.0 Contract Compliant
set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source configuration and libraries
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Main command router
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help)
            show_help
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
        *)
            echo "Error: Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Show help information
show_help() {
    cat << EOF
Open MCT Resource - NASA Mission Control Telemetry Visualization

Usage: resource-openmct <command> [options]

Commands:
    help                    Show this help message
    info [--json]          Show resource information from runtime.json
    manage <subcommand>    Lifecycle management
        install            Install Open MCT and dependencies
        uninstall          Remove Open MCT completely
        start [--wait]     Start Open MCT service
        stop               Stop Open MCT service
        restart            Restart Open MCT service
    test <phase>           Run tests
        smoke              Quick health validation (<30s)
        integration        End-to-end functionality tests
        unit               Library function tests
        all                Run all test phases
    content <subcommand>   Content management
        add <type> <cfg>   Add telemetry source (websocket/mqtt/traccar)
        list               List configured telemetry sources
        get <stream>       Get telemetry data for stream
        remove <id>        Remove telemetry source
        execute <cmd>      Execute command (clear-history/export/import)
    status [--json]        Show detailed status
    logs [--tail N]        View resource logs

Examples:
    # Install and start Open MCT
    resource-openmct manage install
    resource-openmct manage start --wait
    
    # Add a telemetry source
    resource-openmct content add websocket ws://localhost:8080/telemetry
    
    # View telemetry data
    resource-openmct content list
    resource-openmct content get satellite_position
    
    # Export telemetry history
    resource-openmct content execute export telemetry_backup.csv

Default Configuration:
    Port: ${OPENMCT_PORT}
    WebSocket Port: ${OPENMCT_WS_PORT}
    Data Directory: ${OPENMCT_DATA_DIR}
    Demo Mode: ${OPENMCT_ENABLE_DEMO}
    Max Streams: ${OPENMCT_MAX_STREAMS}
    History Days: ${OPENMCT_HISTORY_DAYS}

Dashboard Access:
    URL: http://localhost:${OPENMCT_PORT}
    WebSocket: ws://localhost:${OPENMCT_WS_PORT}/api/telemetry/live

For more information, see: resources/openmct/README.md
EOF
}

# Execute main function
main "$@"