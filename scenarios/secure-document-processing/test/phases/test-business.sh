#!/bin/bash
# Business-layer validation: ensure core workflows and artefacts remain wired.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

SCENARIO_DIR="$TESTING_PHASE_SCENARIO_DIR"

key_documents=(
  PRD.md
  IMPLEMENTATION_PLAN.md
  TEST_IMPLEMENTATION_SUMMARY.md
  TEST_COVERAGE_REPORT.md
)
if testing::phase::check_files "${key_documents[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Validate that the n8n workflow blueprints remain syntactically correct.
workflow_files=(
  initialization/automation/n8n/document-intake.json
  initialization/automation/n8n/process-orchestrator.json
  initialization/automation/n8n/prompt-processor.json
  initialization/automation/n8n/workflow-executor.json
  initialization/automation/n8n/semantic-indexer.json
)
if command -v jq >/dev/null 2>&1; then
  for workflow in "${workflow_files[@]}"; do
    testing::phase::check "Workflow JSON valid: $workflow" jq empty "$workflow"
  done
else
  testing::phase::add_warning "jq not available; skipping workflow JSON validation"
  testing::phase::add_test skipped
fi

# Ensure the database schema retains business critical tables.
if [ -f "$SCENARIO_DIR/initialization/storage/postgres/schema.sql" ]; then
  testing::phase::check "Schema contains business tables" \
    bash -c 'grep -E "(documents|jobs|workflows|audit_logs)" initialization/storage/postgres/schema.sql >/dev/null'
else
  testing::phase::add_warning "Postgres schema missing; cannot verify business tables"
  testing::phase::add_test skipped
fi

# Confirm API source still exposes the documented routes.
if [ -f "$SCENARIO_DIR/api/main.go" ]; then
  testing::phase::check "API exposes document workflow handlers" \
    bash -c 'grep -E "(documents|jobs|workflows)" api/main.go >/dev/null'
else
  testing::phase::add_warning "API source missing; skipping endpoint signature check"
  testing::phase::add_test skipped
fi

optional_configs=(
  initialization/configuration/encryption-config.json
  initialization/configuration/security-policies.json
  initialization/configuration/compliance-config.json
)
for cfg in "${optional_configs[@]}"; do
  if [ ! -f "$SCENARIO_DIR/$cfg" ]; then
    testing::phase::add_warning "Optional configuration absent: $cfg"
  fi
done

testing::phase::end_with_summary "Business logic validation completed"
