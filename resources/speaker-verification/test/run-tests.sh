#!/usr/bin/env bash
# Speaker Verification Test Runner - v2.0 Contract Compliant

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SV_TEST_DIR="${APP_ROOT}/resources/speaker-verification/test"
SV_LIB_DIR="${APP_ROOT}/resources/speaker-verification/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${SV_LIB_DIR}/test.sh"

# Parse arguments
TEST_PHASE="${1:-all}"

case "$TEST_PHASE" in
    smoke)
        speaker_verification::test::smoke
        ;;
    integration)
        speaker_verification::test::integration
        ;;
    unit)
        speaker_verification::test::unit
        ;;
    all)
        speaker_verification::test::all
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
