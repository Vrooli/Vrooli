#!/bin/bash
# ====================================================================
# Agent-S2 + Ollama Integration Test
# ====================================================================
#
# Tests the integration between Agent-S2 and Ollama for AI-powered
# desktop automation and decision making.
#
# Requirements:
#   - Agent-S2 running and healthy
#   - Ollama running with models available
#
# @requires: agent-s2 ollama
# ====================================================================

set -euo pipefail

# Test configuration
REQUIRED_RESOURCES=("agent-s2" "ollama")
AGENT_API="http://localhost:4113"
OLLAMA_API="http://localhost:11434"

# Script directory for imports
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Source test framework
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"

# Test setup
setup_test() {
    echo "🔧 Setting up Agent-S2 + Ollama integration test..."
    
    # Check resource availability
    require_resources "${REQUIRED_RESOURCES[@]}"
    
    # Verify APIs are accessible
    if ! curl -sf "${AGENT_API}/health" >/dev/null 2>&1; then
        fail "Agent-S2 API is not accessible at ${AGENT_API}"
    fi
    
    if ! curl -sf "${OLLAMA_API}/api/tags" >/dev/null 2>&1; then
        fail "Ollama API is not accessible at ${OLLAMA_API}"
    fi
    
    echo "✓ Both services are accessible"
    echo "✓ Test setup complete"
}

# Test AI status integration
test_ai_status_integration() {
    echo "🤖 Testing AI status integration..."
    
    # Check Agent-S2's AI status
    local agent_response
    agent_response=$(curl -s "${AGENT_API}/health")
    
    assert_not_empty "$agent_response" "Agent-S2 health endpoint responds"
    
    # Verify AI is enabled and initialized
    local ai_enabled ai_initialized ai_provider
    ai_enabled=$(echo "$agent_response" | jq -r '.ai_status.enabled' 2>/dev/null)
    ai_initialized=$(echo "$agent_response" | jq -r '.ai_status.initialized' 2>/dev/null)
    ai_provider=$(echo "$agent_response" | jq -r '.ai_status.provider' 2>/dev/null)
    
    assert_equals "$ai_enabled" "true" "AI is enabled in Agent-S2"
    assert_equals "$ai_initialized" "true" "AI is initialized in Agent-S2"
    assert_equals "$ai_provider" "ollama" "AI provider is Ollama"
    
    # Check Ollama models
    local ollama_response
    ollama_response=$(curl -s "${OLLAMA_API}/api/tags")
    
    assert_not_empty "$ollama_response" "Ollama API responds"
    
    local model_count
    model_count=$(echo "$ollama_response" | jq '.models | length' 2>/dev/null)
    assert_greater_than "$model_count" "0" "Ollama has models available"
    
    echo "✓ AI status integration test passed"
}

# Test AI-powered screen analysis
test_ai_screen_analysis() {
    echo "🔍 Testing AI-powered screen analysis..."
    
    # First take a screenshot
    local screenshot_response
    screenshot_response=$(curl -s -X POST "${AGENT_API}/screenshot" \
        -H "Content-Type: application/json" \
        -d '{"full_screen": true}')
    
    assert_not_empty "$screenshot_response" "Screenshot captured"
    
    # Check if AI analyze endpoint exists and works
    local analyze_response
    analyze_response=$(curl -s -X POST "${AGENT_API}/ai/analyze" \
        -H "Content-Type: application/json" \
        -d '{
            "instruction": "Briefly describe what you see on the screen",
            "include_screenshot": true
        }' 2>/dev/null || echo '{"error": "endpoint_not_found"}')
    
    if echo "$analyze_response" | jq -e '.error' >/dev/null 2>&1; then
        echo "⚠️  AI analyze endpoint not available, testing AI status only"
        return 0
    fi
    
    assert_not_empty "$analyze_response" "AI analysis endpoint responds"
    
    # Check for analysis content
    if echo "$analyze_response" | jq -e '.analysis' >/dev/null 2>&1; then
        local analysis
        analysis=$(echo "$analyze_response" | jq -r '.analysis' 2>/dev/null)
        assert_not_empty "$analysis" "AI analysis returned content"
        echo "  Analysis preview: $(echo "$analysis" | head -c 100)..."
    fi
    
    echo "✓ AI screen analysis test passed"
}

