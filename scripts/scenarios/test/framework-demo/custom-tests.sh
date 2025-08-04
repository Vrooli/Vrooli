#!/bin/bash
# Custom tests for framework demo

set -euo pipefail

# Test resource discovery functionality
test_resource_discovery() {
    print_custom_info "Testing resource discovery system"
    
    # Test get_resource_url function
    local ollama_url=""
    if declare -f get_resource_url >/dev/null 2>&1; then
        ollama_url=$(get_resource_url "ollama")
        if [[ -n "$ollama_url" ]]; then
            print_custom_success "Resource URL discovered: $ollama_url"
        else
            print_custom_info "No URL found for ollama (expected for demo)"
        fi
    else
        print_custom_warning "get_resource_url function not available"
    fi
    
    # Test that framework utilities are loaded
    local functions_loaded=0
    local expected_functions=(
        "get_resource_url"
        "is_resource_available" 
        "check_url_health"
        "make_http_request"
        "print_info"
        "print_success"
    )
    
    for func in "${expected_functions[@]}"; do
        if declare -f "$func" >/dev/null 2>&1; then
            ((functions_loaded++))
        fi
    done
    
    print_custom_info "Framework functions loaded: $functions_loaded/${#expected_functions[@]}"
    
    if [[ $functions_loaded -ge 4 ]]; then
        print_custom_success "Framework utilities loaded successfully"
        return 0
    else
        print_custom_error "Insufficient framework functions loaded"
        return 1
    fi
}

# Main function called by framework
run_custom_tests() {
    print_custom_info "Running framework demo custom tests"
    
    local tests_passed=0
    local tests_failed=0
    
    # Run resource discovery test
    if test_resource_discovery; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Report results
    print_custom_info "Custom tests completed: $tests_passed passed, $tests_failed failed"
    
    # Return success if all tests passed
    [[ $tests_failed -eq 0 ]]
}

# Export functions
export -f run_custom_tests
export -f test_resource_discovery