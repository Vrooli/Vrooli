#!/bin/bash
# Structural validation for app-issue-tracker
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(
  "api"
  "ui"
  "cli"
  "issues/open"
  "issues/templates"
  "test/phases"
)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  ".vrooli/service.json"
  "README.md"
  "PRD.md"
  "test/run-tests.sh"
  "cli/app-issue-tracker.sh"
  "issues/manage.sh"
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml is present; phased testing should own validation"
fi

testing::phase::end_with_summary "Structure validation completed"
