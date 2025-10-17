#!/usr/bin/env bash
# OpenEMR Unit Tests
# Library function validation (<60s)

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source dependencies
source "${RESOURCE_DIR}/../../scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

# Run unit tests
main() {
    log::info "OpenEMR Unit Tests"
    log::info "=================="
    
    local failed=0
    local total=5
    
    # Test 1: Configuration loading
    log::info "[1/$total] Testing configuration..."
    if [[ -n "${OPENEMR_PORT:-}" ]] && \
       [[ -n "${OPENEMR_DB_NAME:-}" ]] && \
       [[ -n "${OPENEMR_ADMIN_USER:-}" ]]; then
        log::success "✓ Configuration loaded"
    else
        log::error "✗ Configuration incomplete"
        ((failed++))
    fi
    
    # Test 2: Port validation
    log::info "[2/$total] Testing port configuration..."
    if [[ "$OPENEMR_PORT" -ge 1024 ]] && [[ "$OPENEMR_PORT" -le 65535 ]]; then
        log::success "✓ Port configuration valid"
    else
        log::error "✗ Invalid port configuration"
        ((failed++))
    fi
    
    # Test 3: Directory structure
    log::info "[3/$total] Testing directory structure..."
    local dirs_ok=true
    for dir in "$OPENEMR_DATA_DIR" "$OPENEMR_CONFIG_DIR" "$OPENEMR_LOGS_DIR"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir" 2>/dev/null || dirs_ok=false
        fi
    done
    
    if [[ "$dirs_ok" == "true" ]]; then
        log::success "✓ Directory structure ready"
    else
        log::error "✗ Cannot create directories"
        ((failed++))
    fi
    
    # Test 4: Docker availability
    log::info "[4/$total] Testing Docker..."
    if command -v docker &>/dev/null; then
        if docker ps &>/dev/null; then
            log::success "✓ Docker is available"
        else
            log::error "✗ Docker daemon not accessible"
            ((failed++))
        fi
    else
        log::error "✗ Docker not installed"
        ((failed++))
    fi
    
    # Test 5: Compose file generation
    log::info "[5/$total] Testing compose generation..."
    if openemr::generate_compose; then
        if [[ -f "${OPENEMR_CONFIG_DIR}/docker-compose.yml" ]]; then
            log::success "✓ Compose file generated"
        else
            log::error "✗ Compose file not created"
            ((failed++))
        fi
    else
        log::error "✗ Compose generation failed"
        ((failed++))
    fi
    
    # Summary
    echo ""
    if [[ $failed -eq 0 ]]; then
        log::success "PASSED: All unit tests passed"
        return 0
    else
        log::error "FAILED: $failed/$total tests failed"
        return 1
    fi
}

# Run tests
main "$@"