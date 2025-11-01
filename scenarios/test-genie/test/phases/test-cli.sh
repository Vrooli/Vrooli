#!/bin/bash
# Phase: CLI Testing
# Validates test-genie CLI commands work correctly

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

set -euo pipefail

testing::phase::init --target-time "60s"

CLI_BINARY="test-genie"

if ! command -v "$CLI_BINARY" >/dev/null 2>&1; then
    testing::phase::add_warning "test-genie CLI executable not found on PATH"
    testing::phase::add_test skipped
    testing::phase::end_with_summary "CLI validation skipped"
fi

if ! command -v jq >/dev/null 2>&1; then
    testing::phase::add_warning "jq not available; cannot validate JSON output"
    testing::phase::add_test skipped
    testing::phase::end_with_summary "CLI validation incomplete"
fi

run_cli_test() {
    local description="$1"
    local command="$2"
    local expected="$3"

    echo "🔍 $description"
    if output=$(eval "$command" 2>&1); then
        if echo "$output" | grep -q "$expected"; then
            log::success "✅ $description"
            testing::phase::add_test passed
            return 0
        fi
    fi

    log::error "❌ $description"
    log::error "   Command: $command"
    log::error "   Output: $output"
    testing::phase::add_error "$description"
    testing::phase::add_test failed
    return 1
}

run_cli_test "CLI help command" "$CLI_BINARY --help" "AI-powered comprehensive test generation"
run_cli_test "CLI status command" "$CLI_BINARY status" "Test Genie System Status"
run_cli_test "CLI health command" "$CLI_BINARY health" "Test Genie is healthy"
run_cli_test "CLI generate command" "$CLI_BINARY generate test-cli-demo --types unit --coverage 80 --json" "request_id"
run_cli_test "CLI coverage analysis" "timeout 5 $CLI_BINARY coverage test-cli-demo --depth basic 2>&1 | head -1" "Analyzing coverage"
run_cli_test "CLI vault help" "$CLI_BINARY vault --help 2>&1 || true" "vault"
run_cli_test "CLI maintain help" "$CLI_BINARY maintain --help 2>&1 || true" "maintain"
run_cli_test "CLI execute help" "$CLI_BINARY execute --help 2>&1 || true" "execute"
run_cli_test "CLI results help" "$CLI_BINARY results --help 2>&1 || true" "results"
run_cli_test "CLI invalid command handling" "$CLI_BINARY invalid-command 2>&1 || true" "Unknown command"
run_cli_test "CLI JSON output formatting" "$CLI_BINARY status --json | jq -e '.status' >/dev/null && echo 'valid json'" "valid json"

testing::phase::end_with_summary "CLI validation completed"
