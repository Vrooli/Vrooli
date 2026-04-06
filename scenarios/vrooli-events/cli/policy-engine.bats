#!/usr/bin/env bats
# Integration tests for policy evaluation, violations, and circuit breaker override
# [REQ:POL-006] [REQ:POL-007] [REQ:POL-008]

setup() {
    API_PORT=$(vrooli scenario port vrooli-events API_PORT 2>/dev/null || echo "17654")
    API_URL="http://localhost:${API_PORT}"
}

# ─── Policy Evaluate ─── [REQ:POL-006]

@test "policy evaluate endpoint returns decision" {
    result=$(curl -s -X POST "${API_URL}/api/v1/policies/evaluate" \
        -H "Content-Type: application/json" \
        -d '{"source":"bats-eval","target":"bats-tgt","endpoint":"/test"}')
    echo "$result" | jq -e 'has("allowed")' >/dev/null
}

# ─── Policy Violations ─── [REQ:POL-007]

@test "policy violations endpoint returns array" {
    result=$(curl -s "${API_URL}/api/v1/policies/violations")
    echo "$result" | jq -e 'type == "array"' >/dev/null
}

# ─── Circuit Breaker Override ─── [REQ:POL-008]

@test "circuit breaker override requires circuit_breaker rule" {
    result=$(curl -s -X POST "${API_URL}/api/v1/policies" \
        -H "Content-Type: application/json" \
        -d '{"rule_type":"access","source_scenario":"bats-cb","target_scenario":"bats-tgt","effect":"allow","priority":1,"enabled":true}')
    id=$(echo "$result" | jq -r '.id // empty')
    [ -n "$id" ]
    override_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/v1/policies/${id}/override" \
        -H "Content-Type: application/json" \
        -d '{"state":"open"}')
    [ "$override_code" = "400" ] || [ "$override_code" = "422" ]
    curl -s -o /dev/null -X DELETE "${API_URL}/api/v1/policies/${id}"
}
