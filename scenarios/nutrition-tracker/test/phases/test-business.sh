#!/bin/bash
# Validate end-to-end business logic artefacts (workflows, seeds, AI prompts)

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

if command -v jq >/dev/null 2>&1; then
  workflows=(
    initialization/n8n/calorie-calculator.json
    initialization/n8n/meal-suggester.json
    initialization/n8n/nutrition-analyzer.json
  )
  for workflow in "${workflows[@]}"; do
    if [ -f "$workflow" ]; then
      testing::phase::check "Workflow nodes present: $(basename "$workflow")" jq -e '.nodes | length > 0' "$workflow"
    else
      testing::phase::add_error "Missing required workflow: $workflow"
      testing::phase::add_test failed
    fi
  done
else
  testing::phase::add_warning "jq not available; skipping workflow content validation"
  testing::phase::add_test skipped
fi

if testing::phase::check_files initialization/postgres/schema.sql initialization/postgres/seed.sql; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if [ -f "data/seed/foods.json" ] && command -v jq >/dev/null 2>&1; then
  testing::phase::check "Seed foods include entries" jq -e '.foods | length > 0' data/seed/foods.json
else
  testing::phase::add_warning "Seed foods JSON missing or jq unavailable; skipping dataset validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business logic validation completed"
