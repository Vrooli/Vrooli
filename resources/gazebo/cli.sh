#!/usr/bin/env bash
# Gazebo Robotics Simulation Platform CLI
# v2.0 Universal Contract Implementation

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    GAZEBO_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${GAZEBO_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi

SCRIPT_DIR="${APP_ROOT}/resources/gazebo"
RESOURCE_NAME="gazebo"

# Source required libraries
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
            echo "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Show comprehensive help
show_help() {
    cat << EOF
Gazebo Robotics Simulation Platform

Usage:
  resource-${RESOURCE_NAME} <command> [options]

Commands:
  help                    Show this help message
  info [--json]           Show resource information from runtime.json
  manage <subcommand>     Lifecycle management
    install               Install Gazebo and dependencies
    uninstall             Remove Gazebo installation
    start [--wait]        Start Gazebo server
    stop                  Stop Gazebo server
    restart               Restart Gazebo server
  test <suite>            Run test suites
    smoke                 Quick health validation (<30s)
    integration           Full functionality test
    unit                  Library function tests
    all                   Run all test suites
  content <subcommand>    Content management
    list                  List available worlds and models
    add <type> <file>     Add world or model file
    get <type> <name>     Get world or model details
    remove <type> <name>  Remove world or model
    execute <cmd> [args]  Execute simulation commands
  status [--json]         Show detailed status
  logs [--follow]         View resource logs

Examples:
  # Install and start Gazebo
  resource-${RESOURCE_NAME} manage install
  resource-${RESOURCE_NAME} manage start --wait
  
  # Load and run a world
  resource-${RESOURCE_NAME} content add world examples/cart_pole.world
  resource-${RESOURCE_NAME} content execute run-world cart_pole
  
  # Spawn a robot model
  resource-${RESOURCE_NAME} content add model robots/quadcopter.sdf
  resource-${RESOURCE_NAME} content execute spawn-robot quadcopter 0 0 1
  
  # Check simulation status
  resource-${RESOURCE_NAME} status --json

Configuration:
  Port: ${GAZEBO_PORT:-11456}
  Data Directory: ${GAZEBO_DATA_DIR:-${HOME}/.gazebo}
  Config File: ${SCRIPT_DIR}/config/defaults.sh

For more information: https://gazebosim.org/docs
EOF
}

# Show resource information
show_info() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                json_output=true
                shift
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    if [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        if [[ "$json_output" == "true" ]]; then
            cat "${SCRIPT_DIR}/config/runtime.json"
        else
            echo "Gazebo Resource Information:"
            jq -r '
                "Startup Order: \(.startup_order)",
                "Dependencies: \(.dependencies | join(", "))",
                "Startup Time: \(.startup_time_estimate)",
                "Timeout: \(.startup_timeout)s",
                "Recovery Attempts: \(.recovery_attempts)",
                "Priority: \(.priority)"
            ' "${SCRIPT_DIR}/config/runtime.json"
        fi
    else
        echo "Runtime configuration not found"
        exit 1
    fi
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            gazebo_install "$@"
            ;;
        uninstall)
            gazebo_uninstall "$@"
            ;;
        start)
            gazebo_start "$@"
            ;;
        stop)
            gazebo_stop "$@"
            ;;
        restart)
            gazebo_stop
            gazebo_start "$@"
            ;;
        *)
            echo "Unknown manage subcommand: $subcommand"
            echo "Valid subcommands: install, uninstall, start, stop, restart"
            exit 1
            ;;
    esac
}

# Handle test subcommands
handle_test() {
    local suite="${1:-all}"
    
    case "$suite" in
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
            echo "Unknown test suite: $suite"
            echo "Valid suites: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Handle content subcommands
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        list)
            content_list "$@"
            ;;
        add)
            content_add "$@"
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
            echo "Unknown content subcommand: $subcommand"
            echo "Valid subcommands: list, add, get, remove, execute"
            exit 1
            ;;
    esac
}

# Show status
show_status() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                json_output=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    gazebo_status "$json_output"
}

# Show logs
show_logs() {
    local follow=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --follow|-f)
                follow=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    gazebo_logs "$follow"
}

# Run main function
main "$@"