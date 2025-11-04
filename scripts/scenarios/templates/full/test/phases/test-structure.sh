#!/bin/bash
# Validates required files and directory layout using the modern phase helpers.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"
cd "$TESTING_PHASE_SCENARIO_DIR"

required_files=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_dirs=(
  "api"
  "cli"
  "test"
  "test/phases"
)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1; then
  if jq empty < .vrooli/service.json >/dev/null 2>&1; then
    scenario_name=$(basename "$TESTING_PHASE_SCENARIO_DIR")
    service_name=$(jq -r '.service.name // empty' .vrooli/service.json)
    if [ -n "$service_name" ] && [ "$service_name" = "$scenario_name" ]; then
      log::success "✅ service.json name matches scenario directory"
      testing::phase::add_test passed
    else
      log::warning "⚠️  service.json name missing or mismatched"
      testing::phase::add_warning "service.json service.name should equal scenario directory"
      testing::phase::add_test failed
    fi
  else
    testing::phase::add_error "service.json contains invalid JSON"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "jq not available; skipping service.json validation"
  testing::phase::add_test skipped
fi

# Hook requirements here once the requirements registry is populated (docs/requirements.yaml or requirements/), e.g.:
# testing::phase::add_requirement --id "REQ-STRUCTURE" --status passed --evidence "Structure validations"

testing::phase::end_with_summary "Structure checks completed"
