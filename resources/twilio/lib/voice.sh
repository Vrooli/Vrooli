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

# Voice call history file
VOICE_HISTORY_FILE="${TWILIO_CONFIG_DIR}/voice_history.json"

# Initialize voice history file if it doesn't exist
if [[ ! -f "$VOICE_HISTORY_FILE" ]]; then
    echo "[]" > "$VOICE_HISTORY_FILE"
fi

# Make a voice call
make_call() {
    local to="${1:-}"
    local message="${2:-}"
    local from="${3:-}"
    local voice="${4:-alice}"  # Default voice
    local language="${5:-en-US}"  # Default language
    
    if [[ -z "$to" || -z "$message" ]]; then
        log::error "Usage: resource-twilio make-call <to-number> <message> [from-number] [voice] [language]"
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
    
    log::info "📞 Making voice call..."
    log::debug "To: $to"
    log::debug "From: $from"
    log::debug "Voice: $voice"
    log::debug "Language: $language"
    
    # Apply rate limiting for voice calls
    if ! rate_limiter::check "api"; then
        log::error "Rate limit exceeded. Please try again later."
        return 1
    fi
    
    # Create TwiML for the message
    local twiml="<Response><Say voice=\"$voice\" language=\"$language\">$message</Say></Response>"
    
    # Create a temporary TwiML bin to host the message
    local twiml_response
    twiml_response=$(twilio api:serverless:v1:services:create \
        --friendly-name "voice-call-$(date +%s)" \
        --unique-name "voice-call-$(date +%s)" 2>&1) || {
        log::error "Failed to create TwiML service: $twiml_response"
        return 1
    }
    
    local service_sid
    service_sid=$(echo "$twiml_response" | grep -oP 'SID: \K[A-Z0-9]+' | head -1)
    
    if [[ -z "$service_sid" ]]; then
        # Fallback: Use TwiML bin if available or simple URL method
        local twiml_url="http://twimlets.com/echo?Twiml=${twiml// /%20}"
        
        # Make the call with TwiML URL
        local call_response
        call_response=$(twilio api:core:calls:create \
            --to "$to" \
            --from "$from" \
            --url "$twiml_url" 2>&1)
        
        local exit_code=$?
    else
        # Use the created service
        local twiml_url="https://${service_sid}.twil.io/voice"
        
        # Make the call
        local call_response
        call_response=$(twilio api:core:calls:create \
            --to "$to" \
            --from "$from" \
            --url "$twiml_url" 2>&1)
        
        local exit_code=$?
        
        # Clean up the service
        twilio api:serverless:v1:services:remove --sid "$service_sid" &> /dev/null || true
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        local call_sid
        call_sid=$(echo "$call_response" | grep -oP 'SID: \K[A-Z0-9]+' | head -1)
        
        if [[ -n "$call_sid" ]]; then
            log::success "✅ Call initiated successfully!"
            log::info "Call SID: $call_sid"
            
            # Log to audit
            audit::log "voice_call" "make_call" "$to" "Call initiated with message" "$call_sid"
            
            # Save to voice history
            local history_entry
            history_entry=$(jq -n \
                --arg sid "$call_sid" \
                --arg to "$to" \
                --arg from "$from" \
                --arg message "$message" \
                --arg voice "$voice" \
                --arg language "$language" \
                --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
                '{
                    sid: $sid,
                    to: $to,
                    from: $from,
                    message: $message,
                    voice: $voice,
                    language: $language,
                    status: "initiated",
                    timestamp: $timestamp
                }')
            
            # Append to history
            jq ". += [$history_entry]" "$VOICE_HISTORY_FILE" > "${VOICE_HISTORY_FILE}.tmp" && \
                mv "${VOICE_HISTORY_FILE}.tmp" "$VOICE_HISTORY_FILE"
            
            # Update rate limiter
            rate_limiter::increment "api"
            
            echo "$call_sid"
            return 0
        else
            log::error "Failed to extract call SID from response"
            return 1
        fi
    else
        log::error "❌ Failed to make call"
        log::error "$call_response"
        
        # Log failure to audit
        audit::log "voice_call" "make_call_failed" "$to" "Failed to initiate call" "$call_response"
        
        return 1
    fi
}

# Make a conference call (multiple participants)
make_conference() {
    local conference_name="${1:-}"
    shift
    local participants=("$@")
    
    if [[ -z "$conference_name" || ${#participants[@]} -eq 0 ]]; then
        log::error "Usage: resource-twilio make-conference <conference-name> <participant1> <participant2> ..."
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
    
    local from
    from=$(twilio::get_default_number)
    if [[ -z "$from" ]]; then
        from=$(jq -r '.default_from_number // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null || echo "")
        if [[ -z "$from" ]]; then
            log::error "No default from number configured"
            return 1
        fi
    fi
    
    log::info "📞 Creating conference: $conference_name"
    log::info "Participants: ${participants[*]}"
    
    # TwiML for conference
    local twiml="<Response><Dial><Conference>$conference_name</Conference></Dial></Response>"
    local twiml_url="http://twimlets.com/echo?Twiml=${twiml// /%20}"
    
    local success_count=0
    local failed_count=0
    
    # Call each participant
    for participant in "${participants[@]}"; do
        # Apply rate limiting
        if ! rate_limiter::check "api"; then
            log::warning "Rate limit exceeded. Waiting..."
            sleep 1
        fi
        
        log::info "Calling $participant..."
        
        local call_response
        call_response=$(twilio api:core:calls:create \
            --to "$participant" \
            --from "$from" \
            --url "$twiml_url" 2>&1)
        
        if [[ $? -eq 0 ]]; then
            local call_sid
            call_sid=$(echo "$call_response" | grep -oP 'SID: \K[A-Z0-9]+' | head -1)
            
            if [[ -n "$call_sid" ]]; then
                log::success "✅ Added $participant to conference (SID: $call_sid)"
                ((success_count++))
                
                # Log to audit
                audit::log "conference_call" "add_participant" "$participant" "Added to conference: $conference_name" "$call_sid"
            else
                log::error "Failed to add $participant to conference"
                ((failed_count++))
            fi
        else
            log::error "Failed to call $participant: $call_response"
            ((failed_count++))
        fi
        
        # Update rate limiter
        rate_limiter::increment "api"
    done
    
    log::info "Conference created: $success_count participants added, $failed_count failed"
    
    if [[ $success_count -gt 0 ]]; then
        return 0
    else
        return 1
    fi
}

# Get voice call history
get_voice_history() {
    local limit="${1:-10}"
    
    if [[ ! -f "$VOICE_HISTORY_FILE" ]]; then
        echo "[]"
        return 0
    fi
    
    jq --arg limit "$limit" '
        . | reverse | limit($limit | tonumber; .)
    ' "$VOICE_HISTORY_FILE"
}

# Get voice call statistics
get_voice_stats() {
    if [[ ! -f "$VOICE_HISTORY_FILE" ]]; then
        echo "{\"total\": 0}"
        return 0
    fi
    
    jq '
        {
            total: length,
            by_status: group_by(.status) | map({key: .[0].status, value: length}) | from_entries,
            by_voice: group_by(.voice) | map({key: .[0].voice, value: length}) | from_entries,
            by_language: group_by(.language) | map({key: .[0].language, value: length}) | from_entries
        }
    ' "$VOICE_HISTORY_FILE"
}

# Update call status from Twilio API
update_call_status() {
    local call_sid="${1:-}"
    
    if [[ -z "$call_sid" ]]; then
        log::error "Usage: update_call_status <call-sid>"
        return 1
    fi
    
    # Check if installed
    if ! twilio::is_installed; then
        log::error "$MSG_TWILIO_NOT_INSTALLED"
        return 1
    fi
    
    # Get call details from API
    local call_details
    call_details=$(twilio api:core:calls:fetch --sid "$call_sid" -o json 2>&1)
    
    if [[ $? -eq 0 ]]; then
        local status
        status=$(echo "$call_details" | jq -r '.status // "unknown"')
        local duration
        duration=$(echo "$call_details" | jq -r '.duration // "0"')
        
        # Update history file
        if [[ -f "$VOICE_HISTORY_FILE" ]]; then
            jq --arg sid "$call_sid" \
               --arg status "$status" \
               --arg duration "$duration" \
               '(.[] | select(.sid == $sid)) |= . + {status: $status, duration: $duration}' \
               "$VOICE_HISTORY_FILE" > "${VOICE_HISTORY_FILE}.tmp" && \
                mv "${VOICE_HISTORY_FILE}.tmp" "$VOICE_HISTORY_FILE"
            
            log::success "Updated call status: $status (duration: ${duration}s)"
        fi
        
        return 0
    else
        log::error "Failed to fetch call details: $call_details"
        return 1
    fi
}

# Export voice history to CSV
export_voice_history() {
    local output_file="${1:-voice_history.csv}"
    
    if [[ ! -f "$VOICE_HISTORY_FILE" ]]; then
        log::error "No voice history found"
        return 1
    fi
    
    # Create CSV header
    echo "Call SID,To,From,Message,Voice,Language,Status,Duration,Timestamp" > "$output_file"
    
    # Export data
    jq -r '.[] | [.sid, .to, .from, .message, .voice, .language, .status, (.duration // "N/A"), .timestamp] | @csv' \
        "$VOICE_HISTORY_FILE" >> "$output_file"
    
    log::success "Voice history exported to $output_file"
    return 0
}