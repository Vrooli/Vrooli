#!/usr/bin/env bash
################################################################################
# Codex Unit Tests
# Test individual library functions (<60s)
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CODEX_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${CODEX_DIR}/../.." && pwd)"

# Source utilities and libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${CODEX_DIR}/config/defaults.sh"
source "${CODEX_DIR}/lib/common.sh"

# Test results
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper functions
test_pass() {
    ((TESTS_PASSED++))
    log::success "✓ $1"
}

test_fail() {
    ((TESTS_FAILED++))
    log::error "✗ $1: $2"
}

run_test() {
    ((TESTS_RUN++))
    local test_name="$1"
    local expected="$2"
    shift 2
    local result=$("$@" 2>/dev/null || echo "ERROR")
    
    if [[ "$result" == "$expected" ]]; then
        test_pass "$test_name"
        return 0
    else
        test_fail "$test_name" "Expected '$expected', got '$result'"
        return 1
    fi
}

# Unit Tests

test_config_defaults() {
    log::info "Testing configuration defaults..."
    
    ((TESTS_RUN++))
    if [[ -n "${CODEX_NAME}" ]] && [[ "${CODEX_NAME}" == "codex" ]]; then
        test_pass "CODEX_NAME set correctly"
    else
        test_fail "CODEX_NAME" "Expected 'codex', got '${CODEX_NAME:-empty}'"
    fi
    
    ((TESTS_RUN++))
    if [[ -n "${CODEX_API_ENDPOINT}" ]]; then
        test_pass "CODEX_API_ENDPOINT has default"
    else
        test_fail "CODEX_API_ENDPOINT" "No default endpoint set"
    fi
    
    ((TESTS_RUN++))
    if [[ "${CODEX_DEFAULT_MODEL}" == "gpt-3.5-turbo" ]] || [[ "${CODEX_DEFAULT_MODEL}" == "code-davinci-002" ]]; then
        test_pass "CODEX_DEFAULT_MODEL set"
    else
        test_fail "CODEX_DEFAULT_MODEL" "Unexpected model: ${CODEX_DEFAULT_MODEL}"
    fi
}

test_directory_creation() {
    log::info "Testing directory creation..."
    
    ((TESTS_RUN++))
    if [[ -d "${CODEX_SCRIPTS_DIR}" ]]; then
        test_pass "Scripts directory exists"
    else
        test_fail "Scripts directory" "Directory not created: ${CODEX_SCRIPTS_DIR}"
    fi
    
    ((TESTS_RUN++))
    if [[ -d "${CODEX_OUTPUT_DIR}" ]]; then
        test_pass "Output directory exists"
    else
        test_fail "Output directory" "Directory not created: ${CODEX_OUTPUT_DIR}"
    fi
}

test_api_key_retrieval() {
    log::info "Testing API key retrieval..."
    
    # Test with environment variable
    local original_key="${OPENAI_API_KEY:-}"
    export OPENAI_API_KEY="test-key-123"
    
    ((TESTS_RUN++))
    local retrieved_key=$(codex::get_api_key)
    if [[ "$retrieved_key" == "test-key-123" ]]; then
        test_pass "API key from environment"
    else
        test_fail "API key retrieval" "Failed to get key from environment"
    fi
    
    # Restore original
    if [[ -n "$original_key" ]]; then
        export OPENAI_API_KEY="$original_key"
    else
        unset OPENAI_API_KEY
    fi
}

test_configuration_check() {
    log::info "Testing configuration check..."
    
    # Test without API key
    unset OPENAI_API_KEY 2>/dev/null || true
    
    ((TESTS_RUN++))
    if ! codex::is_configured; then
        test_pass "Detects missing configuration"
    else
        test_fail "Configuration check" "Should detect missing API key"
    fi
    
    # Test with API key
    export OPENAI_API_KEY="test-key"
    ((TESTS_RUN++))
    if codex::is_configured; then
        test_pass "Detects valid configuration"
    else
        test_fail "Configuration check" "Should detect API key presence"
    fi
    
    unset OPENAI_API_KEY
}

test_json_validation() {
    log::info "Testing JSON configuration files..."
    
    # Test runtime.json
    ((TESTS_RUN++))
    if jq empty "${CODEX_DIR}/config/runtime.json" &>/dev/null; then
        test_pass "runtime.json is valid JSON"
    else
        test_fail "runtime.json validation" "Invalid JSON format"
    fi
    
    # Test schema.json
    ((TESTS_RUN++))
    if jq empty "${CODEX_DIR}/config/schema.json" &>/dev/null; then
        test_pass "schema.json is valid JSON"
    else
        test_fail "schema.json validation" "Invalid JSON format"
    fi
}

test_yaml_validation() {
    log::info "Testing YAML configuration files..."
    
    # Test secrets.yaml (if yq is available)
    if command -v yq &>/dev/null; then
        ((TESTS_RUN++))
        if yq eval '.' "${CODEX_DIR}/config/secrets.yaml" &>/dev/null; then
            test_pass "secrets.yaml is valid YAML"
        else
            test_fail "secrets.yaml validation" "Invalid YAML format"
        fi
    else
        log::warning "yq not available, skipping YAML validation"
    fi
}

# Main execution
main() {
    log::info "Running Codex unit tests..."
    
    # Run all tests
    test_config_defaults
    test_directory_creation
    test_api_key_retrieval
    test_configuration_check
    test_json_validation
    test_yaml_validation
    
    # Report results
    log::info "Test Results: $TESTS_PASSED/$TESTS_RUN passed"
    
    if [ $TESTS_FAILED -gt 0 ]; then
        log::error "Unit tests failed: $TESTS_FAILED test(s) failed"
        exit 1
    fi
    
    log::success "All unit tests passed!"
    exit 0
}

main "$@"