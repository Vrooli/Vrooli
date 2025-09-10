#!/usr/bin/env bash
# Test runner for Prometheus + Grafana resource

set -euo pipefail

# Get script directory
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    readonly SCRIPT_DIR
fi
if [[ -z "${RESOURCE_DIR:-}" ]]; then
    RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
    readonly RESOURCE_DIR
fi

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run tests based on argument
TEST_TYPE="${1:-all}"

case "$TEST_TYPE" in
    smoke)
        run_smoke_tests
        ;;
    integration)
        run_integration_tests
        ;;
    unit)
        run_unit_tests
        ;;
    performance)
        run_performance_test
        ;;
    all)
        run_all_tests
        ;;
    *)
        echo "Usage: $0 [smoke|integration|unit|performance|all]"
        exit 1
        ;;
esac