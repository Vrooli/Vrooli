#!/usr/bin/env bash
# GridLAB-D Resource CLI - v2.0 Contract Compliant
set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source configuration
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
GridLAB-D Resource - Power Distribution System Simulator

Usage: resource-gridlabd <command> [options]

Commands:
    help                    Show this help message
    info [--json]          Show resource information from runtime.json
    manage <subcommand>    Lifecycle management
        install            Install GridLAB-D and dependencies
        uninstall          Remove GridLAB-D completely
        start [--wait]     Start GridLAB-D service
        stop               Stop GridLAB-D service
        restart            Restart GridLAB-D service
    test <phase>           Run tests
        smoke              Quick health validation (<30s)
        integration        End-to-end functionality tests
        unit               Library function tests
        all                Run all test phases
    content <subcommand>   Content management
        add <file>         Add model file
        list               List available models
        get <id>           Get simulation results
        remove <id>        Remove model or results
        execute [opts]     Execute simulation
    status [--json]        Show detailed status
    logs [--tail N]        View resource logs

Examples:
    # Install and start GridLAB-D
    resource-gridlabd manage install
    resource-gridlabd manage start --wait
    
    # Run a simulation
    resource-gridlabd content add model.glm
    resource-gridlabd content execute --model model
    
    # Check results
    resource-gridlabd content list
    resource-gridlabd content get results/model_output.csv

Default Configuration:
    Port: ${GRIDLABD_PORT}
    Data Directory: ${GRIDLABD_DATA_DIR}
    Log Level: ${GRIDLABD_LOG_LEVEL}

For more information, see: resources/gridlabd/README.md
EOF
}

# Show resource information
show_info() {
    local json_flag=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_flag=true
                shift
                ;;
            *)
                echo "Error: Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    if [ "$json_flag" = true ]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "GridLAB-D Resource Information:"
        echo "==============================="
        jq -r 'to_entries[] | "  \(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            gridlabd_install "$@"
            ;;
        uninstall)
            gridlabd_uninstall "$@"
            ;;
        start)
            gridlabd_start "$@"
            ;;
        stop)
            gridlabd_stop "$@"
            ;;
        restart)
            gridlabd_restart "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand"
            echo "Valid subcommands: install, uninstall, start, stop, restart"
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
            test_smoke
            ;;
        integration)
            test_integration
            ;;
        unit)
            test_unit
            ;;
        all)
            test_all
            ;;
        *)
            echo "Error: Unknown test phase: $phase"
            echo "Valid phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Handle content subcommands
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
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
            echo "Error: Unknown content subcommand: $subcommand"
            echo "Valid subcommands: add, list, get, remove, execute"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"