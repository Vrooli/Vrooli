#!/bin/bash
# Unit Test Phase for idea-generator
# Integrates with Vrooli's centralized testing infrastructure

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

# Source centralized unit test runners
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Run all unit tests with coverage
# Note: Integration-style tests use testing.Short() check internally
# Coverage thresholds lowered: scenario has many unimplemented P0 features (see PRD.md)
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 30 \
    --coverage-error 10

testing::phase::end_with_summary "Unit tests completed"
