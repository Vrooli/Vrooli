#!/bin/bash
# Validates required files and directories for the scenario shell.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

SCENARIO_DIR="$TESTING_PHASE_SCENARIO_DIR"

required_dirs=(
  api
  ui
  cli
  initialization
  initialization/automation
  initialization/automation/n8n
  initialization/configuration
  initialization/storage
  data
  .vrooli
  test
)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  README.md
  PRD.md
  IMPLEMENTATION_PLAN.md
  MIGRATION_SUMMARY.md
  api/go.mod
  api/main.go
  cli/install.sh
  cli/secure-document-processing
  initialization/storage/postgres/schema.sql
  initialization/storage/postgres/seed.sql
  initialization/automation/n8n/document-intake.json
  initialization/automation/n8n/process-orchestrator.json
  initialization/automation/n8n/prompt-processor.json
  initialization/automation/n8n/workflow-executor.json
  initialization/automation/n8n/semantic-indexer.json
  initialization/automation/windmill/document-portal.json
  initialization/configuration/encryption-config.json
  custom-tests.sh
  test/run-tests.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

optional_files=(
  initialization/configuration/security-policies.json
  initialization/configuration/compliance-config.json
)
for file in "${optional_files[@]}"; do
  if [ ! -f "$file" ]; then
    testing::phase::add_warning "Optional supporting file missing: $file"
  fi
done

testing::phase::end_with_summary "Structure validation completed"
