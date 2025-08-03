#!/usr/bin/env bash
# QuestDB Resource Mock Implementation
# Provides realistic mock responses for QuestDB time-series database service

# Prevent duplicate loading
if [[ "${QUESTDB_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export QUESTDB_MOCK_LOADED="true"

#######################################
# Setup QuestDB mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::questdb::setup() {
    local state="${1:-healthy}"
    
    # Configure QuestDB-specific environment
    export QUESTDB_HTTP_PORT="${QUESTDB_HTTP_PORT:-9010}"
    export QUESTDB_PG_PORT="${QUESTDB_PG_PORT:-8812}"
    export QUESTDB_ILP_PORT="${QUESTDB_ILP_PORT:-9011}"
    export QUESTDB_HOST="${QUESTDB_HOST:-localhost}"
    export QUESTDB_CONTAINER_NAME="${TEST_NAMESPACE}_questdb"
    export QUESTDB_BASE_URL="http://${QUESTDB_HOST}:${QUESTDB_HTTP_PORT}"
    export QUESTDB_PG_URL="postgresql://admin:quest@${QUESTDB_HOST}:${QUESTDB_PG_PORT}/qdb"
    export QUESTDB_PG_USER="${QUESTDB_PG_USER:-admin}"
    export QUESTDB_PG_PASSWORD="${QUESTDB_PG_PASSWORD:-quest}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$QUESTDB_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::questdb::setup_healthy_state
            ;;
        "unhealthy")
            mock::questdb::setup_unhealthy_state
            ;;
        "installing")
            mock::questdb::setup_installing_state
            ;;
        "stopped")
            mock::questdb::setup_stopped_state
            ;;
        *)
            echo "[QUESTDB_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[QUESTDB_MOCK] QuestDB mock configured with state: $state"
}

#######################################
# Setup healthy QuestDB state
#######################################
mock::questdb::setup_healthy_state() {
    # Set container logs
    local log_file="${MOCK_RESPONSES_DIR}/questdb_logs.txt"
    cat > "$log_file" << 'EOF'
2024-01-15T10:00:00.000000Z I server-main QuestDB is now listening on HTTP :9010
2024-01-15T10:00:00.001000Z I server-main QuestDB is now listening on PostgreSQL :8812
2024-01-15T10:00:00.002000Z I server-main QuestDB is now listening on InfluxDB Line Protocol :9011
2024-01-15T10:00:00.003000Z I server-main enjoy
EOF

    # Set up HTTP API endpoints
    mock::http::set_endpoint_response "$QUESTDB_BASE_URL/status" \
        '{"status":"OK","version":"8.1.2","git_hash":"b94dc3b0"}' 200
    
    mock::http::set_endpoint_response "$QUESTDB_BASE_URL/exec?query=SELECT%20table_name%20FROM%20tables()&fmt=json" \
        '{"query":"SELECT table_name FROM tables()","columns":[{"name":"table_name","type":"STRING"}],"dataset":[["system_metrics"],["ai_inference"],["resource_health"],["workflow_metrics"]],"count":4}' 200
    
    mock::http::set_endpoint_response "$QUESTDB_BASE_URL/exec?query=SELECT%20COUNT(*)%20as%20count%20FROM%20system_metrics&fmt=json" \
        '{"query":"SELECT COUNT(*) as count FROM system_metrics","columns":[{"name":"count","type":"LONG"}],"dataset":[[1000]],"count":1}' 200
    
    mock::http::set_endpoint_response "$QUESTDB_BASE_URL/exec?query=SHOW%20COLUMNS%20FROM%20system_metrics&fmt=json" \
        '{"query":"SHOW COLUMNS FROM system_metrics","columns":[{"name":"column","type":"STRING"},{"name":"type","type":"STRING"},{"name":"indexed","type":"BOOLEAN"}],"dataset":[["timestamp","TIMESTAMP",false],["metric_name","STRING",true],["value","DOUBLE",false],["host","STRING",true]],"count":4}' 200
}

