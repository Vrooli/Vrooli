#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
TWILIO_LIB_DIR="${APP_ROOT}/resources/twilio/lib"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# List phone numbers
list_numbers() {
    log::header "📞 Twilio Phone Numbers"
    
    if [[ -f "$TWILIO_PHONE_NUMBERS_FILE" ]]; then
        local count
        count=$(jq '.numbers | length' "$TWILIO_PHONE_NUMBERS_FILE" 2>/dev/null || echo "0")
        
        if [[ "$count" -gt 0 ]]; then
            echo "Configured Numbers:"
            jq -r '.numbers[] | "  \(.phone // .number) - \(.name // "No name") (\(.type // "unknown"))"' \
                "$TWILIO_PHONE_NUMBERS_FILE" 2>/dev/null
        else
            log::info "No phone numbers configured"
        fi
    else
        log::info "No phone numbers file found"
    fi
    
    # If credentials are available, also show account numbers
    if twilio::has_credentials && twilio::is_installed; then
        echo
        echo "Account Numbers (from API):"
        if twilio::setup_auth; then
            twilio phone-numbers:list --properties phoneNumber,friendlyName 2>/dev/null || \
                log::warn "Could not fetch numbers from API"
        fi
    fi
}

twilio::list_numbers() {
    list_numbers "$@"
}

