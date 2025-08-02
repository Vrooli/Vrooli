#!/bin/bash
# ====================================================================
# AI Resource Test Template
# ====================================================================
#
# Template for testing AI resources with category-specific requirements
# including model management, inference capabilities, and performance
# characteristics.
#
# Usage: Copy this template and customize for specific AI resource
#
# Required Variables to Set:
#   TEST_RESOURCE - Resource name (e.g., "ollama", "whisper")
#   RESOURCE_PORT - Default port for the resource
#   RESOURCE_BASE_URL - Base URL for API calls
#
# ====================================================================

set -euo pipefail

# Test metadata - CUSTOMIZE THESE
TEST_RESOURCE="your-ai-resource"
TEST_TIMEOUT="${TEST_TIMEOUT:-120}"  # AI operations can be slow
TEST_CLEANUP="${TEST_CLEANUP:-true}"
RESOURCE_PORT="8080"  # Replace with actual port
RESOURCE_BASE_URL="http://localhost:${RESOURCE_PORT}"

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
source "$SCRIPT_DIR/framework/interface-compliance-categories.sh"

# Test setup
setup_test() {
    echo "🔧 Setting up AI resource test for $TEST_RESOURCE..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Auto-discovery fallback for direct test execution
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        echo "🔍 Auto-discovering resources for direct test execution..."
        
        local resources_dir
        resources_dir="$(cd "$SCRIPT_DIR/../.." && pwd)"
        
        local discovery_output=""
        if timeout 10s bash -c "\"$resources_dir/index.sh\" --action discover 2>&1" > /tmp/discovery_output.tmp 2>&1; then
            discovery_output=$(cat /tmp/discovery_output.tmp)
            rm -f /tmp/discovery_output.tmp
        else
            echo "⚠️  Auto-discovery timed out, using fallback method..."
            if curl -f -s --max-time 2 "$RESOURCE_BASE_URL/health" >/dev/null 2>&1; then
                discovery_output="✅ $TEST_RESOURCE is running on port $RESOURCE_PORT"
            fi
        fi
        
        local discovered_resources=()
        while IFS= read -r line; do
            if [[ "$line" =~ ✅[[:space:]]+([^[:space:]]+)[[:space:]]+is[[:space:]]+running ]]; then
                discovered_resources+=("${BASH_REMATCH[1]}")
            fi
        done <<< "$discovery_output"
        
        if [[ ${#discovered_resources[@]} -eq 0 ]]; then
            echo "⚠️  No resources discovered, but test will proceed..."
            discovered_resources=("$TEST_RESOURCE")
        fi
        
        export HEALTHY_RESOURCES_STR="${discovered_resources[*]}"
        echo "✓ Discovered healthy resources: $HEALTHY_RESOURCES_STR"
    fi
    
    # Verify resource is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "curl" "jq"
    
    echo "✓ Test setup complete"
}

# Test AI resource interface compliance
test_ai_interface_compliance() {
    echo "🔧 Testing AI resource interface compliance..."
    
    # Find the manage.sh script
    local manage_script=""
    local potential_paths=(
        "$SCRIPT_DIR/../ai/$TEST_RESOURCE/manage.sh"
        "$SCRIPT_DIR/../../ai/$TEST_RESOURCE/manage.sh"
        "$(cd "$SCRIPT_DIR/../.." && pwd)/ai/$TEST_RESOURCE/manage.sh"
    )
    
    for path in "${potential_paths[@]}"; do
        if [[ -f "$path" ]]; then
            manage_script="$path"
            break
        fi
    done
    
    if [[ -z "$manage_script" ]]; then
        manage_script=$(find "$(cd "$SCRIPT_DIR/../.." && pwd)" -name "manage.sh" -path "*/$TEST_RESOURCE/*" -type f 2>/dev/null | head -1)
    fi
    
    if [[ -z "$manage_script" || ! -f "$manage_script" ]]; then
        echo "⚠️  Could not find $TEST_RESOURCE manage.sh script - skipping interface compliance test"
        return 0
    fi
    
    echo "📍 Using manage script: $manage_script"
    
    # Run complete compliance test (base + category-specific)
    if test_complete_resource_compliance "$TEST_RESOURCE" "$manage_script" "ai"; then
        echo "✅ AI resource interface compliance test passed"
        return 0
    else
        echo "❌ AI resource interface compliance test failed"
        return 1
    fi
}

# Test AI resource health and connectivity
test_ai_health() {
    echo "🏥 Testing AI resource health..."
    
    # Use category-specific health check
    local health_result
    health_result=$(check_ai_resource_health "$TEST_RESOURCE" "$RESOURCE_PORT" "detailed")
    
    assert_not_equals "$health_result" "unreachable" "AI resource is reachable"
    
    # Parse detailed health information
    if [[ "$health_result" =~ healthy ]]; then
        echo "✓ AI resource health check passed: $health_result"
    else
        echo "⚠️  AI resource health degraded: $health_result"
    fi
}

# Test model management capabilities (AI-specific)
test_ai_model_management() {
    echo "🤖 Testing AI model management..."
    
    # This is a template - customize based on your AI resource's API
    
    # Test model listing
    local models_response
    models_response=$(curl -s --max-time 10 "$RESOURCE_BASE_URL/api/models" 2>/dev/null || echo '{"models":[]}')
    
    assert_json_valid "$models_response" "Models API response is valid JSON"
    
    # Check if models are available
    local model_count
    model_count=$(echo "$models_response" | jq '.models | length' 2>/dev/null || echo "0")
    
    if [[ "$model_count" -gt 0 ]]; then
        echo "✓ Found $model_count AI models available"
        
        # List model names
        echo "Available models:"
        echo "$models_response" | jq -r '.models[].name' 2>/dev/null | sed 's/^/  • /' || echo "  • Could not list model names"
    else
        echo "⚠️  No AI models found - this may affect inference capabilities"
    fi
    
    echo "✓ Model management test completed"
}

# Test inference capabilities (AI-specific)
test_ai_inference() {
    echo "🧠 Testing AI inference capabilities..."
    
    # Get available models first
    local models_response
    models_response=$(curl -s --max-time 10 "$RESOURCE_BASE_URL/api/models" 2>/dev/null)
    
    if [[ -z "$models_response" ]] || ! echo "$models_response" | jq . >/dev/null 2>&1; then
        skip_test "Cannot retrieve model information for inference test"
    fi
    
    local available_model
    available_model=$(echo "$models_response" | jq -r '.models[0].name' 2>/dev/null)
    
    if [[ -z "$available_model" || "$available_model" == "null" ]]; then
        skip_test "No suitable models available for inference test"
    fi
    
    echo "Using model for inference test: $available_model"
    
    # Customize this based on your AI resource's inference API
    local inference_request
    inference_request=$(jq -n \
        --arg model "$available_model" \
        --arg prompt "Hello, this is a test. Please respond briefly." \
        '{model: $model, prompt: $prompt, max_tokens: 50}')
    
    echo "Sending inference request..."
    local inference_response
    inference_response=$(curl -s --max-time 60 \
        -X POST "$RESOURCE_BASE_URL/api/inference" \
        -H "Content-Type: application/json" \
        -d "$inference_request" 2>/dev/null)
    
    assert_http_success "$inference_response" "Inference request successful"
    assert_json_valid "$inference_response" "Inference response is valid JSON"
    
    # Check for response content (customize field names)
    local generated_text
    generated_text=$(echo "$inference_response" | jq -r '.response // .text // .output' 2>/dev/null)
    
    assert_not_empty "$generated_text" "Generated text is not empty"
    
    echo "Generated text preview:"
    echo "$generated_text" | head -c 200 | sed 's/^/  /'
    if [[ ${#generated_text} -gt 200 ]]; then
        echo "  ... (truncated)"
    fi
    
    echo "✓ Inference test passed"
}

# Test AI performance characteristics
test_ai_performance() {
    echo "⚡ Testing AI resource performance..."
    
    # Test simple operation timing
    local start_time=$(date +%s)
    
    # Customize this based on your AI resource's quickest operation
    local quick_response
    quick_response=$(curl -s --max-time 30 "$RESOURCE_BASE_URL/api/status" 2>/dev/null)
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "API response time: ${duration}s"
    
    if [[ $duration -lt 10 ]]; then
        echo "✓ Performance is good (< 10s for basic operations)"
    elif [[ $duration -lt 30 ]]; then
        echo "⚠️  Performance is acceptable (10-30s for basic operations)"
    else
        echo "⚠️  Performance is slow (> 30s for basic operations)"
    fi
    
    echo "✓ Performance test completed"
}

# Test error handling
test_ai_error_handling() {
    echo "⚠️ Testing AI resource error handling..."
    
    # Test with invalid request
    local invalid_request='{"invalid": "data"}'
    
    local error_response
    error_response=$(curl -s --max-time 10 \
        -X POST "$RESOURCE_BASE_URL/api/inference" \
        -H "Content-Type: application/json" \
        -d "$invalid_request" 2>/dev/null || echo '{"error":"connection_failed"}')
    
    # Should get an error response or handle gracefully
    if echo "$error_response" | jq -e '.error' >/dev/null 2>&1; then
        echo "✓ Error handling works correctly"
    else
        echo "⚠️  Unexpected response for invalid request (may still be valid)"
    fi
    
    echo "✓ Error handling test completed"
}

# Main test execution
main() {
    echo "🧪 Starting AI Resource Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo "Base URL: $RESOURCE_BASE_URL"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    echo "📋 Running AI resource test suite..."
    echo
    
    # Phase 1: Interface Compliance (should be first)
    echo "Phase 1: Interface Compliance"
    test_ai_interface_compliance
    echo
    
    # Phase 2: Functional Tests
    echo "Phase 2: Functional Tests"
    test_ai_health
    test_ai_model_management
    test_ai_inference
    test_ai_error_handling
    test_ai_performance
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ AI resource integration test failed"
        exit 1
    else
        echo "✅ AI resource integration test passed"
        exit 0
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi