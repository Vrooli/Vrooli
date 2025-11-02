#!/bin/bash
# Establishes baseline execution time for key CLI flows.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s" --require-runtime

CLI_PATH="${TESTING_PHASE_SCENARIO_DIR}/cli/scenario-dependency-analyzer"

if [ ! -x "$CLI_PATH" ]; then
  testing::phase::add_warning "CLI wrapper unavailable; skipping performance baseline"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Performance phase skipped"
fi

start_time=$(date +%s)
if timeout 120 "$CLI_PATH" analyze chart-generator --output json >/dev/null 2>&1; then
  end_time=$(date +%s)
  duration=$((end_time - start_time))
  if [ "$duration" -le 60 ]; then
    testing::phase::add_test passed
    log::success "⚡ Analysis completed in ${duration}s"
  else
    testing::phase::add_warning "Analysis completed in ${duration}s (exceeds 60s baseline)"
    testing::phase::add_test passed
  fi
else
  testing::phase::add_error "Timed out running CLI analysis benchmark"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Performance benchmarking completed"
