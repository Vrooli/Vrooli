#!/usr/bin/env bash
# farmOS Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Parse arguments
PHASE="${1:-all}"

# Run test phase
run_phase() {
    local phase="$1"
    local script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ -f "$script" ]]; then
        echo "Running ${phase} tests..."
        bash "$script"
        return $?
    else
        echo "Test phase '${phase}' not found"
        return 2
    fi
}

# Main execution
case "$PHASE" in
    all)
        echo "Running all test phases..."
        EXIT_CODE=0
        
        # Run each phase
        for phase in smoke integration unit; do
            if run_phase "$phase"; then
                echo "✓ ${phase} tests passed"
            else
                CODE=$?
                if [[ $CODE -eq 2 ]]; then
                    echo "⊘ ${phase} tests skipped"
                else
                    echo "✗ ${phase} tests failed"
                    EXIT_CODE=1
                fi
            fi
        done
        
        exit $EXIT_CODE
        ;;
    smoke|integration|unit)
        run_phase "$PHASE"
        ;;
    *)
        echo "Unknown test phase: $PHASE"
        echo "Available phases: all, smoke, integration, unit"
        exit 1
        ;;
esac