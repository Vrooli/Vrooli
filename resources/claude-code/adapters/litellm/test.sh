#!/usr/bin/env bash
# LiteLLM Adapter Test Script
# Tests the basic functionality of the LiteLLM adapter

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
ADAPTER_DIR="${APP_ROOT}/resources/claude-code/adapters/litellm"

# Source required libraries
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/state.sh"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/status.sh"

#######################################
# Run adapter tests
#######################################
run_tests() {
    log::header "🧪 Testing LiteLLM Adapter"
    echo
    
    local tests_passed=0
    local tests_failed=0
    
    # Test 1: State initialization
    log::info "Test 1: State initialization"
    if litellm::init_state; then
        log::success "  ✅ State initialization successful"
        tests_passed=$((tests_passed + 1))
    else
        log::error "  ❌ State initialization failed"
        tests_failed=$((tests_failed + 1))
    fi
    
    # Test 2: Configuration get/set
    log::info "Test 2: Configuration get/set"
    local original_value
    original_value=$(litellm::get_config "auto_fallback_enabled")
    litellm::set_config "auto_fallback_enabled" "false"
    local new_value
    new_value=$(litellm::get_config "auto_fallback_enabled")
    if [[ "$new_value" == "false" ]]; then
        log::success "  ✅ Configuration set/get working"
        tests_passed=$((tests_passed + 1))
        # Restore original value
        litellm::set_config "auto_fallback_enabled" "$original_value"
    else
        log::error "  ❌ Configuration set/get failed"
        tests_failed=$((tests_failed + 1))
    fi
    
    # Test 3: Connection state management
    log::info "Test 3: Connection state management"
    litellm::set_connected "false" "test"
    if ! litellm::is_connected; then
        litellm::set_connected "true" "test"
        if litellm::is_connected; then
            log::success "  ✅ Connection state management working"
            tests_passed=$((tests_passed + 1))
            # Clean up
            litellm::set_connected "false" "test"
        else
            log::error "  ❌ Failed to set connected state"
            tests_failed=$((tests_failed + 1))
        fi
    else
        log::error "  ❌ Failed to set disconnected state"
        tests_failed=$((tests_failed + 1))
    fi
    
    # Test 4: Request counting
    log::info "Test 4: Request counting"
    local initial_state
    initial_state=$(litellm::get_state)
    local initial_count
    initial_count=$(echo "$initial_state" | jq -r '.requests_handled')
    litellm::increment_request_count "success"
    local new_state
    new_state=$(litellm::get_state)
    local new_count
    new_count=$(echo "$new_state" | jq -r '.requests_handled')
    if [[ $new_count -gt $initial_count ]]; then
        log::success "  ✅ Request counting working"
        tests_passed=$((tests_passed + 1))
    else
        log::error "  ❌ Request counting failed"
        tests_failed=$((tests_failed + 1))
    fi
    
    # Test 5: Error recording
    log::info "Test 5: Error recording"
    litellm::record_error "Test error message"
    local state_with_error
    state_with_error=$(litellm::get_state)
    local last_error
    last_error=$(echo "$state_with_error" | jq -r '.last_error')
    if [[ "$last_error" == "Test error message" ]]; then
        log::success "  ✅ Error recording working"
        tests_passed=$((tests_passed + 1))
    else
        log::error "  ❌ Error recording failed"
        tests_failed=$((tests_failed + 1))
    fi
    
    # Test 6: Model mapping
    log::info "Test 6: Model mapping"
    local mapped_model
    mapped_model=$(litellm::get_model_mapping "claude-3-5-sonnet")
    if [[ "$mapped_model" == "qwen2.5-coder" ]]; then
        log::success "  ✅ Model mapping working"
        tests_passed=$((tests_passed + 1))
    else
        log::error "  ❌ Model mapping failed (got: $mapped_model)"
        tests_failed=$((tests_failed + 1))
    fi
    
    # Test 7: Auto-disconnect check
    log::info "Test 7: Auto-disconnect check"
    # Set a connection with auto-disconnect in the past
    litellm::set_connected "true" "test"
    # Manually set auto-disconnect to past time
    local temp_file
    temp_file=$(mktemp)
    local past_time
    past_time=$(date -u -d "-1 hour" +%Y-%m-%dT%H:%M:%SZ)
    jq --arg time "$past_time" '.auto_disconnect_at = $time' "$LITELLM_STATE_FILE" > "$temp_file" && mv "$temp_file" "$LITELLM_STATE_FILE"
    
    if litellm::should_auto_disconnect; then
        log::success "  ✅ Auto-disconnect detection working"
        tests_passed=$((tests_passed + 1))
    else
        log::error "  ❌ Auto-disconnect detection failed"
        tests_failed=$((tests_failed + 1))
    fi
    # Clean up
    litellm::set_connected "false" "test"
    
    # Test 8: LiteLLM resource check
    log::info "Test 8: LiteLLM resource availability check"
    if litellm::check_resource_available; then
        log::success "  ✅ LiteLLM resource is available"
        tests_passed=$((tests_passed + 1))
    else
        log::warn "  ⚠️  LiteLLM resource not available (expected if not installed)"
        # Don't count as failure since it's expected
    fi
    
    # Summary
    echo
    log::header "📊 Test Summary"
    log::info "Tests Passed: $tests_passed"
    log::info "Tests Failed: $tests_failed"
    
    if [[ $tests_failed -eq 0 ]]; then
        log::success "✅ All tests passed!"
        return 0
    else
        log::error "❌ Some tests failed"
        return 1
    fi
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_tests
fi