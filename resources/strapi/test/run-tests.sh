#!/usr/bin/env bash
# Strapi Resource Test Runner
# Main entry point for running test phases

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly SCRIPT_DIR

# Source test library
source "${SCRIPT_DIR}/lib/test.sh"

# Parse command line arguments
PHASE="${1:-all}"

# Run requested test phase
case "$PHASE" in
    smoke)
        exec "${SCRIPT_DIR}/test/phases/test-smoke.sh"
        ;;
    integration)
        exec "${SCRIPT_DIR}/test/phases/test-integration.sh"
        ;;
    unit)
        exec "${SCRIPT_DIR}/test/phases/test-unit.sh"
        ;;
    all)
        # Run all phases in sequence
        "${SCRIPT_DIR}/test/phases/test-smoke.sh" || exit 1
        "${SCRIPT_DIR}/test/phases/test-integration.sh" || exit 1
        "${SCRIPT_DIR}/test/phases/test-unit.sh" || exit 1
        ;;
    *)
        echo "Unknown test phase: $PHASE"
        echo "Available phases: smoke, integration, unit, all"
        exit 1
        ;;
esac