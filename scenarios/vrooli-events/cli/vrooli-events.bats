#!/usr/bin/env bats
# Integration tests for vrooli-events core API endpoints
# [REQ:API-001] [REQ:API-002] [REQ:API-003]

setup() {
    API_PORT=$(vrooli scenario port vrooli-events API_PORT 2>/dev/null || echo "17654")
    API_URL="http://localhost:${API_PORT}"
}

# ─── Health ─── [REQ:API-003]

@test "health endpoint returns 200" {
    result=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/health")
    [ "$result" = "200" ]
}

@test "health endpoint returns JSON with status and store stats" {
    result=$(curl -s "${API_URL}/health")
    echo "$result" | jq -e '.status' >/dev/null
    echo "$result" | jq -e '.store.totalEvents' >/dev/null
}

# ─── Event Ingest ─── [REQ:API-001]

@test "ingest endpoint accepts valid event" {
    uid="bats-ingest-$(date +%s%N)"
    result=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/v1/events" \
        -H "Content-Type: application/json" \
        -d "{\"eventId\":\"${uid}\",\"eventType\":\"test.integration.v1\",\"sourceScenario\":\"bats-suite\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}")
    [ "$result" = "200" ] || [ "$result" = "201" ] || [ "$result" = "202" ]
}

@test "ingest endpoint rejects empty body" {
    result=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/v1/events" \
        -H "Content-Type: application/json" \
        -d '{}')
    [ "$result" = "400" ]
}

@test "ingest endpoint rejects missing eventType" {
    uid="bats-notype-$(date +%s%N)"
    result=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/v1/events" \
        -H "Content-Type: application/json" \
        -d "{\"eventId\":\"${uid}\",\"sourceScenario\":\"bats-suite\"}")
    [ "$result" = "400" ]
}

# ─── Event Query ─── [REQ:API-002]

@test "query endpoint returns array" {
    result=$(curl -s "${API_URL}/api/v1/events")
    echo "$result" | jq -e 'type == "array"' >/dev/null
}

@test "query endpoint supports type filter" {
    result=$(curl -s "${API_URL}/api/v1/events?type=test.integration.v1")
    echo "$result" | jq -e 'type == "array"' >/dev/null
}

@test "query endpoint supports source filter" {
    result=$(curl -s "${API_URL}/api/v1/events?source=bats-suite")
    echo "$result" | jq -e 'type == "array"' >/dev/null
}

@test "query endpoint supports limit param" {
    result=$(curl -s "${API_URL}/api/v1/events?limit=1")
    len=$(echo "$result" | jq 'length')
    [ "$len" -le 1 ]
}
