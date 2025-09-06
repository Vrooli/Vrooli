#!/usr/bin/env bash
# Test the scenario generation pipeline

set -euo pipefail

# Colors
GREEN='\033[1;32m'
RED='\033[1;31m'
YELLOW='\033[1;33m'
CYAN='\033[1;36m'
NC='\033[0m'

API_URL="http://localhost:8080"

info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

# Test health endpoint
test_health() {
    info "Testing health endpoint..."
    response=$(curl -s "$API_URL/health")
    
    if echo "$response" | grep -q '"status":"healthy"'; then
        success "API is healthy"
        
        if echo "$response" | grep -q '"pipeline":"active"'; then
            success "Pipeline is active"
        else
            error "Pipeline is not active"
            return 1
        fi
    else
        error "API health check failed"
        return 1
    fi
}

# Test generation endpoint
test_generation() {
    info "Testing scenario generation..."
    
    # Create a simple generation request
    request_json='{
        "name": "test-scenario",
        "description": "A simple test scenario for validation",
        "prompt": "Create a basic todo list application",
        "complexity": "simple",
        "category": "productivity"
    }'
    
    info "Sending generation request..."
    response=$(curl -s -X POST "$API_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d "$request_json")
    
    if echo "$response" | grep -q '"scenario_id"'; then
        scenario_id=$(echo "$response" | grep -o '"scenario_id":"[^"]*' | cut -d'"' -f4)
        success "Generation started with ID: $scenario_id"
        
        # Wait and check status
        info "Waiting 5 seconds before checking status..."
        sleep 5
        
        status_response=$(curl -s "$API_URL/api/generate/status/$scenario_id")
        if echo "$status_response" | grep -q '"status"'; then
            status=$(echo "$status_response" | grep -o '"status":"[^"]*' | cut -d'"' -f4)
            info "Current status: $status"
            
            if [ "$status" = "completed" ]; then
                success "Scenario generation completed!"
            elif [ "$status" = "failed" ]; then
                error "Scenario generation failed"
                return 1
            else
                info "Generation still in progress (status: $status)"
                info "Check again later with: curl $API_URL/api/generate/status/$scenario_id"
            fi
        fi
        
        return 0
    else
        error "Failed to start generation"
        echo "Response: $response"
        return 1
    fi
}

# Test backlog
test_backlog() {
    info "Testing backlog endpoint..."
    
    response=$(curl -s "$API_URL/api/backlog")
    
    if echo "$response" | grep -q '"pending"'; then
        success "Backlog endpoint working"
        
        # Count items
        pending_count=$(echo "$response" | grep -o '"pending":\[[^]]*' | grep -o '"id"' | wc -l)
        info "Found $pending_count items in pending backlog"
    else
        error "Failed to fetch backlog"
        return 1
    fi
}

# Main test execution
main() {
    echo -e "${CYAN}🧪 Testing Scenario Generator Pipeline${NC}"
    echo "=================================="
    
    # Run tests
    test_health || exit 1
    echo
    
    test_backlog || exit 1
    echo
    
    test_generation || exit 1
    echo
    
    success "All tests completed!"
    echo
    echo "To monitor the API logs, run:"
    echo "  tail -f /path/to/api/logs"
    echo
    echo "To manually test generation:"
    echo "  curl -X POST $API_URL/api/generate -H 'Content-Type: application/json' -d '{...}'"
}

# Run main
main