#!/bin/bash
# Ensures language runtimes and package manifests resolve without modifying state.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

SCENARIO_DIR="$TESTING_PHASE_SCENARIO_DIR"

# Go module graph validation
if [ -f "$SCENARIO_DIR/api/go.mod" ]; then
  if command -v go >/dev/null 2>&1; then
    testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
  else
    testing::phase::add_warning "Go toolchain missing; skipping Go dependency check"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "Go module file missing; skipping Go dependency check"
  testing::phase::add_test skipped
fi

# Node.js dependency dry run (supports npm or pnpm based on packageManager field)
if [ -f "$SCENARIO_DIR/ui/package.json" ]; then
  package_manager="$(jq -r '.packageManager // ""' "$SCENARIO_DIR/ui/package.json" 2>/dev/null)"
  if [[ "$package_manager" == pnpm@* ]]; then
    if command -v pnpm >/dev/null 2>&1; then
      testing::phase::check "pnpm install --dry-run" bash -c 'cd ui && pnpm install --lockfile-only >/dev/null'
    else
      testing::phase::add_warning "pnpm not available; skipping Node dependency check"
      testing::phase::add_test skipped
    fi
  else
    if command -v npm >/dev/null 2>&1; then
      testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
    else
      testing::phase::add_warning "npm CLI missing; skipping Node dependency check"
      testing::phase::add_test skipped
    fi
  fi
else
  testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
  testing::phase::add_test skipped
fi

# CLI bats availability (informational)
if [ -f "$SCENARIO_DIR/cli/secure-document-processing.bats" ]; then
  if command -v bats >/dev/null 2>&1; then
    testing::phase::check "BATS CLI available" bats --version >/dev/null
  else
    testing::phase::add_warning "BATS CLI missing; CLI tests may be skipped"
    testing::phase::add_test skipped
  fi
fi

# Enumerate enabled resources so integration can verify them later
if [ -f "$SCENARIO_DIR/.vrooli/service.json" ]; then
  required_resources=$(jq -r '.resources | to_entries[] | select(.value.required == true or .value.enabled == true) | .key' "$SCENARIO_DIR/.vrooli/service.json" 2>/dev/null || true)
  if [ -n "$required_resources" ]; then
    echo "🔗 Declared resources:" >&2
    while IFS= read -r resource; do
      [ -z "$resource" ] && continue
      echo "   • $resource" >&2
    done <<< "$required_resources"
  fi
fi

testing::phase::end_with_summary "Dependency validation completed"
