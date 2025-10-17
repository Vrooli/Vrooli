#!/bin/bash
# GGWave Resource CLI Interface
# Implements v2.0 universal contract

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="ggwave"

# Source configuration
source "${SCRIPT_DIR}/config/defaults.sh"

# Source library functions
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# CLI command router
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
        credentials)
            show_credentials "$@"
            ;;
        *)
            echo "Error: Unknown command '$command'"
            echo "Run 'resource-${RESOURCE_NAME} help' for usage information"
            exit 1
            ;;
    esac
}

# Show help information
show_help() {
    cat << EOF
GGWave Resource - Air-Gapped Data Transmission via Sound

USAGE:
    resource-${RESOURCE_NAME} <COMMAND> [OPTIONS]

COMMANDS:
    help                Show this help message
    info                Show resource information and configuration
    manage              Lifecycle management commands
        install         Install and configure GGWave
        uninstall       Remove GGWave and cleanup
        start           Start the GGWave service
        stop            Stop the GGWave service
        restart         Restart the GGWave service
    test                Run validation tests
        smoke           Quick health check (<30s)
        integration     Full functionality test (<120s)
        unit            Library function tests (<60s)
        all             Run all test phases
    content             Content management operations
        add             Add data for transmission
        list            List available content
        get             Retrieve decoded data
        remove          Remove stored data
        execute         Transmit data via audio
    status              Show service status
    logs                View service logs
    credentials         Display API credentials

EXAMPLES:
    # Install and start GGWave
    resource-${RESOURCE_NAME} manage install
    resource-${RESOURCE_NAME} manage start --wait
    
    # Test audio transmission
    resource-${RESOURCE_NAME} content execute --data "Hello World" --mode normal
    
    # Check service health
    resource-${RESOURCE_NAME} status --verbose
    
    # Run tests
    resource-${RESOURCE_NAME} test smoke

OPTIONS:
    --json              Output in JSON format
    --verbose           Show detailed output
    --force             Skip confirmation prompts
    --wait              Wait for operation to complete

For more information, see: /resources/${RESOURCE_NAME}/README.md
EOF
}

# Show resource information
show_info() {
    local json_flag=false
    
    for arg in "$@"; do
        case "$arg" in
            --json) json_flag=true ;;
        esac
    done
    
    if [[ "$json_flag" == "true" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "=== GGWave Resource Information ==="
        echo "Version: ${GGWAVE_VERSION}"
        echo "Port: ${GGWAVE_PORT}"
        echo "Category: Communication"
        echo ""
        echo "Configuration:"
        echo "  Mode: ${GGWAVE_MODE}"
        echo "  Sample Rate: ${GGWAVE_SAMPLE_RATE} Hz"
        echo "  Volume: ${GGWAVE_VOLUME}"
        echo "  Error Correction: ${GGWAVE_ERROR_CORRECTION}"
        echo ""
        echo "Runtime:"
        jq -r '
            "  Startup Order: \(.startup_order)",
            "  Startup Timeout: \(.startup_timeout)s",
            "  Priority: \(.priority)",
            "  Dependencies: \(.dependencies | join(", "))"
        ' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            ggwave::install "$@"
            ;;
        uninstall)
            ggwave::uninstall "$@"
            ;;
        start)
            ggwave::start "$@"
            ;;
        stop)
            ggwave::stop "$@"
            ;;
        restart)
            ggwave::restart "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand '$subcommand'"
            echo "Available: install, uninstall, start, stop, restart"
            exit 1
            ;;
    esac
}

# Handle test subcommands
handle_test() {
    local phase="${1:-all}"
    shift || true
    
    case "$phase" in
        smoke)
            ggwave::test::smoke "$@"
            ;;
        integration)
            ggwave::test::integration "$@"
            ;;
        unit)
            ggwave::test::unit "$@"
            ;;
        all)
            ggwave::test::all "$@"
            ;;
        *)
            echo "Error: Unknown test phase '$phase'"
            echo "Available: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Handle content subcommands
handle_content() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        add)
            ggwave::content::add "$@"
            ;;
        list)
            ggwave::content::list "$@"
            ;;
        get)
            ggwave::content::get "$@"
            ;;
        remove)
            ggwave::content::remove "$@"
            ;;
        execute)
            ggwave::content::execute "$@"
            ;;
        *)
            echo "Error: Unknown content action '$action'"
            echo "Available: add, list, get, remove, execute"
            exit 1
            ;;
    esac
}

# Show service status
show_status() {
    ggwave::status "$@"
}

# Show service logs
show_logs() {
    ggwave::logs "$@"
}

# Show credentials (not applicable for ggwave)
show_credentials() {
    echo "GGWave does not require authentication credentials."
    echo "The service is designed for air-gapped operation via physical proximity."
    echo ""
    echo "API Endpoint: http://localhost:${GGWAVE_PORT}"
    echo "Health Check: http://localhost:${GGWAVE_PORT}/health"
}

# Run main function
main "$@"