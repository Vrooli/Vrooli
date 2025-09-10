#!/usr/bin/env bash
# Matrix Synapse Resource - Test Runner

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Parse arguments
TEST_PHASE="${1:-all}"

# Run requested tests
case "$TEST_PHASE" in
    smoke)
        test_smoke
        ;;
    integration)
        test_integration
        ;;
    unit)
        test_unit
        ;;
    all)
        test_all
        ;;
    *)
        echo "Unknown test phase: $TEST_PHASE"
        echo "Available: smoke, integration, unit, all"
        exit 1
        ;;
esac