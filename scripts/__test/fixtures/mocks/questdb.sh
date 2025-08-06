#!/usr/bin/env bash
# QuestDB Mock Implementation
#
# Provides a comprehensive mock for QuestDB operations including:
# - QuestDB HTTP API mocking
# - PostgreSQL wire protocol simulation
# - InfluxDB Line Protocol (ILP) support
# - Time-series table management
# - Query execution and response mocking
# - State persistence for BATS compatibility
#
# This mock follows the same standards as docker.sh and redis.sh with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions

# Prevent duplicate loading
[[ -n "${QUESTDB_MOCK_LOADED:-}" ]] && return 0
declare -g QUESTDB_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"
[[ -f "$MOCK_DIR/docker.sh" ]] && source "$MOCK_DIR/docker.sh"
[[ -f "$MOCK_DIR/http.sh" ]] && source "$MOCK_DIR/http.sh"

# Global configuration
declare -g QUESTDB_MOCK_STATE_DIR="${QUESTDB_MOCK_STATE_DIR:-/tmp/questdb-mock-state}"
declare -g QUESTDB_MOCK_DEBUG="${QUESTDB_MOCK_DEBUG:-}"

# Global state arrays
declare -gA QUESTDB_MOCK_TABLES=()          # table_name -> "column1:type1,column2:type2"
declare -gA QUESTDB_MOCK_TABLE_ROWS=()      # table_name -> row_count
declare -gA QUESTDB_MOCK_TABLE_DATA=()      # table_name:row_index -> "value1,value2,value3"
declare -gA QUESTDB_MOCK_QUERIES=()         # query_hash -> cached_response
declare -gA QUESTDB_MOCK_METRICS=()         # metric_name -> latest_value
declare -gA QUESTDB_MOCK_CONFIG=(           # QuestDB configuration
    [host]="localhost"
    [http_port]="9010"
    [pg_port]="8812"
    [ilp_port]="9011"
    [username]="admin"
    [password]="quest"
    [database]="qdb"
    [version]="8.1.2"
    [status]="healthy"
    [container_name]="questdb"
    [connected]="true"
    [error_mode]=""
)

# ILP buffer for batch operations
declare -ga QUESTDB_MOCK_ILP_BUFFER=()

# Initialize state directory
mkdir -p "$QUESTDB_MOCK_STATE_DIR"

# State persistence functions
mock::questdb::save_state() {
    local state_file="$QUESTDB_MOCK_STATE_DIR/questdb-state.sh"
    {
        echo "# QuestDB mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p QUESTDB_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QUESTDB_MOCK_CONFIG=()"
        declare -p QUESTDB_MOCK_TABLES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QUESTDB_MOCK_TABLES=()"
        declare -p QUESTDB_MOCK_TABLE_ROWS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QUESTDB_MOCK_TABLE_ROWS=()"
        declare -p QUESTDB_MOCK_TABLE_DATA 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QUESTDB_MOCK_TABLE_DATA=()"
        declare -p QUESTDB_MOCK_QUERIES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QUESTDB_MOCK_QUERIES=()"
        declare -p QUESTDB_MOCK_METRICS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA QUESTDB_MOCK_METRICS=()"
        declare -p QUESTDB_MOCK_ILP_BUFFER 2>/dev/null | sed 's/declare -a/declare -ga/' || echo "declare -ga QUESTDB_MOCK_ILP_BUFFER=()"
    } > "$state_file"
    
    # Log if function exists
    if declare -f mock::log_state >/dev/null 2>&1; then
        mock::log_state "questdb" "Saved QuestDB state to $state_file"
    fi
}

mock::questdb::load_state() {
    local state_file="$QUESTDB_MOCK_STATE_DIR/questdb-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        # Log if function exists
        if declare -f mock::log_state >/dev/null 2>&1; then
            mock::log_state "questdb" "Loaded QuestDB state from $state_file"
        fi
    fi
}

