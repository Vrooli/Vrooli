#!/bin/bash
# Runs integration-level Go tests that exercise API workflows against the in-memory test database

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if [ -d "api" ]; then
  testing::phase::check "Go API integration flows" bash -c 'cd api && GOFLAGS="${GOFLAGS:-} -tags=testing" go test -run '\''Test(Health|ListFeatureRequests|CreateFeatureRequest|Vote)'\'' -count=1 ./...'
  testing::phase::check "Scenario management integration flows" bash -c 'cd api && GOFLAGS="${GOFLAGS:-} -tags=testing" go test -run '\''Test(UpdateFeatureRequest|DeleteFeatureRequest|ListScenarios|GetScenario)'\'' -count=1 ./...'
else
  testing::phase::add_warning "API directory missing; skipping integration tests"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration tests completed"
