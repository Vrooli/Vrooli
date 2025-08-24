#!/usr/bin/env bash
# QuestDB Mock - Tier 2 (Stateful)
# 
# Provides stateful QuestDB time-series database mocking for testing:
# - Table creation and management
# - SQL query execution
# - ILP (InfluxDB Line Protocol) ingestion
# - Time-series data storage
# - Metrics aggregation
# - Error injection for resilience testing
#
# Coverage: ~80% of common QuestDB operations in 450 lines

# === Configuration ===
declare -gA QUESTDB_TABLES=()             # Table_name -> "columns|row_count"
declare -gA QUESTDB_DATA=()               # Table:row -> "timestamp|values"
declare -gA QUESTDB_METRICS=()            # Metric_name -> "value|timestamp"
declare -gA QUESTDB_QUERIES=()            # Query_hash -> "result_cache"
declare -gA QUESTDB_CONFIG=(              # Service configuration
    [status]="running"
    [http_port]="9010"
    [pg_port]="8812"
    [ilp_port]="9011"
    [database]="qdb"
    [error_mode]=""
    [version]="8.1.2"
)

# Debug mode
declare -g QUESTDB_DEBUG="${QUESTDB_DEBUG:-}"

# === Helper Functions ===
questdb_debug() {
    [[ -n "$QUESTDB_DEBUG" ]] && echo "[MOCK:QUESTDB] $*" >&2
}

questdb_check_error() {
    case "${QUESTDB_CONFIG[error_mode]}" in
        "service_down")
            echo "Error: QuestDB service is not running" >&2
            return 1
            ;;
        "table_not_found")
            echo "Error: Table not found" >&2
            return 1
            ;;
        "query_error")
            echo "Error: Query execution failed" >&2
            return 1
            ;;
    esac
    return 0
}

questdb_hash_query() {
    echo "$1" | md5sum | cut -d' ' -f1
}

# === Main QuestDB Command ===
questdb() {
    questdb_debug "questdb called with: $*"
    
    if ! questdb_check_error; then
        return $?
    fi
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        create)
            questdb_cmd_create "$@"
            ;;
        insert)
            questdb_cmd_insert "$@"
            ;;
        query)
            questdb_cmd_query "$@"
            ;;
        ilp)
            questdb_cmd_ilp "$@"
            ;;
        tables)
            questdb_cmd_tables "$@"
            ;;
        drop)
            questdb_cmd_drop "$@"
            ;;
        status)
            questdb_cmd_status "$@"
            ;;
        start|stop|restart)
            questdb_cmd_service "$command" "$@"
            ;;
        *)
            echo "QuestDB CLI - Time-Series Database"
            echo "Commands:"
            echo "  create  - Create table"
            echo "  insert  - Insert data"
            echo "  query   - Execute SQL query"
            echo "  ilp     - Send ILP data"
            echo "  tables  - List tables"
            echo "  drop    - Drop table"
            echo "  status  - Show service status"
            echo "  start   - Start service"
            echo "  stop    - Stop service"
            echo "  restart - Restart service"
            ;;
    esac
}

# === Create Table ===
questdb_cmd_create() {
    local table="${1:-}"
    local columns="${2:-}"
    
    [[ -z "$table" ]] && { echo "Error: table name required" >&2; return 1; }
    [[ -z "$columns" ]] && { echo "Error: column definitions required" >&2; return 1; }
    
    if [[ -n "${QUESTDB_TABLES[$table]}" ]]; then
        echo "Error: table already exists: $table" >&2
        return 1
    fi
    
    QUESTDB_TABLES[$table]="$columns|0"
    questdb_debug "Created table: $table with columns: $columns"
    echo "Table '$table' created successfully"
}

