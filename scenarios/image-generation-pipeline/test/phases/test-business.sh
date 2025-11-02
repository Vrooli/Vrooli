#!/bin/bash
# Validates business-critical workflows: CLI orchestration and initialization assets.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
CLI_SCRIPT="${TESTING_PHASE_SCENARIO_DIR}/cli/image-generation-pipeline-cli.sh"

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_warning "Unable to resolve API_PORT; skipping CLI workflow validation"
  testing::phase::add_test skipped
else
  if [ -f "$CLI_SCRIPT" ]; then
    if command -v bash >/dev/null 2>&1; then
      testing::phase::check "CLI status command" bash -c "API_PORT=$API_PORT IMAGE_GENERATION_PIPELINE_TOKEN=test-token bash '$CLI_SCRIPT' status"
      testing::phase::check "CLI campaign listing" bash -c "API_PORT=$API_PORT IMAGE_GENERATION_PIPELINE_TOKEN=test-token bash '$CLI_SCRIPT' campaigns"
    else
      testing::phase::add_warning "bash shell not available for CLI execution"
      testing::phase::add_test skipped
    fi
  else
    testing::phase::add_warning "CLI script missing; skipping CLI validation"
    testing::phase::add_test skipped
  fi
fi

# Ensure core initialization assets exist to keep business workflows reproducible.
if testing::phase::check_files \
  initialization/workflows/comfyui/brand-workflow.json \
  initialization/workflows/comfyui/qc-workflow.json \
  initialization/automation/n8n/main-workflow.json; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Business workflow validation completed"
