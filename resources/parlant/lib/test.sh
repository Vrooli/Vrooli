#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"
CONFIG_DIR="${RESOURCE_DIR}/config"

# Source configuration
source "${CONFIG_DIR}/defaults.sh"

# Main test runner
parlant_run_tests() {
    local test_type="${1:-all}"
    local exit_code=0
    
    case "$test_type" in
        smoke)
            parlant_test_smoke || exit_code=$?
            ;;
        integration)
            parlant_test_integration || exit_code=$?
            ;;
        unit)
            parlant_test_unit || exit_code=$?
            ;;
        all)
            echo "Running all Parlant tests..."
            echo ""
            parlant_test_smoke || exit_code=$?
            echo ""
            parlant_test_integration || exit_code=$?
            echo ""
            parlant_test_unit || exit_code=$?
            ;;
        *)
            echo "Error: Unknown test type '$test_type'"
            return 1
            ;;
    esac
    
    return $exit_code
}

# Smoke test - Quick health check
parlant_test_smoke() {
    echo "=== Parlant Smoke Test ==="
    local test_passed=0
    local test_failed=0
    
    # Test 1: Service is running
    echo -n "1. Service running check... "
    if parlant_is_running; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Service not running"
        ((test_failed++))
        echo "   Start service with: vrooli resource parlant manage start"
        return 1
    fi
    
    # Test 2: Health endpoint responds
    echo -n "2. Health endpoint check... "
    if timeout "${PARLANT_HEALTH_CHECK_TIMEOUT}" curl -sf \
        "http://${PARLANT_HOST}:${PARLANT_PORT}/health" &>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Health endpoint not responding"
        ((test_failed++))
    fi
    
    # Test 3: Health response is valid JSON
    echo -n "3. Health response validation... "
    local health_response=$(timeout "${PARLANT_HEALTH_CHECK_TIMEOUT}" \
        curl -sf "http://${PARLANT_HOST}:${PARLANT_PORT}/health" 2>/dev/null || echo "{}")
    
    if echo "$health_response" | jq -e '.status == "healthy"' &>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Invalid health response"
        ((test_failed++))
    fi
    
    # Test 4: Response time is acceptable
    echo -n "4. Response time check... "
    local start_time=$(date +%s%N)
    timeout "${PARLANT_HEALTH_CHECK_TIMEOUT}" curl -sf \
        "http://${PARLANT_HOST}:${PARLANT_PORT}/health" &>/dev/null
    local end_time=$(date +%s%N)
    local response_time=$(( (end_time - start_time) / 1000000 ))
    
    if [[ $response_time -lt 500 ]]; then
        echo "✓ PASSED (${response_time}ms)"
        ((test_passed++))
    else
        echo "✗ FAILED - Response too slow (${response_time}ms > 500ms)"
        ((test_failed++))
    fi
    
    # Summary
    echo ""
    echo "Smoke Test Results: $test_passed passed, $test_failed failed"
    
    if [[ $test_failed -eq 0 ]]; then
        echo "✓ All smoke tests passed"
        return 0
    else
        echo "✗ Some smoke tests failed"
        return 1
    fi
}