# === Insert Data ===
questdb_cmd_insert() {
    local table="${1:-}"
    local values="${2:-}"
    
    [[ -z "$table" ]] && { echo "Error: table name required" >&2; return 1; }
    [[ -z "$values" ]] && { echo "Error: values required" >&2; return 1; }
    
    if [[ -z "${QUESTDB_TABLES[$table]}" ]]; then
        echo "Error: table not found: $table" >&2
        return 1
    fi
    
    local data="${QUESTDB_TABLES[$table]}"
    IFS='|' read -r columns row_count <<< "$data"
    
    # Add timestamp if not provided
    local timestamp=$(date +%s%N)
    local row_key="$table:$row_count"
    QUESTDB_DATA[$row_key]="$timestamp|$values"
    
    # Update row count
    QUESTDB_TABLES[$table]="$columns|$((row_count + 1))"
    
    questdb_debug "Inserted row into $table"
    echo "1 row inserted"
}

# === Query Execution ===
questdb_cmd_query() {
    local sql="${1:-}"
    
    [[ -z "$sql" ]] && { echo "Error: SQL query required" >&2; return 1; }
    
    local query_hash=$(questdb_hash_query "$sql")
    
    # Check cache
    if [[ -n "${QUESTDB_QUERIES[$query_hash]}" ]]; then
        echo "${QUESTDB_QUERIES[$query_hash]}"
        return 0
    fi
    
    # Parse query type
    local result=""
    if [[ "$sql" =~ ^[Ss][Ee][Ll][Ee][Cc][Tt] ]]; then
        # SELECT query
        if [[ "$sql" =~ [Ff][Rr][Oo][Mm][[:space:]]+([^[:space:]]+) ]]; then
            local table="${BASH_REMATCH[1]}"
            
            if [[ -z "${QUESTDB_TABLES[$table]}" ]]; then
                echo "Error: table not found: $table" >&2
                return 1
            fi
            
            local data="${QUESTDB_TABLES[$table]}"
            IFS='|' read -r columns row_count <<< "$data"
            
            echo "timestamp,value,metric"
            echo "$(date -Iseconds),42.5,cpu_usage"
            echo "$(date -Iseconds),128.3,memory_mb"
            echo "$(date -Iseconds),0.85,disk_usage"
            echo "Rows: 3"
        else
            echo "timestamp,count"
            echo "$(date -Iseconds),100"
            echo "Rows: 1"
        fi
    elif [[ "$sql" =~ ^[Ss][Hh][Oo][Ww] ]]; then
        # SHOW query
        questdb_cmd_tables
    else
        echo "Query executed successfully"
    fi
    
    # Cache result
    QUESTDB_QUERIES[$query_hash]="$result"
}

# === ILP Ingestion ===
questdb_cmd_ilp() {
    local data="${1:-}"
    
    [[ -z "$data" ]] && { echo "Error: ILP data required" >&2; return 1; }
    
    # Parse ILP format: measurement,tag=value field=value timestamp
    if [[ "$data" =~ ^([^,]+)(.*)[[:space:]]([^[:space:]]+)[[:space:]]([0-9]+)$ ]]; then
        local measurement="${BASH_REMATCH[1]}"
        local tags="${BASH_REMATCH[2]}"
        local fields="${BASH_REMATCH[3]}"
        local timestamp="${BASH_REMATCH[4]}"
        
        # Create table if not exists
        if [[ -z "${QUESTDB_TABLES[$measurement]}" ]]; then
            QUESTDB_TABLES[$measurement]="timestamp:TIMESTAMP,tags:STRING,fields:STRING|0"
        fi
        
        # Store metric
        QUESTDB_METRICS[$measurement]="$fields|$timestamp"
        
        questdb_debug "ILP ingested: $measurement"
        echo "ILP data ingested successfully"
    else
        echo "Error: Invalid ILP format" >&2
        return 1
    fi
}

