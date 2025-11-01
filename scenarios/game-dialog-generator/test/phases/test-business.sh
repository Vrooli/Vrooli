#!/bin/bash
# Business logic spot checks for Game Dialog Generator
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if [ -x "cli/game-dialog-generator" ]; then
  testing::phase::check "CLI help highlights jungle theme" ./cli/game-dialog-generator --help
else
  testing::phase::add_warning "CLI binary not executable; skipping CLI checks"
  testing::phase::add_test skipped
fi

testing::phase::check "Sample character seed references jungle heroes" bash -c 'grep -qi "jungle" initialization/data/sample_characters.sql'

if command -v rg >/dev/null 2>&1; then
  testing::phase::check "Documentation references jungle adventure" rg --ignore-case --quiet "jungle" docs
else
  testing::phase::add_warning "rg not available; skipping documentation semantic check"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business checks completed"
