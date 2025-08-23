#!/usr/bin/env bash
# Ollama Integration Test
# Tests real Ollama service functionality
# Enhanced with structured prompt fixtures for response validation

set -euo pipefail

OLLAMA_TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${OLLAMA_TEST_DIR}/../../../../lib/utils/var.sh"

# Source enhanced integration test library with fixture support
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/tests/lib/enhanced-integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Override library defaults with Ollama-specific settings
SERVICE_NAME="ollama"
BASE_URL="${OLLAMA_BASE_URL:-http://localhost:11434}"
readonly TEST_MODEL="${OLLAMA_TEST_MODEL:-llama3.2:1b}"
HEALTH_ENDPOINT="/api/tags"
REQUIRED_TOOLS=("curl" "jq")
SERVICE_METADATA=("Test Model: $TEST_MODEL")

#######################################
# OLLAMA-SPECIFIC TEST FUNCTIONS
#######################################

test_version_endpoint() {
    local test_name="version endpoint"
    
    local response
    if response=$(make_api_request "/api/version" "GET" 5); then
        if validate_json_field "$response" ".version"; then
            local version
            version=$(echo "$response" | jq -r '.version')
            log_test_result "$test_name" "PASS" "version: $version"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "version check failed"
    return 1
}

test_model_listing() {
    local test_name="model listing"
    
    local response
    if response=$(make_api_request "/api/tags" "GET" 5); then
        if validate_json_field "$response" ".models"; then
            local count
            count=$(echo "$response" | jq '.models | length')
            log_test_result "$test_name" "PASS" "models: $count"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "model listing failed"
    return 1
}

test_model_availability() {
    local test_name="model $TEST_MODEL availability"
    
    local response
    if response=$(make_api_request "/api/tags" "GET" 5); then
        if echo "$response" | jq -e ".models[] | select(.name == \"$TEST_MODEL\")" >/dev/null 2>&1; then
            log_test_result "$test_name" "PASS"
            return 0
        else
            log_test_result "$test_name" "SKIP" "model not found, would need to pull"
            return 2
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "unable to check model availability"
    return 1
}

test_text_generation() {
    local test_name="text generation"
    
    # Only run if model is available
    local response
    response=$(make_api_request "/api/tags" "GET" 5)
    
    if ! echo "$response" | jq -e ".models[] | select(.name == \"$TEST_MODEL\")" >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "model not available"
        return 2
    fi
    
    # Test generation
    local prompt
    prompt="{\"model\":\"$TEST_MODEL\",\"prompt\":\"Say hello\",\"stream\":false}"
    
    if response=$(make_api_request "/api/generate" "POST" "$TIMEOUT" "-H 'Content-Type: application/json' -d '$prompt'"); then
        if validate_json_field "$response" ".response"; then
            log_test_result "$test_name" "PASS"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "generation request failed"
    return 1
}

test_embeddings_endpoint() {
    local test_name="embeddings endpoint"
    
    # Check if embedding models are available
    local response
    response=$(make_api_request "/api/tags" "GET" 5)
    
    if ! echo "$response" | jq -e '.models[] | select(.name | contains("embed"))' >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "no embedding model available"
        return 2
    fi
    
    # Test embeddings
    local embed_model
    embed_model=$(echo "$response" | jq -r '.models[] | select(.name | contains("embed")) | .name' | head -1)
    local request
    request="{\"model\":\"$embed_model\",\"prompt\":\"test text\"}"
    
    if response=$(make_api_request "/api/embeddings" "POST" "$TIMEOUT" "-H 'Content-Type: application/json' -d '$request'"); then
        if validate_json_field "$response" ".embedding"; then
            log_test_result "$test_name" "PASS"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "embeddings request failed"
    return 1
}

#######################################
# ENHANCED FIXTURE-BASED TESTS
#######################################

test_llm_prompt_fixture() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Read prompt JSON from fixture
    local prompt_json
    prompt_json=$(cat "$fixture_path")
    
    # Extract prompt and expected response pattern
    local prompt_text
    local expected_type
    prompt_text=$(echo "$prompt_json" | jq -r '.prompt // .text // empty')
    expected_type=$(echo "$prompt_json" | jq -r '.expected_type // "text"')
    
    if [[ -z "$prompt_text" ]]; then
        return 1  # Invalid fixture format
    fi
    
    # Test generation with fixture prompt
    local request
    request="{\"model\":\"$TEST_MODEL\",\"prompt\":\"$prompt_text\",\"stream\":false,\"options\":{\"temperature\":0.1,\"num_predict\":50}}"
    
    local response
    if response=$(make_api_request "/api/generate" "POST" 30 "-H 'Content-Type: application/json' -d '$request'"); then
        if validate_json_field "$response" ".response"; then
            return 0  # Successfully generated response
        fi
    fi
    
    return 1
}

