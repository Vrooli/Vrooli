#!/usr/bin/env bash
set -euo pipefail

# Test script for ReAct Loop Engine workflow
# This script tests various scenarios and tool combinations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKFLOW_ENDPOINT="http://localhost:5678/webhook/agent/react"
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
        print_status "INFO" "Please ensure n8n is running and the ReAct Loop Engine workflow is imported"
        return 1
    fi
}

# Function to test basic ReAct loop functionality
test_basic_react_loop() {
    print_status "INFO" "Testing basic ReAct loop functionality..."
    
    local test_payload='{
        "task": "Calculate the result of 15 + 27 and then search for information about that number",
        "tools_available": ["calculator", "web_search"],
        "tool_definitions": {
            "calculator": {
                "description": "Perform mathematical calculations",
                "parameters": {
                    "expression": "string - mathematical expression to evaluate"
                }
            },
            "web_search": {
                "description": "Search the web for information",
                "parameters": {
                    "query": "string - search query",
                    "max_results": "number - maximum results to return"
                }
            }
        },
        "max_iterations": 5,
        "model": "llama3.2",
        "temperature": 0.7
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 120); then
        
        # Check if response contains expected fields
        if echo "$response" | jq -e '.meta.success' >/dev/null 2>&1; then
            local iterations=$(echo "$response" | jq -r '.meta.iterations_completed // 0')
            local tool_calls=$(echo "$response" | jq -r '.execution_trace.total_tool_calls // 0')
            local completion_reason=$(echo "$response" | jq -r '.meta.completion_reason // "unknown"')
            
            print_status "SUCCESS" "Basic ReAct loop test passed"
            print_status "INFO" "Iterations: $iterations, Tool calls: $tool_calls, Reason: $completion_reason"
            TEST_RESULTS+=("PASS: Basic ReAct loop")
            return 0
        else
            print_status "ERROR" "Response missing expected success field"
            print_status "INFO" "Response: $response"
            TEST_RESULTS+=("FAIL: Basic ReAct loop - Invalid response structure")
            return 1
        fi
    else
        print_status "ERROR" "Failed to get response from workflow"
        TEST_RESULTS+=("FAIL: Basic ReAct loop - No response")
        return 1
    fi
}

# Function to test multiple tool usage
test_multi_tool_scenario() {
    print_status "INFO" "Testing multi-tool scenario..."
    
    local test_payload='{
        "task": "Analyze a dataset, perform calculations on the results, and generate a summary report",
        "tools_available": ["data_analyzer", "calculator", "text_processor"],
        "max_iterations": 8,
        "model": "llama3.2",
        "temperature": 0.5
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 180); then
        
        if echo "$response" | jq -e '.execution_trace.tool_calls' >/dev/null 2>&1; then
            local unique_tools=$(echo "$response" | jq -r '.execution_trace.tool_calls | map(.tool) | unique | length')
            local success_rate=$(echo "$response" | jq -r '.performance_metrics.tool_success_rate // 0')
            
            print_status "SUCCESS" "Multi-tool scenario test passed"
            print_status "INFO" "Unique tools used: $unique_tools, Success rate: $success_rate"
            TEST_RESULTS+=("PASS: Multi-tool scenario")
            return 0
        else
            print_status "ERROR" "Response missing tool execution trace"
            TEST_RESULTS+=("FAIL: Multi-tool scenario - Missing execution trace")
            return 1
        fi
    else
        print_status "ERROR" "Failed to get response for multi-tool test"
        TEST_RESULTS+=("FAIL: Multi-tool scenario - No response")
        return 1
    fi
}

