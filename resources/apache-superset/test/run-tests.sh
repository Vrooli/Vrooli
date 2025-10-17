#!/usr/bin/env bash
# Apache Superset Test Runner
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

# Source the CLI
source "${RESOURCE_DIR}/cli.sh"

# Run specified test phase or all
TEST_PHASE="${1:-all}"

case "$TEST_PHASE" in
    smoke)
        superset::test smoke
        ;;
    integration)
        superset::test integration
        ;;
    unit)
        superset::test unit
        ;;
    all)
        superset::test all
        ;;
    *)
        echo "Usage: $0 [smoke|integration|unit|all]"
        exit 1
        ;;
esac