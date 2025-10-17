#\!/usr/bin/env bash
set -u

SCRIPT_DIR="/home/matthalloran8/Vrooli/resources/unstructured-io/test/phases"
RESOURCE_DIR="/home/matthalloran8/Vrooli/resources/unstructured-io"
APP_ROOT="/home/matthalloran8/Vrooli"

set +e
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

for lib in common core install status api process cache-simple validate test content; do
    lib_file="${RESOURCE_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

set -e

log::header "DEBUG Unit Test"

tests_passed=0
tests_failed=0

test_function_exists() {
    local func_name="$1"
    local description="${2:-Function $func_name}"
    
    echo "DEBUG: About to test $func_name"
    
    if command -v "$func_name" &>/dev/null; then
        log::success "✓ $description exists"
        ((tests_passed++))
        echo "DEBUG: Test passed, continuing"
        return 0
    else
        log::error "✗ $description missing"
        ((tests_failed++))
        echo "DEBUG: Test failed"
        return 1
    fi
}

echo "DEBUG: About to start tests"

log::info "Test 1: Core lifecycle functions..."
echo "DEBUG: Testing install function"
test_function_exists "unstructured_io::install" "Install function"

echo "DEBUG: Testing uninstall function" 
test_function_exists "unstructured_io::uninstall" "Uninstall function"

echo "DEBUG: All tests completed"
