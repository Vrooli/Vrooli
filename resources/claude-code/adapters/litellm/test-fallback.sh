#!/usr/bin/env bash
# Test automatic fallback on rate limit
# Simulates a rate limit error and verifies fallback behavior

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../../.." && builtin pwd)}"
ADAPTER_DIR="${APP_ROOT}/resources/claude-code/adapters/litellm"

# Source required libraries
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/../../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/state.sh"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/execute.sh"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/../../lib/common.sh"

#######################################
# Test automatic fallback trigger
#######################################
test_auto_fallback() {
    log::header "🧪 Testing Automatic Fallback on Rate Limit"
    echo
    
    # Reset state
    log::info "Resetting LiteLLM connection state..."
    litellm::set_connected "false" "test_reset"
    
    # Ensure auto-fallback is enabled
    log::info "Enabling auto-fallback..."
    litellm::set_config "auto_fallback_enabled" "true"
    litellm::set_config "auto_fallback_on_rate_limit" "true"
    
    # Create a mock rate limit info
    local rate_info='{
        "detected": true,
        "retry_after": "18000",
        "reset_time": "2025-08-21T23:00:00Z",
        "limit_type": "5_hour",
        "error_message": "Claude AI usage limit reached",
        "timestamp": "2025-08-21T21:00:00Z"
    }'
    
    log::info "Simulating rate limit detection..."
    echo "Rate info:"
    echo "$rate_info" | jq '.'
    echo
    
    # Test 1: Verify auto-fallback triggers
    log::info "Test 1: Auto-fallback trigger"
    if litellm::auto_fallback_on_rate_limit "$rate_info"; then
        if litellm::is_connected; then
            log::success "  ✅ Auto-fallback triggered and connected"
        else
            log::error "  ❌ Auto-fallback triggered but not connected"
            return 1
        fi
    else
        # Check if it's because LiteLLM isn't available
        if ! litellm::check_resource_available; then
            log::warn "  ⚠️  Auto-fallback not triggered (LiteLLM not available - expected)"
        else
            log::error "  ❌ Auto-fallback failed to trigger"
            return 1
        fi
    fi
    
    # Test 2: Verify auto-disconnect time is set
    log::info "Test 2: Auto-disconnect scheduling"
    local state_json
    state_json=$(litellm::get_state)
    local auto_disconnect_at
    auto_disconnect_at=$(echo "$state_json" | jq -r '.auto_disconnect_at // null')
    
    if [[ "$auto_disconnect_at" != "null" && -n "$auto_disconnect_at" ]]; then
        log::success "  ✅ Auto-disconnect scheduled: $auto_disconnect_at"
    else
        log::error "  ❌ Auto-disconnect not scheduled"
    fi
    
    # Test 3: Verify fallback reason is recorded
    log::info "Test 3: Fallback reason recording"
    local fallback_reason
    fallback_reason=$(echo "$state_json" | jq -r '.fallback_reason // null')
    
    if [[ "$fallback_reason" == "rate_limit" ]]; then
        log::success "  ✅ Fallback reason correctly recorded: $fallback_reason"
    else
        log::error "  ❌ Fallback reason not recorded correctly: $fallback_reason"
    fi
    
    # Test 4: Test that auto-fallback doesn't trigger if already connected
    log::info "Test 4: No double-connection"
    local was_connected
    was_connected=$(litellm::is_connected && echo "true" || echo "false")
    
    if [[ "$was_connected" == "true" ]]; then
        # Try to trigger again
        if litellm::auto_fallback_on_rate_limit "$rate_info"; then
            log::success "  ✅ Returns success when already connected (no double-connect)"
        else
            log::error "  ❌ Failed when already connected"
        fi
    else
        log::info "  ⚠️  Skipping (not connected)"
    fi
    
    # Test 5: Test auto-fallback respects configuration
    log::info "Test 5: Configuration respect"
    # Disconnect first
    litellm::set_connected "false" "test"
    # Disable auto-fallback
    litellm::set_config "auto_fallback_enabled" "false"
    
    if ! litellm::auto_fallback_on_rate_limit "$rate_info"; then
        log::success "  ✅ Auto-fallback respects disabled setting"
    else
        log::error "  ❌ Auto-fallback triggered when disabled"
    fi
    
    # Clean up
    log::info ""
    log::info "Cleaning up..."
    litellm::set_connected "false" "test_cleanup"
    litellm::set_config "auto_fallback_enabled" "true"
    
    echo
    log::success "✅ Auto-fallback tests completed"
}

# Test rate limit detection and recording
test_rate_limit_handling() {
    log::header "🧪 Testing Rate Limit Detection and Recording"
    echo
    
    # Test with the actual JSON error from the user
    local test_json='{"type":"result","subtype":"success","is_error":true,"duration_ms":393,"duration_api_ms":0,"num_turns":1,"result":"Claude AI usage limit reached|1755806400","session_id":"71b47827-9527-47f3-8614-34249f877747","total_cost_usd":0,"usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0,"server_tool_use":{"web_search_requests":0}},"service_tier":"standard"}'
    
    log::info "Testing with actual rate limit JSON..."
    local rate_info
    rate_info=$(claude_code::detect_rate_limit "$test_json" "1")
    
    local detected
    detected=$(echo "$rate_info" | jq -r '.detected')
    
    if [[ "$detected" == "true" ]]; then
        log::success "✅ Rate limit detected from JSON"
        echo "$rate_info" | jq '.'
        
        # Record it
        claude_code::record_rate_limit "$rate_info"
        
        # Verify it was recorded
        local usage_json
        usage_json=$(claude_code::get_usage)
        local last_rate_limit
        last_rate_limit=$(echo "$usage_json" | jq -r '.last_rate_limit // null')
        
        if [[ "$last_rate_limit" != "null" ]]; then
            log::success "✅ Rate limit was recorded"
        else
            log::error "❌ Rate limit was not recorded"
        fi
    else
        log::error "❌ Rate limit not detected from JSON"
    fi
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Run tests
    test_rate_limit_handling
    echo
    test_auto_fallback
fi