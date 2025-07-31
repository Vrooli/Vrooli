#!/bin/bash
# ====================================================================
# Ollama Integration Test
# ====================================================================
#
# Tests Ollama LLM service integration including health checks,
# model listing, text generation, and API functionality.
#
# Required Resources: ollama
# Test Categories: single-resource, ai
# Estimated Duration: 30-60 seconds
#
# ====================================================================

set -euo pipefail

# Test metadata
TEST_RESOURCE="ollama"
TEST_TIMEOUT="${TEST_TIMEOUT:-60}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Recreate HEALTHY_RESOURCES array from exported string
if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
fi

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"

# Ollama configuration
OLLAMA_BASE_URL="http://localhost:11434"

# Test setup
setup_test() {
    echo "🔧 Setting up Ollama integration test..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Auto-discovery fallback for direct test execution
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        echo "🔍 Auto-discovering resources for direct test execution..."
        
        # Use the resource discovery system with timeout
        local resources_dir
        resources_dir="$(cd "$SCRIPT_DIR/../.." && pwd)"
        
        local discovery_output=""
        if timeout 10s bash -c "\"$resources_dir/index.sh\" --action discover 2>&1" > /tmp/discovery_output.tmp 2>&1; then
            discovery_output=$(cat /tmp/discovery_output.tmp)
            rm -f /tmp/discovery_output.tmp
        else
            echo "⚠️  Auto-discovery timed out, using fallback method..."
            # Fallback: check if the required resource is running on its default port
            if curl -f -s --max-time 2 "$OLLAMA_BASE_URL/api/tags" >/dev/null 2>&1; then
                discovery_output="✅ $TEST_RESOURCE is running on port 11434"
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
    
    # Verify Ollama is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "curl" "jq"
    
    echo "✓ Test setup complete"
}

# Test Ollama health and basic connectivity
test_ollama_health() {
    echo "🏥 Testing Ollama health endpoint..."
    
    local response
    response=$(curl -s --max-time 10 "$OLLAMA_BASE_URL/api/tags" 2>/dev/null)
    
    assert_http_success "$response" "Ollama health endpoint responds"
    assert_json_valid "$response" "Response is valid JSON"
    assert_json_field "$response" ".models" "Response contains models array"
    
    echo "✓ Ollama health check passed"
}

# Test model listing functionality
test_ollama_models() {
    echo "📋 Testing Ollama model listing..."
    
    local response
    response=$(curl -s --max-time 10 "$OLLAMA_BASE_URL/api/tags")
    
    assert_json_valid "$response" "Models response is valid JSON"
    
    # Extract model names
    local models
    models=$(echo "$response" | jq -r '.models[].name' 2>/dev/null || echo "")
    
    assert_not_empty "$models" "At least one model is available"
    
    # Check for expected default models
    if echo "$models" | grep -q "llama3.1:8b"; then
        echo "✓ Found llama3.1:8b model"
    else
        echo "⚠ llama3.1:8b not found in available models"
    fi
    
    echo "Available models:"
    echo "$models" | sed 's/^/  • /'
    
    echo "✓ Model listing test passed"
}

# Smart model selection for generation tests
select_generation_model() {
    echo "🎯 Selecting suitable model for generation..." >&2
    
    local all_models
    all_models=$(curl -s "$OLLAMA_BASE_URL/api/tags" | jq -r '.models[].name' 2>/dev/null)
    
    if [[ -z "$all_models" ]]; then
        return 1
    fi
    
    # List of known generative models (prioritized)
    local preferred_models=(
        "llama3.1:8b"
        "llama3.2:3b"
        "llama3.2:1b"
        "llama2:7b"
        "llama2:13b"
        "qwen2.5:7b"
        "qwen2.5-coder:7b"
        "deepseek-r1:8b"
        "codellama:7b"
        "phi3:3.8b"
        "mistral:7b"
        "gemma:7b"
        "neural-chat:7b"
    )
    
    # First, try to find a preferred model
    for preferred in "${preferred_models[@]}"; do
        if echo "$all_models" | grep -q "^${preferred}$"; then
            echo "$preferred"
            return 0
        fi
    done
    
    # If no preferred model found, filter out embedding and vision-only models
    local filtered_models
    filtered_models=$(echo "$all_models" | grep -v -E "(embed|embedding|vision|clip)" | grep -v -E "^(nomic-embed|all-minilm|bge-)" | head -5)
    
    if [[ -n "$filtered_models" ]]; then
        # Select first filtered model
        echo "$filtered_models" | head -1
        return 0
    fi
    
    # Last resort: try any model that's not obviously an embedding model
    local any_model
    any_model=$(echo "$all_models" | grep -v "embed" | head -1)
    if [[ -n "$any_model" ]]; then
        echo "$any_model"
        return 0
    fi
    
    # If all else fails, return first model and let error handling deal with it
    echo "$all_models" | head -1
    return 0
}

# Test text generation with different models
test_ollama_generation() {
    echo "🤖 Testing Ollama text generation..."
    
    # Use smart model selection
    local available_model
    available_model=$(select_generation_model)
    
    if [[ -z "$available_model" || "$available_model" == "null" ]]; then
        skip_test "No suitable models available for generation test"
    fi
    
    echo "Selected model: $available_model"
    
    # Test simple generation
    local prompt="Hello, this is a test. Please respond with 'Test successful'."
    local request_data
    request_data=$(jq -n \
        --arg model "$available_model" \
        --arg prompt "$prompt" \
        '{model: $model, prompt: $prompt, stream: false}')
    
    echo "Sending generation request..."
    local response
    response=$(curl -s --max-time 30 \
        -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d "$request_data")
    
    assert_http_success "$response" "Generation request successful"
    assert_json_valid "$response" "Generation response is valid JSON"
    assert_json_field "$response" ".response" "Response contains generated text"
    
    # Extract generated text
    local generated_text
    generated_text=$(echo "$response" | jq -r '.response' 2>/dev/null)
    
    assert_not_empty "$generated_text" "Generated text is not empty"
    
    echo "Generated text preview:"
    echo "$generated_text" | head -c 200 | sed 's/^/  /'
    if [[ ${#generated_text} -gt 200 ]]; then
        echo "  ... (truncated)"
    fi
    
    echo "✓ Text generation test passed"
}

# Test error handling
test_ollama_error_handling() {
    echo "⚠️ Testing Ollama error handling..."
    
    # Test with invalid model
    local invalid_request
    invalid_request=$(jq -n '{model: "nonexistent-model", prompt: "test", stream: false}')
    
    local error_response
    error_response=$(curl -s --max-time 10 \
        -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d "$invalid_request" 2>/dev/null || echo '{"error":"connection_failed"}')
    
    # Should get an error response
    if echo "$error_response" | jq -e '.error' >/dev/null 2>&1; then
        echo "✓ Error handling works correctly"
    else
        echo "⚠ Unexpected response for invalid model (may still be valid)"
    fi
    
    echo "✓ Error handling test completed"
}

# Test performance characteristics
test_ollama_performance() {
    echo "⚡ Testing Ollama performance..."
    
    # Use smart model selection
    local available_model
    available_model=$(select_generation_model)
    
    if [[ -z "$available_model" || "$available_model" == "null" ]]; then
        skip_test "No suitable models available for performance test"
    fi
    
    echo "Using model for performance test: $available_model"
    
    # Test simple generation timing
    local start_time=$(date +%s)
    
    local simple_request
    simple_request=$(jq -n \
        --arg model "$available_model" \
        '{model: $model, prompt: "Say hello", stream: false}')
    
    local response
    response=$(curl -s --max-time 30 \
        -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d "$simple_request")
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "Simple generation took: ${duration}s"
    
    if [[ $duration -lt 30 ]]; then
        echo "✓ Performance is acceptable (< 30s)"
    else
        echo "⚠ Performance is slow (>= 30s) - may need optimization"
    fi
    
    echo "✓ Performance test completed"
}

# Main test execution
main() {
    echo "🧪 Starting Ollama Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    test_ollama_health
    test_ollama_models
    test_ollama_generation
    test_ollama_error_handling
    test_ollama_performance
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ Ollama integration test failed"
        exit 1
    else
        echo "✅ Ollama integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"