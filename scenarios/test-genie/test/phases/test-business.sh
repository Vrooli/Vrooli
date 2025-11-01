#!/bin/bash
# Phase: Business Logic & Delegated Generation
# Validates the delegation pipeline, placeholder persistence, and option handling

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

set -euo pipefail

testing::phase::init --target-time "180s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

API_PORT=$(vrooli scenario port "${TESTING_PHASE_SCENARIO_NAME}" API_PORT 2>/dev/null || true)
if [ -z "$API_PORT" ]; then
    testing::phase::add_error "test-genie runtime unavailable; start the scenario before running business tests"
    testing::phase::add_test failed
    testing::phase::end_with_summary "Business validation incomplete"
fi

API_URL="http://localhost:${API_PORT}"
ISSUE_PORT=$(vrooli scenario port app-issue-tracker API_PORT 2>/dev/null || true)
LAST_REQUEST_ID=""

if ! command -v jq >/dev/null 2>&1; then
    testing::phase::add_error "jq is required for business validation"
    testing::phase::add_test failed
    testing::phase::end_with_summary "Business validation incomplete"
fi

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

check_issue_tracker_health() {
    if [ -z "$ISSUE_PORT" ]; then
        testing::phase::add_warning "App Issue Tracker port not resolved; delegation will use local fallback"
        testing::phase::add_test skipped
        return
    fi

    local response
    response=$(curl -s "http://localhost:${ISSUE_PORT}/health" || true)
    if echo "$response" | jq -e '.success == true' >/dev/null 2>&1; then
        run_step "App Issue Tracker health endpoint responds" true
    else
        testing::phase::add_warning "App Issue Tracker responded unexpectedly; proceeding with local fallback"
        testing::phase::add_test skipped
    fi
}

submit_generation_request() {
    local payload="$1"
    curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
        -H "Content-Type: application/json" \
        -d "$payload"
}

test_delegated_submission() {
    local payload response status request_id
    payload='{
        "scenario_name": "calculator-app",
        "test_types": ["unit"],
        "coverage_target": 90,
        "options": {
            "include_performance_tests": false,
            "include_security_tests": false,
            "custom_test_patterns": ["edge-cases"],
            "execution_timeout": 120
        }
    }'
    response=$(submit_generation_request "$payload")
    status=$(echo "$response" | jq -r '.status // empty' 2>/dev/null)
    request_id=$(echo "$response" | jq -r '.request_id // empty' 2>/dev/null)

    if [[ "$status" =~ ^(submitted|generated_locally)$ ]] && [ -n "$request_id" ]; then
        LAST_REQUEST_ID="$request_id"
        log::info "   Request recorded with status '$status'"
        return 0
    fi

    log::error "   Payload: $response"
    return 1
}

test_placeholder_persisted() {
    if [ -z "$LAST_REQUEST_ID" ]; then
        testing::phase::add_warning "Previous submission missing request id; skipping placeholder verification"
        return 0
    fi

    local response status
    response=$(curl -s "$API_URL/api/v1/test-suites?scenario=calculator-app" || true)
    status=$(echo "$response" | jq -r '.test_suites[0].status // empty' 2>/dev/null)
    if [ -n "$status" ]; then
        log::info "   Placeholder status: $status"
        return 0
    fi

    log::error "   listing payload: $response"
    return 1
}

test_performance_option_submission() {
    local payload response status
    payload='{
        "scenario_name": "web-api",
        "test_types": ["performance"],
        "coverage_target": 70,
        "options": {
            "include_performance_tests": true,
            "include_security_tests": false,
            "execution_timeout": 240
        }
    }'
    response=$(submit_generation_request "$payload")
    status=$(echo "$response" | jq -r '.status // empty' 2>/dev/null)
    [[ "$status" =~ ^(submitted|generated_locally)$ ]]
}

test_security_option_submission() {
    local payload response status
    payload='{
        "scenario_name": "user-auth-api",
        "test_types": ["security"],
        "coverage_target": 65,
        "options": {
            "include_performance_tests": false,
            "include_security_tests": true,
            "custom_test_patterns": ["sql-injection", "xss"],
            "execution_timeout": 200
        }
    }'
    response=$(submit_generation_request "$payload")
    status=$(echo "$response" | jq -r '.status // empty' 2>/dev/null)
    [[ "$status" =~ ^(submitted|generated_locally)$ ]]
}

test_concurrent_submissions() {
    local failure=0
    local pids=()

    for i in {1..3}; do
        (
            payload=$(jq -n --arg idx "$i" '{
                scenario_name: ("concurrent-" + $idx),
                test_types: ["unit"],
                coverage_target: 55,
                options: { execution_timeout: 90 }
            }')
            response=$(submit_generation_request "$payload")
            if ! echo "$response" | jq -e '.status' >/dev/null 2>&1; then
                echo "Concurrent request $i failed: $response" >&2
                exit 1
            fi
        ) &
        pids+=($!)
    done

    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            failure=1
        fi
    done

    [ "$failure" -eq 0 ]
}

check_issue_tracker_health
run_step "Delegated generation submission recorded" test_delegated_submission
run_step "Placeholder suite persisted" test_placeholder_persisted
run_step "Performance option submission handled" test_performance_option_submission
run_step "Security option submission handled" test_security_option_submission
run_step "Concurrent submissions succeed" test_concurrent_submissions

testing::phase::end_with_summary "Business delegation validation completed"
