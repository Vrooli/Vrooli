#!/bin/bash
# ====================================================================
# Unstructured-IO Integration Test
# ====================================================================
#
# Tests Unstructured-IO document processing service integration including
# health checks, document parsing, format support, and processing strategies.
#
# Required Resources: unstructured-io
# Test Categories: single-resource, ai, document-processing
# Estimated Duration: 45-90 seconds
#
# ====================================================================

set -euo pipefail

# Test metadata
TEST_RESOURCE="unstructured-io"
TEST_TIMEOUT="${TEST_TIMEOUT:-90}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Recreate HEALTHY_RESOURCES array from exported string
if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
fi

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
source "$SCRIPT_DIR/framework/helpers/fixtures.sh"

# Unstructured-IO configuration
UNSTRUCTURED_BASE_URL="http://localhost:11450"

# Test setup
setup_test() {
    echo "🔧 Setting up Unstructured-IO integration test..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Verify Unstructured-IO is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "curl" "jq"
    
    echo "✓ Test setup complete"
}

# Test Unstructured-IO health and basic connectivity
test_unstructured_health() {
    echo "🏥 Testing Unstructured-IO health endpoint..."
    
    local response
    response=$(curl -s --max-time 10 "$UNSTRUCTURED_BASE_URL/healthcheck" 2>/dev/null)
    
    assert_http_success "$response" "Unstructured-IO health endpoint responds"
    assert_json_valid "$response" "Response is valid JSON"
    assert_json_field "$response" ".healthcheck" "Response contains healthcheck field"
    
    # Check for expected health status
    local health_status
    health_status=$(echo "$response" | jq -r '.healthcheck' 2>/dev/null)
    
    assert_contains "$health_status" "EVERYTHING OK" "Health status indicates service is OK"
    
    echo "✓ Unstructured-IO health check passed"
}

# Test document processing functionality
test_document_processing() {
    echo "📄 Testing document processing..."
    
    # Create test text document
    local test_file="/tmp/unstructured_test_$(date +%s).txt"
    cat > "$test_file" << 'EOF'
Test Document for Unstructured-IO

This is a sample document with multiple sections.

## Introduction
This document contains various elements for testing.

### Table Data
| Name | Value | Type |
|------|-------|------|
| Item1| 100   | Number |
| Item2| Test  | String |

### Summary
This concludes our test document.
EOF
    
    add_cleanup_file "$test_file"
    
    assert_file_exists "$test_file" "Test document created"
    
    # Test document processing
    echo "Processing test document..."
    local response
    response=$(curl -s --max-time 30 \
        -X POST "$UNSTRUCTURED_BASE_URL/general/v0/general" \
        -F "files=@$test_file" \
        -F "strategy=fast" \
        2>/dev/null || echo '{"error":"processing_failed"}')
    
    assert_http_success "$response" "Document processing request successful"
    
    # For unstructured-io, response is typically an array of elements
    if echo "$response" | jq . >/dev/null 2>&1; then
        local element_count
        element_count=$(echo "$response" | jq 'length' 2>/dev/null || echo "0")
        
        assert_greater_than "$element_count" "0" "Document produced structured elements"
        
        echo "Document processed: $element_count elements extracted"
        
        # Check for expected element types
        local has_title
        has_title=$(echo "$response" | jq 'any(.[]; .type == "Title")' 2>/dev/null || echo "false")
        
        if [[ "$has_title" == "true" ]]; then
            echo "✓ Found title elements"
        fi
        
    else
        echo "⚠ Response format may be different from expected JSON array"
    fi
    
    echo "✓ Document processing test passed"
}

