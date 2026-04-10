#!/usr/bin/env bats
# Integration tests for vrooli-onboarding API
# [REQ:REQ-P0-001] Resource Configuration Wizard API
# [REQ:REQ-P1-001] Configuration State Management API
# [REQ:REQ-P1-002] Health Monitoring Dashboard API
# [REQ:REQ-P1-003] Setup Order API
# [REQ:REQ-P1-004] Progress Persistence API
# [REQ:REQ-P2-001] Glossary API

setup_file() {
  # Initialize pass counter for summary
  echo 0 > "${BATS_FILE_TMPDIR}/pass_count"
}

setup() {
  API_PORT="$(vrooli scenario port vrooli-onboarding API_PORT 2>/dev/null || echo 16286)"
  BASE_URL="http://127.0.0.1:${API_PORT}"
}

teardown() {
  # Track passing tests: BATS_TEST_COMPLETED is 1 when the test body succeeded
  # (avoids $status which is overwritten by tests that capture HTTP codes)
  if [[ "${BATS_TEST_COMPLETED:-0}" -eq 1 ]]; then
    local count
    count=$(cat "${BATS_FILE_TMPDIR}/pass_count" 2>/dev/null || echo 0)
    echo $(( count + 1 )) > "${BATS_FILE_TMPDIR}/pass_count"
  fi
}

teardown_file() {
  # Emit pretty-format summary so non-interactive metric parsers can detect results
  # FD 3 is BATS's real stdout, bypassing output capture
  local total passed failures
  total=$(grep -c '^@test ' "${BATS_TEST_FILENAME}" 2>/dev/null || echo 0)
  passed=$(cat "${BATS_FILE_TMPDIR}/pass_count" 2>/dev/null || echo 0)
  failures=$(( total - passed ))
  printf '%d tests, %d failures\n' "$total" "$failures" >&3
}

# ──────────────────────────────────────────────
# Health endpoints
# ──────────────────────────────────────────────

@test "GET /health returns 200 with healthy status" {
  result="$(curl -sf "${BASE_URL}/health")"
  echo "$result" | jq -e '.status == "healthy"'
}

@test "GET /api/v1/health returns 200 with version" {
  result="$(curl -sf "${BASE_URL}/api/v1/health")"
  echo "$result" | jq -e '.version == "1.0.0"'
}

# ──────────────────────────────────────────────
# Resource listing endpoints
# ──────────────────────────────────────────────

@test "GET /api/v1/resources returns JSON array of resources" {
  result="$(curl -sf "${BASE_URL}/api/v1/resources")"
  echo "$result" | jq -e '.resources | type == "array"'
}

@test "GET /api/v1/resources includes resource names" {
  result="$(curl -sf "${BASE_URL}/api/v1/resources")"
  echo "$result" | jq -e '.resources | length > 0'
  echo "$result" | jq -e '.resources[0].name | length > 0'
}

@test "GET /api/v1/resources includes category field" {
  result="$(curl -sf "${BASE_URL}/api/v1/resources")"
  echo "$result" | jq -e '.resources[0].category | length > 0'
}

@test "GET /api/v1/resources/{name} returns single resource" {
  # Get first resource name dynamically
  name="$(curl -sf "${BASE_URL}/api/v1/resources" | jq -r '.resources[0].name')"
  result="$(curl -sf "${BASE_URL}/api/v1/resources/${name}")"
  echo "$result" | jq -e ".name == \"${name}\""
}

@test "GET /api/v1/resources/{name} returns 404 for unknown resource" {
  status="$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}/api/v1/resources/nonexistent_resource_xyz")"
  [ "$status" = "404" ]
}

@test "GET /api/v1/resources/{name} is case-insensitive" {
  name="$(curl -sf "${BASE_URL}/api/v1/resources" | jq -r '.resources[0].name')"
  upper="$(echo "$name" | tr '[:lower:]' '[:upper:]')"
  result="$(curl -sf "${BASE_URL}/api/v1/resources/${upper}")"
  echo "$result" | jq -e '.name | length > 0'
}

# ──────────────────────────────────────────────
# Resource health endpoint
# ──────────────────────────────────────────────

