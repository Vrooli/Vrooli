#!/bin/bash
# Dependency validation for app-issue-tracker
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

if [ -f "api/go.mod" ]; then
  if command -v go >/dev/null 2>&1; then
    if testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'; then
      :
    fi
  else
    testing::phase::add_warning "Go toolchain not available; skipping Go dependency checks"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "Go module not found; skipping Go dependency checks"
  testing::phase::add_test skipped
fi

if [ -f "ui/package.json" ]; then
  package_manager="npm"
  if [ -f "ui/pnpm-lock.yaml" ] || grep -q '"packageManager"\s*:\s*"pnpm@' "ui/package.json" 2>/dev/null; then
    package_manager="pnpm"
  elif [ -f "ui/yarn.lock" ]; then
    package_manager="yarn"
  fi

  case "$package_manager" in
    pnpm)
      if command -v pnpm >/dev/null 2>&1; then
        if [ ! -d "ui/node_modules" ]; then
          testing::phase::check "Install UI deps (pnpm)" bash -c 'cd ui && pnpm install --frozen-lockfile --silent --ignore-scripts'
        else
          testing::phase::check "Validate UI lockfile (pnpm)" bash -c 'cd ui && pnpm install --frozen-lockfile --silent --ignore-scripts --offline'
        fi
      else
        testing::phase::add_warning "pnpm not available; skipping UI dependency install"
        testing::phase::add_test skipped
      fi
      ;;
    yarn)
      if command -v yarn >/dev/null 2>&1; then
        testing::phase::check "Install UI deps (yarn)" bash -c 'cd ui && yarn install --silent --frozen-lockfile'
      else
        testing::phase::add_warning "yarn not available; skipping UI dependency install"
        testing::phase::add_test skipped
      fi
      ;;
    *)
      if command -v npm >/dev/null 2>&1; then
        testing::phase::check "Install UI deps (npm)" bash -c 'cd ui && npm install --silent'
      else
        testing::phase::add_warning "npm not available; skipping UI dependency install"
        testing::phase::add_test skipped
      fi
      ;;
  esac
else
  testing::phase::add_warning "ui/package.json not found; skipping UI dependency checks"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
