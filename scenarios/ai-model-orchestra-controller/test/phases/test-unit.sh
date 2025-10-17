#!/bin/bash
# AI Model Orchestra Controller - Unit Phase Tests
# Integrated with centralized Vrooli testing infrastructure

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Skip database/Redis/Ollama tests in CI environment
export SKIP_DB_TESTS="${SKIP_DB_TESTS:-true}"
export SKIP_REDIS_TESTS="${SKIP_REDIS_TESTS:-true}"
export SKIP_OLLAMA_TESTS="${SKIP_OLLAMA_TESTS:-true}"

testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

testing::phase::end_with_summary "Unit tests completed"
