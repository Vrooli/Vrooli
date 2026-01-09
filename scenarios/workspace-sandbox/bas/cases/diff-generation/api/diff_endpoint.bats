#!/usr/bin/env bats
# [REQ:REQ-P0-006] Stable Diff Generation
# [REQ:REQ-P0-007] Patch Application
# Phase: api
# Validates diff generation and approval workflow via API

setup() {
    API_PORT="${API_PORT:-$(vrooli scenario port workspace-sandbox API_PORT 2>/dev/null || echo 15427)}"
    API_BASE="http://127.0.0.1:${API_PORT}/api/v1"
}

# [REQ:REQ-P0-006] Diff endpoint returns 404 for missing sandbox
@test "diff endpoint returns 404 for unknown sandbox" {
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "${API_BASE}/sandboxes/00000000-0000-0000-0000-000000000000/diff")
    [ "$http_code" = "404" ]
}

# [REQ:REQ-P0-006] Diff endpoint includes proper error hint
@test "diff 404 response includes hint" {
    response=$(curl -s "${API_BASE}/sandboxes/00000000-0000-0000-0000-000000000000/diff")
    echo "$response" | jq -e '.hint' >/dev/null
}

# [REQ:REQ-P0-007] Approve endpoint returns 404 for missing sandbox
@test "approve endpoint returns 404 for unknown sandbox" {
    http_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/sandboxes/00000000-0000-0000-0000-000000000000/approve" \
        -H "Content-Type: application/json" \
        -d '{"mode": "all"}')
    [ "$http_code" = "404" ]
}

# [REQ:REQ-P0-007] Reject endpoint returns 404 for missing sandbox
@test "reject endpoint returns 404 for unknown sandbox" {
    http_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/sandboxes/00000000-0000-0000-0000-000000000000/reject")
    [ "$http_code" = "404" ]
}

# [REQ:REQ-P0-006] Workspace endpoint returns 404 for missing sandbox
@test "workspace endpoint returns 404 for unknown sandbox" {
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "${API_BASE}/sandboxes/00000000-0000-0000-0000-000000000000/workspace")
    [ "$http_code" = "404" ]
}
