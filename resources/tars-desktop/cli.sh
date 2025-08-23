#!/bin/bash
# TARS-desktop CLI interface

# Get script directory (handle symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # Script is a symlink, resolve it
    TARS_DESKTOP_CLI_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
else
    TARS_DESKTOP_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# Source all library files
source "${TARS_DESKTOP_CLI_DIR}/lib/core.sh"
source "${TARS_DESKTOP_CLI_DIR}/lib/status.sh"
source "${TARS_DESKTOP_CLI_DIR}/lib/install.sh"
source "${TARS_DESKTOP_CLI_DIR}/lib/start.sh"
source "${TARS_DESKTOP_CLI_DIR}/lib/inject.sh"

# Main CLI handler
tars_desktop::cli() {
    local command="${1:-help}"
    shift
    
    case "$command" in
        status)
            # Pass all arguments directly to status function
            tars_desktop::status "$@"
            ;;
        install)
            local verbose="false"
            if [[ "${1:-}" == "--verbose" ]]; then
                verbose="true"
            fi
            tars_desktop::install "$verbose"
            ;;
        uninstall)
            local verbose="false"
            if [[ "${1:-}" == "--verbose" ]]; then
                verbose="true"
            fi
            tars_desktop::uninstall "$verbose"
            ;;
        start)
            local verbose="false"
            if [[ "${1:-}" == "--verbose" ]]; then
                verbose="true"
            fi
            tars_desktop::start "$verbose"
            ;;
        stop)
            local verbose="false"
            if [[ "${1:-}" == "--verbose" ]]; then
                verbose="true"
            fi
            tars_desktop::stop "$verbose"
            ;;
        restart)
            local verbose="false"
            if [[ "${1:-}" == "--verbose" ]]; then
                verbose="true"
            fi
            tars_desktop::restart "$verbose"
            ;;
        health|health-check)
            if tars_desktop::health_check; then
                echo "✓ TARS-desktop is healthy"
                return 0
            else
                echo "✗ TARS-desktop health check failed"
                return 1
            fi
            ;;
        capabilities)
            tars_desktop::get_capabilities | jq '.' 2>/dev/null || tars_desktop::get_capabilities
            ;;
        screenshot)
            local output="${1:-/tmp/tars-screenshot.png}"
            tars_desktop::screenshot "$output"
            ;;
        execute)
            local action="${1:-}"
            local target="${2:-}"
            if [[ -z "$action" ]]; then
                echo "Error: Action required"
                return 1
            fi
            tars_desktop::execute_action "$action" "$target" | jq '.' 2>/dev/null || echo "Action executed"
            ;;
        inject)
            local target="${1:-}"
            local data="${2:-}"
            local verbose="false"
            if [[ "${3:-}" == "--verbose" ]]; then
                verbose="true"
            fi
            tars_desktop::inject "$target" "$data" "$verbose"
            ;;
        help|--help|-h)
            cat <<EOF
TARS Desktop UI Automation CLI

Usage: $0 <command> [options]

Commands:
  status [--verbose] [--format json|text]  Check TARS-desktop status
  install [--verbose]                      Install TARS-desktop
  uninstall [--verbose]                    Uninstall TARS-desktop
  start [--verbose]                        Start TARS-desktop server
  stop [--verbose]                         Stop TARS-desktop server
  restart [--verbose]                      Restart TARS-desktop server
  health|health-check                      Run health check
  capabilities                             Get available capabilities
  screenshot [output_path]                 Capture a screenshot
  execute <action> [target]                Execute a UI action
  inject <target> [data] [--verbose]       Inject config to other resources
  help                                     Show this help message

Actions for execute:
  click, doubleClick, rightClick           Mouse click actions
  moveTo, dragTo                           Mouse movement actions
  scroll                                   Scroll action
  typewrite <text>                         Type text
  hotkey <keys>                            Press hotkey combination

Examples:
  $0 status --verbose
  $0 install
  $0 start
  $0 screenshot /tmp/screen.png
  $0 execute click
  $0 execute typewrite "Hello World"
  $0 execute hotkey "ctrl+c"
  $0 inject n8n

EOF
            ;;
        *)
            echo "Unknown command: $command"
            echo "Run '$0 help' for usage information"
            return 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    set +e  # Disable strict error handling for CLI
    tars_desktop::cli "$@"
fi