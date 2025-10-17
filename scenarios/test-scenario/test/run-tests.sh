#!/bin/bash
# Main test runner for test-scenario - executes all test phases
set -euo pipefail

# Get script directory and scenario root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "$SCENARIO_DIR/../.." && pwd)}"

# Source logging utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "======================================"
echo "Running all tests for test-scenario"
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
        local exit_code=$?
        if [ $exit_code -eq 200 ]; then
            log::warn "⊘ $phase_name skipped"
            ((SKIPPED_PHASES++))
        else
            log::error "✗ $phase_name failed (exit code: $exit_code)"
            ((FAILED_PHASES++))
        fi
        return $exit_code
    fi
}

# Run all test phases in order
run_phase "Dependencies" "$SCRIPT_DIR/phases/test-dependencies.sh" || true
run_phase "Unit Tests" "$SCRIPT_DIR/phases/test-unit.sh" || true
run_phase "Integration Tests" "$SCRIPT_DIR/phases/test-integration.sh" || true
run_phase "Business Logic" "$SCRIPT_DIR/phases/test-business.sh" || true
run_phase "Performance" "$SCRIPT_DIR/phases/test-performance.sh" || true

# Print summary
echo ""
echo "======================================"
echo "Test Summary"
echo "======================================"
echo "Total phases:   $TOTAL_PHASES"
echo "Passed:         $PASSED_PHASES"
echo "Failed:         $FAILED_PHASES"
echo "Skipped:        $SKIPPED_PHASES"
echo "======================================"

# Exit with appropriate code
if [ $FAILED_PHASES -gt 0 ]; then
    echo ""
    log::error "Some test phases failed"
    exit 1
elif [ $PASSED_PHASES -eq 0 ]; then
    echo ""
    log::warn "No tests passed"
    exit 1
else
    echo ""
    log::success "All test phases completed successfully!"
    exit 0
fi
