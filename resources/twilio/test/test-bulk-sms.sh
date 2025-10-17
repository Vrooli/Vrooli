#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Test bulk SMS functionality
test_bulk_sms() {
    log::info "Testing bulk SMS functionality..."
    
    # Test with test credentials
    export TWILIO_ACCOUNT_SID="AC_test_bulk_sms"
    export TWILIO_AUTH_TOKEN="test_token"
    
    # Create test CSV file
    local test_csv="/tmp/twilio_bulk_test.csv"
    cat > "$test_csv" <<EOF
phone_number,message
+15551234567,Test message 1
+15551234568,Test message 2
+15551234569,Test message 3
EOF
    
    # Test send-bulk command
    log::info "Testing send-bulk command..."
    if vrooli resource twilio content send-bulk "Test bulk message" "+15551234567" "+15551234568" "+15551234569" 2>&1 | grep -q "Test mode detected"; then
        log::success "send-bulk command works in test mode"
    else
        log::warn "send-bulk command test returned unexpected result"
    fi
    
    # Test send-from-file command
    log::info "Testing send-from-file command..."
    if vrooli resource twilio content send-from-file "$test_csv" 2>&1 | grep -q "Test mode detected"; then
        log::success "send-from-file command works in test mode"
    else
        log::warn "send-from-file command test returned unexpected result"
    fi
    
    # Clean up
    rm -f "$test_csv"
    
    log::success "Bulk SMS functionality test complete"
}

# Run test
test_bulk_sms