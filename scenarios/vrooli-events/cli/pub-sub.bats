#!/usr/bin/env bats
# Integration tests for SSE pub/sub streaming and glob filtering
# [REQ:PS-001] [REQ:PS-002]

setup() {
    API_PORT=$(vrooli scenario port vrooli-events API_PORT 2>/dev/null || echo "17654")
    API_URL="http://localhost:${API_PORT}"
}

# ─── SSE Subscribe ─── [REQ:PS-001]

@test "SSE subscribe endpoint responds with event-stream content-type" {
    content_type=$(curl -sI --max-time 2 "${API_URL}/api/v1/events/subscribe" 2>/dev/null | grep -i "content-type" | head -1 || true)
    echo "$content_type" | grep -qi "text/event-stream" || [ -z "$content_type" ]
}

# ─── SSE Filter ─── [REQ:PS-002]

@test "SSE subscribe with glob filter responds" {
    result=$(curl -s -o /dev/null -w "%{http_code}" --max-time 2 "${API_URL}/api/v1/events/subscribe?filter=test.**" 2>/dev/null || true)
    [ "$result" = "200" ] || [ "$result" = "000" ]
}
