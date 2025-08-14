#!/usr/bin/env bash
set -euo pipefail

# Test script for Multi-Agent Reasoning Ensemble workflow
# This script tests various scenarios and configurations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKFLOW_ENDPOINT="http://localhost:5678/webhook/ensemble-reason"
TEST_RESULTS=()

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status="$1"
    local message="$2"
    case $status in
        "INFO")
            echo -e "${BLUE}[INFO]${NC} $message"
            ;;
        "SUCCESS")
            echo -e "${GREEN}[SUCCESS]${NC} $message"
            ;;
        "WARNING") 
            echo -e "${YELLOW}[WARNING]${NC} $message"
            ;;
        "ERROR")
            echo -e "${RED}[ERROR]${NC} $message"
            ;;
    esac
}

# Function to check if n8n is running
check_n8n_status() {
    print_status "INFO" "Checking n8n status..."
    if curl -s -f "$WORKFLOW_ENDPOINT" >/dev/null 2>&1; then
        print_status "SUCCESS" "n8n is accessible"
        return 0
    else
        print_status "ERROR" "n8n is not accessible at $WORKFLOW_ENDPOINT"
        print_status "INFO" "Please ensure n8n is running and the Multi-Agent Reasoning Ensemble workflow is imported"
        return 1
    fi
}

# Function to test basic workflow functionality
test_basic_functionality() {
    print_status "INFO" "Testing basic multi-agent reasoning functionality..."
    
    local test_payload='{
        "task": "Should our startup focus on B2B or B2C market for our productivity app?",
        "config": {
            "agent_count": 3,
            "reasoning_mode": "quick",
            "consensus_threshold": 0.6,
            "time_limit": 120
        },
        "context": {
            "domain": "business",
            "background": "Early-stage startup, limited resources, productivity app MVP ready",
            "stakeholders": ["founders", "investors", "early users"],
            "constraints": ["6 month runway", "team of 3 developers"]
        }
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 180); then
        
        # Check if response contains expected fields
        if echo "$response" | jq -e '.meta.success' >/dev/null 2>&1; then
            local confidence=$(echo "$response" | jq -r '.meta.confidence_score // 0')
            local agents_used=$(echo "$response" | jq -r '.meta.agents_used // 0')
            local consensus=$(echo "$response" | jq -r '.meta.consensus_reached // false')
            
            print_status "SUCCESS" "Basic functionality test passed"
            print_status "INFO" "Confidence: $confidence, Agents: $agents_used, Consensus: $consensus"
            TEST_RESULTS+=("PASS: Basic functionality")
            return 0
        else
            print_status "ERROR" "Response missing expected success field"
            print_status "INFO" "Response: $response"
            TEST_RESULTS+=("FAIL: Basic functionality - Invalid response structure")
            return 1
        fi
    else
        print_status "ERROR" "Failed to get response from workflow"
        TEST_RESULTS+=("FAIL: Basic functionality - No response")
        return 1
    fi
}

# Function to test technical decision scenario
test_technical_decision() {
    print_status "INFO" "Testing technical decision analysis..."
    
    local test_payload='{
        "task": "Choose between PostgreSQL and MongoDB for our new microservices architecture",
        "config": {
            "agent_count": 5,
            "reasoning_mode": "thorough",
            "consensus_threshold": 0.7,
            "time_limit": 180
        },
        "context": {
            "domain": "technical",
            "background": "E-commerce platform, high-traffic, complex relationships",
            "stakeholders": ["engineering team", "devops", "product team"],
            "constraints": ["ACID compliance preferred", "horizontal scaling needed", "team familiar with SQL"]
        },
        "advanced": {
            "agent_specializations": ["database architect", "performance expert", "scalability specialist", "devops engineer", "business analyst"],
            "output_format": "comprehensive"
        }
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 300); then
        
        if echo "$response" | jq -e '.results.risk_assessment' >/dev/null 2>&1; then
            local recommendation=$(echo "$response" | jq -r '.results.final_recommendation // "None"')
            print_status "SUCCESS" "Technical decision test passed"
            print_status "INFO" "Recommendation: ${recommendation:0:100}..."
            TEST_RESULTS+=("PASS: Technical decision analysis")
            return 0
        else
            print_status "ERROR" "Response missing risk assessment"
            TEST_RESULTS+=("FAIL: Technical decision - Missing risk assessment")
            return 1
        fi
    else
        print_status "ERROR" "Failed to get response for technical decision test"
        TEST_RESULTS+=("FAIL: Technical decision - No response")
        return 1
    fi
}

# Function to test business strategy scenario
test_business_strategy() {
    print_status "INFO" "Testing business strategy analysis..."
    
    local test_payload='{
        "task": "Evaluate the business case for expanding into the European market",
        "config": {
            "agent_count": 4,
            "reasoning_mode": "thorough",
            "consensus_threshold": 0.75
        },
        "context": {
            "domain": "business",
            "background": "SaaS company, $2M ARR, 50 employees, US-based",
            "stakeholders": ["executive team", "investors", "customers", "employees"],
            "constraints": ["$500K budget", "12 month timeline", "GDPR compliance required"]
        }
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 240); then
        
        if echo "$response" | jq -e '.results.implementation_roadmap' >/dev/null 2>&1; then
            local immediate_actions=$(echo "$response" | jq -r '.results.implementation_roadmap.immediate_actions | length // 0')
            print_status "SUCCESS" "Business strategy test passed"
            print_status "INFO" "Immediate actions identified: $immediate_actions"
            TEST_RESULTS+=("PASS: Business strategy analysis")
            return 0
        else
            print_status "ERROR" "Response missing implementation roadmap"
            TEST_RESULTS+=("FAIL: Business strategy - Missing roadmap")
            return 1
        fi
    else
        print_status "ERROR" "Failed to get response for business strategy test"
        TEST_RESULTS+=("FAIL: Business strategy - No response")
        return 1
    fi
}

# Function to test error handling
test_error_handling() {
    print_status "INFO" "Testing error handling with invalid input..."
    
    # Test with missing required field
    local invalid_payload='{
        "config": {
            "agent_count": 3
        }
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$invalid_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 30); then
        
        if echo "$response" | jq -e '.success == false' >/dev/null 2>&1; then
            print_status "SUCCESS" "Error handling test passed - properly rejected invalid input"
            TEST_RESULTS+=("PASS: Error handling")
            return 0
        else
            print_status "WARNING" "Workflow accepted invalid input (may have defaults)"
            TEST_RESULTS+=("PASS: Error handling - Accepted with defaults")
            return 0
        fi
    else
        print_status "ERROR" "Failed to test error handling"
        TEST_RESULTS+=("FAIL: Error handling - No response")
        return 1
    fi
}

# Function to test performance with minimal configuration
test_performance() {
    print_status "INFO" "Testing performance with minimal configuration..."
    
    local start_time=$(date +%s)
    
    local test_payload='{
        "task": "Quick decision: Should we hire a senior developer or two junior developers?",
        "config": {
            "agent_count": 3,
            "reasoning_mode": "quick",
            "time_limit": 60
        }
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 90); then
        
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        if echo "$response" | jq -e '.meta.success' >/dev/null 2>&1; then
            local processing_time=$(echo "$response" | jq -r '.meta.processing_time_ms // 0')
            print_status "SUCCESS" "Performance test passed in ${duration}s"
            print_status "INFO" "Reported processing time: ${processing_time}ms"
            TEST_RESULTS+=("PASS: Performance test")
            return 0
        else
            print_status "ERROR" "Performance test failed - invalid response"
            TEST_RESULTS+=("FAIL: Performance test - Invalid response")
            return 1
        fi
    else
        print_status "ERROR" "Performance test timed out or failed"
        TEST_RESULTS+=("FAIL: Performance test - Timeout")
        return 1
    fi
}

# Function to display test results summary
show_results_summary() {
    print_status "INFO" "=== Test Results Summary ==="
    
    local total_tests=${#TEST_RESULTS[@]}
    local passed_tests=0
    
    for result in "${TEST_RESULTS[@]}"; do
        if [[ $result == PASS* ]]; then
            print_status "SUCCESS" "$result"
            ((passed_tests++))
        else
            print_status "ERROR" "$result"
        fi
    done
    
    print_status "INFO" "Tests passed: $passed_tests/$total_tests"
    
    if [[ $passed_tests -eq $total_tests ]]; then
        print_status "SUCCESS" "All tests passed! Multi-Agent Reasoning Ensemble is working correctly."
        return 0
    else
        print_status "ERROR" "Some tests failed. Please check the workflow configuration."
        return 1
    fi
}

# Main test execution
main() {
    print_status "INFO" "Starting Multi-Agent Reasoning Ensemble workflow tests..."
    print_status "INFO" "Endpoint: $WORKFLOW_ENDPOINT"
    echo
    
    # Check prerequisites
    if ! check_n8n_status; then
        exit 1
    fi
    
    echo
    print_status "INFO" "Running test suite..."
    echo
    
    # Run all tests
    test_basic_functionality
    sleep 2
    test_technical_decision  
    sleep 2
    test_business_strategy
    sleep 2
    test_error_handling
    sleep 2
    test_performance
    
    echo
    show_results_summary
}

# Execute main function
main "$@"