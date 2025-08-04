#!/usr/bin/env bash
# Ollama Integration Test
# Tests real Ollama service functionality
# Refactored to use shared integration test library

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../../tests/lib/integration-test-lib.sh"

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
    "test_embeddings_endpoint"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi