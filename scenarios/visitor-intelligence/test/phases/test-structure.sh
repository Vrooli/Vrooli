#!/bin/bash
# Structure validation for visitor-intelligence

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

# Ensure core directories exist
required_dirs=(api cli docs initialization test ui .vrooli)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Validate critical files referenced by the PRD and lifecycle
required_files=(
  PRD.md
  README.md
  .vrooli/service.json
  initialization/storage/postgres/schema.sql
  cli/visitor-intelligence
  cli/install.sh
  ui/tracker.js
  docs/api-reference.md
  docs/integration-guide.md
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Confirm Go and CLI entry points are declared
if testing::phase::check "Go API entrypoints present" grep -q "func main" api/main.go; then
  true
fi

if testing::phase::check "CLI script is executable" test -x cli/visitor-intelligence; then
  true
fi

# Ensure README advertises quick start instructions
if testing::phase::check "README has Quick Start" grep -q "Quick Start" README.md; then
  true
fi

# Guard against lingering legacy configuration
if testing::phase::check "Legacy scenario-test.yaml removed" test ! -f scenario-test.yaml; then
  true
fi

testing::phase::end_with_summary "Structure validation completed"
