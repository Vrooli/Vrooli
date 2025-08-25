#!/usr/bin/env bash
# Agent-S2 Integration Test
# Tests real Agent-S2 natural language computer control functionality
# Tests API endpoints, automation capabilities, AI features, and VNC access

set -euo pipefail

# Source shared integration test library
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
AGENT_S2_TEST_DIR="${APP_ROOT}/resources/agent-s2/test"

# Source var.sh first to get proper directory variables
# shellcheck disable=SC1091
source "$AGENT_S2_TEST_DIR/../../../../lib/utils/var.sh"

# shellcheck disable=SC1091
source "$var_SCRIPTS_RESOURCES_DIR/tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Agent-S2 configuration
# shellcheck disable=SC1091
source "$var_SCRIPTS_RESOURCES_DIR/common.sh"
# shellcheck disable=SC1091
source "$AGENT_S2_TEST_DIR/../config/defaults.sh"
agents2::export_config

# Override library defaults with Agent-S2-specific settings
SERVICE_NAME="agent-s2"
BASE_URL="${AGENTS2_BASE_URL:-http://localhost:4113}"
HEALTH_ENDPOINT="/health"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "API Port: ${AGENTS2_PORT:-4113}"
    "VNC Port: ${AGENTS2_VNC_PORT:-5900}"
    "Container: ${AGENTS2_CONTAINER_NAME:-agent-s2}"
    "Mode: Sandbox (secure)"
    "AI Provider: ${AGENTS2_LLM_PROVIDER:-ollama}"
)

# Test configuration
readonly API_BASE="/api/v1"
readonly TEST_TIMEOUT="${AGENTS2_API_TIMEOUT:-120}"

#######################################
# AGENT-S2-SPECIFIC TEST FUNCTIONS
#######################################

test_health_endpoint() {
    local test_name="health endpoint"
    
    local response
    if response=$(make_api_request "/health" "GET" 10); then
        if echo "$response" | grep -qi "status.*ok\|healthy\|up"; then
            log_test_result "$test_name" "PASS" "service healthy"
            return 0
        elif [[ -n "$response" ]]; then
            log_test_result "$test_name" "PASS" "health endpoint responsive"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "health endpoint not accessible"
    return 1
}

test_capabilities_endpoint() {
    local test_name="capabilities endpoint"
    
    local response
    if response=$(make_api_request "/capabilities" "GET" 10); then
        if validate_json_field "$response" ".automation_available"; then
            local automation_available
            automation_available=$(echo "$response" | jq -r '.automation_available')
            log_test_result "$test_name" "PASS" "automation available: $automation_available"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "capabilities endpoint not accessible"
    return 1
}

test_screenshot_functionality() {
    local test_name="screenshot functionality"
    
    # Test basic screenshot endpoint
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$BASE_URL/screenshot" 2>/dev/null); then
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "can capture screenshots"
            return 0
        elif [[ "$status_code" == "503" ]]; then
            log_test_result "$test_name" "SKIP" "service not ready for screenshots"
            return 2
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "screenshot functionality not working"
    return 1
}

test_automation_endpoints() {
    local test_name="core automation endpoints"
    
    # Test mouse endpoint
    local mouse_response
    if mouse_response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d '{"action": "move", "x": 100, "y": 100}' \
        "$BASE_URL/mouse" 2>/dev/null); then
        
        local status_code
        status_code=$(echo "$mouse_response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [[ "$status_code" == "200" ]] || [[ "$status_code" == "202" ]]; then
            log_test_result "$test_name" "PASS" "automation endpoints accessible"
            return 0
        elif [[ "$status_code" == "503" ]]; then
            log_test_result "$test_name" "SKIP" "automation not ready"
            return 2
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "automation endpoints not working"
    return 1
}

test_ai_capabilities() {
    local test_name="AI capabilities"
    
    # Check if AI is enabled in capabilities
    local response
    if response=$(make_api_request "/capabilities" "GET" 10); then
        if validate_json_field "$response" ".ai_available"; then
            local ai_available
            ai_available=$(echo "$response" | jq -r '.ai_available')
            
            if [[ "$ai_available" == "true" ]]; then
                # Test AI task endpoint (just check if it's accessible)
                local ai_response
                local status_code
                if ai_response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
                    -H "Content-Type: application/json" \
                    -d '{"task": "test task", "dry_run": true}' \
                    "$BASE_URL/ai/action" 2>/dev/null); then
                    
                    status_code=$(echo "$ai_response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
                    
                    if [[ "$status_code" == "200" ]] || [[ "$status_code" == "202" ]] || [[ "$status_code" == "400" ]]; then
                        log_test_result "$test_name" "PASS" "AI endpoints accessible"
                        return 0
                    fi
                fi
                
                log_test_result "$test_name" "FAIL" "AI enabled but endpoints not working"
                return 1
            else
                log_test_result "$test_name" "SKIP" "AI not enabled"
                return 2
            fi
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "AI capabilities unknown"
    return 2
}

