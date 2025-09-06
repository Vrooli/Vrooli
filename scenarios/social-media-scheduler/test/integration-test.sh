#!/bin/bash

# Social Media Scheduler Integration Test Script
# This script performs basic integration tests for the scenario

set -euo pipefail

# Configuration
API_PORT="${API_PORT:-18000}"
UI_PORT="${UI_PORT:-38000}"
API_URL="http://localhost:${API_PORT}"
UI_URL="http://localhost:${UI_PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}✗${NC} $1"
    ((TESTS_FAILED++))
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    local description="${3:-}"
    
    ((TESTS_RUN++))
    
    if [ -n "$description" ]; then
        log "Testing: $test_name - $description"
    else
        log "Testing: $test_name"
    fi
    
    if eval "$test_command" >/dev/null 2>&1; then
        success "$test_name"
        return 0
    else
        fail "$test_name"
        return 1
    fi
}

# Test functions
test_api_health() {
    run_test "API Health" \
        "curl -sf $API_URL/health | jq -e '.status == \"healthy\"'" \
        "Check if API server is healthy"
}

test_queue_health() {
    run_test "Queue Health" \
        "curl -sf $API_URL/health/queue | jq -e '.success == true'" \
        "Check if job queue is operational"
}

test_ui_health() {
    run_test "UI Health" \
        "curl -sf $UI_URL/health | jq -e '.status == \"healthy\"'" \
        "Check if UI server is healthy"
}

test_ui_serves_app() {
    run_test "UI Serves App" \
        "curl -sf $UI_URL/ | grep -q 'Social Media Scheduler'" \
        "Check if UI serves the React application"
}

test_platform_config() {
    run_test "Platform Configuration" \
        "curl -sf $API_URL/api/v1/auth/platforms | jq -e '.success == true and (.data | length) >= 4'" \
        "Check if platform configurations are available"
}

test_api_proxy() {
    run_test "API Proxy" \
        "curl -sf $UI_URL/api/v1/auth/platforms | jq -e '.success == true'" \
        "Check if UI properly proxies API requests"
}

test_user_registration() {
    local email="test-$(date +%s)@socialscheduler.test"
    local password="testpass123"
    
    local response=$(curl -sf -X POST "$API_URL/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$email\", \"password\": \"$password\", \"first_name\": \"Test\", \"last_name\": \"User\"}" 2>/dev/null || echo '{"success": false}')
    
    run_test "User Registration" \
        "echo '$response' | jq -e '.success == true'" \
        "Check if new users can register"
    
    # Store token for subsequent tests
    if echo "$response" | jq -e '.success == true' >/dev/null; then
        TEST_TOKEN=$(echo "$response" | jq -r '.data.token')
        TEST_USER_ID=$(echo "$response" | jq -r '.data.user.id')
    fi
}

test_demo_login() {
    local response=$(curl -sf -X POST "$API_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"email": "demo@vrooli.com", "password": "demo123"}' 2>/dev/null || echo '{"success": false}')
    
    run_test "Demo Login" \
        "echo '$response' | jq -e '.success == true'" \
        "Check if demo account login works"
    
    # Store demo token for protected endpoint tests
    if echo "$response" | jq -e '.success == true' >/dev/null; then
        DEMO_TOKEN=$(echo "$response" | jq -r '.data.token')
    fi
}

test_protected_endpoint_without_auth() {
    run_test "Protected Endpoint (No Auth)" \
        "curl -sf $API_URL/api/v1/auth/me 2>/dev/null | jq -e '.success == false'" \
        "Check if protected endpoints reject unauthenticated requests"
}

test_protected_endpoint_with_auth() {
    if [ -n "${DEMO_TOKEN:-}" ]; then
        run_test "Protected Endpoint (With Auth)" \
            "curl -sf -H 'Authorization: Bearer $DEMO_TOKEN' $API_URL/api/v1/auth/me | jq -e '.success == true'" \
            "Check if protected endpoints work with valid authentication"
    else
        warn "Skipping protected endpoint test - no auth token available"
    fi
}

test_post_scheduling() {
    if [ -n "${DEMO_TOKEN:-}" ]; then
        local future_date=$(date -d '+1 hour' -Iseconds | sed 's/+.*/Z/')
        local post_data="{
            \"title\": \"Integration Test Post\",
            \"content\": \"This is a test post from the integration test suite! 🧪 #testing\",
            \"platforms\": [\"twitter\", \"linkedin\"],
            \"scheduled_at\": \"$future_date\",
            \"timezone\": \"UTC\",
            \"auto_optimize\": true
        }"
        
        run_test "Post Scheduling" \
            "curl -sf -X POST -H 'Authorization: Bearer $DEMO_TOKEN' -H 'Content-Type: application/json' -d '$post_data' $API_URL/api/v1/posts/schedule | jq -e '.success == true'" \
            "Check if posts can be scheduled successfully"
    else
        warn "Skipping post scheduling test - no auth token available"
    fi
}

