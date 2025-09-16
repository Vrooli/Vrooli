#!/bin/bash
# Elmer FEM Integration Tests - Full functionality validation (<120s)

set -euo pipefail

echo "=== Elmer FEM Integration Tests ==="

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/config/defaults.sh"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test 1: Create and solve heat transfer case
echo "1. Testing heat transfer simulation..."
CASE_ID="test_heat_$(date +%s)"

# Create case
echo -n "   Creating case... "
if curl -X POST "http://localhost:${ELMER_FEM_PORT}/case/create" \
    -H "Content-Type: application/json" \
    -d "{\"name\": \"${CASE_ID}\", \"type\": \"heat_transfer\"}" \
    -o /dev/null -s -w "%{http_code}" | grep -q "200"; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Execute solve
echo -n "   Solving case... "
if curl -X POST "http://localhost:${ELMER_FEM_PORT}/case/${CASE_ID}/solve" \
    -H "Content-Type: application/json" \
    -d "{\"max_iterations\": 100}" \
    -o /dev/null -s -w "%{http_code}" | grep -q "200"; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Check results
sleep 3
echo -n "   Checking results... "
if curl -sf "http://localhost:${ELMER_FEM_PORT}/case/${CASE_ID}/results" > /dev/null 2>&1; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 2: Mesh import
echo "2. Testing mesh import..."
echo -n "   Importing GMSH format... "
if curl -X POST "http://localhost:${ELMER_FEM_PORT}/mesh/import" \
    -H "Content-Type: application/json" \
    -d '{"format": "gmsh", "name": "test_mesh", "data": "test"}' \
    -o /dev/null -s -w "%{http_code}" | grep -q "200"; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 3: List cases
echo "3. Testing case listing..."
echo -n "   Retrieving case list... "
if curl -sf "http://localhost:${ELMER_FEM_PORT}/cases" | grep -q "heat_transfer"; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 4: Parameter sweep
echo "4. Testing parameter sweep..."
echo -n "   Running conductivity sweep... "
if curl -X POST "http://localhost:${ELMER_FEM_PORT}/batch/sweep" \
    -H "Content-Type: application/json" \
    -d '{
        "case": "heat_transfer",
        "parameter": "conductivity",
        "values": [0.5, 1.0, 2.0],
        "timeout": 60
    }' \
    -o /dev/null -s -w "%{http_code}" | grep -q "200"; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Test 5: Export results
echo "5. Testing result export..."
echo -n "   Exporting VTK format... "
if curl -X GET "http://localhost:${ELMER_FEM_PORT}/case/${CASE_ID}/export?format=vtk" \
    -o /dev/null -s -w "%{http_code}" | grep -q "200"; then
    echo "PASS"
    ((TESTS_PASSED++))
else
    echo "FAIL"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo "=== Integration Test Summary ==="
echo "Passed: $TESTS_PASSED"
echo "Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -gt 0 ]]; then
    exit 1
fi

echo "All integration tests passed!"
exit 0