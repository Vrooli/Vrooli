#!/usr/bin/env bash
# Mesa Test Runner
# Delegates to test phases per v2.0 contract

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly PHASES_DIR="${SCRIPT_DIR}/phases"

# Run specific test phase
run_phase() {
    local phase="$1"
    local script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ -f "$script" ]]; then
        echo "Running $phase tests..."
        bash "$script"
    else
        echo "Test phase '$phase' not found"
        return 1
    fi
}

# Main test runner
main() {
    local phase="${1:-all}"
    
    case "$phase" in
        smoke|integration|unit)
            run_phase "$phase"
            ;;
        all)
            local failed=0
            for p in smoke integration unit; do
                if run_phase "$p"; then
                    echo "✓ $p tests passed"
                else
                    echo "✗ $p tests failed"
                    ((failed++))
                fi
                echo ""
            done
            
            if [[ $failed -eq 0 ]]; then
                echo "All tests passed!"
                return 0
            else
                echo "$failed test phase(s) failed"
                return 1
            fi
            ;;
        *)
            echo "Unknown test phase: $phase"
            echo "Available: smoke, integration, unit, all"
            return 1
            ;;
    esac
}

main "$@"