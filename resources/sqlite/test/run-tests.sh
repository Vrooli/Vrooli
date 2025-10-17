#!/usr/bin/env bash
################################################################################
# SQLite Resource - Test Runner
#
# Main test orchestrator that runs test phases
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source the CLI to get all functions
source "${RESOURCE_DIR}/cli.sh"

# Default to all tests if no phase specified
PHASE="${1:-all}"

case "$PHASE" in
    smoke)
        sqlite::test::smoke
        ;;
    integration)
        sqlite::test::integration
        ;;
    unit)
        sqlite::test::unit
        ;;
    all)
        sqlite::test::all
        ;;
    *)
        echo "Unknown test phase: $PHASE"
        echo "Usage: $0 [smoke|integration|unit|all]"
        exit 1
        ;;
esac