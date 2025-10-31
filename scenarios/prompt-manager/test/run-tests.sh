#!/usr/bin/env bash
# Main test runner for prompt-manager scenario
# Orchestrates all test phases in the correct order

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🧪 Running all tests for prompt-manager"
echo "========================================"
echo ""

# Initialize test results
TOTAL_PHASES=0
PASSED_PHASES=0
FAILED_PHASES=0

# Function to run a test phase
run_phase() {
    local phase_name="$1"
    local phase_script="$SCRIPT_DIR/phases/$phase_name.sh"

    TOTAL_PHASES=$((TOTAL_PHASES + 1))

    echo ""
    echo "📋 Phase $TOTAL_PHASES: $phase_name"
    echo "----------------------------------------"

    if [ ! -f "$phase_script" ]; then
        echo "⚠️  Phase script not found: $phase_script"
        FAILED_PHASES=$((FAILED_PHASES + 1))
        return 1
    fi

    if bash "$phase_script"; then
        echo "✅ Phase passed: $phase_name"
        PASSED_PHASES=$((PASSED_PHASES + 1))
        return 0
    else
        echo "❌ Phase failed: $phase_name"
        FAILED_PHASES=$((FAILED_PHASES + 1))
        return 1
    fi
}

# Run test phases in order
# Order: structure → dependencies → unit → integration → business → cli → performance

run_phase "test-structure" || true
run_phase "test-dependencies" || true
run_phase "test-unit" || true
run_phase "test-integration" || true
run_phase "test-business" || true
run_phase "test-cli" || true
run_phase "test-performance" || true

# Print summary
echo ""
echo "========================================"
echo "📊 Test Summary"
echo "========================================"
echo "Total phases: $TOTAL_PHASES"
echo "Passed: $PASSED_PHASES"
echo "Failed: $FAILED_PHASES"
echo ""

if [ "$FAILED_PHASES" -eq 0 ]; then
    echo "✅ All tests passed!"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi
