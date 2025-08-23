#!/usr/bin/env bash
# PostgreSQL Mock Implementation
# 
# Provides comprehensive mock for PostgreSQL operations including:
# - psql command interception
# - pg_isready connection testing
# - pg_dump backup operations
# - createdb/dropdb database management
# - PostgreSQL server state simulation
# - Connection management with realistic error conditions
#
# This mock follows the same standards as other updated mocks with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions

# Prevent duplicate loading
[[ -n "${POSTGRES_MOCK_LOADED:-}" ]] && return 0
declare -g POSTGRES_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"

# Global configuration
declare -g POSTGRES_MOCK_STATE_DIR="${POSTGRES_MOCK_STATE_DIR:-/tmp/postgres-mock-state}"
declare -g POSTGRES_MOCK_DEBUG="${POSTGRES_MOCK_DEBUG:-}"

# Global state arrays
declare -gA POSTGRES_MOCK_CONFIG=(          # PostgreSQL configuration
    [host]="localhost"
    [port]="5432"
    [user]="postgres"
    [password]="password"
    [database]="testdb"
    [connected]="true"
    [version]="15.3"
    [server_status]="running"
    [error_mode]=""
    [container_name]="postgres_test"
    [max_connections]="100"
    [shared_buffers]="128MB"
)

declare -gA POSTGRES_MOCK_DATABASES=()      # Database existence tracking
declare -gA POSTGRES_MOCK_TABLES=()         # Table existence tracking
declare -gA POSTGRES_MOCK_QUERY_RESULTS=()  # Custom query results

# Initialize state directory
mkdir -p "$POSTGRES_MOCK_STATE_DIR"

# State persistence functions
mock::postgres::save_state() {
    local state_file="$POSTGRES_MOCK_STATE_DIR/postgres-state.sh"
    {
        echo "# PostgreSQL mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p POSTGRES_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA POSTGRES_MOCK_CONFIG=()"
        declare -p POSTGRES_MOCK_DATABASES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA POSTGRES_MOCK_DATABASES=()"
        declare -p POSTGRES_MOCK_TABLES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA POSTGRES_MOCK_TABLES=()"
        declare -p POSTGRES_MOCK_QUERY_RESULTS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA POSTGRES_MOCK_QUERY_RESULTS=()"
    } > "$state_file"
    
    mock::log_state "postgres" "Saved PostgreSQL state to $state_file"
}

mock::postgres::load_state() {
    local state_file="$POSTGRES_MOCK_STATE_DIR/postgres-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        mock::log_state "postgres" "Loaded PostgreSQL state from $state_file"
    fi
}

# Automatically load state when sourced
mock::postgres::load_state

