#!/bin/bash
# Validate essential files and directories for the code-smell scenario shell.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

required_dirs=(
  .vrooli
  api
  cli
  initialization
  initialization/rules
  lib
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
  api/main.go
  api/go.mod
  cli/code-smell
  cli/install.sh
  initialization/rules/default.yaml
  initialization/rules/vrooli-specific.yaml
  lib/rules-engine.js
  package.json
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structure validation completed"
