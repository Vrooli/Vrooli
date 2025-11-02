#!/bin/bash
# Dependency validation phase for knowledge-observatory
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

# Go module verification
if testing::phase::check "Go module graph resolves" bash -c 'cd api && go mod verify >/dev/null'; then
  :
else
  testing::phase::end_with_summary "Go dependency verification failed"
fi

# Optional lint warning
if command -v golangci-lint >/dev/null 2>&1; then
  if testing::phase::check "golangci-lint baseline" bash -c 'cd api && golangci-lint run >/dev/null'; then
    :
  else
    testing::phase::add_warning "golangci-lint reported issues"
  fi
else
  testing::phase::add_warning "golangci-lint not installed; skipping Go lint"
fi

# Node / UI dependencies
if [ -f ui/package.json ]; then
  if command -v npm >/dev/null 2>&1; then
    if testing::phase::check "UI dependency manifest is valid" bash -c 'cd ui && npm pkg get name >/dev/null'; then
      :
    else
      testing::phase::add_warning "Unable to read ui/package.json via npm"
    fi
  else
    testing::phase::add_warning "npm not available; cannot validate UI dependencies"
  fi
fi

# CLI binary existence
if testing::phase::check "CLI binary exists" bash -c '[ -x cli/knowledge-observatory ]'; then
  :
else
  testing::phase::add_warning "CLI binary missing or not executable"
fi

# Resource availability snapshot (best-effort)
if command -v vrooli >/dev/null 2>&1; then
  for resource in qdrant postgres; do
    if testing::phase::check "${resource} resource status" bash -c "vrooli resource status $resource --json >/dev/null"; then
      :
    else
      testing::phase::add_warning "Unable to verify $resource resource"
    fi
  done
else
  testing::phase::add_warning "vrooli CLI not available; skipping resource status checks"
fi

# Ensure Makefile exposes lifecycle targets
for target in start run stop test logs status; do
  testing::phase::check "Makefile target '$target' exists" bash -c "grep -Eq '^${target}:' Makefile"
done


testing::phase::end_with_summary "Dependency validation completed"
