#!/bin/bash
# Structure validation phase for travel-map-filler.
# Ensures key directories/files and configuration stay present and well-formed.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "🏗️  Validating project structure"

required_dirs=(
  ".vrooli"
  "api"
  "cli"
  "initialization"
  "test/phases"
  "ui"
)
missing_dirs=()
for dir in "${required_dirs[@]}"; do
  if [ -d "$dir" ]; then
    log::success "✅ Directory present: $dir"
  else
    log::error "❌ Missing directory: $dir"
    missing_dirs+=("$dir")
  fi
done
if [ ${#missing_dirs[@]} -eq 0 ]; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Required directories missing: ${missing_dirs[*]}"
  testing::phase::add_test failed
fi

required_files=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "api/main.go"
  "api/main_test.go"
  "api/test_helpers.go"
  "api/test_patterns.go"
  "test/phases/test-unit.sh"
  "test/phases/test-integration.sh"
)
missing_files=()
for file in "${required_files[@]}"; do
  if [ -f "$file" ]; then
    log::success "✅ File present: $file"
  else
    log::error "❌ Missing file: $file"
    missing_files+=("$file")
  fi
done
if [ ${#missing_files[@]} -eq 0 ]; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Required files missing: ${missing_files[*]}"
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1; then
  if jq empty .vrooli/service.json >/dev/null 2>&1; then
    log::success "✅ .vrooli/service.json is valid JSON"
    testing::phase::add_test passed
  else
    log::error "❌ .vrooli/service.json is invalid JSON"
    testing::phase::add_error ".vrooli/service.json failed jq validation"
    testing::phase::add_test failed
  fi
else
  log::warning "ℹ️  jq not available; skipping service.json validation"
  testing::phase::add_warning "jq not installed"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
