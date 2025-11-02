#!/bin/bash
# Structure validation for core-debugger scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "20s"

required_dirs=(api cli data scripts .vrooli)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  PRD.md
  README.md
  cli/core-debugger
  api/main.go
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "service.json is valid" jq empty .vrooli/service.json
else
  testing::phase::add_warning "jq not available; skipping service.json validation"
  testing::phase::add_test skipped
fi

if command -v go >/dev/null 2>&1; then
  testing::phase::check "Go sources compile" bash -c 'cd api && go build -o /tmp/core-debugger-struct-check main.go && rm /tmp/core-debugger-struct-check'
else
  testing::phase::add_warning "Go toolchain not available; skipping compile check"
  testing::phase::add_test skipped
fi

if [ -x cli/core-debugger ]; then
  testing::phase::check "CLI shebang present" bash -c 'head -n 1 cli/core-debugger | grep -Eq "^#!/"'
else
  testing::phase::add_warning "CLI binary missing or not executable"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
