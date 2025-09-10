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
    echo -n "  100x100 matrix operations... "
    start=$(date +%s%N)
    docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "
from sage.all import *
import numpy as np
A = matrix(RR, 100, 100, lambda i,j: i+j)
B = A.inverse()
C = A * B
det_A = A.det()
" 2>/dev/null
    end=$(date +%s%N)
    elapsed=$(( (end - start) / 1000000 ))
    echo "${elapsed}ms"
    
    echo ""
    echo "Benchmark 3: Number theory"
    echo -n "  Prime factorization of large numbers... "
    start=$(date +%s%N)
    docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "
from sage.all import *
for n in [10**10 + 7, 10**11 + 3, 10**12 + 39]:
    factor(n)
" 2>/dev/null
    end=$(date +%s%N)
    elapsed=$(( (end - start) / 1000000 ))
    echo "${elapsed}ms"
    
    echo ""
    echo "Benchmark 4: Calculus operations"
    echo -n "  Integration and differentiation... "
    start=$(date +%s%N)
    docker exec "$SAGEMATH_CONTAINER_NAME" sage -c "
from sage.all import *
x = var('x')
f = sin(x) * exp(-x) * x^2
for i in range(10):
    df = diff(f, x, i)
    integral(f, x, 0, pi)
" 2>/dev/null
    end=$(date +%s%N)
    elapsed=$(( (end - start) / 1000000 ))
    echo "${elapsed}ms"
    
    echo ""
    echo "Performance benchmarks complete!"
    return 0
}