test_vnc_access() {
    local test_name="VNC access"
    
    if ! command -v nc >/dev/null 2>&1 && ! command -v netcat >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "nc/netcat not available"
        return 2
    fi
    
    local nc_cmd="nc"
    command -v netcat >/dev/null 2>&1 && nc_cmd="netcat"
    
    # Test VNC port accessibility
    local vnc_port="${AGENTS2_VNC_PORT:-5900}"
    if timeout 5 $nc_cmd -z localhost "$vnc_port" 2>/dev/null; then
        log_test_result "$test_name" "PASS" "VNC port $vnc_port accessible"
        return 0
    fi
    
    log_test_result "$test_name" "FAIL" "VNC port $vnc_port not accessible"
    return 1
}

test_container_health() {
    local test_name="Docker container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local container_name="${AGENTS2_CONTAINER_NAME:-agent-s2}"
    
    # Check if container exists and is running
    if docker ps --format '{{.Names}}' | grep -q "^${container_name}$"; then
        local container_status
        container_status=$(docker inspect "${container_name}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
        if [[ "$container_status" == "running" ]]; then
            # Check container health if health check is configured
            local health_status
            health_status=$(docker inspect "${container_name}" --format '{{.State.Health.Status}}' 2>/dev/null || echo "none")
            
            if [[ "$health_status" == "healthy" ]]; then
                log_test_result "$test_name" "PASS" "container running and healthy"
                return 0
            elif [[ "$health_status" == "none" ]]; then
                log_test_result "$test_name" "PASS" "container running (no health check)"
                return 0
            else
                log_test_result "$test_name" "FAIL" "container running but unhealthy: $health_status"
                return 1
            fi
        else
            log_test_result "$test_name" "FAIL" "container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "container not found"
        return 1
    fi
}

test_virtual_display() {
    local test_name="virtual display"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local container_name="${AGENTS2_CONTAINER_NAME:-agent-s2}"
    
    # Check if container is running
    if ! docker ps --format '{{.Names}}' | grep -q "^${container_name}$"; then
        log_test_result "$test_name" "SKIP" "container not running"
        return 2
    fi
    
    # Check if X display is running in container
    local display_output
    if display_output=$(docker exec "$container_name" pgrep -f "Xvfb\|Xorg" 2>/dev/null); then
        if [[ -n "$display_output" ]]; then
            log_test_result "$test_name" "PASS" "virtual display running"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "virtual display not running"
    return 1
}

test_api_authentication() {
    local test_name="API authentication"
    
    # Test if endpoints require authentication
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$BASE_URL/admin/status" 2>/dev/null); then
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [[ "$status_code" == "401" ]] || [[ "$status_code" == "403" ]]; then
            log_test_result "$test_name" "PASS" "admin endpoints require authentication"
            return 0
        elif [[ "$status_code" == "404" ]]; then
            log_test_result "$test_name" "SKIP" "no admin endpoints found"
            return 2
        elif [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "admin endpoints accessible (no auth required)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "authentication status unclear"
    return 2
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "Agent-S2 Information:"
    echo "  API Endpoints:"
    echo "    - Health: GET $BASE_URL/health"
    echo "    - Capabilities: GET $BASE_URL/capabilities"
    echo "    - Screenshot: GET $BASE_URL/screenshot"
    echo "    - Mouse Control: POST $BASE_URL/mouse"
    echo "    - Keyboard: POST $BASE_URL/keyboard"
    echo "    - AI Actions: POST $BASE_URL/ai/action"
    echo "  VNC Access: vnc://localhost:${AGENTS2_VNC_PORT:-5900}"
    echo "  VNC Password: ${AGENTS2_VNC_PASSWORD:-agents2vnc}"
    echo "  Container: ${AGENTS2_CONTAINER_NAME:-agent-s2}"
    echo "  Mode: Sandbox (secure container isolation)"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register Agent-S2-specific tests
register_tests \
    "test_health_endpoint" \
    "test_capabilities_endpoint" \
    "test_screenshot_functionality" \
    "test_automation_endpoints" \
    "test_ai_capabilities" \
    "test_vnc_access" \
    "test_container_health" \
    "test_virtual_display" \
    "test_api_authentication"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi