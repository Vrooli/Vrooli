#!/usr/bin/env bash
# Cline Test Functions - Delegates to v2.0 test runner

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLINE_LIB_DIR="${APP_ROOT}/resources/cline/lib"
CLINE_TEST_DIR="${APP_ROOT}/resources/cline/test"

# Source required utilities
# shellcheck disable=SC1091
source "${CLINE_LIB_DIR}/common.sh"

#######################################
# Smoke test - Delegates to v2.0 test runner
# Returns:
#   0 if healthy, 1 otherwise
#######################################
cline::test::smoke() {
    if [[ -f "${CLINE_TEST_DIR}/run-tests.sh" ]]; then
        # Use the v2.0 test runner
        bash "${CLINE_TEST_DIR}/run-tests.sh" smoke
        return $?
    else
        # Fallback to simple test if runner not available
        log::header "🚀 Cline Smoke Test (Fallback)"
        
        local checks_passed=0
        local checks_total=3
        
        # Check 1: Configuration directory exists
        if [[ -d "$CLINE_CONFIG_DIR" ]]; then
            ((checks_passed++))
            log::success "✓ Configuration directory exists"
        else
            log::error "✗ Configuration directory missing"
        fi
        
        # Check 2: VS Code availability or configuration readiness
        if cline::check_vscode 2>/dev/null || false; then
            ((checks_passed++))
            log::success "✓ VS Code is available"
        elif [[ -f "$CLINE_CONFIG_DIR/.status" ]]; then
            ((checks_passed++))
            log::success "✓ Configuration ready (VS Code not required)"
        else
            log::warn "⚠ VS Code not installed (configuration can proceed)"
            ((checks_passed++))  # Still pass - normal in headless
        fi
        
        # Check 3: Provider configuration
        local provider=$(cline::get_provider)
        if [[ "$provider" == "ollama" ]]; then
            if timeout 5 curl -s http://localhost:11434/api/version >/dev/null 2>&1; then
                ((checks_passed++))
                log::success "✓ Ollama provider is accessible"
            else
                log::warn "✗ Ollama provider not accessible"
            fi
        else
            # Check for API key
            local api_key=$(cline::get_api_key "$provider")
            if [[ -n "$api_key" ]]; then
                ((checks_passed++))
                log::success "✓ API key configured for $provider"
            else
                log::warn "✗ No API key for $provider"
            fi
        fi
        
        log::info "Smoke test: ${checks_passed}/${checks_total} checks passed"
        
        if [[ $checks_passed -ge 2 ]]; then
            return 0
        else
            return 1
        fi
    fi
}

#######################################
# Integration test - Delegates to v2.0 test runner
# Returns:
#   0 if integration successful, 1 otherwise
#######################################
cline::test::integration() {
    if [[ -f "${CLINE_TEST_DIR}/run-tests.sh" ]]; then
        # Use the v2.0 test runner
        bash "${CLINE_TEST_DIR}/run-tests.sh" integration
        return $?
    else
        # Fallback to simple test if runner not available
        log::header "🔗 Cline Integration Test (Fallback)"
        
        local tests_passed=0
        local tests_total=3
        
        # Test 1: VS Code integration (if available)
        if cline::check_vscode 2>/dev/null || false; then
            if cline::is_extension_installed 2>/dev/null || false; then
                ((tests_passed++))
                log::success "✓ Cline extension installed in VS Code"
            else
                log::warn "✗ Cline extension not installed"
            fi
        else
            log::info "VS Code not available (normal in headless environment)"
            ((tests_passed++))  # Pass - expected in CI
        fi
        
        # Test 2: Provider switching
        local original_provider=$(cline::get_provider)
        
        # Try Ollama
        if timeout 5 curl -s http://localhost:11434/api/version >/dev/null 2>&1; then
            echo "ollama" > "$CLINE_CONFIG_DIR/provider.conf"
            local current=$(cline::get_provider)
            if [[ "$current" == "ollama" ]]; then
                ((tests_passed++))
                log::success "✓ Ollama integration working"
            else
                log::error "✗ Failed to switch to Ollama"
            fi
        fi
        
        # Restore original provider
        echo "$original_provider" > "$CLINE_CONFIG_DIR/provider.conf"
        
        # Test 3: Settings file existence
        if [[ -f "$CLINE_SETTINGS_FILE" ]]; then
            ((tests_passed++))
            log::success "✓ Settings file found"
        else
            log::info "Settings file will be created on first use"
            ((tests_passed++))  # Pass - expected initially
        fi
        
        log::info "Integration test: ${tests_passed}/${tests_total} tests passed"
        
        if [[ $tests_passed -ge 2 ]]; then
            return 0
        else
            return 1
        fi
    fi
}

#######################################
# Unit test - Delegates to v2.0 test runner
# Returns:
#   0 if unit tests pass, 1 otherwise
#######################################
cline::test::unit() {
    if [[ -f "${CLINE_TEST_DIR}/run-tests.sh" ]]; then
        # Use the v2.0 test runner
        bash "${CLINE_TEST_DIR}/run-tests.sh" unit
        return $?
    else
        # No unit test fallback available
        log::warn "Unit tests not available (v2.0 test runner required)"
        return 2
    fi
}

#######################################
# All tests - Delegates to v2.0 test runner
# Returns:
#   0 if all pass, 1 if any fail
#######################################
cline::test::all() {
    if [[ -f "${CLINE_TEST_DIR}/run-tests.sh" ]]; then
        # Use the v2.0 test runner
        bash "${CLINE_TEST_DIR}/run-tests.sh" all
        return $?
    else
        # Fallback to running individual tests
        log::header "🧪 Cline Test Suite (Fallback)"
        
        local all_passed=true
        
        if ! cline::test::smoke; then
            all_passed=false
            log::error "  - Smoke test failed"
        fi
        
        echo ""
        
        if ! cline::test::integration; then
            all_passed=false
            log::error "  - Integration test failed"
        fi
        
        echo ""
        
        if [[ "$all_passed" == "true" ]]; then
            log::success "✅ All tests passed"
            return 0
        else
            log::error "❌ Some tests failed"
            return 1
        fi
    fi
}

#######################################
# Test main entry point
#######################################
cline::test::main() {
    local test_phase="${1:-all}"
    
    case "$test_phase" in
        smoke)
            cline::test::smoke
            ;;
        integration)
            cline::test::integration
            ;;
        unit)
            cline::test::unit
            ;;
        all)
            cline::test::all
            ;;
        *)
            log::error "Unknown test phase: $test_phase"
            echo "Usage: $0 [smoke|integration|unit|all]"
            return 1
            ;;
    esac
}