#!/bin/bash
# Elmer FEM Test Runner

set -euo pipefail

# Get test phase from argument
PHASE="${1:-all}"

# Source test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/test.sh"

# Run tests based on phase
case "$PHASE" in
    smoke)
        "${SCRIPT_DIR}/phases/test-smoke.sh"
        ;;
    integration)
        "${SCRIPT_DIR}/phases/test-integration.sh"
        ;;
    unit)
        "${SCRIPT_DIR}/phases/test-unit.sh"
        ;;
    all)
        "${SCRIPT_DIR}/phases/test-smoke.sh"
        "${SCRIPT_DIR}/phases/test-integration.sh"
        "${SCRIPT_DIR}/phases/test-unit.sh"
        ;;
    *)
        echo "ERROR: Invalid test phase: $PHASE"
        echo "Valid phases: smoke, integration, unit, all"
        exit 1
        ;;
esac