# Integration test - Full functionality
parlant_test_integration() {
    echo "=== Parlant Integration Test ==="
    local test_passed=0
    local test_failed=0
    
    # Ensure service is running
    if ! parlant_is_running; then
        echo "Service not running. Starting for integration tests..."
        parlant_start --wait || return 1
    fi
    
    # Test 1: Create agent
    echo -n "1. Create agent test... "
    local agent_response=$(curl -sf -X POST \
        "http://${PARLANT_HOST}:${PARLANT_PORT}/agents" \
        -H "Content-Type: application/json" \
        -d '{"name": "TestAgent", "description": "Integration test agent"}' 2>/dev/null)
    
    if echo "$agent_response" | jq -e '.agent_id' &>/dev/null; then
        local agent_id=$(echo "$agent_response" | jq -r '.agent_id')
        echo "✓ PASSED (ID: $agent_id)"
        ((test_passed++))
    else
        echo "✗ FAILED - Could not create agent"
        ((test_failed++))
        agent_id="agent_1"
    fi
    
    # Test 2: List agents
    echo -n "2. List agents test... "
    local list_response=$(curl -sf "http://${PARLANT_HOST}:${PARLANT_PORT}/agents" 2>/dev/null)
    
    if echo "$list_response" | jq -e '.agents | length > 0' &>/dev/null; then
        local agent_count=$(echo "$list_response" | jq '.agents | length')
        echo "✓ PASSED ($agent_count agents)"
        ((test_passed++))
    else
        echo "✗ FAILED - Could not list agents"
        ((test_failed++))
    fi
    
    # Test 3: Add guideline
    echo -n "3. Add guideline test... "
    local guideline_response=$(curl -sf -X POST \
        "http://${PARLANT_HOST}:${PARLANT_PORT}/agents/${agent_id}/guidelines" \
        -H "Content-Type: application/json" \
        -d "{\"agent_id\": \"$agent_id\", \"condition\": \"test condition\", \"action\": \"test action\"}" 2>/dev/null)
    
    if echo "$guideline_response" | jq -e '.status == "guideline added"' &>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Could not add guideline"
        ((test_failed++))
    fi
    
    # Test 4: Add tool
    echo -n "4. Add tool test... "
    local tool_response=$(curl -sf -X POST \
        "http://${PARLANT_HOST}:${PARLANT_PORT}/agents/${agent_id}/tools" \
        -H "Content-Type: application/json" \
        -d '{"name": "test_tool", "description": "Test tool", "implementation_type": "external"}' 2>/dev/null)
    
    if echo "$tool_response" | jq -e '.status == "tool registered"' &>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Could not add tool"
        ((test_failed++))
    fi
    
    # Test 5: Chat with agent
    echo -n "5. Chat functionality test... "
    local chat_response=$(curl -sf -X POST \
        "http://${PARLANT_HOST}:${PARLANT_PORT}/agents/${agent_id}/chat" \
        -H "Content-Type: application/json" \
        -d '{"message": "Hello test"}' 2>/dev/null)
    
    if echo "$chat_response" | jq -e '.message' &>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Chat endpoint not working"
        ((test_failed++))
    fi
    
    # Test 6: Get agent details
    echo -n "6. Get agent details test... "
    local agent_details=$(curl -sf "http://${PARLANT_HOST}:${PARLANT_PORT}/agents/${agent_id}" 2>/dev/null)
    
    if echo "$agent_details" | jq -e '.guidelines | length > 0' &>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Could not get agent details"
        ((test_failed++))
    fi
    
    # P1 Feature Tests
    echo ""
    echo "--- P1 Feature Tests ---"
    
    # Test 7: Guideline conflict detection
    echo -n "7. Guideline conflict detection test... "
    # Add conflicting guideline
    curl -sf -X POST \
        "http://${PARLANT_HOST}:${PARLANT_PORT}/agents/${agent_id}/guidelines" \
        -H "Content-Type: application/json" \
        -d "{\"agent_id\": \"$agent_id\", \"condition\": \"test condition\", \"action\": \"do not test action\", \"priority\": 0}" &>/dev/null
    
    local conflicts_response=$(curl -sf "http://${PARLANT_HOST}:${PARLANT_PORT}/agents/${agent_id}/guidelines/conflicts" 2>/dev/null)
    
    if echo "$conflicts_response" | jq -e '.conflicts_found > 0' &>/dev/null; then
        echo "✓ PASSED (conflicts detected)"
        ((test_passed++))
    else
        echo "✗ FAILED - Conflict detection not working"
        ((test_failed++))
    fi
    
    # Test 8: Response templates
    echo -n "8. Response templates test... "
    local template_response=$(curl -sf -X POST \
        "http://${PARLANT_HOST}:${PARLANT_PORT}/agents/${agent_id}/templates" \
        -H "Content-Type: application/json" \
        -d "{\"agent_id\": \"$agent_id\", \"template_id\": \"greeting\", \"pattern\": \"hello\", \"response\": \"Hello {name}!\", \"variables\": [\"name\"]}" 2>/dev/null)
    
    if echo "$template_response" | jq -e '.status == "template added"' &>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Template management not working"
        ((test_failed++))
    fi
    
    # Test 9: Self-critique engine
    echo -n "9. Self-critique engine test... "
    local chat_with_critique=$(curl -sf -X POST \
        "http://${PARLANT_HOST}:${PARLANT_PORT}/agents/${agent_id}/chat" \
        -H "Content-Type: application/json" \
        -d '{"message": "test condition here"}' 2>/dev/null)
    
    if echo "$chat_with_critique" | jq -e '.self_critique.adherence_score' &>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Self-critique not working"
        ((test_failed++))
    fi
    
    # Test 10: Audit logging
    echo -n "10. Audit logging test... "
    local audit_logs=$(curl -sf "http://${PARLANT_HOST}:${PARLANT_PORT}/audit/logs?agent_id=${agent_id}" 2>/dev/null)
    
    if echo "$audit_logs" | jq -e '.returned_logs > 0' &>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Audit logging not working"
        ((test_failed++))
    fi
    
    # Test 11: Audit summary
    echo -n "11. Audit summary test... "
    local audit_summary=$(curl -sf "http://${PARLANT_HOST}:${PARLANT_PORT}/audit/summary" 2>/dev/null)
    
    if echo "$audit_summary" | jq -e '.total_events > 0' &>/dev/null; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Audit summary not working"
        ((test_failed++))
    fi
    
    # Summary
    echo ""
    echo "Integration Test Results: $test_passed passed, $test_failed failed"
    
    if [[ $test_failed -eq 0 ]]; then
        echo "✓ All integration tests passed"
        return 0
    else
        echo "✗ Some integration tests failed"
        return 1
    fi
}

