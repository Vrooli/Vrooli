#!/usr/bin/env bash
# DeepStack Resource - Unit Tests
# Library function validation (must complete in <60s)

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly TEST_DIR="$(dirname "${SCRIPT_DIR}")"
readonly RESOURCE_DIR="$(dirname "${TEST_DIR}")"

# Load test library
source "${RESOURCE_DIR}/lib/test.sh"

# Timeout for unit tests
UNIT_TIMEOUT=60

# Run unit tests
run_unit_tests() {
    local start_time=$(date +%s)
    local failures=0
    
    echo "DeepStack Unit Tests"
    echo "==================="
    
    # Test 1: Configuration loading
    echo ""
    echo "Test 1: Configuration Loading"
    source "${RESOURCE_DIR}/config/defaults.sh"
    
    if [[ -n "${DEEPSTACK_PORT}" ]]; then
        echo "✓ Configuration loaded: PORT=${DEEPSTACK_PORT}"
    else
        echo "✗ Failed to load configuration" >&2
        ((failures++))
    fi
    
    # Test 2: Test library functions
    echo ""
    echo "Test 2: Test Library Functions"
    
    # Test command existence check
    if test_command_exists "bash"; then
        echo "✓ test_command_exists function works"
    else
        echo "✗ test_command_exists function failed" >&2
        ((failures++))
    fi
    
    # Test 3: Directory validation
    echo ""
    echo "Test 3: Directory Validation"
    
    # Create temp directory for testing
    local test_dir="/tmp/deepstack_test_$$"
    mkdir -p "$test_dir"
    
    if test_directory_writable "$test_dir"; then
        echo "✓ test_directory_writable function works"
    else
        echo "✗ test_directory_writable function failed" >&2
        ((failures++))
    fi
    
    rm -rf "$test_dir"
    
    # Test 4: JSON parsing utilities
    echo ""
    echo "Test 4: JSON Utilities"
    
    local test_json='{"success":true,"predictions":[{"label":"person","confidence":0.9}]}'
    
    if echo "$test_json" | jq -e '.success == true' &> /dev/null; then
        echo "✓ JSON parsing works"
        local label=$(echo "$test_json" | jq -r '.predictions[0].label')
        if [[ "$label" == "person" ]]; then
            echo "✓ JSON extraction works"
        else
            echo "✗ JSON extraction failed" >&2
            ((failures++))
        fi
    else
        echo "✗ JSON parsing failed" >&2
        ((failures++))
    fi
    
    # Test 5: Runtime configuration validation
    echo ""
    echo "Test 5: Runtime Configuration"
    
    local runtime_file="${RESOURCE_DIR}/config/runtime.json"
    if [[ -f "$runtime_file" ]]; then
        local startup_order=$(jq -r '.startup_order' "$runtime_file")
        if [[ "$startup_order" == "350" ]]; then
            echo "✓ Runtime configuration valid"
        else
            echo "✗ Invalid startup_order: $startup_order" >&2
            ((failures++))
        fi
    else
        echo "✗ Runtime configuration not found" >&2
        ((failures++))
    fi
    
    # Test 6: Environment variable handling
    echo ""
    echo "Test 6: Environment Variables"
    
    # Test with override
    local original_port="${DEEPSTACK_PORT}"
    export DEEPSTACK_PORT="12345"
    source "${RESOURCE_DIR}/config/defaults.sh"
    
    if [[ "${DEEPSTACK_PORT}" == "12345" ]]; then
        echo "✓ Environment variable override works"
    else
        echo "✗ Environment variable override failed" >&2
        ((failures++))
    fi
    
    # Restore original
    export DEEPSTACK_PORT="${original_port}"
    
    # Test 7: Helper function validation
    echo ""
    echo "Test 7: Helper Functions"
    
    # Test validation functions exist
    if declare -f run_validation_tests &> /dev/null; then
        echo "✓ Validation test functions available"
    else
        echo "✗ Validation test functions not found" >&2
        ((failures++))
    fi
    
    # Test 8: Schema validation
    echo ""
    echo "Test 8: Schema Validation"
    
    local schema_file="${RESOURCE_DIR}/config/schema.json"
    if [[ -f "$schema_file" ]]; then
        if jq -e '.properties.service.properties.port.default == 11453' "$schema_file" &> /dev/null; then
            echo "✓ Schema configuration valid"
        else
            echo "✗ Invalid schema configuration" >&2
            ((failures++))
        fi
    else
        echo "✗ Schema file not found" >&2
        ((failures++))
    fi
    
    # Check total execution time
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "Test duration: ${duration}s"
    
    if [[ $duration -gt $UNIT_TIMEOUT ]]; then
        echo "⚠ Warning: Unit tests exceeded ${UNIT_TIMEOUT}s timeout" >&2
    fi
    
    if [[ $failures -gt 0 ]]; then
        echo ""
        echo "✗ Unit tests failed with $failures error(s)" >&2
        return 1
    else
        echo ""
        echo "✓ All unit tests passed"
        return 0
    fi
}

# Run with timeout enforcement
if timeout "$UNIT_TIMEOUT" bash -c "$(declare -f run_unit_tests); $(declare -f test_command_exists); $(declare -f test_directory_writable); source '${RESOURCE_DIR}/config/defaults.sh'; source '${RESOURCE_DIR}/lib/test.sh'; run_unit_tests"; then
    exit 0
else
    exit_code=$?
    if [[ $exit_code -eq 124 ]]; then
        echo "✗ Unit tests timed out after ${UNIT_TIMEOUT}s" >&2
    fi
    exit $exit_code
fi