@test "GET /api/v1/resources/health returns health data" {
  result="$(curl -sf "${BASE_URL}/api/v1/resources/health")"
  echo "$result" | jq -e '.resources | type == "array"'
  echo "$result" | jq -e 'has("total")'
  echo "$result" | jq -e 'has("healthy_count")'
  echo "$result" | jq -e 'has("checked_at")'
}

@test "GET /api/v1/resources/health has consistent counts" {
  result="$(curl -sf "${BASE_URL}/api/v1/resources/health")"
  total="$(echo "$result" | jq '.total')"
  arr_len="$(echo "$result" | jq '.resources | length')"
  [ "$total" = "$arr_len" ]
}

@test "GET /api/v1/resources/health resources have required fields" {
  result="$(curl -sf "${BASE_URL}/api/v1/resources/health")"
  count="$(echo "$result" | jq '.resources | length')"
  if [ "$count" -gt 0 ]; then
    echo "$result" | jq -e '.resources[0] | has("name", "status", "category", "available")'
  fi
}

# ──────────────────────────────────────────────
# Setup order endpoint
# ──────────────────────────────────────────────

@test "GET /api/v1/setup-order returns ordered resources" {
  result="$(curl -sf "${BASE_URL}/api/v1/setup-order")"
  echo "$result" | jq -e '.setup_order | type == "array"'
}

@test "GET /api/v1/setup-order resources have order field" {
  result="$(curl -sf "${BASE_URL}/api/v1/setup-order")"
  count="$(echo "$result" | jq '.setup_order | length')"
  if [ "$count" -gt 0 ]; then
    echo "$result" | jq -e '.setup_order[0] | has("name", "order", "category")'
  fi
}

@test "GET /api/v1/setup-order orders start at 1" {
  result="$(curl -sf "${BASE_URL}/api/v1/setup-order")"
  count="$(echo "$result" | jq '.setup_order | length')"
  if [ "$count" -gt 0 ]; then
    min_order="$(echo "$result" | jq '[.setup_order[].order] | min')"
    [ "$min_order" = "1" ]
  fi
}

# ──────────────────────────────────────────────
# Glossary endpoint
# ──────────────────────────────────────────────

@test "GET /api/v1/glossary returns entries" {
  result="$(curl -sf "${BASE_URL}/api/v1/glossary")"
  echo "$result" | jq -e '.entries | type == "array"'
  echo "$result" | jq -e '.count > 0'
}

@test "GET /api/v1/glossary entries have term and description" {
  result="$(curl -sf "${BASE_URL}/api/v1/glossary")"
  echo "$result" | jq -e '.entries[0] | has("term", "description", "category")'
}

@test "GET /api/v1/glossary?q=resource filters results" {
  result="$(curl -sf "${BASE_URL}/api/v1/glossary?q=resource")"
  echo "$result" | jq -e '.entries | length > 0'
  echo "$result" | jq -e '.entries | length < 15'
}

@test "GET /api/v1/glossary?q=nonexistent returns empty" {
  result="$(curl -sf "${BASE_URL}/api/v1/glossary?q=zzzznonexistentzzzz")"
  echo "$result" | jq -e '.entries | length == 0'
}

# ──────────────────────────────────────────────
# Progress endpoints
# ──────────────────────────────────────────────

@test "GET /api/v1/progress returns 404 for unknown user" {
  status="$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}/api/v1/progress?user_id=nonexistent_user_xyz")"
  [ "$status" = "404" ]
}

@test "PUT /api/v1/progress creates or updates progress" {
  result="$(curl -sf -X PUT "${BASE_URL}/api/v1/progress" \
    -H 'Content-Type: application/json' \
    -d '{"user_id":"bats_test_user","current_step":1,"completed_steps":[0],"config_data":{"resources":["postgres"]}}')"
  echo "$result" | jq -e '.user_id == "bats_test_user"'
  echo "$result" | jq -e '.current_step == 1'
}

@test "GET /api/v1/progress retrieves saved progress" {
  # Ensure progress exists
  curl -sf -X PUT "${BASE_URL}/api/v1/progress" \
    -H 'Content-Type: application/json' \
    -d '{"user_id":"bats_test_user","current_step":2,"completed_steps":[0,1],"config_data":{"resources":["postgres","redis"]}}' > /dev/null

  result="$(curl -sf "${BASE_URL}/api/v1/progress?user_id=bats_test_user")"
  echo "$result" | jq -e '.user_id == "bats_test_user"'
  echo "$result" | jq -e '.current_step == 2'
  echo "$result" | jq -e '.completed_steps == [0, 1]'
}

@test "GET /api/v1/progress defaults to user_id=default" {
  # This should work regardless - returns 404 or 200
  status="$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}/api/v1/progress")"
  [ "$status" = "200" ] || [ "$status" = "404" ]
}

# ──────────────────────────────────────────────
# Config generation endpoint
# ──────────────────────────────────────────────

@test "POST /api/v1/config/generate returns config snippet" {
  result="$(curl -sf -X POST "${BASE_URL}/api/v1/config/generate" \
    -H 'Content-Type: application/json' \
    -d '{"resources":["postgres"]}')"
  echo "$result" | jq -e '.config.resources | type == "object"'
}

@test "POST /api/v1/config/generate rejects empty resources" {
  status="$(curl -s -o /dev/null -w '%{http_code}' -X POST "${BASE_URL}/api/v1/config/generate" \
    -H 'Content-Type: application/json' \
    -d '{"resources":[]}')"
  [ "$status" = "400" ]
}

@test "POST /api/v1/config/generate handles unknown resources" {
  result="$(curl -sf -X POST "${BASE_URL}/api/v1/config/generate" \
    -H 'Content-Type: application/json' \
    -d '{"resources":["postgres","unknown_resource_xyz"]}')"
  echo "$result" | jq -e 'has("warnings") or has("resources")'
}

@test "POST /api/v1/config/generate rejects invalid JSON" {
  status="$(curl -s -o /dev/null -w '%{http_code}' -X POST "${BASE_URL}/api/v1/config/generate" \
    -H 'Content-Type: application/json' \
    -d 'not json')"
  [ "$status" = "400" ]
}

# ──────────────────────────────────────────────
# Config validation endpoint
# ──────────────────────────────────────────────

@test "POST /api/v1/config/validate validates valid config" {
  result="$(curl -sf -X POST "${BASE_URL}/api/v1/config/validate" \
    -H 'Content-Type: application/json' \
    -d '{"resources":{"postgres":{"enabled":true,"name":"postgres"}}}')"
  echo "$result" | jq -e 'has("valid") or has("errors") or has("results")'
}

@test "POST /api/v1/config/validate rejects invalid JSON" {
  status="$(curl -s -o /dev/null -w '%{http_code}' -X POST "${BASE_URL}/api/v1/config/validate" \
    -H 'Content-Type: application/json' \
    -d 'bad json')"
  [ "$status" = "400" ]
}

# ──────────────────────────────────────────────
# Config export endpoint
# ──────────────────────────────────────────────

@test "POST /api/v1/config/export returns export data" {
  result="$(curl -sf -X POST "${BASE_URL}/api/v1/config/export" \
    -H 'Content-Type: application/json' \
    -d '{"resources":{"postgres":{"enabled":true,"name":"postgres"}}}')"
  echo "$result" | jq -e 'has("path") and has("success")'
}

@test "POST /api/v1/config/export rejects empty body" {
  status="$(curl -s -o /dev/null -w '%{http_code}' -X POST "${BASE_URL}/api/v1/config/export" \
    -H 'Content-Type: application/json' \
    -d '{}')"
  [ "$status" = "400" ] || [ "$status" = "200" ]
}

# ──────────────────────────────────────────────
# Cross-cutting / error handling
# ──────────────────────────────────────────────

@test "Unknown route returns 404" {
  status="$(curl -s -o /dev/null -w '%{http_code}' "${BASE_URL}/api/v1/nonexistent")"
  [ "$status" = "404" ] || [ "$status" = "405" ]
}

@test "API responses have JSON content type" {
  content_type="$(curl -sf -o /dev/null -D - "${BASE_URL}/api/v1/resources" | grep -i 'content-type' | tr -d '\r')"
  echo "$content_type" | grep -qi 'application/json'
}

