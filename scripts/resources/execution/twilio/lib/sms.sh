#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
TWILIO_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# Send SMS
send_sms() {
    local to="${1:-}"
    local message="${2:-}"
    local from="${3:-}"
    
    if [[ -z "$to" || -z "$message" ]]; then
        log::error "Usage: resource-twilio send-sms <to-number> <message> [from-number]"
        return 1
    fi
    
    # Check if installed
    if ! twilio::is_installed; then
        log::error "$MSG_TWILIO_NOT_INSTALLED"
        return 1
    fi
    
    # Check credentials
    if ! twilio::has_credentials; then
        log::error "$MSG_TWILIO_NO_CREDENTIALS"
        return 1
    fi
    
    # Get from number if not provided
    if [[ -z "$from" ]]; then
        from=$(twilio::get_default_number)
        if [[ -z "$from" ]]; then
            from=$(jq -r '.default_from_number // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null || echo "")
            if [[ -z "$from" ]]; then
                log::error "No from number specified and no default configured"
                return 1
            fi
        fi
    fi
    
    log::info "📱 Sending SMS..."
    log::info "   From: $from"
    log::info "   To: $to"
    log::info "   Message: ${message:0:50}..."
    
    # Set up auth and send
    if twilio::setup_auth; then
        local cmd
        cmd=$(twilio::get_command)
        if "$cmd" phone-numbers:update "$from" --sms-url="" 2>/dev/null; then
            if "$cmd" api:core:messages:create \
                --from "$from" \
                --to "$to" \
                --body "$message" 2>&1; then
                log::success "SMS sent successfully!"
            else
                log::error "Failed to send SMS"
                return 1
            fi
        else
            log::warn "Could not update phone number settings"
        fi
    else
        log::error "Authentication failed"
        return 1
    fi
    
    return 0
}

twilio::send_sms() {
    send_sms "$@"
}

