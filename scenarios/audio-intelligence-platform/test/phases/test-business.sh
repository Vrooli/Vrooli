#!/bin/bash
# Business logic validation for audio-intelligence-platform
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s"

if command -v bats >/dev/null 2>&1; then
  if testing::phase::check "CLI contract tests" bats cli/audio-intelligence-platform.bats; then
    :
  fi
else
  testing::phase::add_warning "BATS not available; skipping CLI tests"
  testing::phase::add_test skipped
fi

if testing::phase::check "Custom workflow validation" bash -c 'source custom-tests.sh && custom_tests::run_custom_tests'; then
  :
fi

testing::phase::end_with_summary "Business logic validation completed"
