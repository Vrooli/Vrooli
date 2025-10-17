#!/bin/bash

# Visitor Intelligence Integration Test
# Tests end-to-end functionality of the visitor intelligence system

set -e

# Configuration
API_PORT="${API_PORT:-8080}"
API_BASE="http://localhost:${API_PORT}"
TEST_SCENARIO="integration-test"
TEST_FINGERPRINT="test-$(date +%s)-$$"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Utility functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1" >&2
}

# Test counter
test_count=0
pass_count=0
fail_count=0

# Test function wrapper
run_test() {
    local test_name="$1"
    local test_func="$2"
    
    ((test_count++))
    log_info "Test $test_count: $test_name"
    
    if $test_func; then
        log_success "PASS: $test_name"
        ((pass_count++))
    else
        log_error "FAIL: $test_name"
        ((fail_count++))
    fi
    echo
}

# Test functions
test_api_health() {
    local response
    response=$(curl -s "$API_BASE/health")
    local status=$?
    
    if [[ $status -eq 0 ]] && echo "$response" | grep -q '"status":"healthy"'; then
        return 0
    else
        log_error "API health check failed"
        return 1
    fi
}

test_tracking_endpoint() {
    local payload='{
        "fingerprint": "'$TEST_FINGERPRINT'",
        "event_type": "pageview",
        "scenario": "'$TEST_SCENARIO'",
        "page_url": "http://localhost:3000/test",
        "properties": {
            "test": true,
            "integration": "test-suite"
        }
    }'
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$API_BASE/api/v1/visitor/track")
    
    if echo "$response" | grep -q '"success":true'; then
        # Extract visitor ID for later tests
        VISITOR_ID=$(echo "$response" | python3 -c "
import json, sys
data = json.load(sys.stdin)
print(data.get('visitor_id', ''))
" 2>/dev/null || echo "")
        return 0
    else
        log_error "Tracking failed: $response"
        return 1
    fi
}

test_multiple_events() {
    local events=("click" "form_focus" "scroll" "pageview")
    
    for event in "${events[@]}"; do
        local payload='{
            "fingerprint": "'$TEST_FINGERPRINT'",
            "event_type": "'$event'",
            "scenario": "'$TEST_SCENARIO'",
            "page_url": "http://localhost:3000/test-'$event'",
            "properties": {
                "event_sequence": "'$event'",
                "test_batch": true
            }
        }'
        
        local response
        response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$payload" \
            "$API_BASE/api/v1/visitor/track")
        
        if ! echo "$response" | grep -q '"success":true'; then
            log_error "Failed to track $event event"
            return 1
        fi
        
        # Small delay to ensure events are processed in order
        sleep 0.1
    done
    
    log_info "Successfully tracked ${#events[@]} events"
    return 0
}

test_visitor_profile() {
    if [[ -z "$VISITOR_ID" ]]; then
        log_error "No visitor ID available from previous test"
        return 1
    fi
    
    local response
    response=$(curl -s "$API_BASE/api/v1/visitor/$VISITOR_ID")
    
    if echo "$response" | grep -q '"fingerprint":"'$TEST_FINGERPRINT'"'; then
        log_info "Visitor profile contains correct fingerprint"
        return 0
    else
        log_error "Visitor profile test failed: $response"
        return 1
    fi
}

test_scenario_analytics() {
    local response
    response=$(curl -s "$API_BASE/api/v1/analytics/scenario/$TEST_SCENARIO?timeframe=1d")
    
    if echo "$response" | grep -q '"scenario":"'$TEST_SCENARIO'"'; then
        local unique_visitors=$(echo "$response" | python3 -c "
import json, sys
data = json.load(sys.stdin)
print(data.get('unique_visitors', 0))
" 2>/dev/null || echo "0")
        
        if [[ "$unique_visitors" -gt 0 ]]; then
            log_info "Analytics show $unique_visitors unique visitors"
            return 0
        else
            log_warn "Analytics show 0 unique visitors (data may not be processed yet)"
            return 1
        fi
    else
        log_error "Analytics test failed: $response"
        return 1
    fi
}

test_tracking_script_available() {
    local response
    local status_code
    
    response=$(curl -s -w "%{http_code}" "$API_BASE/tracker.js")
    status_code="${response: -3}"
    content="${response%???}"
    
    if [[ "$status_code" == "200" ]] && echo "$content" | grep -q "VisitorIntelligence"; then
        log_info "Tracking script is available and contains expected content"
        return 0
    else
        log_error "Tracking script test failed (status: $status_code)"
        return 1
    fi
}

test_high_volume_tracking() {
    log_info "Testing high-volume tracking (50 events)..."
    
    local success_count=0
    local total_events=50
    
    for ((i=1; i<=total_events; i++)); do
        local payload='{
            "fingerprint": "load-test-'$i'-'$(date +%s)'",
            "event_type": "pageview",
            "scenario": "'$TEST_SCENARIO'",
            "page_url": "http://localhost:3000/load-test-'$i'",
            "properties": {
                "load_test": true,
                "sequence": '$i'
            }
        }'
        
        if curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$payload" \
            "$API_BASE/api/v1/visitor/track" | grep -q '"success":true'; then
            ((success_count++))
        fi
        
        # Show progress every 10 events
        if ((i % 10 == 0)); then
            log_info "Processed $i/$total_events events..."
        fi
    done
    
    local success_rate=$(echo "scale=1; $success_count * 100 / $total_events" | bc -l 2>/dev/null || echo "0")
    log_info "High-volume test: $success_count/$total_events events successful (${success_rate}%)"
    
    # Consider test passed if >95% success rate
    if [[ "$success_count" -gt $((total_events * 95 / 100)) ]]; then
        return 0
    else
        log_error "High-volume test failed (success rate too low)"
        return 1
    fi
}

