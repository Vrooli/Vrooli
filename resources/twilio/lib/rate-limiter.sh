#!/usr/bin/env bash
################################################################################
# Twilio Rate Limiter Library
# 
# Advanced rate limiting for Twilio API operations
################################################################################

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_DIR="${APP_ROOT}/resources/twilio"
TWILIO_LIB_DIR="${TWILIO_DIR}/lib"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Rate Limiting Configuration
################################################################################

# Default rate limits (per second)
export TWILIO_RATE_LIMIT_SMS="${TWILIO_RATE_LIMIT_SMS:-10}"  # 10 SMS per second
export TWILIO_RATE_LIMIT_API="${TWILIO_RATE_LIMIT_API:-100}" # 100 API calls per second
export TWILIO_RATE_LIMIT_BULK="${TWILIO_RATE_LIMIT_BULK:-5}"  # 5 bulk operations per second

# Rate limit tracking
declare -gA TWILIO_RATE_TRACKER
TWILIO_RATE_TRACKER[last_sms]=0
TWILIO_RATE_TRACKER[sms_count]=0
TWILIO_RATE_TRACKER[last_api]=0
TWILIO_RATE_TRACKER[api_count]=0

################################################################################
# Rate Limiting Functions
################################################################################

# Calculate wait time based on rate limit
twilio::rate_limiter::calculate_wait() {
    local rate_limit="${1:-10}"
    local current_time
    current_time=$(date +%s%N)  # Nanoseconds
    
    # Calculate minimum interval in nanoseconds
    local min_interval=$((1000000000 / rate_limit))  # 1 second in nanoseconds / rate
    
    echo "$min_interval"
}

# Wait if necessary to respect rate limits
twilio::rate_limiter::wait() {
    local operation="${1:-api}"  # sms, api, bulk
    local custom_rate="${2:-}"   # Optional custom rate limit
    
    local rate_limit
    case "$operation" in
        sms)
            rate_limit="${custom_rate:-$TWILIO_RATE_LIMIT_SMS}"
            ;;
        bulk)
            rate_limit="${custom_rate:-$TWILIO_RATE_LIMIT_BULK}"
            ;;
        *)
            rate_limit="${custom_rate:-$TWILIO_RATE_LIMIT_API}"
            ;;
    esac
    
    # Get current time in nanoseconds
    local current_time
    current_time=$(date +%s%N)
    
    # Get last operation time
    local last_time="${TWILIO_RATE_TRACKER[last_${operation}]:-0}"
    
    # Calculate required wait time
    local min_interval
    min_interval=$(twilio::rate_limiter::calculate_wait "$rate_limit")
    
    local elapsed=$((current_time - last_time))
    
    if [[ $elapsed -lt $min_interval ]]; then
        local wait_time=$((min_interval - elapsed))
        # Convert nanoseconds to seconds for sleep
        local wait_seconds=$(echo "scale=9; $wait_time / 1000000000" | bc)
        
        if [[ -n "${TWILIO_DEBUG:-}" ]]; then
            log::debug "Rate limiting: waiting ${wait_seconds}s for $operation"
        fi
        
        sleep "$wait_seconds"
    fi
    
    # Update last operation time
    TWILIO_RATE_TRACKER[last_${operation}]=$(date +%s%N)
}

# Adaptive rate limiting based on error responses
twilio::rate_limiter::adaptive_wait() {
    local operation="${1:-api}"
    local error_code="${2:-0}"
    
    # If we hit rate limit error (429), back off more aggressively
    if [[ "$error_code" -eq 429 ]]; then
        log::warn "Rate limit hit, backing off..."
        sleep 2
        # Reduce rate limit temporarily
        case "$operation" in
            sms)
                TWILIO_RATE_LIMIT_SMS=$((TWILIO_RATE_LIMIT_SMS / 2))
                log::info "Reduced SMS rate limit to: $TWILIO_RATE_LIMIT_SMS/s"
                ;;
            bulk)
                TWILIO_RATE_LIMIT_BULK=$((TWILIO_RATE_LIMIT_BULK / 2))
                log::info "Reduced bulk rate limit to: $TWILIO_RATE_LIMIT_BULK/s"
                ;;
            *)
                TWILIO_RATE_LIMIT_API=$((TWILIO_RATE_LIMIT_API / 2))
                log::info "Reduced API rate limit to: $TWILIO_RATE_LIMIT_API/s"
                ;;
        esac
    else
        # Normal rate limiting
        twilio::rate_limiter::wait "$operation"
    fi
}

# Reset rate limits to defaults
twilio::rate_limiter::reset() {
    TWILIO_RATE_LIMIT_SMS="${TWILIO_RATE_LIMIT_SMS_DEFAULT:-10}"
    TWILIO_RATE_LIMIT_API="${TWILIO_RATE_LIMIT_API_DEFAULT:-100}"
    TWILIO_RATE_LIMIT_BULK="${TWILIO_RATE_LIMIT_BULK_DEFAULT:-5}"
    
    # Clear tracking
    TWILIO_RATE_TRACKER[last_sms]=0
    TWILIO_RATE_TRACKER[sms_count]=0
    TWILIO_RATE_TRACKER[last_api]=0
    TWILIO_RATE_TRACKER[api_count]=0
    
    log::info "Rate limits reset to defaults"
}

# Get current rate limit info
twilio::rate_limiter::info() {
    echo "Current Rate Limits:"
    echo "  SMS: ${TWILIO_RATE_LIMIT_SMS}/s"
    echo "  API: ${TWILIO_RATE_LIMIT_API}/s"
    echo "  Bulk: ${TWILIO_RATE_LIMIT_BULK}/s"
}

################################################################################
# Batch Processing with Rate Limiting
################################################################################

# Process items in batches with rate limiting
twilio::rate_limiter::batch_process() {
    local -n items=$1  # Array of items to process (nameref)
    local batch_size="${2:-10}"
    local operation="${3:-api}"
    local callback="${4:-echo}"  # Function to call for each item
    
    local total=${#items[@]}
    local processed=0
    local failed=0
    
    log::info "Processing $total items in batches of $batch_size"
    
    for ((i=0; i<total; i+=batch_size)); do
        local batch_end=$((i + batch_size))
        if [[ $batch_end -gt $total ]]; then
            batch_end=$total
        fi
        
        log::info "Processing batch: $((i+1))-$batch_end of $total"
        
        for ((j=i; j<batch_end; j++)); do
            # Rate limit before each operation
            twilio::rate_limiter::wait "$operation"
            
            # Process item
            if $callback "${items[$j]}"; then
                processed=$((processed + 1))
            else
                failed=$((failed + 1))
                # On failure, apply adaptive rate limiting
                twilio::rate_limiter::adaptive_wait "$operation" 429
            fi
        done
        
        # Brief pause between batches
        sleep 0.5
    done
    
    log::info "Batch processing complete: $processed successful, $failed failed"
    return $([[ $failed -eq 0 ]] && echo 0 || echo 1)
}

################################################################################
# Export Functions
################################################################################

export -f twilio::rate_limiter::calculate_wait
export -f twilio::rate_limiter::wait
export -f twilio::rate_limiter::adaptive_wait
export -f twilio::rate_limiter::reset
export -f twilio::rate_limiter::info
export -f twilio::rate_limiter::batch_process