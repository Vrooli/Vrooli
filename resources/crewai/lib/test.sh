#!/usr/bin/env bash
################################################################################
# CrewAI Test Library - v2.0 Universal Contract Compliant
#
# Test implementations for CrewAI resource validation
################################################################################

set -euo pipefail

# Source dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CREWAI_ROOT="${APP_ROOT}/resources/crewai"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${CREWAI_ROOT}/lib/core.sh"

################################################################################
# Test Handler Functions
################################################################################

# Run all tests
crewai::test::all() {
    local failed=0
    
    log::header "Running All CrewAI Tests"
    
    # Run each test phase
    if ! crewai::test::smoke; then
        ((failed++))
    fi
    
    if ! crewai::test::integration; then
        ((failed++))
    fi
    
    if ! crewai::test::unit; then
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All tests passed!"
        return 0
    else
        log::error "$failed test suite(s) failed"
        return 1
    fi
}

# Smoke tests - Quick validation (<30s)
crewai::test::smoke() {
    log::header "CrewAI Smoke Tests"
    
    local failed=0
    local test_count=0
    
    # Test 1: Service is running
    ((test_count++))
    log::info "Test $test_count: Service running check"
    if crewai::is_running; then
        log::success "✅ Service is running"
    else
        log::error "❌ Service is not running"
        ((failed++))
    fi
    
    # Test 2: Health endpoint responds
    ((test_count++))
    log::info "Test $test_count: Health endpoint check"
    if timeout 5 curl -sf "http://localhost:${CREWAI_PORT}/health" &>/dev/null; then
        log::success "✅ Health endpoint responds"
    else
        log::error "❌ Health endpoint not responding"
        ((failed++))
    fi
    
    # Test 3: Health response is valid JSON
    ((test_count++))
    log::info "Test $test_count: Health response validation"
    local health_response
    health_response=$(timeout 5 curl -sf "http://localhost:${CREWAI_PORT}/health" 2>/dev/null || echo "{}")
    if echo "$health_response" | jq -e '.status' &>/dev/null; then
        log::success "✅ Health response is valid JSON"
    else
        log::error "❌ Health response is invalid"
        ((failed++))
    fi
    
    # Test 4: Port is correct
    ((test_count++))
    log::info "Test $test_count: Port configuration check"
    if [[ "${CREWAI_PORT}" == "8084" ]]; then
        log::success "✅ Port configured correctly"
    else
        log::error "❌ Port misconfigured (expected 8084, got ${CREWAI_PORT})"
        ((failed++))
    fi
    
    # Test 5: Directories exist
    ((test_count++))
    log::info "Test $test_count: Directory structure check"
    if [[ -d "${CREWAI_DATA_DIR}" ]] && [[ -d "${CREWAI_CREWS_DIR}" ]] && [[ -d "${CREWAI_AGENTS_DIR}" ]]; then
        log::success "✅ Required directories exist"
    else
        log::error "❌ Missing required directories"
        ((failed++))
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All $test_count smoke tests passed!"
        return 0
    else
        log::error "$failed/$test_count smoke tests failed"
        return 1
    fi
}