# Reset QuestDB mock to clean state
mock::questdb::reset() {
    # Recreate as associative arrays
    declare -gA QUESTDB_MOCK_TABLES=()
    declare -gA QUESTDB_MOCK_TABLE_ROWS=()
    declare -gA QUESTDB_MOCK_TABLE_DATA=()
    declare -gA QUESTDB_MOCK_QUERIES=()
    declare -gA QUESTDB_MOCK_METRICS=()
    declare -gA QUESTDB_MOCK_CONFIG=(
        [host]="localhost"
        [http_port]="9010"
        [pg_port]="8812"
        [ilp_port]="9011"
        [username]="admin"
        [password]="quest"
        [database]="qdb"
        [version]="8.1.2"
        [status]="healthy"
        [container_name]="questdb"
        [connected]="true"
        [error_mode]=""
    )
    
    # Reset ILP buffer
    declare -ga QUESTDB_MOCK_ILP_BUFFER=()
    
    # Initialize default tables
    mock::questdb::init_default_tables
    
    # Save clean state
    mock::questdb::save_state
    
    echo "[QUESTDB_MOCK] QuestDB state reset"
}

# Initialize default tables
mock::questdb::init_default_tables() {
    # System metrics table
    QUESTDB_MOCK_TABLES["system_metrics"]="timestamp:TIMESTAMP,metric_name:STRING,value:DOUBLE,host:STRING"
    QUESTDB_MOCK_TABLE_ROWS["system_metrics"]="1000"
    
    # AI inference table
    QUESTDB_MOCK_TABLES["ai_inference"]="timestamp:TIMESTAMP,model:STRING,latency_ms:DOUBLE,tokens:LONG,status:STRING"
    QUESTDB_MOCK_TABLE_ROWS["ai_inference"]="500"
    
    # Resource health table
    QUESTDB_MOCK_TABLES["resource_health"]="timestamp:TIMESTAMP,resource:STRING,cpu_percent:DOUBLE,memory_mb:DOUBLE,status:STRING"
    QUESTDB_MOCK_TABLE_ROWS["resource_health"]="250"
    
    # Workflow metrics table
    QUESTDB_MOCK_TABLES["workflow_metrics"]="timestamp:TIMESTAMP,workflow_id:STRING,duration_ms:DOUBLE,success:BOOLEAN,error_message:STRING"
    QUESTDB_MOCK_TABLE_ROWS["workflow_metrics"]="750"
    
    mock::questdb::save_state
}

# Automatically load state when sourced
mock::questdb::load_state

# Main curl interceptor for HTTP API
curl() {
    # Log command if function exists
    if declare -f mock::log_and_verify >/dev/null 2>&1; then
        mock::log_and_verify "curl" "$@"
    fi
    
    local url=""
    local method="GET"
    local data=""
    local headers=()
    local output_format=""
    local silent=""
    
    # Parse curl arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X|--request)
                method="$2"
                shift 2
                ;;
            -d|--data|--data-raw)
                data="$2"
                shift 2
                ;;
            -H|--header)
                headers+=("$2")
                shift 2
                ;;
            -s|--silent)
                silent="true"
                shift
                ;;
            -o|--output)
                output_format="$2"
                shift 2
                ;;
            http*://*)
                url="$1"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check if this is a QuestDB URL
    local questdb_base="http://${QUESTDB_MOCK_CONFIG[host]}:${QUESTDB_MOCK_CONFIG[http_port]}"
    if [[ "$url" == "$questdb_base"* ]]; then
        mock::questdb::handle_http_request "$url" "$method" "$data"
        return $?
    fi
    
    # Pass through to real curl for non-QuestDB URLs
    if command -v /usr/bin/curl &>/dev/null; then
        /usr/bin/curl "$@"
    else
        echo "curl: command not found" >&2
        return 127
    fi
}

# Handle QuestDB HTTP API requests
mock::questdb::handle_http_request() {
    local url="$1"
    local method="$2"
    local data="$3"
    
    # Check connection status
    if [[ "${QUESTDB_MOCK_CONFIG[connected]}" != "true" ]]; then
        echo "curl: (7) Failed to connect to ${QUESTDB_MOCK_CONFIG[host]} port ${QUESTDB_MOCK_CONFIG[http_port]}: Connection refused" >&2
        return 7
    fi
    
    # Check for error injection
    if [[ -n "${QUESTDB_MOCK_CONFIG[error_mode]}" ]]; then
        case "${QUESTDB_MOCK_CONFIG[error_mode]}" in
            "timeout")
                sleep 5
                echo "curl: (28) Operation timed out" >&2
                return 28
                ;;
            "server_error")
                echo '{"error":"Internal server error"}' 
                return 0
                ;;
            "auth_error")
                echo '{"error":"Authentication required"}' 
                return 0
                ;;
        esac
    fi
    
    # Extract path and query from URL
    local path="${url#*:${QUESTDB_MOCK_CONFIG[http_port]}}"
    local endpoint="${path%%\?*}"
    local query_string="${path#*\?}"
    
    # Handle different endpoints
    case "$endpoint" in
        "/status")
            mock::questdb::handle_status
            ;;
        "/exec")
            mock::questdb::handle_exec "$query_string" "$data"
            ;;
        "/imp")
            mock::questdb::handle_import "$method" "$data"
            ;;
        "/query")
            mock::questdb::handle_query "$query_string"
            ;;
        *)
            echo '{"error":"Unknown endpoint"}'
            return 1
            ;;
    esac
}

