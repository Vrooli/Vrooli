#!/usr/bin/env bash
################################################################################
# ESPHome Test Runner - v2.0 Contract Compliant
################################################################################

set -euo pipefail

# Get the absolute path to the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source required files
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/lib/core.sh"
# shellcheck disable=SC1091
source "${RESOURCE_DIR}/lib/test.sh"

# Export configuration
esphome::export_config

# Parse test phase argument
TEST_PHASE="${1:-all}"

# Execute requested test phase
case "$TEST_PHASE" in
    smoke)
        esphome::test::smoke
        ;;
    integration)
        esphome::test::integration
        ;;
    unit)
        esphome::test::unit
        ;;
    all)
        esphome::test::all
        ;;
    *)
        echo "Unknown test phase: $TEST_PHASE"
        echo "Usage: $0 [smoke|integration|unit|all]"
        exit 1
        ;;
esac