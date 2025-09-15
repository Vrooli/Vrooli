#!/bin/bash
# Business logic tests phase - <180 seconds
# Tests end-to-end business workflows and scenario-specific functionality
set -euo pipefail

# Setup paths and utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "=== Business Logic Tests Phase ==="
start_time=$(date +%s)

error_count=0
test_count=0
scenario_name=$(basename "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)")

# Cleanup function to run on exit
cleanup() {
    echo "Cleaning up test data..."
    # Add scenario-specific cleanup here
    # Example: Clean up test records, temporary files, etc.
}

# Set up cleanup to run on script exit
trap cleanup EXIT

# Source connectivity testing module
source "$APP_ROOT/scripts/scenarios/testing/shell/connectivity.sh"

# Get dynamic URLs using the testing library
API_BASE_URL=$(testing::connectivity::get_api_url "$scenario_name" 2>/dev/null || echo "")
UI_BASE_URL=$(testing::connectivity::get_ui_url "$scenario_name" 2>/dev/null || echo "")

if [ -n "$API_BASE_URL" ]; then
    echo "Using API base URL: $API_BASE_URL"
fi

# Pre-cleanup: Remove any existing test data
echo "Cleaning up any existing test data..."
# Add scenario-specific pre-cleanup here
# Example: Delete test entities, reset test state, etc.

# Test core business functionality
echo "🧪 Testing core business functionality..."

if [ -n "$API_BASE_URL" ] && [ -d "api" ]; then
    # Example business test - customize for your scenario
    echo "  Testing core workflow..."
    
    # NOTE: Replace this with actual business logic tests
    # Example for a typical CRUD scenario:
    
    # Test 1: Create operation
    # echo "    Testing entity creation..."
    # create_response=$(curl -s -X POST "$API_BASE_URL/api/v1/entities" \
    #     -H "Content-Type: application/json" \
    #     -d '{"name":"test-entity","description":"Test entity for business testing"}' \
    #     2>/dev/null || echo "")
    # 
    # if echo "$create_response" | jq -e '.id' >/dev/null 2>&1; then
    #     entity_id=$(echo "$create_response" | jq -r '.id')
    #     log::success "     ✅ Entity creation successful - ID: $entity_id"
    #     test_count=$((test_count + 1))
    #     
    #     # Test 2: Read operation
    #     echo "    Testing entity retrieval..."
    #     get_response=$(curl -s "$API_BASE_URL/api/v1/entities/$entity_id" 2>/dev/null || echo "")
    #     if echo "$get_response" | jq -e '.id' >/dev/null 2>&1; then
    #         log::success "     ✅ Entity retrieval successful"
    #         test_count=$((test_count + 1))
    #     else
    #         log::error "     ❌ Entity retrieval failed"
    #         error_count=$((error_count + 1))
    #     fi
    #     
    #     # Test 3: Update operation
    #     echo "    Testing entity update..."
    #     update_response=$(curl -s -X PUT "$API_BASE_URL/api/v1/entities/$entity_id" \
    #         -H "Content-Type: application/json" \
    #         -d '{"name":"updated-entity","description":"Updated test entity"}' \
    #         2>/dev/null || echo "")
    #     if echo "$update_response" | jq -e '.id' >/dev/null 2>&1; then
    #         log::success "     ✅ Entity update successful"
    #         test_count=$((test_count + 1))
    #     else
    #         log::error "     ❌ Entity update failed"
    #         error_count=$((error_count + 1))
    #     fi
    #     
    #     # Test 4: Delete operation
    #     echo "    Testing entity deletion..."
    #     if curl -s -X DELETE "$API_BASE_URL/api/v1/entities/$entity_id" >/dev/null 2>&1; then
    #         log::success "     ✅ Entity deletion successful"
    #         test_count=$((test_count + 1))
    #     else
    #         log::error "     ❌ Entity deletion failed"
    #         error_count=$((error_count + 1))
    #     fi
    # else
    #     log::error "     ❌ Entity creation failed"
    #     error_count=$((error_count + 1))
    # fi
    
    log::warning "   ⚠️  Business tests need customization for $scenario_name"
    log::info "   💡 Customize business logic tests in test/phases/test-business.sh"
    log::info "   💡 Add scenario-specific workflows, data validation, and edge cases"
    
else
    log::warning "⚠️  API not available, skipping API business tests"
fi

# Test CLI business workflows (if CLI exists)
if [ -d "cli" ]; then
    echo "🧪 Testing CLI business workflows..."
    cli_binary="cli/$scenario_name"
    
    if [ -f "$cli_binary" ] && [ -x "$cli_binary" ]; then
        # Test CLI core functionality
        echo "  Testing CLI core operations..."
        
        # Example: Test basic CLI commands
        # Customize these for your scenario's CLI commands
        
        # if "$cli_binary" list >/dev/null 2>&1; then
        #     log::success "   ✅ CLI list command works"
        #     test_count=$((test_count + 1))
        # else
        #     log::warning "   ⚠️  CLI list command needs implementation"
        # fi
        
        log::info "   💡 Add CLI-specific business workflow tests"
        test_count=$((test_count + 1))  # Placeholder
    else
        log::info "   ℹ️  CLI binary not available for business testing"
    fi
else
    log::info "ℹ️  No CLI directory found, skipping CLI business tests"
fi

# Test data persistence and integrity
echo "🧪 Testing data persistence..."
if [ -n "$API_PORT" ]; then
    # Test that data persists across operations
    # Example: Create data, restart service, verify data still exists
    # NOTE: This is a template - customize for your scenario
    
    log::info "   💡 Add data persistence tests for $scenario_name"
    log::info "   💡 Test data integrity, state management, and recovery"
    
    # Placeholder success for template
    test_count=$((test_count + 1))
else
    log::info "ℹ️  API not available, skipping data persistence tests"
fi

# Test error handling and edge cases
echo "🧪 Testing error handling..."
if [ -n "$API_PORT" ]; then
    # Test various error conditions
    echo "  Testing invalid requests..."
    
    # Test invalid endpoint
    invalid_response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/nonexistent" 2>/dev/null | tail -c 3)
    if [ "$invalid_response" = "404" ]; then
        log::success "   ✅ 404 error handling works"
        test_count=$((test_count + 1))
    else
        log::warning "   ⚠️  404 error handling unexpected: $invalid_response"
    fi
    
    # Test malformed JSON (if API accepts JSON)
    # malformed_response=$(curl -s -w "%{http_code}" -X POST "$API_BASE_URL/api/v1/entities" \
    #     -H "Content-Type: application/json" \
    #     -d '{"invalid":json}' 2>/dev/null | tail -c 3)
    # if [ "$malformed_response" = "400" ]; then
    #     log::success "   ✅ Bad request error handling works"
    #     test_count=$((test_count + 1))
    # else
    #     log::warning "   ⚠️  Bad request handling unexpected: $malformed_response"
    # fi
    
    log::info "   💡 Add comprehensive error handling tests"
else
    log::info "ℹ️  API not available, skipping error handling tests"
fi

# Test integration with other scenarios (if applicable)
echo "🧪 Testing scenario integrations..."
# Add tests for how this scenario integrates with others
# Example: If this scenario provides data to other scenarios
log::info "   💡 Add inter-scenario integration tests if applicable"

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))
echo ""

if [ $error_count -eq 0 ]; then
    log::success "✅ Business logic tests completed successfully in ${duration}s"
    log::success "   Tests run: $test_count, Errors: $error_count"
    
    if [ $test_count -eq 0 ]; then
        log::warning "⚠️  No business tests were actually run - customize this phase!"
    fi
else
    log::error "❌ Business logic tests failed with $error_count errors in ${duration}s"
    log::error "   Tests run: $test_count, Errors: $error_count"
    echo ""
    log::info "💡 Troubleshooting tips:"
    echo "   • Ensure scenario is fully running: vrooli scenario start $scenario_name"
    echo "   • Check scenario logs: vrooli scenario logs $scenario_name"
    echo "   • Verify all resources are healthy: vrooli resource status"
    echo "   • Review and customize business logic tests for your scenario"
fi

if [ $duration -gt 180 ]; then
    log::warning "⚠️  Business phase exceeded 180s target"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi