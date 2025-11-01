#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

declare -a required_files=(
  "PRD.md"
  "README.md"
  "Makefile"
  ".vrooli/service.json"
  "api/main.go"
  "ui/server.js"
  "cli/install.sh"
)

for file in "${required_files[@]}"; do
  testing::phase::check "Required file present: $file" test -f "$file"
done

declare -a required_directories=(
  "api"
  "ui"
  "cli"
  "initialization"
  "test"
)

for dir in "${required_directories[@]}"; do
  testing::phase::check "Required directory present: $dir" test -d "$dir"
done

if [ -f "api/go.mod" ]; then
  testing::phase::check "Go module declaration present" grep -q "^module " api/go.mod
else
  testing::phase::add_warning "Go module not found; API structure may be incomplete"
  testing::phase::add_test skipped
fi

if [ -f "ui/package.json" ]; then
  if command -v node >/dev/null 2>&1; then
    testing::phase::check "UI package.json parses" bash -c 'cd ui && node -e "require(\"./package.json\")"'
  else
    testing::phase::add_warning "Node.js not available; skipping package.json validation"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "UI package.json missing"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
