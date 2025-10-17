#!/usr/bin/env bash
################################################################################
# SageMath Unit Tests - v2.0 Universal Contract Compliant
#
# Tests for library functions
################################################################################

set -euo pipefail

# Setup paths
PHASES_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${PHASES_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
APP_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# Source libraries
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/lib/common.sh"

echo "Running SageMath unit tests..."

# Test 1: Configuration values
echo -n "Testing configuration... "
if [[ -n "$SAGEMATH_CONTAINER_NAME" ]] && \
   [[ -n "$SAGEMATH_IMAGE" ]] && \
   [[ -n "$SAGEMATH_PORT_JUPYTER" ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Configuration variables not set"
    exit 1
fi

# Test 2: Directory creation function
echo -n "Testing directory creation... "
if type -t sagemath_ensure_directories > /dev/null; then
    # Function exists, try to call it
    if sagemath_ensure_directories; then
        echo "✓"
    else
        echo "✗"
        echo "Error: Directory creation failed"
        exit 1
    fi
else
    echo "✗"
    echo "Error: sagemath_ensure_directories function not found"
    exit 1
fi

# Test 3: Container existence check
echo -n "Testing container check functions... "
if type -t sagemath_container_exists > /dev/null && \
   type -t sagemath_container_running > /dev/null; then
    echo "✓"
else
    echo "✗"
    echo "Error: Container check functions not found"
    exit 1
fi

# Test 4: Data directories exist
echo -n "Testing data directories... "
if [[ -d "$SAGEMATH_SCRIPTS_DIR" ]] && \
   [[ -d "$SAGEMATH_NOTEBOOKS_DIR" ]] && \
   [[ -d "$SAGEMATH_OUTPUTS_DIR" ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Data directories do not exist"
    exit 1
fi

# Test 5: Test script exists
echo -n "Testing example scripts... "
if [[ -f "$SAGEMATH_SCRIPTS_DIR/test.sage" ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Test script not found"
    exit 1
fi

echo ""
echo "All unit tests passed!"
exit 0