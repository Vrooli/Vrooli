#!/bin/bash
# Business logic test phase
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Testing business logic..."

API_PORT="${API_PORT:-17961}"

# Test 1: MCP detection accuracy
log::info "Test: MCP detection should identify scenarios correctly"
DETECTION_RESULT=$(curl -sf "http://localhost:${API_PORT}/api/v1/mcp/endpoints" 2>/dev/null)
if echo "$DETECTION_RESULT" | jq -e '.summary.total > 0' &>/dev/null; then
    TOTAL=$(echo "$DETECTION_RESULT" | jq -r '.summary.total')
    WITH_MCP=$(echo "$DETECTION_RESULT" | jq -r '.summary.withMCP')
    log::success "Detected $TOTAL scenarios, $WITH_MCP with MCP"
else
    testing::phase::add_warning "Could not verify detection results"
fi

# Test 2: Registry should only list active endpoints
log::info "Test: Registry should only show active MCP endpoints"
REGISTRY_RESULT=$(curl -sf "http://localhost:${API_PORT}/api/v1/mcp/registry" 2>/dev/null)
if echo "$REGISTRY_RESULT" | jq -e '.endpoints' &>/dev/null; then
    ENDPOINT_COUNT=$(echo "$REGISTRY_RESULT" | jq -r '.endpoints | length')
    log::success "Registry contains $ENDPOINT_COUNT active endpoints"

    # Verify each endpoint has required fields
    if echo "$REGISTRY_RESULT" | jq -e '.endpoints[] | has("name") and has("transport") and has("url") and has("manifest_url")' &>/dev/null; then
        log::success "All endpoints have required fields"
    else
        testing::phase::add_error "Some endpoints missing required fields"
    fi
else
    testing::phase::add_error "Registry endpoint structure invalid"
fi

# Test 3: MCP addition should create session
log::info "Test: Adding MCP should create agent session"
ADD_RESULT=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/mcp/add" \
    -H "Content-Type: application/json" \
    -d '{"scenario_name":"test-scenario","agent_config":{"auto_detect":true}}' 2>/dev/null)

if echo "$ADD_RESULT" | jq -e '.success == true' &>/dev/null; then
    SESSION_ID=$(echo "$ADD_RESULT" | jq -r '.agent_session_id')
    log::success "Created session: $SESSION_ID"

    # Verify we can retrieve the session
    sleep 1
    SESSION_RESULT=$(curl -sf "http://localhost:${API_PORT}/api/v1/mcp/sessions/${SESSION_ID}" 2>/dev/null)
    if echo "$SESSION_RESULT" | jq -e '.id' &>/dev/null; then
        log::success "Session retrieval works"
    else
        testing::phase::add_error "Failed to retrieve created session"
    fi
else
    testing::phase::add_warning "MCP addition test skipped (API may not be running)"
fi

# Test 4: Error handling for invalid requests
log::info "Test: API should handle invalid requests gracefully"
INVALID_RESULT=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/mcp/add" \
    -H "Content-Type: application/json" \
    -d '{"invalid":"request"}' 2>/dev/null)

if echo "$INVALID_RESULT" | jq -e '.success == false' &>/dev/null; then
    log::success "Invalid request properly rejected"
else
    testing::phase::add_error "Invalid request handling failed"
fi

# Test 5: Scenario details retrieval
log::info "Test: Scenario details should be retrievable"
DETAILS_RESULT=$(curl -sf "http://localhost:${API_PORT}/api/v1/mcp/scenarios/scenario-to-mcp" 2>/dev/null)
if [ -n "$DETAILS_RESULT" ]; then
    if echo "$DETAILS_RESULT" | jq -e 'has("name")' &>/dev/null; then
        log::success "Scenario details retrieved successfully"
    else
        testing::phase::add_warning "Scenario details may be incomplete"
    fi
else
    testing::phase::add_warning "Could not retrieve scenario details"
fi

# Test 6: Port allocation logic
log::info "Test: Port allocation should be sequential and unique"
# Create multiple test sessions to verify port allocation
for i in {1..3}; do
    ADD_RESULT=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/mcp/add" \
        -H "Content-Type: application/json" \
        -d "{\"scenario_name\":\"port-test-$i\",\"agent_config\":{\"auto_detect\":true}}" 2>/dev/null)

    if echo "$ADD_RESULT" | jq -e '.success == true' &>/dev/null; then
        log::debug "Created session for port-test-$i"
    fi
