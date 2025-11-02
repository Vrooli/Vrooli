#!/bin/bash
# Validate required files and directories for resume-screening-assistant.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

set -euo pipefail

testing::phase::init --target-time "45s"

required_dirs=(
  .vrooli
  api
  cli
  initialization
  initialization/n8n
  initialization/postgres
  initialization/qdrant
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
  TEST_IMPLEMENTATION_SUMMARY.md
  custom-tests.sh
  test/test-search-endpoint.sh
  api/main.go
  api/go.mod
  cli/install.sh
  cli/resume-screening-assistant
  initialization/n8n/job-management-workflow.json
  initialization/n8n/resume-processing-pipeline.json
  initialization/n8n/semantic-search-workflow.json
  initialization/postgres/schema.sql
  initialization/postgres/seed.sql
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structure validation completed"
