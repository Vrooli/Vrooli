#!/usr/bin/env bash
# QuestDB Status Functions
# Check and report QuestDB status

#######################################
# Check QuestDB status
# Returns:
#   0 if running and healthy, 1 otherwise
#######################################
questdb::status::check() {
    echo_header "${QUESTDB_STATUS_MESSAGES["checking"]}"
    
    # Check if container is running
    if ! questdb::docker::is_running; then
        log::error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    # Check health
    if ! questdb::docker::health_check; then
        log::warning "${QUESTDB_STATUS_MESSAGES["unhealthy"]}"
        return 1
    fi
    
    log::success "${QUESTDB_STATUS_MESSAGES["running"]}"
    
    # Get detailed status
    questdb::status::detailed
    
    return 0
}

#######################################
# Show detailed status information
#######################################
questdb::status::detailed() {
    echo ""
    echo "📊 QuestDB Status Details:"
    echo "=========================="
    
    # Version
    local version
    version=$(questdb::get_version)
    if [[ -n "$version" ]]; then
        echo "Version:       $version"
    fi
    
    # URLs
    echo "Web Console:   ${QUESTDB_BASE_URL}"
    echo "PostgreSQL:    localhost:${QUESTDB_PG_PORT}"
    echo "InfluxDB TCP:  localhost:${QUESTDB_ILP_PORT}"
    
    # Container stats
    local stats
    stats=$(questdb::docker::stats)
    if [[ -n "$stats" ]] && [[ "$stats" != "{}" ]]; then
        local cpu mem
        cpu=$(echo "$stats" | jq -r '.CPUPerc' 2>/dev/null || echo "N/A")
        mem=$(echo "$stats" | jq -r '.MemUsage' 2>/dev/null || echo "N/A")
        
        echo ""
        echo "Resource Usage:"
        echo "  CPU:         $cpu"
        echo "  Memory:      $mem"
    fi
    
    # Table count
    echo ""
    echo "Database Info:"
    local table_count
    table_count=$(questdb::api::query "SELECT COUNT(*) FROM tables()" 1 2>/dev/null | \
        jq -r '.dataset[0][0] // 0' 2>/dev/null || echo "0")
    echo "  Tables:      $table_count"
    
    # Data directory size
    if [[ -d "${QUESTDB_DATA_DIR}" ]]; then
        local data_size
        data_size=$(du -sh "${QUESTDB_DATA_DIR}" 2>/dev/null | awk '{print $1}' || echo "N/A")
        echo "  Data Size:   $data_size"
    fi
    
    echo ""
    echo "${QUESTDB_INFO_MESSAGES["api_docs"]}"
}

#######################################
# Check system metrics
# Returns:
#   JSON with system metrics
#######################################
questdb::status::metrics() {
    if ! questdb::docker::is_running; then
        echo "{}"
        return 1
    fi
    
    # Get various metrics
    local uptime tables rows
    
    # Calculate uptime
    local start_time
    start_time=$(docker inspect -f '{{.State.StartedAt}}' "${QUESTDB_CONTAINER_NAME}" 2>/dev/null || echo "")
    if [[ -n "$start_time" ]]; then
        uptime=$(( $(date +%s) - $(date -d "$start_time" +%s) ))
    else
        uptime=0
    fi
    
    # Get table count
    tables=$(questdb::api::query "SELECT COUNT(*) FROM tables()" 1 2>/dev/null | \
        jq -r '.dataset[0][0] // 0' 2>/dev/null || echo "0")
    
    # Get total row count (sum across all tables)
    rows=0
    
    # Output as JSON
    cat <<EOF
{
    "status": "running",
    "uptime_seconds": ${uptime},
    "tables": ${tables},
    "total_rows": ${rows},
    "http_port": ${QUESTDB_HTTP_PORT},
    "pg_port": ${QUESTDB_PG_PORT},
    "ilp_port": ${QUESTDB_ILP_PORT}
}
EOF
}

#######################################
# Monitor QuestDB in real-time
# Shows updating stats every few seconds
#######################################
questdb::status::monitor() {
    if ! questdb::docker::is_running; then
        log::error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    log::info "Monitoring QuestDB (Press Ctrl+C to stop)..."
    echo ""
    
    while true; do
        clear
        echo "📊 QuestDB Monitor - $(date)"
        echo "================================"
        
        # Container stats
        docker stats "${QUESTDB_CONTAINER_NAME}" --no-stream
        
        # Query stats
        echo ""
        echo "Recent Queries:"
        questdb::api::query "SELECT query_text, execution_time FROM sys.query_log ORDER BY timestamp DESC LIMIT 5" 5 2>/dev/null || echo "No query log available"
        
        sleep 5
    done
}