# Integration tests - End-to-end functionality (<120s)
crewai::test::integration() {
    log::header "CrewAI Integration Tests"
    
    local failed=0
    local test_count=0
    
    # Test 1: API root endpoint
    ((test_count++))
    log::info "Test $test_count: API root endpoint"
    local root_response
    root_response=$(timeout 5 curl -sf "http://localhost:${CREWAI_PORT}/" 2>/dev/null || echo "{}")
    if echo "$root_response" | jq -e '.name' &>/dev/null; then
        log::success "✅ API root endpoint works"
    else
        log::error "❌ API root endpoint failed"
        ((failed++))
    fi
    
    # Test 2: List crews endpoint
    ((test_count++))
    log::info "Test $test_count: List crews endpoint"
    local crews_response
    crews_response=$(timeout 5 curl -sf "http://localhost:${CREWAI_PORT}/crews" 2>/dev/null || echo "{}")
    if echo "$crews_response" | jq -e '.crews' &>/dev/null; then
        log::success "✅ List crews endpoint works"
    else
        log::error "❌ List crews endpoint failed"
        ((failed++))
    fi
    
    # Test 3: List agents endpoint
    ((test_count++))
    log::info "Test $test_count: List agents endpoint"
    local agents_response
    agents_response=$(timeout 5 curl -sf "http://localhost:${CREWAI_PORT}/agents" 2>/dev/null || echo "{}")
    if echo "$agents_response" | jq -e '.agents' &>/dev/null; then
        log::success "✅ List agents endpoint works"
    else
        log::error "❌ List agents endpoint failed"
        ((failed++))
    fi
    
    # Test 4: Inject endpoint (with test data)
    ((test_count++))
    log::info "Test $test_count: Inject endpoint"
    # Create a test crew file
    local test_crew="/tmp/test_crew_$$.py"
    echo "# Test crew for integration testing" > "$test_crew"
    
    local inject_response
    inject_response=$(curl -sf -X POST "http://localhost:${CREWAI_PORT}/inject" \
        -H "Content-Type: application/json" \
        -d "{\"file_path\": \"$test_crew\", \"file_type\": \"crew\"}" 2>/dev/null || echo "{}")
    
    if echo "$inject_response" | jq -e '.status == "injected"' &>/dev/null; then
        log::success "✅ Inject endpoint works"
        # Clean up injected test file
        rm -f "${CREWAI_CREWS_DIR}/test_crew_$$.py"
    else
        log::error "❌ Inject endpoint failed"
        ((failed++))
    fi
    rm -f "$test_crew"
    
    # Test 5: Service restart
    ((test_count++))
    log::info "Test $test_count: Service restart capability"
    stop_crewai &>/dev/null
    sleep 2
    start_crewai &>/dev/null
    sleep 2
    if crewai::is_running; then
        log::success "✅ Service restart works"
    else
        log::error "❌ Service restart failed"
        ((failed++))
    fi
    
    # Test 6: Create agent API
    ((test_count++))
    log::info "Test $test_count: Create agent API"
    local create_agent_response
    create_agent_response=$(timeout 5 curl -sf -X POST "http://localhost:${CREWAI_PORT}/agents" \
        -H "Content-Type: application/json" \
        -d '{"name": "test_agent", "role": "tester", "goal": "test system"}' 2>/dev/null || echo "{}")
    if echo "$create_agent_response" | jq -e '.status == "created"' &>/dev/null; then
        log::success "✅ Create agent API works"
    else
        log::error "❌ Create agent API failed"
        ((failed++))
    fi
    
    # Test 7: Create crew API
    ((test_count++))
    log::info "Test $test_count: Create crew API"
    local create_crew_response
    create_crew_response=$(timeout 5 curl -sf -X POST "http://localhost:${CREWAI_PORT}/crews" \
        -H "Content-Type: application/json" \
        -d '{"name": "test_crew_integration", "agents": ["test_agent"], "tasks": ["test_task"]}' 2>/dev/null || echo "{}")
    if echo "$create_crew_response" | jq -e '.status == "created"' &>/dev/null; then
        log::success "✅ Create crew API works"
    else
        log::error "❌ Create crew API failed"
        ((failed++))
    fi
    
    # Test 8: Execute crew API
    ((test_count++))
    log::info "Test $test_count: Execute crew API"
    local execute_response
    execute_response=$(timeout 5 curl -sf -X POST "http://localhost:${CREWAI_PORT}/execute" \
        -H "Content-Type: application/json" \
        -d '{"crew": "test_crew_integration", "input": {"test": "data"}}' 2>/dev/null || echo "{}")
    if echo "$execute_response" | jq -e '.status == "started"' &>/dev/null; then
        log::success "✅ Execute crew API works"
        
        # Check task status
        local task_id
        task_id=$(echo "$execute_response" | jq -r '.task_id' 2>/dev/null || echo "")
        if [[ -n "$task_id" ]]; then
            sleep 2
            local task_status
            task_status=$(timeout 5 curl -sf "http://localhost:${CREWAI_PORT}/tasks/$task_id" 2>/dev/null || echo "{}")
            if echo "$task_status" | jq -e '.status' &>/dev/null; then
                log::success "  ✅ Task tracking works"
            else
                log::error "  ❌ Task tracking failed"
                ((failed++))
            fi
        fi
    else
        log::error "❌ Execute crew API failed"
        ((failed++))
    fi
    
    # Test 9: Delete crew API
    ((test_count++))
    log::info "Test $test_count: Delete crew API"
    local delete_response
    delete_response=$(timeout 5 curl -sf -X DELETE "http://localhost:${CREWAI_PORT}/crews/test_crew_integration" 2>/dev/null || echo "{}")
    if echo "$delete_response" | jq -e '.status == "deleted"' &>/dev/null; then
        log::success "✅ Delete crew API works"
    else
        log::error "❌ Delete crew API failed"
        ((failed++))
    fi
    
    # Test 10: Delete agent API
    ((test_count++))
    log::info "Test $test_count: Delete agent API"
    local delete_agent_response
    delete_agent_response=$(timeout 5 curl -sf -X DELETE "http://localhost:${CREWAI_PORT}/agents/test_agent" 2>/dev/null || echo "{}")
    if echo "$delete_agent_response" | jq -e '.status == "deleted"' &>/dev/null; then
        log::success "✅ Delete agent API works"
    else
        log::error "❌ Delete agent API failed"
        ((failed++))
    fi
    
    # Test 11: Tools endpoint
    ((test_count++))
    log::info "Test $test_count: Tools endpoint"
    local tools_response
    tools_response=$(timeout 5 curl -sf "http://localhost:${CREWAI_PORT}/tools" 2>/dev/null || echo "{}")
    if echo "$tools_response" | jq -e '.tools' &>/dev/null; then
        log::success "✅ Tools endpoint works"
    else
        log::error "❌ Tools endpoint failed"
        ((failed++))
    fi
    
    # Test 12: Create agent with tools
    ((test_count++))
    log::info "Test $test_count: Create agent with tools"
    local agent_with_tools_response
    agent_with_tools_response=$(timeout 5 curl -sf -X POST "http://localhost:${CREWAI_PORT}/agents" \
        -H "Content-Type: application/json" \
        -d '{"name": "tooled_agent", "role": "analyst", "goal": "analyze data", "tools": ["file_reader", "api_caller"]}' 2>/dev/null || echo "{}")
    if echo "$agent_with_tools_response" | jq -e '.agent.tools | length > 0' &>/dev/null; then
        log::success "✅ Create agent with tools works"
        # Clean up
        curl -sf -X DELETE "http://localhost:${CREWAI_PORT}/agents/tooled_agent" &>/dev/null || true
    else
        log::error "❌ Create agent with tools failed"
        ((failed++))
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All $test_count integration tests passed!"
        return 0
    else
        log::error "$failed/$test_count integration tests failed"
        return 1
    fi
}

