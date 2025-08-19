#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
TWILIO_MANAGE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Main router - delegates to lib scripts
main() {
    # Handle --action flag for vrooli resource compatibility
    if [[ "${1:-}" == "--action" ]]; then
        shift
    fi
    
    local cmd="${1:-help}"
    shift || true
    
    case "$cmd" in
        status|start|stop|install|logs|config|inject|send-sms|list-numbers)
            "$TWILIO_MANAGE_DIR/lib/${cmd}.sh" "$@"
            ;;
        help)
            "$TWILIO_MANAGE_DIR/cli.sh" help
            ;;
        *)
            echo "Error: Unknown command '$cmd'"
            "$TWILIO_MANAGE_DIR/cli.sh" help
            exit 1
            ;;
    esac
}

main "$@"