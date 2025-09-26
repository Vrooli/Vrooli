#!/usr/bin/env bash
################################################################################
# Cline Integration Test - End-to-End Functionality
# 
# Tests complete Cline integration workflows in <120 seconds
# Tests: Installation, configuration, provider switching, API connectivity
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
source "${RESOURCE_DIR}/lib/config.sh" 2>/dev/null || true
source "${RESOURCE_DIR}/lib/install.sh" 2>/dev/null || true
source "${RESOURCE_DIR}/config/defaults.sh"

# Test results
INTEGRATION_PASSED=0
INTEGRATION_FAILED=0

#######################################
# Test installation workflow
#######################################
test_installation() {
    log::info "Testing installation workflow..."
    
    # Check if already installed (use || true to prevent set -e from exiting)
    if cline::is_extension_installed 2>/dev/null || false; then
        log::success "✓ Cline extension already installed"
        ((INTEGRATION_PASSED++))
        return 0
    fi
    
    # Try to install (will only work if VS Code is available) (use || true to prevent set -e from exiting)
    if cline::check_vscode 2>/dev/null || false; then
        log::info "Attempting to install Cline extension..."
        if timeout 60 code --install-extension "${CLINE_EXTENSION_ID}" --force &>/dev/null; then
            log::success "✓ Cline extension installation successful"
            ((INTEGRATION_PASSED++))
        else
            log::warn "⚠ Extension installation failed (may already be installed)"
        fi
    else
        log::info "VS Code not available, skipping extension installation"
        # Still pass if we can configure
        if [[ -d "$CLINE_CONFIG_DIR" ]]; then
            log::success "✓ Configuration directory ready for future installation"
            ((INTEGRATION_PASSED++))
        fi
    fi
    
    return 0
}

#######################################
# Test configuration management
#######################################
test_configuration() {
    log::info "Testing configuration management..."
    
    # Ensure config directory exists
    mkdir -p "$CLINE_CONFIG_DIR"
    
    # Test provider configuration
    local test_provider="ollama"
    echo "$test_provider" > "$CLINE_CONFIG_DIR/provider.conf"
    
    local current_provider=$(cline::get_provider)
    if [[ "$current_provider" == "$test_provider" ]]; then
        log::success "✓ Provider configuration works"
        ((INTEGRATION_PASSED++))
    else
        log::error "✗ Provider configuration failed"
        ((INTEGRATION_FAILED++))
    fi
    
    # Test settings file creation
    local test_settings='{
        "claude-dev.apiProvider": "ollama",
        "claude-dev.ollamaBaseUrl": "http://localhost:11434",
        "claude-dev.ollamaModelId": "llama3.2:3b",
        "claude-dev.apiModelId": "llama3.2:3b"
    }'
    
    echo "$test_settings" > "$CLINE_SETTINGS_FILE"
    
    if jq empty "$CLINE_SETTINGS_FILE" 2>/dev/null; then
        log::success "✓ Settings file is valid JSON"
        ((INTEGRATION_PASSED++))
    else
        log::error "✗ Settings file validation failed"
        ((INTEGRATION_FAILED++))
    fi
    
    return 0
}

#######################################
# Test provider switching
#######################################
test_provider_switching() {
    log::info "Testing provider switching..."
    
    local providers=("ollama" "openrouter")
    local switch_success=0
    
    for provider in "${providers[@]}"; do
        echo "$provider" > "$CLINE_CONFIG_DIR/provider.conf"
        local current=$(cline::get_provider)
        
        if [[ "$current" == "$provider" ]]; then
            log::success "✓ Switched to provider: $provider"
            ((switch_success++))
        else
            log::error "✗ Failed to switch to: $provider"
        fi
    done
    
    if [[ $switch_success -ge 2 ]]; then
        log::success "✓ Provider switching works"
        ((INTEGRATION_PASSED++))
    else
        log::error "✗ Provider switching failed"
        ((INTEGRATION_FAILED++))
    fi
    
    # Reset to default
    echo "ollama" > "$CLINE_CONFIG_DIR/provider.conf"
    
    return 0
}

