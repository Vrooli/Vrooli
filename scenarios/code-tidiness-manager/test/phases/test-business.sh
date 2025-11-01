#!/bin/bash
# Business logic checks ensure core cleanup artefacts remain intact

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

rules_file="initialization/storage/postgres/seed.sql"
if [ -f "$rules_file" ]; then
  if testing::phase::check "default cleanup rules present" grep -q "INSERT INTO tidiness.cleanup_rules" "$rules_file"; then
    :
  fi
else
  testing::phase::add_warning "Seed file missing: $rules_file"
  testing::phase::add_test skipped
fi

if [ -d "lib" ]; then
  if testing::phase::check "lib contains analyzer modules" find lib -maxdepth 1 -type f -name '*.go' | grep -q '.'; then
    :
  fi
else
  testing::phase::add_warning "lib directory missing"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validation completed"
