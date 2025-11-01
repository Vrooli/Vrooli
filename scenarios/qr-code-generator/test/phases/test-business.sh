#!/bin/bash
# Validates end-to-end QR generation workflows exposed to operators.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s" --require-runtime

CLI_BIN="${TESTING_PHASE_SCENARIO_DIR}/cli/qr-generator"
TMP_FILE=""

cleanup_tmp_artifacts() {
  if [ -n "$TMP_FILE" ] && [ -f "$TMP_FILE" ]; then
    rm -f "$TMP_FILE"
  fi
}

testing::phase::register_cleanup cleanup_tmp_artifacts

if [ -f "$CLI_BIN" ]; then
  TMP_FILE="$(mktemp -t qr-generator-business-XXXXXX).png"
  if bash "$CLI_BIN" generate "Business workflow test" --output "$TMP_FILE" --size 512 --format png --style retro; then
    if [ -f "$TMP_FILE" ]; then
      testing::phase::add_test passed
    else
      testing::phase::add_error "Business workflow did not produce expected output file"
      testing::phase::add_test failed
    fi
  else
    testing::phase::add_error "Business workflow CLI command failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "CLI script not found; skipping business workflow validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflow validation completed"
