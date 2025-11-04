#!/bin/bash
# Verifies language/tooling prerequisites and resource health.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/resources.sh"

testing::phase::init --target-time "60s"
cd "$TESTING_PHASE_SCENARIO_DIR"

for tool in jq curl; do
  testing::phase::check "Tool available: $tool" command -v "$tool"
done

if [ -d "api" ]; then
  testing::phase::check "Go toolchain present" command -v go
fi

if [ -d "ui" ]; then
  testing::phase::check "Node.js runtime present" command -v node
fi

if [ -d "scripts" ]; then
  testing::phase::check "Python runtime present" command -v python3
fi

if command -v jq >/dev/null 2>&1 && [ -f .vrooli/service.json ]; then
  if testing::resources::test_all "$TESTING_PHASE_SCENARIO_NAME"; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Resource integration checks reported failures"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "service.json missing or jq unavailable; resource checks skipped"
  testing::phase::add_test skipped
fi

# Example requirement mapping once the requirements registry defines dependency IDs.
# testing::phase::add_requirement --id "REQ-DEPENDENCIES" --status passed --evidence "Dependency validations"

testing::phase::end_with_summary "Dependency checks completed"
