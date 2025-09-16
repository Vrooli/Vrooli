#!/usr/bin/env bash
# OpenRocket Test Runner

set -euo pipefail

RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="${RESOURCE_DIR}/test"

# Source core library
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Test type from argument
TEST_TYPE="${1:-all}"

# Run requested tests
case "$TEST_TYPE" in
    smoke)
        "${TEST_DIR}/phases/test-smoke.sh"
        ;;
    integration)
        "${TEST_DIR}/phases/test-integration.sh"
        ;;
    unit)
        "${TEST_DIR}/phases/test-unit.sh"
        ;;
    all)
        echo "Running all OpenRocket tests..."
        "${TEST_DIR}/phases/test-smoke.sh" || exit 1
        "${TEST_DIR}/phases/test-unit.sh" || exit 1
        "${TEST_DIR}/phases/test-integration.sh" || exit 1
        echo "All tests completed successfully!"
        ;;
    *)
        echo "Unknown test type: $TEST_TYPE"
        echo "Available: smoke, integration, unit, all"
        exit 1
        ;;
esac