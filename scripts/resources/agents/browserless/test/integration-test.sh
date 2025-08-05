#!/usr/bin/env bash
# Browserless Integration Test
# Tests real Browserless Chrome-as-a-Service functionality
# Tests API endpoints, screenshot capture, PDF generation, and browser automation

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../../tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Browserless configuration
RESOURCES_DIR="$SCRIPT_DIR/../../.."
# shellcheck disable=SC1091
source "$RESOURCES_DIR/common.sh"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../config/defaults.sh"
browserless::export_config

# Override library defaults with Browserless-specific settings
SERVICE_NAME="browserless"
BASE_URL="${BROWSERLESS_BASE_URL:-http://localhost:4110}"
HEALTH_ENDPOINT="/pressure"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "API Port: ${BROWSERLESS_PORT:-4110}"
    "Container: ${BROWSERLESS_CONTAINER_NAME:-browserless}"
    "Max Browsers: ${BROWSERLESS_MAX_BROWSERS:-5}"
    "Timeout: ${BROWSERLESS_TIMEOUT:-30000}ms"
)

# Test configuration
readonly API_TIMEOUT="${BROWSERLESS_API_TIMEOUT:-10}"

#######################################
# BROWSERLESS-SPECIFIC TEST FUNCTIONS
#######################################

test_pressure_endpoint() {
    local test_name="pressure/health endpoint"
    
    local response
    if response=$(make_api_request "/pressure" "GET" "$API_TIMEOUT"); then
        if validate_json_field "$response" ".pressure"; then
            local pressure
            pressure=$(echo "$response" | jq -r '.pressure // 0')
            log_test_result "$test_name" "PASS" "pressure: $pressure"
            return 0
        elif [[ -n "$response" ]]; then
            log_test_result "$test_name" "PASS" "health endpoint responsive"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "pressure endpoint not accessible"
    return 1
}

test_metrics_endpoint() {
    local test_name="metrics endpoint"
    
    local response
    if response=$(make_api_request "/metrics" "GET" "$API_TIMEOUT"); then
        if echo "$response" | grep -q "browserless_\|# HELP\|# TYPE"; then
            log_test_result "$test_name" "PASS" "Prometheus metrics available"
            return 0
        elif [[ -n "$response" ]]; then
            log_test_result "$test_name" "PASS" "metrics endpoint responsive"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "metrics endpoint not available"
    return 2
}

test_screenshot_endpoint() {
    local test_name="screenshot functionality"
    
    # Test basic screenshot API
    local screenshot_data
    screenshot_data='{"url": "data:text/html,<html><body><h1>Test Page</h1></body></html>"}'
    
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "$screenshot_data" \
        "$BASE_URL/chrome/screenshot" 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "can capture screenshots"
            return 0
        elif [[ "$status_code" == "429" ]]; then
            log_test_result "$test_name" "SKIP" "rate limited (service busy)"
            return 2
        elif [[ "$status_code" == "503" ]]; then
            log_test_result "$test_name" "SKIP" "service unavailable"
            return 2
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "screenshot endpoint not working"
    return 1
}

test_pdf_endpoint() {
    local test_name="PDF generation"
    
    # Test basic PDF generation API
    local pdf_data
    pdf_data='{"url": "data:text/html,<html><body><h1>Test PDF</h1></body></html>"}'
    
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "$pdf_data" \
        "$BASE_URL/chrome/pdf" 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "can generate PDFs"
            return 0
        elif [[ "$status_code" == "429" ]]; then
            log_test_result "$test_name" "SKIP" "rate limited (service busy)"
            return 2
        elif [[ "$status_code" == "503" ]]; then
            log_test_result "$test_name" "SKIP" "service unavailable"
            return 2
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "PDF endpoint not working"
    return 1
}

