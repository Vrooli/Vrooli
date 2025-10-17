#!/usr/bin/env bash
# OpenEMR Smoke Tests
# Quick health validation (<30s)

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source dependencies
source "${RESOURCE_DIR}/../../scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Run smoke tests
main() {
    log::info "OpenEMR Smoke Tests"
    log::info "==================="
    
    local failed=0
    local total=5
    
    # Test 1: Container status
    log::info "[1/$total] Checking container status..."
    if docker ps --format '{{.Names}}' | grep -q "openemr-web"; then
        log::success "✓ OpenEMR web container running"
        if docker ps --format '{{.Names}}' | grep -q "openemr-mysql"; then
            log::success "✓ OpenEMR database container running"
        else
            log::error "✗ OpenEMR database container not found"
            ((failed++))
        fi
    else
        log::error "✗ OpenEMR web container not found"
        ((failed++))
    fi
    
    # Test 2: Network connectivity
    log::info "[2/$total] Checking network..."
    if docker network ls | grep -q "openemr-network"; then
        log::success "✓ Docker network exists"
    else
        log::error "✗ Docker network missing"
        ((failed++))
    fi
    
    # Test 3: Web health check
    log::info "[3/$total] Checking web health..."
    if timeout 5 curl -sf "http://localhost:${OPENEMR_PORT}/interface/login/login.php?site=default" &>/dev/null; then
        log::success "✓ Web interface responding"
    else
        log::error "✗ Web interface not responding on port ${OPENEMR_PORT}"
        ((failed++))
    fi
    
    # Test 4: API health check
    log::info "[4/$total] Checking API health..."
    if timeout 5 curl -sf "http://localhost:${OPENEMR_PORT}/apis/default/auth" &>/dev/null; then
        log::success "✓ API endpoint responding"
    elif timeout 5 curl -sf "http://localhost:${OPENEMR_PORT}/apis" &>/dev/null; then
        log::warning "△ API endpoint partially available"
    else
        log::warning "△ API endpoint not fully configured"
    fi
    
    # Test 5: Response time
    log::info "[5/$total] Checking response time..."
    local start_time=$(date +%s%N)
    timeout 5 curl -sf "http://localhost:${OPENEMR_PORT}/interface/login/login.php?site=default" &>/dev/null
    local end_time=$(date +%s%N)
    local response_time=$(( (end_time - start_time) / 1000000 ))
    
    if [[ $response_time -lt 1000 ]]; then
        log::success "✓ Response time: ${response_time}ms"
    else
        log::warning "△ Response time: ${response_time}ms (slow)"
    fi
    
    # Summary
    echo ""
    if [[ $failed -eq 0 ]]; then
        log::success "PASSED: All smoke tests passed"
        return 0
    else
        log::error "FAILED: $failed/$total tests failed"
        return 1
    fi
}

# Run tests
main "$@"