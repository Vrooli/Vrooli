#!/bin/bash
# JupyterHub Test Runner

set -euo pipefail

# Determine script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source test phases
PHASES_DIR="${SCRIPT_DIR}/phases"

# Run test phases in order
run_test_phase() {
    local phase="$1"
    local phase_script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ -f "$phase_script" ]]; then
        echo "Running $phase tests..."
        bash "$phase_script"
    else
        echo "⚠️  Test phase '$phase' not found, skipping"
    fi
}

# Main test execution
main() {
    echo "🧪 JupyterHub Resource Test Suite"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Check if service is running
    if ! docker ps --filter name=vrooli-jupyterhub -q | grep -q .; then
        echo "⚠️  JupyterHub is not running. Starting service for tests..."
        "${RESOURCE_DIR}/cli.sh" manage start --wait
    fi
    
    # Run test phases
    run_test_phase "smoke"
    run_test_phase "unit"
    run_test_phase "integration"
    
    echo ""
    echo "✅ All tests completed"
}

main "$@"