test_scrape_endpoint() {
    local test_name="web scraping functionality"
    
    # Test basic scraping API
    local scrape_data
    scrape_data='{
        "url": "data:text/html,<html><body><h1>Test Title</h1><p>Test content</p></body></html>",
        "elements": [{"selector": "h1", "property": "innerText"}]
    }'
    
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "$scrape_data" \
        "$BASE_URL/chrome/scrape" 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        if [[ "$status_code" == "200" ]]; then
            log_test_result "$test_name" "PASS" "can scrape web content"
            return 0
        elif [[ "$status_code" == "429" ]]; then
            log_test_result "$test_name" "SKIP" "rate limited (service busy)"
            return 2
        elif [[ "$status_code" == "503" ]]; then
            log_test_result "$test_name" "SKIP" "service unavailable"
            return 2
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "scrape endpoint not working"
    return 1
}

test_browser_pool_status() {
    local test_name="browser pool status"
    
    local response
    if response=$(make_api_request "/pressure" "GET" "$API_TIMEOUT"); then
        if validate_json_field "$response" ".running"; then
            local running browsers
            running=$(echo "$response" | jq -r '.running // 0')
            browsers=$(echo "$response" | jq -r '.maxBrowsers // "unknown"')
            log_test_result "$test_name" "PASS" "running: $running/$browsers browsers"
            return 0
        elif validate_json_field "$response" ".pressure"; then
            local pressure
            pressure=$(echo "$response" | jq -r '.pressure // 0')
            log_test_result "$test_name" "PASS" "pool pressure: $pressure"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "browser pool status unavailable"
    return 1
}

test_websocket_endpoint() {
    local test_name="WebSocket endpoint availability"
    
    # Test if websocket endpoint is accessible (we expect connection to be refused/upgraded)
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -H "Connection: Upgrade" \
        -H "Upgrade: websocket" \
        "$BASE_URL/chrome" 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # WebSocket upgrade attempts typically return 400 or 426 when not proper websocket request
        if [[ "$status_code" == "400" ]] || [[ "$status_code" == "426" ]] || [[ "$status_code" == "101" ]]; then
            log_test_result "$test_name" "PASS" "WebSocket endpoint accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "WebSocket endpoint status unclear"
    return 2
}

test_container_health() {
    local test_name="Docker container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local container_name="${BROWSERLESS_CONTAINER_NAME:-browserless}"
    
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

test_resource_limits() {
    local test_name="resource usage monitoring"
    
    local response
    if response=$(make_api_request "/pressure" "GET" "$API_TIMEOUT"); then
        # Check if we can get resource information
        if validate_json_field "$response" ".running"; then
            local running max_browsers
            running=$(echo "$response" | jq -r '.running // 0')
            max_browsers=$(echo "$response" | jq -r '.maxBrowsers // 5')
            
            # Validate resource usage is within limits
            if [[ "$running" -le "$max_browsers" ]]; then
                log_test_result "$test_name" "PASS" "resource usage within limits ($running/$max_browsers)"
                return 0
            else
                log_test_result "$test_name" "FAIL" "resource usage exceeds limits ($running/$max_browsers)"
                return 1
            fi
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "resource monitoring not available"
    return 2
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "Browserless Information:"
    echo "  API Endpoints:"
    echo "    - Health/Pressure: GET $BASE_URL/pressure"
    echo "    - Screenshots: POST $BASE_URL/chrome/screenshot"
    echo "    - PDF Generation: POST $BASE_URL/chrome/pdf"
    echo "    - Web Scraping: POST $BASE_URL/chrome/scrape"
    echo "    - WebSocket: WS $BASE_URL/chrome"
    echo "    - Metrics: GET $BASE_URL/metrics"
    echo "  Container: ${BROWSERLESS_CONTAINER_NAME:-browserless}"
    echo "  Max Browsers: ${BROWSERLESS_MAX_BROWSERS:-5}"
    echo "  Timeout: ${BROWSERLESS_TIMEOUT:-30000}ms"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register Browserless-specific tests
register_tests \
    "test_pressure_endpoint" \
    "test_metrics_endpoint" \
    "test_screenshot_endpoint" \
    "test_pdf_endpoint" \
    "test_scrape_endpoint" \
    "test_browser_pool_status" \
    "test_websocket_endpoint" \
    "test_container_health" \
    "test_resource_limits"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi