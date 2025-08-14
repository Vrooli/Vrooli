#!/bin/bash
#
# Test script for Chain of Thought Orchestrator n8n workflow
# Tests the step-by-step reasoning with validation capabilities
#

set -euo pipefail

# Configuration
N8N_URL="http://localhost:5678"
ENDPOINT="/webhook/agent/chain-of-thought"
TIMEOUT=600  # 10 minutes timeout for comprehensive reasoning

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if n8n is running
check_n8n_health() {
    log_info "Checking n8n health..."
    if ! curl -s -f "${N8N_URL}/healthz" > /dev/null 2>&1; then
        log_error "n8n is not running at ${N8N_URL}"
        log_info "Please start n8n first: docker-compose up n8n"
        exit 1
    fi
    log_success "n8n is running"
}

# Test function for chain of thought processing
test_chain_of_thought() {
    local test_name="$1"
    local test_data="$2"
    local expected_stages="$3"
    
    log_info "Testing: $test_name"
    
    # Make request to Chain of Thought Orchestrator
    local response
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        "${N8N_URL}${ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d "$test_data" \
        --max-time $TIMEOUT)
    
    local http_code
    http_code=$(echo "$response" | tail -n1)
    local response_body
    response_body=$(echo "$response" | head -n -1)
    
    if [[ "$http_code" == "200" ]]; then
        log_success "Request succeeded (HTTP $http_code)"
        
        # Validate response structure
        local problem=$(echo "$response_body" | jq -r '.problem // empty')
        local success=$(echo "$response_body" | jq -r '.success // false')
        local completed_stages=$(echo "$response_body" | jq -r '.processing_metadata.completed_stages // 0')
        local total_stages=$(echo "$response_body" | jq -r '.processing_metadata.total_stages // 0')
        local quality_score=$(echo "$response_body" | jq -r '.quality_metrics.overall_quality_score // 0')
        local total_time=$(echo "$response_body" | jq -r '.quality_metrics.total_processing_time_ms // 0')
        
        if [[ "$success" == "true" ]]; then
            log_success "Chain of Thought processing completed successfully"
            log_info "Problem: ${problem:0:100}..."
            log_info "Stages completed: $completed_stages/$total_stages"
            log_info "Quality score: $quality_score"
            log_info "Processing time: ${total_time}ms"
            
            # Check if expected number of stages were completed
            if [[ "$completed_stages" -ge "$expected_stages" ]]; then
                log_success "Expected stages completed ($completed_stages >= $expected_stages)"
            else
                log_warning "Fewer stages completed than expected ($completed_stages < $expected_stages)"
            fi
            
            # Validate reasoning trace
            local reasoning_trace_length=$(echo "$response_body" | jq '.reasoning_trace | length')
            if [[ "$reasoning_trace_length" -gt 0 ]]; then
                log_success "Reasoning trace captured ($reasoning_trace_length entries)"
            else
                log_warning "No reasoning trace found"
            fi
            
            # Validate final synthesis
            local has_synthesis=$(echo "$response_body" | jq -r '.final_synthesis.executive_summary // empty' | wc -w)
            if [[ "$has_synthesis" -gt 10 ]]; then
                log_success "Final synthesis generated (${has_synthesis} words)"
            else
                log_warning "Final synthesis seems incomplete"
            fi
            
            # Check validation history
            local validation_count=$(echo "$response_body" | jq '.validation_history | length')
            log_info "Validation history: $validation_count validations performed"
            
            # Display key findings if available
            local key_findings=$(echo "$response_body" | jq -r '.final_synthesis.key_findings[]? // empty' | head -3)
            if [[ -n "$key_findings" ]]; then
                log_info "Key findings:"
                echo "$key_findings" | while read -r finding; do
                    echo "  - $finding"
                done
            fi
            
        else
            log_error "Chain of Thought processing failed"
            local error_message=$(echo "$response_body" | jq -r '.message // "Unknown error"')
            log_error "Error: $error_message"
            return 1
        fi
        
    else
        log_error "Request failed (HTTP $http_code)"
        log_error "Response: $response_body"
        return 1
    fi
    
    echo ""
}

# Test cases
run_tests() {
    log_info "Starting Chain of Thought Orchestrator tests..."
    echo ""
    
    # Test 1: Basic microservices architecture problem
    test_chain_of_thought "Microservices Architecture Design" '{
        "problem": "Design a microservices architecture for an e-commerce platform that needs to handle user management, product catalog, inventory, order processing, payments, and notifications. The system should support 10,000 concurrent users and be highly available.",
        "thinking_stages": [
            "identify_requirements", 
            "enumerate_services",
            "define_interfaces",
            "consider_tradeoffs", 
            "final_recommendation"
        ],
        "validation_mode": "self_critique",
        "model": "qwen2.5-coder:32b",
        "quality_threshold": 0.7,
        "max_validation_retries": 2,
        "advanced": {
            "enable_self_reflection": true,
            "detailed_trace": true,
            "require_evidence": true
        }
    }' 5
    
    # Test 2: Custom thinking stages with different problem
    test_chain_of_thought "Database Optimization Strategy" '{
        "problem": "A social media application with 1 million users is experiencing slow query performance. Database queries are taking 5-10 seconds for timeline feeds, and the system struggles during peak hours. Design an optimization strategy.",
        "thinking_stages": [
            "analyze_context",
            "identify_requirements", 
            "generate_solution",
            "evaluate_tradeoffs"
        ],
        "validation_mode": "self_critique",
        "model": "qwen2.5-coder:32b",
        "quality_threshold": 0.6,
        "max_validation_retries": 1,
        "temperature": 0.4
    }' 4
    
    # Test 3: Complex problem with high quality threshold
    test_chain_of_thought "AI Ethics Framework Design" '{
        "problem": "Design a comprehensive ethics framework for AI decision-making in healthcare applications, considering patient privacy, algorithmic bias, transparency, accountability, and regulatory compliance.",
        "thinking_stages": [
            "identify_requirements",
            "analyze_context",
            "generate_solution",
            "evaluate_tradeoffs",
            "final_recommendation"
        ],
        "validation_mode": "self_critique",
        "model": "qwen2.5-coder:32b",
        "quality_threshold": 0.8,
        "max_validation_retries": 3,
        "advanced": {
            "enable_self_reflection": true,
            "detailed_trace": true,
            "require_evidence": true,
            "cross_validation": false
        }
    }' 5
    
    # Test 4: Simple problem with minimal stages
    test_chain_of_thought "API Rate Limiting Strategy" '{
        "problem": "Implement a rate limiting strategy for a REST API that serves both free and premium users, with different limits for each tier.",
        "thinking_stages": [
            "identify_requirements",
            "generate_solution",
            "final_recommendation"
        ],
        "validation_mode": "self_critique",
        "model": "qwen2.5-coder:32b",
        "quality_threshold": 0.7,
        "max_validation_retries": 1
    }' 3
    
    log_success "All Chain of Thought Orchestrator tests completed!"
}

# Performance test
run_performance_test() {
    log_info "Running performance test..."
    
    local start_time=$(date +%s%3N)
    
    test_chain_of_thought "Performance Test - Quick Problem" '{
        "problem": "Choose between MongoDB and PostgreSQL for a new web application that will store user profiles, posts, and comments.",
        "thinking_stages": ["analyze_context", "evaluate_tradeoffs", "final_recommendation"],
        "validation_mode": "self_critique",
        "model": "qwen2.5-coder:32b",
        "quality_threshold": 0.6,
        "max_validation_retries": 1,
        "temperature": 0.5
    }' 3
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    log_info "Performance test completed in ${duration}ms"
    
    if [[ $duration -lt 120000 ]]; then  # Less than 2 minutes
        log_success "Performance test passed (< 2 minutes)"
    else
        log_warning "Performance test took longer than expected (${duration}ms)"
    fi
}

# Error handling test
test_error_handling() {
    log_info "Testing error handling..."
    
    # Test with missing required field
    log_info "Testing missing problem field..."
    local response
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        "${N8N_URL}${ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d '{"thinking_stages": ["test"]}' \
        --max-time 30)
    
    local http_code
    http_code=$(echo "$response" | tail -n1)
    
    if [[ "$http_code" == "500" ]] || [[ "$http_code" == "400" ]]; then
        log_success "Error handling working correctly for missing problem field"
    else
        log_warning "Unexpected response for missing problem field (HTTP $http_code)"
    fi
    
    # Test with invalid thinking stages
    log_info "Testing empty thinking stages..."
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        "${N8N_URL}${ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d '{"problem": "test", "thinking_stages": []}' \
        --max-time 30)
    
    http_code=$(echo "$response" | tail -n1)
    
    if [[ "$http_code" == "500" ]] || [[ "$http_code" == "400" ]]; then
        log_success "Error handling working correctly for empty thinking stages"
    else
        log_warning "Unexpected response for empty thinking stages (HTTP $http_code)"
    fi
    
    echo ""
}

# Main execution
main() {
    echo "=================================="
    echo "Chain of Thought Orchestrator Tests"
    echo "=================================="
    echo ""
    
    check_n8n_health
    echo ""
    
    # Run error handling tests first
    test_error_handling
    
    # Run main functionality tests
    run_tests
    
    # Run performance test
    echo ""
    run_performance_test
    
    echo ""
    echo "=================================="
    log_success "All tests completed successfully!"
    echo "=================================="
    
    log_info "The Chain of Thought Orchestrator workflow provides:"
    echo "  ✓ Step-by-step reasoning with validation loops"
    echo "  ✓ Configurable thinking stages"
    echo "  ✓ Quality scoring and validation"
    echo "  ✓ Context preservation across stages"
    echo "  ✓ Loop-back capability on validation failure"
    echo "  ✓ Comprehensive reasoning trace"
    echo "  ✓ Final synthesis of all stages"
    echo "  ✓ Quality metrics and processing metadata"
    echo ""
    echo "Endpoint: POST ${N8N_URL}${ENDPOINT}"
    echo "Use this workflow to significantly improve LLM reasoning quality"
    echo "through structured, validated thinking processes."
}

# Run if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi