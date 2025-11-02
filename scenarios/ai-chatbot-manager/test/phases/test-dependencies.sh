#!/bin/bash
# Ensure language runtimes and package manifests are healthy

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

# Core toolchain availability
for bin in go node npm; do
  testing::phase::check "binary available: ${bin}" command -v "$bin"
done

# Validate Go module graph without downloading if module exists
if [ -f "api/go.mod" ]; then
  if command -v go >/dev/null 2>&1; then
    testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
  else
    testing::phase::add_warning "Go toolchain missing; skipping Go dependency check"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "api/go.mod not found; skipping Go dependency check"
  testing::phase::add_test skipped
fi

# Validate UI dependencies via npm dry-run
if [ -f "ui/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
  else
    testing::phase::add_warning "npm CLI unavailable; cannot verify UI dependencies"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "ui/package.json not found; skipping Node dependency check"
  testing::phase::add_test skipped
fi

# Confirm key manifest files remain present
if testing::phase::check_files .vrooli/service.json README.md Makefile; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Validate service.json structure when jq is available
if command -v jq >/dev/null 2>&1; then
  testing::phase::check "service.json parses" jq . .vrooli/service.json >/dev/null
else
  testing::phase::add_warning "jq not available; skipping service.json lint"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
