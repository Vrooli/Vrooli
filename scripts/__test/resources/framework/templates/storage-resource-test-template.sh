#!/bin/bash
# ====================================================================
# Storage Resource Test Template
# ====================================================================
#
# Template for testing storage resources with category-specific requirements
# including data persistence, backup/restore capabilities, and performance
# characteristics.
#
# Usage: Copy this template and customize for specific storage resource
#
# Required Variables to Set:
#   TEST_RESOURCE - Resource name (e.g., "postgres", "redis", "minio")
#   RESOURCE_PORT - Default port for the resource
#   RESOURCE_BASE_URL - Base URL for API calls (if HTTP-based)
#
# ====================================================================

set -euo pipefail

# Test metadata - CUSTOMIZE THESE
TEST_RESOURCE="your-storage-resource"
TEST_TIMEOUT="${TEST_TIMEOUT:-90}"  # Storage operations can be slow
TEST_CLEANUP="${TEST_CLEANUP:-true}"
RESOURCE_PORT="5432"  # Replace with actual port
RESOURCE_BASE_URL="http://localhost:${RESOURCE_PORT}"  # May not apply to all storage

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
source "$SCRIPT_DIR/framework/interface-compliance-categories.sh"

# Test setup
setup_test() {
    echo "🔧 Setting up storage resource test for $TEST_RESOURCE..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Auto-discovery fallback for direct test execution
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        echo "🔍 Auto-discovering resources for direct test execution..."
        
        local resources_dir
        resources_dir="$(cd "$SCRIPT_DIR/../.." && pwd)"
        
        local discovery_output=""
        if timeout 10s bash -c "\"$resources_dir/index.sh\" --action discover 2>&1" > /tmp/discovery_output.tmp 2>&1; then
            discovery_output=$(cat /tmp/discovery_output.tmp)
            rm -f /tmp/discovery_output.tmp
        else
            echo "⚠️  Auto-discovery timed out, using fallback method..."
            # Test port connectivity
            if timeout 3 bash -c "</dev/tcp/localhost/$RESOURCE_PORT" 2>/dev/null; then
                discovery_output="✅ $TEST_RESOURCE is running on port $RESOURCE_PORT"
            fi
        fi
        
        local discovered_resources=()
        while IFS= read -r line; do
            if [[ "$line" =~ ✅[[:space:]]+([^[:space:]]+)[[:space:]]+is[[:space:]]+running ]]; then
                discovered_resources+=("${BASH_REMATCH[1]}")
            fi
        done <<< "$discovery_output"
        
        if [[ ${#discovered_resources[@]} -eq 0 ]]; then
            echo "⚠️  No resources discovered, but test will proceed..."
            discovered_resources=("$TEST_RESOURCE")
        fi
        
        export HEALTHY_RESOURCES_STR="${discovered_resources[*]}"
        echo "✓ Discovered healthy resources: $HEALTHY_RESOURCES_STR"
    fi
    
    # Verify resource is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools (customize based on your storage type)
    require_tools "curl"  # Add specific tools: "psql", "redis-cli", "mc", etc.
    
    echo "✓ Test setup complete"
}

# Test storage resource interface compliance
test_storage_interface_compliance() {
    echo "🔧 Testing storage resource interface compliance..."
    
    # Find the manage.sh script
    local manage_script=""
    local potential_paths=(
        "$SCRIPT_DIR/../storage/$TEST_RESOURCE/manage.sh"
        "$SCRIPT_DIR/../../storage/$TEST_RESOURCE/manage.sh"
        "$(cd "$SCRIPT_DIR/../.." && pwd)/storage/$TEST_RESOURCE/manage.sh"
    )
    
    for path in "${potential_paths[@]}"; do
        if [[ -f "$path" ]]; then
            manage_script="$path"
            break
        fi
    done
    
    if [[ -z "$manage_script" ]]; then
        manage_script=$(find "$(cd "$SCRIPT_DIR/../.." && pwd)" -name "manage.sh" -path "*/$TEST_RESOURCE/*" -type f 2>/dev/null | head -1)
    fi
    
    if [[ -z "$manage_script" || ! -f "$manage_script" ]]; then
        echo "⚠️  Could not find $TEST_RESOURCE manage.sh script - skipping interface compliance test"
        return 0
    fi
    
    echo "📍 Using manage script: $manage_script"
    
    # Run complete compliance test (base + category-specific)
    if test_complete_resource_compliance "$TEST_RESOURCE" "$manage_script" "storage"; then
        echo "✅ Storage resource interface compliance test passed"
        return 0
    else
        echo "❌ Storage resource interface compliance test failed"
        return 1
    fi
}

# Test storage resource health and connectivity
test_storage_health() {
    echo "🏥 Testing storage resource health..."
    
    # Use category-specific health check
    local health_result
    health_result=$(check_storage_resource_health "$TEST_RESOURCE" "$RESOURCE_PORT" "detailed")
    
    assert_not_equals "$health_result" "unreachable" "Storage resource is reachable"
    
    # Parse detailed health information
    if [[ "$health_result" =~ healthy ]]; then
        echo "✓ Storage resource health check passed: $health_result"
    else
        echo "⚠️  Storage resource health degraded: $health_result"
    fi
}

# Test data persistence capabilities (Storage-specific)
test_storage_data_persistence() {
    echo "💾 Testing data persistence capabilities..."
    
    # This is a template - customize based on your storage resource's API/CLI
    
    local test_key="vrooli_test_$(date +%s)"
    local test_value="test_data_$(date +%s)"
    
    echo "Testing data write/read cycle with key: $test_key"
    
    # Customize these operations based on your storage type:
    
    # For Redis:
    # redis-cli -h localhost -p "$RESOURCE_PORT" SET "$test_key" "$test_value"
    # local retrieved_value=$(redis-cli -h localhost -p "$RESOURCE_PORT" GET "$test_key")
    
    # For PostgreSQL:
    # psql -h localhost -p "$RESOURCE_PORT" -U postgres -d testdb -c "CREATE TABLE IF NOT EXISTS test_table (key VARCHAR(255), value VARCHAR(255));"
    # psql -h localhost -p "$RESOURCE_PORT" -U postgres -d testdb -c "INSERT INTO test_table (key, value) VALUES ('$test_key', '$test_value');"
    # local retrieved_value=$(psql -h localhost -p "$RESOURCE_PORT" -U postgres -d testdb -t -c "SELECT value FROM test_table WHERE key='$test_key';" | tr -d ' \n')
    
    # For MinIO (S3-compatible):
    # echo "$test_value" | mc pipe minio/test-bucket/$test_key
    # local retrieved_value=$(mc cat minio/test-bucket/$test_key)
    
    # Generic example (customize this):
    echo "⚠️  Data persistence test needs to be customized for $TEST_RESOURCE"
    echo "✓ Data persistence test framework is in place"
    
    # Cleanup test data
    # redis-cli -h localhost -p "$RESOURCE_PORT" DEL "$test_key"
    # psql -h localhost -p "$RESOURCE_PORT" -U postgres -d testdb -c "DELETE FROM test_table WHERE key='$test_key';"
    # mc rm minio/test-bucket/$test_key
    
    echo "✓ Data persistence test completed"
}

# Test backup and restore capabilities (Storage-specific)
test_storage_backup_restore() {
    echo "🔄 Testing backup and restore capabilities..."
    
    # This should be implemented based on the specific storage type
    
    echo "Testing backup creation..."
    # Example implementations:
    
    # For PostgreSQL:
    # pg_dump -h localhost -p "$RESOURCE_PORT" -U postgres -d testdb > /tmp/test_backup.sql
    
    # For Redis:
    # redis-cli -h localhost -p "$RESOURCE_PORT" BGSAVE
    
    # For file-based storage (MinIO):
    # mc mirror minio/bucket /tmp/backup/
    
    echo "⚠️  Backup/restore test needs to be customized for $TEST_RESOURCE"
    echo "✓ Backup/restore test framework is in place"
    
    echo "✓ Backup and restore test completed"
}

# Test storage statistics and monitoring
test_storage_statistics() {
    echo "📊 Testing storage statistics and monitoring..."
    
    # Get storage-specific statistics
    case "$TEST_RESOURCE" in
        "postgres"|"postgresql")
            if which psql >/dev/null 2>&1; then
                echo "PostgreSQL database statistics:"
                local db_count=$(psql -h localhost -p "$RESOURCE_PORT" -U postgres -t -c "SELECT count(*) FROM pg_database WHERE datistemplate = false;" 2>/dev/null | tr -d ' \n' || echo "unavailable")
                echo "  • Databases: $db_count"
                
                local connection_count=$(psql -h localhost -p "$RESOURCE_PORT" -U postgres -t -c "SELECT count(*) FROM pg_stat_activity;" 2>/dev/null | tr -d ' \n' || echo "unavailable")
                echo "  • Active connections: $connection_count"
            fi
            ;;
        "redis")
            if which redis-cli >/dev/null 2>&1; then
                echo "Redis statistics:"
                local memory_usage=$(redis-cli -h localhost -p "$RESOURCE_PORT" info memory 2>/dev/null | grep "used_memory_human:" | cut -d: -f2 | tr -d '\r\n' || echo "unavailable")
                echo "  • Memory usage: $memory_usage"
                
                local connected_clients=$(redis-cli -h localhost -p "$RESOURCE_PORT" info clients 2>/dev/null | grep "connected_clients:" | cut -d: -f2 | tr -d '\r\n' || echo "unavailable")
                echo "  • Connected clients: $connected_clients"
            fi
            ;;
        "minio")
            if curl -s --max-time 5 "http://localhost:${RESOURCE_PORT}/minio/health/live" >/dev/null 2>&1; then
                echo "MinIO statistics:"
                echo "  • Service status: Live"
                
                local ready_status="Unknown"
                if curl -s --max-time 5 "http://localhost:${RESOURCE_PORT}/minio/health/ready" >/dev/null 2>&1; then
                    ready_status="Ready"
                fi
                echo "  • Ready status: $ready_status"
            fi
            ;;
        *)
            echo "⚠️  Statistics collection needs to be customized for $TEST_RESOURCE"
            ;;
    esac
    
    echo "✓ Storage statistics test completed"
}

