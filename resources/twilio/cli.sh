#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    TWILIO_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${TWILIO_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
TWILIO_CLI_DIR="${APP_ROOT}/resources/twilio"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
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
    inject [file]   Inject templates, TwiML, or workflows
    list-injected   List all injected data
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
            twilio::status::new "$@"
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
        list-injected)
            twilio::list_injected "$@"
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
