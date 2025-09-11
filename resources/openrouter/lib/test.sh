#!/usr/bin/env bash
################################################################################
# OpenRouter Test Library - v2.0 Universal Contract Compliant
# 
# Test implementation functions for the OpenRouter resource
################################################################################

set -euo pipefail

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_LIB_DIR="${APP_ROOT}/resources/openrouter/lib"
OPENROUTER_RESOURCE_DIR="${APP_ROOT}/resources/openrouter"
OPENROUTER_TEST_DIR="${OPENROUTER_RESOURCE_DIR}/test"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${OPENROUTER_RESOURCE_DIR}/config/defaults.sh"
source "${OPENROUTER_LIB_DIR}/core.sh"

# Test all phases
openrouter::test::all() {
    log::info "Running all OpenRouter tests..."
    
    local test_runner="${OPENROUTER_TEST_DIR}/run-tests.sh"
    
    if [[ ! -f "$test_runner" ]]; then
        log::error "Test runner not found at $test_runner"
        return 1
    fi
    
    # Make executable and run
    chmod +x "$test_runner"
    "$test_runner" all
}

# Smoke test - quick health check
openrouter::test::smoke() {
    log::info "Running OpenRouter smoke test..."
    
    local test_script="${OPENROUTER_TEST_DIR}/phases/test-smoke.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::error "Smoke test script not found at $test_script"
        return 1
    fi
    
    # Make executable and run
    chmod +x "$test_script"
    "$test_script"
}

# Integration test
openrouter::test::integration() {
    log::info "Running OpenRouter integration tests..."
    
    local test_script="${OPENROUTER_TEST_DIR}/phases/test-integration.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::error "Integration test script not found at $test_script"
        return 1
    fi
    
    # Make executable and run
    chmod +x "$test_script"
    "$test_script"
}

# Unit test
openrouter::test::unit() {
    log::info "Running OpenRouter unit tests..."
    
    local test_script="${OPENROUTER_TEST_DIR}/phases/test-unit.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::error "Unit test script not found at $test_script"
        return 1
    fi
    
    # Make executable and run
    chmod +x "$test_script"
    "$test_script"
}

# Test API connectivity specifically
openrouter::test::api() {
    log::info "Testing OpenRouter API connectivity..."
    
    # Initialize API key if not set
    if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
        if ! openrouter::init; then
            log::error "Failed to initialize OpenRouter API key"
            return 1
        fi
    fi
    
    # Test connection with timeout
    if openrouter::test_connection "$OPENROUTER_HEALTH_CHECK_TIMEOUT" "$OPENROUTER_HEALTH_CHECK_MODEL"; then
        log::success "API connectivity test passed"
        return 0
    else
        log::error "API connectivity test failed"
        return 1
    fi
}

# Test model listing
openrouter::test::models() {
    log::info "Testing OpenRouter model listing..."
    
    # Initialize API key if not set
    if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
        if ! openrouter::init; then
            log::error "Failed to initialize OpenRouter API key"
            return 1
        fi
    fi
    
    # List models
    local models
    if models=$(openrouter::list_models "$OPENROUTER_TIMEOUT"); then
        local model_count
        model_count=$(echo "$models" | jq -r '.data | length' 2>/dev/null || echo "0")
        
        if [[ "$model_count" -gt 0 ]]; then
            log::success "Found $model_count available models"
            return 0
        else
            log::error "No models found"
            return 1
        fi
    else
        log::error "Failed to list models"
        return 1
    fi
}

# Validate configuration
openrouter::test::config() {
    log::info "Validating OpenRouter configuration..."
    
    local errors=0
    
    # Check required configuration
    if [[ -z "${OPENROUTER_API_BASE:-}" ]]; then
        log::error "OPENROUTER_API_BASE not set"
        ((errors++))
    fi
    
    if [[ -z "${OPENROUTER_DEFAULT_MODEL:-}" ]]; then
        log::error "OPENROUTER_DEFAULT_MODEL not set"
        ((errors++))
    fi
    
    if [[ -z "${OPENROUTER_TIMEOUT:-}" ]]; then
        log::error "OPENROUTER_TIMEOUT not set"
        ((errors++))
    fi
    
    # Check configuration files
    if [[ ! -f "${OPENROUTER_RESOURCE_DIR}/config/defaults.sh" ]]; then
        log::error "defaults.sh not found"
        ((errors++))
    fi
    
    if [[ ! -f "${OPENROUTER_RESOURCE_DIR}/config/runtime.json" ]]; then
        log::error "runtime.json not found"
        ((errors++))
    fi
    
    if [[ ! -f "${OPENROUTER_RESOURCE_DIR}/config/secrets.yaml" ]]; then
        log::error "secrets.yaml not found"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        log::success "Configuration validation passed"
        return 0
    else
        log::error "Configuration validation failed with $errors errors"
        return 1
    fi
}

# Wrapper for v2.0 contract compliance
openrouter::test() {
    local phase="${1:-all}"
    
    case "$phase" in
        all)
            openrouter::test::all
            ;;
        smoke)
            openrouter::test::smoke
            ;;
        integration)
            openrouter::test::integration
            ;;
        unit)
            openrouter::test::unit
            ;;
        api)
            openrouter::test::api
            ;;
        models)
            openrouter::test::models
            ;;
        config)
            openrouter::test::config
            ;;
        *)
            log::error "Unknown test phase: $phase"
            echo "Available phases: all, smoke, integration, unit, api, models, config"
            return 1
            ;;
    esac
}