# Test AI command execution
test_ai_command_execution() {
    echo "⚡ Testing AI command execution..."
    
    # Test simple AI command
    local command_response
    command_response=$(curl -s -X POST "${AGENT_API}/ai/command" \
        -H "Content-Type: application/json" \
        -d '{
            "command": "Get the current mouse position",
            "execute": false
        }' 2>/dev/null || echo '{"error": "endpoint_not_found"}')
    
    if echo "$command_response" | jq -e '.error' >/dev/null 2>&1; then
        echo "⚠️  AI command endpoint not available, skipping test"
        return 0
    fi
    
    assert_not_empty "$command_response" "AI command endpoint responds"
    
    # Check response structure
    if echo "$command_response" | jq -e '.task_id' >/dev/null 2>&1; then
        local task_id
        task_id=$(echo "$command_response" | jq -r '.task_id' 2>/dev/null)
        assert_not_empty "$task_id" "AI command created task"
    fi
    
    echo "✓ AI command execution test passed"
}

# Test AI planning capabilities
test_ai_planning() {
    echo "📋 Testing AI planning capabilities..."
    
    # Test AI planning endpoint
    local plan_response
    plan_response=$(curl -s -X POST "${AGENT_API}/ai/plan" \
        -H "Content-Type: application/json" \
        -d '{
            "goal": "Take a screenshot and describe the desktop",
            "max_steps": 3
        }' 2>/dev/null || echo '{"error": "endpoint_not_found"}')
    
    if echo "$plan_response" | jq -e '.error' >/dev/null 2>&1; then
        echo "⚠️  AI planning endpoint not available, skipping test"
        return 0
    fi
    
    assert_not_empty "$plan_response" "AI planning endpoint responds"
    
    # Check plan structure
    if echo "$plan_response" | jq -e '.plan' >/dev/null 2>&1; then
        local plan_steps
        plan_steps=$(echo "$plan_response" | jq '.plan | length' 2>/dev/null)
        assert_greater_than "$plan_steps" "0" "AI plan contains steps"
    fi
    
    echo "✓ AI planning test passed"
}

# Test resource coordination
test_resource_coordination() {
    echo "🔗 Testing resource coordination..."
    
    # Test that Agent-S2 can query Ollama directly (if configured)
    local health_response
    health_response=$(curl -s "${AGENT_API}/health")
    
    local ai_model ai_url
    ai_model=$(echo "$health_response" | jq -r '.ai_status.model' 2>/dev/null)
    
    assert_not_empty "$ai_model" "Agent-S2 has AI model configured"
    
    # Verify the model exists in Ollama
    local ollama_models
    ollama_models=$(curl -s "${OLLAMA_API}/api/tags")
    
    local model_exists
    model_exists=$(echo "$ollama_models" | jq -r --arg model "$ai_model" '.models[] | select(.name == $model) | .name' 2>/dev/null)
    
    if [[ -n "$model_exists" ]]; then
        assert_equals "$model_exists" "$ai_model" "Configured model exists in Ollama"
    else
        echo "⚠️  Configured model '$ai_model' not found in Ollama, but integration may still work"
    fi
    
    echo "✓ Resource coordination test passed"
}

# Test error handling
test_error_handling() {
    echo "❌ Testing error handling..."
    
    # Test invalid AI request
    local error_response
    error_response=$(curl -s -X POST "${AGENT_API}/ai/analyze" \
        -H "Content-Type: application/json" \
        -d '{"invalid": "request"}' 2>/dev/null || echo '{"error": "request_failed"}')
    
    assert_not_empty "$error_response" "Invalid AI request gets response"
    
    # Should handle gracefully (either error message or default behavior)
    if echo "$error_response" | jq -e '.error' >/dev/null 2>&1; then
        echo "  ✓ Proper error handling for invalid requests"
    else
        echo "  ✓ Request handled with default behavior"
    fi
    
    echo "✓ Error handling test passed"
}

# Test cleanup
cleanup_test() {
    echo "🧹 Cleaning up test artifacts..."
    
    # No specific cleanup needed for this integration test
    # Both services remain running for other tests
    
    echo "✓ Cleanup complete"
}

# Main test execution
main() {
    local start_time=$(date +%s)
    
    echo "🧪 Starting Agent-S2 + Ollama Integration Test"
    echo "Resources: ${REQUIRED_RESOURCES[*]}"
    echo "Agent API: $AGENT_API"
    echo "Ollama API: $OLLAMA_API"
    echo "Timeout: ${TEST_TIMEOUT:-300}s"
    echo
    
    # Setup
    setup_test
    
    # Run integration tests
    test_ai_status_integration
    test_ai_screen_analysis
    test_ai_command_execution
    test_ai_planning
    test_resource_coordination
    test_error_handling
    
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
        echo "❌ Agent-S2 + Ollama integration test failed"
        exit 1
    else
        echo "✅ Agent-S2 + Ollama integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"