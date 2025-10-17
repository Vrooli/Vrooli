#!/bin/bash
set -e

# Initialize centralized testing framework
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

# Import centralized unit test runner
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Set up test database URL for Go tests
export TEST_POSTGRES_URL="${POSTGRES_URL:-postgresql://postgres:postgres@localhost:5432/workflow_scheduler_test}"

# Create test database if it doesn't exist
psql -U postgres -h localhost -c "CREATE DATABASE workflow_scheduler_test;" 2>/dev/null || true

# Run all unit tests with coverage
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

# CLI tests if CLI exists
if [ -f "${TESTING_PHASE_SCENARIO_DIR}/cli/scheduler-cli" ]; then
  echo "Testing CLI..."
  if "${TESTING_PHASE_SCENARIO_DIR}/cli/scheduler-cli" --help > /dev/null 2>&1; then
    echo "✅ CLI help command works"
  else
    echo "❌ CLI help command failed"
    exit 1
  fi
fi

testing::phase::end_with_summary "Unit tests completed"