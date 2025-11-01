#!/usr/bin/env bash
# Validates scenario structure, required files, and documentation assets.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

required_dirs=(
  api
  cli
  data
  docs
  initialization
  initialization/postgres
  initialization/n8n
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
  PROBLEMS.md
  Makefile
  api/main.go
  api/go.mod
  initialization/postgres/schema.sql
  initialization/postgres/seed.sql
  initialization/n8n/security-sandbox.json
  initialization/n8n/injection-tester.json
  cli/install.sh
  cli/prompt-injection-arena
  ui/index.html
  ui/package.json
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

documentation_files=(
  docs/api.md
  docs/cli.md
  docs/security.md
)
for doc_path in "${documentation_files[@]}"; do
  if testing::phase::check "Documentation present: ${doc_path}" test -f "$doc_path"; then
    :
  fi
done

testing::phase::end_with_summary "Structure validation completed"
