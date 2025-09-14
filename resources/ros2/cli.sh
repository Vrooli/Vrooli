#!/usr/bin/env bash

# ROS2 Resource - CLI Interface
# Provides Robot Operating System 2 middleware for robotics applications

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="ros2"

# Load common libraries
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Help function
show_help() {
    cat <<EOF
ROS2 Resource - Robot Operating System 2 Middleware

Usage: vrooli resource ${RESOURCE_NAME} [command] [options]

Commands:
  help                    Show this help message
  info                    Show resource information
  manage install          Install ROS2 and dependencies
  manage uninstall        Remove ROS2 completely
  manage start            Start ROS2 daemon and core services
  manage stop             Stop ROS2 services
  manage restart          Restart ROS2 services
  test smoke              Quick health check (<30s)
  test integration        Full functionality test (<120s)
  test unit               Library function tests (<60s)
  test all                Run all tests (<600s)
  content list-nodes      List active ROS2 nodes
  content launch-node     Launch a ROS2 node
  content list-topics     List available topics
  content publish         Publish to a topic
  content list-services   List available services
  content call-service    Call a ROS2 service
  status                  Show detailed status
  logs                    View ROS2 logs

Examples:
  # Install and start ROS2
  vrooli resource ${RESOURCE_NAME} manage install
  vrooli resource ${RESOURCE_NAME} manage start
  
  # Check health
  vrooli resource ${RESOURCE_NAME} test smoke
  
  # Launch nodes
  vrooli resource ${RESOURCE_NAME} content launch-node talker
  vrooli resource ${RESOURCE_NAME} content launch-node listener
  
  # List topics and publish
  vrooli resource ${RESOURCE_NAME} content list-topics
  vrooli resource ${RESOURCE_NAME} content publish /chatter "Hello ROS2"

Configuration:
  Port: ${ROS2_PORT:-11501}
  Domain ID: ${ROS2_DOMAIN_ID:-0}
  Middleware: ${ROS2_MIDDLEWARE:-fastdds}

For more information, see: resources/${RESOURCE_NAME}/README.md
EOF
}

# Info function
show_info() {
    if [[ "${1:-}" == "--json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "ROS2 Resource Information:"
        echo "=========================="
        jq -r 'to_entries | .[] | "  \(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

# Main command handler
main() {
    local command="${1:-help}"
    shift || true
    
    case "${command}" in
        help|--help|-h)
            show_help
            ;;
        info)
            show_info "$@"
            ;;
        manage)
            local subcommand="${1:-}"
            shift || true
            case "${subcommand}" in
                install)
                    ros2_install "$@"
                    ;;
                uninstall)
                    ros2_uninstall "$@"
                    ;;
                start)
                    ros2_start "$@"
                    ;;
                stop)
                    ros2_stop "$@"
                    ;;
                restart)
                    ros2_stop "$@"
                    ros2_start "$@"
                    ;;
                *)
                    echo "Error: Unknown manage subcommand: ${subcommand}" >&2
                    show_help
                    exit 1
                    ;;
            esac
            ;;
        test)
            local test_type="${1:-all}"
            shift || true
            ros2_test "${test_type}" "$@"
            ;;
        content)
            local content_cmd="${1:-}"
            shift || true
            ros2_content "${content_cmd}" "$@"
            ;;
        status)
            ros2_status "$@"
            ;;
        logs)
            ros2_logs "$@"
            ;;
        *)
            echo "Error: Unknown command: ${command}" >&2
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"