# Main psql interceptor
psql() {
    mock::log_and_verify "psql" "$@"
    
    # Always reload state at the beginning to handle BATS subshells
    mock::postgres::load_state
    
    # Check if PostgreSQL is connected
    if [[ "${POSTGRES_MOCK_CONFIG[connected]}" != "true" ]]; then
        echo "psql: error: could not connect to server: Connection refused" >&2
        echo "Is the server running on host \"${POSTGRES_MOCK_CONFIG[host]}\" (${POSTGRES_MOCK_CONFIG[host]}) and accepting" >&2
        echo "TCP/IP connections on port ${POSTGRES_MOCK_CONFIG[port]}?" >&2
        mock::postgres::save_state
        return 2
    fi
    
    # Check for error injection
    if [[ -n "${POSTGRES_MOCK_CONFIG[error_mode]}" ]]; then
        case "${POSTGRES_MOCK_CONFIG[error_mode]}" in
            "connection_timeout")
                echo "psql: error: connection to server timed out" >&2
                mock::postgres::save_state
                return 2
                ;;
            "auth_failed")
                echo "psql: error: FATAL: password authentication failed for user \"${POSTGRES_MOCK_CONFIG[user]}\"" >&2
                mock::postgres::save_state
                return 2
                ;;
            "database_not_exist")
                echo "psql: error: FATAL: database \"${POSTGRES_MOCK_CONFIG[database]}\" does not exist" >&2
                mock::postgres::save_state
                return 2
                ;;
            "too_many_connections")
                echo "psql: error: FATAL: sorry, too many clients already" >&2
                mock::postgres::save_state
                return 2
                ;;
        esac
    fi
    
    # Parse command line arguments
    local host="${POSTGRES_MOCK_CONFIG[host]}"
    local port="${POSTGRES_MOCK_CONFIG[port]}"
    local user="${POSTGRES_MOCK_CONFIG[user]}"
    local database="${POSTGRES_MOCK_CONFIG[database]}"
    local command=""
    local file=""
    local output_file=""
    local quiet_mode=false
    local tuples_only=false
    local no_password=false
    local interactive=true
    
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
                interactive=false
                shift 2
                ;;
            -f|--file)
                file="$2"
                interactive=false
                shift 2
                ;;
            -o|--output)
                output_file="$2"
                shift 2
                ;;
            -q|--quiet)
                quiet_mode=true
                shift
                ;;
            -t|--tuples-only)
                tuples_only=true
                shift
                ;;
            -W|--password)
                # Password prompt simulation (ignored in mock)
                shift
                ;;
            -w|--no-password)
                no_password=true
                shift
                ;;
            --help)
                mock::postgres::show_psql_help
                mock::postgres::save_state
                return 0
                ;;
            --version)
                echo "psql (PostgreSQL) ${POSTGRES_MOCK_CONFIG[version]}"
                mock::postgres::save_state
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Handle interactive mode (not supported in mock)
    if [[ "$interactive" == "true" ]]; then
        echo "Mock psql: Interactive mode not supported in tests" >&2
        mock::postgres::save_state
        return 1
    fi
    
    # Handle file input
    if [[ -n "$file" ]]; then
        if [[ ! -f "$file" ]]; then
            echo "psql: error: $file: No such file or directory" >&2
            mock::postgres::save_state
            return 2
        fi
        # Simulate executing SQL file
        if [[ "$quiet_mode" != "true" ]]; then
            echo "-- Executing SQL file: $file"
        fi
        mock::postgres::save_state
        return 0
    fi
    
    # Execute the SQL command - call directly to preserve state changes
    if [[ -n "$output_file" ]]; then
        # Redirect output to file
        mock::postgres::execute_sql "$command" "$quiet_mode" "$tuples_only" > "$output_file"
        local exit_code=$?
    else
        # Output to stdout
        mock::postgres::execute_sql "$command" "$quiet_mode" "$tuples_only"
        local exit_code=$?
    fi
    
    mock::postgres::save_state
    return $exit_code
}