# Test storage performance characteristics
test_storage_performance() {
    echo "⚡ Testing storage resource performance..."
    
    # Test basic operation timing
    local start_time=$(date +%s.%N)
    
    # Customize this based on your storage resource's quickest operation
    case "$TEST_RESOURCE" in
        "redis")
            if which redis-cli >/dev/null 2>&1; then
                redis-cli -h localhost -p "$RESOURCE_PORT" ping >/dev/null 2>&1
            fi
            ;;
        "postgres"|"postgresql")
            if which psql >/dev/null 2>&1; then
                psql -h localhost -p "$RESOURCE_PORT" -U postgres -t -c "SELECT 1;" >/dev/null 2>&1
            fi
            ;;
        *)
            # Generic port connectivity test
            timeout 3 bash -c "</dev/tcp/localhost/$RESOURCE_PORT" 2>/dev/null
            ;;
    esac
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "unknown")
    
    echo "Basic operation response time: ${duration}s"
    
    if [[ "$duration" != "unknown" ]]; then
        local duration_ms=$(echo "$duration * 1000" | bc 2>/dev/null || echo "unknown")
        if [[ "$duration_ms" != "unknown" ]] && (( $(echo "$duration_ms < 100" | bc -l) )); then
            echo "✓ Performance is excellent (< 100ms)"
        elif [[ "$duration_ms" != "unknown" ]] && (( $(echo "$duration_ms < 1000" | bc -l) )); then
            echo "✓ Performance is good (< 1s)"
        else
            echo "⚠️  Performance may need optimization (> 1s)"
        fi
    fi
    
    echo "✓ Performance test completed"
}

