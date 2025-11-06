#!/bin/bash
# Performance validation including Lighthouse audits, load tests, and benchmarks.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime
cd "$TESTING_PHASE_SCENARIO_DIR"

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
  log::info "To enable: Create .lighthouse/config.json or run: lighthouse::init_scenario ."
fi

# Additional performance tests (wrk, k6, JMeter, custom benchmarks)
# Populate this block with load/latency regression checks as needed
# Example:
# if command -v wrk >/dev/null 2>&1; then
#   log::info "Running wrk load test..."
#   wrk -t4 -c100 -d30s "http://localhost:${UI_PORT}/" > test/artifacts/wrk-results.txt
#   testing::phase::add_requirement --id "REQ-LOAD-TEST" --status passed --evidence "wrk load test"
# fi

testing::phase::end_with_summary "Performance checks completed"
