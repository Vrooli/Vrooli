#!/bin/bash
#
# PRD Control Tower Test Suite Runner
# Orchestrates all test phases in dependency order
#

set -e

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCENARIO_DIR"

echo "==================================="
echo "PRD Control Tower Test Suite"
echo "==================================="
echo ""

# Track test results
PASSED=0
FAILED=0
PHASE_COUNT=0

run_phase() {
    local phase_script="$1"
    local phase_name=$(basename "$phase_script" .sh)

    PHASE_COUNT=$((PHASE_COUNT + 1))
    echo "[$PHASE_COUNT] Running $phase_name..."

    if bash "$phase_script"; then
        echo "✅ $phase_name passed"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo "❌ $phase_name failed"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

# Run test phases in order
run_phase "test/phases/test-structure.sh"
run_phase "test/phases/test-dependencies.sh"
run_phase "test/phases/test-unit.sh"
run_phase "test/phases/test-integration.sh"
run_phase "test/phases/test-ui.sh"
run_phase "test/phases/test-business.sh"
run_phase "test/phases/test-performance.sh"

# Summary
echo ""
echo "==================================="
echo "Test Suite Summary"
echo "==================================="
echo "Total phases: $PHASE_COUNT"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo "✅ All tests passed!"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi
