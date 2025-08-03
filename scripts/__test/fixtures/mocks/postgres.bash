#!/usr/bin/env bash
# PostgreSQL Resource Mock Implementation
# Provides realistic mock responses for PostgreSQL database service

# Prevent duplicate loading
if [[ "${POSTGRES_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export POSTGRES_MOCK_LOADED="true"

#######################################
# Setup PostgreSQL mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::postgres::setup() {
    local state="${1:-healthy}"
    
    # Configure PostgreSQL-specific environment
    export POSTGRES_PORT="${POSTGRES_PORT:-5432}"
    export POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
    export POSTGRES_CONTAINER_NAME="${TEST_NAMESPACE}_postgres"
    export POSTGRES_USER="${POSTGRES_USER:-postgres}"
    export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-password}"
    export POSTGRES_DB="${POSTGRES_DB:-testdb}"
    export DATABASE_URL="postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$POSTGRES_CONTAINER_NAME" "$state"
    
    # Configure psql mock based on state
    case "$state" in
        "healthy")
            mock::postgres::setup_healthy_state
            ;;
        "unhealthy")
            mock::postgres::setup_unhealthy_state
            ;;
        "installing")
            mock::postgres::setup_installing_state
            ;;
        "stopped")
            mock::postgres::setup_stopped_state
            ;;
        *)
            echo "[POSTGRES_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[POSTGRES_MOCK] PostgreSQL mock configured with state: $state"
}

#######################################
# Setup healthy PostgreSQL state
#######################################
mock::postgres::setup_healthy_state() {
    # Set container logs
    local log_file="${MOCK_RESPONSES_DIR}/postgres_logs.txt"
    cat > "$log_file" << 'EOF'
PostgreSQL init process complete; ready for start up.
LOG:  database system is ready to accept connections
EOF
    
    # Set psql response file
    local psql_response="${MOCK_RESPONSES_DIR}/psql_response.txt"
    cat > "$psql_response" << 'EOF'
 version 
---------
 PostgreSQL 15.3 on x86_64-pc-linux-gnu, compiled by gcc (GCC) 11.3.0, 64-bit
(1 row)
EOF
}

#######################################
# Setup unhealthy PostgreSQL state
#######################################
mock::postgres::setup_unhealthy_state() {
    # Set error logs
    local log_file="${MOCK_RESPONSES_DIR}/postgres_logs.txt"
    cat > "$log_file" << 'EOF'
FATAL:  could not access file "pg_stat": No such file or directory
LOG:  database system is shut down
EOF
}

#######################################
# Setup installing PostgreSQL state
#######################################
mock::postgres::setup_installing_state() {
    # Set initialization logs
    local log_file="${MOCK_RESPONSES_DIR}/postgres_logs.txt"
    cat > "$log_file" << 'EOF'
The files belonging to this database system will be owned by user "postgres".
This user must also own the server process.
Initializing database...
Creating initial tables...
Progress: 60%
EOF
}

#######################################
# Setup stopped PostgreSQL state
#######################################
mock::postgres::setup_stopped_state() {
    # No logs when stopped
    local log_file="${MOCK_RESPONSES_DIR}/postgres_logs.txt"
    echo "" > "$log_file"
}

#######################################
# Mock psql command
#######################################
psql() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "psql $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Parse arguments
    local host="$POSTGRES_HOST"
    local port="$POSTGRES_PORT"
    local user="$POSTGRES_USER"
    local database="$POSTGRES_DB"
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
    local container_state=$(docker inspect "$POSTGRES_CONTAINER_NAME" --format '{{.State.Status}}' 2>/dev/null || echo "stopped")
    
    if [[ "$container_state" != "running" ]]; then
        echo "psql: error: could not connect to server: Connection refused" >&2
        return 2
    fi
    
    # Handle different commands
    case "$command" in
        "SELECT version();")
            echo " version "
            echo "---------"
            echo " PostgreSQL 15.3 on x86_64-pc-linux-gnu, compiled by gcc (GCC) 11.3.0, 64-bit"
            echo "(1 row)"
            ;;
        "\\l"|"\\list")
            echo "                                                 List of databases"
            echo "   Name    |  Owner   | Encoding |  Collate   |   Ctype    | ICU Locale | Locale Provider |   Access privileges   "
            echo "-----------+----------+----------+------------+------------+------------+-----------------+-----------------------"
            echo " postgres  | postgres | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | "
            echo " template0 | postgres | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | =c/postgres          +"
            echo "           |          |          |            |            |            |                 | postgres=CTc/postgres"
            echo " template1 | postgres | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | =c/postgres          +"
            echo "           |          |          |            |            |            |                 | postgres=CTc/postgres"
            echo " $database | postgres | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | "
            echo "(4 rows)"
            ;;
        "\\dt"|"\\dt+")
            echo "                    List of relations"
            echo " Schema |    Name    | Type  |  Owner   | Persistence"
            echo "--------+------------+-------+----------+-------------"
            echo " public | users      | table | postgres | permanent"
            echo " public | products   | table | postgres | permanent"
            echo " public | orders     | table | postgres | permanent"
            echo "(3 rows)"
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
# Mock pg_isready command
#######################################
pg_isready() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "pg_isready $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local host="$POSTGRES_HOST"
    local port="$POSTGRES_PORT"
    
    # Parse arguments
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
            *)
                shift
                ;;
        esac
    done
    
    # Check container state
    local container_state=$(docker inspect "$POSTGRES_CONTAINER_NAME" --format '{{.State.Status}}' 2>/dev/null || echo "stopped")
    
    if [[ "$container_state" == "running" ]]; then
        echo "$host:$port - accepting connections"
        return 0
    else
        echo "$host:$port - no response"
        return 2
    fi
}

#######################################
# Mock PostgreSQL-specific operations
#######################################

# Mock database creation
mock::postgres::create_database() {
    local db_name="$1"
    
    echo "CREATE DATABASE"
    echo "Database '$db_name' created successfully"
}

# Mock table creation
mock::postgres::create_table() {
    local table_name="$1"
    
    echo "CREATE TABLE"
    echo "Table '$table_name' created successfully"
}

# Mock data insertion
mock::postgres::insert_data() {
    local table="$1"
    local count="${2:-1}"
    
    echo "INSERT 0 $count"
    echo "$count row(s) inserted into '$table'"
}

# Mock backup operation
mock::postgres::backup() {
    local db_name="$1"
    local backup_file="${2:-backup.sql}"
    
    echo "-- PostgreSQL database dump"
    echo "-- Dumped from database version 15.3"
    echo "-- Dumped by pg_dump version 15.3"
    echo ""
    echo "Database '$db_name' backed up to '$backup_file'"
}

#######################################
# Export mock functions
#######################################
export -f psql pg_isready
export -f mock::postgres::setup
export -f mock::postgres::setup_healthy_state
export -f mock::postgres::setup_unhealthy_state
export -f mock::postgres::setup_installing_state
export -f mock::postgres::setup_stopped_state
export -f mock::postgres::create_database
export -f mock::postgres::create_table
export -f mock::postgres::insert_data
export -f mock::postgres::backup

echo "[POSTGRES_MOCK] PostgreSQL mock implementation loaded"