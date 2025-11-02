#!/bin/bash
# Validates core files and directory layout for vrooli-orchestrator.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(
  api
  cli
  data
  initialization
  initialization/postgres
  test
  test/phases
  ui
)

if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  PRD.md
  README.md
  Makefile
  api/main.go
  api/go.mod
  cli/vrooli-orchestrator
  cli/install.sh
  cli/vrooli-orchestrator.bats
  initialization/postgres/schema.sql
  initialization/postgres/seed.sql
  test/run-tests.sh
  test/phases/test-structure.sh
  test/phases/test-dependencies.sh
  test/phases/test-unit.sh
  test/phases/test-integration.sh
  test/phases/test-business.sh
  test/phases/test-performance.sh
)

if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structure validation completed"
