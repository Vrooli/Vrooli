#!/bin/bash
# Runs build-level and domain validation checks that stitch multiple components together.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s"

cd "$TESTING_PHASE_SCENARIO_DIR"

TEMP_BUILD="$(mktemp -t retro-api-build-XXXXXX)"
cleanup_build() {
  rm -f "$TEMP_BUILD"
}
testing::phase::register_cleanup cleanup_build

if command -v go >/dev/null 2>&1 && [ -f "api/main.go" ]; then
  testing::phase::check "Go API builds" bash -c "cd api && go build -o '$TEMP_BUILD' main.go"
else
  testing::phase::add_warning "Skipping Go build check (toolchain or source missing)"
  testing::phase::add_test skipped
fi

testing::phase::check "CLI exposes help output" bash -c 'cd cli && ./retro-game-launcher --help >/dev/null'

testing::phase::check "Seed SQL includes game fixtures" grep -q "Neon Snake" initialization/storage/postgres/seed.sql

testing::phase::check "Seed SQL defines prompt_templates" grep -q "prompt_templates" initialization/storage/postgres/seed.sql

if [ -d "ui/src" ]; then
  testing::phase::check "UI source contains App entry" bash -c 'cd ui && find src -maxdepth 1 -name "App.*" | grep -q .'
else
  testing::phase::add_warning "UI source directory missing"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
