#!/bin/bash
# Ensures language and tooling dependencies are available for smart-shopping-assistant.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Client tooling checks (warnings if missing).
if command -v psql >/dev/null 2>&1; then
  testing::phase::add_test passed
  log::success "PostgreSQL client available"
else
  testing::phase::add_warning "psql client not found"
  testing::phase::add_test skipped
fi

if command -v redis-cli >/dev/null 2>&1; then
  testing::phase::add_test passed
  log::success "redis-cli available"
else
  testing::phase::add_warning "redis-cli not found"
  testing::phase::add_test skipped
fi

# Go module verification.
if [[ -f "api/go.mod" ]]; then
  testing::phase::check "Go module graph verifies" bash -c 'cd api && go mod verify'
else
  testing::phase::add_warning "No Go module file present"
  testing::phase::add_test skipped
fi

# Node dependency sanity (ensure manifest is readable when Node is present).
if [[ -f "ui/package.json" ]]; then
  if command -v node >/dev/null 2>&1; then
    testing::phase::check "UI package manifest parses" node -e 'require("./ui/package.json")'
  else
    testing::phase::add_warning "Node runtime missing; skipping UI manifest validation"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "UI package.json missing"
  testing::phase::add_test skipped
fi

# Optional integration scenarios (informational only).
for optional in scenario-authenticator deep-research contact-book; do
  if [[ -f "../${optional}/.vrooli/service.json" ]]; then
    log::info "Optional integration available: ${optional}"
  fi
done

testing::phase::end_with_summary "Dependency validation completed"
