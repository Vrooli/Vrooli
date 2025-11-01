#!/bin/bash
# Validate language dependencies and critical resources
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Go modules must resolve successfully
if [ -f "api/go.mod" ]; then
  testing::phase::check "Go modules resolve" bash -c 'cd api && go list ./... >/dev/null'
else
  testing::phase::add_warning "api/go.mod missing; skipping Go dependency validation"
  testing::phase::add_test skipped
fi

# Optional vulnerability scan
if command -v govulncheck >/dev/null 2>&1; then
  testing::phase::check "govulncheck reports no vulnerabilities" bash -c 'cd api && govulncheck ./... >/dev/null'
else
  testing::phase::add_warning "govulncheck not installed; skipping vulnerability scan"
  testing::phase::add_test skipped
fi

# Ensure required initialization assets exist
init_files=(
  initialization/storage/postgres/schema.sql
  initialization/storage/postgres/seed.sql
)
if testing::phase::check_files "${init_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Verify CLI binary is executable when present
if [ -f "cli/referral-program-generator" ]; then
  testing::phase::check "CLI binary executable" test -x "cli/referral-program-generator"
else
  testing::phase::add_warning "CLI binary not built; run cli/install.sh"
  testing::phase::add_test skipped
fi

# Helper to evaluate resource availability
check_resource() {
  local name="$1"
  local command_str="$2"
  local optional="$3"

  local binary
  binary=$(printf '%s' "$command_str" | awk '{print $1}')

  if ! command -v "$binary" >/dev/null 2>&1; then
    testing::phase::add_warning "$name helper ($binary) not found"
    testing::phase::add_test skipped
    return
  fi

  if bash -c "$command_str" >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    if [ "$optional" = "true" ]; then
      testing::phase::add_warning "$name resource unavailable (optional)"
      testing::phase::add_test skipped
    else
      testing::phase::add_error "$name resource unavailable"
      testing::phase::add_test failed
    fi
  fi
}

check_resource "PostgreSQL" "resource-postgres status" false
check_resource "Scenario Authenticator" "vrooli scenario status scenario-authenticator" true
check_resource "Qdrant" "resource-qdrant status" true
check_resource "Browserless" "resource-browserless status" true
# For scenario dependencies, ensure API env example file exists
if testing::phase::check "Example environment file present" test -f "api/.env.example"; then
  :
fi

testing::phase::end_with_summary "Dependency validation completed"
