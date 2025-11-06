#!/bin/bash
# Performance validation including Lighthouse audits for app-monitor

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s" --require-runtime

# Run Lighthouse audits if configured
if [ -f "${TESTING_PHASE_SCENARIO_DIR}/.lighthouse/config.json" ]; then
  source "${APP_ROOT}/scripts/scenarios/testing/lighthouse/runner.sh"

  log::info "Running Lighthouse performance audits..."
  if lighthouse::run_audits; then
    log::success "✅ Lighthouse audits passed"
    testing::phase::add_test passed
  else
    testing::phase::add_error "Lighthouse audits failed (see test/artifacts/lighthouse/)"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "No .lighthouse/config.json found; skipping Lighthouse audits"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance checks completed"