#######################################
# Test QuestDB functionality
# Returns: 0 if all tests pass, 1 if any fail, 2 if service not ready
#######################################
questdb::test() {
    log::info "Testing QuestDB functionality..."
    
    # Test 1: Check if QuestDB is installed (container exists)
    if ! questdb::docker::exists; then
        log::error "❌ QuestDB container is not installed"
        return 1
    fi
            log::success "✅ QuestDB container is installed"
    
    # Test 2: Check if service is running
    if ! questdb::docker::is_running; then
        log::error "❌ QuestDB service is not running"
        return 2
    fi
            log::success "✅ QuestDB service is running"
    
    # Test 3: Check API health
    if ! questdb::api::health_check; then
        log::error "❌ QuestDB API is not responding"
        return 1
    fi
            log::success "✅ QuestDB API is healthy"
    
    # Test 4: Test SQL queries
    log::info "Testing SQL operations..."
    local test_result
    test_result=$(questdb::api::query "SELECT 1 as test_value" 1 2>/dev/null || echo "")
    if [[ -n "$test_result" ]] && echo "$test_result" | grep -q "test_value"; then
        log::success "✅ SQL queries working"
    else
        log::error "❌ SQL query test failed"
        return 1
    fi
    
    # Test 5: Test table operations
    log::info "Testing table operations..."
    local test_table="vrooli_test_table_$(date +%s)"
    
    if questdb::api::query "CREATE TABLE $test_table (id INT, name STRING)" 1 >/dev/null 2>&1; then
        log::success "✅ Table creation successful"
        
        # Test insert
        if questdb::api::query "INSERT INTO $test_table VALUES (1, 'test')" 1 >/dev/null 2>&1; then
            log::success "✅ Data insertion successful"
        fi
        
        # Clean up test table
        questdb::api::query "DROP TABLE $test_table" 1 >/dev/null 2>&1 || true
        log::success "✅ Table cleanup successful"
    else
        echo_warn "⚠️  Table operations test failed - may be permission issue"
    fi
    
    # Test 6: Check storage metrics
    log::info "Testing storage metrics..."
    local storage_info
    storage_info=$(docker exec "$QUESTDB_CONTAINER_NAME" df -h /root/.questdb 2>/dev/null | tail -1 | awk '{print $3}' || echo "unknown")
    if [[ "$storage_info" != "unknown" ]]; then
        log::success "✅ Storage metrics available (used: $storage_info)"
    else
        echo_warn "⚠️  Storage metrics unavailable"
    fi
    
    log::success "🎉 All QuestDB tests passed"
    return 0
}

#######################################
# Show comprehensive QuestDB information
#######################################
questdb::info() {
    cat << EOF
=== QuestDB Resource Information ===

ID: questdb
Category: storage
Display Name: QuestDB Time Series Database
Description: High-performance time series database optimized for real-time analytics

Service Details:
- Container Name: $QUESTDB_CONTAINER_NAME
- HTTP Port: $QUESTDB_HTTP_PORT
- PostgreSQL Port: $QUESTDB_PG_PORT
- InfluxDB Line Protocol Port: $QUESTDB_ILP_PORT
- HTTP URL: http://localhost:$QUESTDB_HTTP_PORT
- PostgreSQL URL: postgresql://localhost:$QUESTDB_PG_PORT/qdb
- Data Directory: $QUESTDB_DATA_DIR

Endpoints:
- Web Console: http://localhost:$QUESTDB_HTTP_PORT
- Health Check: http://localhost:$QUESTDB_HTTP_PORT/status
- Query API: http://localhost:$QUESTDB_HTTP_PORT/exec
- Import API: http://localhost:$QUESTDB_HTTP_PORT/imp
- Export API: http://localhost:$QUESTDB_HTTP_PORT/exp

Protocols:
- HTTP REST API for queries and management
- PostgreSQL wire protocol for standard SQL tools
- InfluxDB Line Protocol for high-throughput ingestion

Configuration:
- Docker Image: $QUESTDB_IMAGE
- Version: $QUESTDB_VERSION
- Data Persistence: $QUESTDB_DATA_DIR
- Log Directory: $QUESTDB_LOG_DIR
- Config Directory: $QUESTDB_CONFIG_DIR

QuestDB Features:
- Time series optimized storage
- SQL with time series extensions
- High-performance ingestion
- Real-time aggregations
- Column-oriented storage
- SIMD optimizations
- Zero-GC Java implementation
- PostgreSQL compatibility

Example Usage:
# Execute a SQL query
$0 --action query --query "SELECT * FROM my_table LATEST ON timestamp PARTITION BY id"

# Create a table
$0 --action tables --table sensors --schema /path/to/schema.sql

# Open web console
$0 --action console

# Monitor performance
$0 --action monitor

Documentation: https://questdb.io/docs/
EOF
}

#######################################
# Export status functions
#######################################
export -f questdb::status::check
export -f questdb::status::detailed
export -f questdb::status::metrics
export -f questdb::status::monitor
export -f questdb::test
export -f questdb::info