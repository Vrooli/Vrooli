#!/usr/bin/env bash
# PostgreSQL Mock - Tier 2 (Stateful)
# 
# Provides stateful PostgreSQL mock with essential operations for testing:
# - Database connection and queries
# - Table operations (CREATE, INSERT, SELECT, UPDATE, DELETE)
# - Transaction support (BEGIN, COMMIT, ROLLBACK)
# - Basic schema management
# - Error injection for resilience testing
#
# Coverage: ~80% of common PostgreSQL use cases in 450 lines

# === Configuration ===
declare -gA POSTGRES_TABLES=()        # Table storage: table_name -> "column1|column2|..."
declare -gA POSTGRES_DATA=()          # Data storage: table_name -> "row1||row2||..."
declare -gA POSTGRES_SEQUENCES=()     # Sequence storage for auto-increment

# Debug and error modes
declare -g POSTGRES_DEBUG="${POSTGRES_DEBUG:-}"
declare -g POSTGRES_ERROR_MODE="${POSTGRES_ERROR_MODE:-}"

# Configuration
declare -g POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
declare -g POSTGRES_PORT="${POSTGRES_PORT:-5432}"
declare -g POSTGRES_USER="${POSTGRES_USER:-postgres}"
declare -g POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
declare -g POSTGRES_DATABASE="${POSTGRES_DATABASE:-test_db}"
declare -g POSTGRES_VERSION="15.0"

# Transaction state
declare -g POSTGRES_TRANSACTION_MODE=""
declare -ga POSTGRES_TRANSACTION_BACKUP=()

# === Helper Functions ===
postgres_debug() {
    [[ -n "$POSTGRES_DEBUG" ]] && echo "[MOCK:POSTGRES] $*" >&2
}

postgres_check_error() {
    case "$POSTGRES_ERROR_MODE" in
        "connection_failed")
            echo "psql: error: connection to server at \"$POSTGRES_HOST\" ($POSTGRES_HOST), port $POSTGRES_PORT failed: Connection refused" >&2
            return 1
            ;;
        "auth_failed")
            echo "psql: error: FATAL: password authentication failed for user \"$POSTGRES_USER\"" >&2
            return 1
            ;;
        "database_not_found")
            echo "psql: error: FATAL: database \"$POSTGRES_DATABASE\" does not exist" >&2
            return 1
            ;;
        "timeout")
            sleep 5
            echo "psql: error: timeout expired" >&2
            return 1
            ;;
    esac
    return 0
}

# Format output as PostgreSQL table
postgres_format_table() {
    local headers="$1"
    local rows="$2"
    
    # If no rows, show empty result
    if [[ -z "$rows" ]]; then
        echo "$headers"
        echo "----+----"
        echo "(0 rows)"
        return
    fi
    
    # Simple table format
    echo "$headers"
    local header_count
    header_count=$(echo "$headers" | tr '|' '\n' | wc -l)
    local separator=""
    for ((i=1; i<=header_count; i++)); do
        [[ -n "$separator" ]] && separator="${separator}+----"
        [[ -z "$separator" ]] && separator="----"
    done
    echo "$separator"
    
    # Print rows
    local IFS='||'
    local row_array=($rows)
    local row_count=0
    for row in "${row_array[@]}"; do
        echo "$row"
        ((row_count++))
    done
    echo "($row_count rows)"
}

# === Main PostgreSQL CLI Mock ===
psql() {
    postgres_debug "Called with: $*"
    
    # Check for errors
    postgres_check_error || return $?
    
    # Parse flags
    local database="$POSTGRES_DATABASE"
    local user="$POSTGRES_USER"
    local host="$POSTGRES_HOST"
    local port="$POSTGRES_PORT"
    local command=""
    local command_mode=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -d|--dbname) database="$2"; shift 2 ;;
            -U|--username) user="$2"; shift 2 ;;
            -h|--host) host="$2"; shift 2 ;;
            -p|--port) port="$2"; shift 2 ;;
            -c|--command) command="$2"; command_mode=true; shift 2 ;;
            -t|--tuples-only) shift ;;  # Ignore, we always show simple format
            -A|--no-align) shift ;;     # Ignore alignment options
            -q|--quiet) shift ;;        # Ignore quiet mode
            --version) 
                echo "psql (PostgreSQL) $POSTGRES_VERSION"
                return 0
                ;;
            *) shift ;;
        esac
    done
    
    # If no command provided, show error (no interactive mode)
    if [[ "$command_mode" != "true" ]] || [[ -z "$command" ]]; then
        echo "psql: Interactive mode not supported in mock" >&2
        echo "Use: psql -c \"SQL COMMAND\"" >&2
        return 1
    fi
    
    # Execute SQL command
    postgres_execute_sql "$command"
}

