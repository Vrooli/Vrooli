#!/bin/bash
# Validates business logic artefacts such as database schema coverage and CLI ergonomics

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if command -v rg >/dev/null 2>&1; then
  testing::phase::check "Schema defines feature_requests table" rg --fixed-strings --quiet "CREATE TABLE feature_requests" initialization/postgres/schema.sql
  testing::phase::check "Seed data includes default scenarios" rg --fixed-strings --quiet "INSERT INTO scenarios" initialization/postgres/seed.sql
else
  testing::phase::add_warning "ripgrep not available; skipping schema content checks"
  testing::phase::add_test skipped
fi

if [ -x "cli/feature-voting" ]; then
  testing::phase::check "CLI help command" bash -c 'cd cli && ./feature-voting --help >/dev/null'
else
  testing::phase::add_warning "feature-voting CLI not executable; skipping CLI validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validations completed"
