#!/bin/bash
# Main test runner for scenario-to-extension - executes all test phases
set -euo pipefail

# Get script directory and scenario root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "$SCENARIO_DIR/../.." && pwd)}"

# Source logging utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "======================================"
echo "Running all tests for scenario-to-extension"
echo "======================================"
echo ""

# Track overall test results
TOTAL_PHASES=0
PASSED_PHASES=0
FAILED_PHASES=0
SKIPPED_PHASES=0

# Function to run a test phase
run_phase() {
    local phase_name=$1
    local phase_script=$2

    ((TOTAL_PHASES++))

    echo ""
    echo "--------------------------------------"
    echo "Running phase: $phase_name"
    echo "--------------------------------------"

    if [ ! -f "$phase_script" ]; then
        log::warn "Phase script not found: $phase_script"
        ((SKIPPED_PHASES++))
        return 0
    fi

    if ! [ -x "$phase_script" ]; then
        chmod +x "$phase_script"
    fi

    if "$phase_script"; then
        log::success "✓ $phase_name passed"
        ((PASSED_PHASES++))
        return 0
    else
        log::error "✗ $phase_name failed"
        ((FAILED_PHASES++))
        return 1
    fi
}

# Define test phases in execution order
PHASE_DIR="$SCRIPT_DIR/phases"

# Run each test phase
run_phase "Dependencies" "$PHASE_DIR/test-dependencies.sh" || true
run_phase "Structure" "$PHASE_DIR/test-structure.sh" || true
run_phase "CLI" "$SCRIPT_DIR/cli/test-cli.sh" || true
run_phase "API Unit Tests" "$PHASE_DIR/test-api.sh" || true

# Print summary
echo ""
echo "======================================"
echo "Test Summary"
echo "======================================"
echo "Total phases: $TOTAL_PHASES"
echo "Passed: $PASSED_PHASES"
echo "Failed: $FAILED_PHASES"
echo "Skipped: $SKIPPED_PHASES"
echo ""

if [ $FAILED_PHASES -gt 0 ]; then
    log::error "Some tests failed!"
    exit 1
elif [ $PASSED_PHASES -eq 0 ]; then
    log::warn "No tests were run!"
    exit 1
else
    log::success "All tests passed!"
    exit 0
fi
