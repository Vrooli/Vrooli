#!/bin/bash
# LiteLLM test functionality

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
LITELLM_TEST_DIR="${SCRIPT_DIR}"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
LITELLM_RESOURCE_DIR="${RESOURCE_DIR}"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# Source dependencies
source "${LITELLM_TEST_DIR}/core.sh"
source "${LITELLM_TEST_DIR}/docker.sh"

# Smoke test - Basic health check (required by Universal Contract)
litellm::test::smoke() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Running LiteLLM smoke test"
    
    # Check if container is running
    if ! litellm::is_running; then
        echo "❌ Smoke test failed: LiteLLM container not running"
        return 1
    fi
    
    # Test basic connectivity
    if ! litellm::test_connection 10 "$verbose"; then
        echo "❌ Smoke test failed: API not responding"
        return 1
    fi
    
    echo "✅ Smoke test passed: LiteLLM is healthy"
    return 0
}

# Integration test - Full functionality test (required by Universal Contract)
litellm::test::integration() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Running LiteLLM integration test"
    
    # Run smoke test first
    if ! litellm::test::smoke "$verbose"; then
        echo "❌ Integration test failed: Smoke test did not pass"
        return 1
    fi
    
    # Test model listing
    local models
    models=$(litellm::list_models 10 false 2>/dev/null)
    if [[ $? -ne 0 ]]; then
        echo "❌ Integration test failed: Cannot list models"
        return 1
    fi
    
    # Test configuration access
    if [[ ! -f "$LITELLM_CONFIG_FILE" ]]; then
        echo "❌ Integration test failed: Configuration file not found"
        return 1
    fi
    
    # Test master key availability
    if ! litellm::get_master_key >/dev/null 2>&1; then
        echo "❌ Integration test failed: Master key not accessible"
        return 1
    fi
    
    echo "✅ Integration test passed: Full functionality verified"
    return 0
}

# All tests - Run all available tests (required by Universal Contract)
litellm::test::all() {
    local verbose="${1:-false}"
    local failed=0
    
    [[ "$verbose" == "true" ]] && log::info "Running all LiteLLM tests"
    
    echo "🧪 Running LiteLLM Test Suite"
    echo "=============================="
    
    # Run smoke test
    echo -n "Smoke test... "
    if litellm::test::smoke "$verbose"; then
        echo "PASS"
    else
        echo "FAIL"
        failed=$((failed + 1))
    fi
    
    # Run integration test
    echo -n "Integration test... "
    if litellm::test::integration "$verbose"; then
        echo "PASS"
    else
        echo "FAIL" 
        failed=$((failed + 1))
    fi
    
    # Summary
    echo "=============================="
    if [[ $failed -eq 0 ]]; then
        echo "✅ All tests passed (2/2)"
        return 0
    else
        echo "❌ Some tests failed ($failed/2 failed)"
        return 1
    fi
}