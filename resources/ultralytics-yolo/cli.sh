#!/bin/bash
# CLI interface for Ultralytics YOLO resource
# Follows v2.0 universal contract

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/core.sh"

#######################################
# Display help information
#######################################
show_help() {
    cat << EOF
Ultralytics YOLO Vision Intelligence Suite

USAGE:
    resource-ultralytics-yolo <command> [options]

COMMANDS:
    help                Show this help message
    info                Show runtime configuration
    status              Show detailed service status
    logs                View service logs
    
    manage <action>     Lifecycle management
        install         Install YOLO with dependencies
        start           Start YOLO service
        stop            Stop YOLO service
        restart         Restart YOLO service
        uninstall       Remove YOLO and dependencies
    
    test <phase>        Run validation tests
        smoke           Quick health validation (<30s)
        integration     End-to-end functionality (<120s)
        unit            Library function tests (<60s)
        all             Run all test phases (<600s)
    
    content <action>    Model management
        list            List available models
        add <model>     Download specific model
        get <model>     Get model information
        remove <model>  Remove cached model
    
    detect              Run object detection
        --image PATH    Image file to process
        --video PATH    Video file to process
        --model NAME    Model variant (yolov8n/s/m/l/x)
        --output PATH   Output file path
    
    segment             Run instance segmentation
        --image PATH    Image file to process
        --model NAME    Model variant
        --output PATH   Output file path
    
    classify            Run image classification
        --image PATH    Image file to process
        --model NAME    Model variant
        --output PATH   Output file path

EXAMPLES:
    # Install and start
    resource-ultralytics-yolo manage install
    resource-ultralytics-yolo manage start --wait
    
    # Run detection
    resource-ultralytics-yolo detect --image photo.jpg --model yolov8m
    
    # List models
    resource-ultralytics-yolo content list
    
    # Run tests
    resource-ultralytics-yolo test smoke

EOF
}

#######################################
# Main CLI router
#######################################
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        help|--help|-h)
            show_help
            ;;
        info)
            yolo::info
            ;;
        status)
            yolo::status "$@"
            ;;
        logs)
            yolo::logs "$@"
            ;;
        manage)
            local action="${1:-}"
            shift || true
            case "$action" in
                install)
                    yolo::install "$@"
                    ;;
                start)
                    yolo::start "$@"
                    ;;
                stop)
                    yolo::stop "$@"
                    ;;
                restart)
                    yolo::restart "$@"
                    ;;
                uninstall)
                    yolo::uninstall "$@"
                    ;;
                *)
                    echo "Unknown manage action: $action"
                    show_help
                    exit 1
                    ;;
            esac
            ;;
        test)
            local phase="${1:-all}"
            shift || true
            yolo::test "$phase" "$@"
            ;;
        content)
            local action="${1:-list}"
            shift || true
            case "$action" in
                list)
                    yolo::content_list "$@"
                    ;;
                add)
                    yolo::content_add "$@"
                    ;;
                get)
                    yolo::content_get "$@"
                    ;;
                remove)
                    yolo::content_remove "$@"
                    ;;
                *)
                    echo "Unknown content action: $action"
                    show_help
                    exit 1
                    ;;
            esac
            ;;
        detect)
            yolo::detect "$@"
            ;;
        segment)
            yolo::segment "$@"
            ;;
        classify)
            yolo::classify "$@"
            ;;
        *)
            echo "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"