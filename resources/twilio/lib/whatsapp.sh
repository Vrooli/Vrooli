#!/usr/bin/env bash
set -uo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_LIB_DIR="${APP_ROOT}/resources/twilio/lib"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"
source "$TWILIO_LIB_DIR/history.sh"
source "$TWILIO_LIB_DIR/rate-limiter.sh"
source "$TWILIO_LIB_DIR/audit.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# WhatsApp history file
WHATSAPP_HISTORY_FILE="${TWILIO_CONFIG_DIR}/whatsapp_history.json"

# Initialize WhatsApp history file if it doesn't exist
if [[ ! -f "$WHATSAPP_HISTORY_FILE" ]]; then
    echo "[]" > "$WHATSAPP_HISTORY_FILE"
fi

# Send WhatsApp message
send_whatsapp() {
    local to="${1:-}"
    local message="${2:-}"
    local from="${3:-}"
    local media_url="${4:-}"  # Optional media URL
    
    if [[ -z "$to" || -z "$message" ]]; then
        log::error "Usage: resource-twilio send-whatsapp <to-number> <message> [from-number] [media-url]"
        log::info "Note: WhatsApp numbers must be in format: whatsapp:+1234567890"
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
    
    # Format WhatsApp numbers if needed
    if [[ ! "$to" =~ ^whatsapp: ]]; then
        to="whatsapp:$to"
    fi
    
    # Get from number if not provided
    if [[ -z "$from" ]]; then
        # Try to get WhatsApp-enabled number from config
        from=$(jq -r '.whatsapp_from_number // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null || echo "")
        if [[ -z "$from" ]]; then
            # Fall back to default number with whatsapp prefix
            local default_num
            default_num=$(twilio::get_default_number)
            if [[ -n "$default_num" ]]; then
                from="whatsapp:$default_num"
            else
                log::error "No WhatsApp from number configured"
                log::info "Add 'whatsapp_from_number' to your Twilio config or provide from number"
                return 1
            fi
        fi
    fi
    
    # Format from number if needed
    if [[ ! "$from" =~ ^whatsapp: ]]; then
        from="whatsapp:$from"
    fi
    
    log::info "💬 Sending WhatsApp message..."
    log::debug "To: $to"
    log::debug "From: $from"
    if [[ -n "$media_url" ]]; then
        log::debug "Media: $media_url"
    fi
    
    # Apply rate limiting
    if ! rate_limiter::check "api"; then
        log::error "Rate limit exceeded. Please try again later."
        return 1
    fi
    
    # Build command arguments
    local cmd_args=(
        "--to" "$to"
        "--from" "$from"
        "--body" "$message"
    )
    
    # Add media URL if provided
    if [[ -n "$media_url" ]]; then
        cmd_args+=("--media-url" "$media_url")
    fi
    
    # Send the message
    local response
    response=$(twilio api:core:messages:create "${cmd_args[@]}" 2>&1)
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        local message_sid
        message_sid=$(echo "$response" | grep -oP 'SID: \K[A-Z0-9]+' | head -1)
        
        if [[ -n "$message_sid" ]]; then
            log::success "✅ WhatsApp message sent successfully!"
            log::info "Message SID: $message_sid"
            
            # Log to audit
            audit::log "whatsapp" "send_message" "$to" "WhatsApp message sent" "$message_sid"
            
            # Save to WhatsApp history
            local history_entry
            history_entry=$(jq -n \
                --arg sid "$message_sid" \
                --arg to "$to" \
                --arg from "$from" \
                --arg body "$message" \
                --arg media_url "$media_url" \
                --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
                '{
                    sid: $sid,
                    to: $to,
                    from: $from,
                    body: $body,
                    media_url: $media_url,
                    status: "sent",
                    timestamp: $timestamp
                }')
            
            # Append to history
            jq ". += [$history_entry]" "$WHATSAPP_HISTORY_FILE" > "${WHATSAPP_HISTORY_FILE}.tmp" && \
                mv "${WHATSAPP_HISTORY_FILE}.tmp" "$WHATSAPP_HISTORY_FILE"
            
            # Update rate limiter
            rate_limiter::increment "api"
            
            echo "$message_sid"
            return 0
        else
            log::error "Failed to extract message SID from response"
            return 1
        fi
    else
        log::error "❌ Failed to send WhatsApp message"
        log::error "$response"
        
        # Check for common errors
        if echo "$response" | grep -q "unverified"; then
            log::info "💡 Tip: Make sure the recipient's number is verified in WhatsApp sandbox"
            log::info "   Join sandbox at: https://www.twilio.com/console/sms/whatsapp/sandbox"
        fi
        
        # Log failure to audit
        audit::log "whatsapp" "send_failed" "$to" "Failed to send WhatsApp message" "$response"
        
        return 1
    fi
}

# Send WhatsApp template message
send_whatsapp_template() {
    local to="${1:-}"
    local template_sid="${2:-}"
    local from="${3:-}"
    shift 3
    local template_vars=("$@")
    
    if [[ -z "$to" || -z "$template_sid" ]]; then
        log::error "Usage: resource-twilio send-whatsapp-template <to-number> <template-sid> [from-number] [var1] [var2] ..."
        log::info "Note: Template SID format: HX... (from Twilio Console)"
        return 1
    fi
    
    # Check if installed
    if ! twilio::is_installed; then
        log::error "$MSG_TWILIO_NOT_INSTALLED"
        return 1
    fi
    
    # Format WhatsApp numbers
    if [[ ! "$to" =~ ^whatsapp: ]]; then
        to="whatsapp:$to"
    fi
    
    # Get from number if not provided
    if [[ -z "$from" ]]; then
        from=$(jq -r '.whatsapp_from_number // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null || echo "")
        if [[ -z "$from" ]]; then
            local default_num
            default_num=$(twilio::get_default_number)
            if [[ -n "$default_num" ]]; then
                from="whatsapp:$default_num"
            else
                log::error "No WhatsApp from number configured"
                return 1
            fi
        fi
    fi
    
    if [[ ! "$from" =~ ^whatsapp: ]]; then
        from="whatsapp:$from"
    fi
    
    log::info "💬 Sending WhatsApp template message..."
    log::debug "To: $to"
    log::debug "Template: $template_sid"
    
    # Apply rate limiting
    if ! rate_limiter::check "api"; then
        log::error "Rate limit exceeded. Please try again later."
        return 1
    fi
    
    # Build content variables JSON
    local content_variables="{}"
    local idx=1
    for var in "${template_vars[@]}"; do
        content_variables=$(echo "$content_variables" | jq --arg key "$idx" --arg val "$var" '. + {($key): $val}')
        ((idx++))
    done
    
    # Send template message
    local response
    response=$(twilio api:content:v1:contents:create \
        --to "$to" \
        --from "$from" \
        --content-sid "$template_sid" \
        --content-variables "$content_variables" 2>&1)
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        local message_sid
        message_sid=$(echo "$response" | grep -oP 'SID: \K[A-Z0-9]+' | head -1)
        
        if [[ -n "$message_sid" ]]; then
            log::success "✅ WhatsApp template message sent!"
            log::info "Message SID: $message_sid"
            
            # Log to audit
            audit::log "whatsapp_template" "send_template" "$to" "Template sent: $template_sid" "$message_sid"
            
            # Update rate limiter
            rate_limiter::increment "api"
            
            echo "$message_sid"
            return 0
        else
            log::error "Failed to extract message SID"
            return 1
        fi
    else
        log::error "❌ Failed to send WhatsApp template"
        log::error "$response"
        return 1
    fi
}

# Send bulk WhatsApp messages
send_whatsapp_bulk() {
    local message="${1:-}"
    shift
    local recipients=("$@")
    
    if [[ -z "$message" || ${#recipients[@]} -eq 0 ]]; then
        log::error "Usage: resource-twilio send-whatsapp-bulk <message> <recipient1> <recipient2> ..."
        return 1
    fi
    
    log::info "📱 Sending bulk WhatsApp messages to ${#recipients[@]} recipients..."
    
    local success_count=0
    local failed_count=0
    
    for recipient in "${recipients[@]}"; do
        # Apply rate limiting
        if ! rate_limiter::check "bulk"; then
            log::info "Rate limit reached, waiting..."
            sleep 0.5
        fi
        
        if send_whatsapp "$recipient" "$message" > /dev/null 2>&1; then
            log::success "✅ Sent to $recipient"
            ((success_count++))
        else
            log::error "❌ Failed to send to $recipient"
            ((failed_count++))
        fi
        
        # Small delay between messages
        sleep 0.1
    done
    
    log::info "Bulk send complete: $success_count successful, $failed_count failed"
    
    if [[ $success_count -gt 0 ]]; then
        return 0
    else
        return 1
    fi
}

# Get WhatsApp message history
get_whatsapp_history() {
    local limit="${1:-10}"
    
    if [[ ! -f "$WHATSAPP_HISTORY_FILE" ]]; then
        echo "[]"
        return 0
    fi
    
    jq --arg limit "$limit" '
        . | reverse | limit($limit | tonumber; .)
    ' "$WHATSAPP_HISTORY_FILE"
}

# Get WhatsApp statistics
get_whatsapp_stats() {
    if [[ ! -f "$WHATSAPP_HISTORY_FILE" ]]; then
        echo "{\"total\": 0}"
        return 0
    fi
    
    jq '
        {
            total: length,
            with_media: [.[] | select(.media_url != "")] | length,
            by_status: group_by(.status) | map({key: .[0].status, value: length}) | from_entries,
            today: [.[] | select(.timestamp | startswith((now | strftime("%Y-%m-%d"))))] | length
        }
    ' "$WHATSAPP_HISTORY_FILE"
}

# Update WhatsApp message status
update_whatsapp_status() {
    local message_sid="${1:-}"
    
    if [[ -z "$message_sid" ]]; then
        log::error "Usage: update_whatsapp_status <message-sid>"
        return 1
    fi
    
    # Check if installed
    if ! twilio::is_installed; then
        log::error "$MSG_TWILIO_NOT_INSTALLED"
        return 1
    fi
    
    # Get message details from API
    local message_details
    message_details=$(twilio api:core:messages:fetch --sid "$message_sid" -o json 2>&1)
    
    if [[ $? -eq 0 ]]; then
        local status
        status=$(echo "$message_details" | jq -r '.status // "unknown"')
        
        # Update history file
        if [[ -f "$WHATSAPP_HISTORY_FILE" ]]; then
            jq --arg sid "$message_sid" \
               --arg status "$status" \
               '(.[] | select(.sid == $sid)) |= . + {status: $status}' \
               "$WHATSAPP_HISTORY_FILE" > "${WHATSAPP_HISTORY_FILE}.tmp" && \
                mv "${WHATSAPP_HISTORY_FILE}.tmp" "$WHATSAPP_HISTORY_FILE"
            
            log::success "Updated message status: $status"
        fi
        
        return 0
    else
        log::error "Failed to fetch message details: $message_details"
        return 1
    fi
}

# Export WhatsApp history to CSV
export_whatsapp_history() {
    local output_file="${1:-whatsapp_history.csv}"
    
    if [[ ! -f "$WHATSAPP_HISTORY_FILE" ]]; then
        log::error "No WhatsApp history found"
        return 1
    fi
    
    # Create CSV header
    echo "Message SID,To,From,Body,Media URL,Status,Timestamp" > "$output_file"
    
    # Export data
    jq -r '.[] | [.sid, .to, .from, .body, .media_url, .status, .timestamp] | @csv' \
        "$WHATSAPP_HISTORY_FILE" >> "$output_file"
    
    log::success "WhatsApp history exported to $output_file"
    return 0
}

# WhatsApp sandbox setup helper
setup_whatsapp_sandbox() {
    log::info "📱 WhatsApp Sandbox Setup"
    log::info ""
    log::info "To use WhatsApp with Twilio, you need to set up the sandbox:"
    log::info ""
    log::info "1. Go to: https://www.twilio.com/console/sms/whatsapp/sandbox"
    log::info "2. Follow the instructions to join the sandbox"
    log::info "3. Save your sandbox number in the config as 'whatsapp_from_number'"
    log::info ""
    log::info "For production use:"
    log::info "1. Request WhatsApp Business API access"
    log::info "2. Submit your business for approval"
    log::info "3. Configure your WhatsApp Business number"
    log::info ""
    log::info "Current configuration:"
    
    local whatsapp_num
    whatsapp_num=$(jq -r '.whatsapp_from_number // "Not configured"' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null || echo "Not configured")
    log::info "WhatsApp From Number: $whatsapp_num"
    
    return 0
}