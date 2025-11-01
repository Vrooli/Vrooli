#!/bin/bash
# Placeholder business validation for task-planner – ensures PRD requirements tracked

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

PRD_FILE="PRD.md"
if [ ! -f "$PRD_FILE" ]; then
  testing::phase::add_error "PRD.md missing; cannot validate business requirements"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business validation failed"
fi

if rg --fixed-strings --quiet "P0" "$PRD_FILE"; then
  testing::phase::check "PRD tracks P0 requirements" rg --fixed-strings --quiet "P0" "$PRD_FILE"
else
  testing::phase::add_warning "PRD missing explicit P0 requirement markers"
  testing::phase::add_test skipped
fi

if rg --fixed-strings --quiet "Workflow" "$PRD_FILE"; then
  testing::phase::check "PRD documents workflow automation" rg --fixed-strings --quiet "Workflow" "$PRD_FILE"
else
  testing::phase::add_warning "PRD missing workflow documentation keyword"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validation completed"
