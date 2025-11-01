#!/usr/bin/env bash
# Ensures required resources, toolchains, and package manifests are available.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

# Resource availability
for resource in postgres ollama n8n; do
  testing::phase::check "Resource available: ${resource}" vrooli resource status "$resource"
done

if ! vrooli resource status qdrant >/dev/null 2>&1; then
  testing::phase::add_warning "Qdrant vector database not available; similarity search tests will be skipped"
else
  testing::phase::check "Optional resource available: qdrant" vrooli resource status qdrant
fi

# Tooling
for command in go node npm jq curl bats; do
  if ! testing::phase::check "Command available: ${command}" command -v "$command"; then
    testing::phase::add_warning "${command} not on PATH"
  fi
done

# Go module graph
if [ -f "api/go.mod" ]; then
  testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
else
  testing::phase::add_warning "Go module definition missing"
  testing::phase::add_test skipped
fi

# Node dependencies (dry-run install to verify manifests)
if [ -f "ui/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
  else
    testing::phase::add_warning "npm missing; cannot verify UI dependencies"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "UI package.json missing"
  testing::phase::add_test skipped
fi

# CLI binary sanity check
testing::phase::check "CLI binary present" test -x "cli/prompt-injection-arena"

testing::phase::end_with_summary "Dependency validation completed"