# Handle status endpoint
mock::questdb::handle_status() {
    if [[ "${QUESTDB_MOCK_CONFIG[status]}" == "healthy" ]]; then
        echo "{\"status\":\"OK\",\"version\":\"${QUESTDB_MOCK_CONFIG[version]}\",\"git_hash\":\"b94dc3b0\"}"
    else
        echo "{\"status\":\"${QUESTDB_MOCK_CONFIG[status]}\",\"version\":\"${QUESTDB_MOCK_CONFIG[version]}\"}"
    fi
}

# Handle exec endpoint (SQL execution)
mock::questdb::handle_exec() {
    local query_string="$1"
    local post_data="$2"
    
    # Extract query from query string or POST data
    local query=""
    if [[ -n "$query_string" ]]; then
        # URL decode the query parameter
        query=$(echo "$query_string" | sed 's/query=//' | sed 's/%20/ /g' | sed 's/%28/(/g' | sed 's/%29/)/g' | sed 's/%2A/*/g')
    elif [[ -n "$post_data" ]]; then
        query="$post_data"
    fi
    
    # Remove format parameter if present
    query="${query%%&fmt=*}"
    
    # Handle different query types
    case "$query" in
        *"SELECT table_name FROM tables()"*)
            mock::questdb::list_tables_json
            ;;
        *"COUNT(*)"*)
            local table=$(echo "$query" | grep -oP 'FROM\s+\K\w+' | head -1)
            mock::questdb::count_table_json "$table"
            ;;
        *"SHOW COLUMNS"*)
            local table=$(echo "$query" | grep -oP 'FROM\s+\K\w+' | head -1)
            mock::questdb::show_columns_json "$table"
            ;;
        *"CREATE TABLE"*)
            mock::questdb::create_table_from_sql "$query"
            ;;
        *"INSERT INTO"*)
            echo '{"ddl":"INSERT","rows":1}'
            ;;
        *"DROP TABLE"*)
            local table=$(echo "$query" | grep -oP 'TABLE\s+\K\w+' | head -1)
            mock::questdb::drop_table "$table"
            ;;
        *"SELECT 1"*)
            echo '{"query":"SELECT 1","columns":[{"name":"?column?","type":"INTEGER"}],"dataset":[[1]],"count":1}'
            ;;
        *)
            # Generic SELECT response
            echo "{\"query\":\"$query\",\"columns\":[],\"dataset\":[],\"count\":0}"
            ;;
    esac
}

# List tables in JSON format
mock::questdb::list_tables_json() {
    local tables_json='{"query":"SELECT table_name FROM tables()","columns":[{"name":"table_name","type":"STRING"}],"dataset":['
    local first=true
    
    for table in "${!QUESTDB_MOCK_TABLES[@]}"; do
        if [[ "$first" == "false" ]]; then
            tables_json+=","
        fi
        tables_json+="[\"$table\"]"
        first=false
    done
    
    local count="${#QUESTDB_MOCK_TABLES[@]}"
    tables_json+="],\"count\":$count}"
    echo "$tables_json"
}

# Count table rows in JSON format
mock::questdb::count_table_json() {
    local table="$1"
    local count="${QUESTDB_MOCK_TABLE_ROWS[$table]:-0}"
    echo "{\"query\":\"SELECT COUNT(*) FROM $table\",\"columns\":[{\"name\":\"count\",\"type\":\"LONG\"}],\"dataset\":[[$count]],\"count\":1}"
}