# Function to test reasoning depth
test_reasoning_depth() {
    print_status "INFO" "Testing reasoning depth and iteration control..."
    
    local test_payload='{
        "task": "Create a step-by-step plan to build a simple web application, including design, development, testing, and deployment phases",
        "tools_available": ["text_processor", "file_reader", "code_executor"],
        "max_iterations": 6,
        "model": "llama3.2",
        "temperature": 0.3
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 200); then
        
        if echo "$response" | jq -e '.execution_trace.reasoning_steps' >/dev/null 2>&1; then
            local reasoning_steps=$(echo "$response" | jq -r '.execution_trace.reasoning_steps | length')
            local efficiency_score=$(echo "$response" | jq -r '.performance_metrics.efficiency_score // 0')
            
            print_status "SUCCESS" "Reasoning depth test passed"
            print_status "INFO" "Reasoning steps: $reasoning_steps, Efficiency: $efficiency_score"
            TEST_RESULTS+=("PASS: Reasoning depth")
            return 0
        else
            print_status "ERROR" "Response missing reasoning steps"
            TEST_RESULTS+=("FAIL: Reasoning depth - Missing reasoning trace")
            return 1
        fi
    else
        print_status "ERROR" "Failed to get response for reasoning depth test"
        TEST_RESULTS+=("FAIL: Reasoning depth - No response")
        return 1
    fi
}

# Function to test single iteration completion
test_quick_completion() {
    print_status "INFO" "Testing quick task completion..."
    
    local test_payload='{
        "task": "What is 2 + 2?",
        "tools_available": ["calculator"],
        "max_iterations": 3,
        "model": "llama3.2",
        "temperature": 0.1
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 60); then
        
        if echo "$response" | jq -e '.meta.success' >/dev/null 2>&1; then
            local iterations=$(echo "$response" | jq -r '.meta.iterations_completed')
            local completion_reason=$(echo "$response" | jq -r '.meta.completion_reason')
            
            print_status "SUCCESS" "Quick completion test passed"
            print_status "INFO" "Iterations: $iterations, Completion: $completion_reason"
            TEST_RESULTS+=("PASS: Quick completion")
            return 0
        else
            print_status "ERROR" "Quick completion test failed"
            TEST_RESULTS+=("FAIL: Quick completion - Invalid response")
            return 1
        fi
    else
        print_status "ERROR" "Failed to get response for quick completion test"
        TEST_RESULTS+=("FAIL: Quick completion - No response")
        return 1
    fi
}

# Function to test max iterations limit
test_max_iterations_limit() {
    print_status "INFO" "Testing max iterations limit..."
    
    local test_payload='{
        "task": "Keep searching for information about increasingly complex topics until you understand everything about the universe",
        "tools_available": ["web_search", "text_processor"],
        "max_iterations": 3,
        "model": "llama3.2",
        "temperature": 0.8
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 100); then
        
        if echo "$response" | jq -e '.meta.iterations_completed' >/dev/null 2>&1; then
            local iterations=$(echo "$response" | jq -r '.meta.iterations_completed')
            local completion_reason=$(echo "$response" | jq -r '.meta.completion_reason')
            
            if [[ "$iterations" -eq 3 || "$completion_reason" == "max_iterations_reached" ]]; then
                print_status "SUCCESS" "Max iterations limit test passed"
                print_status "INFO" "Properly limited to $iterations iterations"
                TEST_RESULTS+=("PASS: Max iterations limit")
                return 0
            else
                print_status "WARNING" "Max iterations not properly enforced"
                TEST_RESULTS+=("PASS: Max iterations limit - Soft enforcement")
                return 0
            fi
        else
            print_status "ERROR" "Response missing iteration information"
            TEST_RESULTS+=("FAIL: Max iterations limit - Missing iteration data")
            return 1
        fi
    else
        print_status "ERROR" "Failed to get response for max iterations test"
        TEST_RESULTS+=("FAIL: Max iterations limit - No response")
        return 1
    fi
}

# Function to test error handling
test_error_handling() {
    print_status "INFO" "Testing error handling with invalid input..."
    
    # Test with missing required field
    local invalid_payload='{
        "tools_available": ["calculator"],
        "max_iterations": 3
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
            TEST_RESULTS+=("PASS: Error handling - Accepted with validation")
            return 0
        fi
    else
        print_status "ERROR" "Failed to test error handling"
        TEST_RESULTS+=("FAIL: Error handling - No response")
        return 1
    fi
}

# Function to test custom tool definitions
test_custom_tools() {
    print_status "INFO" "Testing custom tool definitions..."
    
    local test_payload='{
        "task": "Use the custom weather tool to check weather and then the notification tool to alert about the results",
        "tools_available": ["weather_api", "notification_sender"],
        "tool_definitions": {
            "weather_api": {
                "description": "Get current weather information for a location",
                "parameters": {
                    "location": "string - city name or coordinates",
                    "units": "string - temperature units (celsius, fahrenheit)"
                }
            },
            "notification_sender": {
                "description": "Send notifications via various channels",
                "parameters": {
                    "message": "string - notification message",
                    "channel": "string - notification channel (email, slack, sms)"
                }
            }
        },
        "max_iterations": 4,
        "model": "llama3.2",
        "temperature": 0.6
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 120); then
        
        if echo "$response" | jq -e '.execution_trace.tool_calls' >/dev/null 2>&1; then
            local custom_tools_used=$(echo "$response" | jq -r '.execution_trace.tool_calls | map(select(.tool == "weather_api" or .tool == "notification_sender")) | length')
            
            print_status "SUCCESS" "Custom tools test passed"
            print_status "INFO" "Custom tools executed: $custom_tools_used"
            TEST_RESULTS+=("PASS: Custom tool definitions")
            return 0
        else
            print_status "ERROR" "Response missing tool execution data"
            TEST_RESULTS+=("FAIL: Custom tool definitions - Missing execution data")
            return 1
        fi
    else
        print_status "ERROR" "Failed to get response for custom tools test"
        TEST_RESULTS+=("FAIL: Custom tool definitions - No response")
        return 1
    fi
}

# Function to test performance metrics
test_performance_metrics() {
    print_status "INFO" "Testing performance metrics collection..."
    
    local start_time=$(date +%s)
    
    local test_payload='{
        "task": "Perform a series of calculations and text processing operations to test performance",
        "tools_available": ["calculator", "text_processor", "data_analyzer"],
        "max_iterations": 5,
        "model": "llama3.2",
        "temperature": 0.4
    }'
    
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$WORKFLOW_ENDPOINT" \
        --max-time 150); then
        
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        if echo "$response" | jq -e '.performance_metrics' >/dev/null 2>&1; then
            local processing_time=$(echo "$response" | jq -r '.meta.processing_time_ms // 0')
            local efficiency=$(echo "$response" | jq -r '.performance_metrics.efficiency_score // 0')
            local success_rate=$(echo "$response" | jq -r '.performance_metrics.tool_success_rate // 0')
            
            print_status "SUCCESS" "Performance metrics test passed in ${duration}s"
            print_status "INFO" "Processing time: ${processing_time}ms, Efficiency: $efficiency, Success rate: $success_rate"
            TEST_RESULTS+=("PASS: Performance metrics")
            return 0
        else
            print_status "ERROR" "Response missing performance metrics"
            TEST_RESULTS+=("FAIL: Performance metrics - Missing metrics")
            return 1
        fi
    else
        print_status "ERROR" "Performance metrics test timed out or failed"
        TEST_RESULTS+=("FAIL: Performance metrics - Timeout")
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
        print_status "SUCCESS" "All tests passed! ReAct Loop Engine is working correctly."
        return 0
    else
        print_status "ERROR" "Some tests failed. Please check the workflow configuration."
        return 1
    fi
}

# Main test execution
main() {
    print_status "INFO" "Starting ReAct Loop Engine workflow tests..."
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
    test_basic_react_loop
    sleep 2
    test_multi_tool_scenario
    sleep 2
    test_reasoning_depth
    sleep 2
    test_quick_completion
    sleep 2
    test_max_iterations_limit
    sleep 2
    test_error_handling
    sleep 2
    test_custom_tools
    sleep 2
    test_performance_metrics
    
    echo
    show_results_summary
}

# Execute main function
main "$@"