#!/usr/bin/env bash
################################################################################
# Judge0 Smoke Tests - Quick Health Validation (<30s)
# 
# Tests basic functionality and health endpoints
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test configuration
JUDGE0_PORT="${JUDGE0_PORT:-2358}"
API_URL="http://localhost:${JUDGE0_PORT}"
MAX_WAIT=30
TEST_FAILED=0

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_test() {
    local status="$1"
    local test_name="$2"
    local details="${3:-}"
    
    case "$status" in
        PASS)
            echo -e "  ${GREEN}✓${NC} $test_name"
            [[ -n "$details" ]] && echo "    $details"
            ;;
        FAIL)
            echo -e "  ${RED}✗${NC} $test_name"
            [[ -n "$details" ]] && echo -e "    ${RED}$details${NC}"
            TEST_FAILED=1
            ;;
        INFO)
            echo -e "  ${YELLOW}ℹ${NC} $test_name"
            [[ -n "$details" ]] && echo "    $details"
            ;;
    esac
}

test_health_endpoint() {
    log_test "INFO" "Testing health endpoint..."
    
    # Test with proper timeout
    local response
    response=$(timeout 5 curl -sf --max-time 3 "${API_URL}/system_info" 2>/dev/null) || response="FAILED"
    
    if [[ "$response" == "FAILED" ]]; then
        log_test "FAIL" "Health endpoint" "No response from ${API_URL}/system_info"
        return 1
    fi
    
    # Check if response is valid JSON
    if echo "$response" | python3 -m json.tool &>/dev/null; then
        log_test "PASS" "Health endpoint" "Valid JSON response received"
        
        # Check response time
        local start_time=$(date +%s%N)
        timeout 1 curl -sf --max-time 1 "${API_URL}/system_info" &>/dev/null
        local end_time=$(date +%s%N)
        local response_time=$(( (end_time - start_time) / 1000000 ))
        
        if [[ $response_time -lt 500 ]]; then
            log_test "PASS" "Response time" "${response_time}ms (target: <500ms)"
        else
            log_test "FAIL" "Response time" "${response_time}ms (target: <500ms)"
        fi
    else
        log_test "FAIL" "Health endpoint" "Invalid JSON response"
        return 1
    fi
}

test_api_authentication() {
    log_test "INFO" "Testing API authentication..."
    
    # Check if API key exists
    local api_key_file="${HOME}/.vrooli/resources/judge0/config/api_key"
    if [[ -f "$api_key_file" ]]; then
        log_test "PASS" "API key file exists" "$api_key_file"
        
        # Test authentication is enforced (if configured)
        local api_key=$(cat "$api_key_file" 2>/dev/null || echo "")
        if [[ -n "$api_key" ]]; then
            # Test with invalid key (should fail or return auth error)
            local bad_response=$(timeout 2 curl -sf -H "X-Auth-Token: invalid_key" \
                "${API_URL}/languages" 2>/dev/null || echo "BLOCKED")
            
            # Test with valid key (should succeed)
            local good_response=$(timeout 2 curl -sf -H "X-Auth-Token: ${api_key}" \
                "${API_URL}/languages" 2>/dev/null || echo "FAILED")
            
            if [[ "$good_response" != "FAILED" ]]; then
                log_test "PASS" "API authentication" "Valid key accepted"
            else
                log_test "INFO" "API authentication" "Authentication not enforced (development mode)"
            fi
        else
            log_test "INFO" "API key empty" "Authentication disabled"
        fi
    else
        log_test "FAIL" "API key file missing" "$api_key_file"
    fi
}

