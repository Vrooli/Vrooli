#!/bin/bash
# Dependencies phase – verifies language/toolchain prerequisites and critical resources.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

if command -v go >/dev/null 2>&1; then
  testing::phase::check "Go module graph resolves" \
    bash -c 'set -euo pipefail; cd "$TESTING_PHASE_SCENARIO_DIR/api" && go list ./... >/dev/null'
else
  testing::phase::add_warning "Go toolchain not present; skipping Go dependency verification"
  testing::phase::add_test skipped
fi

if [ -x "$TESTING_PHASE_SCENARIO_DIR/cli/install.sh" ]; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "cli/install.sh is not executable"
  testing::phase::add_test failed
fi

if command -v node >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "Node.js runtime not available; UI health checks may be limited"
  testing::phase::add_test skipped
fi

if command -v vrooli >/dev/null 2>&1; then
  testing::phase::check "MinIO resource reachable" \
    bash -c 'set -euo pipefail; vrooli resource status minio >/dev/null 2>&1'
else
  testing::phase::add_warning "vrooli CLI unavailable; cannot confirm resource status"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
