#!/usr/bin/env bats
# Integration tests for event store persistence and schema
# [REQ:ES-001] [REQ:ES-002]

setup() {
    API_PORT=$(vrooli scenario port vrooli-events API_PORT 2>/dev/null || echo "17654")
    API_URL="http://localhost:${API_PORT}"
}

# ─── Event Persistence ─── [REQ:ES-001]

@test "event store persists and retrieves events" {
    uid="bats-persist-$(date +%s%N)"
    curl -s -o /dev/null -X POST "${API_URL}/api/v1/events" \
        -H "Content-Type: application/json" \
        -d "{\"eventId\":\"${uid}\",\"eventType\":\"test.persist.v1\",\"sourceScenario\":\"bats-store\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"
    result=$(curl -s "${API_URL}/api/v1/events?type=test.persist.v1&limit=1")
    echo "$result" | jq -e 'length > 0' >/dev/null
}

@test "event store enforces unique eventId" {
    uid="bats-dup-$(date +%s%N)"
    curl -s -o /dev/null -X POST "${API_URL}/api/v1/events" \
        -H "Content-Type: application/json" \
        -d "{\"eventId\":\"${uid}\",\"eventType\":\"test.dup.v1\",\"sourceScenario\":\"bats-store\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"
    result=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/v1/events" \
        -H "Content-Type: application/json" \
        -d "{\"eventId\":\"${uid}\",\"eventType\":\"test.dup.v1\",\"sourceScenario\":\"bats-store\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}")
    # Not 200/201/202 — confirms uniqueness enforcement
    [ "$result" != "200" ] && [ "$result" != "201" ] && [ "$result" != "202" ]
}

# ─── Event Schema ─── [REQ:ES-002]

@test "stored events contain required schema fields" {
    uid="bats-schema-$(date +%s%N)"
    curl -s -o /dev/null -X POST "${API_URL}/api/v1/events" \
        -H "Content-Type: application/json" \
        -d "{\"eventId\":\"${uid}\",\"eventType\":\"test.schema.v1\",\"sourceScenario\":\"bats-store\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"
    result=$(curl -s "${API_URL}/api/v1/events?type=test.schema.v1&limit=1")
    echo "$result" | jq -e '.[0].eventId' >/dev/null
    echo "$result" | jq -e '.[0].eventType' >/dev/null
    echo "$result" | jq -e '.[0].sourceScenario' >/dev/null
}