#######################################
# Setup unhealthy QuestDB state
#######################################
mock::questdb::setup_unhealthy_state() {
    # Set error logs
    local log_file="${MOCK_RESPONSES_DIR}/questdb_logs.txt"
    cat > "$log_file" << 'EOF'
2024-01-15T10:00:00.000000Z E server-main could not bind socket [errno=98, address=0.0.0.0:9010]
2024-01-15T10:00:00.001000Z E server-main server start failed
EOF

    # Set up failing HTTP endpoints
    mock::http::set_endpoint_response "$QUESTDB_BASE_URL/status" \
        '{"error":"Service unavailable"}' 503
    
    mock::http::set_endpoint_unreachable "$QUESTDB_BASE_URL"
}

#######################################
# Setup installing QuestDB state
#######################################
mock::questdb::setup_installing_state() {
    # Set initialization logs
    local log_file="${MOCK_RESPONSES_DIR}/questdb_logs.txt"
    cat > "$log_file" << 'EOF'
2024-01-15T10:00:00.000000Z I init starting database
2024-01-15T10:00:00.001000Z I init creating default tables
2024-01-15T10:00:00.002000Z I init loading configuration
2024-01-15T10:00:00.003000Z I init starting HTTP server...
EOF

    # Set up loading state endpoint
    mock::http::set_endpoint_response "$QUESTDB_BASE_URL/status" \
        '{"status":"STARTING","version":"8.1.2","git_hash":"b94dc3b0"}' 503
}

#######################################
# Setup stopped QuestDB state
#######################################
mock::questdb::setup_stopped_state() {
    # No logs when stopped
    local log_file="${MOCK_RESPONSES_DIR}/questdb_logs.txt"
    echo "" > "$log_file"
    
    # Make all endpoints unreachable
    mock::http::set_endpoint_unreachable "$QUESTDB_BASE_URL"
}

#######################################
# Mock psql command for QuestDB PostgreSQL interface
#######################################
mock::questdb::psql() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "psql $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Parse arguments
    local host="$QUESTDB_HOST"
    local port="$QUESTDB_PG_PORT"
    local user="$QUESTDB_PG_USER"
    local database="qdb"
    local command=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--host)
                host="$2"
                shift 2
                ;;
            -p|--port)
                port="$2"
                shift 2
                ;;
            -U|--username)
                user="$2"
                shift 2
                ;;
            -d|--dbname)
                database="$2"
                shift 2
                ;;
            -c|--command)
                command="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check container state
    local container_state=$(docker inspect "$QUESTDB_CONTAINER_NAME" --format '{{.State.Status}}' 2>/dev/null || echo "stopped")
    
    if [[ "$container_state" != "running" ]]; then
        echo "psql: error: could not connect to server: Connection refused" >&2
        return 2
    fi
    
    # Handle QuestDB-specific SQL commands
    case "$command" in
        "SELECT version();")
            echo " version "
            echo "---------"
            echo " QuestDB 8.1.2"
            echo "(1 row)"
            ;;
        "SELECT table_name FROM tables();"|"SELECT table_name FROM tables()")
            echo " table_name "
            echo "-------------"
            echo " system_metrics"
            echo " ai_inference"
            echo " resource_health"
            echo " workflow_metrics"
            echo "(4 rows)"
            ;;
        "SHOW COLUMNS FROM system_metrics;"|"SHOW COLUMNS FROM system_metrics")
            echo "    column    |   type    | indexed "
            echo "-------------+-----------+---------"
            echo " timestamp   | TIMESTAMP | f"
            echo " metric_name | STRING    | t"
            echo " value       | DOUBLE    | f"
            echo " host        | STRING    | t"
            echo "(4 rows)"
            ;;
        "SELECT COUNT(*) FROM system_metrics;"|"SELECT COUNT(*) FROM system_metrics")
            echo " count "
            echo "-------"
            echo "  1000"
            echo "(1 row)"
            ;;
        *"CREATE TABLE"*)
            echo "CREATE TABLE"
            ;;
        *"INSERT INTO"*)
            echo "INSERT 0 1"
            ;;
        *"DROP TABLE"*)
            echo "DROP TABLE"
            ;;
        "SELECT 1;")
            echo " ?column? "
            echo "----------"
            echo "        1"
            echo "(1 row)"
            ;;
        *)
            # Generic success response
            echo "Query executed successfully"
            ;;
    esac
    
    return 0
}

