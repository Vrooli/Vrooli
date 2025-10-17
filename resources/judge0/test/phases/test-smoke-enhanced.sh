#!/usr/bin/env bash
################################################################################
# Judge0 Enhanced Smoke Tests - Works with workarounds
# 
# Tests basic functionality including alternative execution methods
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"

# Test configuration
JUDGE0_PORT="${JUDGE0_PORT:-2358}"
API_URL="http://localhost:${JUDGE0_PORT}"
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
        # Try alternative endpoint
        response=$(timeout 5 curl -sf --max-time 3 "${API_URL}/version" 2>/dev/null) || response="FAILED"
        
        if [[ "$response" == "FAILED" ]]; then
            log_test "FAIL" "Health endpoint" "No response from Judge0 API"
            return 1
        fi
    fi
    
    log_test "PASS" "Health endpoint" "Judge0 API is responding"
}

test_containers() {
    log_test "INFO" "Testing container status..."
    
    # Check server container
    local server_running=$(docker ps --filter "name=vrooli-judge0-server" --format "{{.Status}}" 2>/dev/null | head -1)
    if [[ -n "$server_running" ]]; then
        log_test "PASS" "Server container" "Running"
    else
        log_test "FAIL" "Server container" "Not found"
    fi
    
    # Check worker containers
    local worker_count=$(docker ps --filter "name=judge0-workers" --format "{{.Names}}" 2>/dev/null | wc -l)
    if [[ $worker_count -gt 0 ]]; then
        log_test "PASS" "Worker containers" "$worker_count active"
    else
        log_test "FAIL" "Worker containers" "None found"
    fi
}

test_execution_methods() {
    log_test "INFO" "Testing execution methods..."
    
    # Test execution manager if available
    if [[ -f "${RESOURCE_DIR}/lib/execution-manager.sh" ]]; then
        local best_method=$("${RESOURCE_DIR}/lib/execution-manager.sh" methods 2>/dev/null || echo "none")
        
        if [[ "$best_method" != "none" ]]; then
            log_test "PASS" "Execution manager" "Best method: $best_method"
            
            # Try a simple execution
            local exec_result=$("${RESOURCE_DIR}/lib/execution-manager.sh" execute python3 'print(1)' 2>/dev/null || echo '{"status": "error"}')
            
            if [[ "$exec_result" =~ \"status\":\ *\"accepted\" ]]; then
                log_test "PASS" "Code execution" "Python execution successful"
            else
                log_test "FAIL" "Code execution" "Python execution failed"
            fi
        else
            log_test "FAIL" "Execution manager" "No methods available"
        fi
    else
        log_test "FAIL" "Execution manager" "Not found"
    fi
    
    # Test direct executor if available
    if [[ -f "${RESOURCE_DIR}/lib/direct-executor.sh" ]]; then
        local direct_result=$("${RESOURCE_DIR}/lib/direct-executor.sh" test 2>/dev/null || echo "FAILED")
        
        if [[ "$direct_result" =~ \"status\":\ *\"accepted\" ]]; then
            log_test "PASS" "Direct executor" "Working"
        else
            log_test "INFO" "Direct executor" "Available but test failed"
        fi
    fi
    
    # Test simple executor if available
    if [[ -f "${RESOURCE_DIR}/lib/simple-exec.sh" ]]; then
        local simple_result=$("${RESOURCE_DIR}/lib/simple-exec.sh" test 2>/dev/null || echo "FAILED")
        
        if [[ "$simple_result" =~ \"status\":\ *\"accepted\" ]]; then
            log_test "PASS" "Simple executor" "Working"
        else
            log_test "INFO" "Simple executor" "Available but test failed"
        fi
    fi
}

test_language_support() {
    log_test "INFO" "Testing language support..."
    
    local languages=$(timeout 5 curl -sf "${API_URL}/languages" 2>/dev/null || echo "[]")
    local lang_count=$(echo "$languages" | jq 'length' 2>/dev/null || echo "0")
    
    if [[ $lang_count -gt 20 ]]; then
        log_test "PASS" "Language configuration" "$lang_count languages configured"
    else
        log_test "FAIL" "Language configuration" "Only $lang_count languages"
    fi
    
    # Check Docker images if docker-image-manager exists
    if [[ -f "${RESOURCE_DIR}/lib/docker-image-manager.sh" ]]; then
        local installed_count=$("${RESOURCE_DIR}/lib/docker-image-manager.sh" status 2>/dev/null | grep "✅ Installed:" | grep -oE '[0-9]+' || echo "0")
        
        if [[ $installed_count -gt 0 ]]; then
            log_test "PASS" "Docker images" "$installed_count language images available"
        else
            log_test "INFO" "Docker images" "No language images installed yet"
        fi
    fi
}

main() {
    echo -e "\n${GREEN}Running Judge0 Enhanced Smoke Tests${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Run all smoke tests
    test_health_endpoint
    test_containers
    test_execution_methods
    test_language_support
    
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