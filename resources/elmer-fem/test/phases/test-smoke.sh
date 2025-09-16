#!/bin/bash
# Elmer FEM Smoke Tests - Quick health validation (<30s)

set -euo pipefail

echo "=== Elmer FEM Smoke Tests ==="

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/config/defaults.sh"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test 1: Health endpoint
echo -n "1. Testing health endpoint... "
if timeout 5 curl -sf "http://localhost:${ELMER_FEM_PORT}/health" > /dev/null 2>&1; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 2: Version endpoint
echo -n "2. Testing version endpoint... "
if timeout 5 curl -sf "http://localhost:${ELMER_FEM_PORT}/version" 2>/dev/null | grep -q "elmer"; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 3: Docker container running
echo -n "3. Testing Docker container... "
if docker ps --format '{{.Names}}' | grep -q "${ELMER_CONTAINER_NAME}"; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 4: Data directories exist
echo -n "4. Testing data directories... "
DIRS_OK=true
for dir in cases meshes results temp; do
    if [[ ! -d "${ELMER_FEM_DATA_DIR}/${dir}" ]]; then
        DIRS_OK=false
        break
    fi
done

if [[ "$DIRS_OK" == true ]]; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 5: Example case exists
echo -n "5. Testing example cases... "
if [[ -f "${ELMER_FEM_DATA_DIR}/cases/heat_transfer/case.sif" ]]; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo "=== Smoke Test Summary ==="
echo "Passed: $TESTS_PASSED"
echo "Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -gt 0 ]]; then
    exit 1
fi

echo "All smoke tests passed!"
exit 0