# Show columns in JSON format
mock::questdb::show_columns_json() {
    local table="$1"
    local schema="${QUESTDB_MOCK_TABLES[$table]:-}"
    
    if [[ -z "$schema" ]]; then
        echo '{"error":"Table not found"}'
        return 1
    fi
    
    local json='{"query":"SHOW COLUMNS FROM '"$table"'","columns":[{"name":"column","type":"STRING"},{"name":"type","type":"STRING"},{"name":"indexed","type":"BOOLEAN"}],"dataset":['
    local first=true
    
    IFS=',' read -ra columns <<< "$schema"
    for col in "${columns[@]}"; do
        IFS=':' read -r name type <<< "$col"
        local indexed="false"
        [[ "$name" == "timestamp" || "$name" =~ _id$ ]] && indexed="true"
        
        if [[ "$first" == "false" ]]; then
            json+=","
        fi
        json+="[\"$name\",\"$type\",$indexed]"
        first=false
    done
    
    json+="],\"count\":${#columns[@]}}"
    echo "$json"
}

# Main psql interceptor for PostgreSQL wire protocol
psql() {
    # Log command if function exists
    if declare -f mock::log_and_verify >/dev/null 2>&1; then
        mock::log_and_verify "psql" "$@"
    fi
    
    # Parse arguments
    local host="${QUESTDB_MOCK_CONFIG[host]}"
    local port="${QUESTDB_MOCK_CONFIG[pg_port]}"
    local user="${QUESTDB_MOCK_CONFIG[username]}"
    local database="${QUESTDB_MOCK_CONFIG[database]}"
    local command=""
    local tuples_only=""
    local no_align=""
    
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
            -t|--tuples-only)
                tuples_only="true"
                shift
                ;;
            -A|--no-align)
                no_align="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check connection
    if [[ "${QUESTDB_MOCK_CONFIG[connected]}" != "true" ]]; then
        echo "psql: error: could not connect to server: Connection refused" >&2
        echo "        Is the server running on host \"$host\" and accepting" >&2
        echo "        TCP/IP connections on port $port?" >&2
        return 2
    fi
    
    # Handle different SQL commands
    case "$command" in
        "SELECT version();"*)
            echo " version "
            echo "---------"
            echo " QuestDB ${QUESTDB_MOCK_CONFIG[version]}"
            echo "(1 row)"
            ;;
        "SELECT table_name FROM tables();"*|"SELECT table_name FROM tables()"*)
            mock::questdb::list_tables_psql
            ;;
        "SHOW COLUMNS FROM"*)
            local table=$(echo "$command" | grep -oP 'FROM\s+\K\w+' | head -1)
            mock::questdb::show_columns_psql "$table"
            ;;
        "SELECT COUNT(*)"*)
            local table=$(echo "$command" | grep -oP 'FROM\s+\K\w+' | head -1)
            mock::questdb::count_table_psql "$table"
            ;;
        "CREATE TABLE"*)
            echo "CREATE TABLE"
            ;;
        "INSERT INTO"*)
            echo "INSERT 0 1"
            ;;
        "DROP TABLE"*)
            echo "DROP TABLE"
            ;;
        "SELECT 1;"*)
            if [[ "$tuples_only" == "true" ]]; then
                echo "1"
            else
                echo " ?column? "
                echo "----------"
                echo "        1"
                echo "(1 row)"
            fi
            ;;
        *)
            echo "Query executed successfully"
            ;;
    esac
    
    return 0
}

# List tables in psql format
mock::questdb::list_tables_psql() {
    echo "    table_name    "
    echo "------------------"
    for table in "${!QUESTDB_MOCK_TABLES[@]}"; do
        echo " $table"
    done
    echo "(${#QUESTDB_MOCK_TABLES[@]} rows)"
}

# Show columns in psql format
mock::questdb::show_columns_psql() {
    local table="$1"
    local schema="${QUESTDB_MOCK_TABLES[$table]:-}"
    
    if [[ -z "$schema" ]]; then
        echo "ERROR: Table '$table' does not exist" >&2
        return 1
    fi
    
    echo "    column    |   type    | indexed "
    echo "-------------+-----------+---------"
    
    IFS=',' read -ra columns <<< "$schema"
    for col in "${columns[@]}"; do
        IFS=':' read -r name type <<< "$col"
        local indexed="f"
        [[ "$name" == "timestamp" || "$name" =~ _id$ ]] && indexed="t"
        printf " %-11s | %-9s | %s\n" "$name" "$type" "$indexed"
    done
    echo "(${#columns[@]} rows)"
}

# Count table rows in psql format
mock::questdb::count_table_psql() {
    local table="$1"
    local count="${QUESTDB_MOCK_TABLE_ROWS[$table]:-0}"
    echo " count "
    echo "-------"
    printf " %5d\n" "$count"
    echo "(1 row)"
}

# Main nc interceptor for ILP (InfluxDB Line Protocol)
nc() {
    # Log command if function exists
    if declare -f mock::log_and_verify >/dev/null 2>&1; then
        mock::log_and_verify "nc" "$@"
    fi
    
    local host="localhost"
    local port=""
    local timeout=""
    local udp=""
    local original_args=("$@")
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -w)
                timeout="$2"
                shift 2
                ;;
            -u)
                udp="true"
                shift
                ;;
            -z)
                # Port scan mode - just check connectivity
                shift
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
    if [[ "$port" == "${QUESTDB_MOCK_CONFIG[ilp_port]}" ]]; then
        # Check connection
        if [[ "${QUESTDB_MOCK_CONFIG[connected]}" != "true" ]]; then
            return 1
        fi
        
        # Read ILP data from stdin
        local line_count=0
        while IFS= read -r line; do
            line_count=$((line_count + 1))
            mock::questdb::process_ilp_line "$line"
        done
        
        # Flush ILP buffer
        mock::questdb::flush_ilp_buffer
        
        # Save state immediately since we might be in a subshell due to pipe
        mock::questdb::save_state
        
        return 0
    fi
    
    # Pass through to real nc for other ports
    if command -v /usr/bin/nc &>/dev/null; then
        /usr/bin/nc "${original_args[@]}"
    else
        echo "nc: command not found" >&2
        return 127
    fi
}

# Process ILP line
mock::questdb::process_ilp_line() {
    local line="$1"
    
    # Skip empty lines
    [[ -z "$line" ]] && return
    
    # Add to buffer
    QUESTDB_MOCK_ILP_BUFFER+=("$line")
    
    
    # Parse line: measurement[,tag_set] field_set [timestamp]
    # More flexible regex to handle various ILP formats
    local measurement=""
    local tags=""
    local fields=""
    local timestamp=""
    
    # Try to parse the line
    if [[ "$line" =~ ^([^,[:space:]]+)(,[^[:space:]]+)?[[:space:]]+([^[:space:]]+.*) ]]; then
        measurement="${BASH_REMATCH[1]}"
        tags="${BASH_REMATCH[2]#,}"  # Remove leading comma
        local remainder="${BASH_REMATCH[3]}"
        
        # Check if remainder has timestamp at the end
        if [[ "$remainder" =~ ^(.+)[[:space:]]+([0-9]+)$ ]]; then
            fields="${BASH_REMATCH[1]}"
            timestamp="${BASH_REMATCH[2]}"
        else
            fields="$remainder"
            timestamp="$(date +%s)000000000"
        fi
        
        # Update metrics
        QUESTDB_MOCK_METRICS["$measurement"]="$fields"
        
        # Create table if it doesn't exist
        if [[ -z "${QUESTDB_MOCK_TABLES[$measurement]:-}" ]]; then
            QUESTDB_MOCK_TABLES["$measurement"]="timestamp:TIMESTAMP,tags:STRING,fields:STRING"
            QUESTDB_MOCK_TABLE_ROWS["$measurement"]="0"
        fi
        
        # Increment row count
        local current_count="${QUESTDB_MOCK_TABLE_ROWS[$measurement]:-0}"
        QUESTDB_MOCK_TABLE_ROWS["$measurement"]=$((current_count + 1))
    fi
}

# Flush ILP buffer
mock::questdb::flush_ilp_buffer() {
    if [[ ${#QUESTDB_MOCK_ILP_BUFFER[@]} -gt 0 ]]; then
        # Log if function exists
        if declare -f mock::log_state >/dev/null 2>&1; then
            mock::log_state "questdb" "Flushed ${#QUESTDB_MOCK_ILP_BUFFER[@]} ILP lines"
        fi
        QUESTDB_MOCK_ILP_BUFFER=()
        mock::questdb::save_state
    fi
}

# Configuration functions
mock::questdb::set_status() {
    local status="$1"
    QUESTDB_MOCK_CONFIG[status]="$status"
    
    case "$status" in
        "healthy")
            QUESTDB_MOCK_CONFIG[connected]="true"
            QUESTDB_MOCK_CONFIG[error_mode]=""
            ;;
        "unhealthy"|"stopped")
            QUESTDB_MOCK_CONFIG[connected]="false"
            ;;
        "starting")
            QUESTDB_MOCK_CONFIG[connected]="false"
            QUESTDB_MOCK_CONFIG[status]="starting"
            ;;
    esac
    
    mock::questdb::save_state
}

