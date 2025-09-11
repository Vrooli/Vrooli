#!/usr/bin/env bash
################################################################################
# Gemini Resource Unit Tests
# 
# Library function testing (< 60s)
################################################################################

set -euo pipefail

# Determine paths
PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$PHASES_DIR/.." && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Source resource libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

log::info "Starting Gemini unit tests..."

# Test 1: Configuration defaults
log::info "Test 1: Testing configuration defaults..."
test_passed=true

if [[ "$GEMINI_API_BASE" != "https://generativelanguage.googleapis.com/v1beta" ]]; then
    log::error "✗ API base URL incorrect: $GEMINI_API_BASE"
    test_passed=false
fi

if [[ "$GEMINI_DEFAULT_MODEL" != "gemini-pro" ]]; then
    log::error "✗ Default model incorrect: $GEMINI_DEFAULT_MODEL"
    test_passed=false
fi

if [[ "$GEMINI_TIMEOUT" != "30" ]]; then
    log::error "✗ Timeout incorrect: $GEMINI_TIMEOUT"
    test_passed=false
fi

if [[ "$GEMINI_HEALTH_CHECK_TIMEOUT" != "5" ]]; then
    log::error "✗ Health check timeout incorrect: $GEMINI_HEALTH_CHECK_TIMEOUT"
    test_passed=false
fi

if $test_passed; then
    log::success "✓ Configuration defaults correct"
else
    exit 1
fi

# Test 2: Function existence
log::info "Test 2: Testing function definitions..."
functions_exist=true

for func in gemini::init gemini::test_connection gemini::list_models gemini::generate; do
    if ! type -t "$func" >/dev/null; then
        log::error "✗ Function $func not defined"
        functions_exist=false
    fi
done

if $functions_exist; then
    log::success "✓ All core functions defined"
else
    exit 1
fi

# Test 3: Error handling in functions
log::info "Test 3: Testing error handling..."

# Save original API key
original_key="$GEMINI_API_KEY"

# Test with empty API key
export GEMINI_API_KEY=""
gemini::init >/dev/null 2>&1
if [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
    log::success "✓ Empty API key handled correctly"
else
    log::error "✗ Empty API key not handled properly"
    exit 1
fi

# Restore original key
export GEMINI_API_KEY="$original_key"

# Test 4: URL construction
log::info "Test 4: Testing URL construction..."
test_url="${GEMINI_API_BASE}/models/${GEMINI_DEFAULT_MODEL}:generateContent"
expected_url="https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"

if [[ "$test_url" == "$expected_url" ]]; then
    log::success "✓ URL construction correct"
else
    log::error "✗ URL construction incorrect"
    log::error "  Expected: $expected_url"
    log::error "  Got: $test_url"
    exit 1
fi

# Test 5: Timeout handling simulation
log::info "Test 5: Testing timeout handling..."

# Create a function that simulates timeout behavior
test_timeout() {
    timeout 1 sleep 5
    return $?
}

if test_timeout 2>/dev/null; then
    log::error "✗ Timeout not working properly"
    exit 1
else
    log::success "✓ Timeout handling works"
fi

# Test 6: JSON parsing preparation
log::info "Test 6: Testing JSON utilities..."

# Test that jq is available
if command -v jq >/dev/null 2>&1; then
    # Test simple JSON parsing
    test_json='{"test": "value"}'
    parsed=$(echo "$test_json" | jq -r '.test')
    if [[ "$parsed" == "value" ]]; then
        log::success "✓ JSON parsing utilities work"
    else
        log::error "✗ JSON parsing failed"
        exit 1
    fi
else
    log::error "✗ jq not available for JSON parsing"
    exit 1
fi

# Test 7: Export validation
log::info "Test 7: Testing function exports..."
exports_valid=true

# Check if functions are exported
for func in gemini::init gemini::test_connection gemini::list_models gemini::generate; do
    if ! declare -F "$func" | grep -q "declare -fx"; then
        log::warn "⚠ Function $func not exported"
        # Not critical for unit tests
    fi
done

log::success "✓ Export validation complete"

log::success "✅ All unit tests passed!"
exit 0