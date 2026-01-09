#!/usr/bin/env bats
# [REQ:REQ-P0-010] Health Check API Endpoint validation
# Phase: api
# Validates that the health endpoint returns correct responses and metrics

setup() {
    # Get the API port dynamically
    API_PORT="${API_PORT:-$(vrooli scenario port workspace-sandbox API_PORT 2>/dev/null || echo 15427)}"
    API_BASE="http://127.0.0.1:${API_PORT}"
}

# [REQ:REQ-P0-010] Health endpoint returns 200 OK when service is operational
@test "health endpoint returns 200 OK" {
    response=$(curl -s -w "\n%{http_code}" "${API_BASE}/health")
    http_code=$(echo "$response" | tail -n1)
    [ "$http_code" = "200" ]
}

# [REQ:REQ-P0-010] Health response includes service name
@test "health response includes service field" {
    response=$(curl -s "${API_BASE}/health")
    echo "$response" | jq -e '.service' >/dev/null
}

# [REQ:REQ-P0-010] Health response includes status field
@test "health response includes status field" {
    response=$(curl -s "${API_BASE}/health")
    status=$(echo "$response" | jq -r '.status')
    [ "$status" = "healthy" ] || [ "$status" = "unhealthy" ]
}

# [REQ:REQ-P0-010] Health response includes timestamp
@test "health response includes timestamp" {
    response=$(curl -s "${API_BASE}/health")
    echo "$response" | jq -e '.timestamp' >/dev/null
}

# [REQ:REQ-P0-010] Health response includes dependencies status
@test "health response includes dependencies" {
    response=$(curl -s "${API_BASE}/health")
    echo "$response" | jq -e '.dependencies' >/dev/null
}

# [REQ:REQ-P0-010] Health endpoint response time is acceptable
@test "health endpoint responds within 500ms" {
    start_time=$(date +%s%N)
    curl -s "${API_BASE}/health" >/dev/null
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 ))
    [ "$duration" -lt 500 ]
}

# [REQ:REQ-P0-010] Health endpoint returns correct content-type
@test "health endpoint returns application/json" {
    # Use -i instead of -I since HEAD may not be supported
    # Match "Content-Type:" at start of line to avoid matching Allow-Headers
    content_type=$(curl -si "${API_BASE}/health" | grep -i "^content-type:" | head -1)
    echo "$content_type" | grep -iq "application/json"
}
