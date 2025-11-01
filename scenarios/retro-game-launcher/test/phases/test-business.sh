#!/bin/bash
# Business logic checks ensure core workflows and seed content remain intact.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

SEED_FILE="initialization/storage/postgres/seed.sql"
if [ -f "$SEED_FILE" ]; then
  testing::phase::check "Seed contains starter users" grep -q "retro_arcade" "$SEED_FILE"
  testing::phase::check "Seed contains prompt templates" grep -q "INSERT INTO prompt_templates" "$SEED_FILE"
  testing::phase::check "Seed contains game records" grep -q "INSERT INTO games" "$SEED_FILE"
else
  testing::phase::add_error "Seed file missing at $SEED_FILE"
fi

if [ -f "PRD.md" ]; then
  testing::phase::check "PRD documents revenue model" grep -qi "revenue" PRD.md
else
  testing::phase::add_warning "PRD.md missing"
  testing::phase::add_test skipped
fi

if command -v retro-game-launcher >/dev/null 2>&1; then
  testing::phase::check "Installed CLI lists commands" retro-game-launcher --help >/dev/null
else
  testing::phase::add_warning "retro-game-launcher CLI not on PATH; skip CLI smoke"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflow validation completed"
