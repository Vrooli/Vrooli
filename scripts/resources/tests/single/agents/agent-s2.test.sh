#!/bin/bash
# ====================================================================
# Agent-S2 Integration Test
# ====================================================================
#
# Tests Agent-S2 autonomous computer interaction service functionality
# including health checks, capabilities, GUI automation, and AI integration.
#
# Requirements:
#   - Agent-S2 running and healthy
#   - API accessible on port 4113
#
# ====================================================================

set -euo pipefail

# Test configuration
TEST_RESOURCE="agent-s2"
REQUIRED_RESOURCES=("agent-s2")
API_BASE="http://localhost:4113"

# Script directory for imports
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Source test framework
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"

# Test setup
setup_test() {
    echo "🔧 Setting up Agent-S2 integration test..."
    
    # Check resource availability
    require_resource "$TEST_RESOURCE"
    
    # Verify API is accessible
    if ! curl -sf "${API_BASE}/health" >/dev/null 2>&1; then
        fail "Agent-S2 API is not accessible at ${API_BASE}"
    fi
    
    echo "✓ Test setup complete"
}

# Test health endpoint
test_agent_s2_health() {
    echo "🏥 Testing Agent-S2 health endpoint..."
    
    local response
    response=$(curl -s "${API_BASE}/health")
    
    assert_not_empty "$response" "Health endpoint responds"
    
    # Check health status
    local status
    status=$(echo "$response" | jq -r '.status' 2>/dev/null)
    assert_equals "$status" "healthy" "Service status is healthy"
    
    # Check display configuration
    local display
    display=$(echo "$response" | jq -r '.display' 2>/dev/null)
    assert_not_empty "$display" "Display is configured"
    
    # Check screen size
    local width height
    width=$(echo "$response" | jq -r '.screen_size.width' 2>/dev/null)
    height=$(echo "$response" | jq -r '.screen_size.height' 2>/dev/null)
    assert_not_empty "$width" "Screen width is set"
    assert_not_empty "$height" "Screen height is set"
    
    # Check AI status
    local ai_enabled
    ai_enabled=$(echo "$response" | jq -r '.ai_status.enabled' 2>/dev/null)
    assert_equals "$ai_enabled" "true" "AI is enabled"
    
    # Check mode info
    local current_mode
    current_mode=$(echo "$response" | jq -r '.mode_info.current_mode' 2>/dev/null)
    assert_equals "$current_mode" "sandbox" "Running in sandbox mode"
    
    echo "✓ Health check passed"
}

# Test capabilities endpoint
test_agent_s2_capabilities() {
    echo "🎯 Testing Agent-S2 capabilities..."
    
    local response
    response=$(curl -s "${API_BASE}/capabilities")
    
    assert_not_empty "$response" "Capabilities endpoint responds"
    
    # Check core capabilities
    local capabilities
    capabilities=$(echo "$response" | jq -r '.capabilities' 2>/dev/null)
    assert_not_empty "$capabilities" "Capabilities object exists"
    
    # Verify essential capabilities
    assert_json_true "$response" ".capabilities.screenshot" "Screenshot capability"
    assert_json_true "$response" ".capabilities.gui_automation" "GUI automation capability"
    assert_json_true "$response" ".capabilities.mouse_control" "Mouse control capability"
    assert_json_true "$response" ".capabilities.keyboard_control" "Keyboard control capability"
    assert_json_true "$response" ".capabilities.multi_step_tasks" "Multi-step task capability"
    
    # Check supported tasks
    local supported_tasks
    supported_tasks=$(echo "$response" | jq -r '.supported_tasks[]' 2>/dev/null | wc -l)
    assert_greater_than "$supported_tasks" "0" "Has supported tasks"
    
    # Check mode info
    local apps_available
    apps_available=$(echo "$response" | jq -r '.mode_info.applications_available' 2>/dev/null)
    assert_greater_than "$apps_available" "0" "Applications available in sandbox"
    
    echo "✓ Capabilities test passed"
}

# Test screenshot functionality
test_agent_s2_screenshot() {
    echo "📸 Testing screenshot capture..."
    
    local response
    response=$(curl -s -X POST "${API_BASE}/screenshot" \
        -H "Content-Type: application/json" \
        -d '{"full_screen": true}')
    
    assert_not_empty "$response" "Screenshot endpoint responds"
    
    # Check for image data
    local image_data
    image_data=$(echo "$response" | jq -r '.image_base64' 2>/dev/null)
    assert_not_empty "$image_data" "Screenshot data returned"
    
    # Verify it's base64
    if ! echo "$image_data" | base64 -d >/dev/null 2>&1; then
        fail "Screenshot data is not valid base64"
    fi
    
    # Check dimensions
    local width height
    width=$(echo "$response" | jq -r '.width' 2>/dev/null)
    height=$(echo "$response" | jq -r '.height' 2>/dev/null)
    assert_greater_than "$width" "0" "Screenshot width is positive"
    assert_greater_than "$height" "0" "Screenshot height is positive"
    
    echo "✓ Screenshot test passed"
}

