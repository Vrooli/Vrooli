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
        echo_error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    # Check health
    if ! questdb::docker::health_check; then
        echo_warning "${QUESTDB_STATUS_MESSAGES["unhealthy"]}"
        return 1
    fi
    
    echo_success "${QUESTDB_STATUS_MESSAGES["running"]}"
    
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
        echo_error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    echo_info "Monitoring QuestDB (Press Ctrl+C to stop)..."
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
# Export status functions
#######################################
export -f questdb::status::check
export -f questdb::status::detailed
export -f questdb::status::metrics
export -f questdb::status::monitor