#######################################
# Mock nc (netcat) for InfluxDB Line Protocol
#######################################
mock::questdb::nc() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "nc $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local host=""
    local port=""
    local timeout=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -w)
                timeout="$2"
                shift 2
                ;;
            *)
                if [[ "$1" =~ ^[0-9]+$ ]]; then
                    port="$1"
                elif [[ ! "$1" =~ ^- ]]; then
                    host="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Check if targeting QuestDB ILP port
    if [[ "$port" == "$QUESTDB_ILP_PORT" ]]; then
        # Check container state
        local container_state=$(docker inspect "$QUESTDB_CONTAINER_NAME" --format '{{.State.Status}}' 2>/dev/null || echo "stopped")
        
        if [[ "$container_state" == "running" ]]; then
            # Accept the data (read from stdin but don't output it)
            while IFS= read -r line; do
                # Simulate successful data ingestion
                true
            done
            return 0
        else
            return 1
        fi
    fi
    
    # Default nc behavior for other ports
    return 1
}

#######################################
# Mock QuestDB-specific operations
#######################################

# Mock table creation
mock::questdb::create_table() {
    local table_name="$1"
    local schema="${2:-timestamp TIMESTAMP, value DOUBLE}"
    
    echo "CREATE TABLE '$table_name' ($schema)"
}

# Mock data insertion via ILP
mock::questdb::insert_ilp() {
    local measurement="$1"
    local tags="${2:-host=localhost}"
    local fields="${3:-value=42.0}"
    local timestamp="${4:-$(date +%s)000000000}"
    
    echo "${measurement},${tags} ${fields} ${timestamp}" | mock::questdb::nc -w 5 localhost "$QUESTDB_ILP_PORT"
}

# Mock bulk CSV import
mock::questdb::import_csv() {
    local table="$1"
    local csv_file="$2"
    
    if [[ ! -f "$csv_file" ]]; then
        echo "CSV file not found: $csv_file" >&2
        return 1
    fi
    
    echo "Imported $(wc -l < "$csv_file") rows into table '$table'"
}

# Mock query execution
mock::questdb::query() {
    local sql="$1"
    local format="${2:-json}"
    
    # Simulate different query responses
    case "$sql" in
        *"COUNT(*)"*)
            echo '{"query":"'"$sql"'","columns":[{"name":"count","type":"LONG"}],"dataset":[[1000]],"count":1}'
            ;;
        *"SELECT table_name FROM tables()"*)
            echo '{"query":"'"$sql"'","columns":[{"name":"table_name","type":"STRING"}],"dataset":[["system_metrics"],["ai_inference"],["resource_health"],["workflow_metrics"]],"count":4}'
            ;;
        *"SHOW COLUMNS"*)
            echo '{"query":"'"$sql"'","columns":[{"name":"column","type":"STRING"},{"name":"type","type":"STRING"},{"name":"indexed","type":"BOOLEAN"}],"dataset":[["timestamp","TIMESTAMP",false],["metric_name","STRING",true],["value","DOUBLE",false]],"count":3}'
            ;;
        *)
            echo '{"query":"'"$sql"'","columns":[],"dataset":[],"count":0}'
            ;;
    esac
}

# Mock table row count
mock::questdb::table_count() {
    local table="$1"
    
    case "$table" in
        "system_metrics")
            echo "1000"
            ;;
        "ai_inference")
            echo "500"
            ;;
        "resource_health")
            echo "250"
            ;;
        "workflow_metrics")
            echo "750"
            ;;
        *)
            echo "0"
            ;;
    esac
}

# Mock health check
mock::questdb::health_check() {
    local container_state=$(docker inspect "$QUESTDB_CONTAINER_NAME" --format '{{.State.Status}}' 2>/dev/null || echo "stopped")
    
    if [[ "$container_state" == "running" ]]; then
        echo "OK"
        return 0
    else
        echo "FAILED"
        return 1
    fi
}

#######################################
# Export mock functions
#######################################
export -f mock::questdb::setup
export -f mock::questdb::setup_healthy_state
export -f mock::questdb::setup_unhealthy_state
export -f mock::questdb::setup_installing_state
export -f mock::questdb::setup_stopped_state
export -f mock::questdb::psql
export -f mock::questdb::nc
export -f mock::questdb::create_table
export -f mock::questdb::insert_ilp
export -f mock::questdb::import_csv
export -f mock::questdb::query
export -f mock::questdb::table_count
export -f mock::questdb::health_check

echo "[QUESTDB_MOCK] QuestDB mock implementation loaded"