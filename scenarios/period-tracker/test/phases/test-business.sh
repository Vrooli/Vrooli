#!/bin/bash
# Static validation that core business routes and privacy artifacts remain in place
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

business_routes=(
  "/api/v1/cycles"
  "/api/v1/symptoms"
  "/api/v1/predictions"
  "/api/v1/health/encryption"
  "/api/v1/auth/status"
)

for route in "${business_routes[@]}"; do
  if testing::phase::check "Route registered: ${route}" rg --fixed-strings --quiet "${route}" api/main.go; then
    :
  fi
done

if testing::phase::check "Seed data protects privacy" rg --fixed-strings --quiet "ANONYMIZED" initialization/postgres/seed.sql; then
  :
fi

testing::phase::end_with_summary "Business logic validation completed"
