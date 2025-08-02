#!/bin/bash

# Basic Integration Test Template
# Simple template for resource integration validation

source ../../../framework/helpers/test-helpers.sh

SCENARIO_ID="basic-integration-test"
TEST_TIMEOUT=${TEST_TIMEOUT:-300}

log_info "Starting basic integration test"

#####################################
# RESOURCE VALIDATION
#####################################

validate_resources() {
    log_info "Validating resource availability"
    
    # Example: Test Ollama (replace with your required resources)
    check_service_health "ollama" "http://localhost:11434" || {
        log_error "Ollama not available"
        return 1
    }
    
    # Example: Test optional PostgreSQL (replace with your optional resources)
    if check_service_health "postgres" "postgresql://postgres@localhost:5432/vrooli"; then
        log_info "PostgreSQL available - enhanced features enabled"
        export POSTGRES_AVAILABLE=true
    else
        log_info "PostgreSQL not available - basic features only"
        export POSTGRES_AVAILABLE=false
    fi
    
    log_success "Resource validation completed"
    return 0
}

#####################################
# BASIC FUNCTIONALITY TESTS
#####################################

test_basic_functionality() {
    log_info "Testing basic functionality"
    
    # Example: Test AI response (replace with your functionality)
    log_info "Testing AI response capability"
    
    local response=$(curl -s -X POST http://localhost:11434/api/generate \
        -d '{"model": "llama3.1:8b", "prompt": "Hello, respond with just OK", "stream": false}' \
        -H "Content-Type: application/json")
    
    if echo "$response" | grep -q "OK"; then
        log_success "AI response test passed"
    else
        log_error "AI response test failed"
        return 1
    fi
    
    # Example: Test database if available (replace with your functionality)
    if [ "$POSTGRES_AVAILABLE" = "true" ]; then
        log_info "Testing database connectivity"
        
        if psql -h localhost -U postgres -d vrooli -c "SELECT 1;" > /dev/null 2>&1; then
            log_success "Database connectivity test passed"
        else
            log_warning "Database connectivity test failed"
            # Non-failing for basic test
        fi
    fi
    
    log_success "Basic functionality tests completed"
    return 0
}

#####################################
# INTEGRATION TESTS
#####################################

test_integration() {
    log_info "Testing resource integration"
    
    # Example: Test that resources work together
    # Replace with your actual integration logic
    
    log_info "Testing basic integration workflow"
    
    # Simple workflow: AI processes input and result is handled
    local test_input="test integration"
    local ai_response=$(curl -s -X POST http://localhost:11434/api/generate \
        -d "{\"model\": \"llama3.1:8b\", \"prompt\": \"Echo: $test_input\", \"stream\": false}" \
        -H "Content-Type: application/json")
    
    if echo "$ai_response" | grep -q "$test_input"; then
        log_success "Basic integration test passed"
    else
        log_error "Basic integration test failed"
        return 1
    fi
    
    log_success "Integration tests completed"
    return 0
}

#####################################
# PERFORMANCE TESTS
#####################################

test_basic_performance() {
    log_info "Testing basic performance"
    
    # Example: Simple response time test
    local start_time=$(date +%s)
    
    curl -s -X POST http://localhost:11434/api/generate \
        -d '{"model": "llama3.1:8b", "prompt": "Quick test", "stream": false}' \
        -H "Content-Type: application/json" > /dev/null
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $duration -le 10 ]; then
        log_success "Performance test passed (${duration}s response time)"
    else
        log_warning "Performance test slow (${duration}s response time)"
        # Non-failing for basic test
    fi
    
    return 0
}

#####################################
# MAIN TEST EXECUTION
#####################################

main() {
    log_info "=== Basic Integration Test ==="
    log_info "Scenario ID: $SCENARIO_ID"
    log_info "Test timeout: $TEST_TIMEOUT seconds"
    
    # Validate resources (critical)
    validate_resources || {
        log_error "Resource validation failed"
        exit 1
    }
    
    # Test basic functionality (critical)
    test_basic_functionality || {
        log_error "Basic functionality test failed"
        exit 1
    }
    
    # Test integration (critical)
    test_integration || {
        log_error "Integration test failed" 
        exit 1
    }
    
    # Test performance (non-critical)
    test_basic_performance || {
        log_warning "Performance test failed"
        # Non-failing for basic test
    }
    
    log_success "=== All Basic Tests Passed ==="
    return 0
}

# Execute main test function with timeout
timeout $TEST_TIMEOUT main

exit_code=$?
if [ $exit_code -eq 124 ]; then
    log_error "Test timed out after $TEST_TIMEOUT seconds"
    exit 1
elif [ $exit_code -ne 0 ]; then
    log_error "Test failed with exit code $exit_code"
    exit $exit_code
fi

log_success "Basic integration test completed successfully"