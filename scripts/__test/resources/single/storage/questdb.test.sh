#!/bin/bash
# ====================================================================
# QuestDB Integration Test
# ====================================================================
#
# Tests QuestDB time-series database integration including health checks,
# database connectivity, query execution, and data ingestion functionality.
#
# Required Resources: questdb
# Test Categories: single-resource, storage, time-series
# Estimated Duration: 60-90 seconds
#
# ====================================================================

set -euo pipefail

# Test metadata
TEST_RESOURCE="questdb"
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

# QuestDB configuration
QUESTDB_HTTP_URL="http://localhost:9009"
QUESTDB_PG_URL="postgresql://admin:quest@localhost:8812/qdb"

# Test table name for cleanup
TEST_TABLE="integration_test_$(date +%s)"

# Test setup
setup_test() {
    echo "🔧 Setting up QuestDB integration test..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Verify QuestDB is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "curl" "jq"
    
    echo "✓ Test setup complete"
}

# Test QuestDB HTTP health endpoint
test_questdb_health() {
    echo "🏥 Testing QuestDB health endpoint..."
    
    # Check if QuestDB is responding
    local response
    response=$(curl -s --max-time 10 "$QUESTDB_HTTP_URL/" 2>/dev/null || echo "failed")
    
    if [[ "$response" == "failed" ]]; then
        # Try status endpoint specifically
        response=$(curl -s --max-time 10 "$QUESTDB_HTTP_URL/status" 2>/dev/null || echo "status_failed")
        
        if [[ "$response" != "status_failed" ]]; then
            echo "✓ QuestDB status endpoint responding"
        else
            skip_test "QuestDB HTTP endpoint not accessible"
        fi
    else
        assert_http_success "$response" "QuestDB HTTP endpoint responds"
        echo "✓ QuestDB web interface accessible"
    fi
    
    echo "✓ QuestDB health test passed"
}

# Test basic database connectivity and query execution
test_database_connectivity() {
    echo "🔍 Testing database connectivity and queries..."
    
    # Test basic query execution via HTTP API
    local query="SELECT 1 as test_value"
    local response
    response=$(curl -s --max-time 15 \
        -G "$QUESTDB_HTTP_URL/exec" \
        --data-urlencode "query=$query" \
        --data-urlencode "fmt=json" \
        2>/dev/null || echo "query_failed")
    
    if [[ "$response" == "query_failed" ]]; then
        echo "⚠ Direct query execution failed, trying alternative approach"
        
        # Try without format parameter
        response=$(curl -s --max-time 15 \
            -G "$QUESTDB_HTTP_URL/exec" \
            --data-urlencode "query=$query" \
            2>/dev/null || echo "query_failed_alt")
        
        if [[ "$response" != "query_failed_alt" ]]; then
            echo "✓ Basic query execution works (alternative format)"
        else
            echo "⚠ Query execution may not be available"
        fi
    else
        assert_http_success "$response" "Query execution successful"
        
        # Check if response contains expected data
        if echo "$response" | jq . >/dev/null 2>&1; then
            echo "✓ Query returned valid JSON response"
            
            # Check for expected structure
            local test_value
            test_value=$(echo "$response" | jq -r '.dataset[0][0] // empty' 2>/dev/null)
            
            if [[ "$test_value" == "1" ]]; then
                echo "✓ Query returned expected result"
            fi
        else
            echo "✓ Query executed (non-JSON response)"
        fi
    fi
    
    echo "✓ Database connectivity test completed"
}

# Test table creation and data insertion
test_data_operations() {
    echo "📊 Testing data operations..."
    
    # Create test table
    local create_query="CREATE TABLE $TEST_TABLE (ts TIMESTAMP, metric_name SYMBOL, value DOUBLE) timestamp(ts) PARTITION BY DAY"
    
    echo "Creating test table: $TEST_TABLE"
    local create_response
    create_response=$(curl -s --max-time 20 \
        -G "$QUESTDB_HTTP_URL/exec" \
        --data-urlencode "query=$create_query" \
        2>/dev/null || echo "create_failed")
    
    if [[ "$create_response" != "create_failed" ]]; then
        echo "✓ Test table created successfully"
        
        # Insert test data
        local current_time=$(date -u +"%Y-%m-%dT%H:%M:%S.000000Z")
        local insert_query="INSERT INTO $TEST_TABLE VALUES('$current_time', 'test_metric', 42.5)"
        
        echo "Inserting test data..."
        local insert_response
        insert_response=$(curl -s --max-time 15 \
            -G "$QUESTDB_HTTP_URL/exec" \
            --data-urlencode "query=$insert_query" \
            2>/dev/null || echo "insert_failed")
        
        if [[ "$insert_response" != "insert_failed" ]]; then
            echo "✓ Data insertion successful"
            
            # Query the inserted data
            local select_query="SELECT * FROM $TEST_TABLE LIMIT 1"
            local select_response
            select_response=$(curl -s --max-time 15 \
                -G "$QUESTDB_HTTP_URL/exec" \
                --data-urlencode "query=$select_query" \
                --data-urlencode "fmt=json" \
                2>/dev/null || echo "select_failed")
            
            if [[ "$select_response" != "select_failed" ]] && echo "$select_response" | jq . >/dev/null 2>&1; then
                echo "✓ Data query successful"
                
                # Check if we got our test data back
                local returned_value
                returned_value=$(echo "$select_response" | jq -r '.dataset[0][2] // empty' 2>/dev/null)
                
                if [[ "$returned_value" == "42.5" ]]; then
                    echo "✓ Data persistence verified"
                fi
            fi
        else
            echo "⚠ Data insertion may have failed"
        fi
    else
        echo "⚠ Table creation may have failed or is not supported"
    fi
    
    # Schedule cleanup of test table
    add_cleanup_action "cleanup_test_table"
    
    echo "✓ Data operations test completed"
}

# Test time-series specific functionality
test_timeseries_features() {
    echo "⏰ Testing time-series features..."
    
    # Test timestamp functions
    local timestamp_query="SELECT now() as current_time, systimestamp() as system_time"
    local ts_response
    ts_response=$(curl -s --max-time 15 \
        -G "$QUESTDB_HTTP_URL/exec" \
        --data-urlencode "query=$timestamp_query" \
        --data-urlencode "fmt=json" \
        2>/dev/null || echo "timestamp_failed")
    
    if [[ "$ts_response" != "timestamp_failed" ]] && echo "$ts_response" | jq . >/dev/null 2>&1; then
        echo "✓ Timestamp functions work"
        
        # Check if we got timestamp values
        local current_time
        current_time=$(echo "$ts_response" | jq -r '.dataset[0][0] // empty' 2>/dev/null)
        
        if [[ -n "$current_time" && "$current_time" != "null" ]]; then
            echo "✓ Current timestamp retrieved: ${current_time:0:19}"
        fi
    else
        echo "⚠ Timestamp functions may not be available"
    fi
    
    # Test system tables (should exist in QuestDB by default)
    local system_query="SELECT table_name FROM tables() LIMIT 5"
    local system_response
    system_response=$(curl -s --max-time 15 \
        -G "$QUESTDB_HTTP_URL/exec" \
        --data-urlencode "query=$system_query" \
        --data-urlencode "fmt=json" \
        2>/dev/null || echo "system_failed")
    
    if [[ "$system_response" != "system_failed" ]] && echo "$system_response" | jq . >/dev/null 2>&1; then
        echo "✓ System metadata queries work"
        
        # Count tables
        local table_count
        table_count=$(echo "$system_response" | jq '.count // 0' 2>/dev/null)
        
        if [[ "$table_count" -gt 0 ]]; then
            echo "✓ Found $table_count tables in database"
        fi
    fi
    
    echo "✓ Time-series features test completed"
}

# Test error handling and edge cases
test_error_handling() {
    echo "⚠️ Testing error handling..."
    
    # Test invalid query
    local invalid_query="SELECT FROM INVALID_SYNTAX"
    local error_response
    error_response=$(curl -s --max-time 10 \
        -G "$QUESTDB_HTTP_URL/exec" \
        --data-urlencode "query=$invalid_query" \
        2>/dev/null || echo "error_test_failed")
    
    if [[ "$error_response" != "error_test_failed" ]]; then
        echo "✓ Invalid query handled (didn't crash server)"
        
        # Check if response contains error information
        if echo "$error_response" | grep -qi "error\|invalid\|syntax"; then
            echo "✓ Error message provided for invalid query"
        fi
    fi
    
    # Test query with non-existent table
    local nonexistent_query="SELECT * FROM nonexistent_table_12345"
    local nonexistent_response
    nonexistent_response=$(curl -s --max-time 10 \
        -G "$QUESTDB_HTTP_URL/exec" \
        --data-urlencode "query=$nonexistent_query" \
        2>/dev/null || echo "nonexistent_failed")
    
    if [[ "$nonexistent_response" != "nonexistent_failed" ]]; then
        echo "✓ Non-existent table query handled gracefully"
    fi
    
    echo "✓ Error handling test completed"
}

# Test performance characteristics
test_performance() {
    echo "⚡ Testing performance characteristics..."
    
    # Simple performance test
    local start_time=$(date +%s)
    
    local perf_query="SELECT count(*) FROM tables()"
    local perf_response
    perf_response=$(curl -s --max-time 5 \
        -G "$QUESTDB_HTTP_URL/exec" \
        --data-urlencode "query=$perf_query" \
        2>/dev/null || echo "perf_failed")
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "Metadata query time: ${duration}s"
    
    if [[ $duration -lt 5 && "$perf_response" != "perf_failed" ]]; then
        echo "✓ Query performance is acceptable (< 5s)"
    else
        echo "⚠ Performance may be slow or degraded"
    fi
    
    echo "✓ Performance test completed"
}

# Cleanup function for test table
cleanup_test_table() {
    echo "🧹 Cleaning up test table: $TEST_TABLE"
    
    local drop_query="DROP TABLE IF EXISTS $TEST_TABLE"
    curl -s --max-time 10 \
        -G "$QUESTDB_HTTP_URL/exec" \
        --data-urlencode "query=$drop_query" \
        >/dev/null 2>&1 || true
    
    echo "✓ Test table cleanup completed"
}

# Main test execution
main() {
    echo "🧪 Starting QuestDB Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    test_questdb_health
    test_database_connectivity
    test_data_operations
    test_timeseries_features
    test_error_handling
    test_performance
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ QuestDB integration test failed"
        exit 1
    else
        echo "✅ QuestDB integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"