test_calendar_posts() {
    if [ -n "${DEMO_TOKEN:-}" ]; then
        local start_date=$(date -d '1 month ago' -I)
        local end_date=$(date -d '1 month' -I)
        
        run_test "Calendar Posts" \
            "curl -sf -H 'Authorization: Bearer $DEMO_TOKEN' '$API_URL/api/v1/posts/calendar?start_date=$start_date&end_date=$end_date' | jq -e '.success == true'" \
            "Check if calendar posts can be retrieved"
    else
        warn "Skipping calendar posts test - no auth token available"
    fi
}

test_database_connection() {
    local postgres_port="${RESOURCE_PORTS_postgres:-5432}"
    
    run_test "Database Connection" \
        "psql -h localhost -p $postgres_port -U postgres -d vrooli_social_media_scheduler -c 'SELECT COUNT(*) FROM users;' >/dev/null" \
        "Check if PostgreSQL database is accessible"
}

test_redis_connection() {
    local redis_port="${RESOURCE_PORTS_redis:-6379}"
    
    run_test "Redis Connection" \
        "redis-cli -h localhost -p $redis_port ping | grep -q PONG" \
        "Check if Redis is accessible"
}

test_ollama_connection() {
    local ollama_port="${RESOURCE_PORTS_ollama:-11434}"
    
    run_test "Ollama Connection" \
        "curl -sf http://localhost:$ollama_port/api/tags >/dev/null" \
        "Check if Ollama AI service is accessible"
}

test_minio_connection() {
    local minio_port="${RESOURCE_PORTS_minio:-9000}"
    
    run_test "MinIO Connection" \
        "curl -sf http://localhost:$minio_port/minio/health/live >/dev/null" \
        "Check if MinIO storage is accessible"
}

# Performance tests
test_api_performance() {
    log "Running API performance test..."
    
    local response_times=()
    for i in {1..10}; do
        local start_time=$(date +%s%3N)
        curl -sf "$API_URL/health" >/dev/null
        local end_time=$(date +%s%3N)
        local response_time=$((end_time - start_time))
        response_times+=($response_time)
    done
    
    local total=0
    for time in "${response_times[@]}"; do
        total=$((total + time))
    done
    local avg_time=$((total / 10))
    
    if [ $avg_time -lt 500 ]; then
        success "API Performance (avg: ${avg_time}ms)"
    else
        fail "API Performance (avg: ${avg_time}ms - exceeds 500ms threshold)"
    fi
    
    ((TESTS_RUN++))
}

# Main test execution
main() {
    echo ""
    log "🧪 Social Media Scheduler Integration Tests"
    log "==========================================="
    echo ""
    
    # Check dependencies
    if ! command -v curl >/dev/null; then
        fail "curl is required for tests"
        exit 1
    fi
    
    if ! command -v jq >/dev/null; then
        fail "jq is required for JSON parsing"
        exit 1
    fi
    
    # Basic connectivity tests
    log "🔗 Testing Basic Connectivity"
    test_api_health
    test_queue_health
    test_ui_health
    test_ui_serves_app
    test_platform_config
    test_api_proxy
    
    echo ""
    log "🗄️  Testing Resource Connectivity"
    test_database_connection
    test_redis_connection
    test_ollama_connection
    test_minio_connection
    
    echo ""
    log "🔐 Testing Authentication"
    test_user_registration
    test_demo_login
    test_protected_endpoint_without_auth
    test_protected_endpoint_with_auth
    
    echo ""
    log "📅 Testing Core Functionality"
    test_post_scheduling
    test_calendar_posts
    
    echo ""
    log "⚡ Testing Performance"
    test_api_performance
    
    # Summary
    echo ""
    log "📊 Test Summary"
    log "==============="
    echo ""
    
    if [ $TESTS_FAILED -eq 0 ]; then
        success "All tests passed! ✨"
        success "Tests run: $TESTS_RUN, Passed: $TESTS_PASSED, Failed: $TESTS_FAILED"
        echo ""
        log "🚀 Social Media Scheduler is ready for use!"
        echo ""
        log "Access your social media scheduler at:"
        log "  📅 Web Interface: $UI_URL"
        log "  🔗 API Endpoint: $API_URL"
        log "  🛠️  CLI: social-media-scheduler --help"
        
        # Show demo credentials
        echo ""
        log "🎯 Demo Account:"
        log "  Email: demo@vrooli.com"
        log "  Password: demo123"
    else
        fail "Some tests failed!"
        fail "Tests run: $TESTS_RUN, Passed: $TESTS_PASSED, Failed: $TESTS_FAILED"
        echo ""
        
        if [ $TESTS_PASSED -gt 0 ]; then
            warn "Partial functionality may still be available"
            warn "Check the failed tests above for specific issues"
        fi
        
        exit 1
    fi
}

# Run tests with timeout
timeout 300 main "$@" || {
    fail "Tests timed out after 5 minutes"
    exit 1
}