# Test mouse position
test_agent_s2_mouse_position() {
    echo "🖱️ Testing mouse position..."
    
    local response
    response=$(curl -s "${API_BASE}/mouse/position")
    
    assert_not_empty "$response" "Mouse position endpoint responds"
    
    # Check coordinates
    local x y
    x=$(echo "$response" | jq -r '.x' 2>/dev/null)
    y=$(echo "$response" | jq -r '.y' 2>/dev/null)
    assert_not_empty "$x" "X coordinate exists"
    assert_not_empty "$y" "Y coordinate exists"
    assert_greater_than_or_equal "$x" "0" "X coordinate is non-negative"
    assert_greater_than_or_equal "$y" "0" "Y coordinate is non-negative"
    
    echo "✓ Mouse position test passed"
}

# Test mode information (using health endpoint)
test_agent_s2_mode_info() {
    echo "🔒 Testing mode configuration from health endpoint..."
    
    local response
    response=$(curl -s "${API_BASE}/health")
    
    assert_not_empty "$response" "Health endpoint responds"
    
    # Check current mode from health response
    local mode
    mode=$(echo "$response" | jq -r '.mode_info.current_mode' 2>/dev/null)
    assert_equals "$mode" "sandbox" "Current mode is sandbox"
    
    # Check security level
    local security_level
    security_level=$(echo "$response" | jq -r '.mode_info.security_level' 2>/dev/null)
    assert_equals "$security_level" "high" "Security level is high"
    
    # Check host mode status
    local host_enabled
    host_enabled=$(echo "$response" | jq -r '.mode_info.host_mode_enabled' 2>/dev/null)
    assert_equals "$host_enabled" "false" "Host mode is disabled by default"
    
    echo "✓ Mode configuration test passed"
}

# Test error handling
test_agent_s2_error_handling() {
    echo "❌ Testing error handling..."
    
    # Test invalid endpoint
    local response status_code
    response=$(curl -s -o /dev/null -w "%{http_code}" "${API_BASE}/invalid-endpoint")
    assert_equals "$response" "404" "Returns 404 for invalid endpoint"
    
    # Test invalid screenshot request
    response=$(curl -s -X POST "${API_BASE}/screenshot" \
        -H "Content-Type: application/json" \
        -d '{"invalid_param": true}')
    assert_not_empty "$response" "Invalid request gets response"
    
    # Test invalid mouse move
    response=$(curl -s -X POST "${API_BASE}/mouse/move" \
        -H "Content-Type: application/json" \
        -d '{"x": -100, "y": -100}')
    # Should handle gracefully (clamp to screen bounds or error)
    assert_not_empty "$response" "Invalid mouse move gets response"
    
    echo "✓ Error handling test passed"
}

# Test AI integration if available
test_agent_s2_ai_integration() {
    echo "🤖 Testing AI integration..."
    
    # First check if AI is initialized
    local health_response
    health_response=$(curl -s "${API_BASE}/health")
    local ai_initialized
    ai_initialized=$(echo "$health_response" | jq -r '.ai_status.initialized' 2>/dev/null)
    
    if [[ "$ai_initialized" != "true" ]]; then
        echo "⚠️  AI not initialized, skipping AI tests"
        return 0
    fi
    
    # Test AI status endpoint
    local response
    response=$(curl -s "${API_BASE}/ai/status")
    assert_not_empty "$response" "AI status endpoint responds"
    
    # Check AI configuration
    local provider model
    provider=$(echo "$response" | jq -r '.provider' 2>/dev/null)
    model=$(echo "$response" | jq -r '.model' 2>/dev/null)
    assert_not_empty "$provider" "AI provider is configured"
    assert_not_empty "$model" "AI model is configured"
    
    echo "✓ AI integration test passed"
}

# Test resource cleanup
cleanup_test() {
    echo "🧹 Cleaning up test artifacts..."
    
    # No specific cleanup needed for agent-s2 tests
    # The service remains running for other tests
    
    echo "✓ Cleanup complete"
}

# Main test execution
main() {
    local start_time=$(date +%s)
    
    echo "🧪 Starting Agent-S2 Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "API: $API_BASE"
    echo "Timeout: ${TEST_TIMEOUT:-300}s"
    echo
    
    # Setup
    setup_test
    
    # Run tests
    test_agent_s2_health
    test_agent_s2_capabilities
    test_agent_s2_screenshot
    test_agent_s2_mouse_position
    test_agent_s2_mode_info
    test_agent_s2_error_handling
    test_agent_s2_ai_integration
    
    # Cleanup
    cleanup_test
    
    # Summary
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo
    print_assertion_summary
    echo "⏱️  Test duration: ${duration}s"
    
    # Exit based on assertion results
    if [[ ${ASSERTION_FAILURES:-0} -gt 0 ]]; then
        echo "❌ Agent-S2 integration test failed"
        exit 1
    else
        echo "✅ Agent-S2 integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"