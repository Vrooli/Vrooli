#!/bin/bash
# Verifies language/runtime dependencies resolve without fetching remote resources
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

# Go module graph
if [ -f "api/go.mod" ]; then
  testing::phase::check "Go module list" bash -c 'cd api && go list ./... >/dev/null'
else
  testing::phase::add_warning "api/go.mod missing; skipping Go dependency check"
  testing::phase::add_test skipped
fi

# CLI bats availability
if command -v bats >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "bats not available; CLI tests will be skipped"
  testing::phase::add_test skipped
fi

# Tooling required by integration/business scripts
for cmd in curl jq; do
  if command -v "$cmd" >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Missing required command: $cmd"
  fi
done

testing::phase::end_with_summary "Dependency validation completed"
