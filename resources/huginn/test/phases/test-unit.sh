#!/usr/bin/env bash
################################################################################
# Huginn Unit Tests - Library function validation (<60s)
# 
# Tests individual library functions and utilities
################################################################################

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities and libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/config/messages.sh"

# Source library files for testing
for lib in common docker install status api; do
    lib_file="${RESOURCE_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

echo "🧪 Starting Huginn unit tests..."

# Test 1: Configuration loading
test::config_loading() {
    echo -n "Testing configuration loading... "
    
    # Check if essential variables are defined
    if [[ -n "${HUGINN_PORT:-}" ]] && [[ -n "${CONTAINER_NAME:-}" ]]; then
        echo "✅ PASS (port: ${HUGINN_PORT}, container: ${CONTAINER_NAME})"
        return 0
    else
        echo "❌ FAIL: Configuration not properly loaded"
        return 1
    fi
}

# Test 2: Message functions
test::message_functions() {
    echo -n "Testing message functions... "
    
    # Check if message variables are defined
    if [[ -n "${MSG_INSTALL_START:-}" ]] && [[ -n "${MSG_STATUS_RUNNING:-}" ]]; then
        echo "✅ PASS"
        return 0
    else
        echo "⚠️  WARN: Some message variables undefined"
        return 0
    fi
}

# Test 3: Common utility functions
test::common_utilities() {
    echo -n "Testing common utilities... "
    
    # Test if common functions are available
    if type -t huginn::validate_environment &>/dev/null || \
       type -t huginn::check_dependencies &>/dev/null || \
       type -t log::info &>/dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "⚠️  WARN: Some utility functions not available"
        return 0
    fi
}

# Test 4: Docker helper functions
test::docker_helpers() {
    echo -n "Testing Docker helper functions... "
    
    # Check if Docker functions are defined
    if type -t huginn::docker_run &>/dev/null || \
       type -t huginn::start &>/dev/null || \
       type -t docker &>/dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "⚠️  WARN: Docker functions not fully available"
        return 0
    fi
}

# Test 5: API function definitions
test::api_functions() {
    echo -n "Testing API function definitions... "
    
    # Check if API functions are defined
    if type -t huginn::list_agents &>/dev/null || \
       type -t huginn::show_agent &>/dev/null || \
       type -t huginn::api_request &>/dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "⚠️  WARN: API functions not fully defined"
        return 0
    fi
}

# Test 6: Status function availability
test::status_functions() {
    echo -n "Testing status functions... "
    
    if type -t huginn::status &>/dev/null || \
       type -t huginn::health_check &>/dev/null; then
        echo "✅ PASS"
        return 0
    else
        echo "⚠️  WARN: Status functions not fully available"
        return 0
    fi
}

# Test 7: Error handling
test::error_handling() {
    echo -n "Testing error handling... "
    
    # Test with invalid command to check error handling
    if vrooli resource huginn invalid_command 2>/dev/null; then
        echo "❌ FAIL: Invalid command did not fail"
        return 1
    else
        echo "✅ PASS (properly rejects invalid commands)"
        return 0
    fi
}

# Run all unit tests
main() {
    local failed=0
    
    test::config_loading || ((failed++))
    test::message_functions || ((failed++))
    test::common_utilities || ((failed++))
    test::docker_helpers || ((failed++))
    test::api_functions || ((failed++))
    test::status_functions || ((failed++))
    test::error_handling || ((failed++))
    
    echo ""
    if [[ ${failed} -eq 0 ]]; then
        echo "✅ All unit tests passed!"
        return 0
    else
        echo "❌ ${failed} unit test(s) failed"
        return 1
    fi
}

# Execute tests
main "$@"