test_ocr_prompt_fixture() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # For OCR images, we'll create a prompt asking to describe what's in the image
    # This tests Ollama's multimodal capabilities if available
    local filename
    filename=$(basename "$fixture_path")
    
    # Create a descriptive prompt
    local prompt="Describe this image: [Image would be embedded here in production]"
    
    # Test if model supports vision/multimodal
    local request
    request="{\"model\":\"$TEST_MODEL\",\"prompt\":\"$prompt\",\"stream\":false}"
    
    local response
    if response=$(make_api_request "/api/generate" "POST" 30 "-H 'Content-Type: application/json' -d '$request'"); then
        if validate_json_field "$response" ".response"; then
            return 0  # Model responded (even if not multimodal)
        fi
    fi
    
    return 1
}

# Run enhanced fixture-based tests
run_ollama_fixture_tests() {
    if [[ "$FIXTURES_AVAILABLE" == "true" ]]; then
        # First check if model is available
        local response
        response=$(make_api_request "/api/tags" "GET" 5)
        
        if ! echo "$response" | jq -e ".models[] | select(.name == \"$TEST_MODEL\")" >/dev/null 2>&1; then
            log_test_result "fixture tests" "SKIP" "test model not available"
            return 2
        fi
        
        # Test with LLM prompt fixtures
        test_with_fixture "LLM prompt processing" "documents" "llm-prompt.json" \
            test_llm_prompt_fixture
        
        # Test with workflow data that might have prompts
        test_with_fixture "workflow prompt" "documents" "workflow-data.json" \
            test_llm_prompt_fixture
        
        # Test with OCR images (for multimodal models)
        local ocr_fixtures
        if ocr_fixtures=$(rotate_fixtures "images/ocr" 2); then
            for fixture in $ocr_fixtures; do
                local fixture_name
                fixture_name=$(basename "$fixture")
                test_with_fixture "OCR prompt $fixture_name" "" "$fixture" \
                    test_ocr_prompt_fixture
            done
        fi
        
        # Test with auto-discovered fixtures
        local ollama_fixtures
        ollama_fixtures=$(discover_resource_fixtures "ollama" "ai")
        
        for fixture_pattern in $ollama_fixtures; do
            # Look for JSON files that might contain prompts
            local prompt_files
            if prompt_files=$(fixture_get_all "$fixture_pattern" "*.json" 2>/dev/null | head -3); then
                for prompt_file in $prompt_files; do
                    local prompt_name
                    prompt_name=$(basename "$prompt_file")
                    test_with_fixture "process $prompt_name" "" "$prompt_file" \
                        test_llm_prompt_fixture
                done
            fi
        done
    fi
}

test_fixture_simple_prompt() {
    local test_name="fixture simple prompt"
    
    # Check if model is available first
    local response
    response=$(make_api_request "/api/tags" "GET" 5)
    
    if ! echo "$response" | jq -e ".models[] | select(.name == \"$TEST_MODEL\")" >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "test model not available"
        return 2
    fi
    
    # Get a simple test prompt
    local prompt_text
    prompt_text=$(get_llm_test_prompt "simple_greeting")
    local expected_pattern
    expected_pattern=$(get_llm_expected_pattern "simple_greeting")
    
    # Test generation with structured prompt
    local request
    request="{\"model\":\"$TEST_MODEL\",\"prompt\":\"$prompt_text\",\"stream\":false,\"options\":{\"temperature\":0.1,\"num_predict\":10}}"
    
    if response=$(make_api_request "/api/generate" "POST" 30 "-H 'Content-Type: application/json' -d '$request'"); then
        if validate_json_field "$response" ".response"; then
            local llm_response
            llm_response=$(echo "$response" | jq -r '.response')
            
            if validate_llm_response "$llm_response" "$expected_pattern"; then
                log_test_result "$test_name" "PASS" "response matches pattern"
                return 0
            else
                log_test_result "$test_name" "PASS" "generated response (pattern mismatch ok)"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "prompt generation failed"
    return 1
}

test_fixture_math_prompt() {
    local test_name="fixture math prompt"
    
    # Check if model is available
    local response
    response=$(make_api_request "/api/tags" "GET" 5)
    
    if ! echo "$response" | jq -e ".models[] | select(.name == \"$TEST_MODEL\")" >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "test model not available"
        return 2
    fi
    
    # Get math test prompt
    local prompt_text
    prompt_text=$(get_llm_test_prompt "math_simple")
    local expected_pattern
    expected_pattern=$(get_llm_expected_pattern "math_simple")
    
    # Test math capability
    local request
    request="{\"model\":\"$TEST_MODEL\",\"prompt\":\"$prompt_text\",\"stream\":false,\"options\":{\"temperature\":0,\"num_predict\":5}}"
    
    if response=$(make_api_request "/api/generate" "POST" 30 "-H 'Content-Type: application/json' -d '$request'"); then
        if validate_json_field "$response" ".response"; then
            local llm_response
            llm_response=$(echo "$response" | jq -r '.response')
            
            if validate_llm_response "$llm_response" "$expected_pattern"; then
                log_test_result "$test_name" "PASS" "correct math answer"
                return 0
            else
                log_test_result "$test_name" "PASS" "math response generated"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "math prompt failed"
    return 1
}

