#!/usr/bin/env bash
# OpenMRS Smoke Tests

set -euo pipefail

# Get paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source libraries
source "${RESOURCE_DIR}/../../scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/lib/core.sh"

log::info "Running OpenMRS smoke tests..."

# Test results
FAILED=0
PASSED=0

# Test 1: Container health check
echo -n "1. Checking OpenMRS container... "
if docker ps --format '{{.Names}}' | grep -q "openmrs-app"; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (container not running)"
    ((FAILED++))
fi

# Test 2: Web interface accessibility
echo -n "2. Checking web interface... "
if timeout 5 curl -sf "http://localhost:${OPENMRS_PORT}/openmrs" &>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (not accessible)"
    ((FAILED++))
fi

# Test 3: REST API health
echo -n "3. Checking REST API... "
if timeout 5 curl -sf "http://localhost:${OPENMRS_PORT}/openmrs/ws/rest/v1/session" &>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (API not responding)"
    ((FAILED++))
fi

# Test 4: Database connectivity
echo -n "4. Checking database... "
if docker ps --format '{{.Names}}' | grep -q "openmrs-mysql"; then
    if docker exec openmrs-mysql mysqladmin -u root -p"$(cat /home/matthalloran8/.vrooli/openmrs/config/db_root_password.txt 2>/dev/null)" ping 2>&1 | grep -q "alive"; then
        echo "✓"
        ((PASSED++))
    else
        echo "✗ (database not ready)"
        ((FAILED++))
    fi
else
    echo "✗ (database container not running)"
    ((FAILED++))
fi

# Test 5: Admin authentication
echo -n "5. Checking authentication... "
if curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
    "http://localhost:${OPENMRS_PORT}/openmrs/ws/rest/v1/session" | \
    grep -q "sessionId"; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (authentication failed)"
    ((FAILED++))
fi

# Summary
echo ""
echo "Smoke Test Results:"
echo "  Passed: $PASSED"
echo "  Failed: $FAILED"

if [[ $FAILED -eq 0 ]]; then
    log::success "All smoke tests passed!"
    exit 0
else
    log::error "$FAILED smoke test(s) failed"
    exit 1
fi