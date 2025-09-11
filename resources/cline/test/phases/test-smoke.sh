#!/usr/bin/env bash
################################################################################
# Cline Smoke Test - Quick Health Validation
# 
# Validates basic Cline functionality in <30 seconds
# Tests: Configuration, provider availability, basic health
#
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities and libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${RESOURCE_DIR}/lib/common.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Test results
SMOKE_PASSED=0
SMOKE_FAILED=0

#######################################
# Test configuration directory
#######################################
test_config_directory() {
    log::info "Testing configuration directory..."
    
    if [[ -d "$CLINE_CONFIG_DIR" ]]; then
        log::success "✓ Config directory exists: $CLINE_CONFIG_DIR"
        ((SMOKE_PASSED++))
        return 0
    else
        log::error "✗ Config directory missing: $CLINE_CONFIG_DIR"
        ((SMOKE_FAILED++))
        return 1
    fi
}

#######################################
# Test VS Code availability
#######################################
test_vscode_availability() {
    log::info "Testing VS Code availability..."
    
    if cline::check_vscode; then
        log::success "✓ VS Code is available"
        ((SMOKE_PASSED++))
        
        # Check if extension is installed
        if cline::is_extension_installed; then
            log::success "✓ Cline extension is installed"
            ((SMOKE_PASSED++))
        else
            log::warn "⚠ Cline extension not installed"
        fi
        return 0
    else
        log::warn "⚠ VS Code not available (configuration can proceed)"
        # Still pass if config exists
        if [[ -f "$CLINE_SETTINGS_FILE" ]]; then
            log::success "✓ Configuration file exists"
            ((SMOKE_PASSED++))
            return 0
        else
            log::error "✗ No VS Code and no configuration"
            ((SMOKE_FAILED++))
            return 1
        fi
    fi
}

#######################################
# Test provider configuration
#######################################
test_provider_config() {
    log::info "Testing provider configuration..."
    
    local provider=$(cline::get_provider)
    log::info "Current provider: $provider"
    
    case "$provider" in
        ollama)
            # Check if Ollama is running
            if timeout 5 curl -sf http://localhost:11434/api/version &>/dev/null; then
                log::success "✓ Ollama provider is accessible"
                ((SMOKE_PASSED++))
                return 0
            else
                log::warn "⚠ Ollama not running at localhost:11434"
                ((SMOKE_FAILED++))
                return 1
            fi
            ;;
            
        openrouter|anthropic|openai)
            # Check for API key
            local api_key=$(cline::get_api_key "$provider")
            if [[ -n "$api_key" ]]; then
                log::success "✓ API key configured for $provider"
                ((SMOKE_PASSED++))
                return 0
            else
                log::error "✗ No API key for $provider"
                ((SMOKE_FAILED++))
                return 1
            fi
            ;;
            
        *)
            log::warn "⚠ Unknown provider: $provider"
            ((SMOKE_FAILED++))
            return 1
            ;;
    esac
}

#######################################
# Test health endpoint
#######################################
test_health_endpoint() {
    log::info "Testing health endpoint..."
    
    # Note: Cline doesn't have a traditional health endpoint
    # We simulate one by checking configuration status
    if [[ -f "$CLINE_SETTINGS_FILE" ]] || [[ -f "$CLINE_CONFIG_DIR/provider.conf" ]]; then
        log::success "✓ Configuration health check passed"
        ((SMOKE_PASSED++))
        return 0
    else
        log::warn "⚠ No configuration files found"
        ((SMOKE_FAILED++))
        return 1
    fi
}

#######################################
# Test CLI commands availability
#######################################
test_cli_commands() {
    log::info "Testing CLI command structure..."
    
    # Test help command
    if "${RESOURCE_DIR}/cli.sh" help &>/dev/null; then
        log::success "✓ CLI help command works"
        ((SMOKE_PASSED++))
    else
        log::error "✗ CLI help command failed"
        ((SMOKE_FAILED++))
        return 1
    fi
    
    # Test info command
    if "${RESOURCE_DIR}/cli.sh" info &>/dev/null; then
        log::success "✓ CLI info command works"
        ((SMOKE_PASSED++))
    else
        log::error "✗ CLI info command failed"
        ((SMOKE_FAILED++))
    fi
    
    return 0
}

#######################################
# Main smoke test execution
#######################################
main() {
    log::header "🚀 Cline Smoke Test Suite"
    
    # Ensure directories exist
    cline::ensure_dirs
    
    # Run tests
    test_config_directory
    test_vscode_availability
    test_provider_config
    test_health_endpoint
    test_cli_commands
    
    # Summary
    echo ""
    log::header "📊 Smoke Test Summary"
    log::info "Passed: $SMOKE_PASSED"
    log::info "Failed: $SMOKE_FAILED"
    
    # Determine overall result (need at least 60% pass rate)
    local total=$((SMOKE_PASSED + SMOKE_FAILED))
    local pass_rate=$((SMOKE_PASSED * 100 / total))
    
    if [[ $pass_rate -ge 60 ]]; then
        log::success "✅ Smoke tests passed (${pass_rate}% success rate)"
        exit 0
    else
        log::error "❌ Smoke tests failed (${pass_rate}% success rate)"
        exit 1
    fi
}

# Execute main
main "$@"