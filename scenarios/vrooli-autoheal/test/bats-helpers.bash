#!/bin/bash
# BATS test helpers for vrooli-autoheal CLI tests

# Ensure we're in the cli directory
cd "$(dirname "${BATS_TEST_FILENAME}")/../cli" || exit 1

# Helper to check if scenario is running
scenario_running() {
    local port
    port=$(vrooli scenario port vrooli-autoheal API_PORT 2>/dev/null || echo "")
    [[ -n "$port" ]]
}

# Helper to make API request
api_get() {
    local endpoint="$1"
    curl -s "http://localhost:${API_PORT}${endpoint}"
}

api_post() {
    local endpoint="$1"
    curl -s -X POST "http://localhost:${API_PORT}${endpoint}"
}
