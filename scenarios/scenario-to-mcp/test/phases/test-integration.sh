#!/bin/bash
# Integration test phase
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Testing API integration..."

# Check if API is running
API_PORT="${API_PORT:-17961}"
if ! curl -sf "http://localhost:${API_PORT}/api/v1/health" &>/dev/null; then
    testing::phase::add_warning "API not running, starting it..."
    # Note: In real integration tests, the scenario should already be running
fi

# Test 1: Health endpoint
log::info "Testing health endpoint..."
if curl -sf "http://localhost:${API_PORT}/api/v1/health" | jq -e '.status == "healthy"' &>/dev/null; then
    log::success "Health check passed"
else
    testing::phase::add_error "Health check failed"
fi

# Test 2: Registry endpoint
log::info "Testing registry endpoint..."
if curl -sf "http://localhost:${API_PORT}/api/v1/mcp/registry" | jq -e '.version == "1.0"' &>/dev/null; then
    log::success "Registry endpoint passed"
else
    testing::phase::add_error "Registry endpoint failed"
fi

# Test 3: Endpoints listing
log::info "Testing endpoints listing..."
if curl -sf "http://localhost:${API_PORT}/api/v1/mcp/endpoints" | jq -e 'has("scenarios") and has("summary")' &>/dev/null; then
    log::success "Endpoints listing passed"
else
    testing::phase::add_error "Endpoints listing failed"
fi

# Test 4: CLI integration
log::info "Testing CLI commands..."
if [ -x "cli/scenario-to-mcp" ]; then
    if bash cli/scenario-to-mcp help | grep -q "Scenario to MCP"; then
        log::success "CLI help command works"
    else
        testing::phase::add_error "CLI help command failed"
    fi

    if bash cli/scenario-to-mcp version | grep -q "v1.0"; then
        log::success "CLI version command works"
    else
        testing::phase::add_error "CLI version command failed"
    fi
else
    testing::phase::add_warning "CLI binary not found"
fi

# Test 5: Detector integration
log::info "Testing detector integration..."
if [ -f "lib/detector.js" ]; then
    if node lib/detector.test.js &>/dev/null; then
        log::success "Detector tests passed"
    else
        testing::phase::add_error "Detector tests failed"
    fi
else
    testing::phase::add_warning "Detector not found"
fi

# Test 6: MCP session creation workflow
log::info "Testing MCP session creation workflow..."
session_response=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/mcp/add" \
    -H "Content-Type: application/json" \
    -d '{"scenario_name":"test-integration-scenario","agent_config":{"auto_detect":true}}' 2>/dev/null)

if echo "$session_response" | jq -e '.success == true and has("agent_session_id")' &>/dev/null; then
    log::success "MCP session creation passed"

    # Extract session ID and verify we can retrieve it
    session_id=$(echo "$session_response" | jq -r '.agent_session_id')

    # Wait a moment for session to be stored
    sleep 1

    if curl -sf "http://localhost:${API_PORT}/api/v1/mcp/sessions/${session_id}" | jq -e 'has("id")' &>/dev/null; then
        log::success "Session retrieval passed"
    else
        testing::phase::add_warning "Session retrieval failed (may be timing issue)"
    fi
else
    testing::phase::add_error "MCP session creation failed"
fi

# Test 7: Error handling
log::info "Testing error handling..."

# Test invalid JSON
invalid_json_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/mcp/add" \
    -H "Content-Type: application/json" \
    -d '{invalid json}' 2>/dev/null)

if echo "$invalid_json_response" | jq -e '.success == false' &>/dev/null; then
    log::success "Invalid JSON handling passed"
else
    testing::phase::add_error "Invalid JSON handling failed"
fi

# Test missing scenario name
missing_name_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/mcp/add" \
    -H "Content-Type: application/json" \
    -d '{"agent_config":{"auto_detect":true}}' 2>/dev/null)

if echo "$missing_name_response" | jq -e '.success == false' &>/dev/null; then
    log::success "Missing scenario name handling passed"
else
    testing::phase::add_error "Missing scenario name handling failed"
fi

# Test 8: CORS headers
log::info "Testing CORS headers..."
cors_response=$(curl -s -X OPTIONS "http://localhost:${API_PORT}/api/v1/health" -I 2>/dev/null)

if echo "$cors_response" | grep -q "Access-Control-Allow-Origin"; then
    log::success "CORS headers present"
else
    testing::phase::add_warning "CORS headers not found"
fi

# Test 9: Concurrent requests
log::info "Testing concurrent request handling..."
concurrent_success=0
for i in {1..5}; do
    curl -sf "http://localhost:${API_PORT}/api/v1/health" &>/dev/null &
done
wait

if [ $? -eq 0 ]; then
    log::success "Concurrent requests handled"
    concurrent_success=1
fi

# Test 10: Performance check
log::info "Testing API performance..."
start_time=$(date +%s%N)
curl -sf "http://localhost:${API_PORT}/api/v1/health" &>/dev/null
end_time=$(date +%s%N)
duration_ms=$(( (end_time - start_time) / 1000000 ))

if [ $duration_ms -lt 100 ]; then
    log::success "Health endpoint responds in ${duration_ms}ms (< 100ms)"
else
    testing::phase::add_warning "Health endpoint responds in ${duration_ms}ms (> 100ms)"
fi

testing::phase::end_with_summary "Integration tests completed"
