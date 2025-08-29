#!/usr/bin/env bash
# Cline Test Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLINE_LIB_DIR="${APP_ROOT}/resources/cline/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CLINE_LIB_DIR}/common.sh"

#######################################
# Smoke test - Basic health check for Cline
# Returns:
#   0 if healthy, 1 otherwise
#######################################
cline::test::smoke() {
    log::header "🚀 Cline Smoke Test"
    
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
    if cline::check_vscode; then
        ((checks_passed++))
        log::success "✓ VS Code is available"
    elif [[ -f "$CLINE_CONFIG_DIR/.status" ]]; then
        ((checks_passed++))
        log::success "✓ Configuration ready (VS Code not required)"
    else
        log::error "✗ No VS Code and no configuration"
    fi
    
    # Check 3: Provider configuration
    local provider=$(cline::get_provider)
    if [[ "$provider" == "ollama" ]]; then
        if curl -s http://localhost:11434/api/version >/dev/null 2>&1; then
            ((checks_passed++))
            log::success "✓ Ollama provider is accessible"
        else
            log::warn "✗ Ollama provider not accessible"
        fi
    elif [[ -n "$(cline::get_api_key "$provider")" ]]; then
        ((checks_passed++))
        log::success "✓ API key configured for $provider"
    else
        log::warn "✗ No API key for $provider"
    fi
    
    log::info "Smoke test: $checks_passed/$checks_total checks passed"
    
    # Consider healthy if at least 2 out of 3 checks pass
    if [[ $checks_passed -ge 2 ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Integration test - Test integration with VS Code and API providers
# Returns:
#   0 if integration works, 1 otherwise
#######################################
cline::test::integration() {
    log::header "🔗 Cline Integration Test"
    
    local integration_passed=0
    local integration_total=3
    
    # Test 1: VS Code integration
    if cline::check_vscode; then
        if cline::is_installed; then
            ((integration_passed++))
            local version=$(cline::get_version)
            log::success "✓ Cline extension installed (v$version)"
        else
            log::warn "✗ Cline extension not installed"
        fi
    else
        log::warn "✗ VS Code not available for integration test"
    fi
    
    # Test 2: Provider integration
    local provider=$(cline::get_provider)
    if [[ "$provider" == "ollama" ]]; then
        if curl -s http://localhost:11434/api/version >/dev/null 2>&1; then
            ((integration_passed++))
            log::success "✓ Ollama integration working"
        else
            log::error "✗ Cannot connect to Ollama"
        fi
    else
        local api_key=$(cline::get_api_key "$provider")
        if [[ -n "$api_key" ]]; then
            ((integration_passed++))
            log::success "✓ API key configured for $provider"
        else
            log::error "✗ No API key for $provider"
        fi
    fi
    
    # Test 3: Configuration file integrity
    if [[ -f "$CLINE_SETTINGS_FILE" ]]; then
        if jq empty "$CLINE_SETTINGS_FILE" 2>/dev/null; then
            ((integration_passed++))
            log::success "✓ Settings file is valid JSON"
        else
            log::error "✗ Settings file is invalid JSON"
        fi
    else
        log::warn "✗ No settings file found"
    fi
    
    log::info "Integration test: $integration_passed/$integration_total tests passed"
    
    # Consider successful if at least 2 out of 3 tests pass
    if [[ $integration_passed -ge 2 ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# All tests - Run smoke and integration tests
# Returns:
#   0 if all pass, 1 otherwise
#######################################
cline::test::all() {
    log::header "🧪 Cline Test Suite"
    
    local smoke_result=0
    local integration_result=0
    
    # Run smoke test
    if ! cline::test::smoke; then
        smoke_result=1
    fi
    
    echo ""
    
    # Run integration test
    if ! cline::test::integration; then
        integration_result=1
    fi
    
    echo ""
    
    # Summary
    if [[ $smoke_result -eq 0 && $integration_result -eq 0 ]]; then
        log::success "✅ All tests passed!"
        return 0
    else
        log::error "❌ Some tests failed"
        [[ $smoke_result -ne 0 ]] && log::error "  - Smoke test failed"
        [[ $integration_result -ne 0 ]] && log::error "  - Integration test failed"
        return 1
    fi
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-smoke}" in
        smoke)
            cline::test::smoke
            ;;
        integration)
            cline::test::integration
            ;;
        all)
            cline::test::all
            ;;
        *)
            echo "Usage: $0 [smoke|integration|all]"
            exit 1
            ;;
    esac
fi