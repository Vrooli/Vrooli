#!/usr/bin/env bash
# Splink Resource CLI - Probabilistic Record Linkage at Scale
# Implements v2.0 universal resource contract

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="splink"

# Source the core library
source "${SCRIPT_DIR}/lib/core.sh"

# Main command handler
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
            echo "Error: Unknown command: $command"
            echo "Run 'resource-${RESOURCE_NAME} help' for usage information"
            exit 1
            ;;
    esac
}

# Show help information
show_help() {
    cat << EOF
Splink - Probabilistic Record Linkage at Scale

USAGE:
    resource-${RESOURCE_NAME} <COMMAND> [OPTIONS]

COMMANDS:
    help            Show this help message
    info            Display runtime configuration and dependencies
    manage          Lifecycle management (install/start/stop/restart/uninstall)
    test            Run validation tests (smoke/integration/unit/all)
    content         Manage linkage operations (execute/list/get/remove)
    status          Show detailed service status
    logs            View service logs
    credentials     Display integration credentials

EXAMPLES:
    # Start the service
    resource-${RESOURCE_NAME} manage start --wait
    
    # Deduplicate a dataset
    resource-${RESOURCE_NAME} content execute deduplicate --dataset customers.csv
    
    # Link two datasets
    resource-${RESOURCE_NAME} content execute link --dataset1 orders.csv --dataset2 customers.csv
    
    # Run tests
    resource-${RESOURCE_NAME} test all

DOCUMENTATION:
    See resources/${RESOURCE_NAME}/README.md for detailed documentation
    API docs: resources/${RESOURCE_NAME}/docs/api.md

EOF
}

# Show runtime info
show_info() {
    echo "Splink Resource Information"
    echo "=========================="
    echo "Version: 3.x"
    echo "Port: ${SPLINK_PORT:-8096}"
    echo "Backend: ${SPLINK_BACKEND:-duckdb}"
    echo "Config: ${SCRIPT_DIR}/config/runtime.json"
    echo ""
    
    if [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        echo "Runtime Configuration:"
        cat "${SCRIPT_DIR}/config/runtime.json"
    fi
}

# Handle lifecycle management
handle_manage() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        install)
            install_dependencies "$@"
            ;;
        start)
            start_service "$@"
            ;;
        stop)
            stop_service "$@"
            ;;
        restart)
            restart_service "$@"
            ;;
        uninstall)
            uninstall_service "$@"
            ;;
        *)
            echo "Error: Unknown manage action: $action"
            echo "Available actions: install, start, stop, restart, uninstall"
            exit 1
            ;;
    esac
}

# Handle test commands
handle_test() {
    local test_type="${1:-all}"
    shift || true
    
    if [[ -f "${SCRIPT_DIR}/lib/test.sh" ]]; then
        source "${SCRIPT_DIR}/lib/test.sh"
        run_tests "$test_type" "$@"
    else
        echo "Error: Test library not found at ${SCRIPT_DIR}/lib/test.sh"
        exit 1
    fi
}

# Handle content operations
handle_content() {
    local operation="${1:-list}"
    shift || true
    
    case "$operation" in
        execute)
            execute_linkage "$@"
            ;;
        list)
            list_jobs "$@"
            ;;
        get)
            get_results "$@"
            ;;
        remove)
            remove_job "$@"
            ;;
        visualize)
            visualize_results "$@"
            ;;
        *)
            echo "Error: Unknown content operation: $operation"
            echo "Available operations: execute, list, get, remove, visualize"
            exit 1
            ;;
    esac
}

# Show service status
show_status() {
    echo "Splink Service Status"
    echo "==================="
    
    if check_service_health; then
        echo "Status: RUNNING"
        echo "Health: OK"
        echo "Port: ${SPLINK_PORT:-8096}"
        echo "PID: $(get_service_pid)"
        echo "Uptime: $(get_service_uptime)"
        
        # Show resource usage if available
        if command -v docker &> /dev/null && docker ps --format "{{.Names}}" | grep -q "splink"; then
            echo ""
            echo "Resource Usage:"
            docker stats --no-stream splink 2>/dev/null || true
        fi
    else
        echo "Status: STOPPED"
        echo "Health: N/A"
    fi
}

# Show logs
show_logs() {
    local lines="${1:-50}"
    local follow="${2:-}"
    
    local log_file="${VROOLI_ROOT:-$HOME/Vrooli}/.vrooli/logs/resources/${RESOURCE_NAME}.log"
    
    if [[ ! -f "$log_file" ]]; then
        echo "No logs found at: $log_file"
        return 0
    fi
    
    if [[ "$follow" == "--follow" || "$follow" == "-f" ]]; then
        tail -f "$log_file"
    else
        tail -n "$lines" "$log_file"
    fi
}

# Show credentials for integration
show_credentials() {
    echo "Splink Integration Credentials"
    echo "============================="
    echo "API Endpoint: http://localhost:${SPLINK_PORT:-8096}"
    echo "Health Check: http://localhost:${SPLINK_PORT:-8096}/health"
    echo ""
    echo "Example Usage:"
    echo "  curl -X POST http://localhost:${SPLINK_PORT:-8096}/linkage/deduplicate \\"
    echo "    -H 'Content-Type: application/json' \\"
    echo "    -d '{\"dataset_id\": \"customers\", \"settings\": {}}'"
}

# Execute the main function
main "$@"