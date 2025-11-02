#!/bin/bash
# Verify language dependencies and critical JSON assets

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

if [ -f "api/go.mod" ]; then
  testing::phase::check "Go modules resolve" bash -c 'cd api && go list ./... >/dev/null'
else
  testing::phase::add_warning "Go module not found; skipping Go dependency check"
  testing::phase::add_test skipped
fi

if [ -f "ui/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run --ignore-scripts --silent >/dev/null'
  else
    testing::phase::add_warning "npm CLI missing; unable to verify Node dependencies"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "No ui/package.json present; skipping Node dependency check"
  testing::phase::add_test skipped
fi

if command -v jq >/dev/null 2>&1; then
  shopt -s nullglob
  workflow_files=(initialization/n8n/*.json)
  if [ ${#workflow_files[@]} -eq 0 ]; then
    testing::phase::add_warning "No n8n workflows found for validation"
    testing::phase::add_test skipped
  else
    for workflow in "${workflow_files[@]}"; do
      testing::phase::check "Workflow JSON valid: $(basename "$workflow")" jq -e '.nodes | length > 0' "$workflow"
    done
  fi
  shopt -u nullglob
else
  testing::phase::add_warning "jq not available; skipping workflow validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