# Execute SQL commands
mock::postgres::execute_sql() {
    local sql="$1"
    local quiet_mode="${2:-false}"
    local tuples_only="${3:-false}"
    
    # Handle different SQL commands
    case "$sql" in
        "SELECT version();"|"SELECT version();")
            if [[ "$tuples_only" == "true" ]]; then
                echo "PostgreSQL ${POSTGRES_MOCK_CONFIG[version]} on x86_64-pc-linux-gnu, compiled by gcc (GCC) 11.3.0, 64-bit"
            else
                echo "                                                          version"
                echo "--------------------------------------------------------------------------------------------------------------------------"
                echo " PostgreSQL ${POSTGRES_MOCK_CONFIG[version]} on x86_64-pc-linux-gnu, compiled by gcc (GCC) 11.3.0, 64-bit"
                echo "(1 row)"
            fi
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
            echo " ${POSTGRES_MOCK_CONFIG[database]} | postgres | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | "
            
            # Add custom databases
            for db in "${!POSTGRES_MOCK_DATABASES[@]}"; do
                echo " $db | postgres | UTF8     | en_US.utf8 | en_US.utf8 |            | libc            | "
            done
            
            local db_count=$((5 + ${#POSTGRES_MOCK_DATABASES[@]}))
            echo "($db_count rows)"
            ;;
        "\\dt"|"\\dt+")
            if [[ ${#POSTGRES_MOCK_TABLES[@]} -eq 0 ]]; then
                echo "Did not find any relations."
            else
                echo "                    List of relations"
                echo " Schema |    Name    | Type  |  Owner   | Persistence"
                echo "--------+------------+-------+----------+-------------"
                for table in "${!POSTGRES_MOCK_TABLES[@]}"; do
                    echo " public | $table | table | postgres | permanent"
                done
                echo "(${#POSTGRES_MOCK_TABLES[@]} rows)"
            fi
            ;;
        "\\c "*|"\\connect "*)
            local new_db
            new_db=$(echo "$sql" | sed 's/\\c\|\\connect//' | xargs)
            if [[ -n "$new_db" ]]; then
                POSTGRES_MOCK_CONFIG[database]="$new_db"
                echo "You are now connected to database \"$new_db\" as user \"${POSTGRES_MOCK_CONFIG[user]}\"."
            fi
            ;;
        "SELECT 1;"|"SELECT 1;")
            if [[ "$tuples_only" == "true" ]]; then
                echo "1"
            else
                echo " ?column? "
                echo "----------"
                echo "        1"
                echo "(1 row)"
            fi
            ;;
        "\\q"|"\\quit")
            return 0
            ;;
        "")
            # Empty command
            return 0
            ;;
        *)
            # Check for custom query results first
            if [[ -n "${POSTGRES_MOCK_QUERY_RESULTS[$sql]:-}" ]]; then
                echo "${POSTGRES_MOCK_QUERY_RESULTS[$sql]}"
                return 0
            fi
            
            # Handle CREATE DATABASE (case insensitive)
            if [[ "$sql" =~ ^[Cc][Rr][Ee][Aa][Tt][Ee][[:space:]]+[Dd][Aa][Tt][Aa][Bb][Aa][Ss][Ee][[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
                local db_name="${BASH_REMATCH[1]}"
                POSTGRES_MOCK_DATABASES["$db_name"]="true"
                echo "CREATE DATABASE"
                return 0
            fi
            
            # Handle DROP DATABASE (case insensitive)
            if [[ "$sql" =~ ^[Dd][Rr][Oo][Pp][[:space:]]+[Dd][Aa][Tt][Aa][Bb][Aa][Ss][Ee][[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
                local db_name="${BASH_REMATCH[1]}"
                unset POSTGRES_MOCK_DATABASES["$db_name"]
                echo "DROP DATABASE"
                return 0
            fi
            
            # Handle CREATE TABLE (case insensitive)
            if [[ "$sql" =~ ^[Cc][Rr][Ee][Aa][Tt][Ee][[:space:]]+[Tt][Aa][Bb][Ll][Ee][[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
                local table_name="${BASH_REMATCH[1]}"
                POSTGRES_MOCK_TABLES["$table_name"]="true"
                echo "CREATE TABLE"
                return 0
            fi
            
            # Handle DROP TABLE (case insensitive)
            if [[ "$sql" =~ ^[Dd][Rr][Oo][Pp][[:space:]]+[Tt][Aa][Bb][Ll][Ee][[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
                local table_name="${BASH_REMATCH[1]}"
                unset POSTGRES_MOCK_TABLES["$table_name"]
                echo "DROP TABLE"
                return 0
            fi
            
            # Handle INSERT statements (case insensitive)
            if [[ "$sql" =~ ^[Ii][Nn][Ss][Ee][Rr][Tt][[:space:]]+[Ii][Nn][Tt][Oo] ]]; then
                echo "INSERT 0 1"
                return 0
            fi
            
            # Handle UPDATE statements (case insensitive)
            if [[ "$sql" =~ ^[Uu][Pp][Dd][Aa][Tt][Ee] ]]; then
                echo "UPDATE 1"
                return 0
            fi
            
            # Handle DELETE statements (case insensitive)
            if [[ "$sql" =~ ^[Dd][Ee][Ll][Ee][Tt][Ee][[:space:]]+[Ff][Rr][Oo][Mm] ]]; then
                echo "DELETE 1"
                return 0
            fi
            
            # Handle SELECT statements (case insensitive)
            if [[ "$sql" =~ ^[Ss][Ee][Ll][Ee][Cc][Tt] ]]; then
                echo "(No rows)"
                return 0
            fi
            
            # Generic success response for other SQL
            if [[ "$quiet_mode" != "true" ]]; then
                echo "Command executed successfully"
            fi
            ;;
    esac
    
    return 0
}

# Show psql help
mock::postgres::show_psql_help() {
    cat << 'EOF'
psql is the PostgreSQL interactive terminal.

Usage:
  psql [OPTION]... [DBNAME [USERNAME]]

General options:
  -c, --command=COMMAND    run only single command (SQL or internal) and exit
  -d, --dbname=DBNAME      database name to connect to (default: "postgres")
  -f, --file=FILENAME      execute commands from file, then exit
  -l, --list               list available databases, then exit
  -v, --set=, --variable=NAME=VALUE
                           set psql variable NAME to VALUE
  -V, --version            output version information, then exit
  -X, --no-psqlrc          do not read startup file (~/.psqlrc)
  -1, --single-transaction execute as a single transaction (if non-interactive)
  -?, --help[=options]     show this help, then exit

Input and output options:
  -a, --echo-all           echo all input from script
  -b, --echo-errors        echo failed commands
  -e, --echo-queries       echo commands sent to server
  -E, --echo-hidden        display queries that internal commands generate
  -L, --log-file=FILENAME  send session log to file
  -n, --no-readline        disable enhanced command line editing (readline)
  -o, --output=FILENAME    send query results to file (or |pipe)
  -q, --quiet              run quietly (no messages, only query output)
  -s, --single-step        single-step mode (confirm each query)
  -S, --single-line        single-line mode (end of line terminates SQL command)

Output format options:
  -A, --no-align           unaligned table output mode
  -F, --field-separator=STRING
                           field separator for unaligned output (default: "|")
  -H, --html               HTML table output mode
  -P, --pset=VAR[=ARG]     set printing option VAR to ARG (see \pset command)
  -R, --record-separator=STRING
                           record separator for unaligned output (default: newline)
  -t, --tuples-only        print rows only
  -T, --table-attr=TEXT    set HTML table tag attributes (e.g., width, border)
  -x, --expanded           turn on expanded table output
  -z, --field-separator-zero
                           set field separator for unaligned output to zero byte
  -0, --record-separator-zero
                           set record separator for unaligned output to zero byte

Connection options:
  -h, --host=HOSTNAME      database server host or socket directory (default: "local socket")
  -p, --port=PORT          database server port (default: "5432")
  -U, --username=USERNAME  database user name (default: "postgres")
  -w, --no-password        never prompt for password
  -W, --password           force password prompt (should happen automatically)
EOF
}

# Mock pg_isready command
pg_isready() {
    mock::log_and_verify "pg_isready" "$@"
    
    # Always reload state at the beginning to handle BATS subshells
    mock::postgres::load_state
    
    local host="${POSTGRES_MOCK_CONFIG[host]}"
    local port="${POSTGRES_MOCK_CONFIG[port]}"
    local user="${POSTGRES_MOCK_CONFIG[user]}"
    local database="${POSTGRES_MOCK_CONFIG[database]}"
    local timeout="3"
    local quiet=false
    
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
            -U|--username)
                user="$2"
                shift 2
                ;;
            -d|--dbname)
                database="$2"
                shift 2
                ;;
            -t|--timeout)
                timeout="$2"
                shift 2
                ;;
            -q|--quiet)
                quiet=true
                shift
                ;;
            --help)
                echo "pg_isready tests whether a PostgreSQL server is ready to accept connections."
                echo ""
                echo "Usage: pg_isready [OPTION]..."
                echo ""
                echo "Options:"
                echo "  -d, --dbname=DBNAME      database name"
                echo "  -h, --host=HOSTNAME      database server host or socket directory"
                echo "  -p, --port=PORT          database server port"
                echo "  -q, --quiet              run quietly"
                echo "  -t, --timeout=SECS       seconds to wait when attempting connection"
                echo "  -U, --username=USERNAME  user name to connect as"
                echo "  -V, --version            output version information, then exit"
                echo "  -?, --help               show this help, then exit"
                mock::postgres::save_state
                return 0
                ;;
            --version)
                echo "pg_isready (PostgreSQL) ${POSTGRES_MOCK_CONFIG[version]}"
                mock::postgres::save_state
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check connection status
    if [[ "${POSTGRES_MOCK_CONFIG[connected]}" == "true" && "${POSTGRES_MOCK_CONFIG[server_status]}" == "running" ]]; then
        if [[ "$quiet" != "true" ]]; then
            echo "$host:$port - accepting connections"
        fi
        mock::postgres::save_state
        return 0
    else
        if [[ "$quiet" != "true" ]]; then
            echo "$host:$port - no response"
        fi
        mock::postgres::save_state
        return 2
    fi
}

# Mock pg_dump command  
pg_dump() {
    mock::log_and_verify "pg_dump" "$@"
    
    # Always reload state at the beginning to handle BATS subshells
    mock::postgres::load_state
    
    # Check for error injection before processing
    if [[ -n "${POSTGRES_MOCK_CONFIG[error_mode]}" ]]; then
        case "${POSTGRES_MOCK_CONFIG[error_mode]}" in
            "connection_timeout"|"auth_failed"|"database_not_exist"|"too_many_connections")
                echo "pg_dump: error: could not connect to database: Connection refused" >&2
                mock::postgres::save_state
                return 1
                ;;
        esac
    fi
    
    local host="${POSTGRES_MOCK_CONFIG[host]}"
    local port="${POSTGRES_MOCK_CONFIG[port]}"
    local user="${POSTGRES_MOCK_CONFIG[user]}"
    local database=""
    local output_file=""
    local format="plain"
    local schema_only=false
    local data_only=false
    
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
            -U|--username)
                user="$2"
                shift 2
                ;;
            -d|--dbname)
                database="$2"
                shift 2
                ;;
            -f|--file)
                output_file="$2"
                shift 2
                ;;
            -F|--format)
                format="$2"
                shift 2
                ;;
            -s|--schema-only)
                schema_only=true
                shift
                ;;
            -a|--data-only)
                data_only=true
                shift
                ;;
            --help)
                echo "pg_dump dumps a database as a text file or to other formats."
                echo ""
                echo "Usage: pg_dump [OPTION]... [DBNAME]"
                echo ""
                echo "General options:"
                echo "  -f, --file=FILENAME          output file or directory name"
                echo "  -F, --format=c|d|t|p         output file format (custom, directory, tar, plain text (default))"
                echo "  -j, --jobs=NUM               use this many parallel jobs to dump"
                echo "  -v, --verbose                verbose mode"
                echo "  -V, --version                output version information, then exit"
                echo "  -Z, --compress=0-9           compression level for compressed formats"
                echo "  --help                       show this help, then exit"
                mock::postgres::save_state
                return 0
                ;;
            --version)
                echo "pg_dump (PostgreSQL) ${POSTGRES_MOCK_CONFIG[version]}"
                mock::postgres::save_state
                return 0
                ;;
            *)
                # Treat as database name if no -d specified
                if [[ -z "$database" ]]; then
                    database="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Check if PostgreSQL is connected
    if [[ "${POSTGRES_MOCK_CONFIG[connected]}" != "true" ]]; then
        echo "pg_dump: error: could not connect to database \"$database\": Connection refused" >&2
        mock::postgres::save_state
        return 1
    fi
    
    # Generate dump content
    local dump_content
    dump_content=$(mock::postgres::generate_dump "$database" "$schema_only" "$data_only")
    
    # Output to file or stdout
    if [[ -n "$output_file" ]]; then
        echo "$dump_content" > "$output_file"
        echo "-- Database dump completed successfully to $output_file" >&2
    else
        echo "$dump_content"
    fi
    
    mock::postgres::save_state
    return 0
}

# Generate database dump content
mock::postgres::generate_dump() {
    local database="${1:-${POSTGRES_MOCK_CONFIG[database]}}"
    local schema_only="${2:-false}"
    local data_only="${3:-false}"
    
    if [[ "$data_only" != "true" ]]; then
        cat << EOF
--
-- PostgreSQL database dump
--

-- Dumped from database version ${POSTGRES_MOCK_CONFIG[version]}
-- Dumped by pg_dump version ${POSTGRES_MOCK_CONFIG[version]}

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: $database; Type: DATABASE; Schema: -; Owner: postgres
--

CREATE DATABASE $database WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE = 'en_US.utf8';

ALTER DATABASE $database OWNER TO postgres;

\\connect $database

EOF
        
        # Add table schemas
        for table in "${!POSTGRES_MOCK_TABLES[@]}"; do
            echo "--"
            echo "-- Name: $table; Type: TABLE; Schema: public; Owner: postgres"
            echo "--"
            echo ""
            echo "CREATE TABLE public.$table ("
            echo "    id integer NOT NULL,"
            echo "    name character varying(100),"
            echo "    created_at timestamp without time zone DEFAULT now()"
            echo ");"
            echo ""
            echo "ALTER TABLE public.$table OWNER TO postgres;"
            echo ""
        done
    fi
    
    if [[ "$schema_only" != "true" ]]; then
        echo "--"
        echo "-- Data for tables (mock data)"
        echo "--"
        echo ""
        
        for table in "${!POSTGRES_MOCK_TABLES[@]}"; do
            echo "COPY public.$table (id, name, created_at) FROM stdin;"
            echo "1\tSample Data\t$(date '+%Y-%m-%d %H:%M:%S')"
            echo "\\."
            echo ""
        done
    fi
    
    echo "--"
    echo "-- PostgreSQL database dump complete"
    echo "--"
}

# Mock createdb command
createdb() {
    mock::log_and_verify "createdb" "$@"
    
    # Always reload state at the beginning to handle BATS subshells
    mock::postgres::load_state
    
    # Check for error injection before processing
    if [[ -n "${POSTGRES_MOCK_CONFIG[error_mode]}" ]]; then
        case "${POSTGRES_MOCK_CONFIG[error_mode]}" in
            "connection_timeout"|"auth_failed"|"database_not_exist"|"too_many_connections")
                echo "createdb: error: could not connect to database template1: Connection refused" >&2
                mock::postgres::save_state
                return 1
                ;;
        esac
    fi
    
    local host="${POSTGRES_MOCK_CONFIG[host]}"
    local port="${POSTGRES_MOCK_CONFIG[port]}"
    local user="${POSTGRES_MOCK_CONFIG[user]}"
    local database_name=""
    local owner=""
    local template="template1"
    local encoding="UTF8"
    
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
            -U|--username)
                user="$2"
                shift 2
                ;;
            -O|--owner)
                owner="$2"
                shift 2
                ;;
            -T|--template)
                template="$2"
                shift 2
                ;;
            -E|--encoding)
                encoding="$2"
                shift 2
                ;;
            --help)
                echo "createdb creates a PostgreSQL database."
                echo ""
                echo "Usage: createdb [OPTION]... [DBNAME] [DESCRIPTION]"
                echo ""
                echo "Options:"
                echo "  -D, --tablespace=TABLESPACE  default tablespace for the database"
                echo "  -e, --echo                   show the commands being sent to the server"
                echo "  -E, --encoding=ENCODING      encoding for the database"
                echo "  -l, --locale=LOCALE          locale settings for the database"
                echo "  -O, --owner=OWNER            database user to own the new database"
                echo "  -T, --template=TEMPLATE      template database to copy"
                echo "  -V, --version                output version information, then exit"
                echo "  -?, --help                   show this help, then exit"
                mock::postgres::save_state
                return 0
                ;;
            --version)
                echo "createdb (PostgreSQL) ${POSTGRES_MOCK_CONFIG[version]}"
                mock::postgres::save_state
                return 0
                ;;
            -*)
                echo "createdb: invalid option -- '$1'" >&2
                echo "Try 'createdb --help' for more information." >&2
                mock::postgres::save_state
                return 1
                ;;
            *)
                if [[ -z "$database_name" ]]; then
                    database_name="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Default database name to username if not specified
    if [[ -z "$database_name" ]]; then
        database_name="${POSTGRES_MOCK_CONFIG[user]}"
    fi
    
    # Check if PostgreSQL is connected
    if [[ "${POSTGRES_MOCK_CONFIG[connected]}" != "true" ]]; then
        echo "createdb: error: could not connect to database template1: Connection refused" >&2
        mock::postgres::save_state
        return 1
    fi
    
    # Check if database already exists
    if [[ -n "${POSTGRES_MOCK_DATABASES[$database_name]:-}" ]]; then
        echo "createdb: error: database \"$database_name\" already exists" >&2
        mock::postgres::save_state
        return 1
    fi
    
    # Create the database
    POSTGRES_MOCK_DATABASES["$database_name"]="true"
    
    mock::postgres::save_state
    return 0
}

# Mock dropdb command
dropdb() {
    mock::log_and_verify "dropdb" "$@"
    
    # Always reload state at the beginning to handle BATS subshells
    mock::postgres::load_state
    
    # Check for error injection before processing
    if [[ -n "${POSTGRES_MOCK_CONFIG[error_mode]}" ]]; then
        case "${POSTGRES_MOCK_CONFIG[error_mode]}" in
            "connection_timeout"|"auth_failed"|"database_not_exist"|"too_many_connections")
                echo "dropdb: error: could not connect to database template1: Connection refused" >&2
                mock::postgres::save_state
                return 1
                ;;
        esac
    fi
    
    local host="${POSTGRES_MOCK_CONFIG[host]}"
    local port="${POSTGRES_MOCK_CONFIG[port]}"
    local user="${POSTGRES_MOCK_CONFIG[user]}"
    local database_name=""
    local interactive=false
    local force=false
    
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
            -U|--username)
                user="$2"
                shift 2
                ;;
            -i|--interactive)
                interactive=true
                shift
                ;;
            -f|--force)
                force=true
                shift
                ;;
            --help)
                echo "dropdb removes a PostgreSQL database."
                echo ""
                echo "Usage: dropdb [OPTION]... DBNAME"
                echo ""
                echo "Options:"
                echo "  -e, --echo               show the commands being sent to the server"
                echo "  -f, --force              try to terminate other connections before dropping"
                echo "  -i, --interactive        prompt before deleting anything"
                echo "  -V, --version            output version information, then exit"
                echo "  -?, --help               show this help, then exit"
                mock::postgres::save_state
                return 0
                ;;
            --version)
                echo "dropdb (PostgreSQL) ${POSTGRES_MOCK_CONFIG[version]}"
                mock::postgres::save_state
                return 0
                ;;
            -*)
                echo "dropdb: invalid option -- '$1'" >&2
                echo "Try 'dropdb --help' for more information." >&2
                mock::postgres::save_state
                return 1
                ;;
            *)
                if [[ -z "$database_name" ]]; then
                    database_name="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Database name is required
    if [[ -z "$database_name" ]]; then
        echo "dropdb: error: missing required argument database name" >&2
        echo "Try 'dropdb --help' for more information." >&2
        mock::postgres::save_state
        return 1
    fi
    
    # Check if PostgreSQL is connected
    if [[ "${POSTGRES_MOCK_CONFIG[connected]}" != "true" ]]; then
        echo "dropdb: error: could not connect to database template1: Connection refused" >&2
        mock::postgres::save_state
        return 1
    fi
    
    # Check if database exists
    if [[ -z "${POSTGRES_MOCK_DATABASES[$database_name]:-}" ]]; then
        echo "dropdb: error: database \"$database_name\" does not exist" >&2
        mock::postgres::save_state
        return 1
    fi
    
    # Interactive confirmation (simulated in mock)
    if [[ "$interactive" == "true" ]]; then
        echo "Database \"$database_name\" will be permanently removed." >&2
        echo "Are you sure? (y/N) y" >&2  # Simulate user input
    fi
    
    # Drop the database
    unset POSTGRES_MOCK_DATABASES["$database_name"]
    
    mock::postgres::save_state
    return 0
}

# Test helper functions
mock::postgres::reset() {
    # Optional parameter to control whether to save state after reset
    local save_state="${1:-true}"
    
    # Clear all data
    POSTGRES_MOCK_DATABASES=()
    POSTGRES_MOCK_TABLES=()
    POSTGRES_MOCK_QUERY_RESULTS=()
    
    # Reset configuration
    POSTGRES_MOCK_CONFIG=(
        [host]="localhost"
        [port]="5432"
        [user]="postgres"
        [password]="password"
        [database]="testdb"
        [connected]="true"
        [version]="15.3"
        [server_status]="running"
        [error_mode]=""
        [container_name]="postgres_test"
        [max_connections]="100"
        [shared_buffers]="128MB"
    )
    
    # Save the reset state to file if requested (default: true)
    if [[ "$save_state" == "true" ]]; then
        mock::postgres::save_state
    fi
    
    mock::log_state "postgres" "PostgreSQL mock reset to initial state"
}

mock::postgres::set_error() {
    local error_mode="$1"
    POSTGRES_MOCK_CONFIG[error_mode]="$error_mode"
    mock::postgres::save_state
    mock::log_state "postgres" "Set PostgreSQL error mode: $error_mode"
}

mock::postgres::set_connected() {
    local connected="$1"
    POSTGRES_MOCK_CONFIG[connected]="$connected"
    mock::postgres::save_state
    mock::log_state "postgres" "Set PostgreSQL connected: $connected"
}

mock::postgres::set_server_status() {
    local status="$1"
    POSTGRES_MOCK_CONFIG[server_status]="$status"
    mock::postgres::save_state
    mock::log_state "postgres" "Set PostgreSQL server status: $status"
}

mock::postgres::set_config() {
    local key="$1"
    local value="$2"
    POSTGRES_MOCK_CONFIG[$key]="$value"
    mock::postgres::save_state
    mock::log_state "postgres" "Set PostgreSQL config: $key=$value"
}

mock::postgres::add_database() {
    local db_name="$1"
    POSTGRES_MOCK_DATABASES["$db_name"]="true"
    mock::postgres::save_state
    mock::log_state "postgres" "Added database: $db_name"
}

mock::postgres::add_table() {
    local table_name="$1"
    POSTGRES_MOCK_TABLES["$table_name"]="true"
    mock::postgres::save_state
    mock::log_state "postgres" "Added table: $table_name"
}

mock::postgres::set_query_result() {
    local query="$1"
    local result="$2"
    POSTGRES_MOCK_QUERY_RESULTS["$query"]="$result"
    mock::postgres::save_state
    mock::log_state "postgres" "Set custom query result for: $query"
}

# Test assertions
mock::postgres::assert_database_exists() {
    local db_name="$1"
    if [[ -n "${POSTGRES_MOCK_DATABASES[$db_name]:-}" ]]; then
        return 0
    else
        echo "Assertion failed: Database '$db_name' does not exist" >&2
        return 1
    fi
}

mock::postgres::assert_table_exists() {
    local table_name="$1"
    if [[ -n "${POSTGRES_MOCK_TABLES[$table_name]:-}" ]]; then
        return 0
    else
        echo "Assertion failed: Table '$table_name' does not exist" >&2
        return 1
    fi
}

mock::postgres::assert_config_value() {
    local key="$1"
    local expected_value="$2"
    local actual_value="${POSTGRES_MOCK_CONFIG[$key]:-}"
    
    if [[ "$actual_value" == "$expected_value" ]]; then
        return 0
    else
        echo "Assertion failed: Config '$key' value mismatch" >&2
        echo "  Expected: '$expected_value'" >&2
        echo "  Actual: '$actual_value'" >&2
        return 1
    fi
}

# Debug functions
mock::postgres::dump_state() {
    echo "=== PostgreSQL Mock State ==="
    echo "Configuration:"
    for key in "${!POSTGRES_MOCK_CONFIG[@]}"; do
        echo "  $key: ${POSTGRES_MOCK_CONFIG[$key]}"
    done
    
    echo "Databases:"
    if [[ ${#POSTGRES_MOCK_DATABASES[@]} -eq 0 ]]; then
        echo "  (none)"
    else
        for db in "${!POSTGRES_MOCK_DATABASES[@]}"; do
            echo "  $db: ${POSTGRES_MOCK_DATABASES[$db]}"
        done
    fi
    
    echo "Tables:"
    if [[ ${#POSTGRES_MOCK_TABLES[@]} -eq 0 ]]; then
        echo "  (none)"
    else
        for table in "${!POSTGRES_MOCK_TABLES[@]}"; do
            echo "  $table: ${POSTGRES_MOCK_TABLES[$table]}"
        done
    fi
    
    echo "Custom Query Results:"
    if [[ ${#POSTGRES_MOCK_QUERY_RESULTS[@]} -eq 0 ]]; then
        echo "  (none)"
    else
        for query in "${!POSTGRES_MOCK_QUERY_RESULTS[@]}"; do
            echo "  $query: ${POSTGRES_MOCK_QUERY_RESULTS[$query]}"
        done
    fi
    echo "============================="
}

# Export all functions
export -f psql pg_isready pg_dump createdb dropdb
export -f mock::postgres::save_state
export -f mock::postgres::load_state
export -f mock::postgres::execute_sql
export -f mock::postgres::show_psql_help
export -f mock::postgres::generate_dump
export -f mock::postgres::reset
export -f mock::postgres::set_error
export -f mock::postgres::set_connected
export -f mock::postgres::set_server_status
export -f mock::postgres::set_config
export -f mock::postgres::add_database
export -f mock::postgres::add_table
export -f mock::postgres::set_query_result
export -f mock::postgres::assert_database_exists
export -f mock::postgres::assert_table_exists
export -f mock::postgres::assert_config_value
export -f mock::postgres::dump_state

# Save initial state
mock::postgres::save_state

mock::log_state "postgres" "PostgreSQL mock implementation loaded"