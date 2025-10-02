#!/bin/bash

set -e

echo "=================================="
echo "Contact Book Phased Testing Suite"
echo "=================================="
echo ""

# Change to scenario directory
cd "$(dirname "$0")/.."

# Track test results
TOTAL_PHASES=0
PASSED_PHASES=0
FAILED_PHASES=()

# Function to run a test phase
run_phase() {
    local phase_name=$1
    local phase_script=$2

    TOTAL_PHASES=$((TOTAL_PHASES + 1))
    echo "Running Phase: $phase_name"
    echo "-----------------------------------"

    if [ -f "$phase_script" ]; then
        chmod +x "$phase_script"
        if bash "$phase_script"; then
            PASSED_PHASES=$((PASSED_PHASES + 1))
            echo "✅ $phase_name PASSED"
        else
            FAILED_PHASES+=("$phase_name")
            echo "❌ $phase_name FAILED"
        fi
    else
        echo "⚠️  $phase_script not found, skipping"
    fi
    echo ""
}

# Run all test phases in order
run_phase "Structure" "test/phases/test-structure.sh"
run_phase "Dependencies" "test/phases/test-dependencies.sh"
run_phase "Unit Tests" "test/phases/test-unit.sh"
run_phase "Integration Tests" "test/phases/test-integration.sh"
run_phase "Performance Tests" "test/phases/test-performance.sh"
run_phase "Business Value Tests" "test/phases/test-business.sh"

# Summary
echo "=================================="
echo "Test Summary"
echo "=================================="
echo "Total Phases: $TOTAL_PHASES"
echo "Passed: $PASSED_PHASES"
echo "Failed: $((TOTAL_PHASES - PASSED_PHASES))"

if [ ${#FAILED_PHASES[@]} -gt 0 ]; then
    echo ""
    echo "Failed Phases:"
    for phase in "${FAILED_PHASES[@]}"; do
        echo "  - $phase"
    done
    exit 1
else
    echo ""
    echo "✅ All test phases passed!"
    exit 0
fi
