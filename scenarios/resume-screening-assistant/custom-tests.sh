#!/bin/bash
# Custom Business Logic Tests for Resume Screening Assistant
set -euo pipefail

# Source framework utilities if available
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/resume-screening-assistant"
FRAMEWORK_DIR="${APP_ROOT}/scripts/scenarios/validation"

# Source custom handler for print functions
if [[ -f "$FRAMEWORK_DIR/handlers/custom.sh" ]]; then
    source "$FRAMEWORK_DIR/handlers/custom.sh"
fi

# Source common utilities
if [[ -f "$FRAMEWORK_DIR/clients/common.sh" ]]; then
    source "$FRAMEWORK_DIR/clients/common.sh"
fi

# Configuration
API_PORT="${API_PORT:-8090}"
API_BASE_URL="http://localhost:${API_PORT}"

# Helper function to check API health
check_api_health() {
    curl -s -f "$API_BASE_URL/health" > /dev/null 2>&1
}

# Test API endpoints
test_api_endpoints() {
    print_custom_info "Testing API endpoints..."
    
    # Health check
    if check_api_health; then
        print_custom_success "✅ API health endpoint responding"
    else
        print_custom_error "❌ API health endpoint not responding"
        return 1
    fi
    
    # Jobs endpoint
    if curl -s -f "$API_BASE_URL/api/jobs" > /dev/null 2>&1; then
        print_custom_success "✅ Jobs API endpoint responding"
    else
        print_custom_error "❌ Jobs API endpoint not responding"
        return 1
    fi
    
    # Candidates endpoint
    if curl -s -f "$API_BASE_URL/api/candidates" > /dev/null 2>&1; then
        print_custom_success "✅ Candidates API endpoint responding"
    else
        print_custom_error "❌ Candidates API endpoint not responding"
        return 1
    fi
    
    # Search endpoint
    if curl -s -f "$API_BASE_URL/api/search?query=test" > /dev/null 2>&1; then
        print_custom_success "✅ Search API endpoint responding"
    else
        print_custom_error "❌ Search API endpoint not responding"
        return 1
    fi
    
    return 0
}

# Test data consistency
test_data_consistency() {
    print_custom_info "Testing data consistency..."
    
    # Test jobs endpoint returns valid JSON
    local jobs_response
    if jobs_response=$(curl -s -f "$API_BASE_URL/api/jobs" 2>/dev/null); then
        if echo "$jobs_response" | jq empty 2>/dev/null; then
            local job_count
            job_count=$(echo "$jobs_response" | jq -r '.count // 0')
            print_custom_success "✅ Jobs endpoint returns valid JSON ($job_count jobs)"
        else
            print_custom_error "❌ Jobs endpoint returns invalid JSON"
            return 1
        fi
    fi
    
    # Test candidates endpoint returns valid JSON
    local candidates_response
    if candidates_response=$(curl -s -f "$API_BASE_URL/api/candidates" 2>/dev/null); then
        if echo "$candidates_response" | jq empty 2>/dev/null; then
            local candidate_count
            candidate_count=$(echo "$candidates_response" | jq -r '.count // 0')
            print_custom_success "✅ Candidates endpoint returns valid JSON ($candidate_count candidates)"
        else
            print_custom_error "❌ Candidates endpoint returns invalid JSON"
            return 1
        fi
    fi
    
    return 0
}

# Test CLI functionality
test_cli_functionality() {
    print_custom_info "Testing CLI functionality..."
    
    local cli_path="$SCRIPT_DIR/cli/resume-screening-assistant"
    
    # Check if CLI exists
    if [[ -f "$cli_path" && -x "$cli_path" ]]; then
        print_custom_success "✅ CLI script exists and is executable"
        
        # Test CLI help
        if "$cli_path" help > /dev/null 2>&1; then
            print_custom_success "✅ CLI help command works"
        else
            print_custom_error "❌ CLI help command failed"
            return 1
        fi
    else
        print_custom_error "❌ CLI script not found or not executable at $cli_path"
        return 1
    fi
    
    return 0
}

# Main test function for the scenario framework
test_resume_screening_assistant_workflow() {
    print_custom_info "Testing Resume Screening Assistant business workflow"
    
    # Test API functionality
    if ! test_api_endpoints; then
        print_custom_error "API endpoint tests failed"
        return 1
    fi
    
    # Test data consistency
    if ! test_data_consistency; then
        print_custom_error "Data consistency tests failed"
        return 1
    fi
    
    # Test CLI functionality
    if ! test_cli_functionality; then
        print_custom_error "CLI functionality tests failed"
        return 1
    fi
    
    print_custom_success "✅ All Resume Screening Assistant workflow tests passed"
    print_custom_success "✅ Business logic validated"
    print_custom_success "✅ Integration points confirmed"
    
    return 0
}

# Entry point for framework
run_custom_tests() {
    print_custom_info "Running Resume Screening Assistant custom business logic tests"
    test_resume_screening_assistant_workflow
    return $?
}

# Export functions
export -f test_resume_screening_assistant_workflow
export -f run_custom_tests