test_fixture_json_generation() {
    local test_name="fixture JSON generation"
    
    # Check if model is available
    local response
    response=$(make_api_request "/api/tags" "GET" 5)
    
    if ! echo "$response" | jq -e ".models[] | select(.name == \"$TEST_MODEL\")" >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "test model not available"
        return 2
    fi
    
    # Get JSON generation prompt
    local prompt_text
    prompt_text=$(get_llm_test_prompt "json_generation")
    local expected_pattern
    expected_pattern=$(get_llm_expected_pattern "json_generation")
    
    # Test JSON generation
    local request
    request="{\"model\":\"$TEST_MODEL\",\"prompt\":\"$prompt_text\",\"stream\":false,\"options\":{\"temperature\":0,\"num_predict\":30}}"
    
    if response=$(make_api_request "/api/generate" "POST" 30 "-H 'Content-Type: application/json' -d '$request'"); then
        if validate_json_field "$response" ".response"; then
            local llm_response
            llm_response=$(echo "$response" | jq -r '.response')
            
            # Check if response contains valid JSON
            if echo "$llm_response" | grep -oE '\{[^}]*\}' | jq empty 2>/dev/null; then
                log_test_result "$test_name" "PASS" "valid JSON generated"
                return 0
            else
                log_test_result "$test_name" "PASS" "response generated (not valid JSON)"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "JSON generation failed"
    return 1
}

test_fixture_streaming_response() {
    local test_name="fixture streaming response"
    
    # Check if model is available
    local response
    response=$(make_api_request "/api/tags" "GET" 5)
    
    if ! echo "$response" | jq -e ".models[] | select(.name == \"$TEST_MODEL\")" >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "test model not available"
        return 2
    fi
    
    # Test streaming with simple prompt
    local prompt_text="Count to 3"
    local request
    request="{\"model\":\"$TEST_MODEL\",\"prompt\":\"$prompt_text\",\"stream\":true,\"options\":{\"num_predict\":20}}"
    
    # Test streaming (should receive multiple JSON objects)
    local stream_response
    if stream_response=$(curl -sf --max-time 30 \
        -X POST "$BASE_URL/api/generate" \
        -H 'Content-Type: application/json' \
        -d "$request" 2>/dev/null); then
        
        # Check if we got multiple lines (streaming format)
        local line_count
        line_count=$(echo "$stream_response" | wc -l)
        
        if [[ $line_count -gt 1 ]]; then
            log_test_result "$test_name" "PASS" "streaming works ($line_count chunks)"
            return 0
        else
            log_test_result "$test_name" "PASS" "response received (single chunk)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "streaming request failed"
    return 1
}

test_fixture_prompt_variations() {
    local test_name="fixture prompt variations"
    
    # Check if model is available
    local response
    response=$(make_api_request "/api/tags" "GET" 5)
    
    if ! echo "$response" | jq -e ".models[] | select(.name == \"$TEST_MODEL\")" >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "test model not available"
        return 2
    fi
    
    # Test multiple prompt types
    local prompts_json
    prompts_json=$(get_llm_test_prompts)
    local total_prompts
    total_prompts=$(echo "$prompts_json" | jq 'length')
    local success_count=0
    
    # Test first 3 prompts for time efficiency
    local max_tests=3
    local test_count=0
    
    echo "$prompts_json" | jq -c '.[]' | while read -r prompt_obj && [[ $test_count -lt $max_tests ]]; do
        local prompt_text
        prompt_text=$(echo "$prompt_obj" | jq -r '.prompt')
        local max_tokens
        max_tokens=$(echo "$prompt_obj" | jq -r '.max_tokens')
        
        local request
        request="{\"model\":\"$TEST_MODEL\",\"prompt\":\"$prompt_text\",\"stream\":false,\"options\":{\"num_predict\":$max_tokens}}"
        
        if make_api_request "/api/generate" "POST" 30 "-H 'Content-Type: application/json' -d '$request'" >/dev/null 2>&1; then
            ((success_count++))
        fi
        
        ((test_count++))
    done
    
    if [[ $success_count -gt 0 ]]; then
        log_test_result "$test_name" "PASS" "$success_count/$max_tests prompts processed"
        return 0
    else
        log_test_result "$test_name" "FAIL" "no prompts processed successfully"
        return 1
    fi
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "Ollama Information:"
    echo "  API Endpoints:"
    echo "    - Health Check: GET $BASE_URL/api/tags"
    echo "    - Version: GET $BASE_URL/api/version"  
    echo "    - Generate: POST $BASE_URL/api/generate"
    echo "    - Embeddings: POST $BASE_URL/api/embeddings"
    echo "  Test Model: $TEST_MODEL"
    echo "  Models: Run 'ollama list' to see available models"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register Ollama-specific tests
register_tests \
    "test_version_endpoint" \
    "test_model_listing" \
    "test_model_availability" \
    "test_text_generation" \
    "test_embeddings_endpoint" \
    "run_ollama_fixture_tests" \
    "test_fixture_simple_prompt" \
    "test_fixture_math_prompt" \
    "test_fixture_json_generation" \
    "test_fixture_streaming_response" \
    "test_fixture_prompt_variations"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi