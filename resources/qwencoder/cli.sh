#!/usr/bin/env bash
# QwenCoder Resource CLI - v2.0 Contract Compliant
# Provides code generation and completion capabilities via QwenCoder models

set -euo pipefail

# Resource metadata
readonly RESOURCE_NAME="qwencoder"
readonly RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly LIB_DIR="${RESOURCE_DIR}/lib"
readonly CONFIG_DIR="${RESOURCE_DIR}/config"
readonly TEST_DIR="${RESOURCE_DIR}/test"

# Load core libraries
source "${LIB_DIR}/core.sh"
source "${LIB_DIR}/test.sh"

# Main CLI handler
main() {
    local command="${1:-}"
    shift || true
    
    case "${command}" in
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
            echo "Error: Unknown command '${command}'"
            show_help
            exit 1
            ;;
    esac
}

# Show help information
show_help() {
    cat << EOF
QwenCoder Resource - Code Generation LLM

USAGE:
    ${0##*/} <command> [options]

COMMANDS:
    help         Show this help message
    info         Show resource information (--json for JSON output)
    manage       Lifecycle management (install|start|stop|restart|uninstall)
    test         Run tests (smoke|integration|unit|all)
    content      Content management (add|list|get|remove|execute)
    status       Show resource status (--json for JSON output)
    logs         View resource logs (--follow to tail)
    credentials  Display integration credentials

EXAMPLES:
    # Install and start QwenCoder
    ${0##*/} manage install
    ${0##*/} manage start
    
    # Check status
    ${0##*/} status
    
    # Run code completion
    ${0##*/} content execute --prompt "def sort_array(arr):" --language python
    
    # Download a specific model
    ${0##*/} content add --model qwencoder-1.5b
    
    # Run tests
    ${0##*/} test smoke

CONFIGURATION:
    Port:          ${QWENCODER_PORT:-11452}
    Default Model: ${QWENCODER_MODEL:-qwencoder-1.5b}
    Device:        ${QWENCODER_DEVICE:-auto}
    
For detailed documentation, see: ${RESOURCE_DIR}/README.md
EOF
}

# Show resource information
show_info() {
    local json_flag="${1:-}"
    
    if [[ "${json_flag}" == "--json" ]]; then
        cat "${CONFIG_DIR}/runtime.json"
    else
        echo "QwenCoder Resource Information:"
        echo "================================"
        jq -r '
            "Startup Order: \(.startup_order)",
            "Dependencies: \(.dependencies | join(", "))",
            "Startup Time: \(.startup_time_estimate)",
            "Timeout: \(.startup_timeout)s",
            "Recovery Attempts: \(.recovery_attempts)",
            "Priority: \(.priority)"
        ' "${CONFIG_DIR}/runtime.json"
    fi
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        install)
            qwencoder_install "$@"
            ;;
        start)
            qwencoder_start "$@"
            ;;
        stop)
            qwencoder_stop "$@"
            ;;
        restart)
            qwencoder_stop "$@"
            qwencoder_start "$@"
            ;;
        uninstall)
            qwencoder_uninstall "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand '${subcommand}'"
            echo "Valid subcommands: install, start, stop, restart, uninstall"
            exit 1
            ;;
    esac
}

# Handle test subcommands
handle_test() {
    local test_type="${1:-all}"
    
    case "${test_type}" in
        smoke)
            run_smoke_tests
            ;;
        integration)
            run_integration_tests
            ;;
        unit)
            run_unit_tests
            ;;
        all)
            run_all_tests
            ;;
        *)
            echo "Error: Unknown test type '${test_type}'"
            echo "Valid types: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Handle content subcommands
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        add)
            content_add "$@"
            ;;
        list)
            content_list "$@"
            ;;
        get)
            content_get "$@"
            ;;
        remove)
            content_remove "$@"
            ;;
        execute)
            content_execute "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand '${subcommand}'"
            echo "Valid subcommands: add, list, get, remove, execute"
            exit 1
            ;;
    esac
}

# Show resource status
show_status() {
    local json_flag="${1:-}"
    qwencoder_status "${json_flag}"
}

# Show resource logs
show_logs() {
    local follow_flag="${1:-}"
    qwencoder_logs "${follow_flag}"
}

# Show credentials
show_credentials() {
    echo "QwenCoder Credentials:"
    echo "======================"
    echo "API Endpoint: http://localhost:${QWENCODER_PORT:-11452}"
    echo "Health Check: http://localhost:${QWENCODER_PORT:-11452}/health"
    echo "OpenAI Compatible: Yes"
    echo ""
    echo "Integration Example:"
    echo "  export OPENAI_API_BASE=http://localhost:${QWENCODER_PORT:-11452}/v1"
    echo "  export OPENAI_API_KEY=dummy"
}

# Execute main function
main "$@"