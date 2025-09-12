#!/usr/bin/env bash
################################################################################
# SageMath Test Library - v2.0 Universal Contract Compliant
#
# Test handlers for the SageMath resource
################################################################################

# Define paths
SAGEMATH_LIB_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
SAGEMATH_RESOURCE_DIR="$(builtin cd "${SAGEMATH_LIB_DIR}/.." && builtin pwd)"
SAGEMATH_TEST_DIR="${SAGEMATH_RESOURCE_DIR}/test"

# Source common library for shared functions and variables
source "${SAGEMATH_LIB_DIR}/common.sh"

# Universal Contract v2.0 handler for test::smoke
sagemath::test::smoke() {
    if [[ -f "${SAGEMATH_TEST_DIR}/phases/test-smoke.sh" ]]; then
        bash "${SAGEMATH_TEST_DIR}/phases/test-smoke.sh"
    else
        # Fallback to simple health check if test file doesn't exist
        sagemath::status::check
    fi
}

# Universal Contract v2.0 handler for test::unit
sagemath::test::unit() {
    if [[ -f "${SAGEMATH_TEST_DIR}/phases/test-unit.sh" ]]; then
        bash "${SAGEMATH_TEST_DIR}/phases/test-unit.sh"
    else
        echo "Unit tests not available"
        return 2
    fi
}

# Universal Contract v2.0 handler for test::integration
sagemath::test::integration() {
    if [[ -f "${SAGEMATH_TEST_DIR}/phases/test-integration.sh" ]]; then
        bash "${SAGEMATH_TEST_DIR}/phases/test-integration.sh"
    else
        echo "Integration tests not available"
        return 2
    fi
}

# Universal Contract v2.0 handler for test::all
sagemath::test::all() {
    if [[ -f "${SAGEMATH_TEST_DIR}/run-tests.sh" ]]; then
        bash "${SAGEMATH_TEST_DIR}/run-tests.sh" all
    else
        echo "Test runner not available"
        return 2
    fi
}

# Performance test handler (SageMath-specific)
sagemath::test::performance() {
    echo "Running SageMath performance benchmarks..."
    
    # Check prerequisites
    if ! sagemath_container_running; then
        echo "Error: SageMath container is not running"
        return 1
    fi
    
    echo ""
    echo "Benchmark 1: Symbolic computation speed"
    echo -n "  Solving 100 quadratic equations... "
    local start=$(date +%s%N)
    docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "
from sage.all import *
import time
x = var('x')
for i in range(100):
    solve(x^2 + i*x - i^2 == 0, x)
" 2>/dev/null
    local end=$(date +%s%N)
    local elapsed=$(( (end - start) / 1000000 ))
    echo "${elapsed}ms"
    
    echo ""
    echo "Benchmark 2: Linear algebra operations"
    echo -n "  20x20 matrix multiplication... "
    start=$(date +%s%N)
    timeout 3 docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "
from sage.all import *
A = random_matrix(RR, 20, 20)
B = random_matrix(RR, 20, 20)
C = A * B
D = A.transpose()
E = D * A
print('done')
" &>/dev/null
    if [ $? -eq 0 ]; then
        end=$(date +%s%N)
        elapsed=$(( (end - start) / 1000000 ))
        echo "${elapsed}ms"
    else
        echo "timeout (>3s)"
    fi
    
    echo ""
    echo "Benchmark 3: Number theory"
    echo -n "  Prime factorization of large numbers... "
    start=$(date +%s%N)
    timeout 5 docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "
from sage.all import *
for n in [10**8 + 7, 10**9 + 3]:
    factor(n)
print('done')
" &>/dev/null
    if [ $? -eq 0 ]; then
        end=$(date +%s%N)
        elapsed=$(( (end - start) / 1000000 ))
        echo "${elapsed}ms"
    else
        echo "timeout (>5s)"
    fi
    
    echo ""
    echo "Benchmark 4: Calculus operations"
    echo -n "  Integration and differentiation... "
    start=$(date +%s%N)
    timeout 5 docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "
from sage.all import *
x = var('x')
f = sin(x) * exp(-x)
for i in range(5):
    df = diff(f, x)
    integral(f, x)
print('done')
" &>/dev/null
    if [ $? -eq 0 ]; then
        end=$(date +%s%N)
        elapsed=$(( (end - start) / 1000000 ))
        echo "${elapsed}ms"
    else
        echo "timeout (>5s)"
    fi
    
    echo ""
    echo "Performance benchmarks complete!"
    return 0
}