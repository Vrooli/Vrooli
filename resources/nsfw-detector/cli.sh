#!/usr/bin/env bash
# NSFW Detector Resource CLI - v2.0 Contract Implementation
# Provides AI-powered NSFW content detection and moderation

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
NSFW Detector Resource - AI-powered content moderation

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
    content <subcommand>   Manage content and models
        add               Add content/model
        list              List available content
        get               Retrieve specific content
        remove            Remove content
        execute           Execute classification
    status [--json]        Show service status
    logs [--tail N]        View service logs

EXAMPLES:
    # Install and start the service
    $(basename "$0") manage install
    $(basename "$0") manage start --wait

    # Classify an image
    $(basename "$0") content execute --file image.jpg

    # Check service health
    $(basename "$0") status

    # Run tests
    $(basename "$0") test smoke

DEFAULT CONFIGURATION:
    Port:       11451 (from resource registry)
    Model:      nsfwjs
    Threshold:  0.7 (configurable)
    Timeout:    5000ms

For more information, see README.md
EOF
}

# Show resource information from runtime.json
show_info() {
    local format="text"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                format="json"
                shift
                ;;
            *)
                echo "Error: Unknown option: $1" >&2
                exit 1
                ;;
        esac
    done
    
    if [[ "$format" == "json" ]]; then
        cat "${SCRIPT_DIR}/config/runtime.json"
    else
        echo "NSFW Detector Resource Information:"
        echo "------------------------------------"
        jq -r 'to_entries | .[] | "\(.key): \(.value)"' "${SCRIPT_DIR}/config/runtime.json"
    fi
}

# Handle manage subcommands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        install)
            manage_install "$@"
            ;;
        uninstall)
            manage_uninstall "$@"
            ;;
        start)
            manage_start "$@"
            ;;
        stop)
            manage_stop "$@"
            ;;
        restart)
            manage_restart "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# Handle test subcommands
handle_test() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        all)
            test_all "$@"
            ;;
        smoke)
            test_smoke "$@"
            ;;
        integration)
            test_integration "$@"
            ;;
        unit)
            test_unit "$@"
            ;;
        *)
            echo "Error: Unknown test subcommand: $subcommand" >&2
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
            echo "Error: Unknown content subcommand: $subcommand" >&2
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"