# Unit tests - Library function validation (<60s)
crewai::test::unit() {
    log::header "CrewAI Unit Tests"
    
    local failed=0
    local test_count=0
    
    # Test 1: check_python function
    ((test_count++))
    log::info "Test $test_count: check_python function"
    if check_python &>/dev/null; then
        log::success "✅ Python detection works"
    else
        log::error "❌ Python detection failed"
        ((failed++))
    fi
    
    # Test 2: init_directories function
    ((test_count++))
    log::info "Test $test_count: init_directories function"
    # Temporarily use a test directory
    local orig_data_dir="${CREWAI_DATA_DIR}"
    export CREWAI_DATA_DIR="/tmp/crewai_test_$$"
    export CREWAI_CREWS_DIR="${CREWAI_DATA_DIR}/crews"
    export CREWAI_AGENTS_DIR="${CREWAI_DATA_DIR}/agents"
    export CREWAI_WORKSPACE_DIR="${CREWAI_DATA_DIR}/workspace"
    
    init_directories
    if [[ -d "${CREWAI_DATA_DIR}" ]] && [[ -d "${CREWAI_CREWS_DIR}" ]]; then
        log::success "✅ Directory creation works"
        rm -rf "${CREWAI_DATA_DIR}"
    else
        log::error "❌ Directory creation failed"
        ((failed++))
    fi
    
    # Restore original paths
    export CREWAI_DATA_DIR="${orig_data_dir}"
    export CREWAI_CREWS_DIR="${CREWAI_DATA_DIR}/crews"
    export CREWAI_AGENTS_DIR="${CREWAI_DATA_DIR}/agents"
    export CREWAI_WORKSPACE_DIR="${CREWAI_DATA_DIR}/workspace"
    
    # Test 3: get_health function
    ((test_count++))
    log::info "Test $test_count: get_health function"
    local health_json
    health_json=$(get_health)
    if [[ -n "$health_json" ]]; then
        log::success "✅ get_health function works"
    else
        log::error "❌ get_health function failed"
        ((failed++))
    fi
    
    # Test 4: Port configuration from registry
    ((test_count++))
    log::info "Test $test_count: Port registry integration"
    local registry_port
    registry_port=$("${APP_ROOT}/scripts/resources/port_registry.sh" crewai | grep -E "crewai\s+:" | awk '{print $3}')
    if [[ "$registry_port" == "8084" ]]; then
        log::success "✅ Port registry integration correct"
    else
        log::error "❌ Port registry mismatch (got: $registry_port)"
        ((failed++))
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All $test_count unit tests passed!"
        return 0
    else
        log::error "$failed/$test_count unit tests failed"
        return 1
    fi
}

# Export test functions for CLI framework
export -f crewai::test::all
export -f crewai::test::smoke
export -f crewai::test::integration
export -f crewai::test::unit