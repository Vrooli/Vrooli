#!/bin/bash

# Test Data Generator - Integration Test
# This script tests the full integration of the test-data-generator scenario

set -e

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_URL="${API_URL:-http://localhost:3001}"
UI_URL="${UI_URL:-http://localhost:3002}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}ℹ️  $*${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $*${NC}"
}

print_error() {
    echo -e "${RED}❌ $*${NC}"
}

# Test API integration
test_api_integration() {
    print_info "Testing API integration..."
    
    # Health check
    if ! curl -s "$API_URL/health" | grep -q "healthy"; then
        print_error "API health check failed"
        return 1
    fi
    
    # Generate test data
    local response=$(curl -s -X POST -H 'Content-Type: application/json' \
        -d '{"count":5,"format":"json"}' \
        "$API_URL/api/generate/users")
    
    if ! echo "$response" | grep -q '"success":true'; then
        print_error "User generation failed"
        return 1
    fi
    
    # Test custom schema
    local custom_response=$(curl -s -X POST -H 'Content-Type: application/json' \
        -d '{"count":2,"schema":{"id":"uuid","title":"string","active":"boolean"}}' \
        "$API_URL/api/generate/custom")
    
    if ! echo "$custom_response" | grep -q '"success":true'; then
        print_error "Custom schema generation failed"
        return 1
    fi
    
    print_success "API integration tests passed"
    return 0
}

# Test UI integration
test_ui_integration() {
    print_info "Testing UI integration..."
    
    # Health check
    if ! curl -s "$UI_URL/health" | grep -q "healthy"; then
        print_error "UI health check failed"
        return 1
    fi
    
    # Check main page loads
    if ! curl -s "$UI_URL/" | grep -q "Test Data Generator"; then
        print_error "UI main page failed to load"
        return 1
    fi
    
    print_success "UI integration tests passed"
    return 0
}

# Test CLI integration
test_cli_integration() {
    print_info "Testing CLI integration..."
    
    local cli="$SCENARIO_DIR/cli/test-data-generator"
    
    if [ ! -x "$cli" ]; then
        print_error "CLI script not found or not executable"
        return 1
    fi
    
    # Test CLI commands
    if ! "$cli" health | grep -q "healthy"; then
        print_error "CLI health check failed"
        return 1
    fi
    
    if ! "$cli" types | grep -q "Available data types"; then
        print_error "CLI types command failed"
        return 1
    fi
    
    print_success "CLI integration tests passed"
    return 0
}

# Test data consistency
test_data_consistency() {
    print_info "Testing data consistency with seeds..."
    
    local seed="test123"
    
    # Generate same data twice with same seed
    local data1=$(curl -s -X POST -H 'Content-Type: application/json' \
        -d "{\"count\":3,\"seed\":\"$seed\"}" \
        "$API_URL/api/generate/users")
    
    local data2=$(curl -s -X POST -H 'Content-Type: application/json' \
        -d "{\"count\":3,\"seed\":\"$seed\"}" \
        "$API_URL/api/generate/users")
    
    # Extract the data arrays and compare
    local hash1=$(echo "$data1" | jq -c '.data' | md5sum | cut -d' ' -f1)
    local hash2=$(echo "$data2" | jq -c '.data' | md5sum | cut -d' ' -f1)
    
    if [ "$hash1" != "$hash2" ]; then
        print_error "Data consistency test failed - same seed produced different results"
        return 1
    fi
    
    print_success "Data consistency tests passed"
    return 0
}

# Test error handling
test_error_handling() {
    print_info "Testing error handling..."
    
    # Test invalid endpoint
    local status=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/invalid")
    if [ "$status" != "404" ]; then
        print_error "Expected 404 for invalid endpoint, got $status"
        return 1
    fi
    
    # Test invalid data
    local error_response=$(curl -s -X POST -H 'Content-Type: application/json' \
        -d '{"count":-1}' \
        "$API_URL/api/generate/users")
    
    if ! echo "$error_response" | grep -q "error"; then
        print_error "Error handling test failed"
        return 1
    fi
    
    print_success "Error handling tests passed"
    return 0
}

# Main integration test
main() {
    print_info "Starting Test Data Generator integration tests..."
    echo
    
    local failed=0
    
    # Run all integration tests
    test_api_integration || failed=1
    test_ui_integration || failed=1
    test_cli_integration || failed=1
    
    # Only run these if basic tests pass
    if [ $failed -eq 0 ]; then
        test_data_consistency || failed=1
        test_error_handling || failed=1
    fi
    
    echo
    if [ $failed -eq 0 ]; then
        print_success "All integration tests passed! 🎉"
        echo
        print_info "Test Data Generator is fully functional:"
        print_info "  🌐 API: $API_URL"
        print_info "  🖥️  UI:  $UI_URL"
        print_info "  ⚡ CLI: test-data-generator"
        echo
        return 0
    else
        print_error "Some integration tests failed"
        return 1
    fi
}

main "$@"