# === List Tables ===
questdb_cmd_tables() {
    echo "Tables:"
    if [[ ${#QUESTDB_TABLES[@]} -eq 0 ]]; then
        echo "  (none)"
    else
        for table in "${!QUESTDB_TABLES[@]}"; do
            local data="${QUESTDB_TABLES[$table]}"
            IFS='|' read -r columns row_count <<< "$data"
            echo "  $table - Rows: $row_count"
        done
    fi
}

# === Drop Table ===
questdb_cmd_drop() {
    local table="${1:-}"
    
    [[ -z "$table" ]] && { echo "Error: table name required" >&2; return 1; }
    
    if [[ -n "${QUESTDB_TABLES[$table]}" ]]; then
        unset QUESTDB_TABLES[$table]
        # Remove associated data
        for key in "${!QUESTDB_DATA[@]}"; do
            if [[ "$key" =~ ^$table: ]]; then
                unset QUESTDB_DATA[$key]
            fi
        done
        echo "Table '$table' dropped"
    else
        echo "Error: table not found: $table" >&2
        return 1
    fi
}

# === Status Command ===
questdb_cmd_status() {
    echo "QuestDB Status"
    echo "=============="
    echo "Service: ${QUESTDB_CONFIG[status]}"
    echo "HTTP Port: ${QUESTDB_CONFIG[http_port]}"
    echo "PostgreSQL Port: ${QUESTDB_CONFIG[pg_port]}"
    echo "ILP Port: ${QUESTDB_CONFIG[ilp_port]}"
    echo "Database: ${QUESTDB_CONFIG[database]}"
    echo "Version: ${QUESTDB_CONFIG[version]}"
    echo ""
    echo "Tables: ${#QUESTDB_TABLES[@]}"
    echo "Metrics: ${#QUESTDB_METRICS[@]}"
    echo "Cached Queries: ${#QUESTDB_QUERIES[@]}"
}

# === Service Management ===
questdb_cmd_service() {
    local action="$1"
    
    case "$action" in
        start)
            if [[ "${QUESTDB_CONFIG[status]}" == "running" ]]; then
                echo "QuestDB is already running"
            else
                QUESTDB_CONFIG[status]="running"
                echo "QuestDB started"
                echo "HTTP: localhost:${QUESTDB_CONFIG[http_port]}"
                echo "PostgreSQL: localhost:${QUESTDB_CONFIG[pg_port]}"
                echo "ILP: localhost:${QUESTDB_CONFIG[ilp_port]}"
            fi
            ;;
        stop)
            QUESTDB_CONFIG[status]="stopped"
            echo "QuestDB stopped"
            ;;
        restart)
            QUESTDB_CONFIG[status]="stopped"
            QUESTDB_CONFIG[status]="running"
            echo "QuestDB restarted"
            ;;
    esac
}

# === HTTP API Mock (via curl interceptor) ===
curl() {
    questdb_debug "curl called with: $*"
    
    local url="" method="GET" data=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X) method="$2"; shift 2 ;;
            -d|--data) data="$2"; shift 2 ;;
            http*) url="$1"; shift ;;
            *) shift ;;
        esac
    done
    
    # Check if this is a QuestDB API call
    if [[ "$url" =~ localhost:9010 || "$url" =~ 127.0.0.1:9010 ]]; then
        questdb_handle_api "$method" "$url" "$data"
        return $?
    fi
    
    echo "curl: Not a QuestDB endpoint"
    return 0
}

questdb_handle_api() {
    local method="$1" url="$2" data="$3"
    
    case "$url" in
        */exec)
            if [[ "$method" == "GET" ]] || [[ "$method" == "POST" ]]; then
                echo '{"query":"SELECT","count":3,"dataset":[["'$(date -Iseconds)'",42.5],["'$(date -Iseconds)'",128.3]]}'
            else
                echo '{"error":"Method not allowed"}'
            fi
            ;;
        */imp)
            if [[ "$method" == "POST" ]]; then
                echo '{"status":"OK","rowsInserted":1}'
            else
                echo '{"error":"Method not allowed"}'
            fi
            ;;
        */health)
            if [[ "${QUESTDB_CONFIG[status]}" == "running" ]]; then
                echo '{"status":"healthy","database":"'${QUESTDB_CONFIG[database]}'","version":"'${QUESTDB_CONFIG[version]}'"}'
            else
                echo '{"status":"unhealthy","error":"Service not running"}'
            fi
            ;;
        *)
            echo '{"status":"ok","version":"'${QUESTDB_CONFIG[version]}'"}'
            ;;
    esac
}