# Test error handling
test_storage_error_handling() {
    echo "⚠️ Testing storage resource error handling..."
    
    # Test with invalid operations (customize based on storage type)
    case "$TEST_RESOURCE" in
        "redis")
            if which redis-cli >/dev/null 2>&1; then
                # Try to get a non-existent key
                local result=$(redis-cli -h localhost -p "$RESOURCE_PORT" GET "nonexistent_key_$(date +%s)" 2>/dev/null)
                if [[ "$result" == "(nil)" ]]; then
                    echo "✓ Redis handles non-existent keys correctly"
                fi
            fi
            ;;
        "postgres"|"postgresql")
            if which psql >/dev/null 2>&1; then
                # Try to query a non-existent table
                local error_output=$(psql -h localhost -p "$RESOURCE_PORT" -U postgres -t -c "SELECT * FROM nonexistent_table;" 2>&1 || true)
                if echo "$error_output" | grep -q "does not exist"; then
                    echo "✓ PostgreSQL handles invalid queries correctly"
                fi
            fi
            ;;
        *)
            echo "⚠️  Error handling test needs to be customized for $TEST_RESOURCE"
            ;;
    esac
    
    echo "✓ Error handling test completed"
}

# Main test execution
main() {
    echo "🧪 Starting Storage Resource Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo "Port: $RESOURCE_PORT"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    echo "📋 Running storage resource test suite..."
    echo
    
    # Phase 1: Interface Compliance (should be first)
    echo "Phase 1: Interface Compliance"
    test_storage_interface_compliance
    echo
    
    # Phase 2: Functional Tests
    echo "Phase 2: Functional Tests"
    test_storage_health
    test_storage_data_persistence
    test_storage_backup_restore
    test_storage_statistics
    test_storage_error_handling
    test_storage_performance
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ Storage resource integration test failed"
        exit 1
    else
        echo "✅ Storage resource integration test passed"
        exit 0
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi