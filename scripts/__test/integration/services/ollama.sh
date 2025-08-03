#!/usr/bin/env bash
# Ollama Service Integration Test
# Tests real Ollama service functionality (not mocked)

set -euo pipefail

# Configuration
readonly SERVICE_NAME="ollama"
readonly BASE_URL="${OLLAMA_BASE_URL:-http://localhost:11434}"
readonly TEST_MODEL="${OLLAMA_TEST_MODEL:-llama3.2:1b}"
readonly TIMEOUT="${OLLAMA_TIMEOUT:-30}"

# Colors for output
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly NC='\033[0m'

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0

# Test functions
test_ollama_api_available() {
    echo -n "Testing Ollama API availability... "
    
    if curl -sf --max-time 5 "$BASE_URL/api/tags" > /dev/null; then
        echo -e "${GREEN}PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

test_ollama_version() {
    echo -n "Testing Ollama version endpoint... "
    
    local response
    response=$(curl -sf --max-time 5 "$BASE_URL/api/version" 2>/dev/null)
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.version' > /dev/null 2>&1; then
        local version=$(echo "$response" | jq -r '.version')
        echo -e "${GREEN}PASS${NC} (version: $version)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

test_ollama_list_models() {
    echo -n "Testing model listing... "
    
    local response
    response=$(curl -sf --max-time 5 "$BASE_URL/api/tags" 2>/dev/null)
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.models' > /dev/null 2>&1; then
        local count=$(echo "$response" | jq '.models | length')
        echo -e "${GREEN}PASS${NC} (models: $count)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

test_ollama_model_available() {
    echo -n "Testing if model $TEST_MODEL is available... "
    
    local response
    response=$(curl -sf --max-time 5 "$BASE_URL/api/tags" 2>/dev/null)
    
    if echo "$response" | jq -e ".models[] | select(.name == \"$TEST_MODEL\")" > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}SKIP${NC} (model not found, would need to pull)"
        return 0  # Don't fail test if model isn't available
    fi
}

test_ollama_generate() {
    echo -n "Testing text generation... "
    
    # Only run if model is available
    local response
    response=$(curl -sf --max-time 5 "$BASE_URL/api/tags" 2>/dev/null)
    
    if ! echo "$response" | jq -e ".models[] | select(.name == \"$TEST_MODEL\")" > /dev/null 2>&1; then
        echo -e "${GREEN}SKIP${NC} (model not available)"
        return 0
    fi
    
    # Test generation
    local prompt='{"model":"'"$TEST_MODEL"'","prompt":"Say hello","stream":false}'
    response=$(curl -sf --max-time "$TIMEOUT" -X POST "$BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d "$prompt" 2>/dev/null)
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.response' > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

test_ollama_embeddings() {
    echo -n "Testing embeddings endpoint... "
    
    # Only run if model supports embeddings
    local response
    response=$(curl -sf --max-time 5 "$BASE_URL/api/tags" 2>/dev/null)
    
    if ! echo "$response" | jq -e '.models[] | select(.name | contains("embed"))' > /dev/null 2>&1; then
        echo -e "${GREEN}SKIP${NC} (no embedding model available)"
        return 0
    fi
    
    # Test embeddings
    local embed_model=$(echo "$response" | jq -r '.models[] | select(.name | contains("embed")) | .name' | head -1)
    local request='{"model":"'"$embed_model"'","prompt":"test text"}'
    
    response=$(curl -sf --max-time "$TIMEOUT" -X POST "$BASE_URL/api/embeddings" \
        -H "Content-Type: application/json" \
        -d "$request" 2>/dev/null)
    
    if [[ -n "$response" ]] && echo "$response" | jq -e '.embedding' > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Main execution
main() {
    echo "======================================="
    echo "    Ollama Integration Test Suite     "
    echo "======================================="
    echo "Base URL: $BASE_URL"
    echo "Test Model: $TEST_MODEL"
    echo
    
    # Run tests
    test_ollama_api_available
    test_ollama_version
    test_ollama_list_models
    test_ollama_model_available
    test_ollama_generate
    test_ollama_embeddings
    
    # Summary
    echo
    echo "======================================="
    echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed${NC}"
        exit 1
    fi
}

# Check for required tools
command -v curl >/dev/null 2>&1 || { echo "curl is required"; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq is required"; exit 1; }

# Run main
main