# Execute SQL command
postgres_execute_sql() {
    local sql="$1"
    # Convert to uppercase for parsing, but preserve original for values
    local sql_upper
    sql_upper=$(echo "$sql" | tr '[:lower:]' '[:upper:]')
    
    # Remove trailing semicolon if present
    sql="${sql%;}"
    sql_upper="${sql_upper%;}"
    
    postgres_debug "Executing SQL: $sql"
    
    # Parse SQL command
    case "$sql_upper" in
        "SELECT VERSION()"*)
            echo " version"
            echo "---------"
            echo " PostgreSQL $POSTGRES_VERSION on x86_64-pc-linux-gnu"
            echo "(1 row)"
            ;;
            
        "\\DT"|"\\D")
            # List tables
            if [[ ${#POSTGRES_TABLES[@]} -eq 0 ]]; then
                echo "No relations found."
            else
                echo " Schema | Name | Type | Owner"
                echo "--------+------+------+-------"
                for table in "${!POSTGRES_TABLES[@]}"; do
                    echo " public | $table | table | $POSTGRES_USER"
                done
                echo "(${#POSTGRES_TABLES[@]} rows)"
            fi
            ;;
            
        "BEGIN"*)
            if [[ "$POSTGRES_TRANSACTION_MODE" == "active" ]]; then
                echo "WARNING: there is already a transaction in progress"
            else
                POSTGRES_TRANSACTION_MODE="active"
                # Backup current state
                POSTGRES_TRANSACTION_BACKUP=()
                for table in "${!POSTGRES_DATA[@]}"; do
                    POSTGRES_TRANSACTION_BACKUP+=("${table}:${POSTGRES_DATA[$table]}")
                done
                echo "BEGIN"
            fi
            ;;
            
        "COMMIT"*)
            if [[ "$POSTGRES_TRANSACTION_MODE" != "active" ]]; then
                echo "WARNING: there is no transaction in progress"
            else
                POSTGRES_TRANSACTION_MODE=""
                POSTGRES_TRANSACTION_BACKUP=()
                echo "COMMIT"
            fi
            ;;
            
        "ROLLBACK"*)
            if [[ "$POSTGRES_TRANSACTION_MODE" != "active" ]]; then
                echo "WARNING: there is no transaction in progress"
            else
                # Restore from backup
                POSTGRES_DATA=()
                for backup_item in "${POSTGRES_TRANSACTION_BACKUP[@]}"; do
                    local table="${backup_item%%:*}"
                    local data="${backup_item#*:}"
                    POSTGRES_DATA[$table]="$data"
                done
                POSTGRES_TRANSACTION_MODE=""
                POSTGRES_TRANSACTION_BACKUP=()
                echo "ROLLBACK"
            fi
            ;;
            
        "CREATE TABLE"*)
            # Parse: CREATE TABLE table_name (columns)
            local table_name
            table_name=$(echo "$sql" | sed -n 's/.*CREATE TABLE \([^ (]*\).*/\1/ip')
            table_name="${table_name//\"/}"  # Remove quotes
            
            if [[ -n "${POSTGRES_TABLES[$table_name]:-}" ]]; then
                echo "ERROR: relation \"$table_name\" already exists"
                return 1
            fi
            
            # Extract columns (simplified - just store column names)
            local columns
            columns=$(echo "$sql" | sed -n 's/.*(\(.*\)).*/\1/p')
            # Extract just column names (ignore types)
            columns=$(echo "$columns" | sed 's/[A-Z ]*//g' | tr ',' '|')
            
            POSTGRES_TABLES[$table_name]="$columns"
            POSTGRES_DATA[$table_name]=""
            postgres_debug "Created table: $table_name with columns: $columns"
            echo "CREATE TABLE"
            ;;
            
        "DROP TABLE"*)
            local table_name
            table_name=$(echo "$sql" | sed -n 's/.*DROP TABLE \([^ ;]*\).*/\1/ip')
            table_name="${table_name//\"/}"
            
            if [[ -z "${POSTGRES_TABLES[$table_name]:-}" ]]; then
                if [[ ! "$sql_upper" =~ "IF EXISTS" ]]; then
                    echo "ERROR: table \"$table_name\" does not exist"
                    return 1
                fi
            else
                unset "POSTGRES_TABLES[$table_name]"
                unset "POSTGRES_DATA[$table_name]"
                unset "POSTGRES_SEQUENCES[${table_name}_id_seq]"
            fi
            echo "DROP TABLE"
            ;;
            
        "INSERT INTO"*)
            # Parse: INSERT INTO table_name (columns) VALUES (values)
            local table_name
            table_name=$(echo "$sql" | sed -n 's/.*INSERT INTO \([^ (]*\).*/\1/ip')
            table_name="${table_name//\"/}"
            
            if [[ -z "${POSTGRES_TABLES[$table_name]:-}" ]]; then
                echo "ERROR: relation \"$table_name\" does not exist"
                return 1
            fi
            
            # Extract values (simplified - just store as is)
            local values
            values=$(echo "$sql" | sed -n "s/.*VALUES *(\(.*\))/\1/ip")
            values="${values//\'/}"  # Remove quotes
            values=$(echo "$values" | sed 's/, /|/g' | sed 's/,/|/g')  # Better comma handling
            
            # Add to data
            if [[ -z "${POSTGRES_DATA[$table_name]}" ]]; then
                POSTGRES_DATA[$table_name]="$values"
            else
                POSTGRES_DATA[$table_name]="${POSTGRES_DATA[$table_name]}||$values"
            fi
            
            # Handle RETURNING clause
            if [[ "$sql_upper" =~ "RETURNING" ]]; then
                # Get next ID from sequence
                local seq_name="${table_name}_id_seq"
                local next_id="${POSTGRES_SEQUENCES[$seq_name]:-1}"
                POSTGRES_SEQUENCES[$seq_name]=$((next_id + 1))
                
                echo " id"
                echo "----"
                echo " $next_id"
                echo "(1 row)"
            else
                echo "INSERT 0 1"
            fi
            ;;
            
        "SELECT"*)
            # Simplified SELECT parsing
            local table_name
            if [[ "$sql_upper" =~ "FROM" ]]; then
                table_name=$(echo "$sql" | sed -n 's/.*FROM \([^ ;]*\).*/\1/ip')
                table_name="${table_name//\"/}"
                
                if [[ -z "${POSTGRES_TABLES[$table_name]:-}" ]]; then
                    echo "ERROR: relation \"$table_name\" does not exist"
                    return 1
                fi
                
                # Get columns
                local columns="${POSTGRES_TABLES[$table_name]}"
                local data="${POSTGRES_DATA[$table_name]}"
                
                # Handle COUNT(*)
                if [[ "$sql_upper" =~ "COUNT(\*)" ]] || [[ "$sql_upper" =~ "COUNT(*)" ]]; then
                    local count=0
                    if [[ -n "$data" ]]; then
                        local IFS='||'
                        local rows=($data)
                        count=${#rows[@]}
                    fi
                    echo " count"
                    echo "-------"
                    echo " $count"
                    echo "(1 row)"
                else
                    # Return all data
                    postgres_format_table "$columns" "$data"
                fi
            else
                # Simple SELECT without FROM
                echo " ?column?"
                echo "----------"
                echo " 1"
                echo "(1 row)"
            fi
            ;;
            
        "UPDATE"*)
            # Parse: UPDATE table_name SET column = value WHERE condition
            local table_name
            table_name=$(echo "$sql" | sed -n 's/.*UPDATE \([^ ]*\).*/\1/ip')
            table_name="${table_name//\"/}"
            
            if [[ -z "${POSTGRES_TABLES[$table_name]:-}" ]]; then
                echo "ERROR: relation \"$table_name\" does not exist"
                return 1
            fi
            
            # For mock, just report success
            echo "UPDATE 1"
            ;;
            
        "DELETE FROM"*)
            local table_name
            table_name=$(echo "$sql" | sed -n 's/.*DELETE FROM \([^ ;]*\).*/\1/ip')
            table_name="${table_name//\"/}"
            
            if [[ -z "${POSTGRES_TABLES[$table_name]:-}" ]]; then
                echo "ERROR: relation \"$table_name\" does not exist"
                return 1
            fi
            
            # Clear table data (simplified - doesn't handle WHERE)
            POSTGRES_DATA[$table_name]=""
            echo "DELETE 1"
            ;;
            
        "\\Q"|"EXIT"|"QUIT")
            echo "\\q"
            return 0
            ;;
            
        *)
            # Default response for unknown commands
            echo "Mock: Command executed successfully"
            ;;
    esac
}

