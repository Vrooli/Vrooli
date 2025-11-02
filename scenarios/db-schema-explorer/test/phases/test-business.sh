#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

if ! command -v bats >/dev/null 2>&1; then
  testing::phase::add_warning "BATS not installed; skipping CLI workflow checks"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Business workflow checks skipped"
fi

if testing::phase::check "CLI smoke tests" bats cli/db-schema-explorer.bats; then
  :
fi

testing::phase::end_with_summary "Business workflow validation completed"
