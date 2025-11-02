#!/bin/bash
# Validates scenario structure for tech-tree-designer
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"
cd "${TESTING_PHASE_SCENARIO_DIR}"

REQUIRED_DIRS=(
  ".vrooli"
  "api"
  "cli"
  "docs"
  "initialization"
  "test"
  "test/phases"
  "ui"
)
if testing::phase::check_directories "${REQUIRED_DIRS[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

REQUIRED_FILES=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "PROBLEMS.md"
  "Makefile"
  "api/main.go"
  "api/go.mod"
  "ui/package.json"
  "cli/install.sh"
)
if testing::phase::check_files "${REQUIRED_FILES[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::check "service.json parses" jq empty .vrooli/service.json
if [ -x "cli/tech-tree-designer" ]; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "CLI binary not yet built; skipping executable check"
  testing::phase::add_test skipped
fi

testing::phase::check "Makefile exposes lifecycle targets" bash -c '
  required_targets=(help start run stop test logs status clean fmt lint)
  missing=0
  for target in "${required_targets[@]}"; do
    if ! grep -q "^${target}:" Makefile; then
      missing=$((missing + 1))
    fi
  done
  [ "$missing" -eq 0 ]
'

testing::phase::check "Test directory wired" bash -c '[ -f test/run-tests.sh ] && [ -d test/phases ]'

testing::phase::end_with_summary "Structure validation completed"
