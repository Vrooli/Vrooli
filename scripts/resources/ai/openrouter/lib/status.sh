#!/bin/bash
# OpenRouter status functionality

# Get script directory
OPENROUTER_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${OPENROUTER_STATUS_DIR}/core.sh"

# Main status function
openrouter::status() {
    local verbose="${1:-false}"
    local format="${2:-text}"
    
    # Source format utilities
    source "${OPENROUTER_STATUS_DIR}/../../../../lib/utils/format.sh"
    
    # Initialize
    openrouter::init >/dev/null 2>&1
    
    local status="unknown"
    local message=""
    local api_base="$OPENROUTER_API_BASE"
    local model="$OPENROUTER_DEFAULT_MODEL"
    
    # Check if API key is configured
    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        status="not_configured"
        message="OpenRouter API key not configured"
    else
        # Check if it's a placeholder key
        if [[ "$OPENROUTER_API_KEY" == "sk-placeholder-key" ]]; then
            # For API services with placeholder keys, use "configured" status
            # This indicates it's installed but needs real credentials
            status="configured"
            message="OpenRouter configured with placeholder key (requires real API key)"
        else
            # Test connection with real API key
            if openrouter::test_connection; then
                status="running"
                message="OpenRouter API is accessible"
            else
                status="error"
                message="OpenRouter API is not accessible (check API key validity)"
            fi
        fi
    fi
    
    # Determine running status
    # API services are considered "running" if they're configured or operational
    local running="false"
    if [[ "$status" == "running" || "$status" == "configured" ]]; then
        running="true"
    fi
    
    # Build output data for format utility
    local -a output_data=(
        "status" "$status"
        "running" "$running"
        "message" "$message"
        "api_base" "$api_base"
        "default_model" "$model"
    )
    
    # Add verbose details if requested and available
    if [[ "$verbose" == "true" && "$status" == "running" ]]; then
        # Get usage info
        local usage
        usage=$(openrouter::get_usage 2>/dev/null)
        if [[ -n "$usage" ]]; then
            local limit usage_amount
            limit=$(echo "$usage" | jq -r '.data.limit' 2>/dev/null || echo "unknown")
            usage_amount=$(echo "$usage" | jq -r '.data.usage' 2>/dev/null || echo "unknown")
            output_data+=("limit" "$limit" "usage" "$usage_amount")
        fi
    fi
    
    # If JSON format requested, use format utility
    if [[ "$format" == "json" ]]; then
        format::key_value "$format" "${output_data[@]}"
        return
    fi
    
    # Otherwise provide nice visual output
    # Source logging utilities for colored output
    source "${OPENROUTER_STATUS_DIR}/../../../../lib/utils/log.sh"
    
    log::header "🌐 OpenRouter Status"
    echo
    
    log::info "📊 Basic Status:"
    if [[ "$status" == "running" ]]; then
        log::success "   ✅ Status: Running (API accessible)"
    elif [[ "$status" == "configured" ]]; then
        log::warn "   ⚠️  Status: Configured (needs real API key)"
    elif [[ "$status" == "not_configured" ]]; then
        log::error "   ❌ Status: Not configured"
    else
        log::error "   ❌ Status: Error"
    fi
    
    log::info "   📝 Message: $message"
    echo
    
    log::info "🌐 Service Configuration:"
    log::info "   🔗 API Base: $api_base"
    log::info "   🤖 Default Model: $model"
    
    if [[ "$verbose" == "true" && "$status" == "running" ]]; then
        echo
        log::info "💰 Usage Information:"
        if [[ -n "${output_data[limit]:-}" ]]; then
            log::info "   💳 Limit: ${output_data[limit]}"
            log::info "   📊 Usage: ${output_data[usage]}"
        else
            log::info "   ℹ️  Usage data not available"
        fi
    fi
    
    echo
    log::info "🎯 Quick Actions:"
    if [[ "$status" == "configured" ]]; then
        log::info "   🔑 Set real API key: export OPENROUTER_API_KEY='your-key'"
        log::info "   💾 Save to vault: vrooli resource openrouter set-key 'your-key'"
    fi
    log::info "   🧪 Test connection: resource-openrouter test"
    log::info "   📊 Check models: resource-openrouter list-models"
}

# Export function
export -f openrouter::status