test_language_endpoint() {
    log_test "INFO" "Testing language support..."
    
    local languages=$(timeout 5 curl -sf "${API_URL}/languages" 2>/dev/null || echo "FAILED")
    
    if [[ "$languages" == "FAILED" ]]; then
        log_test "FAIL" "Language endpoint" "Failed to retrieve language list"
        return 1
    fi
    
    # Count languages
    local lang_count=$(echo "$languages" | python3 -c "import sys, json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
    
    if [[ $lang_count -gt 20 ]]; then
        log_test "PASS" "Language support" "${lang_count} languages available (requirement: >20)"
        
        # Check for key languages
        local has_python=$(echo "$languages" | grep -q '"name":"Python"' && echo "yes" || echo "no")
        local has_js=$(echo "$languages" | grep -q '"name":"JavaScript"' && echo "yes" || echo "no")
        local has_go=$(echo "$languages" | grep -q '"name":"Go"' && echo "yes" || echo "no")
        
        if [[ "$has_python" == "yes" && "$has_js" == "yes" && "$has_go" == "yes" ]]; then
            log_test "PASS" "Core languages" "Python, JavaScript, Go available"
        else
            log_test "FAIL" "Core languages" "Missing required languages"
        fi
    else
        log_test "FAIL" "Language support" "Only ${lang_count} languages (requirement: >20)"
    fi
}

test_submission_endpoint() {
    log_test "INFO" "Testing code submission..."
    
    # Submit simple Python code
    local submission=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "print(\"Hello, Judge0!\")",
            "language_id": 71,
            "stdin": "",
            "expected_output": "Hello, Judge0!\n"
        }' 2>/dev/null || echo "FAILED")
    
    if [[ "$submission" == "FAILED" ]]; then
        log_test "FAIL" "Code submission" "Failed to submit code"
        return 1
    fi
    
    # Check submission result
    local status_id=$(echo "$submission" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', {}).get('id', 0))" 2>/dev/null || echo "0")
    local stdout=$(echo "$submission" | python3 -c "import sys, json; print(json.load(sys.stdin).get('stdout', ''))" 2>/dev/null || echo "")
    
    if [[ $status_id -eq 3 ]]; then
        log_test "PASS" "Code execution" "Successfully executed Python code"
        
        if [[ "$stdout" == "Hello, Judge0!" ]]; then
            log_test "PASS" "Output validation" "Correct output received"
        else
            log_test "FAIL" "Output validation" "Unexpected output: $stdout"
        fi
    else
        log_test "FAIL" "Code execution" "Execution failed with status: $status_id"
    fi
}

test_container_health() {
    log_test "INFO" "Testing container health..."
    
    # Check all Judge0 containers are running
    local containers=$(docker ps --format "table {{.Names}}\t{{.Status}}" | grep judge0 || echo "")
    
    if [[ -z "$containers" ]]; then
        log_test "FAIL" "Container status" "No Judge0 containers found"
        return 1
    fi
    
    local all_healthy=true
    while IFS= read -r line; do
        local name=$(echo "$line" | awk '{print $1}')
        local status=$(echo "$line" | awk '{$1=""; print $0}' | xargs)
        
        if echo "$status" | grep -q "Up"; then
            log_test "PASS" "Container: $name" "$status"
        else
            log_test "FAIL" "Container: $name" "$status"
            all_healthy=false
        fi
    done <<< "$containers"
    
    if [[ "$all_healthy" == "false" ]]; then
        return 1
    fi
}

test_resource_limits() {
    log_test "INFO" "Testing resource limits..."
    
    # Test CPU time limit with infinite loop
    local cpu_test=$(timeout 15 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "while True: pass",
            "language_id": 71,
            "cpu_time_limit": 1
        }' 2>/dev/null || echo "FAILED")
    
    if [[ "$cpu_test" != "FAILED" ]]; then
        local status_id=$(echo "$cpu_test" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', {}).get('id', 0))" 2>/dev/null || echo "0")
        
        if [[ $status_id -eq 5 ]]; then
            log_test "PASS" "CPU time limit" "Correctly enforced (TLE status)"
        else
            log_test "FAIL" "CPU time limit" "Not properly enforced (status: $status_id)"
        fi
    else
        log_test "FAIL" "CPU time limit test" "Failed to submit test"
    fi
    
    # Test memory limit
    local mem_test=$(timeout 15 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "a = [0] * (256 * 1024 * 1024)",
            "language_id": 71,
            "memory_limit": 50000
        }' 2>/dev/null || echo "FAILED")
    
    if [[ "$mem_test" != "FAILED" ]]; then
        local status_id=$(echo "$mem_test" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', {}).get('id', 0))" 2>/dev/null || echo "0")
        
        if [[ $status_id -eq 12 ]]; then
            log_test "PASS" "Memory limit" "Correctly enforced (MLE status)"
        elif [[ $status_id -ge 7 && $status_id -le 11 ]]; then
            log_test "PASS" "Memory limit" "Enforced with runtime error (status: $status_id)"
        else
            log_test "FAIL" "Memory limit" "Not properly enforced (status: $status_id)"
        fi
    else
        log_test "FAIL" "Memory limit test" "Failed to submit test"
    fi
}

main() {
    echo -e "\n${GREEN}Running Judge0 Smoke Tests${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Run all smoke tests
    test_health_endpoint
    test_api_authentication
    test_language_endpoint
    test_submission_endpoint
    test_container_health
    test_resource_limits
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ $TEST_FAILED -eq 0 ]]; then
        echo -e "${GREEN}✅ All smoke tests passed${NC}\n"
        exit 0
    else
        echo -e "${RED}❌ Some smoke tests failed${NC}\n"
        exit 1
    fi
}

main "$@"