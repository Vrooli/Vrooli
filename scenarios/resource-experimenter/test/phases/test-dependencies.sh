#!/bin/bash
# Ensures language and resource dependencies are in place for resource-experimenter
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

if [ -f "api/go.mod" ]; then
  testing::phase::check "Go modules download" bash -c 'cd api && go mod download'
  testing::phase::check "Go module graph" bash -c 'cd api && go list ./... >/dev/null'
else
  testing::phase::add_warning "Go module not found; skipping Go dependency checks"
  testing::phase::add_test skipped
fi

if [ -f "ui/package.json" ]; then
  if [ -f "ui/pnpm-lock.yaml" ]; then
    testing::phase::check "pnpm lockfile exists" test -s ui/pnpm-lock.yaml
  elif [ -d "ui/node_modules" ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_warning "UI dependencies not yet installed; run pnpm install or npm install"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "UI package.json not present; skipping Node checks"
  testing::phase::add_test skipped
fi

if command -v jq >/dev/null 2>&1; then
  for resource in postgres claude-code; do
    testing::phase::check "resource declared: ${resource}" jq -e --arg r "$resource" '.resources | tostring | contains($r)' .vrooli/service.json
  done
else
  testing::phase::add_warning "jq not installed; skipping resource declaration validation"
  testing::phase::add_test skipped
fi

if [ -d "initialization/postgres" ]; then
  testing::phase::check "postgres schema present" test -f initialization/postgres/schema.sql
  if [ -f initialization/postgres/seed.sql ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_warning "postgres seed file not found"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_error "initialization/postgres directory missing"
fi

testing::phase::end_with_summary "Dependency validation completed"
