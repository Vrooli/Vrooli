#!/usr/bin/env bash
# Matrix Synapse Resource - Test Library Functions

set -euo pipefail

# Load core functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh"

# Test utilities
test_passed() {
    echo "✓ $1"
    return 0
}

test_failed() {
    echo "✗ $1: $2"
    return 1
}

# Smoke test - Basic health check
test_smoke() {
    log_info "Running smoke tests..."
    
    local failed=0
    
    # Test 1: Service is running
    if is_running; then
        test_passed "Service is running"
    else
        test_failed "Service is running" "Process not found"
        ((failed++))
    fi
    
    # Test 2: Health endpoint responds
    if timeout 5 curl -sf "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/versions" &>/dev/null; then
        test_passed "Health endpoint responds"
    else
        test_failed "Health endpoint responds" "No response from API"
        ((failed++))
    fi
    
    # Test 3: Database connection
    if PGPASSWORD="${SYNAPSE_DB_PASSWORD}" psql -h "${MATRIX_SYNAPSE_DB_HOST}" -p "${MATRIX_SYNAPSE_DB_PORT}" -U "${MATRIX_SYNAPSE_DB_USER}" -d "${MATRIX_SYNAPSE_DB_NAME}" -c "SELECT 1" &>/dev/null; then
        test_passed "Database connection works"
    else
        test_failed "Database connection works" "Cannot connect to PostgreSQL"
        ((failed++))
    fi
    
    # Test 4: Configuration file exists
    if [[ -f "${MATRIX_SYNAPSE_CONFIG_DIR}/homeserver.yaml" ]]; then
        test_passed "Configuration file exists"
    else
        test_failed "Configuration file exists" "homeserver.yaml not found"
        ((failed++))
    fi
    
    return $failed
}

# Integration tests - Full functionality
test_integration() {
    log_info "Running integration tests..."
    
    local failed=0
    
    # Test 1: Create test user
    local test_user="test_user_$(date +%s)"
    local test_pass="TestPass123!"
    
    if create_user "$test_user" "$test_pass" false &>/dev/null; then
        test_passed "User creation works"
    else
        test_failed "User creation works" "Failed to create user"
        ((failed++))
    fi
    
    # Test 2: Login with test user
    local login_response
    login_response=$(curl -sf -X POST "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/v3/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"type\": \"m.login.password\",
            \"user\": \"${test_user}\",
            \"password\": \"${test_pass}\"
        }" 2>/dev/null)
    
    if [[ -n "$login_response" ]] && echo "$login_response" | grep -q "access_token"; then
        test_passed "User login works"
        local access_token
        access_token=$(echo "$login_response" | grep -oP '"access_token"\s*:\s*"\K[^"]+')
    else
        test_failed "User login works" "Login failed"
        ((failed++))
    fi
    
    # Test 3: Create room
    if [[ -n "${access_token:-}" ]]; then
        local room_response
        room_response=$(curl -sf -X POST "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/v3/createRoom" \
            -H "Authorization: Bearer ${access_token}" \
            -H "Content-Type: application/json" \
            -d '{"name": "Test Room"}' 2>/dev/null)
        
        if [[ -n "$room_response" ]] && echo "$room_response" | grep -q "room_id"; then
            test_passed "Room creation works"
            local room_id
            room_id=$(echo "$room_response" | grep -oP '"room_id"\s*:\s*"\K[^"]+')
        else
            test_failed "Room creation works" "Failed to create room"
            ((failed++))
        fi
    else
        test_failed "Room creation works" "No access token"
        ((failed++))
    fi
    
    # Test 4: Send message
    if [[ -n "${access_token:-}" ]] && [[ -n "${room_id:-}" ]]; then
        local msg_response
        msg_response=$(curl -sf -X PUT "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/v3/rooms/${room_id}/send/m.room.message/$(date +%s)" \
            -H "Authorization: Bearer ${access_token}" \
            -H "Content-Type: application/json" \
            -d '{"msgtype": "m.text", "body": "Test message"}' 2>/dev/null)
        
        if [[ -n "$msg_response" ]] && echo "$msg_response" | grep -q "event_id"; then
            test_passed "Message sending works"
        else
            test_failed "Message sending works" "Failed to send message"
            ((failed++))
        fi
    else
        test_failed "Message sending works" "No room or token"
        ((failed++))
    fi
    
    # Test 5: Admin API access
    local admin_response
    admin_response=$(curl -sf "http://localhost:${MATRIX_SYNAPSE_PORT}/_synapse/admin/v1/server_version" 2>/dev/null)
    
    if [[ -n "$admin_response" ]]; then
        test_passed "Admin API accessible"
    else
        test_failed "Admin API accessible" "Cannot access admin endpoints"
        ((failed++))
    fi
    
    return $failed
}

# Unit tests - Library functions
test_unit() {
    log_info "Running unit tests..."
    
    local failed=0
    
    # Test 1: Configuration loading
    if [[ -n "${MATRIX_SYNAPSE_PORT:-}" ]]; then
        test_passed "Configuration loads correctly"
    else
        test_failed "Configuration loads correctly" "Port not set"
        ((failed++))
    fi
    
    # Test 2: Directory creation
    ensure_directories
    if [[ -d "${MATRIX_SYNAPSE_DATA_DIR}" ]]; then
        test_passed "Directory creation works"
    else
        test_failed "Directory creation works" "Data directory not created"
        ((failed++))
    fi
    
    # Test 3: Status detection
    local status
    status=$(get_status)
    if [[ -n "$status" ]]; then
        test_passed "Status detection works (status: $status)"
    else
        test_failed "Status detection works" "Cannot determine status"
        ((failed++))
    fi
    
    # Test 4: Health check function
    if declare -f check_health &>/dev/null; then
        test_passed "Health check function exists"
    else
        test_failed "Health check function exists" "Function not defined"
        ((failed++))
    fi
    
    return $failed
}

# Run all tests
test_all() {
    local total_failed=0
    
    log_info "Running all tests..."
    
    # Run test suites
    test_smoke || ((total_failed+=$?))
    test_integration || ((total_failed+=$?))
    test_unit || ((total_failed+=$?))
    
    # Summary
    if [[ $total_failed -eq 0 ]]; then
        log_info "All tests passed!"
    else
        log_error "$total_failed test(s) failed"
    fi
    
    return $total_failed
}

# Export test functions
export -f test_smoke test_integration test_unit test_all
export -f test_passed test_failed