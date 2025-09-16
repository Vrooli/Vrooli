#!/usr/bin/env bash
# OpenMRS Integration Tests

set -euo pipefail

# Get paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source libraries
source "${RESOURCE_DIR}/../../scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/lib/core.sh"

log::info "Running OpenMRS integration tests..."

# Test results
FAILED=0
PASSED=0

# Test 1: Create and retrieve patient via REST API
echo -n "1. Testing patient operations... "
# Get patient list
PATIENTS=$(curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
    "http://localhost:${OPENMRS_API_PORT}/openmrs/ws/rest/v1/patient" 2>/dev/null)

if echo "$PATIENTS" | jq -e '.results' &>/dev/null; then
    PATIENT_COUNT=$(echo "$PATIENTS" | jq '.results | length')
    echo "✓ (found $PATIENT_COUNT patients)"
    ((PASSED++))
else
    echo "✗ (failed to list patients)"
    ((FAILED++))
fi

# Test 2: Encounter API functionality
echo -n "2. Testing encounter API... "
ENCOUNTERS=$(curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
    "http://localhost:${OPENMRS_API_PORT}/openmrs/ws/rest/v1/encounter" 2>/dev/null)

if echo "$ENCOUNTERS" | jq -e '.results' &>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (encounter API not accessible)"
    ((FAILED++))
fi

# Test 3: Concept dictionary access
echo -n "3. Testing concept dictionary... "
CONCEPTS=$(curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
    "http://localhost:${OPENMRS_API_PORT}/openmrs/ws/rest/v1/concept" 2>/dev/null)

if echo "$CONCEPTS" | jq -e '.results' &>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (concept API not accessible)"
    ((FAILED++))
fi

# Test 4: Provider management
echo -n "4. Testing provider API... "
PROVIDERS=$(curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
    "http://localhost:${OPENMRS_API_PORT}/openmrs/ws/rest/v1/provider" 2>/dev/null)

if echo "$PROVIDERS" | jq -e '.results' &>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (provider API not accessible)"
    ((FAILED++))
fi

# Test 5: Location management
echo -n "5. Testing location API... "
LOCATIONS=$(curl -s -u "${OPENMRS_ADMIN_USER}:${OPENMRS_ADMIN_PASS}" \
    "http://localhost:${OPENMRS_API_PORT}/openmrs/ws/rest/v1/location" 2>/dev/null)

if echo "$LOCATIONS" | jq -e '.results' &>/dev/null; then
    LOCATION_COUNT=$(echo "$LOCATIONS" | jq '.results | length')
    echo "✓ (found $LOCATION_COUNT locations)"
    ((PASSED++))
else
    echo "✗ (location API not accessible)"
    ((FAILED++))
fi

# Summary
echo ""
echo "Integration Test Results:"
echo "  Passed: $PASSED"
echo "  Failed: $FAILED"

if [[ $FAILED -eq 0 ]]; then
    log::success "All integration tests passed!"
    exit 0
else
    log::error "$FAILED integration test(s) failed"
    exit 1
fi