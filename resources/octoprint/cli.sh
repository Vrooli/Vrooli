#!/bin/bash

# OctoPrint Resource CLI
# Web-based 3D printer management platform

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="octoprint"

# Source configuration
source "${SCRIPT_DIR}/config/defaults.sh"

# Source core library
source "${SCRIPT_DIR}/lib/core.sh"

# Source test library for test commands
source "${SCRIPT_DIR}/lib/test.sh"

# Main CLI handler
main() {
    local command="${1:-help}"
    shift || true
    
    case "${command}" in
        help)
            show_help
            ;;
        info)
            show_info
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
            show_status
            ;;
        logs)
            show_logs "$@"
            ;;
        credentials)
            show_credentials
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
OctoPrint Resource CLI - 3D Printer Management Platform

Usage: vrooli resource ${RESOURCE_NAME} <command> [options]

Commands:
  help                Show this help message
  info                Show runtime configuration
  manage              Lifecycle management
    install           Install OctoPrint
    start [--wait]    Start OctoPrint service
    stop              Stop OctoPrint service
    restart           Restart OctoPrint service
    uninstall         Remove OctoPrint
  test                Run tests
    smoke             Quick health check (30s)
    integration       Full functionality test (120s)
    unit              Library function tests (60s)
    all               Run all tests
  content             Manage print files
    list              List G-code files
    add <file>        Upload G-code file
    get <file>        Download G-code file
    remove <file>     Delete G-code file
    execute <file>    Start print job
  status              Show detailed status
  logs [--tail N]     View service logs
  credentials         Display API credentials

Environment Variables:
  OCTOPRINT_PORT              Web interface port (default: 8197)
  OCTOPRINT_PRINTER_PORT      Printer serial port (default: /dev/ttyUSB0)
  OCTOPRINT_VIRTUAL_PRINTER   Use virtual printer (default: false)
  OCTOPRINT_API_KEY          API key (default: auto-generated)

Examples:
  # Start OctoPrint with virtual printer for testing
  OCTOPRINT_VIRTUAL_PRINTER=true vrooli resource ${RESOURCE_NAME} manage start --wait
  
  # Upload and print a G-code file
  vrooli resource ${RESOURCE_NAME} content add model.gcode
  vrooli resource ${RESOURCE_NAME} content execute model.gcode
  
  # Monitor printer status
  vrooli resource ${RESOURCE_NAME} status

Documentation: https://docs.octoprint.org
EOF
}

# Show runtime configuration
show_info() {
    echo "OctoPrint Runtime Configuration"
    echo "================================"
    echo "Port: ${OCTOPRINT_PORT}"
    echo "Host: ${OCTOPRINT_HOST}"
    echo "Printer Port: ${OCTOPRINT_PRINTER_PORT}"
    echo "Virtual Printer: ${OCTOPRINT_VIRTUAL_PRINTER}"
    echo "API Enabled: ${OCTOPRINT_API_ENABLED}"
    echo "WebSocket Enabled: ${OCTOPRINT_WEBSOCKET_ENABLED}"
    echo "Camera Enabled: ${OCTOPRINT_CAMERA_ENABLED}"
    echo "Data Directory: ${OCTOPRINT_DATA_DIR}"
    echo "G-Code Directory: ${OCTOPRINT_GCODE_DIR}"
    echo "Docker Image: ${OCTOPRINT_DOCKER_IMAGE}"
}

# Handle lifecycle management commands
handle_manage() {
    local action="${1:-}"
    shift || true
    
    case "${action}" in
        install)
            octoprint_install
            ;;
        start)
            octoprint_start "$@"
            ;;
        stop)
            octoprint_stop
            ;;
        restart)
            octoprint_restart
            ;;
        uninstall)
            octoprint_uninstall
            ;;
        *)
            echo "Error: Unknown manage action '${action}'"
            echo "Valid actions: install, start, stop, restart, uninstall"
            exit 1
            ;;
    esac
}

# Handle test commands
handle_test() {
    local test_type="${1:-all}"
    
    case "${test_type}" in
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
            echo "Error: Unknown test type '${test_type}'"
            echo "Valid types: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Handle content management
handle_content() {
    local action="${1:-list}"
    shift || true
    
    case "${action}" in
        list)
            content_list
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
            echo "Error: Unknown content action '${action}'"
            echo "Valid actions: list, add, get, remove, execute"
            exit 1
            ;;
    esac
}

# Show service status
show_status() {
    octoprint_status
}

# Show logs
show_logs() {
    local tail_lines="${1:-50}"
    if [[ "${1:-}" == "--tail" ]]; then
        tail_lines="${2:-50}"
    fi
    
    octoprint_logs "${tail_lines}"
}

# Show credentials
show_credentials() {
    octoprint_credentials
}

# Run main function
main "$@"