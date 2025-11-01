#!/bin/bash
# Phase: Integration
# Validates core API flows, coverage analysis, and vault lifecycle end-to-end

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

set -euo pipefail

testing::phase::init --target-time "240s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

API_PORT=$(vrooli scenario port "${TESTING_PHASE_SCENARIO_NAME}" API_PORT 2>/dev/null || true)
if [ -z "$API_PORT" ]; then
    testing::phase::add_error "test-genie runtime unavailable; start the scenario before running integration tests"
    testing::phase::add_test failed
    testing::phase::end_with_summary "Integration tests incomplete"
fi

API_URL="http://localhost:${API_PORT}"
TEST_WORKDIR="$(mktemp -d -t test-genie-integration-XXXXXX)"
CLI_PATH="test-genie"
SUITE_ID=""
VAULT_ID=""

cleanup() {
    rm -rf "$TEST_WORKDIR"
}

testing::phase::register_cleanup cleanup

if ! command -v jq >/dev/null 2>&1; then
    testing::phase::add_error "jq is required for integration validation"
    testing::phase::add_test failed
    testing::phase::end_with_summary "Integration tests incomplete"
fi

setup_test_environment() {
    rm -rf "$TEST_WORKDIR"
    mkdir -p "$TEST_WORKDIR/src" "$TEST_WORKDIR/api" "$TEST_WORKDIR/ui" "$TEST_WORKDIR/cli"

    cat <<'JS' >"$TEST_WORKDIR/src/main.js"
console.log('test scenario main');
JS
    cat <<'JS' >"$TEST_WORKDIR/src/calculator.js"
export function calculate() { return 42; }
JS
    cat <<'JS' >"$TEST_WORKDIR/src/user.js"
export class UserManager { getUser(id) { return { id }; } }
JS
    cat <<'GO' >"$TEST_WORKDIR/api/main.go"
package main

func Add(a, b int) int { return a + b }
GO
    cat <<'HTML' >"$TEST_WORKDIR/ui/index.html"
<html><body>Test App</body></html>
HTML
    cat <<'SH' >"$TEST_WORKDIR/cli/app"
#!/bin/bash
echo "Test CLI"
SH
    chmod +x "$TEST_WORKDIR/cli/app"
}

run_step() {
    local description="$1"
    shift

    if "$@"; then
        log::success "✅ ${description}"
        testing::phase::add_test passed
        return 0
    else
        log::error "❌ ${description}"
        testing::phase::add_error "${description}"
        testing::phase::add_test failed
        return 1
    fi
}

test_api_health() {
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/health" || true)
    [ "$code" = "200" ]
}

test_suite_generation() {
    local response
    response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
        -H "Content-Type: application/json" \
        -d '{
            "scenario_name": "integration-demo",
            "test_types": ["unit", "integration"],
            "coverage_target": 80,
            "options": {
                "include_performance_tests": true,
                "include_security_tests": false,
                "execution_timeout": 300
            }
        }')

    SUITE_ID=$(echo "$response" | jq -r '.suite_id // empty' 2>/dev/null)
    if [ -z "$SUITE_ID" ]; then
        log::error "   Payload: $response"
        return 1
    fi

    log::info "   Generated suite ID: $SUITE_ID"
    log::info "   Tests generated: $(echo "$response" | jq -r '.generated_tests' 2>/dev/null)"
    return 0
}

test_suite_details() {
    local response
    response=$(curl -s "$API_URL/api/v1/test-suite/$SUITE_ID" || true)
    echo "$response" | jq -e '.id == $id' --arg id "$SUITE_ID" >/dev/null 2>&1
}

test_coverage_analysis() {
    local response overall priority
    local payload
    payload=$(jq -n --arg path "$TEST_WORKDIR/src" '{
        scenario_name: "integration-demo",
        source_code_paths: [$path],
        existing_test_paths: [],
        analysis_depth: "comprehensive"
    }')
    response=$(curl -s -X POST "$API_URL/api/v1/test-analysis/coverage" \
        -H "Content-Type: application/json" \
        -d "$payload")
    overall=$(echo "$response" | jq -r '.overall_coverage // empty' 2>/dev/null)
    if [ -z "$overall" ]; then
        log::error "   Payload: $response"
        return 1
    fi
    priority=$(echo "$response" | jq -r '.priority_areas | length' 2>/dev/null)
    log::info "   Overall coverage: ${overall}% (priority areas: ${priority})"
    return 0
}

test_vault_creation() {
    local response
    response=$(curl -s -X POST "$API_URL/api/v1/test-vault/create" \
        -H "Content-Type: application/json" \
        -d '{
            "scenario_name": "integration-demo",
            "vault_name": "integration-vault",
            "phases": ["setup", "develop", "test", "cleanup"],
            "phase_configurations": {
                "setup": {"timeout": 300},
                "test": {"timeout": 600, "parallel": true}
            },
            "success_criteria": {
                "min_test_pass_rate": 0.8,
                "max_execution_time": 1800
            }
        }')

    VAULT_ID=$(echo "$response" | jq -r '.vault_id // empty' 2>/dev/null)
    if [ -z "$VAULT_ID" ]; then
        log::error "   Payload: $response"
        return 1
    fi

    log::info "   Created vault ID: $VAULT_ID"
    return 0
}

test_system_status() {
    local response status
    response=$(curl -s "$API_URL/api/v1/system/status" || true)
    status=$(echo "$response" | jq -r '.status // empty' 2>/dev/null)
    if [ "$status" != "" ]; then
        log::info "   System reported status: $status"
        return 0
    fi

    log::error "   Payload: $response"
    return 1
}

test_cli_integration() {
    if ! command -v "$CLI_PATH" >/dev/null 2>&1; then
        testing::phase::add_warning "CLI binary not installed; skipping CLI integration checks"
        return 0
    fi

    if ! "$CLI_PATH" --help >/dev/null 2>&1; then
        log::error "   CLI help command failed"
        return 1
    fi

    if ! "$CLI_PATH" generate integration-demo --types unit --dry-run >/dev/null 2>&1; then
        log::error "   CLI generate command failed"
        return 1
    fi

    if ! "$CLI_PATH" status --json | jq -e '.status' >/dev/null 2>&1; then
        log::error "   CLI status json output invalid"
        return 1
    fi

    return 0
}

test_basic_load() {
    local start end duration
    start=$(date +%s)
    for _ in {1..5}; do
        curl -s "$API_URL/health" >/dev/null &
    done
    wait
    end=$(date +%s)
    duration=$((end - start))
    log::info "   Parallel health probes completed in ${duration}s"
    [ "$duration" -lt 10 ]
}

run_step "Test environment prepared" setup_test_environment
run_step "API health endpoint responds" test_api_health
run_step "Suite generation API returns an identifier" test_suite_generation
if [ -n "$SUITE_ID" ]; then
    run_step "Generated suite retrievable" test_suite_details
else
    testing::phase::add_warning "previous suite generation failed; skipping detail validation"
    testing::phase::add_test skipped
fi
run_step "Coverage analysis endpoint responds" test_coverage_analysis
run_step "Vault creation succeeds" test_vault_creation
run_step "System status endpoint responds" test_system_status
if command -v "$CLI_PATH" >/dev/null 2>&1; then
    run_step "CLI commands execute" test_cli_integration
else
    testing::phase::add_warning "test-genie CLI missing from PATH; CLI integration skipped"
    testing::phase::add_test skipped
fi
run_step "Health endpoint handles basic load" test_basic_load

testing::phase::end_with_summary "Integration validation completed"
