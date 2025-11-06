#!/bin/bash
# Performance validation including Lighthouse audits and build metrics for browser-automation-studio.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

# Run Lighthouse audits if config exists
if [ -f "${TESTING_PHASE_SCENARIO_DIR}/.lighthouse/config.json" ]; then
  source "${APP_ROOT}/scripts/scenarios/testing/lighthouse/runner.sh"

  if lighthouse::run_audits; then
    log::success "✅ Lighthouse audits passed"
  else
    testing::phase::add_error "Lighthouse audits failed (see test/artifacts/lighthouse/)"
  fi
else
  log::info "No .lighthouse/config.json found; skipping Lighthouse audits"
  log::info "To enable Lighthouse testing:"
  log::info "  1. cd ${TESTING_PHASE_SCENARIO_DIR}"
  log::info "  2. Create .lighthouse/config.json (see docs/testing/guides/lighthouse-integration.md)"
fi

# Additional performance checks can be added here:
# - Bundle size analysis
# - API response time benchmarks
# - Memory usage profiling

testing::phase::end_with_summary "Performance checks completed"
