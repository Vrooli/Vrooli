#!/bin/bash
# Structure validation for quiz-generator scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

REQUIRED_FILES=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "Makefile"
  "api/main.go"
  "api/go.mod"
  "cli/quiz-generator"
  "cli/install.sh"
  "initialization/storage/postgres/schema.sql"
  "ui/package.json"
  "ui/src/main.tsx"
  "test/run-tests.sh"
)

REQUIRED_DIRS=(
  "api"
  "cli"
  "ui"
  "initialization"
  "initialization/storage/postgres"
  "test"
  "test/phases"
)

if testing::phase::check_files "${REQUIRED_FILES[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if testing::phase::check_directories "${REQUIRED_DIRS[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1; then
  if testing::phase::check "service.json is valid" jq empty .vrooli/service.json; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
  if testing::phase::check "service.name present" jq -e '.service.name' .vrooli/service.json; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "jq not available; skipping service.json validation"
  testing::phase::add_test skipped
fi

if testing::phase::check "CLI binary present" test -f "cli/quiz-generator"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if testing::phase::check "API binary executable" test -x "api/quiz-generator-api"; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "API binary missing or not executable; run setup to build"
  testing::phase::add_test skipped
fi

if command -v quiz-generator >/dev/null 2>&1; then
  if testing::phase::check "CLI --help command" quiz-generator --help >/dev/null; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "quiz-generator CLI not on PATH"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
