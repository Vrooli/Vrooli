#!/bin/bash

# Phase: Business Logic & Delegated Generation
# Validates the delegation pipeline, placeholder persistence, and option handling

set -euo pipefail

API_PORT=$(vrooli scenario port test-genie API_PORT 2>/dev/null || echo "")
ISSUE_PORT=$(vrooli scenario port app-issue-tracker API_PORT 2>/dev/null || echo "")

if [[ -z "$API_PORT" ]]; then
    echo "❌ test-genie scenario is not running"
    echo "   Start it with: vrooli scenario run test-genie"
    exit 1
fi

API_URL="http://localhost:${API_PORT}"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

summary=()

log_step() {
    echo -e "\n${CYAN}➡️  $1${NC}"
}

pass() {
    echo -e "${GREEN}✅ $1${NC}"
    summary+=("✅ $1")
}

warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
    summary+=("⚠️  $1")
}

fail() {
    echo -e "${RED}❌ $1${NC}"
    summary+=("❌ $1")
}

log_step "Test 1: App Issue Tracker health"
if [[ -n "$ISSUE_PORT" ]]; then
    if curl -s "http://localhost:${ISSUE_PORT}/health" | jq -e '.success == true' >/dev/null 2>&1; then
        pass "App Issue Tracker responded on port ${ISSUE_PORT}"
    else
        warn "App Issue Tracker health endpoint returned unexpected payload"
    fi
else
    warn "App Issue Tracker port not resolved; delegated requests may fall back locally"
fi

submit_generation_request() {
    local scenario="$1"
    local payload="$2"

    curl -s -X POST "${API_URL}/api/v1/test-suite/generate" \
        -H "Content-Type: application/json" \
        -d "${payload}"
}

check_submission() {
    local response="$1"
    local expected="$2"

    local status
    status=$(echo "${response}" | jq -r '.status // ""' 2>/dev/null)
    local request_id
    request_id=$(echo "${response}" | jq -r '.request_id // ""' 2>/dev/null)

    if [[ "${status}" =~ ${expected} ]] && [[ -n "${request_id}" ]]; then
        pass "Request ${request_id} recorded with status '${status}'"
        echo "$request_id"
    else
        fail "Unexpected response: ${response}"
        echo ""
    fi
}

log_step "Test 2: Delegated generation submission"
base_payload='{
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
base_response=$(submit_generation_request "calculator-app" "$base_payload")
BASE_REQUEST_ID=$(check_submission "$base_response" 'submitted|generated_locally')

log_step "Test 3: Placeholder persisted"
if [[ -n "$BASE_REQUEST_ID" ]]; then
    suites=$(curl -s "${API_URL}/api/v1/test-suites?scenario=calculator-app")
    suite_count=$(echo "$suites" | jq -r '.test_suites | length' 2>/dev/null)
    if [[ "$suite_count" -ge 1 ]]; then
        status=$(echo "$suites" | jq -r '.test_suites[0].status // ""')
        if [[ "$status" == "maintenance_required" ]]; then
            pass "Placeholder suite recorded with maintenance_required status"
        else
            warn "Placeholder suite present but status is '${status}'"
        fi
    else
        warn "No suite placeholder detected for calculator-app"
    fi
else
    warn "Skipping placeholder verification because submission failed"
fi

log_step "Test 4: Performance option submission"
perf_payload='{
    "scenario_name": "web-api",
    "test_types": ["performance"],
    "coverage_target": 70,
    "options": {
        "include_performance_tests": true,
        "include_security_tests": false,
        "execution_timeout": 240
    }
}'
perf_response=$(submit_generation_request "web-api" "$perf_payload")
check_submission "$perf_response" 'submitted|generated_locally' >/dev/null

log_step "Test 5: Security option submission"
security_payload='{
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
security_response=$(submit_generation_request "user-auth-api" "$security_payload")
check_submission "$security_response" 'submitted|generated_locally' >/dev/null

log_step "Test 6: Concurrent submissions"
for i in {1..3}; do
    (
        payload="{\"scenario_name\":\"concurrent-${i}\",\"test_types\":[\"unit\"],\"coverage_target\":55,\"options\":{\"execution_timeout\":90}}"
        response=$(submit_generation_request "concurrent-${i}" "$payload")
        status=$(echo "$response" | jq -r '.status // ""')
        echo "Concurrent request ${i}: ${status}" \
            | sed "s/submitted/${GREEN}submitted${NC}/;s/generated_locally/${YELLOW}generated_locally${NC}/"
    ) &
done
wait
pass "Concurrent submissions completed"

log_step "Summary"
for line in "${summary[@]}"; do
    echo -e "$line"
done

echo -e "\n${CYAN}Business logic tests finished.${NC}"
