#!/bin/bash
# Integration test phase - uses centralized testing library
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 120-second target for integration tests
testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running integration tests for game-dialog-generator"

# Integration tests require external services (database, ollama, qdrant)
# Currently skipped - will be implemented when dependency injection is added
testing::phase::add_warning "Integration tests require dependency injection refactor; skipping"
testing::phase::add_test skipped

testing::phase::end_with_summary "Integration tests skipped"
