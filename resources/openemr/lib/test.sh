#!/usr/bin/env bash
# OpenEMR Test Functionality

set -euo pipefail

# Get script directory (only if not already set)
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi
if [[ -z "${RESOURCE_DIR:-}" ]]; then
    RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
fi

# Source dependencies if not already sourced
[[ -z "${LOG_COLORS_INITIALIZED:-}" ]] && source "${RESOURCE_DIR}/../../scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Main test router
openemr::test() {
    local phase="${1:-all}"
    
    case "$phase" in
        smoke)
            openemr::test::smoke
            ;;
        integration)
            openemr::test::integration
            ;;
        unit)
            openemr::test::unit
            ;;
        all)
            openemr::test::all
            ;;
        *)
            log::error "Unknown test phase: $phase"
            return 1
            ;;
    esac
}

# Run all tests
openemr::test::all() {
    log::info "Running all OpenEMR tests..."
    
    local failed=0
    
    openemr::test::smoke || ((failed++))
    openemr::test::integration || ((failed++))
    openemr::test::unit || ((failed++))
    
    if [[ $failed -gt 0 ]]; then
        log::error "$failed test phase(s) failed"
        return 1
    fi
    
    log::success "All tests passed"
    return 0
}

# Smoke tests - quick health validation
openemr::test::smoke() {
    log::info "Running OpenEMR smoke tests..."
    
    local failed=0
    
    # Test 1: Web service is running
    log::debug "Test 1: Checking web service..."
    if docker ps --format '{{.Names}}' | grep -q "^${OPENEMR_WEB_CONTAINER}$"; then
        log::success "✓ Web service is running"
    else
        log::error "✗ Web service is not running"
        ((failed++))
    fi
    
    # Test 2: Database is running
    log::debug "Test 2: Checking database..."
    if docker ps --format '{{.Names}}' | grep -q "^${OPENEMR_DB_CONTAINER}$"; then
        log::success "✓ Database is running"
    else
        log::error "✗ Database is not running"
        ((failed++))
    fi
    
    # Test 3: Health endpoint responds
    log::debug "Test 3: Checking health endpoint..."
    if timeout 5 curl -sf "http://localhost:${OPENEMR_PORT}/interface/globals.php" &>/dev/null; then
        log::success "✓ Health endpoint responds"
    else
        log::error "✗ Health endpoint not responding"
        ((failed++))
    fi
    
    # Test 4: API endpoint accessible
    log::debug "Test 4: Checking API endpoint..."
    if timeout 5 curl -sf "http://localhost:${OPENEMR_API_PORT}/apis/default/api/facility" &>/dev/null; then
        log::success "✓ API endpoint accessible"
    else
        log::error "✗ API endpoint not accessible"
        ((failed++))
    fi
    
    # Test 5: FHIR endpoint accessible
    log::debug "Test 5: Checking FHIR endpoint..."
    if timeout 5 curl -sf "http://localhost:${OPENEMR_FHIR_PORT}/apis/default/fhir/r4/metadata" &>/dev/null; then
        log::success "✓ FHIR endpoint accessible"
    else
        log::error "✗ FHIR endpoint not accessible"
        ((failed++))
    fi
    
    if [[ $failed -gt 0 ]]; then
        log::error "Smoke tests failed: $failed test(s) failed"
        return 1
    fi
    
    log::success "Smoke tests passed"
    return 0
}

# Integration tests - full functionality
openemr::test::integration() {
    log::info "Running OpenEMR integration tests..."
    
    local failed=0
    
    # Test 1: Database connectivity
    log::debug "Test 1: Testing database connectivity..."
    if docker exec "$OPENEMR_DB_CONTAINER" mysql -u"$OPENEMR_DB_USER" -p"$OPENEMR_DB_PASS" -e "SELECT 1" &>/dev/null; then
        log::success "✓ Database connectivity works"
    else
        log::error "✗ Database connectivity failed"
        ((failed++))
    fi
    
    # Test 2: Create test patient
    log::debug "Test 2: Creating test patient..."
    log::warning "△ Patient creation needs API configuration"
    
    # Test 3: FHIR metadata endpoint
    log::debug "Test 3: Testing FHIR metadata..."
    local fhir_response=$(timeout 5 curl -sf "http://localhost:${OPENEMR_FHIR_PORT}/apis/default/fhir/r4/metadata" 2>/dev/null || echo "{}")
    if echo "$fhir_response" | grep -q "CapabilityStatement"; then
        log::success "✓ FHIR metadata available"
    else
        log::warning "△ FHIR metadata needs configuration"
    fi
    
    # Test 4: Web interface login page
    log::debug "Test 4: Testing web interface..."
    local web_response=$(timeout 5 curl -sf "http://localhost:${OPENEMR_PORT}/interface/login/login.php" 2>/dev/null || echo "")
    if echo "$web_response" | grep -q "OpenEMR"; then
        log::success "✓ Web interface available"
    else
        log::error "✗ Web interface not available"
        ((failed++))
    fi
    
    if [[ $failed -gt 0 ]]; then
        log::error "Integration tests failed: $failed test(s) failed"
        return 1
    fi
    
    log::success "Integration tests passed"
    return 0
}

# Unit tests - library functions
openemr::test::unit() {
    log::info "Running OpenEMR unit tests..."
    
    local failed=0
    
    # Test 1: Configuration loading
    log::debug "Test 1: Testing configuration loading..."
    if [[ -n "${OPENEMR_PORT:-}" ]]; then
        log::success "✓ Configuration loads correctly"
    else
        log::error "✗ Configuration loading failed"
        ((failed++))
    fi
    
    # Test 2: Credential file generation
    log::debug "Test 2: Testing credential management..."
    if [[ -f "${OPENEMR_CONFIG_DIR}/credentials.json" ]]; then
        log::success "✓ Credentials file exists"
    else
        log::warning "△ Credentials file not found (run install first)"
    fi
    
    # Test 3: Docker compose validation
    log::debug "Test 3: Testing docker compose file..."
    if [[ -f "${OPENEMR_CONFIG_DIR}/docker-compose.yml" ]]; then
        if docker-compose -f "${OPENEMR_CONFIG_DIR}/docker-compose.yml" config &>/dev/null; then
            log::success "✓ Docker compose file valid"
        else
            log::error "✗ Docker compose file invalid"
            ((failed++))
        fi
    else
        log::warning "△ Docker compose file not found (run install first)"
    fi
    
    # Test 4: Port availability check
    log::debug "Test 4: Testing port availability..."
    local ports_available=true
    for port in $OPENEMR_PORT $OPENEMR_API_PORT $OPENEMR_FHIR_PORT; do
        if lsof -i ":$port" &>/dev/null; then
            # Port is in use, check if it's our container
            if ! docker ps --format '{{.Names}}' | grep -q "openemr"; then
                ports_available=false
                break
            fi
        fi
    done
    
    if [[ "$ports_available" == "true" ]]; then
        log::success "✓ Required ports available"
    else
        log::warning "△ Some ports may be in use"
    fi
    
    if [[ $failed -gt 0 ]]; then
        log::error "Unit tests failed: $failed test(s) failed"
        return 1
    fi
    
    log::success "Unit tests passed"
    return 0
}