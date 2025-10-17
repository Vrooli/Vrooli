#!/bin/bash
# Traccar Test Runner

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Test phase to run
PHASE="${1:-all}"

# Run specific phase or all phases
case "$PHASE" in
    smoke)
        echo "Running smoke tests..."
        bash "${PHASES_DIR}/test-smoke.sh"
        ;;
    integration)
        echo "Running integration tests..."
        bash "${PHASES_DIR}/test-integration.sh"
        ;;
    unit)
        echo "Running unit tests..."
        bash "${PHASES_DIR}/test-unit.sh"
        ;;
    all)
        echo "Running all test phases..."
        
        # Run smoke tests
        echo ""
        echo "=== SMOKE TESTS ==="
        if bash "${PHASES_DIR}/test-smoke.sh"; then
            echo "✓ Smoke tests passed"
        else
            echo "✗ Smoke tests failed"
            exit 1
        fi
        
        # Run integration tests
        echo ""
        echo "=== INTEGRATION TESTS ==="
        if bash "${PHASES_DIR}/test-integration.sh"; then
            echo "✓ Integration tests passed"
        else
            echo "✗ Integration tests failed"
            exit 1
        fi
        
        # Run unit tests
        echo ""
        echo "=== UNIT TESTS ==="
        if bash "${PHASES_DIR}/test-unit.sh"; then
            echo "✓ Unit tests passed"
        else
            echo "✗ Unit tests failed"
            exit 1
        fi
        
        echo ""
        echo "=== ALL TESTS PASSED ==="
        ;;
    *)
        echo "Unknown test phase: $PHASE"
        echo "Valid phases: smoke, integration, unit, all"
        exit 1
        ;;
esac