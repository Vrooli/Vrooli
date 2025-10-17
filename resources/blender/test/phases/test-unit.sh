#!/usr/bin/env bash
################################################################################
# Blender Unit Test - Library Function Validation
# 
# Tests individual functions in the Blender library files
# Validates: core functions, inject operations, helper utilities
#
################################################################################

set -euo pipefail

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities and Blender libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/inject.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
test_start() {
    echo "[TEST] $1"
    ((TESTS_RUN++))
}

test_pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++))
}

test_fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++))
}

# Test 1: Configuration loading
test_config_loading() {
    test_start "Configuration loading"
    
    # Check required environment variables are set
    if [[ -n "${BLENDER_PORT}" ]]; then
        test_pass "BLENDER_PORT is set: ${BLENDER_PORT}"
    else
        test_fail "BLENDER_PORT not set"
    fi
    
    if [[ -n "${BLENDER_DATA_DIR}" ]]; then
        test_pass "BLENDER_DATA_DIR is set: ${BLENDER_DATA_DIR}"
    else
        test_fail "BLENDER_DATA_DIR not set"
    fi
    
    if [[ -n "${BLENDER_CONTAINER_NAME}" ]]; then
        test_pass "BLENDER_CONTAINER_NAME is set: ${BLENDER_CONTAINER_NAME}"
    else
        test_fail "BLENDER_CONTAINER_NAME not set"
    fi
}

# Test 2: Directory initialization
test_directory_init() {
    test_start "Directory initialization (blender::init)"
    
    # Run init function
    blender::init
    
    # Check directories were created
    local all_created=true
    for dir in "$BLENDER_DATA_DIR" "$BLENDER_SCRIPTS_DIR" "$BLENDER_OUTPUT_DIR" \
               "${BLENDER_DATA_DIR}/config" "${BLENDER_DATA_DIR}/addons" "${BLENDER_DATA_DIR}/temp"; do
        if [[ -d "$dir" ]]; then
            test_pass "Created: $(basename $dir)"
        else
            test_fail "Failed to create: $dir"
            all_created=false
        fi
    done
    
    # Check config file
    if [[ -f "${BLENDER_DATA_DIR}/config/blender.conf" ]]; then
        test_pass "Config file created"
    else
        test_fail "Config file not created"
        all_created=false
    fi
    
    [[ "$all_created" == true ]]
}

# Test 3: Script path validation
test_script_validation() {
    test_start "Script path validation (blender::validate_script)"
    
    # Create test scripts
    local valid_script=$(mktemp /tmp/valid_XXXXX.py)
    local invalid_script="/tmp/nonexistent_$(date +%s).py"
    
    echo "print('test')" > "$valid_script"
    
    # Test valid script
    if blender::validate_script "$valid_script" 2>/dev/null; then
        test_pass "Valid script accepted"
    else
        test_fail "Valid script rejected"
    fi
    
    # Test invalid script
    if ! blender::validate_script "$invalid_script" 2>/dev/null; then
        test_pass "Invalid script rejected"
    else
        test_fail "Invalid script accepted"
    fi
    
    # Test non-Python file
    local non_python=$(mktemp /tmp/test_XXXXX.txt)
    if ! blender::validate_script "$non_python" 2>/dev/null; then
        test_pass "Non-Python file rejected"
    else
        test_fail "Non-Python file accepted"
    fi
    
    rm -f "$valid_script" "$non_python"
}

# Test 4: List injected scripts
test_list_injected() {
    test_start "List injected scripts (blender::list_injected)"
    
    # Create test scripts in scripts directory
    touch "${BLENDER_SCRIPTS_DIR}/test_1.py"
    touch "${BLENDER_SCRIPTS_DIR}/test_2.py"
    touch "${BLENDER_SCRIPTS_DIR}/test_3.blend"  # Non-Python file
    
    # Get list
    local output=$(blender::list_injected 2>/dev/null)
    
    if echo "$output" | grep -q "test_1.py"; then
        test_pass "Found test_1.py"
    else
        test_fail "Missing test_1.py"
    fi
    
    if echo "$output" | grep -q "test_2.py"; then
        test_pass "Found test_2.py"
    else
        test_fail "Missing test_2.py"
    fi
    
    # Clean up
    rm -f "${BLENDER_SCRIPTS_DIR}/test_"*.py "${BLENDER_SCRIPTS_DIR}/test_"*.blend
}

# Test 5: Export listing
test_export_list() {
    test_start "Export file listing (blender::list_exports)"
    
    # Create test output files
    touch "${BLENDER_OUTPUT_DIR}/render_001.png"
    touch "${BLENDER_OUTPUT_DIR}/model.obj"
    touch "${BLENDER_OUTPUT_DIR}/animation.mp4"
    
    # Get list
    local output=$(blender::list_exports 2>/dev/null)
    
    local found_count=0
    for file in "render_001.png" "model.obj" "animation.mp4"; do
        if echo "$output" | grep -q "$file"; then
            ((found_count++))
        fi
    done
    
    if [[ $found_count -eq 3 ]]; then
        test_pass "All test exports listed"
    else
        test_fail "Only $found_count/3 exports found"
    fi
    
    # Clean up
    rm -f "${BLENDER_OUTPUT_DIR}/render_001.png" \
          "${BLENDER_OUTPUT_DIR}/model.obj" \
          "${BLENDER_OUTPUT_DIR}/animation.mp4"
}

# Test 6: Port registry integration
test_port_registry() {
    test_start "Port registry integration"
    
    # Source port registry
    if [[ -f "${APP_ROOT}/scripts/resources/port_registry.sh" ]]; then
        source "${APP_ROOT}/scripts/resources/port_registry.sh"
        
        # Check if Blender port matches registry
        if [[ "${RESOURCE_PORTS[blender]}" == "8093" ]]; then
            test_pass "Port registry has correct Blender port"
        else
            test_fail "Port registry has wrong port: ${RESOURCE_PORTS[blender]}"
        fi
        
        # Verify our config uses registry port
        if [[ "${BLENDER_PORT}" == "8093" ]]; then
            test_pass "Config uses registry port"
        else
            test_fail "Config not using registry port: ${BLENDER_PORT}"
        fi
    else
        test_fail "Port registry not found"
    fi
}

# Main test execution
main() {
    echo "======================================"
    echo "Blender Unit Tests"
    echo "======================================"
    
    # Run all tests
    test_config_loading || true
    test_directory_init || true
    test_script_validation || true
    test_list_injected || true
    test_export_list || true
    test_port_registry || true
    
    # Summary
    echo "======================================"
    echo "Tests Run: $TESTS_RUN"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    echo "======================================"
    
    # Return appropriate exit code
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "[SUCCESS] All unit tests passed"
        exit 0
    else
        echo "[FAILURE] $TESTS_FAILED test(s) failed"
        exit 1
    fi
}

main "$@"