# === Convention-based Test Functions ===
test_postgres_connection() {
    postgres_debug "Testing connection..."
    
    local result
    result=$(psql -c "SELECT version()" 2>&1)
    
    if [[ "$result" =~ "PostgreSQL" ]]; then
        postgres_debug "Connection test passed"
        return 0
    else
        postgres_debug "Connection test failed: $result"
        return 1
    fi
}

test_postgres_health() {
    postgres_debug "Testing health..."
    
    # Test connection
    test_postgres_connection || return 1
    
    # Test basic operations
    psql -c "CREATE TABLE health_test (id INT, value TEXT)" >/dev/null 2>&1 || return 1
    psql -c "INSERT INTO health_test VALUES (1, 'healthy')" >/dev/null 2>&1 || return 1
    local result
    result=$(psql -c "SELECT COUNT(*) FROM health_test" 2>/dev/null)
    psql -c "DROP TABLE health_test" >/dev/null 2>&1
    
    if [[ "$result" =~ "1" ]]; then
        postgres_debug "Health test passed"
        return 0
    else
        postgres_debug "Health test failed"
        return 1
    fi
}

test_postgres_basic() {
    postgres_debug "Testing basic operations..."
    
    # Test CREATE TABLE
    psql -c "CREATE TABLE test_table (id INT, name TEXT)" >/dev/null 2>&1 || return 1
    
    # Test INSERT
    psql -c "INSERT INTO test_table VALUES (1, 'test')" >/dev/null 2>&1 || return 1
    
    # Test SELECT
    local result
    result=$(psql -c "SELECT COUNT(*) FROM test_table" 2>/dev/null)
    [[ "$result" =~ "1" ]] || return 1
    
    # Test UPDATE
    psql -c "UPDATE test_table SET name = 'updated'" >/dev/null 2>&1 || return 1
    
    # Test DELETE
    psql -c "DELETE FROM test_table" >/dev/null 2>&1 || return 1
    
    # Test DROP
    psql -c "DROP TABLE test_table" >/dev/null 2>&1 || return 1
    
    postgres_debug "Basic test passed"
    return 0
}

