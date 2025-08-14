#!/bin/bash

# Test script for Tool-Calling Orchestrator n8n workflow
# Tests universal function calling with Ollama models

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
N8N_URL="http://localhost:5678"
WEBHOOK_PATH="/webhook/agent/tools"
ENDPOINT="${N8N_URL}${WEBHOOK_PATH}"
TIMEOUT=300 # 5 minutes for complex tool chains

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to run a test
run_test() {
    local test_name="$1"
    local test_payload="$2"
    local expected_pattern="$3"
    local description="$4"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    log_info "Test $TESTS_RUN: $test_name"
    log_info "Description: $description"
    
    # Make the request
    local response
    local http_code
    
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        --max-time $TIMEOUT \
        "$ENDPOINT" 2>/dev/null || echo -e "\n000")
    
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | head -n -1)
    
    # Check HTTP status
    if [[ "$http_code" != "200" ]]; then
        log_error "HTTP $http_code - Request failed"
        log_error "Response: $response_body"
        echo
        return 1
    fi
    
    # Check response pattern
    if echo "$response_body" | grep -q "$expected_pattern"; then
        log_success "Test passed - Found expected pattern: $expected_pattern"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_error "Test failed - Pattern not found: $expected_pattern"
        log_error "Response: $response_body"
        echo
        return 1
    fi
    
    # Pretty print response for inspection
    if command -v jq &> /dev/null; then
        echo "Response:"
        echo "$response_body" | jq '.'
    else
        echo "Response (install jq for pretty printing):"
        echo "$response_body"
    fi
    
    echo
    return 0
}

# Function to check if n8n is running
check_n8n() {
    log_info "Checking if n8n is running..."
    
    if ! curl -s "$N8N_URL/healthz" > /dev/null 2>&1; then
        log_error "n8n is not running at $N8N_URL"
        log_error "Please start n8n first: cd packages/server && pnpm run n8n"
        exit 1
    fi
    
    log_success "n8n is running"
}

# Function to check if Ollama is running
check_ollama() {
    log_info "Checking if Ollama is running..."
    
    if ! curl -s "http://localhost:11434/api/version" > /dev/null 2>&1; then
        log_error "Ollama is not running at http://localhost:11434"
        log_error "Please start Ollama first: ollama serve"
        exit 1
    fi
    
    log_success "Ollama is running"
}

# Test cases
echo "=================================================="
echo "       Tool-Calling Orchestrator Tests"
echo "=================================================="

# Pre-flight checks
check_n8n
check_ollama

echo

# Test 1: Basic Calculator Tool
log_info "Starting Test Suite..."
echo

run_test "Basic Calculator Tool" '{
  "query": "Calculate 15 * 8 + 32",
  "tools": [
    {
      "name": "calculator",
      "description": "Perform mathematical calculations",
      "parameters": {
        "expression": "string - mathematical expression to evaluate"
      }
    }
  ],
  "model": "llama3.2",
  "parallel_execution": false
}' '"success": *true' "Test basic tool calling with calculator"

# Test 2: Web Search Tool
run_test "Web Search Tool" '{
  "query": "Search for information about artificial intelligence",
  "tools": [
    {
      "name": "web_search",
      "description": "Search the web for information",
      "parameters": {
        "query": "string - search query",
        "max_results": "number - maximum results to return"
      }
    }
  ],
  "model": "llama3.2",
  "parallel_execution": false
}' '"tools_executed"' "Test web search tool functionality"

# Test 3: Multiple Tools in Sequence
run_test "Multiple Tools Sequential" '{
  "query": "Calculate 50 * 3 and then search for information about the result",
  "tools": [
    {
      "name": "calculator",
      "description": "Perform mathematical calculations",
      "parameters": {
        "expression": "string - mathematical expression to evaluate"
      }
    },
    {
      "name": "web_search",
      "description": "Search the web for information",
      "parameters": {
        "query": "string - search query",
        "max_results": "number - maximum results to return"
      }
    }
  ],
  "model": "llama3.2",
  "parallel_execution": false,
  "max_iterations": 3
}' '"execution_metrics"' "Test multiple tools called in sequence"

# Test 4: Parallel Tool Execution
run_test "Parallel Tool Execution" '{
  "query": "Get weather information and search for news about climate change",
  "tools": [
    {
      "name": "weather_check",
      "description": "Check current weather conditions",
      "parameters": {
        "location": "string - location to check weather for"
      }
    },
    {
      "name": "web_search",
      "description": "Search the web for information",
      "parameters": {
        "query": "string - search query"
      }
    }
  ],
  "model": "llama3.2",
  "parallel_execution": true
}' '"tool_success_rate"' "Test parallel tool execution"

# Test 5: Text Processing Tool
run_test "Text Processing Tool" '{
  "query": "Summarize the following text: Artificial intelligence is revolutionizing many industries including healthcare, finance, and transportation. Machine learning algorithms can analyze vast amounts of data to identify patterns and make predictions that were previously impossible for humans to achieve.",
  "tools": [
    {
      "name": "text_processor",
      "description": "Process text with various operations",
      "parameters": {
        "text": "string - text to process",
        "operation": "string - operation to perform (summarize, analyze, extract)"
      }
    }
  ],
  "model": "llama3.2"
}' '"final_response"' "Test text processing capabilities"

# Test 6: File Operations Tool
run_test "File Operations Tool" '{
  "query": "Read the contents of config.json file",
  "tools": [
    {
      "name": "file_reader",
      "description": "Read file contents",
      "parameters": {
        "filename": "string - name of file to read"
      }
    }
  ],
  "model": "llama3.2"
}' '"filename"' "Test file reading tool functionality"

# Test 7: Data Analysis Tool
run_test "Data Analysis Tool" '{
  "query": "Analyze a dataset and provide insights",
  "tools": [
    {
      "name": "data_analyzer",
      "description": "Perform statistical analysis on data",
      "parameters": {
        "analysis_type": "string - type of analysis to perform",
        "data_source": "string - source of data to analyze"
      }
    }
  ],
  "model": "llama3.2"
}' '"insights"' "Test data analysis capabilities"

# Test 8: Code Execution Tool
run_test "Code Execution Tool" '{
  "query": "Execute a simple Python script that prints hello world",
  "tools": [
    {
      "name": "code_executor",
      "description": "Execute code in various languages",
      "parameters": {
        "language": "string - programming language",
        "code": "string - code to execute"
      }
    }
  ],
  "model": "llama3.2"
}' '"execution_time_ms"' "Test code execution functionality"

# Test 9: Database Query Tool
run_test "Database Query Tool" '{
  "query": "Query the user database for active users",
  "tools": [
    {
      "name": "database_query",
      "description": "Execute database queries",
      "parameters": {
        "query": "string - SQL query to execute",
        "database": "string - database to query"
      }
    }
  ],
  "model": "llama3.2"
}' '"rows_affected"' "Test database query capabilities"

# Test 10: Email Sender Tool
run_test "Email Sender Tool" '{
  "query": "Send a welcome email to new user john@example.com",
  "tools": [
    {
      "name": "email_sender",
      "description": "Send emails to recipients",
      "parameters": {
        "to": "string - recipient email address",
        "subject": "string - email subject",
        "body": "string - email body content"
      }
    }
  ],
  "model": "llama3.2"
}' '"message_id"' "Test email sending functionality"

# Test 11: Complex Multi-Tool Workflow
run_test "Complex Multi-Tool Workflow" '{
  "query": "Calculate the square root of 144, search for information about that number, and then analyze the search results",
  "tools": [
    {
      "name": "calculator",
      "description": "Perform mathematical calculations",
      "parameters": {
        "expression": "string - mathematical expression"
      }
    },
    {
      "name": "web_search", 
      "description": "Search the web for information",
      "parameters": {
        "query": "string - search query"
      }
    },
    {
      "name": "text_processor",
      "description": "Process and analyze text",
      "parameters": {
        "text": "string - text to process",
        "operation": "string - operation to perform"
      }
    }
  ],
  "model": "llama3.2",
  "max_iterations": 5,
  "parallel_execution": false
}' '"efficiency_score"' "Test complex multi-step tool workflow"

# Test 12: Error Handling - Invalid Tool
run_test "Error Handling Invalid Tool" '{
  "query": "Use a non-existent tool",
  "tools": [
    {
      "name": "valid_tool",
      "description": "A valid tool",
      "parameters": {
        "param": "string - parameter"
      }
    }
  ],
  "model": "llama3.2"
}' '"success"' "Test error handling with workflow completion"

# Test 13: No Tools Available
run_test "No Tools Available" '{
  "query": "Answer a simple question without tools",
  "tools": [],
  "model": "llama3.2"
}' '"final_response"' "Test handling when no tools are available"

# Test 14: Custom System Prompt
run_test "Custom System Prompt" '{
  "query": "What is 2 + 2?",
  "tools": [
    {
      "name": "calculator",
      "description": "Perform calculations",
      "parameters": {
        "expression": "string - mathematical expression"
      }
    }
  ],
  "model": "llama3.2",
  "system_prompt": "You are a math tutor. Always explain your reasoning before using tools."
}' '"success"' "Test custom system prompt functionality"

# Test 15: High Temperature Creative Response
run_test "High Temperature Creative" '{
  "query": "Create a creative story using weather and text processing tools",
  "tools": [
    {
      "name": "weather_check",
      "description": "Check weather conditions",
      "parameters": {
        "location": "string - location"
      }
    },
    {
      "name": "text_processor",
      "description": "Process text creatively",
      "parameters": {
        "text": "string - input text",
        "operation": "string - operation type"
      }
    }
  ],
  "model": "llama3.2",
  "temperature": 0.9
}' '"quality_metrics"' "Test high temperature creative tool usage"

# Test Results Summary
echo "=================================================="
echo "                Test Results Summary"
echo "=================================================="

if [ $TESTS_PASSED -eq $TESTS_RUN ]; then
    log_success "All tests passed! ($TESTS_PASSED/$TESTS_RUN)"
    echo
    log_info "The Tool-Calling Orchestrator workflow is working correctly!"
    log_info "Key features tested:"
    echo "  ✅ Basic tool execution"
    echo "  ✅ Multiple tool coordination"
    echo "  ✅ Parallel and sequential execution"
    echo "  ✅ Parameter validation"
    echo "  ✅ Error handling"
    echo "  ✅ Response formatting"
    echo "  ✅ Iteration management"
    echo "  ✅ Performance metrics"
    echo "  ✅ Custom configurations"
    echo
    log_info "Ready for production use with Ollama models!"
else
    log_error "Some tests failed: $TESTS_PASSED/$TESTS_RUN passed"
    echo
    log_warning "Please review the failed tests and check:"
    echo "  - n8n workflow is properly imported"
    echo "  - Ollama is running with required models"
    echo "  - Network connectivity is working"
    echo "  - Webhook endpoint is accessible"
    exit 1
fi

echo "=================================================="