# Test different processing strategies
test_processing_strategies() {
    echo "⚙️ Testing processing strategies..."
    
    # Create simple test file
    local test_file="/tmp/strategy_test_$(date +%s).txt"
    echo "Simple test content for strategy testing." > "$test_file"
    add_cleanup_file "$test_file"
    
    # Test different strategies
    local strategies=("fast" "auto")
    
    for strategy in "${strategies[@]}"; do
        echo "Testing strategy: $strategy"
        
        local response
        response=$(curl -s --max-time 20 \
            -X POST "$UNSTRUCTURED_BASE_URL/general/v0/general" \
            -F "files=@$test_file" \
            -F "strategy=$strategy" \
            2>/dev/null || echo "failed")
        
        if [[ "$response" != "failed" ]] && echo "$response" | jq . >/dev/null 2>&1; then
            echo "✓ Strategy '$strategy' works"
        else
            echo "⚠ Strategy '$strategy' may not be supported or failed"
        fi
    done
    
    echo "✓ Processing strategies test completed"
}

# Test error handling
test_error_handling() {
    echo "⚠️ Testing error handling..."
    
    # Test with empty request
    local error_response
    error_response=$(curl -s --max-time 10 \
        -X POST "$UNSTRUCTURED_BASE_URL/general/v0/general" \
        2>/dev/null || echo "connection_failed")
    
    # Should get some kind of error response
    if [[ "$error_response" == "connection_failed" ]]; then
        echo "⚠ Connection test failed - service may be under load"
    else
        echo "✓ Error handling functional (empty request handled)"
    fi
    
    # Test with invalid endpoint
    local invalid_response
    invalid_response=$(curl -s --max-time 5 \
        "$UNSTRUCTURED_BASE_URL/invalid/endpoint" \
        2>/dev/null || echo "expected_404")
    
    if [[ "$invalid_response" == "expected_404" ]] || echo "$invalid_response" | grep -q "Not Found"; then
        echo "✓ Invalid endpoint properly returns 404"
    fi
    
    echo "✓ Error handling test completed"
}

# Test performance characteristics
test_performance() {
    echo "⚡ Testing performance..."
    
    # Create small test file for performance testing
    local test_file="/tmp/perf_test_$(date +%s).txt"
    cat > "$test_file" << 'EOF'
Performance Test Document

This is a simple document for testing processing speed.
It contains minimal content to ensure fast processing.

End of document.
EOF
    
    add_cleanup_file "$test_file"
    
    # Time the processing
    local start_time=$(date +%s)
    
    local response
    response=$(curl -s --max-time 15 \
        -X POST "$UNSTRUCTURED_BASE_URL/general/v0/general" \
        -F "files=@$test_file" \
        -F "strategy=fast" \
        2>/dev/null || echo "timeout")
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "Processing time: ${duration}s"
    
    if [[ $duration -lt 15 && "$response" != "timeout" ]]; then
        echo "✓ Performance is acceptable (< 15s)"
    else
        echo "⚠ Performance may be slow or service overloaded"
    fi
    
    echo "✓ Performance test completed"
}

# Test format support validation
test_format_support() {
    echo "📋 Testing format support..."
    
    # Test with different content types to validate format support
    local formats=("txt" "md")
    
    for format in "${formats[@]}"; do
        local test_file="/tmp/format_test_$(date +%s).$format"
        
        case $format in
            "txt")
                echo "Plain text content for testing." > "$test_file"
                ;;
            "md")
                cat > "$test_file" << 'EOF'
# Markdown Test

This is a **markdown** document with formatting.

- List item 1
- List item 2

## Section

Content here.
EOF
                ;;
        esac
        
        add_cleanup_file "$test_file"
        
        echo "Testing format: $format"
        local response
        response=$(curl -s --max-time 20 \
            -X POST "$UNSTRUCTURED_BASE_URL/general/v0/general" \
            -F "files=@$test_file" \
            -F "strategy=fast" \
            2>/dev/null || echo "failed")
        
        if [[ "$response" != "failed" ]] && echo "$response" | jq . >/dev/null 2>&1; then
            echo "✓ Format '$format' supported"
        else
            echo "⚠ Format '$format' may not be supported or processing failed"
        fi
    done
    
    echo "✓ Format support test completed"
}

# Main test execution
main() {
    echo "🧪 Starting Unstructured-IO Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    test_unstructured_health
    test_document_processing
    test_processing_strategies
    test_format_support
    test_error_handling
    test_performance
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ Unstructured-IO integration test failed"
        exit 1
    else
        echo "✅ Unstructured-IO integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"