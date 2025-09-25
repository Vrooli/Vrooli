#!/bin/bash

# Test script for Data Structurer Schema API endpoints

set -euo pipefail

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:${API_PORT:-15770}}"
SCHEMA_DIR="$(dirname "$0")/../examples"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[TEST INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[TEST PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[TEST FAIL]${NC} $1" >&2
}

# Test helpers
test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local expected_status="${3:-200}"
    local data="${4:-}"
    
    local response
    local status_code
    
    if [[ "$method" == "GET" ]]; then
        response=$(curl -s -w "\n%{http_code}" "$API_BASE_URL$endpoint")
    elif [[ "$method" == "POST" ]]; then
        response=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    elif [[ "$method" == "DELETE" ]]; then
        response=$(curl -s -w "\n%{http_code}" -X DELETE "$API_BASE_URL$endpoint")
    fi
    
    status_code=$(echo "$response" | tail -1)
    response_body=$(echo "$response" | sed '$d')
    
    if [[ "$status_code" == "$expected_status" ]]; then
        log_success "$method $endpoint - Status: $status_code"
        echo "$response_body"
        return 0
    else
        log_error "$method $endpoint - Expected: $expected_status, Got: $status_code"
        echo "$response_body" >&2
        return 1
    fi
}

# Main test function
main() {
    log_info "Starting Data Structurer Schema API tests..."
    
    local test_count=0
    local pass_count=0
    
    # Test 1: Health check
    log_info "Test 1: Health check"
    test_count=$((test_count + 1))
    if test_endpoint "GET" "/health" "200" > /dev/null; then
        pass_count=$((pass_count + 1))
    fi
    
    # Test 2: List schemas (should work even if empty)
    log_info "Test 2: List schemas"
    test_count=$((test_count + 1))
    if test_endpoint "GET" "/api/v1/schemas" "200" > /dev/null; then
        pass_count=$((pass_count + 1))
    fi
    
    # Test 3: Create schema
    log_info "Test 3: Create schema"
    test_count=$((test_count + 1))
    local schema_data
    if [[ -f "$SCHEMA_DIR/contact-schema.json" ]]; then
        schema_data=$(jq -n \
            --arg name "test-contact-schema" \
            --arg description "Test contact schema for API validation" \
            --argjson schema_definition "$(cat "$SCHEMA_DIR/contact-schema.json")" \
            '{
                name: $name,
                description: $description,
                schema_definition: $schema_definition
            }')
        
        local create_response
        if create_response=$(test_endpoint "POST" "/api/v1/schemas" "201" "$schema_data"); then
            pass_count=$((pass_count + 1))
            
            # Extract schema ID for further tests
            local schema_id
            schema_id=$(echo "$create_response" | jq -r '.id')
            
            # Test 4: Get created schema
            log_info "Test 4: Get created schema"
            test_count=$((test_count + 1))
            if test_endpoint "GET" "/api/v1/schemas/$schema_id" "200" > /dev/null; then
                pass_count=$((pass_count + 1))
            fi
            
            # Test 5: Delete schema
            log_info "Test 5: Delete schema"
            test_count=$((test_count + 1))
            if test_endpoint "DELETE" "/api/v1/schemas/$schema_id" "200" > /dev/null; then
                pass_count=$((pass_count + 1))
            fi
        fi
    else
        log_error "Schema file not found: $SCHEMA_DIR/contact-schema.json"
    fi
    
    # Test 6: List schema templates
    log_info "Test 6: List schema templates"
    test_count=$((test_count + 1))
    if test_endpoint "GET" "/api/v1/schema-templates" "200" > /dev/null; then
        pass_count=$((pass_count + 1))
    fi
    
    # Test 7: Invalid schema creation (should fail)
    log_info "Test 7: Invalid schema creation"
    test_count=$((test_count + 1))
    local invalid_data='{"name": "", "schema_definition": "invalid"}'
    if test_endpoint "POST" "/api/v1/schemas" "400" "$invalid_data" > /dev/null 2>&1; then
        pass_count=$((pass_count + 1))
    fi
    
    # Test 8: Get non-existent schema (should 404)
    log_info "Test 8: Get non-existent schema"
    test_count=$((test_count + 1))
    local fake_uuid="123e4567-e89b-12d3-a456-426614174000"
    if test_endpoint "GET" "/api/v1/schemas/$fake_uuid" "404" > /dev/null 2>&1; then
        pass_count=$((pass_count + 1))
    fi
    
    # Summary
    echo
    log_info "Test Summary:"
    echo "Total Tests: $test_count"
    echo "Passed: $pass_count"
    echo "Failed: $((test_count - pass_count))"
    
    if [[ $pass_count -eq $test_count ]]; then
        log_success "All schema API tests passed!"
        exit 0
    else
        log_error "Some tests failed!"
        exit 1
    fi
}

# Run tests
main "$@"