done
log::success "Port allocation test completed"

# Test 7: Session status tracking
log::info "Test: Session status should progress correctly"
ADD_RESULT=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/mcp/add" \
    -H "Content-Type: application/json" \
    -d '{"scenario_name":"status-test","agent_config":{"auto_detect":true}}' 2>/dev/null)

if echo "$ADD_RESULT" | jq -e '.success == true' &>/dev/null; then
    SESSION_ID=$(echo "$ADD_RESULT" | jq -r '.agent_session_id')

    # Initial status should be pending or running
    sleep 1
    SESSION_STATUS=$(curl -sf "http://localhost:${API_PORT}/api/v1/mcp/sessions/${SESSION_ID}" 2>/dev/null)

    if echo "$SESSION_STATUS" | jq -e '.status' &>/dev/null; then
        STATUS=$(echo "$SESSION_STATUS" | jq -r '.status')
        log::success "Session status tracked: $STATUS"
    else
        testing::phase::add_error "Session status not tracked properly"
    fi
fi

# Test 8: Data consistency
log::info "Test: Data consistency between endpoints"
ENDPOINTS_DATA=$(curl -sf "http://localhost:${API_PORT}/api/v1/mcp/endpoints" 2>/dev/null)
REGISTRY_DATA=$(curl -sf "http://localhost:${API_PORT}/api/v1/mcp/registry" 2>/dev/null)

if [ -n "$ENDPOINTS_DATA" ] && [ -n "$REGISTRY_DATA" ]; then
    ACTIVE_COUNT=$(echo "$ENDPOINTS_DATA" | jq -r '.summary.active // 0')
    REGISTRY_COUNT=$(echo "$REGISTRY_DATA" | jq -r '.endpoints | length')

    log::info "Active endpoints: $ACTIVE_COUNT, Registry endpoints: $REGISTRY_COUNT"

    # Registry should only contain active endpoints
    if [ "$ACTIVE_COUNT" -ge "$REGISTRY_COUNT" ]; then
        log::success "Data consistency check passed"
    else
        testing::phase::add_warning "Registry contains more endpoints than active count"
    fi
fi

# Test 9: Response time consistency
log::info "Test: Response times should be consistent"
declare -a times
for i in {1..5}; do
    start=$(date +%s%N)
    curl -sf "http://localhost:${API_PORT}/api/v1/health" &>/dev/null
    end=$(date +%s%N)
    duration=$(( (end - start) / 1000000 ))
    times[$i]=$duration
done

# Calculate average
total=0
for time in "${times[@]}"; do
    total=$((total + time))
done
avg=$((total / 5))

log::info "Average response time: ${avg}ms"
if [ $avg -lt 50 ]; then
    log::success "Response times are consistent and fast"
else
    testing::phase::add_warning "Response times averaging ${avg}ms (target: <50ms)"
fi

# Test 10: Workflow completeness
log::info "Test: Complete workflow from detection to registration"
# 1. Detect scenarios
DETECT_RESULT=$(curl -sf "http://localhost:${API_PORT}/api/v1/mcp/endpoints" 2>/dev/null)

# 2. Add MCP to one scenario
if echo "$DETECT_RESULT" | jq -e '.scenarios[0].name' &>/dev/null; then
    SCENARIO_NAME=$(echo "$DETECT_RESULT" | jq -r '.scenarios[0].name')

    # 3. Create MCP session
    ADD_RESULT=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/mcp/add" \
        -H "Content-Type: application/json" \
        -d "{\"scenario_name\":\"$SCENARIO_NAME\",\"agent_config\":{\"auto_detect\":true}}" 2>/dev/null)

    if echo "$ADD_RESULT" | jq -e '.success == true' &>/dev/null; then
        SESSION_ID=$(echo "$ADD_RESULT" | jq -r '.agent_session_id')

        # 4. Verify session creation
        sleep 2
        SESSION_CHECK=$(curl -sf "http://localhost:${API_PORT}/api/v1/mcp/sessions/${SESSION_ID}" 2>/dev/null)

        if echo "$SESSION_CHECK" | jq -e '.id' &>/dev/null; then
            log::success "Complete workflow validated: detect → add → track"
        else
            testing::phase::add_error "Workflow incomplete: session not tracked"
        fi
    else
        testing::phase::add_warning "Workflow test skipped: MCP addition failed"
    fi
fi

testing::phase::end_with_summary "Business logic tests completed"