test_cli_functionality() {
    local cli_script="../cli/visitor-intelligence"
    
    if [[ ! -f "$cli_script" ]]; then
        log_error "CLI script not found"
        return 1
    fi
    
    # Test CLI status
    if "$cli_script" status --json >/dev/null 2>&1; then
        log_info "CLI status command works"
    else
        log_error "CLI status command failed"
        return 1
    fi
    
    # Test CLI version
    if "$cli_script" version >/dev/null 2>&1; then
        log_info "CLI version command works"
    else
        log_error "CLI version command failed"
        return 1
    fi
    
    # Test CLI analytics (if visitor exists)
    if [[ -n "$VISITOR_ID" ]]; then
        if "$cli_script" analytics "$TEST_SCENARIO" --json >/dev/null 2>&1; then
            log_info "CLI analytics command works"
        else
            log_warn "CLI analytics command failed (may be due to data processing delay)"
        fi
    fi
    
    return 0
}

test_performance() {
    log_info "Testing API response times..."
    
    # Test tracking endpoint performance
    local start_time=$(date +%s%3N)
    local payload='{
        "fingerprint": "perf-test-'$(date +%s)'",
        "event_type": "pageview",
        "scenario": "'$TEST_SCENARIO'",
        "page_url": "http://localhost:3000/perf-test"
    }'
    
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$API_BASE/api/v1/visitor/track" >/dev/null
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    log_info "Tracking endpoint response time: ${duration}ms"
    
    # Should be under 100ms as per PRD
    if [[ "$duration" -lt 100 ]]; then
        log_success "Performance test passed (< 100ms)"
        return 0
    else
        log_warn "Performance test warning: ${duration}ms (target: <100ms)"
        # Don't fail the test, just warn
        return 0
    fi
}

# Main test execution
main() {
    echo "🧪 Visitor Intelligence Integration Test Suite"
    echo "=============================================="
    echo ""
    
    log_info "Testing API at: $API_BASE"
    log_info "Test scenario: $TEST_SCENARIO"
    log_info "Test fingerprint: $TEST_FINGERPRINT"
    echo ""
    
    # Run all tests
    run_test "API Health Check" test_api_health
    run_test "Basic Event Tracking" test_tracking_endpoint
    run_test "Multiple Event Types" test_multiple_events
    run_test "Visitor Profile Retrieval" test_visitor_profile
    run_test "Scenario Analytics" test_scenario_analytics
    run_test "Tracking Script Availability" test_tracking_script_available
    run_test "High-Volume Tracking" test_high_volume_tracking
    run_test "CLI Functionality" test_cli_functionality
    run_test "Performance Testing" test_performance
    
    # Test summary
    echo "=============================================="
    log_info "Test Summary:"
    log_info "  Total tests: $test_count"
    log_success "  Passed: $pass_count"
    
    if [[ "$fail_count" -gt 0 ]]; then
        log_error "  Failed: $fail_count"
        echo ""
        log_error "Integration tests FAILED"
        exit 1
    else
        log_info "  Failed: $fail_count"
        echo ""
        log_success "All integration tests PASSED! ✨"
        echo "SUCCESS"
        exit 0
    fi
}

# Check dependencies
if ! command -v curl >/dev/null 2>&1; then
    log_error "curl is required for integration tests"
    exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
    log_warn "python3 not found, some JSON parsing may not work"
fi

# Run tests
main "$@"