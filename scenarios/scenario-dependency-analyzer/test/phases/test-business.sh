#!/bin/bash
# Validates higher-level business workflows via the CLI facade.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

CLI_PATH="${TESTING_PHASE_SCENARIO_DIR}/cli/scenario-dependency-analyzer"
TMP_DIR=$(mktemp -d -t sda-business-XXXXXX)

cleanup_tmp_dir() {
  rm -rf "${TMP_DIR}"
}

testing::phase::register_cleanup cleanup_tmp_dir

if [ ! -x "$CLI_PATH" ]; then
  testing::phase::add_warning "CLI wrapper unavailable; skipping business workflow checks"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Business validation skipped"
fi

ANALYZE_OUTPUT="${TMP_DIR}/analyze.json"
GRAPH_OUTPUT="${TMP_DIR}/graph.json"

if timeout 180 "$CLI_PATH" analyze all --output json >"${ANALYZE_OUTPUT}" 2>"${TMP_DIR}/analyze.err"; then
  testing::phase::add_test passed
  log::success "Generated full analysis report"
else
  testing::phase::add_error "CLI analyze all failed (see ${TMP_DIR}/analyze.err)"
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1 && [ -s "${ANALYZE_OUTPUT}" ]; then
  if jq -e '.analyses | length > 0' "${ANALYZE_OUTPUT}" >/dev/null 2>&1; then
    testing::phase::add_test passed
    log::success "Analysis output contains dependency entries"
  else
    testing::phase::add_error "Analysis output missing dependency entries"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "jq not available or analysis output empty; skipping JSON validation"
  testing::phase::add_test skipped
fi

if timeout 120 "$CLI_PATH" graph combined --format json >"${GRAPH_OUTPUT}" 2>"${TMP_DIR}/graph.err"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Graph generation failed (see ${TMP_DIR}/graph.err)"
  testing::phase::add_test failed
fi

if timeout 60 "$CLI_PATH" status --json >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_error "CLI status command failed"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Business workflow validation completed"
