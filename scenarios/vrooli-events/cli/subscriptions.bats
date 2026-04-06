#!/usr/bin/env bats
# Integration tests for subscription CRUD, health tracking, and test endpoint
# [REQ:SUB-001] [REQ:SUB-004] [REQ:SUB-005]

setup() {
    API_PORT=$(vrooli scenario port vrooli-events API_PORT 2>/dev/null || echo "17654")
    API_URL="http://localhost:${API_PORT}"
}

# ─── Subscription CRUD ─── [REQ:SUB-001]

@test "subscriptions list endpoint returns array" {
    result=$(curl -s "${API_URL}/api/v1/subscriptions")
    echo "$result" | jq -e 'type == "array"' >/dev/null
}

@test "subscription create returns id and can be fetched with full fields" {
    result=$(curl -s -X POST "${API_URL}/api/v1/subscriptions" \
        -H "Content-Type: application/json" \
        -d '{"name":"bats-test-sub","owner_scenario":"bats-suite","event_pattern":"test.**","delivery_type":"webhook","delivery_target":"http://localhost:1/hook","enabled":true}')
    id=$(echo "$result" | jq -r '.id // empty')
    [ -n "$id" ]
    get_result=$(curl -s "${API_URL}/api/v1/subscriptions/${id}")
    echo "$get_result" | jq -e '.name == "bats-test-sub"' >/dev/null
    echo "$get_result" | jq -e '.event_pattern == "test.**"' >/dev/null
    curl -s -o /dev/null -X DELETE "${API_URL}/api/v1/subscriptions/${id}"
}

@test "subscription create and get by id" {
    result=$(curl -s -X POST "${API_URL}/api/v1/subscriptions" \
        -H "Content-Type: application/json" \
        -d '{"name":"bats-getid-sub","owner_scenario":"bats-suite","event_pattern":"test.get.**","delivery_type":"webhook","delivery_target":"http://localhost:1/hook","enabled":true}')
    id=$(echo "$result" | jq -r '.id // empty')
    [ -n "$id" ]
    get_result=$(curl -s "${API_URL}/api/v1/subscriptions/${id}")
    got_id=$(echo "$get_result" | jq -r '.id // empty')
    [ "$got_id" = "$id" ]
    echo "$get_result" | jq -e '.name == "bats-getid-sub"' >/dev/null
    curl -s -o /dev/null -X DELETE "${API_URL}/api/v1/subscriptions/${id}"
}

@test "subscription update changes fields" {
    result=$(curl -s -X POST "${API_URL}/api/v1/subscriptions" \
        -H "Content-Type: application/json" \
        -d '{"name":"bats-upd-sub","owner_scenario":"bats-suite","event_pattern":"test.upd.**","delivery_type":"webhook","delivery_target":"http://localhost:1/hook","enabled":true}')
    id=$(echo "$result" | jq -r '.id // empty')
    [ -n "$id" ]
    curl -s -X PUT "${API_URL}/api/v1/subscriptions/${id}" \
        -H "Content-Type: application/json" \
        -d '{"name":"bats-updated-sub","owner_scenario":"bats-suite","event_pattern":"test.upd.**","delivery_type":"webhook","delivery_target":"http://localhost:2/hook","enabled":false}'
    get_result=$(curl -s "${API_URL}/api/v1/subscriptions/${id}")
    echo "$get_result" | jq -e '.name == "bats-updated-sub"' >/dev/null
    curl -s -o /dev/null -X DELETE "${API_URL}/api/v1/subscriptions/${id}"
}

@test "subscription create and delete lifecycle" {
    result=$(curl -s -X POST "${API_URL}/api/v1/subscriptions" \
        -H "Content-Type: application/json" \
        -d '{"name":"bats-del-sub","owner_scenario":"bats-suite","event_pattern":"test.del.**","delivery_type":"webhook","delivery_target":"http://localhost:1/hook","enabled":true}')
    id=$(echo "$result" | jq -r '.id // empty')
    [ -n "$id" ]
    delete_code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_URL}/api/v1/subscriptions/${id}")
    [ "$delete_code" = "200" ] || [ "$delete_code" = "204" ]
    get_code=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}/api/v1/subscriptions/${id}")
    [ "$get_code" = "404" ]
}

# ─── Subscription Health ─── [REQ:SUB-004]

@test "subscription health endpoint returns stats" {
    result=$(curl -s -X POST "${API_URL}/api/v1/subscriptions" \
        -H "Content-Type: application/json" \
        -d '{"name":"bats-health-sub","owner_scenario":"bats-suite","event_pattern":"test.health.**","delivery_type":"webhook","delivery_target":"http://localhost:1/hook","enabled":true}')
    id=$(echo "$result" | jq -r '.id // empty')
    [ -n "$id" ]
    health=$(curl -s "${API_URL}/api/v1/subscriptions/${id}/health")
    echo "$health" | jq -e '.total_delivered' >/dev/null || echo "$health" | jq -e '.subscription_id' >/dev/null
    curl -s -o /dev/null -X DELETE "${API_URL}/api/v1/subscriptions/${id}"
}

# ─── Subscription Test ─── [REQ:SUB-005]

@test "subscription test endpoint returns delivery result" {
    result=$(curl -s -X POST "${API_URL}/api/v1/subscriptions" \
        -H "Content-Type: application/json" \
        -d '{"name":"bats-testep-sub","owner_scenario":"bats-suite","event_pattern":"test.testep.**","delivery_type":"webhook","delivery_target":"http://localhost:1/hook","enabled":true}')
    id=$(echo "$result" | jq -r '.id // empty')
    [ -n "$id" ]
    test_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/v1/subscriptions/${id}/test")
    [ "$test_code" = "200" ] || [ "$test_code" = "502" ]
    curl -s -o /dev/null -X DELETE "${API_URL}/api/v1/subscriptions/${id}"
}
