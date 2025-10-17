#!/usr/bin/env bash
# DeepStack Computer Vision Resource CLI - v2.0 Contract Implementation
# Cross-platform AI engine for computer vision via REST API

set -euo pipefail

# Script directory for relative imports
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Load core library
source "${SCRIPT_DIR}/lib/core.sh"

# Main CLI entrypoint
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
            echo "Error: Unknown command: $command" >&2
            echo "Run '$(basename "$0") help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Show comprehensive help
show_help() {
    cat <<EOF
DeepStack Computer Vision Resource - AI-powered object detection and recognition

USAGE:
    $(basename "$0") <command> [options]

COMMANDS:
    help                    Show this help message
    info [--json]          Show resource information
    manage <subcommand>    Manage resource lifecycle
        install            Install resource and dependencies
        uninstall         Remove resource completely
        start [--wait]    Start the service
        stop              Stop the service
        restart           Restart the service
    test <subcommand>      Run validation tests
        all               Run all test suites
        smoke             Quick health validation
        integration       Integration tests
        unit              Unit tests
    content <subcommand>   Manage models and process images
        add               Add custom model
        list              List available models
        get               Retrieve model info
        remove            Remove custom model
        execute           Process image for detection
    status [--json]        Show service status
    logs [--tail N]        View service logs

EXAMPLES:
    # Install and start the service
    $(basename "$0") manage install
    $(basename "$0") manage start --wait

    # Detect objects in an image
    $(basename "$0") content execute --file image.jpg --type object

    # Register a face for recognition
    $(basename "$0") content add --type face --name "John" --file john.jpg

    # Check service health
    $(basename "$0") status

    # Run tests
    $(basename "$0") test smoke

DEFAULT CONFIGURATION:
    Port:           11453 (from resource registry)
    Models:         YOLOv5 (object), RetinaFace (face)
    Confidence:     0.45 (configurable)
    GPU:            Auto-detect with CPU fallback
    Cache:          Redis on port 6380 (optional)

DETECTION TYPES:
    object          Detect 80+ object classes (COCO dataset)
    face            Detect faces with landmarks
    face-register   Register face for recognition
    face-recognize  Match face against registered faces
    scene           Classify scene/environment

For more information, see README.md
EOF
}

# Call main function
main "$@"