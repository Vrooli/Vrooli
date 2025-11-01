#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if command -v go >/dev/null 2>&1; then
  testing::phase::check "go.mod present" test -f api/go.mod
  go_required=(
    "github.com/lib/pq"
    "github.com/google/uuid"
  )
  for pkg in "${go_required[@]}"; do
    testing::phase::check "Require Go package $pkg" bash -c "cd api && grep -q '$pkg' go.mod"
  done
  testing::phase::check "go mod verify" bash -c 'cd api && go mod verify'
  testing::phase::check "List Go packages" bash -c 'cd api && go list ./... >/dev/null'
else
  testing::phase::add_warning "Go toolchain not found; skipping Go dependency checks"
  testing::phase::add_test skipped
fi

if [ -f ui/package.json ]; then
  if command -v jq >/dev/null 2>&1; then
    testing::phase::check "UI package.json is valid" jq empty ui/package.json
  else
    testing::phase::add_warning "jq not available; skipping UI manifest validation"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "UI package.json missing; UI dependency checks skipped"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
