#!/bin/bash
# Validates end-to-end business workflows or API contracts.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime
cd "$TESTING_PHASE_SCENARIO_DIR"

ENDPOINTS=()

if command -v rg >/dev/null 2>&1 && [ ${#ENDPOINTS[@]} -gt 0 ]; then
  for endpoint in "${ENDPOINTS[@]}"; do
    testing::phase::check "API route present: $endpoint" rg --fixed-strings --quiet "$endpoint" api
  done
else
  testing::phase::add_warning "Business validations not configured; populate ENDPOINTS or add workflow checks"
  testing::phase::add_test skipped
fi

# Inject requirement IDs once business KPIs are defined.
# testing::phase::add_requirement --id "REQ-BUSINESS" --status passed --evidence "Business workflow validation"

testing::phase::end_with_summary "Business checks completed"
