#!/usr/bin/env bash
# Apache Airflow Resource CLI - v2.0 Contract Implementation
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="apache-airflow"

# Source core libraries
source "${SCRIPT_DIR}/lib/core.sh"
# Note: cli-command-framework-v2.sh if needed would be at:
# source "${SCRIPT_DIR}/../../scripts/resources/lib/cli-command-framework-v2.sh"

# Main command handler
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
        *)
            echo "Error: Unknown command '$command'"
            show_help
            exit 1
            ;;
    esac
}

# Show help information
show_help() {
    cat << EOF
Apache Airflow Resource CLI

USAGE:
    resource-apache-airflow <command> [options]

COMMANDS:
    help                Show this help message
    info               Show resource information from runtime.json
    manage             Lifecycle management commands
      install          Install Apache Airflow and dependencies
      start            Start Airflow services
      stop             Stop Airflow services
      restart          Restart Airflow services
      uninstall        Remove Apache Airflow completely
    test               Resource validation commands
      smoke            Quick health check (<30s)
      integration      Full functionality test (<2m)
      unit             Library function tests (<1m)
      all              Run all tests
    content            DAG management commands
      add              Add a new DAG file
      list             List available DAGs
      get              Get DAG definition
      remove           Remove a DAG
      execute          Trigger DAG execution
    status             Show detailed service status
    logs               View service logs

EXAMPLES:
    # Install and start Airflow
    resource-apache-airflow manage install
    resource-apache-airflow manage start --wait
    
    # Add and execute a DAG
    resource-apache-airflow content add --file my_dag.py
    resource-apache-airflow content execute --dag my_dag
    
    # Check service health
    resource-apache-airflow test smoke
    resource-apache-airflow status --json

DEFAULT CONFIGURATION:
    Web UI Port: 48080
    Redis Port: 46379 (internal)
    PostgreSQL Port: 45432 (internal)
    DAG Directory: /dags
    Log Directory: /logs

EOF
    return 0
}

# Show resource information
show_info() {
    local json_flag=false
    
    for arg in "$@"; do
        case "$arg" in
            --json) json_flag=true ;;
        esac
    done
    
    if [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        if [[ "$json_flag" == true ]]; then
            cat "${SCRIPT_DIR}/config/runtime.json"
        else
            echo "Apache Airflow Resource Information:"
            jq -r '
                "Startup Order: \(.startup_order)",
                "Dependencies: \(.dependencies | join(", "))",
                "Startup Time: \(.startup_time_estimate)",
                "Startup Timeout: \(.startup_timeout)s",
                "Recovery Attempts: \(.recovery_attempts)",
                "Priority: \(.priority)"
            ' "${SCRIPT_DIR}/config/runtime.json"
        fi
    else
        echo "Runtime configuration not found"
        return 1
    fi
}

# Handle lifecycle management commands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            "${SCRIPT_DIR}/lib/core.sh" install "$@"
            ;;
        start)
            "${SCRIPT_DIR}/lib/core.sh" start "$@"
            ;;
        stop)
            "${SCRIPT_DIR}/lib/core.sh" stop "$@"
            ;;
        restart)
            "${SCRIPT_DIR}/lib/core.sh" restart "$@"
            ;;
        uninstall)
            "${SCRIPT_DIR}/lib/core.sh" uninstall "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand '$subcommand'"
            show_help
            exit 1
            ;;
    esac
}

# Handle test commands
handle_test() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        smoke)
            "${SCRIPT_DIR}/lib/test.sh" smoke "$@"
            ;;
        integration)
            "${SCRIPT_DIR}/lib/test.sh" integration "$@"
            ;;
        unit)
            "${SCRIPT_DIR}/lib/test.sh" unit "$@"
            ;;
        all)
            "${SCRIPT_DIR}/lib/test.sh" all "$@"
            ;;
        *)
            echo "Error: Unknown test subcommand '$subcommand'"
            show_help
            exit 1
            ;;
    esac
}

# Handle content management commands
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        add)
            "${SCRIPT_DIR}/lib/dag-manager.sh" add "$@"
            ;;
        list)
            "${SCRIPT_DIR}/lib/dag-manager.sh" list "$@"
            ;;
        get)
            "${SCRIPT_DIR}/lib/dag-manager.sh" get "$@"
            ;;
        remove)
            "${SCRIPT_DIR}/lib/dag-manager.sh" remove "$@"
            ;;
        execute)
            "${SCRIPT_DIR}/lib/dag-manager.sh" execute "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand '$subcommand'"
            show_help
            exit 1
            ;;
    esac
}

# Show service status
show_status() {
    "${SCRIPT_DIR}/lib/core.sh" status "$@"
}

# Show service logs
show_logs() {
    "${SCRIPT_DIR}/lib/core.sh" logs "$@"
}

# Execute main function
main "$@"