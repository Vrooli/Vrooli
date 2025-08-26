#!/usr/bin/env bash
# Codex Test Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_TEST_DIR="${APP_ROOT}/resources/codex/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CODEX_TEST_DIR}/common.sh"

#######################################
# Smoke test - Basic health check for Codex service
# Returns:
#   0 if healthy, 1 if unhealthy
#######################################
codex::test::smoke() {
    log::header "Codex Smoke Test"
    
    local test_passed=0
    
    # Test 1: Configuration check
    echo "1. Checking configuration..."
    if codex::is_configured; then
        log::success "✓ API configuration found"
    else
        log::error "✗ API not configured"
        test_passed=1
    fi
    
    # Test 2: API connectivity check
    echo "2. Checking API connectivity..."
    if codex::is_available; then
        log::success "✓ API is accessible"
    else
        log::error "✗ API not accessible"
        test_passed=1
    fi
    
    # Test 3: Directory structure check
    echo "3. Checking directory structure..."
    local dirs_ok=true
    for dir in "${CODEX_SCRIPTS_DIR}" "${CODEX_OUTPUT_DIR}" "${CODEX_INJECTED_DIR}"; do
        if [[ -d "${dir}" ]]; then
            log::success "✓ Directory exists: ${dir}"
        else
            log::warn "⚠ Directory missing: ${dir}"
            # Try to create it
            if mkdir -p "${dir}" 2>/dev/null; then
                log::success "✓ Created directory: ${dir}"
            else
                log::error "✗ Cannot create directory: ${dir}"
                dirs_ok=false
            fi
        fi
    done
    
    if [[ "$dirs_ok" == "false" ]]; then
        test_passed=1
    fi
    
    # Test 4: Model info check
    echo "4. Checking model configuration..."
    local version
    version=$(codex::get_version)
    if [[ -n "${version}" ]]; then
        log::success "✓ Model configured: ${version}"
    else
        log::warn "⚠ No model configured"
    fi
    
    # Summary
    echo ""
    if [[ $test_passed -eq 0 ]]; then
        log::success "All smoke tests passed - Codex service is healthy"
        return 0
    else
        log::error "Smoke tests failed - Codex service is unhealthy"
        return 1
    fi
}

#######################################
# Integration test - Full functionality test
# Returns:
#   0 if all tests pass, 1 if any fail
#######################################
codex::test::integration() {
    log::header "Codex Integration Test"
    
    # First run smoke test
    if ! codex::test::smoke; then
        log::error "Smoke test failed - skipping integration test"
        return 1
    fi
    
    echo ""
    log::info "Running integration tests..."
    
    local test_passed=0
    
    # Test script injection (if inject function exists)
    if type -t codex::inject &>/dev/null; then
        echo "5. Testing script injection..."
        
        # Create a test script
        local test_script="/tmp/codex_test_script.py"
        cat > "${test_script}" <<EOF
# Test script for Codex
print("Hello from Codex test!")

def test_function():
    return "This is a test function"
EOF
        
        if codex::inject "${test_script}"; then
            log::success "✓ Script injection test passed"
        else
            log::error "✗ Script injection test failed"
            test_passed=1
        fi
        
        # Clean up
        rm -f "${test_script}"
    else
        log::warn "⚠ Script injection function not available"
    fi
    
    # Test script listing
    echo "6. Testing script listing..."
    local script_count
    if [[ -d "${CODEX_SCRIPTS_DIR}" ]]; then
        script_count=$(find "${CODEX_SCRIPTS_DIR}" -name "*.py" 2>/dev/null | wc -l)
        log::success "✓ Found ${script_count} scripts in directory"
    else
        log::warn "⚠ Scripts directory not found"
    fi
    
    # Test status reporting
    echo "7. Testing status reporting..."
    if codex::status --fast > /dev/null 2>&1; then
        log::success "✓ Status reporting works"
    else
        log::error "✗ Status reporting failed"
        test_passed=1
    fi
    
    # Summary
    echo ""
    if [[ $test_passed -eq 0 ]]; then
        log::success "All integration tests passed"
        return 0
    else
        log::error "Some integration tests failed"
        return 1
    fi
}

#######################################
# Run all tests
# Returns:
#   0 if all tests pass, 1 if any fail
#######################################
codex::test::all() {
    log::header "Codex All Tests"
    
    local overall_result=0
    
    # Run smoke test
    if ! codex::test::smoke; then
        overall_result=1
    fi
    
    echo ""
    
    # Run integration test
    if ! codex::test::integration; then
        overall_result=1
    fi
    
    echo ""
    if [[ $overall_result -eq 0 ]]; then
        log::success "All tests passed - Codex service is fully functional"
    else
        log::error "Some tests failed - Codex service has issues"
    fi
    
    return $overall_result
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "$1" in
        smoke)
            codex::test::smoke
            ;;
        integration)
            codex::test::integration
            ;;
        all)
            codex::test::all
            ;;
        *)
            echo "Usage: $0 {smoke|integration|all}"
            exit 1
            ;;
    esac
fi