# === State Management ===
postgres_mock_reset() {
    postgres_debug "Resetting mock state"
    POSTGRES_TABLES=()
    POSTGRES_DATA=()
    POSTGRES_SEQUENCES=()
    POSTGRES_TRANSACTION_MODE=""
    POSTGRES_TRANSACTION_BACKUP=()
    POSTGRES_ERROR_MODE=""
    
    # Create default tables
    POSTGRES_TABLES["users"]="id|name|email"
    POSTGRES_DATA["users"]="1|admin|admin@test.com"
    POSTGRES_SEQUENCES["users_id_seq"]=2
}

postgres_mock_set_error() {
    POSTGRES_ERROR_MODE="$1"
    postgres_debug "Set error mode: $1"
}

postgres_mock_dump_state() {
    echo "=== PostgreSQL Mock State ==="
    echo "Database: $POSTGRES_DATABASE"
    echo "Tables: ${#POSTGRES_TABLES[@]}"
    for table in "${!POSTGRES_TABLES[@]}"; do
        echo "  $table: ${POSTGRES_TABLES[$table]}"
        local data="${POSTGRES_DATA[$table]:-}"
        if [[ -n "$data" ]]; then
            local IFS='||'
            local rows=($data)
            echo "    Rows: ${#rows[@]}"
        else
            echo "    Rows: 0"
        fi
    done
    echo "Transaction: ${POSTGRES_TRANSACTION_MODE:-inactive}"
    echo "Error Mode: ${POSTGRES_ERROR_MODE:-none}"
    echo "============================="
}

# Additional helper for creating test data
postgres_mock_create_table() {
    local table_name="$1"
    local columns="$2"  # Format: "col1|col2|col3"
    POSTGRES_TABLES[$table_name]="$columns"
    POSTGRES_DATA[$table_name]=""
    postgres_debug "Created table: $table_name"
}

postgres_mock_insert_data() {
    local table_name="$1"
    local data="$2"  # Format: "val1|val2|val3"
    if [[ -z "${POSTGRES_DATA[$table_name]}" ]]; then
        POSTGRES_DATA[$table_name]="$data"
    else
        POSTGRES_DATA[$table_name]="${POSTGRES_DATA[$table_name]}||$data"
    fi
    postgres_debug "Inserted data into $table_name"
}

# === Export Functions ===
export -f psql
export -f test_postgres_connection
export -f test_postgres_health
export -f test_postgres_basic
export -f postgres_mock_reset
export -f postgres_mock_set_error
export -f postgres_mock_dump_state
export -f postgres_mock_create_table
export -f postgres_mock_insert_data
export -f postgres_debug
export -f postgres_check_error
export -f postgres_execute_sql
export -f postgres_format_table

# Initialize with default state
postgres_mock_reset
postgres_debug "PostgreSQL Tier 2 mock initialized"