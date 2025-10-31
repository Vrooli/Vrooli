#!/bin/bash
# Business-layer validation: ensure core workflow endpoints remain wired.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

if ! command -v rg >/dev/null 2>&1; then
  testing::phase::add_warning "ripgrep (rg) not available; skipping endpoint signature checks"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Business logic checks skipped"
fi

endpoints=(
  "/api/v1/workflows"
  "/api/v1/projects"
  "/health"
)

for endpoint in "${endpoints[@]}"; do
  testing::phase::check "API route present: ${endpoint}" rg --fixed-strings --quiet "$endpoint" api
  # check() already updates test counters
done

testing::phase::end_with_summary "Business logic validation completed"