# Inject errors for testing
mock::questdb::inject_error() {
    local error_type="$1"
    QUESTDB_MOCK_CONFIG[error_mode]="$error_type"
    mock::questdb::save_state
}

# Create a table
mock::questdb::create_table() {
    local table="$1"
    local schema="$2"
    
    # Validate table name
    if [[ -z "$table" ]]; then
        echo "Error: Table name cannot be empty" >&2
        return 1
    fi
    
    QUESTDB_MOCK_TABLES["$table"]="$schema"
    QUESTDB_MOCK_TABLE_ROWS["$table"]="0"
    mock::questdb::save_state
    
    echo "Table '$table' created successfully"
}

# Drop a table
mock::questdb::drop_table() {
    local table="$1"
    
    if [[ -z "${QUESTDB_MOCK_TABLES[$table]:-}" ]]; then
        echo '{"error":"Table not found"}'
        return 1
    fi
    
    unset QUESTDB_MOCK_TABLES["$table"]
    unset QUESTDB_MOCK_TABLE_ROWS["$table"]
    
    # Remove table data
    for key in "${!QUESTDB_MOCK_TABLE_DATA[@]}"; do
        if [[ "$key" == "$table:"* ]]; then
            unset QUESTDB_MOCK_TABLE_DATA["$key"]
        fi
    done
    
    mock::questdb::save_state
    echo '{"ddl":"DROP"}'
}

# Insert data into table
mock::questdb::insert_data() {
    local table="$1"
    local row_data="$2"
    
    if [[ -z "${QUESTDB_MOCK_TABLES[$table]:-}" ]]; then
        echo "Table '$table' does not exist" >&2
        return 1
    fi
    
    local row_count="${QUESTDB_MOCK_TABLE_ROWS[$table]:-0}"
    QUESTDB_MOCK_TABLE_DATA["$table:$row_count"]="$row_data"
    QUESTDB_MOCK_TABLE_ROWS["$table"]=$((row_count + 1))
    
    mock::questdb::save_state
}

# Query helper for tests
mock::questdb::query() {
    local sql="$1"
    local format="${2:-json}"
    
    # Use curl mock to execute query
    curl -s "http://${QUESTDB_MOCK_CONFIG[host]}:${QUESTDB_MOCK_CONFIG[http_port]}/exec?query=$(echo "$sql" | sed 's/ /%20/g')&fmt=$format"
}

# Health check helper
mock::questdb::health_check() {
    if [[ "${QUESTDB_MOCK_CONFIG[status]}" == "healthy" && "${QUESTDB_MOCK_CONFIG[connected]}" == "true" ]]; then
        echo "OK"
        return 0
    else
        echo "FAILED"
        return 1
    fi
}

# Get table count helper
mock::questdb::get_table_count() {
    local table="$1"
    echo "${QUESTDB_MOCK_TABLE_ROWS[$table]:-0}"
}

# Export mock functions
export -f curl
export -f psql
export -f nc
export -f mock::questdb::save_state
export -f mock::questdb::load_state
export -f mock::questdb::reset
export -f mock::questdb::init_default_tables
export -f mock::questdb::handle_http_request
export -f mock::questdb::handle_status
export -f mock::questdb::handle_exec
export -f mock::questdb::list_tables_json
export -f mock::questdb::count_table_json
export -f mock::questdb::show_columns_json
export -f mock::questdb::list_tables_psql
export -f mock::questdb::show_columns_psql
export -f mock::questdb::count_table_psql
export -f mock::questdb::process_ilp_line
export -f mock::questdb::flush_ilp_buffer
export -f mock::questdb::set_status
export -f mock::questdb::inject_error
export -f mock::questdb::create_table
export -f mock::questdb::drop_table
export -f mock::questdb::insert_data
export -f mock::questdb::query
export -f mock::questdb::health_check
export -f mock::questdb::get_table_count

echo "[QUESTDB_MOCK] QuestDB mock implementation loaded"