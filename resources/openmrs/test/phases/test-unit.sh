#!/usr/bin/env bash
# OpenMRS Unit Tests

set -euo pipefail

# Get paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source libraries
source "${RESOURCE_DIR}/../../scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/lib/core.sh"

log::info "Running OpenMRS unit tests..."

# Test results
FAILED=0
PASSED=0

# Test 1: Configuration loading
echo -n "1. Testing configuration... "
if [[ -n "${OPENMRS_PORT}" ]] && [[ -n "${OPENMRS_API_PORT}" ]] && [[ -n "${OPENMRS_FHIR_PORT}" ]]; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (configuration not loaded)"
    ((FAILED++))
fi

# Test 2: Directory structure
echo -n "2. Testing directory structure... "
if [[ -d "${OPENMRS_DIR}" ]] || mkdir -p "${OPENMRS_DIR}" 2>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (cannot create directories)"
    ((FAILED++))
fi

# Test 3: Docker availability
echo -n "3. Testing Docker... "
if docker version &>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (Docker not available)"
    ((FAILED++))
fi

# Test 4: Docker Compose availability
echo -n "4. Testing Docker Compose... "
if docker-compose version &>/dev/null || docker compose version &>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (Docker Compose not available)"
    ((FAILED++))
fi

# Test 5: Network configuration
echo -n "5. Testing network configuration... "
if docker network ls | grep -q "${OPENMRS_NETWORK}" || docker network create "${OPENMRS_NETWORK}" &>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (cannot configure network)"
    ((FAILED++))
fi

# Test 6: jq availability
echo -n "6. Testing jq availability... "
if command -v jq &>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (jq not available)"
    ((FAILED++))
fi

# Test 7: curl availability
echo -n "7. Testing curl availability... "
if command -v curl &>/dev/null; then
    echo "✓"
    ((PASSED++))
else
    echo "✗ (curl not available)"
    ((FAILED++))
fi

# Summary
echo ""
echo "Unit Test Results:"
echo "  Passed: $PASSED"
echo "  Failed: $FAILED"

if [[ $FAILED -eq 0 ]]; then
    log::success "All unit tests passed!"
    exit 0
else
    log::error "$FAILED unit test(s) failed"
    exit 1
fi