# === Mock Control Functions ===
questdb_mock_reset() {
    questdb_debug "Resetting mock state"
    
    QUESTDB_TABLES=()
    QUESTDB_DATA=()
    QUESTDB_METRICS=()
    QUESTDB_QUERIES=()
    QUESTDB_CONFIG[error_mode]=""
    QUESTDB_CONFIG[status]="running"
    
    # Initialize defaults
    questdb_mock_init_defaults
}

questdb_mock_init_defaults() {
    # Default tables
    QUESTDB_TABLES["system_metrics"]="timestamp:TIMESTAMP,metric:STRING,value:DOUBLE|0"
    QUESTDB_TABLES["ai_inference"]="timestamp:TIMESTAMP,model:STRING,latency_ms:DOUBLE|0"
    QUESTDB_TABLES["resource_health"]="timestamp:TIMESTAMP,resource:STRING,cpu:DOUBLE,memory:DOUBLE|0"
}

questdb_mock_set_error() {
    QUESTDB_CONFIG[error_mode]="$1"
    questdb_debug "Set error mode: $1"
}

questdb_mock_dump_state() {
    echo "=== QuestDB Mock State ==="
    echo "Status: ${QUESTDB_CONFIG[status]}"
    echo "HTTP Port: ${QUESTDB_CONFIG[http_port]}"
    echo "PG Port: ${QUESTDB_CONFIG[pg_port]}"
    echo "ILP Port: ${QUESTDB_CONFIG[ilp_port]}"
    echo "Tables: ${#QUESTDB_TABLES[@]}"
    for table in "${!QUESTDB_TABLES[@]}"; do
        local data="${QUESTDB_TABLES[$table]}"
        IFS='|' read -r columns row_count <<< "$data"
        echo "  $table: $row_count rows"
    done
    echo "Metrics: ${#QUESTDB_METRICS[@]}"
    echo "Error Mode: ${QUESTDB_CONFIG[error_mode]:-none}"
    echo "====================="
}

# === Convention-based Test Functions ===
test_questdb_connection() {
    questdb_debug "Testing connection..."
    
    local result
    result=$(curl -s http://localhost:9010/health 2>&1)
    
    if [[ "$result" =~ "status" ]]; then
        questdb_debug "Connection test passed"
        return 0
    else
        questdb_debug "Connection test failed"
        return 1
    fi
}

test_questdb_health() {
    questdb_debug "Testing health..."
    
    test_questdb_connection || return 1
    
    questdb tables >/dev/null 2>&1 || return 1
    questdb create test_table "timestamp:TIMESTAMP,value:DOUBLE" >/dev/null 2>&1 || return 1
    questdb drop test_table >/dev/null 2>&1 || return 1
    
    questdb_debug "Health test passed"
    return 0
}

test_questdb_basic() {
    questdb_debug "Testing basic operations..."
    
    questdb create metrics "timestamp:TIMESTAMP,value:DOUBLE" >/dev/null 2>&1 || return 1
    questdb insert metrics "42.5" >/dev/null 2>&1 || return 1
    questdb query "SELECT * FROM metrics" >/dev/null 2>&1 || return 1
    questdb ilp "cpu,host=server1 usage=0.85 $(date +%s%N)" >/dev/null 2>&1 || return 1
    
    questdb_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f questdb curl
export -f test_questdb_connection test_questdb_health test_questdb_basic
export -f questdb_mock_reset questdb_mock_set_error questdb_mock_dump_state
export -f questdb_debug questdb_check_error

# Initialize with defaults
questdb_mock_reset
questdb_debug "QuestDB Tier 2 mock initialized"