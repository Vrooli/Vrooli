#!/usr/bin/env bash
# Temporal Resource CLI - v2.0 Contract Compliant
# Durable workflow orchestration with guaranteed execution

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="temporal"

# Load core library
source "${SCRIPT_DIR}/lib/core.sh"

# Main command router
main() {
    local command="${1:-}"
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
            echo "Run 'resource-temporal help' for usage information"
            exit 1
            ;;
    esac
}

# Show comprehensive help
show_help() {
    cat << EOF
Temporal Resource - Durable Workflow Orchestration

USAGE:
    resource-temporal <command> [subcommand] [options]

COMMANDS:
    help        Show this help message
    info        Show resource information from runtime.json
    manage      Lifecycle management (install/start/stop/restart/uninstall)
    test        Run validation tests (smoke/integration/unit/all)
    content     Manage workflows (add/list/get/remove/execute)
    status      Show detailed service status
    logs        View service logs
    credentials Display connection credentials

EXAMPLES:
    # Start Temporal server
    resource-temporal manage start
    
    # Check service health
    resource-temporal status
    
    # Run smoke tests
    resource-temporal test smoke
    
    # List workflows
    resource-temporal content list
    
    # View logs
    resource-temporal logs --tail 50

DEFAULT CONFIGURATION:
    Web UI Port: 7233
    gRPC Port: 7234
    Database: PostgreSQL
    Default Namespace: default

For more information, see: /resources/temporal/README.md
EOF
    exit 0
}

# Show resource information
show_info() {
    local json_flag="${1:-}"
    
    if [[ "$json_flag" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "Temporal Resource Information:"
        echo "=============================="
        jq -r '
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Startup Time: \(.startup_time_estimate)",
            "Startup Timeout: \(.startup_timeout)s",
            "Recovery Attempts: \(.recovery_attempts)",
            "Priority: \(.priority)"
        ' "${SCRIPT_DIR}/config/runtime.json"
    fi
    exit 0
}

# Handle lifecycle management
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            install_temporal "$@"
            ;;
        uninstall)
            uninstall_temporal "$@"
            ;;
        start)
            start_temporal "$@"
            ;;
        stop)
            stop_temporal "$@"
            ;;
        restart)
            stop_temporal "$@"
            start_temporal "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand"
            echo "Valid subcommands: install, uninstall, start, stop, restart"
            exit 1
            ;;
    esac
}

# Handle test commands
handle_test() {
    local subcommand="${1:-}"
    shift || true
    
    # Source test library for test functions
    source "${SCRIPT_DIR}/lib/test.sh"
    
    case "$subcommand" in
        smoke)
            run_smoke_tests "$@"
            ;;
        integration)
            run_integration_tests "$@"
            ;;
        unit)
            run_unit_tests "$@"
            ;;
        all)
            run_all_tests "$@"
            ;;
        *)
            echo "Error: Unknown test subcommand: $subcommand"
            echo "Valid subcommands: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Handle content management
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        add)
            add_workflow "$@"
            ;;
        list)
            list_workflows "$@"
            ;;
        get)
            get_workflow "$@"
            ;;
        remove)
            remove_workflow "$@"
            ;;
        execute)
            execute_workflow "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand: $subcommand"
            echo "Valid subcommands: add, list, get, remove, execute"
            exit 1
            ;;
    esac
}

# Run the main function
main "$@"