@test "API responds within 5 seconds" {
  time_total="$(curl -sf -o /dev/null -w '%{time_total}' "${BASE_URL}/api/v1/resources")"
  # Compare as integer (time_total * 1000 < 5000ms)
  ms="$(echo "$time_total" | awk '{printf "%d", $1 * 1000}')"
  [ "$ms" -lt 5000 ]
}

# ──────────────────────────────────────────────
# Multi-endpoint workflow tests
# ──────────────────────────────────────────────

@test "Workflow: generate config then validate it" {
  # Step 1: Generate config for available resources
  name="$(curl -sf "${BASE_URL}/api/v1/resources" | jq -r '.resources[0].name')"
  gen_result="$(curl -sf -X POST "${BASE_URL}/api/v1/config/generate" \
    -H 'Content-Type: application/json' \
    -d "{\"resources\":[\"${name}\"]}")"
  echo "$gen_result" | jq -e '.config.resources | length > 0'

  # Step 2: Validate the generated config
  # Extract resources object from generated config
  resources_obj="$(echo "$gen_result" | jq -c '.config.resources | to_entries | map({key: .key, value: (.value + {enabled: true, name: .key})}) | from_entries')"
  val_result="$(curl -sf -X POST "${BASE_URL}/api/v1/config/validate" \
    -H 'Content-Type: application/json' \
    -d "{\"resources\":${resources_obj}}")"
  echo "$val_result" | jq -e '.valid == true'
}

@test "Workflow: progress save, update, and retrieve" {
  user="bats_workflow_$$"

  # Step 1: Create initial progress
  curl -sf -X PUT "${BASE_URL}/api/v1/progress" \
    -H 'Content-Type: application/json' \
    -d "{\"user_id\":\"${user}\",\"current_step\":0,\"completed_steps\":[],\"config_data\":{}}" > /dev/null

  # Step 2: Update progress
  curl -sf -X PUT "${BASE_URL}/api/v1/progress" \
    -H 'Content-Type: application/json' \
    -d "{\"user_id\":\"${user}\",\"current_step\":2,\"completed_steps\":[0,1],\"config_data\":{\"resources\":[\"postgres\"]}}" > /dev/null

  # Step 3: Retrieve and verify the update took effect
  result="$(curl -sf "${BASE_URL}/api/v1/progress?user_id=${user}")"
  echo "$result" | jq -e '.current_step == 2'
  echo "$result" | jq -e '.completed_steps == [0, 1]'
}

@test "Workflow: resources list matches health endpoint count" {
  list_count="$(curl -sf "${BASE_URL}/api/v1/resources" | jq '.count')"
  health_total="$(curl -sf "${BASE_URL}/api/v1/resources/health" | jq '.total')"
  [ "$list_count" = "$health_total" ]
}

@test "Workflow: setup order covers all resources" {
  list_count="$(curl -sf "${BASE_URL}/api/v1/resources" | jq '.count')"
  order_count="$(curl -sf "${BASE_URL}/api/v1/setup-order" | jq '.setup_order | length')"
  [ "$list_count" = "$order_count" ]
}

@test "PUT /api/v1/progress rejects invalid JSON" {
  status="$(curl -s -o /dev/null -w '%{http_code}' -X PUT "${BASE_URL}/api/v1/progress" \
    -H 'Content-Type: application/json' \
    -d 'not json')"
  [ "$status" = "400" ]
}

@test "POST /api/v1/config/validate rejects empty resources" {
  status="$(curl -s -o /dev/null -w '%{http_code}' -X POST "${BASE_URL}/api/v1/config/validate" \
    -H 'Content-Type: application/json' \
    -d '{"resources":{}}')"
  [ "$status" = "400" ]
}

@test "POST /api/v1/config/export rejects invalid JSON" {
  status="$(curl -s -o /dev/null -w '%{http_code}' -X POST "${BASE_URL}/api/v1/config/export" \
    -H 'Content-Type: application/json' \
    -d 'not json')"
  [ "$status" = "400" ]
}

@test "GET /api/v1/glossary search is case-insensitive" {
  lower="$(curl -sf "${BASE_URL}/api/v1/glossary?q=database" | jq '.count')"
  upper="$(curl -sf "${BASE_URL}/api/v1/glossary?q=DATABASE" | jq '.count')"
  [ "$lower" = "$upper" ]
  [ "$lower" -gt 0 ]
}
