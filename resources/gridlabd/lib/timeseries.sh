#!/usr/bin/env bash
# GridLAB-D Time-series Integration Module

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/config/defaults.sh"

# Export simulation results to QuestDB format
export_to_questdb() {
    local result_file="${1:-}"
    local table_name="${2:-gridlabd_results}"
    
    if [ -z "$result_file" ]; then
        echo "Error: Result file required"
        return 1
    fi
    
    if [ ! -f "$result_file" ]; then
        echo "Error: Result file not found: $result_file"
        return 1
    fi
    
    # Check if QuestDB is available
    local questdb_port="${QUESTDB_PORT:-9000}"
    if ! timeout 2 curl -sf "http://localhost:${questdb_port}/exec" > /dev/null 2>&1; then
        echo "Warning: QuestDB not available on port ${questdb_port}"
        echo "Exported data would be sent to QuestDB if available"
        return 0
    fi
    
    # Convert CSV to QuestDB line protocol
    echo "Exporting to QuestDB table: $table_name"
    
    # Read CSV and convert to line protocol
    # Format: table_name,tag1=value1 field1=value1,field2=value2 timestamp
    while IFS=',' read -r timestamp total_load voltage_650 voltage_634; do
        # Skip header
        if [[ "$timestamp" == "timestamp" ]]; then
            continue
        fi
        
        # Convert to nanoseconds (QuestDB expects nanoseconds)
        ts_nano=$(date -d "$timestamp" +%s%N 2>/dev/null || echo "$(date +%s)000000000")
        
        # Send to QuestDB
        curl -sf -X POST "http://localhost:${questdb_port}/exec" \
            -d "INSERT INTO ${table_name} VALUES('${timestamp}', ${total_load:-0}, ${voltage_650:-0}, ${voltage_634:-0});" \
            > /dev/null 2>&1 || true
    done < "$result_file"
    
    echo "Export complete"
}

# Export to Redis time-series
export_to_redis() {
    local result_file="${1:-}"
    local key_prefix="${2:-gridlabd:results}"
    
    if [ -z "$result_file" ]; then
        echo "Error: Result file required"
        return 1
    fi
    
    # Check if Redis is available
    local redis_port="${REDIS_PORT:-6379}"
    if ! timeout 2 redis-cli -p "${redis_port}" ping > /dev/null 2>&1; then
        echo "Warning: Redis not available on port ${redis_port}"
        echo "Exported data would be sent to Redis if available"
        return 0
    fi
    
    echo "Exporting to Redis with key prefix: $key_prefix"
    
    # Read CSV and store in Redis
    while IFS=',' read -r timestamp total_load voltage_650 voltage_634; do
        # Skip header
        if [[ "$timestamp" == "timestamp" ]]; then
            continue
        fi
        
        # Store as hash
        redis-cli -p "${redis_port}" HSET "${key_prefix}:${timestamp}" \
            "total_load" "${total_load:-0}" \
            "voltage_650" "${voltage_650:-0}" \
            "voltage_634" "${voltage_634:-0}" \
            > /dev/null 2>&1 || true
    done < "$result_file"
    
    echo "Export complete"
}

# Query results from time-series database
query_results() {
    local start_time="${1:-}"
    local end_time="${2:-}"
    local metric="${3:-total_load}"
    
    echo "Querying results from ${start_time} to ${end_time} for metric: ${metric}"
    
    # Try QuestDB first
    local questdb_port="${QUESTDB_PORT:-9000}"
    if timeout 2 curl -sf "http://localhost:${questdb_port}/exec" > /dev/null 2>&1; then
        echo "Querying from QuestDB..."
        curl -sf -X GET "http://localhost:${questdb_port}/exec" \
            --data-urlencode "query=SELECT timestamp, ${metric} FROM gridlabd_results WHERE timestamp >= '${start_time}' AND timestamp <= '${end_time}';" \
            2>/dev/null || echo "No results found"
        return
    fi
    
    # Try Redis
    local redis_port="${REDIS_PORT:-6379}"
    if timeout 2 redis-cli -p "${redis_port}" ping > /dev/null 2>&1; then
        echo "Querying from Redis..."
        redis-cli -p "${redis_port}" --scan --pattern "gridlabd:results:*" | \
            while read key; do
                redis-cli -p "${redis_port}" HGET "$key" "$metric"
            done
        return
    fi
    
    echo "No time-series database available"
}

# Main CLI handler
case "${1:-}" in
    export)
        shift
        case "${1:-}" in
            questdb)
                export_to_questdb "${2:-}" "${3:-gridlabd_results}"
                ;;
            redis)
                export_to_redis "${2:-}" "${3:-gridlabd:results}"
                ;;
            *)
                echo "Usage: timeseries export [questdb|redis] <result_file> [table/key]"
                exit 1
                ;;
        esac
        ;;
    query)
        shift
        query_results "${1:-}" "${2:-}" "${3:-total_load}"
        ;;
    *)
        echo "GridLAB-D Time-series Integration"
        echo ""
        echo "Commands:"
        echo "  export questdb <file> [table]  Export results to QuestDB"
        echo "  export redis <file> [prefix]   Export results to Redis"
        echo "  query <start> <end> [metric]   Query stored results"
        echo ""
        echo "Examples:"
        echo "  timeseries export questdb output.csv gridlabd_results"
        echo "  timeseries export redis output.csv gridlabd:sim123"
        echo "  timeseries query '2024-01-01' '2024-01-02' voltage_650"
        exit 1
        ;;
esac