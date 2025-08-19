#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
TWILIO_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$TWILIO_CLI_DIR/../../../lib/utils/var.sh"
source "$TWILIO_CLI_DIR/../../../lib/utils/format.sh"
source "$TWILIO_CLI_DIR/../../../lib/utils/log.sh"
source "$TWILIO_CLI_DIR/lib/common.sh"

# Source all lib functions
for lib_file in "$TWILIO_CLI_DIR"/lib/*.sh; do
    [[ -f "$lib_file" ]] && source "$lib_file"
done

# Help function
show_help() {
    cat << HELP
Twilio Resource CLI

Usage: resource-twilio <command> [options]

Commands:
    status          Check Twilio status
    install         Install Twilio CLI
    start           Start Twilio service (monitor mode)
    stop            Stop Twilio monitor
    logs            Show Twilio logs
    config          View/update Twilio configuration
    inject [file]   Inject phone numbers or workflows
    send-sms        Send a test SMS message
    list-numbers    List configured phone numbers
    help            Show this help message

Examples:
    resource-twilio status
    resource-twilio send-sms "+1234567890" "Test message"
    resource-twilio inject phone-numbers.json

HELP
}

# Main command router
main() {
    local cmd="${1:-help}"
    shift || true
    
    case "$cmd" in
        status)
            check_status "$@"
            ;;
        install)
            twilio::install "$@"
            ;;
        start)
            twilio::start "$@"
            ;;
        stop)
            twilio::stop "$@"
            ;;
        logs)
            twilio::logs "$@"
            ;;
        config)
            twilio::config "$@"
            ;;
        inject)
            twilio::inject "$@"
            ;;
        send-sms)
            twilio::send_sms "$@"
            ;;
        list-numbers)
            twilio::list_numbers "$@"
            ;;
        help)
            show_help
            ;;
        *)
            echo "Error: Unknown command '$cmd'"
            show_help
            exit 1
            ;;
    esac
}

main "$@"