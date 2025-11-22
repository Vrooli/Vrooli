#!/bin/bash
# Runs Browser Automation Studio workflow automations and CLI BATS tests from requirements registry.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/integration.sh"

# Run BATS CLI tests first
# TESTING_PHASE_SCENARIO_DIR is the scenario root, not test/ directory
SCENARIO_DIR="${TESTING_PHASE_SCENARIO_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
CLI_TEST_DIR="${SCENARIO_DIR}/test/cli"

if [ -d "$CLI_TEST_DIR" ] && command -v bats >/dev/null 2>&1; then
    echo "🧪 Running CLI BATS tests..."
    # Unset API_PORT to let CLI detect it dynamically
    # (test framework may set stale values from previous scenarios)
    unset API_PORT
    if bats "$CLI_TEST_DIR"/*.bats; then
        echo "✅ BATS tests passed"
    else
        echo "❌ BATS tests failed" >&2
        exit 1
    fi
else
    [ ! -d "$CLI_TEST_DIR" ] && echo "⚠️  No CLI tests found at $CLI_TEST_DIR"
    [ ! command -v bats >/dev/null 2>&1 ] && echo "⚠️  bats not installed"
fi

# Run BAS workflow validations
testing::integration::validate_all