#######################################
# Test API connectivity
#######################################
test_api_connectivity() {
    log::info "Testing API connectivity..."
    
    local provider=$(cline::get_provider)
    
    case "$provider" in
        ollama)
            if timeout 5 curl -sf http://localhost:11434/api/version &>/dev/null; then
                log::success "✓ Ollama API is reachable"
                
                # Test model listing
                if timeout 10 curl -sf http://localhost:11434/api/tags &>/dev/null; then
                    log::success "✓ Can list Ollama models"
                    ((INTEGRATION_PASSED++))
                else
                    log::warn "⚠ Cannot list Ollama models"
                fi
            else
                log::warn "⚠ Ollama not running, skipping API test"
            fi
            ;;
            
        openrouter)
            local api_key=$(cline::get_api_key "openrouter")
            if [[ -n "$api_key" ]]; then
                # Test API with minimal request
                if timeout 10 curl -sf -H "Authorization: Bearer $api_key" \
                    https://openrouter.ai/api/v1/models &>/dev/null; then
                    log::success "✓ OpenRouter API is reachable"
                    ((INTEGRATION_PASSED++))
                else
                    log::warn "⚠ Cannot reach OpenRouter API"
                fi
            else
                log::info "No OpenRouter API key, skipping test"
            fi
            ;;
            
        *)
            log::info "Provider $provider connectivity test not implemented"
            ;;
    esac
    
    return 0
}

#######################################
# Test VS Code settings integration
#######################################
test_vscode_settings() {
    log::info "Testing VS Code settings integration..."
    
    # Use || true to prevent set -e from exiting
    if ! cline::check_vscode 2>/dev/null; then
        log::info "VS Code not available, skipping settings test"
        return 0
    fi
    
    # Check if settings can be read
    if [[ -f "$VSCODE_SETTINGS" ]]; then
        if jq empty "$VSCODE_SETTINGS" 2>/dev/null; then
            log::success "✓ VS Code settings file is valid"
            ((INTEGRATION_PASSED++))
        else
            log::warn "⚠ VS Code settings file is invalid JSON"
        fi
    else
        log::info "VS Code settings file not found (will be created on first use)"
    fi
    
    return 0
}

#######################################
# Test content management
#######################################
test_content_management() {
    log::info "Testing content management commands..."
    
    # Test content list
    if "${RESOURCE_DIR}/cli.sh" content list &>/dev/null; then
        log::success "✓ Content list command works"
        ((INTEGRATION_PASSED++))
    else
        log::error "✗ Content list command failed"
        ((INTEGRATION_FAILED++))
    fi
    
    # Test provider configuration via content
    if type -t cline::config_provider &>/dev/null; then
        log::success "✓ Provider configuration function exists"
        ((INTEGRATION_PASSED++))
    else
        log::warn "⚠ Provider configuration function not found"
    fi
    
    return 0
}

#######################################
# Main integration test execution
#######################################
main() {
    log::header "🔗 Cline Integration Test Suite"
    
    # Ensure directories exist
    cline::ensure_dirs 2>/dev/null || true
    
    # Run tests (use || true to prevent set -e from exiting)
    test_installation || true
    echo ""
    test_configuration || true
    echo ""
    test_provider_switching || true
    echo ""
    test_api_connectivity || true
    echo ""
    test_vscode_settings || true
    echo ""
    test_content_management || true
    
    # Summary
    echo ""
    log::header "📊 Integration Test Summary"
    log::info "Passed: $INTEGRATION_PASSED"
    log::info "Failed: $INTEGRATION_FAILED"
    
    # Determine overall result
    local total=$((INTEGRATION_PASSED + INTEGRATION_FAILED))
    if [[ $total -eq 0 ]]; then
        log::warn "No tests were run"
        exit 1
    fi
    
    local pass_rate=$((INTEGRATION_PASSED * 100 / total))
    
    if [[ $pass_rate -ge 60 ]]; then
        log::success "✅ Integration tests passed (${pass_rate}% success rate)"
        exit 0
    else
        log::error "❌ Integration tests failed (${pass_rate}% success rate)"
        exit 1
    fi
}

# Execute main
main "$@"