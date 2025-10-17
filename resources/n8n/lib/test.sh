#!/usr/bin/env bash
################################################################################
# n8n Test Implementation - v2.0 Universal Contract Compliant
# 
# Handles resource validation testing (NOT business functionality testing)
# Tests the n8n SERVICE itself - health, connectivity, library functions
################################################################################

set -euo pipefail

# Get directory paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
N8N_DIR="${APP_ROOT}/resources/n8n"

# Source utilities and configuration
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${N8N_DIR}/config/defaults.sh"

# Source core functions for testing
source "${N8N_DIR}/lib/core.sh" 2>/dev/null || true
source "${N8N_DIR}/lib/health.sh" 2>/dev/null || true

################################################################################
# Test orchestration - delegates to test phases
################################################################################

n8n::test::all() {
    local exit_code=0
    
    log::info "Running all n8n test suites..."
    
    # Run each test phase
    if ! n8n::test::smoke; then
        exit_code=1
    fi
    
    if ! n8n::test::integration; then
        exit_code=1
    fi
    
    if ! n8n::test::unit; then
        exit_code=1
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        log::success "All n8n tests passed"
    else
        log::error "One or more n8n test suites failed"
    fi
    
    return $exit_code
}

n8n::test::smoke() {
    log::info "Running n8n smoke tests..."
    
    if [[ -f "${N8N_DIR}/test/phases/test-smoke.sh" ]]; then
        if bash "${N8N_DIR}/test/phases/test-smoke.sh"; then
            log::success "Smoke tests passed"
            return 0
        else
            log::error "Smoke tests failed"
            return 1
        fi
    else
        log::warn "Smoke test file not found: ${N8N_DIR}/test/phases/test-smoke.sh"
        return 2
    fi
}

n8n::test::integration() {
    log::info "Running n8n integration tests..."
    
    if [[ -f "${N8N_DIR}/test/phases/test-integration.sh" ]]; then
        if bash "${N8N_DIR}/test/phases/test-integration.sh"; then
            log::success "Integration tests passed"
            return 0
        else
            log::error "Integration tests failed"
            return 1
        fi
    else
        log::warn "Integration test file not found: ${N8N_DIR}/test/phases/test-integration.sh"
        return 2
    fi
}

n8n::test::unit() {
    log::info "Running n8n unit tests..."
    
    if [[ -f "${N8N_DIR}/test/phases/test-unit.sh" ]]; then
        if bash "${N8N_DIR}/test/phases/test-unit.sh"; then
            log::success "Unit tests passed"
            return 0
        else
            log::error "Unit tests failed"
            return 1
        fi
    else
        log::warn "Unit test file not found: ${N8N_DIR}/test/phases/test-unit.sh"
        return 2
    fi
}

################################################################################
# Export functions
################################################################################

export -f n8n::test::all
export -f n8n::test::smoke
export -f n8n::test::integration
export -f n8n::test::unit