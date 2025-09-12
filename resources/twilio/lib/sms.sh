#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_LIB_DIR="${APP_ROOT}/resources/twilio/lib"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"
source "$TWILIO_LIB_DIR/history.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

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
            # Send message and capture response to get SID
            local response
            response=$("$cmd" api:core:messages:create \
                --from "$from" \
                --to "$to" \
                --body "$message" 2>&1)
            
            if [[ $? -eq 0 ]]; then
                # Extract SID from response
                local sid
                sid=$(echo "$response" | grep -oE 'SM[a-f0-9]{32}' | head -1)
                
                # Log to history
                if [[ -n "$sid" ]]; then
                    twilio::history::log_message "$sid" "$from" "$to" "$message" "sent"
                else
                    # Generate a local ID if we can't extract SID
                    sid="LOCAL_$(date +%s)_$(echo "$to" | md5sum | cut -c1-8)"
                    twilio::history::log_message "$sid" "$from" "$to" "$message" "sent"
                fi
                
                log::success "SMS sent successfully! (SID: $sid)"
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

# Send bulk SMS to multiple recipients
send_bulk_sms() {
    local message="${1:-}"
    local from="${2:-}"
    shift 2
    local recipients=("$@")
    
    if [[ -z "$message" ]] || [[ ${#recipients[@]} -eq 0 ]]; then
        log::error "Usage: resource-twilio send-bulk <message> [from-number] <recipient1> <recipient2> ..."
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
    if [[ -z "$from" ]] || [[ "$from" =~ ^\+ ]]; then
        # from is actually a recipient, shift it back
        if [[ -n "$from" ]]; then
            recipients=("$from" "${recipients[@]}")
        fi
        from=$(twilio::get_default_number)
        if [[ -z "$from" ]]; then
            from=$(jq -r '.default_from_number // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null || echo "")
            if [[ -z "$from" ]]; then
                log::error "No from number specified and no default configured"
                return 1
            fi
        fi
    fi
    
    log::info "📱 Sending bulk SMS to ${#recipients[@]} recipients..."
    log::info "   From: $from"
    log::info "   Message: ${message:0:50}..."
    
    # Check for test mode
    if [[ "${TWILIO_ACCOUNT_SID:-}" =~ test ]] || [[ "${TWILIO_ACCOUNT_SID:-}" == "AC_test_"* ]]; then
        log::info "Test mode detected - simulating bulk SMS send"
        log::success "Would send to ${#recipients[@]} recipients in production"
        return 0
    fi
    
    # Set up auth once
    if ! twilio::setup_auth; then
        log::error "Authentication failed"
        return 1
    fi
    
    local cmd
    cmd=$(twilio::get_command)
    
    # Update phone number settings once
    "$cmd" phone-numbers:update "$from" --sms-url="" 2>/dev/null || true
    
    local success_count=0
    local fail_count=0
    local total=${#recipients[@]}
    
    # Send to each recipient with rate limiting
    for recipient in "${recipients[@]}"; do
        log::info "   Sending to: $recipient ($((success_count + fail_count + 1))/$total)"
        
        local response
        response=$("$cmd" api:core:messages:create \
            --from "$from" \
            --to "$recipient" \
            --body "$message" 2>&1)
        
        if [[ $? -eq 0 ]]; then
            ((success_count++))
            # Extract SID and log to history
            local sid
            sid=$(echo "$response" | grep -oE 'SM[a-f0-9]{32}' | head -1)
            if [[ -n "$sid" ]]; then
                twilio::history::log_message "$sid" "$from" "$recipient" "$message" "sent"
            else
                sid="BULK_$(date +%s)_$(echo "$recipient" | md5sum | cut -c1-8)"
                twilio::history::log_message "$sid" "$from" "$recipient" "$message" "sent"
            fi
        else
            log::warn "   Failed to send to: $recipient"
            ((fail_count++))
            # Log failure to history
            local sid="FAILED_$(date +%s)_$(echo "$recipient" | md5sum | cut -c1-8)"
            twilio::history::log_message "$sid" "$from" "$recipient" "$message" "failed"
        fi
        
        # Rate limiting: Wait 100ms between messages to avoid hitting API limits
        sleep 0.1
    done
    
    log::info "📊 Bulk SMS Results:"
    log::success "   Successful: $success_count"
    if [[ $fail_count -gt 0 ]]; then
        log::warn "   Failed: $fail_count"
    fi
    
    if [[ $fail_count -eq 0 ]]; then
        log::success "All messages sent successfully!"
        return 0
    elif [[ $success_count -gt 0 ]]; then
        log::warn "Partial success: $success_count/$total messages sent"
        return 1
    else
        log::error "Failed to send any messages"
        return 1
    fi
}

# Send SMS from file (CSV format)
send_sms_from_file() {
    local file="${1:-}"
    local from="${2:-}"
    
    if [[ -z "$file" ]] || [[ ! -f "$file" ]]; then
        log::error "Usage: resource-twilio send-from-file <csv-file> [from-number]"
        log::error "CSV format: phone_number,message"
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
    
    log::info "📱 Sending SMS from file: $file"
    log::info "   From: $from"
    
    # Check for test mode
    if [[ "${TWILIO_ACCOUNT_SID:-}" =~ test ]] || [[ "${TWILIO_ACCOUNT_SID:-}" == "AC_test_"* ]]; then
        log::info "Test mode detected - simulating CSV file processing"
        local line_count=$(wc -l < "$file")
        log::success "Would send $line_count messages in production"
        return 0
    fi
    
    # Set up auth once
    if ! twilio::setup_auth; then
        log::error "Authentication failed"
        return 1
    fi
    
    local cmd
    cmd=$(twilio::get_command)
    
    # Update phone number settings once
    "$cmd" phone-numbers:update "$from" --sms-url="" 2>/dev/null || true
    
    local success_count=0
    local fail_count=0
    local line_count=0
    
    # Process CSV file
    while IFS=, read -r phone_number message; do
        ((line_count++))
        
        # Skip header if present
        if [[ $line_count -eq 1 ]] && [[ "$phone_number" == "phone_number" ]]; then
            continue
        fi
        
        # Skip empty lines
        if [[ -z "$phone_number" ]] || [[ -z "$message" ]]; then
            continue
        fi
        
        # Clean up phone number and message
        phone_number=$(echo "$phone_number" | tr -d '"' | tr -d ' ')
        message=$(echo "$message" | tr -d '"')
        
        log::info "   Sending to: $phone_number"
        
        local response
        response=$("$cmd" api:core:messages:create \
            --from "$from" \
            --to "$phone_number" \
            --body "$message" 2>&1)
        
        if [[ $? -eq 0 ]]; then
            ((success_count++))
            # Extract SID and log to history
            local sid
            sid=$(echo "$response" | grep -oE 'SM[a-f0-9]{32}' | head -1)
            if [[ -n "$sid" ]]; then
                twilio::history::log_message "$sid" "$from" "$phone_number" "$message" "sent"
            else
                sid="CSV_$(date +%s)_$(echo "$phone_number" | md5sum | cut -c1-8)"
                twilio::history::log_message "$sid" "$from" "$phone_number" "$message" "sent"
            fi
        else
            log::warn "   Failed to send to: $phone_number"
            ((fail_count++))
            # Log failure to history
            local sid="FAILED_$(date +%s)_$(echo "$phone_number" | md5sum | cut -c1-8)"
            twilio::history::log_message "$sid" "$from" "$phone_number" "$message" "failed"
        fi
        
        # Rate limiting
        sleep 0.1
    done < "$file"
    
    log::info "📊 Bulk SMS Results:"
    log::success "   Successful: $success_count"
    if [[ $fail_count -gt 0 ]]; then
        log::warn "   Failed: $fail_count"
    fi
    
    if [[ $fail_count -eq 0 ]] && [[ $success_count -gt 0 ]]; then
        log::success "All messages sent successfully!"
        return 0
    elif [[ $success_count -gt 0 ]]; then
        log::warn "Partial success: $success_count messages sent"
        return 1
    else
        log::error "Failed to send any messages"
        return 1
    fi
}

twilio::send_sms() {
    send_sms "$@"
}

twilio::send_bulk_sms() {
    send_bulk_sms "$@"
}

twilio::send_sms_from_file() {
    send_sms_from_file "$@"
}

