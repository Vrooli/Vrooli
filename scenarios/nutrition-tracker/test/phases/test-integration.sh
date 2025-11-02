#!/bin/bash
# Exercise cross-component integrations for nutrition-tracker

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s"

if [ -x "cli/nutrition-tracker" ]; then
  testing::phase::check "CLI smoke test" ./cli/nutrition-tracker --help
else
  testing::phase::add_warning "CLI binary missing; skipping CLI smoke test"
  testing::phase::add_test skipped
fi

if [ -f "api/go.mod" ]; then
  testing::phase::check "Go integration suite (tags=testing)" bash -c 'cd api && go test -tags testing -run TestFull ./...'
else
  testing::phase::add_warning "API module missing; skipping Go integration tests"
  testing::phase::add_test skipped
fi

if command -v jq >/dev/null 2>&1 && [ -f "data/seed/foods.json" ]; then
  testing::phase::check "Seed data includes nutrition entries" jq -e '.foods | length > 0' data/seed/foods.json
else
  testing::phase::add_warning "data/seed/foods.json not available or jq missing; skipping data validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration tests completed"
