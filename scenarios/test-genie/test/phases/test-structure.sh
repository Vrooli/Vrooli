#!/bin/bash
# Phase: Structure validation for Test Genie

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

required_dirs=(
    api
    cli
    ui
    prompts
    initialization
    test/phases
)

required_files=(
    Makefile
    PRD.md
    README.md
    .vrooli/service.json
    test/run-tests.sh
    api/main.go
    cli/test-genie
    ui/server.js
)

if testing::phase::check_directories "${required_dirs[@]}"; then
    testing::phase::add_test passed
else
    testing::phase::add_test failed
fi

if testing::phase::check_files "${required_files[@]}"; then
    testing::phase::add_test passed
else
    testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structural validation completed"
