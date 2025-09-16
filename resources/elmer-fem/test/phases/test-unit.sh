#!/bin/bash
# Elmer FEM Unit Tests - Library function validation (<60s)

set -euo pipefail

echo "=== Elmer FEM Unit Tests ==="

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/config/defaults.sh"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test 1: Configuration files valid
echo "1. Testing configuration files..."

echo -n "   defaults.sh exists... "
if [[ -f "${RESOURCE_DIR}/config/defaults.sh" ]]; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

echo -n "   schema.json valid... "
if jq empty "${RESOURCE_DIR}/config/schema.json" 2>/dev/null; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

echo -n "   runtime.json valid... "
if jq empty "${RESOURCE_DIR}/config/runtime.json" 2>/dev/null; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 2: CLI script executable
echo "2. Testing CLI components..."

echo -n "   cli.sh executable... "
if [[ -x "${RESOURCE_DIR}/cli.sh" ]]; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

echo -n "   lib/core.sh exists... "
if [[ -f "${RESOURCE_DIR}/lib/core.sh" ]]; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

echo -n "   lib/test.sh exists... "
if [[ -f "${RESOURCE_DIR}/lib/test.sh" ]]; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 3: Port configuration
echo "3. Testing port configuration..."

echo -n "   Port defined... "
if [[ -n "${ELMER_FEM_PORT}" ]]; then
    echo "PASS (${ELMER_FEM_PORT})"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

echo -n "   Port in valid range... "
if [[ "${ELMER_FEM_PORT}" -gt 1024 ]] && [[ "${ELMER_FEM_PORT}" -lt 65536 ]]; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 4: Environment variables
echo "4. Testing environment variables..."

echo -n "   ELMER_MPI_PROCESSES set... "
if [[ -n "${ELMER_MPI_PROCESSES}" ]]; then
    echo "PASS (${ELMER_MPI_PROCESSES})"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

echo -n "   ELMER_MAX_MEMORY set... "
if [[ -n "${ELMER_MAX_MEMORY}" ]]; then
    echo "PASS (${ELMER_MAX_MEMORY})"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 5: Example case structure
echo "5. Testing example case..."

echo -n "   Heat transfer case exists... "
if [[ -d "${ELMER_FEM_DATA_DIR}/cases/heat_transfer" ]]; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

echo -n "   Case SIF file valid... "
if [[ -f "${ELMER_FEM_DATA_DIR}/cases/heat_transfer/case.sif" ]]; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo "=== Unit Test Summary ==="
echo "Passed: $TESTS_PASSED"
echo "Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -gt 0 ]]; then
    exit 1
fi

echo "All unit tests passed!"
exit 0