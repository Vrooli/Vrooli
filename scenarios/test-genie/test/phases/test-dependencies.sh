#!/bin/bash
# Phase: Dependencies
# Exercises database connectivity, persistence, and concurrent access guarantees

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

set -euo pipefail

testing::phase::init --target-time "150s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

API_PORT=$(vrooli scenario port "${TESTING_PHASE_SCENARIO_NAME}" API_PORT 2>/dev/null || true)
if [ -z "$API_PORT" ]; then
    testing::phase::add_error "test-genie runtime unavailable; start the scenario before running dependency tests"
    testing::phase::add_test failed
    testing::phase::end_with_summary "Dependency validation incomplete"
fi

API_URL="http://localhost:${API_PORT}"
LAST_CREATED_SUITE=""

if ! command -v jq >/dev/null 2>&1; then
    testing::phase::add_error "jq is required for dependency validation"
    testing::phase::add_test failed
    testing::phase::end_with_summary "Dependency validation incomplete"
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

test_database_health() {
    local response status
    response=$(curl -s "$API_URL/health" || true)
    status=$(echo "$response" | jq -r '.database.status // empty' 2>/dev/null)
    if [ "$status" = "healthy" ]; then
        log::info "   Database status reported healthy"
        return 0
    fi
    log::error "   Health payload: $response"
    return 1
}

test_suite_persistence() {
    local create_response suite_id retrieve_response retrieved_id
    create_response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
        -H "Content-Type: application/json" \
        -d '{
            "scenario_name": "db-persistence-test",
            "test_types": ["unit"],
            "coverage_target": 75,
            "options": {"execution_timeout": 120}
        }')

    suite_id=$(echo "$create_response" | jq -r '.suite_id // empty' 2>/dev/null)
    if [ -z "$suite_id" ]; then
        log::error "   Creation payload: $create_response"
        return 1
    fi

    retrieve_response=$(curl -s "$API_URL/api/v1/test-suite/$suite_id" || true)
    retrieved_id=$(echo "$retrieve_response" | jq -r '.id // empty' 2>/dev/null)

    if [ "$retrieved_id" = "$suite_id" ]; then
        log::info "   Persisted suite accessible (ID: $suite_id)"
        LAST_CREATED_SUITE="$suite_id"
        return 0
    fi

    log::error "   Retrieval payload: $retrieve_response"
    return 1
}

test_listing_contains_suite() {
    local list_response
    if [ -z "$LAST_CREATED_SUITE" ]; then
        log::warning "   No suite created earlier; skipping listing assertion"
        testing::phase::add_warning "suite listing skipped"
        return 0
    fi

    list_response=$(curl -s "$API_URL/api/v1/test-suites" || true)
    echo "$list_response" | jq -e --arg id "$LAST_CREATED_SUITE" '(.test_suites // []) | map(.id) | index($id) != null' >/dev/null 2>&1
    if [ $? -eq 0 ]; then
        log::info "   Recently created suite present in listing"
        return 0
    fi

    log::error "   Listing payload: $list_response"
    return 1
}

test_coverage_analysis() {
    local coverage_response overall
    coverage_response=$(curl -s -X POST "$API_URL/api/v1/test-analysis/coverage" \
        -H "Content-Type: application/json" \
        -d '{
            "scenario_name": "db-query-test",
            "source_code_paths": ["/tmp/test"],
            "existing_test_paths": [],
            "analysis_depth": "detailed"
        }')

    overall=$(echo "$coverage_response" | jq -r '.overall_coverage // empty' 2>/dev/null)
    if [ -n "$overall" ]; then
        log::info "   Coverage endpoint responded (${overall}% reported)"
        return 0
    fi

    log::error "   Coverage payload: $coverage_response"
    return 1
}

test_vault_integrity() {
    local vault_response vault_id vault_details
    vault_response=$(curl -s -X POST "$API_URL/api/v1/test-vault/create" \
        -H "Content-Type: application/json" \
        -d '{
            "scenario_name": "db-transaction-test",
            "vault_name": "integrity-test-vault",
            "phases": ["setup", "test", "cleanup"],
            "phase_configurations": {
                "setup": {"timeout": 300},
                "test": {"timeout": 600},
                "cleanup": {"timeout": 300}
            },
            "success_criteria": {
                "min_test_pass_rate": 0.9,
                "max_execution_time": 1200
            }
        }')

    vault_id=$(echo "$vault_response" | jq -r '.vault_id // empty' 2>/dev/null)
    if [ -z "$vault_id" ]; then
        log::error "   Vault creation payload: $vault_response"
        return 1
    fi

    vault_details=$(curl -s "$API_URL/api/v1/test-vault/$vault_id" || true)
    if echo "$vault_details" | jq -e '.id == $vault_id' --arg vault_id "$vault_id" >/dev/null 2>&1; then
        log::info "   Vault stored with ID $vault_id"
        return 0
    fi

    log::error "   Vault retrieval payload: $vault_details"
    return 1
}

test_concurrent_writes() {
    local failure=0
    local pids=()

    for i in {1..5}; do
        (
            response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
                -H "Content-Type: application/json" \
                -d "{\"scenario_name\":\"concurrent-db-test-$i\",\"test_types\":[\"unit\"],\"coverage_target\":50,\"options\":{\"execution_timeout\":60}}")
            if ! echo "$response" | jq -e '.suite_id' >/dev/null 2>&1; then
                echo "Concurrent DB operation $i failed: $response" >&2
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

test_database_performance() {
    local start_time end_time duration_ms
    start_time=$(date +%s%N)

    for _ in {1..10}; do
        curl -s "$API_URL/health" >/dev/null &
    done
    wait

    end_time=$(date +%s%N)
    duration_ms=$(( (end_time - start_time) / 1000000 ))
    log::info "   10 health checks completed in ${duration_ms}ms"
    [ "$duration_ms" -lt 5000 ]
}

run_step "Database connection healthy" test_database_health
run_step "Suite persistence via API" test_suite_persistence
run_step "Suite listing contains latest entry" test_listing_contains_suite
run_step "Coverage analysis endpoint responds" test_coverage_analysis
run_step "Vault transactions succeed" test_vault_integrity
run_step "Concurrent suite creation succeeds" test_concurrent_writes
run_step "Health endpoint responds within 5s aggregated" test_database_performance

testing::phase::end_with_summary "Dependency validation completed"
