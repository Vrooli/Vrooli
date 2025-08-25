#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL Data Injection Adapter
# This script handles injection of schemas, data, and migrations into PostgreSQL
# Part of the Vrooli resource data injection system

export DESCRIPTION="Inject schemas, data, and migrations into PostgreSQL database"

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/postgres"

# Source var.sh first for directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh"

# Source common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

# Source PostgreSQL configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default PostgreSQL connection settings
readonly DEFAULT_PG_HOST="localhost"
readonly DEFAULT_PG_PORT="5433"
readonly DEFAULT_PG_USER="postgres"
readonly DEFAULT_PG_DATABASE="vrooli"

# Connection settings (can be overridden by environment)
PG_HOST="${PG_HOST:-$DEFAULT_PG_HOST}"
PG_PORT="${PG_PORT:-$DEFAULT_PG_PORT}"
PG_USER="${PG_USER:-$DEFAULT_PG_USER}"
PG_PASSWORD="${PG_PASSWORD:-}"
PG_DATABASE="${PG_DATABASE:-$DEFAULT_PG_DATABASE}"

# Operation tracking
declare -a POSTGRES_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
postgres_inject::usage() {
    cat << EOF
PostgreSQL Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects schemas, data, and migrations into PostgreSQL based on scenario
    configuration. Supports validation, injection, status checks, and rollback.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected data
    --rollback    Rollback injected data
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "data": [
        {
          "type": "schema",
          "file": "path/to/schema.sql"
        },
        {
          "type": "seed",
          "file": "path/to/seed-data.sql"
        },
        {
          "type": "migration",
          "file": "path/to/migration.sql"
        }
      ],
      "databases": [
        {
          "name": "db_name",
          "owner": "db_owner"
        }
      ],
      "users": [
        {
          "name": "username",
          "password": "password",
          "privileges": ["CREATE", "CONNECT"]
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"data": [{"type": "schema", "file": "schema.sql"}]}'
    
    # Inject data
    $0 --inject '{"data": [{"type": "seed", "file": "data.sql"}]}'

EOF
}

#######################################
# Check if PostgreSQL is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
postgres_inject::check_accessibility() {
    # Check if psql is available
    if ! system::is_command "psql"; then
        log::warn "psql command not available, checking Docker alternative"
        
        # Check if we can use Docker PostgreSQL
        if docker::run ps --format "{{.Names}}" | grep -q "vrooli-postgres"; then
            log::debug "PostgreSQL accessible via Docker container"
            return 0
        else
            log::error "PostgreSQL client not available"
            log::info "Install psql or ensure PostgreSQL Docker container is running"
            return 1
        fi
    fi
    
    # Test connection
    local test_query="SELECT version();"
    
    if [[ -n "$PG_PASSWORD" ]]; then
        export PGPASSWORD="$PG_PASSWORD"
    fi
    
    if psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DATABASE" -c "$test_query" >/dev/null 2>&1; then
        log::debug "PostgreSQL is accessible at $PG_HOST:$PG_PORT"
        return 0
    else
        log::error "PostgreSQL is not accessible at $PG_HOST:$PG_PORT"
        log::info "Check connection settings and ensure PostgreSQL is running"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
postgres_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    POSTGRES_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added PostgreSQL rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
postgres_inject::execute_rollback() {
    if [[ ${#POSTGRES_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No PostgreSQL rollback actions to execute"
        return 0
    fi
    
    log::info "Executing PostgreSQL rollback actions..."
    
    local success_count=0
    local total_count=${#POSTGRES_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#POSTGRES_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${POSTGRES_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "PostgreSQL rollback completed: $success_count/$total_count actions successful"
    POSTGRES_ROLLBACK_ACTIONS=()
}

#######################################
# Execute SQL file
# Arguments:
#   $1 - SQL file path
#   $2 - database name (optional)
# Returns:
#   0 if successful, 1 if failed
#######################################
postgres_inject::execute_sql_file() {
    local sql_file="$1"
    local database="${2:-$PG_DATABASE}"
    
    if [[ -n "$PG_PASSWORD" ]]; then
        export PGPASSWORD="$PG_PASSWORD"
    fi
    
    # Check if using Docker PostgreSQL
    if ! system::is_command "psql" && docker::run ps --format "{{.Names}}" | grep -q "vrooli-postgres"; then
        # Use Docker exec to run psql
        local container_name
        container_name=$(docker::run ps --format "{{.Names}}" | grep "vrooli-postgres" | head -1)
        
        if docker::run exec -i "$container_name" psql -U "$PG_USER" -d "$database" < "$sql_file" 2>/dev/null; then
            return 0
        else
            return 1
        fi
    else
        # Use local psql
        if psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$database" -f "$sql_file" >/dev/null 2>&1; then
            return 0
        else
            return 1
        fi
    fi
}

#######################################
# Validate data configuration
# Arguments:
#   $1 - data configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
postgres_inject::validate_data() {
    local data_config="$1"
    
    log::debug "Validating data configurations..."
    
    # Check if data is an array
    local data_type
    data_type=$(echo "$data_config" | jq -r 'type')
    
    if [[ "$data_type" != "array" ]]; then
        log::error "Data configuration must be an array, got: $data_type"
        return 1
    fi
    
    # Validate each data item
    local data_count
    data_count=$(echo "$data_config" | jq 'length')
    
    for ((i=0; i<data_count; i++)); do
        local data_item
        data_item=$(echo "$data_config" | jq -c ".[$i]")
        
        # Check required fields
        local type file
        type=$(echo "$data_item" | jq -r '.type // empty')
        file=$(echo "$data_item" | jq -r '.file // empty')
        
        if [[ -z "$type" ]]; then
            log::error "Data item at index $i missing required 'type' field"
            return 1
        fi
        
        if [[ -z "$file" ]]; then
            log::error "Data item at index $i missing required 'file' field"
            return 1
        fi
        
        # Validate type
        case "$type" in
            schema|seed|migration|file|directory)
                log::debug "Data item has valid type: $type"
                ;;
            *)
                log::error "Data item has invalid type: $type"
                return 1
                ;;
        esac
        
        # Check if file exists
        local sql_file="$VROOLI_PROJECT_ROOT/$file"
        if [[ ! -f "$sql_file" ]]; then
            log::error "SQL file not found: $sql_file"
            return 1
        fi
        
        log::debug "Data item '$file' configuration is valid"
    done
    
    log::success "All data configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
postgres_inject::validate_config() {
    local config="$1"
    
    log::info "Validating PostgreSQL injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in PostgreSQL injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_data has_databases has_users
    has_data=$(echo "$config" | jq -e '.data' >/dev/null 2>&1 && echo "true" || echo "false")
    has_databases=$(echo "$config" | jq -e '.databases' >/dev/null 2>&1 && echo "true" || echo "false")
    has_users=$(echo "$config" | jq -e '.users' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_data" == "false" && "$has_databases" == "false" && "$has_users" == "false" ]]; then
        log::error "PostgreSQL injection configuration must have 'data', 'databases', or 'users'"
        return 1
    fi
    
    # Validate data if present
    if [[ "$has_data" == "true" ]]; then
        local data
        data=$(echo "$config" | jq -c '.data')
        
        if ! postgres_inject::validate_data "$data"; then
            return 1
        fi
    fi
    
    # Note: Database and user validation would require more checks
    if [[ "$has_databases" == "true" ]]; then
        log::warn "Database creation validation not yet fully implemented"
    fi
    
    if [[ "$has_users" == "true" ]]; then
        log::warn "User creation validation not yet fully implemented"
    fi
    
    log::success "PostgreSQL injection configuration is valid"
    return 0
}

#######################################
# Import data item into PostgreSQL
# Arguments:
#   $1 - data configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
postgres_inject::import_data_item() {
    local data_config="$1"
    
    local type file database
    type=$(echo "$data_config" | jq -r '.type')
    file=$(echo "$data_config" | jq -r '.file')
    database=$(echo "$data_config" | jq -r '.database // empty')
    
    # Use specified database or default
    if [[ -z "$database" ]]; then
        database="$PG_DATABASE"
    fi
    
    log::info "Importing $type: $file"
    
    # Resolve file path
    local sql_file="$VROOLI_PROJECT_ROOT/$file"
    
    # Create backup point for rollback (if type is schema or migration)
    if [[ "$type" == "schema" ]] || [[ "$type" == "migration" ]]; then
        # Note: This is a simplified rollback. In production, you'd want
        # to create a proper backup or use database transactions
        log::debug "Creating rollback point for $type"
        
        # Add rollback action (simplified - just logs the need for manual rollback)
        postgres_inject::add_rollback_action \
            "Rollback $type: $file" \
            "echo 'Manual rollback required for $type: $file in database $database'"
    fi
    
    # Execute SQL file
    if postgres_inject::execute_sql_file "$sql_file" "$database"; then
        log::success "Imported $type: $file"
        return 0
    else
        log::error "Failed to import $type: $file"
        return 1
    fi
}

#######################################
# Create database
# Arguments:
#   $1 - database configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
postgres_inject::create_database() {
    local db_config="$1"
    
    local name owner
    name=$(echo "$db_config" | jq -r '.name')
    owner=$(echo "$db_config" | jq -r '.owner // empty')
    
    log::info "Creating database: $name"
    
    if [[ -n "$PG_PASSWORD" ]]; then
        export PGPASSWORD="$PG_PASSWORD"
    fi
    
    # Create database SQL
    local create_sql="CREATE DATABASE \"$name\""
    if [[ -n "$owner" ]]; then
        create_sql="$create_sql OWNER \"$owner\""
    fi
    create_sql="$create_sql;"
    
    # Execute create database
    if psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d postgres -c "$create_sql" 2>/dev/null; then
        log::success "Created database: $name"
        
        # Add rollback action
        postgres_inject::add_rollback_action \
            "Drop database: $name" \
            "psql -h '$PG_HOST' -p '$PG_PORT' -U '$PG_USER' -d postgres -c 'DROP DATABASE IF EXISTS \"$name\";' >/dev/null 2>&1"
        
        return 0
    else
        # Database might already exist
        log::warn "Database '$name' may already exist"
        return 0
    fi
}

#######################################
# Inject data into PostgreSQL
# Arguments:
#   $1 - data configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
postgres_inject::inject_data_items() {
    local data_config="$1"
    
    log::info "Injecting data into PostgreSQL..."
    
    local data_count
    data_count=$(echo "$data_config" | jq 'length')
    
    if [[ "$data_count" -eq 0 ]]; then
        log::info "No data to inject"
        return 0
    fi
    
    local failed_items=()
    
    for ((i=0; i<data_count; i++)); do
        local data_item
        data_item=$(echo "$data_config" | jq -c ".[$i]")
        
        local file
        file=$(echo "$data_item" | jq -r '.file')
        
        if ! postgres_inject::import_data_item "$data_item"; then
            failed_items+=("$file")
        fi
    done
    
    if [[ ${#failed_items[@]} -eq 0 ]]; then
        log::success "All data items injected successfully"
        return 0
    else
        log::error "Failed to inject data items: ${failed_items[*]}"
        return 1
    fi
}

#######################################
# Inject databases
# Arguments:
#   $1 - databases configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
postgres_inject::inject_databases() {
    local databases_config="$1"
    
    log::info "Creating databases..."
    
    local db_count
    db_count=$(echo "$databases_config" | jq 'length')
    
    if [[ "$db_count" -eq 0 ]]; then
        log::info "No databases to create"
        return 0
    fi
    
    local failed_databases=()
    
    for ((i=0; i<db_count; i++)); do
        local database
        database=$(echo "$databases_config" | jq -c ".[$i]")
        
        local db_name
        db_name=$(echo "$database" | jq -r '.name')
        
        if ! postgres_inject::create_database "$database"; then
            failed_databases+=("$db_name")
        fi
    done
    
    if [[ ${#failed_databases[@]} -eq 0 ]]; then
        log::success "All databases created successfully"
        return 0
    else
        log::error "Failed to create databases: ${failed_databases[*]}"
        return 1
    fi
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
postgres_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into PostgreSQL"
    
    # Check PostgreSQL accessibility
    if ! postgres_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    POSTGRES_ROLLBACK_ACTIONS=()
    
    # Inject databases if present
    local has_databases
    has_databases=$(echo "$config" | jq -e '.databases' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_databases" == "true" ]]; then
        local databases
        databases=$(echo "$config" | jq -c '.databases')
        
        if ! postgres_inject::inject_databases "$databases"; then
            log::error "Failed to create databases"
            postgres_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject data if present
    local has_data
    has_data=$(echo "$config" | jq -e '.data' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_data" == "true" ]]; then
        local data
        data=$(echo "$config" | jq -c '.data')
        
        if ! postgres_inject::inject_data_items "$data"; then
            log::error "Failed to inject data"
            postgres_inject::execute_rollback
            return 1
        fi
    fi
    
    # Note: User creation not yet fully implemented
    local has_users
    has_users=$(echo "$config" | jq -e '.users' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_users" == "true" ]]; then
        log::warn "User creation not yet fully implemented for PostgreSQL"
    fi
    
    log::success "✅ PostgreSQL data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
postgres_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking PostgreSQL injection status"
    
    # Check PostgreSQL accessibility
    if ! postgres_inject::check_accessibility; then
        return 1
    fi
    
    if [[ -n "$PG_PASSWORD" ]]; then
        export PGPASSWORD="$PG_PASSWORD"
    fi
    
    # Check database status
    local has_databases
    has_databases=$(echo "$config" | jq -e '.databases' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_databases" == "true" ]]; then
        local databases
        databases=$(echo "$config" | jq -c '.databases')
        
        log::info "Checking database status..."
        
        local db_count
        db_count=$(echo "$databases" | jq 'length')
        
        for ((i=0; i<db_count; i++)); do
            local database
            database=$(echo "$databases" | jq -c ".[$i]")
            
            local name
            name=$(echo "$database" | jq -r '.name')
            
            # Check if database exists
            if psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d postgres -lqt | cut -d \| -f 1 | grep -qw "$name"; then
                log::success "✅ Database '$name' exists"
            else
                log::error "❌ Database '$name' not found"
            fi
        done
    fi
    
    # Data status would require tracking what was imported
    log::info "Data injection status tracking not yet implemented"
    
    return 0
}

#######################################
# Main execution function
#######################################
postgres_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        postgres_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            postgres_inject::validate_config "$config"
            ;;
        "--inject")
            postgres_inject::inject_data "$config"
            ;;
        "--status")
            postgres_inject::check_status "$config"
            ;;
        "--rollback")
            postgres_inject::execute_rollback
            ;;
        "--help")
            postgres_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            postgres_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        postgres_inject::usage
        exit 1
    fi
    
    postgres_inject::main "$@"
fi