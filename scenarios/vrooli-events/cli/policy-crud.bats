#!/usr/bin/env bats
# Integration tests for policy CRUD operations
# [REQ:POL-004]

setup() {
    API_PORT=$(vrooli scenario port vrooli-events API_PORT 2>/dev/null || echo "17654")
    API_URL="http://localhost:${API_PORT}"
}

# ─── Policy CRUD ─── [REQ:POL-004]

@test "policies list endpoint returns array" {
    result=$(curl -s "${API_URL}/api/v1/policies")
    echo "$result" | jq -e 'type == "array"' >/dev/null
}

@test "policy create returns id and can be fetched with full fields" {
    result=$(curl -s -X POST "${API_URL}/api/v1/policies" \
        -H "Content-Type: application/json" \
        -d '{"rule_type":"access","source_scenario":"bats-create","target_scenario":"bats-tgt","effect":"allow","priority":1,"enabled":true}')
    id=$(echo "$result" | jq -r '.id // empty')
    [ -n "$id" ]
    get_result=$(curl -s "${API_URL}/api/v1/policies/${id}")
    echo "$get_result" | jq -e '.rule_type == "access"' >/dev/null
    echo "$get_result" | jq -e '.effect == "allow"' >/dev/null
    curl -s -o /dev/null -X DELETE "${API_URL}/api/v1/policies/${id}"
}

@test "policy create and get by id" {
    result=$(curl -s -X POST "${API_URL}/api/v1/policies" \
        -H "Content-Type: application/json" \
        -d '{"rule_type":"access","source_scenario":"bats-getid","target_scenario":"bats-tgt","effect":"deny","priority":5,"enabled":true}')
    id=$(echo "$result" | jq -r '.id // empty')
    [ -n "$id" ]
    get_result=$(curl -s "${API_URL}/api/v1/policies/${id}")
    got_id=$(echo "$get_result" | jq -r '.id // empty')
    [ "$got_id" = "$id" ]
    echo "$get_result" | jq -e '.effect == "deny"' >/dev/null
    curl -s -o /dev/null -X DELETE "${API_URL}/api/v1/policies/${id}"
}

@test "policy update changes fields" {
    result=$(curl -s -X POST "${API_URL}/api/v1/policies" \
        -H "Content-Type: application/json" \
        -d '{"rule_type":"access","source_scenario":"bats-upd","target_scenario":"bats-tgt","effect":"allow","priority":1,"enabled":true}')
    id=$(echo "$result" | jq -r '.id // empty')
    [ -n "$id" ]
    curl -s -X PUT "${API_URL}/api/v1/policies/${id}" \
        -H "Content-Type: application/json" \
        -d '{"rule_type":"access","source_scenario":"bats-upd","target_scenario":"bats-tgt","effect":"deny","priority":10,"enabled":true}'
    get_result=$(curl -s "${API_URL}/api/v1/policies/${id}")
    echo "$get_result" | jq -e '.effect == "deny"' >/dev/null
    echo "$get_result" | jq -e '.priority == 10' >/dev/null
    curl -s -o /dev/null -X DELETE "${API_URL}/api/v1/policies/${id}"
}

@test "policy create and delete lifecycle" {
    create_result=$(curl -s -X POST "${API_URL}/api/v1/policies" \
        -H "Content-Type: application/json" \
        -d '{"rule_type":"access","source_scenario":"bats-del","target_scenario":"bats-tgt","effect":"allow","priority":1,"enabled":true}')
    id=$(echo "$create_result" | jq -r '.id // empty')
    [ -n "$id" ]
    delete_result=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_URL}/api/v1/policies/${id}")
    [ "$delete_result" = "200" ] || [ "$delete_result" = "204" ]
    get_result=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/api/v1/policies/${id}")
    [ "$get_result" = "404" ]
}

@test "policy create rejects invalid rule_type" {
    result=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/v1/policies" \
        -H "Content-Type: application/json" \
        -d '{"rule_type":"invalid","source_scenario":"bats","target_scenario":"bats","effect":"allow","priority":1,"enabled":true}')
    [ "$result" = "400" ]
}
