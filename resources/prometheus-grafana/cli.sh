#!/usr/bin/env bash
# Prometheus + Grafana Resource CLI
# Provides unified monitoring and observability for Vrooli

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Source configuration and libraries
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Resource metadata
readonly RESOURCE_NAME="prometheus-grafana"
readonly RESOURCE_VERSION="1.0.0"

#######################################
# Show help information
#######################################
show_help() {
    cat << EOF
Prometheus + Grafana Observability Stack

USAGE:
    resource-prometheus-grafana <command> [options]

COMMANDS:
    help                    Show this help message
    info                    Show resource information
    manage <subcommand>     Lifecycle management
        install            Install the monitoring stack
        start              Start all services
        stop               Stop all services
        restart            Restart all services
        uninstall          Remove the monitoring stack
    test <type>            Run tests
        smoke              Quick health check
        integration        Full integration tests
        unit               Unit tests
        all                Run all tests
    content <subcommand>    Content management
        list [--type]      List metrics, dashboards, or alerts
        add <type> <file>  Add dashboard, alert rule, or config
        get <item>         Get specific metric or dashboard
        remove <item>      Remove dashboard or alert
        execute <query>    Execute PromQL query
    status                 Show detailed status
    logs [service]         View service logs
    credentials            Display access credentials

OPTIONS:
    --json                 Output in JSON format
    --verbose             Verbose output
    --help                Show help for specific command

EXAMPLES:
    # Install and start monitoring
    resource-prometheus-grafana manage install
    resource-prometheus-grafana manage start

    # Check service health
    resource-prometheus-grafana status

    # View Grafana credentials
    resource-prometheus-grafana credentials

    # Execute a PromQL query
    resource-prometheus-grafana content execute "up"

    # Add a custom dashboard
    resource-prometheus-grafana content add dashboard ./my-dashboard.json

EOF
}

#######################################
# Show resource information
#######################################
show_info() {
    local json_output="${1:-false}"
    
    if [[ "$json_output" == "true" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "Resource: prometheus-grafana"
        echo "Version: ${RESOURCE_VERSION}"
        echo "Status: $(get_status)"
        echo ""
        echo "Configuration:"
        jq -r 'to_entries[] | "  \(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

#######################################
# Main command router
#######################################
main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        help)
            show_help
            ;;
        info)
            local json_output="false"
            [[ "${1:-}" == "--json" ]] && json_output="true"
            show_info "$json_output"
            ;;
        manage)
            handle_manage_command "$@"
            ;;
        test)
            handle_test_command "$@"
            ;;
        content)
            handle_content_command "$@"
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
            echo "Run 'resource-prometheus-grafana help' for usage"
            exit 1
            ;;
    esac
}

#######################################
# Handle manage subcommands
#######################################
handle_manage_command() {
    local subcommand="${1:-}"
    shift || true

    case "$subcommand" in
        install)
            install_resource "$@"
            ;;
        start|develop)
            start_resource "$@"
            ;;
        stop)
            stop_resource "$@"
            ;;
        restart)
            restart_resource "$@"
            ;;
        uninstall)
            uninstall_resource "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand '$subcommand'"
            echo "Valid subcommands: install, start, stop, restart, uninstall"
            exit 1
            ;;
    esac
}

#######################################
# Handle test subcommands
#######################################
handle_test_command() {
    local test_type="${1:-all}"
    shift || true

    case "$test_type" in
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
            echo "Error: Unknown test type '$test_type'"
            echo "Valid types: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

#######################################
# Handle content subcommands
#######################################
handle_content_command() {
    local subcommand="${1:-}"
    shift || true

    case "$subcommand" in
        list)
            list_content "$@"
            ;;
        add)
            add_content "$@"
            ;;
        get)
            get_content "$@"
            ;;
        remove)
            remove_content "$@"
            ;;
        execute)
            execute_query "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand '$subcommand'"
            echo "Valid subcommands: list, add, get, remove, execute"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"