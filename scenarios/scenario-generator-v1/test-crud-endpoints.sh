#!/usr/bin/env bash
# Test script for updateScenario and deleteScenario endpoints

set -euo pipefail

API_PORT="${API_PORT:-8080}"
API_URL="${API_URL:-http://localhost:${API_PORT}}"

# Colors
GREEN='\033[1;32m'
RED='\033[1;31m'
YELLOW='\033[1;33m'
CYAN='\033[1;36m'
NC='\033[0m'

info() { echo -e "${CYAN}ℹ️  $1${NC}"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; }
warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }

# Test Update Scenario
test_update_scenario() {
    info "Testing UPDATE scenario endpoint..."
    
    # First, create a test scenario
    info "Creating test scenario..."
    CREATE_RESPONSE=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Test Scenario for Update",
            "description": "Original description",
            "prompt": "Original prompt",
            "complexity": "simple",
            "category": "test"
        }' \
        "$API_URL/api/scenarios")
    
    if [ $? -ne 0 ]; then
        error "Failed to create test scenario"
        return 1
    fi
    
    # Extract scenario ID
    SCENARIO_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id')
    if [ -z "$SCENARIO_ID" ] || [ "$SCENARIO_ID" = "null" ]; then
        error "Failed to extract scenario ID"
        return 1
    fi
    
    info "Created scenario with ID: $SCENARIO_ID"
    
    # Test update with new name and description
    info "Updating scenario..."
    UPDATE_RESPONSE=$(curl -sf -X PUT \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Updated Test Scenario",
            "description": "Updated description",
            "complexity": "intermediate",
            "estimated_revenue": 25000
        }' \
        "$API_URL/api/scenarios/$SCENARIO_ID")
    
    if [ $? -ne 0 ]; then
        error "Failed to update scenario"
        return 1
    fi
    
    # Verify the update
    UPDATED_NAME=$(echo "$UPDATE_RESPONSE" | jq -r '.name')
    UPDATED_DESC=$(echo "$UPDATE_RESPONSE" | jq -r '.description')
    UPDATED_COMPLEXITY=$(echo "$UPDATE_RESPONSE" | jq -r '.complexity')
    
    if [ "$UPDATED_NAME" = "Updated Test Scenario" ] && \
       [ "$UPDATED_DESC" = "Updated description" ] && \
       [ "$UPDATED_COMPLEXITY" = "intermediate" ]; then
        success "Scenario updated successfully!"
        success "  - Name: $UPDATED_NAME"
        success "  - Description: $UPDATED_DESC"
        success "  - Complexity: $UPDATED_COMPLEXITY"
        return 0
    else
        error "Update verification failed"
        echo "Response: $UPDATE_RESPONSE"
        return 1
    fi
}

# Test Delete Scenario
test_delete_scenario() {
    info "Testing DELETE scenario endpoint..."
    
    # Create a test scenario to delete
    info "Creating test scenario for deletion..."
    CREATE_RESPONSE=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Test Scenario for Deletion",
            "description": "This will be deleted",
            "prompt": "Test prompt",
            "complexity": "simple",
            "category": "test"
        }' \
        "$API_URL/api/scenarios")
    
    if [ $? -ne 0 ]; then
        error "Failed to create test scenario"
        return 1
    fi
    
    # Extract scenario ID
    SCENARIO_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id')
    if [ -z "$SCENARIO_ID" ] || [ "$SCENARIO_ID" = "null" ]; then
        error "Failed to extract scenario ID"
        return 1
    fi
    
    info "Created scenario with ID: $SCENARIO_ID"
    
    # Delete the scenario
    info "Deleting scenario..."
    DELETE_RESPONSE=$(curl -sf -X DELETE -w "\n%{http_code}" "$API_URL/api/scenarios/$SCENARIO_ID")
    HTTP_CODE=$(echo "$DELETE_RESPONSE" | tail -n 1)
    
    if [ "$HTTP_CODE" = "204" ]; then
        success "Scenario deleted successfully (204 No Content)"
    else
        error "Failed to delete scenario (HTTP $HTTP_CODE)"
        return 1
    fi
    
    # Verify deletion by trying to fetch it
    info "Verifying deletion..."
    FETCH_RESPONSE=$(curl -sf -w "\n%{http_code}" "$API_URL/api/scenarios/$SCENARIO_ID" 2>/dev/null || true)
    HTTP_CODE=$(echo "$FETCH_RESPONSE" | tail -n 1)
    
    if [ "$HTTP_CODE" = "404" ]; then
        success "Deletion verified - scenario not found (404)"
        return 0
    else
        error "Scenario still exists after deletion (HTTP $HTTP_CODE)"
        return 1
    fi
}

# Test invalid operations
test_invalid_operations() {
    info "Testing error handling..."
    
    # Test update on non-existent scenario
    info "Testing update on non-existent scenario..."
    UPDATE_RESPONSE=$(curl -sf -X PUT -w "\n%{http_code}" \
        -H "Content-Type: application/json" \
        -d '{"name": "Should fail"}' \
        "$API_URL/api/scenarios/00000000-0000-0000-0000-000000000000" 2>/dev/null || true)
    HTTP_CODE=$(echo "$UPDATE_RESPONSE" | tail -n 1)
    
    if [ "$HTTP_CODE" = "404" ]; then
        success "Correctly returned 404 for non-existent scenario update"
    else
        warn "Unexpected response code: $HTTP_CODE"
    fi
    
    # Test delete on non-existent scenario
    info "Testing delete on non-existent scenario..."
    DELETE_RESPONSE=$(curl -sf -X DELETE -w "\n%{http_code}" \
        "$API_URL/api/scenarios/00000000-0000-0000-0000-000000000000" 2>/dev/null || true)
    HTTP_CODE=$(echo "$DELETE_RESPONSE" | tail -n 1)
    
    if [ "$HTTP_CODE" = "404" ]; then
        success "Correctly returned 404 for non-existent scenario delete"
    else
        warn "Unexpected response code: $HTTP_CODE"
    fi
    
    return 0
}

# Main test execution
main() {
    echo "=========================================="
    info "🧪 Testing CRUD Endpoints for Scenario Generator"
    echo "=========================================="
    echo
    
    # Check if API is running
    if ! curl -sf "$API_URL/health" > /dev/null; then
        error "API is not running at $API_URL"
        echo "Please start the API server first"
        exit 1
    fi
    
    # Check if jq is installed
    if ! command -v jq &> /dev/null; then
        error "jq is required but not installed"
        echo "Install with: apt-get install jq (or brew install jq on macOS)"
        exit 1
    fi
    
    local total_tests=0
    local passed_tests=0
    
    # Run tests
    total_tests=$((total_tests + 1))
    if test_update_scenario; then
        passed_tests=$((passed_tests + 1))
    fi
    echo
    
    total_tests=$((total_tests + 1))
    if test_delete_scenario; then
        passed_tests=$((passed_tests + 1))
    fi
    echo
    
    total_tests=$((total_tests + 1))
    if test_invalid_operations; then
        passed_tests=$((passed_tests + 1))
    fi
    echo
    
    # Summary
    echo "=========================================="
    if [ $passed_tests -eq $total_tests ]; then
        success "🎉 All tests passed! ($passed_tests/$total_tests)"
        success "UPDATE and DELETE endpoints are working correctly!"
    else
        error "Some tests failed ($passed_tests/$total_tests passed)"
        exit 1
    fi
}

# Run tests
main "$@"