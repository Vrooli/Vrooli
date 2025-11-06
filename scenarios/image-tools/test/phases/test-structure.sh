#!/bin/bash
# Structure phase – validates required files and directories for image-tools.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_files=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "api/main.go"
  "api/go.mod"
  "api/storage.go"
  "cli/image-tools"
  "cli/install.sh"
  "ui/src/index.html"
  "ui/src/app.js"
  "ui/src/styles.css"
  "ui/server.js"
  "Makefile"
)

if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_dirs=(
  "api"
  "api/plugins"
  "api/plugins/jpeg"
  "api/plugins/png"
  "api/plugins/webp"
  "api/plugins/svg"
  "cli"
  "ui"
  "initialization"
  "test"
  "test/phases"
)

if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structure validation completed"
