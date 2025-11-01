#!/bin/bash
# Validate business workflows and data readiness
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Ensure template catalog is populated when Postgres is available
if command -v psql >/dev/null 2>&1 && command -v resource-postgres >/dev/null 2>&1; then
  if resource-postgres status >/dev/null 2>&1; then
    testing::phase::check "Seed template catalog present" bash -c "psql -U postgres -d vrooli_db -t -c 'SELECT COUNT(*) FROM referral_templates' | tr -d '[:space:]' | grep -Eq '^[1-9][0-9]*$'"
  else
    testing::phase::add_warning "Postgres resource not running; skipping template validation"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "psql or resource-postgres unavailable; skipping template validation"
  testing::phase::add_test skipped
fi

# Validate CLI can produce usage output if installed
if command -v referral-program-generator >/dev/null 2>&1; then
  testing::phase::check "CLI responds to --help" referral-program-generator --help
else
  testing::phase::add_warning "CLI not installed in PATH; skipping CLI generation smoke test"
  testing::phase::add_test skipped
fi

# Confirm automation scripts exist for downstream workflows
analytics_scripts=(
  scripts/analyze-scenario.sh
  scripts/claude-code-integration.sh
  scripts/generate-referral-pattern.sh
)
if testing::phase::check_files "${analytics_scripts[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Business validation completed"
