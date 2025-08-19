#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
TWILIO_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# Show configuration
show_config() {
    log::header "⚙️  Twilio Configuration"
    
    echo "Directories:"
    echo "  Config: $TWILIO_CONFIG_DIR"
    echo "  Data: $TWILIO_DATA_DIR"
    echo "  Workflows: $TWILIO_WORKFLOWS_DIR"
    echo
    
    if twilio::has_credentials; then
        local account_sid
        account_sid=$(twilio::get_account_sid)
        echo "Credentials:"
        echo "  Account SID: ${account_sid:0:10}..."
        echo "  Status: Configured"
    else
        echo "Credentials: Not configured"
        echo "  File: $TWILIO_CREDENTIALS_FILE"
    fi
    echo
    
    if [[ -f "$TWILIO_PHONE_NUMBERS_FILE" ]]; then
        local count
        count=$(jq '.numbers | length' "$TWILIO_PHONE_NUMBERS_FILE" 2>/dev/null || echo "0")
        echo "Phone Numbers: $count configured"
    else
        echo "Phone Numbers: None configured"
    fi
}

main() {
    show_config "$@"
}

main "$@"