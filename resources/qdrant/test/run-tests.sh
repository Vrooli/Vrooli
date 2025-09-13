#!/usr/bin/env bash
# Qdrant Test Runner - v2.0 Contract Compliant
# Main test orchestrator that delegates to phases

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
QDRANT_TEST_DIR="${APP_ROOT}/resources/qdrant/test"
QDRANT_LIB_DIR="${APP_ROOT}/resources/qdrant/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/test.sh"

# Parse arguments
TEST_PHASE="${1:-all}"

case "$TEST_PHASE" in
    smoke)
        qdrant::check_basic_health
        ;;
    integration)
        qdrant::test_integration
        ;;
    unit)
        qdrant::test_unit
        ;;
    all)
        qdrant::test_all
        ;;
    *)
        echo "Usage: $0 {smoke|integration|unit|all}"
        echo
        echo "Test phases:"
        echo "  smoke       - Quick health check (<30s)"
        echo "  integration - End-to-end functionality (<120s)"
        echo "  unit        - Library function tests (<60s)"
        echo "  all         - Run all test phases"
        exit 1
        ;;
esac