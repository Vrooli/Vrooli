#!/usr/bin/env bash
################################################################################
# SageMath Integration Tests - v2.0 Universal Contract Compliant
#
# End-to-end functionality testing
################################################################################

set -euo pipefail

# Setup paths
PHASES_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${PHASES_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
APP_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# Source config
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Running SageMath integration tests..."

# Test 1: Content management - Add script
echo -n "Testing content add... "
test_script="/tmp/test_integration_$$.sage"
cat > "$test_script" << 'EOF'
# Integration test script
x = var('x')
eq = x^2 - 4 == 0
solutions = solve(eq, x)
print(f"Solutions: {solutions}")
EOF

if resource-sagemath content add --file "$test_script" --name "integration_test.sage" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: Failed to add content"
    rm -f "$test_script"
    exit 1
fi
rm -f "$test_script"

# Test 2: Content management - List
echo -n "Testing content list... "
# Small delay to ensure file system sync
sleep 0.5
if resource-sagemath content list 2>&1 | grep -q "integration_test.sage"; then
    echo "✓"
else
    echo "✗"
    echo "Error: Added content not found in list"
    exit 1
fi

# Test 3: Content management - Execute
echo -n "Testing content execute... "
output=$(resource-sagemath content execute --name "integration_test.sage" 2>/dev/null || echo "error")
if [[ "$output" == *"Solutions"* ]] && [[ "$output" == *"2"* ]] && [[ "$output" == *"-2"* ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Script execution failed or returned wrong result"
    echo "Output: $output"
    exit 1
fi

# Test 4: Mathematical operations - Symbolic computation
echo -n "Testing symbolic computation... "
result=$(resource-sagemath content calculate "diff(sin(x), x)" 2>/dev/null || echo "error")
if [[ "$result" == *"cos(x)"* ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Symbolic differentiation failed"
    echo "Result: $result"
    exit 1
fi

# Test 5: Mathematical operations - Linear algebra
echo -n "Testing linear algebra... "
matrix_calc='matrix([[1,2],[3,4]]).det()'
result=$(resource-sagemath content calculate "$matrix_calc" 2>/dev/null || echo "error")
if [[ "$result" == *"-2"* ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Matrix determinant calculation failed"
    echo "Result: $result"
    exit 1
fi

# Test 6: Mathematical operations - Number theory
echo -n "Testing number theory... "
result=$(resource-sagemath content calculate "factor(2025)" 2>/dev/null || echo "error")
if [[ "$result" == *"3"* ]] && [[ "$result" == *"5"* ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Factorization failed"
    echo "Result: $result"
    exit 1
fi

# Test 7: Content management - Remove
echo -n "Testing content remove... "
if resource-sagemath content remove --name "integration_test.sage" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: Failed to remove content"
    exit 1
fi

# Test 8: Jupyter notebook API
echo -n "Testing Jupyter API... "
api_response=$(curl -sf "http://localhost:${SAGEMATH_PORT_JUPYTER}/api/status" 2>/dev/null || echo "{}")
if [[ "$api_response" == *"started"* ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Jupyter API not responding correctly"
    exit 1
fi

echo ""
echo "All integration tests passed!"
exit 0