#!/usr/bin/env bash
# Segment Anything Resource - CLI Interface
# Provides foundation segmentation capabilities with SAM2/HQ-SAM models

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="${SCRIPT_DIR}"

# Load core library
source "${RESOURCE_DIR}/lib/core.sh"

# Show help information
show_help() {
    cat <<EOF
Segment Anything Resource - Foundation Segmentation Service

USAGE:
    resource-segment-anything <command> [options]

COMMANDS:
    help                Show this help message
    info [--json]       Show resource information
    manage <action>     Lifecycle management
        install         Install resource and dependencies
        uninstall       Remove resource and cleanup
        start [--wait]  Start segmentation service
        stop            Stop service gracefully
        restart         Restart service
    test <type>         Run tests
        smoke           Quick health validation (<30s)
        integration     Full functionality test (<120s)
        unit            Library function tests (<60s)
        all             Run all test suites
    content <action>    Content operations
        list            List available models
        add <model>     Add/download a model
        get <id>        Get segmentation result
        remove <id>     Remove cached result
        execute <args>  Run segmentation task
    status [--json]     Show service status
    logs [--follow]     View service logs
    credentials         Display integration credentials

EXAMPLES:
    # Start the service
    resource-segment-anything manage start --wait

    # Run segmentation on an image
    resource-segment-anything content execute --image /path/to/image.jpg --prompt "point:100,200"

    # List available models
    resource-segment-anything content list

    # Check service health
    resource-segment-anything test smoke

    # View logs
    resource-segment-anything logs --follow

CONFIGURATION:
    Port: ${SEGMENT_ANYTHING_PORT:-11454}
    Model: ${SEGMENT_ANYTHING_MODEL_TYPE:-sam2}-${SEGMENT_ANYTHING_MODEL_SIZE:-base}
    Device: ${SEGMENT_ANYTHING_DEVICE:-auto}
    Data Dir: ${SEGMENT_ANYTHING_DATA_DIR}

For more information, see: resources/segment-anything/README.md
EOF
}

# Main command router
main() {
    local command="${1:-}"
    shift || true

    case "$command" in
        help|--help|-h)
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
            show_credentials
            ;;
        *)
            echo "Error: Unknown command: $command" >&2
            echo "Run 'resource-segment-anything help' for usage information" >&2
            exit 1
            ;;
    esac
}

# Run main function
main "$@"