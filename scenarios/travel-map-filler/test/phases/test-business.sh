#!/bin/bash
# Business logic validation for travel-map-filler.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "💼 Validating business logic flows"

has_business_tests=false
if find api -maxdepth 1 -name '*business*_test.go' -print -quit 2>/dev/null | grep -q .; then
  has_business_tests=true
elif grep -q "TestBusinessLogic" api/comprehensive_test.go 2>/dev/null; then
  has_business_tests=true
fi

if [ "$has_business_tests" = true ]; then
  if (cd api && go test -v -run "TestBusinessLogic|TestIntegration" -timeout 90s -coverprofile=business_coverage.out); then
    testing::phase::add_test passed
    log::success "✅ Business logic tests passed"
  else
    testing::phase::add_error "Business logic tests failed"
    testing::phase::add_test failed
  fi
else
  log::warning "ℹ️  No dedicated business logic tests found; skipping"
  testing::phase::add_warning "Business logic suite missing"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business logic validation completed"
