#!/usr/bin/env bash
################################################################################
# Twilio Rate Limit Configuration
# 
# Manage rate limiting settings for Twilio resource
################################################################################

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_DIR="${APP_ROOT}/resources/twilio"
TWILIO_CONFIG_DIR="${var_vrooli_dir:-${HOME}/.vrooli}/twilio"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${TWILIO_DIR}/lib/rate-limiter.sh"

# Configuration file
RATE_CONFIG_FILE="${TWILIO_CONFIG_DIR}/rate-limits.conf"

################################################################################
# Configuration Management
################################################################################

# Load rate limit configuration
twilio::rate_config::load() {
    if [[ -f "$RATE_CONFIG_FILE" ]]; then
        source "$RATE_CONFIG_FILE"
        log::info "Loaded rate limit configuration from: $RATE_CONFIG_FILE"
    else
        log::info "Using default rate limits"
    fi
}

# Save current rate limits to config
twilio::rate_config::save() {
    mkdir -p "$TWILIO_CONFIG_DIR"
    
    cat > "$RATE_CONFIG_FILE" <<EOF
# Twilio Rate Limit Configuration
# Generated: $(date)

# SMS rate limit (messages per second)
export TWILIO_RATE_LIMIT_SMS="${TWILIO_RATE_LIMIT_SMS}"

# API rate limit (calls per second)
export TWILIO_RATE_LIMIT_API="${TWILIO_RATE_LIMIT_API}"

# Bulk operation rate limit (operations per second)
export TWILIO_RATE_LIMIT_BULK="${TWILIO_RATE_LIMIT_BULK}"
EOF
    
    log::success "Rate limit configuration saved to: $RATE_CONFIG_FILE"
}

# Set a specific rate limit
twilio::rate_config::set() {
    local type="${1:-}"
    local value="${2:-}"
    
    if [[ -z "$type" ]] || [[ -z "$value" ]]; then
        log::error "Usage: rate-config set <sms|api|bulk> <value>"
        return 1
    fi
    
    # Validate value is a positive integer
    if ! [[ "$value" =~ ^[1-9][0-9]*$ ]]; then
        log::error "Rate limit must be a positive integer"
        return 1
    fi
    
    case "$type" in
        sms)
            export TWILIO_RATE_LIMIT_SMS="$value"
            log::success "SMS rate limit set to: $value/s"
            ;;
        api)
            export TWILIO_RATE_LIMIT_API="$value"
            log::success "API rate limit set to: $value/s"
            ;;
        bulk)
            export TWILIO_RATE_LIMIT_BULK="$value"
            log::success "Bulk rate limit set to: $value/s"
            ;;
        *)
            log::error "Unknown rate limit type: $type"
            log::info "Valid types: sms, api, bulk"
            return 1
            ;;
    esac
    
    # Save configuration
    twilio::rate_config::save
}

# Display current rate limits
twilio::rate_config::show() {
    log::header "Current Rate Limits"
    echo "SMS:  ${TWILIO_RATE_LIMIT_SMS}/s"
    echo "API:  ${TWILIO_RATE_LIMIT_API}/s"
    echo "Bulk: ${TWILIO_RATE_LIMIT_BULK}/s"
    echo
    echo "Config file: $RATE_CONFIG_FILE"
}

# Reset to defaults
twilio::rate_config::reset() {
    export TWILIO_RATE_LIMIT_SMS="10"
    export TWILIO_RATE_LIMIT_API="100"
    export TWILIO_RATE_LIMIT_BULK="5"
    
    twilio::rate_config::save
    log::success "Rate limits reset to defaults"
    twilio::rate_config::show
}

################################################################################
# Auto-load configuration on source
################################################################################

# Load configuration automatically
twilio::rate_config::load

################################################################################
# Export Functions
################################################################################

export -f twilio::rate_config::load
export -f twilio::rate_config::save
export -f twilio::rate_config::set
export -f twilio::rate_config::show
export -f twilio::rate_config::reset