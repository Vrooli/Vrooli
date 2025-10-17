#!/usr/bin/env bash
################################################################################
# OWASP ZAP Resource CLI - v2.0 Universal Contract Compliant
# 
# Web application security scanner with automated vulnerability detection
#
# Usage:
#   resource-owasp-zap <command> [options]
#   resource-owasp-zap <group> <subcommand> [options]
#
################################################################################

set -e

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ZAP_CLI_DIR="$SCRIPT_DIR"
RESOURCE_NAME="owasp-zap"

# Source configuration
source "$SCRIPT_DIR/config/defaults.sh"

# Source library functions
source "$SCRIPT_DIR/lib/core.sh"
source "$SCRIPT_DIR/lib/test.sh"

# Helper functions for CLI
log_info() {
    echo "[INFO] $*" >&2
}

log_success() {
    echo "[SUCCESS] $*" >&2
}

log_error() {
    echo "[ERROR] $*" >&2
}

log_warn() {
    echo "[WARN] $*" >&2
}

# Show help
show_help() {
    cat <<EOF
OWASP ZAP Security Scanner Resource - v2.0 Contract Compliant

Usage:
  resource-owasp-zap <command> [options]
  resource-owasp-zap <group> <subcommand> [options]

Commands:
  help                    Show this help message
  info [--json]           Show resource information
  status [--json]         Show current status
  logs                    Show resource logs
  
  manage install          Install OWASP ZAP
  manage uninstall        Remove OWASP ZAP
  manage start [--wait]   Start ZAP daemon
  manage stop             Stop ZAP daemon
  manage restart          Restart ZAP daemon
  
  test smoke              Run quick health checks
  test integration        Run integration tests  
  test unit               Run unit tests
  test all                Run all tests
  
  content add <target>    Add scan target
  content execute <url>   Run security scan
  content list [--json]   List scan results
  content get [format]    Get scan report

Examples:
  resource-owasp-zap manage install
  resource-owasp-zap manage start --wait
  resource-owasp-zap content execute http://localhost:3000
  resource-owasp-zap content list --json
  resource-owasp-zap status

Configuration:
  API Port: ${ZAP_API_PORT}
  Proxy Port: ${ZAP_PROXY_PORT}
  Data Directory: ${ZAP_DATA_DIR}
EOF
}

# Show info
show_info() {
    local format="${1:-text}"
    
    if [[ "$format" == "--json" ]]; then
        cat "$SCRIPT_DIR/config/runtime.json"
    else
        echo "OWASP ZAP Security Scanner"
        echo "Version: 2.0.0"
        echo "API Port: ${ZAP_API_PORT}"
        echo "Proxy Port: ${ZAP_PROXY_PORT}"
        echo "Data Directory: ${ZAP_DATA_DIR}"
        jq -r '.description' "$SCRIPT_DIR/config/runtime.json"
    fi
}

# Handle manage commands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            zap::install "$@"
            ;;
        uninstall)
            zap::uninstall "$@"
            ;;
        start)
            zap::start "$@"
            ;;
        stop)
            zap::stop
            ;;
        restart)
            zap::restart
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand"
            echo "Available: install, uninstall, start, stop, restart"
            return 1
            ;;
    esac
}

# Handle test commands
handle_test() {
    local test_type="${1:-all}"
    zap::test "$test_type"
}

# Handle content commands
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        add)
            zap::content_add "$@"
            ;;
        execute)
            zap::content_execute "$@"
            ;;
        list)
            zap::content_list "$@"
            ;;
        get)
            zap::content_get "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand: $subcommand"
            echo "Available: add, execute, list, get"
            return 1
            ;;
    esac
}

# Show status
show_status() {
    local format="${1:-text}"
    zap::status "$format"
}

# Show logs
show_logs() {
    if [[ -f "$ZAP_LOG_FILE" ]]; then
        tail -n 50 "$ZAP_LOG_FILE"
    else
        if zap::is_running; then
            docker logs --tail 50 "$ZAP_CONTAINER_NAME"
        else
            echo "No logs available (ZAP not running)"
        fi
    fi
}

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
        *)
            echo "Error: Unknown command: $command"
            show_help
            return 1
            ;;
    esac
}

# Run main function
main "$@"