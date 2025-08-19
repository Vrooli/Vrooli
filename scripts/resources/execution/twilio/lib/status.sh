#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
TWILIO_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# Check Twilio status
twilio::status() {
    local format="${1:-text}"
    local verbose="${VERBOSE:-false}"
    
    # Prepare status data
    local installed="false"
    local running="false"
    local health="unhealthy"
    local version="not installed"
    local credentials_configured="false"
    local phone_numbers="0"
    local account_sid=""
    
    if twilio::is_installed; then
        installed="true"
        version=$(twilio::get_version)
        
        if twilio::has_credentials; then
            credentials_configured="true"
            account_sid=$(twilio::get_account_sid)
            
            # Twilio is a stateless service - if credentials are configured, it's "running"
            # For test credentials, we can't validate with the real API
            if [[ "$account_sid" == "AC_test_"* ]]; then
                # Test mode - consider it healthy if credentials exist
                running="true"
                health="healthy"
            else
                # Production mode - try to connect to Twilio API
                local cmd
                if cmd=$(twilio::get_command 2>/dev/null); then
                    if twilio::setup_auth && timeout 5 "$cmd" api:core:accounts:list --limit 1 >/dev/null 2>&1; then
                        health="healthy"
                        running="true"
                    else
                        health="auth_error"
                    fi
                fi
            fi
        else
            health="no_credentials"
        fi
        
        # Count phone numbers
        if [[ -f "$TWILIO_PHONE_NUMBERS_FILE" ]]; then
            phone_numbers=$(jq '.numbers | length' "$TWILIO_PHONE_NUMBERS_FILE" 2>/dev/null || echo "0")
        fi
    fi
    
    # Format output based on requested format
    if [[ "$format" == "json" ]]; then
        # Use format.sh for consistent JSON output
        format::key_value json \
            name "$TWILIO_NAME" \
            category "$TWILIO_CATEGORY" \
            status "$( [[ "$running" == "true" ]] && echo "running" || echo "stopped" )" \
            running "$running" \
            health "$health" \
            version "$version" \
            installed "$installed" \
            credentials_configured "$credentials_configured" \
            phone_numbers "$phone_numbers" \
            account_sid "$account_sid"
    else
        # Text format
        log::header "📱 Twilio Status"
        log::info "📝 Description: SMS, voice, and messaging automation"
        log::info "📂 Category: $TWILIO_CATEGORY"
        echo
        
        log::info "📊 Basic Status:"
        if [[ "$installed" == "true" ]]; then
            log::success "   ✅ Installed: Yes (v$version)"
        else
            log::error "   ❌ Installed: No"
        fi
        
        if [[ "$credentials_configured" == "true" ]]; then
            log::success "   ✅ Credentials: Configured"
            if [[ -n "$account_sid" ]]; then
                log::info "   📝 Account: ${account_sid:0:10}..."
            fi
        else
            log::warn "   ⚠️  Credentials: Not configured"
        fi
        
        if [[ "$running" == "true" ]]; then
            log::success "   ✅ API Access: Working"
        else
            log::warn "   ⚠️  API Access: Not available"
        fi
        
        echo
        log::info "📞 Phone Numbers: $phone_numbers configured"
        
        if [[ "$verbose" == "true" && "$installed" == "true" ]]; then
            echo
            log::info "📁 Configuration:"
            log::info "   Config: $TWILIO_CONFIG_DIR"
            log::info "   Data: $TWILIO_DATA_DIR"
            log::info "   Credentials: $TWILIO_CREDENTIALS_FILE"
        fi
    fi
}

# Parse command line arguments - wrapper for CLI use
check_status() {
    local format="text"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                format="json"
                shift
                ;;
            --format)
                format="${2:-text}"
                shift 2
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    twilio::status "$format"
}