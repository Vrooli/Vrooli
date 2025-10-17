#!/bin/bash
# Main test runner for scenario-to-desktop
# This script orchestrates all test phases

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "==========================================="
echo "Running test suite for scenario-to-desktop"
echo "==========================================="

# Change to scenario directory
cd "$SCENARIO_DIR"

# Build the API if needed
if [[ ! -f "api/scenario-to-desktop-api" ]]; then
    echo "Building API binary..."
    cd api
    go build -tags testing -o scenario-to-desktop-api .
    cd ..
fi

# Track test results
FAILED_PHASES=()

# Run unit tests
echo ""
echo "=== Phase 1: Unit Tests ==="
if bash test/phases/test-unit.sh; then
    echo "✅ Unit tests passed"
else
    echo "❌ Unit tests failed"
    FAILED_PHASES+=("unit")
fi

# Run integration tests
echo ""
echo "=== Phase 2: Integration Tests ==="
if bash test/phases/test-integration.sh; then
    echo "✅ Integration tests passed"
else
    echo "❌ Integration tests failed"
    FAILED_PHASES+=("integration")
fi

# Run performance tests
echo ""
echo "=== Phase 3: Performance Tests ==="
if bash test/phases/test-performance.sh; then
    echo "✅ Performance tests passed"
else
    echo "❌ Performance tests failed"
    FAILED_PHASES+=("performance")
fi

# Summary
echo ""
echo "==========================================="
echo "Test Suite Summary"
echo "==========================================="

if [[ ${#FAILED_PHASES[@]} -eq 0 ]]; then
    echo "✅ All test phases passed!"
    exit 0
else
    echo "❌ Some test phases failed:"
    for phase in "${FAILED_PHASES[@]}"; do
        echo "  - $phase"
    done
    exit 1
fi
