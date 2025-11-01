#!/bin/bash
# Integration test phase for travel-map-filler.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "📊 Running integration test suite"

has_integration_tests=false
if find api -maxdepth 1 -name '*integration_test.go' -print -quit 2>/dev/null | grep -q .; then
  has_integration_tests=true
fi

if [ "$has_integration_tests" = true ]; then
  if (cd api && go test -v -run "TestDatabaseIntegration|TestConcurrentOperations|TestDatabaseResilience" -timeout 120s -coverprofile=integration_coverage.out); then
    testing::phase::add_test passed
    log::success "✅ Go integration tests passed"
  else
    testing::phase::add_error "Go integration tests failed"
    testing::phase::add_test failed
  fi
else
  log::warning "ℹ️  No Go integration tests present; skipping"
  testing::phase::add_warning "Integration suite missing"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration tests completed"