# Unit test - Library functions
parlant_test_unit() {
    echo "=== Parlant Unit Test ==="
    local test_passed=0
    local test_failed=0
    
    # Test 1: Configuration loading
    echo -n "1. Configuration loading test... "
    if [[ -n "${PARLANT_PORT}" ]] && [[ "${PARLANT_PORT}" == "11458" ]]; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Configuration not loaded correctly"
        ((test_failed++))
    fi
    
    # Test 2: Directory structure
    echo -n "2. Directory structure test... "
    local dirs_exist=true
    for dir in "${PARLANT_DATA_DIR}" "${CONFIG_DIR}" "${SCRIPT_DIR}"; do
        if [[ ! -d "$dir" ]]; then
            dirs_exist=false
            break
        fi
    done
    
    if [[ "$dirs_exist" == true ]]; then
        echo "✓ PASSED"
        ((test_passed++))
    else
        echo "✗ FAILED - Required directories missing"
        ((test_failed++))
    fi
    
    # Test 3: Python availability
    echo -n "3. Python environment test... "
    if command -v python3 &>/dev/null; then
        local python_version=$(python3 --version 2>&1 | cut -d' ' -f2)
        echo "✓ PASSED (Python $python_version)"
        ((test_passed++))
    else
        echo "✗ FAILED - Python 3 not available"
        ((test_failed++))
    fi
    
    # Test 4: Runtime configuration
    echo -n "4. Runtime configuration test... "
    if [[ -f "${CONFIG_DIR}/runtime.json" ]]; then
        if jq -e '.startup_order == 550' "${CONFIG_DIR}/runtime.json" &>/dev/null; then
            echo "✓ PASSED"
            ((test_passed++))
        else
            echo "✗ FAILED - Invalid runtime configuration"
            ((test_failed++))
        fi
    else
        echo "✗ FAILED - Runtime configuration missing"
        ((test_failed++))
    fi
    
    # Test 5: Schema validation
    echo -n "5. Schema validation test... "
    if [[ -f "${CONFIG_DIR}/schema.json" ]]; then
        if jq -e '.properties.port.default == 11458' "${CONFIG_DIR}/schema.json" &>/dev/null; then
            echo "✓ PASSED"
            ((test_passed++))
        else
            echo "✗ FAILED - Invalid schema"
            ((test_failed++))
        fi
    else
        echo "✗ FAILED - Schema file missing"
        ((test_failed++))
    fi
    
    # Summary
    echo ""
    echo "Unit Test Results: $test_passed passed, $test_failed failed"
    
    if [[ $test_failed -eq 0 ]]; then
        echo "✓ All unit tests passed"
        return 0
    else
        echo "✗ Some unit tests failed"
        return 1
    fi
}

# Helper function (if not already defined in core.sh)
if ! declare -f parlant_is_running &>/dev/null; then
    parlant_is_running() {
        if [[ -f "${PARLANT_PID_FILE}" ]]; then
            local pid=$(cat "${PARLANT_PID_FILE}")
            if kill -0 "$pid" 2>/dev/null; then
                return 0
            fi
        fi
        return 1
    }
fi

# Helper function for starting service (if not already defined)
if ! declare -f parlant_start &>/dev/null; then
    source "${SCRIPT_DIR}/core.sh"
fi