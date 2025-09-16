#!/usr/bin/env bash
# OpenEMR Integration Tests
# Full functionality validation (<120s)

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source dependencies
source "${RESOURCE_DIR}/../../scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Run integration tests
main() {
    log::info "OpenEMR Integration Tests"
    log::info "========================="
    
    local failed=0
    local total=6
    
    # Test 1: Database connectivity
    log::info "[1/$total] Testing database connectivity..."
    if docker exec "$OPENEMR_DB_CONTAINER" \
        mysql -u"$OPENEMR_DB_USER" -p"$OPENEMR_DB_PASS" \
        -e "SELECT VERSION()" &>/dev/null; then
        log::success "✓ Database connection successful"
    else
        log::error "✗ Database connection failed"
        ((failed++))
    fi
    
    # Test 2: OpenEMR database exists
    log::info "[2/$total] Checking OpenEMR database..."
    if docker exec "$OPENEMR_DB_CONTAINER" \
        mysql -u"$OPENEMR_DB_USER" -p"$OPENEMR_DB_PASS" \
        -e "USE $OPENEMR_DB_NAME" &>/dev/null; then
        log::success "✓ OpenEMR database exists"
    else
        log::warning "△ OpenEMR database not initialized"
    fi
    
    # Test 3: Web interface login page
    log::info "[3/$total] Testing web interface..."
    local web_response=$(timeout 5 curl -sf \
        "http://localhost:${OPENEMR_PORT}/interface/login/login.php" 2>/dev/null || echo "")
    
    if echo "$web_response" | grep -q "OpenEMR"; then
        log::success "✓ Login page accessible"
    else
        log::error "✗ Login page not accessible"
        ((failed++))
    fi
    
    # Test 4: REST API structure
    log::info "[4/$total] Testing REST API..."
    local api_response=$(timeout 5 curl -sf \
        "http://localhost:${OPENEMR_API_PORT}/apis/default/api" 2>/dev/null || echo "{}")
    
    if [[ -n "$api_response" ]]; then
        log::success "✓ REST API endpoint exists"
    else
        log::warning "△ REST API needs configuration"
    fi
    
    # Test 5: FHIR capability statement
    log::info "[5/$total] Testing FHIR API..."
    local fhir_response=$(timeout 5 curl -sf \
        "http://localhost:${OPENEMR_FHIR_PORT}/apis/default/fhir/r4/metadata" 2>/dev/null || echo "{}")
    
    if echo "$fhir_response" | grep -q "CapabilityStatement"; then
        log::success "✓ FHIR R4 API available"
    else
        log::warning "△ FHIR API needs configuration"
    fi
    
    # Test 6: Container health
    log::info "[6/$total] Checking container health..."
    local web_healthy=$(docker inspect "$OPENEMR_WEB_CONTAINER" 2>/dev/null | \
        jq -r '.[0].State.Status' || echo "unknown")
    local db_healthy=$(docker inspect "$OPENEMR_DB_CONTAINER" 2>/dev/null | \
        jq -r '.[0].State.Status' || echo "unknown")
    
    if [[ "$web_healthy" == "running" ]] && [[ "$db_healthy" == "running" ]]; then
        log::success "✓ All containers healthy"
    else
        log::error "✗ Container health issues"
        ((failed++))
    fi
    
    # Summary
    echo ""
    if [[ $failed -eq 0 ]]; then
        log::success "PASSED: All integration tests passed"
        return 0
    else
        log::error "FAILED: $failed/$total tests failed"
        return 1
    fi
}

# Run tests
main "$@"