#!/usr/bin/env bash
set -euo pipefail

# QuestDB Timeseries Database Injection Adapter
# This script handles injection of tables and timeseries data into QuestDB
# Part of the Vrooli resource data injection system

export DESCRIPTION="Inject tables and timeseries data into QuestDB database"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common utilities
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"

# Source QuestDB configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default QuestDB settings
readonly DEFAULT_QUESTDB_HTTP_PORT="9000"
readonly DEFAULT_QUESTDB_PG_PORT="8812"
readonly DEFAULT_QUESTDB_HOST="localhost"

# QuestDB settings (can be overridden by environment)
QUESTDB_HTTP_PORT="${QUESTDB_HTTP_PORT:-$DEFAULT_QUESTDB_HTTP_PORT}"
QUESTDB_PG_PORT="${QUESTDB_PG_PORT:-$DEFAULT_QUESTDB_PG_PORT}"
QUESTDB_HOST="${QUESTDB_HOST:-$DEFAULT_QUESTDB_HOST}"

# Operation tracking
declare -a QUESTDB_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
questdb_inject::usage() {
    cat << EOF
QuestDB Timeseries Database Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects tables and timeseries data into QuestDB based on scenario configuration.
    Supports validation, injection, status checks, and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected tables
    --rollback    Rollback injected tables
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "tables": [
        {
          "name": "sensors",
          "schema": "timestamp TIMESTAMP, sensor_id SYMBOL, temperature DOUBLE, humidity DOUBLE",
          "partitionBy": "DAY",
          "timestamp": "timestamp"
        }
      ],
      "data": [
        {
          "table": "sensors",
          "type": "csv",
          "file": "path/to/sensor_data.csv",
          "format": "timestamp,sensor_id,temperature,humidity"
        },
        {
          "table": "metrics",
          "type": "influx",
          "data": "path/to/metrics.txt"
        }
      ],
      "indices": [
        {
          "table": "sensors",
          "column": "sensor_id",
          "type": "symbol"
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"tables": [{"name": "metrics", "schema": "ts TIMESTAMP, value DOUBLE"}]}'
    
    # Create tables and inject data
    $0 --inject '{"tables": [{"name": "logs", "schema": "timestamp TIMESTAMP, level SYMBOL, message STRING"}], "data": [{"table": "logs", "type": "csv", "file": "data/logs.csv"}]}'

EOF
}

#######################################
# Check if QuestDB is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
questdb_inject::check_accessibility() {
    # Check if QuestDB HTTP endpoint is accessible
    if curl -s --max-time 5 "http://${QUESTDB_HOST}:${QUESTDB_HTTP_PORT}/exec?query=SELECT%201" >/dev/null 2>&1; then
        log::debug "QuestDB is accessible at ${QUESTDB_HOST}:${QUESTDB_HTTP_PORT}"
        return 0
    else
        log::error "QuestDB is not accessible at ${QUESTDB_HOST}:${QUESTDB_HTTP_PORT}"
        log::info "Ensure QuestDB is running: ./scripts/resources/storage/questdb/manage.sh --action start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
questdb_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    QUESTDB_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added QuestDB rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
questdb_inject::execute_rollback() {
    if [[ ${#QUESTDB_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No QuestDB rollback actions to execute"
        return 0
    fi
    
    log::info "Executing QuestDB rollback actions..."
    
    local success_count=0
    local total_count=${#QUESTDB_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#QUESTDB_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${QUESTDB_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "QuestDB rollback completed: $success_count/$total_count actions successful"
    QUESTDB_ROLLBACK_ACTIONS=()
}

#######################################
# Execute SQL query
# Arguments:
#   $1 - SQL query
# Returns:
#   0 if successful, 1 if failed
#######################################
questdb_inject::execute_query() {
    local query="$1"
    
    # URL encode the query
    local encoded_query
    # shellcheck disable=SC2034
    encoded_query=$(echo "$query" | jq -sRr @uri)
    
    # Execute query via HTTP API
    local response
    if response=$(curl -s -G "http://${QUESTDB_HOST}:${QUESTDB_HTTP_PORT}/exec" \
        --data-urlencode "query=${query}" 2>&1); then
        # Check for error in response
        if echo "$response" | grep -q '"error"'; then
            log::error "Query failed: $(echo "$response" | jq -r '.error // "Unknown error"')"
            return 1
        else
            return 0
        fi
    else
        log::error "Failed to execute query"
        return 1
    fi
}

#######################################
# Validate table configuration
# Arguments:
#   $1 - tables configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
questdb_inject::validate_tables() {
    local tables_config="$1"
    
    log::debug "Validating table configurations..."
    
    # Check if tables is an array
    local tables_type
    tables_type=$(echo "$tables_config" | jq -r 'type')
    
    if [[ "$tables_type" != "array" ]]; then
        log::error "Tables configuration must be an array, got: $tables_type"
        return 1
    fi
    
    # Validate each table
    local table_count
    table_count=$(echo "$tables_config" | jq 'length')
    
    for ((i=0; i<table_count; i++)); do
        local table
        table=$(echo "$tables_config" | jq -c ".[$i]")
        
        # Check required fields
        local name schema
        name=$(echo "$table" | jq -r '.name // empty')
        schema=$(echo "$table" | jq -r '.schema // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Table at index $i missing required 'name' field"
            return 1
        fi
        
        if [[ -z "$schema" ]]; then
            log::error "Table '$name' missing required 'schema' field"
            return 1
        fi
        
        # Validate table name
        if [[ ! "$name" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
            log::error "Invalid table name: $name (must start with letter/underscore, contain only letters/numbers/underscores)"
            return 1
        fi
        
        # Validate partitionBy if specified
        local partition_by
        partition_by=$(echo "$table" | jq -r '.partitionBy // empty')
        
        if [[ -n "$partition_by" ]]; then
            case "$partition_by" in
                NONE|HOUR|DAY|MONTH|YEAR)
                    log::debug "Table '$name' has valid partition: $partition_by"
                    ;;
                *)
                    log::error "Table '$name' has invalid partition: $partition_by"
                    return 1
                    ;;
            esac
        fi
        
        log::debug "Table '$name' configuration is valid"
    done
    
    log::success "All table configurations are valid"
    return 0
}

#######################################
# Validate data configuration
# Arguments:
#   $1 - data configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
questdb_inject::validate_data() {
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
        local table type
        table=$(echo "$data_item" | jq -r '.table // empty')
        type=$(echo "$data_item" | jq -r '.type // empty')
        
        if [[ -z "$table" ]]; then
            log::error "Data item at index $i missing required 'table' field"
            return 1
        fi
        
        if [[ -z "$type" ]]; then
            log::error "Data item for table '$table' missing required 'type' field"
            return 1
        fi
        
        # Validate type and type-specific fields
        case "$type" in
            csv)
                local file
                file=$(echo "$data_item" | jq -r '.file // empty')
                
                if [[ -z "$file" ]]; then
                    log::error "CSV data item missing required 'file' field"
                    return 1
                fi
                
                # Check if file exists
                local file_path="$VROOLI_PROJECT_ROOT/$file"
                if [[ ! -f "$file_path" ]]; then
                    log::error "Data file not found: $file_path"
                    return 1
                fi
                ;;
            influx)
                local data
                data=$(echo "$data_item" | jq -r '.data // empty')
                
                if [[ -z "$data" ]]; then
                    log::error "InfluxDB data item missing required 'data' field"
                    return 1
                fi
                
                # Check if data file exists
                local data_path="$VROOLI_PROJECT_ROOT/$data"
                if [[ ! -f "$data_path" ]]; then
                    log::error "Data file not found: $data_path"
                    return 1
                fi
                ;;
            json)
                local file
                file=$(echo "$data_item" | jq -r '.file // empty')
                
                if [[ -z "$file" ]]; then
                    log::error "JSON data item missing required 'file' field"
                    return 1
                fi
                
                # Check if file exists
                local file_path="$VROOLI_PROJECT_ROOT/$file"
                if [[ ! -f "$file_path" ]]; then
                    log::error "Data file not found: $file_path"
                    return 1
                fi
                ;;
            *)
                log::error "Data item has invalid type: $type"
                return 1
                ;;
        esac
        
        log::debug "Data item for table '$table' is valid"
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
questdb_inject::validate_config() {
    local config="$1"
    
    log::info "Validating QuestDB injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in QuestDB injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_tables has_data has_indices
    has_tables=$(echo "$config" | jq -e '.tables' >/dev/null 2>&1 && echo "true" || echo "false")
    has_data=$(echo "$config" | jq -e '.data' >/dev/null 2>&1 && echo "true" || echo "false")
    has_indices=$(echo "$config" | jq -e '.indices' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_tables" == "false" && "$has_data" == "false" && "$has_indices" == "false" ]]; then
        log::error "QuestDB injection configuration must have 'tables', 'data', or 'indices'"
        return 1
    fi
    
    # Validate tables if present
    if [[ "$has_tables" == "true" ]]; then
        local tables
        tables=$(echo "$config" | jq -c '.tables')
        
        if ! questdb_inject::validate_tables "$tables"; then
            return 1
        fi
    fi
    
    # Validate data if present
    if [[ "$has_data" == "true" ]]; then
        local data
        data=$(echo "$config" | jq -c '.data')
        
        if ! questdb_inject::validate_data "$data"; then
            return 1
        fi
    fi
    
    log::success "QuestDB injection configuration is valid"
    return 0
}

#######################################
# Create table
# Arguments:
#   $1 - table configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
questdb_inject::create_table() {
    local table_config="$1"
    
    local name schema partition_by timestamp_col
    name=$(echo "$table_config" | jq -r '.name')
    schema=$(echo "$table_config" | jq -r '.schema')
    partition_by=$(echo "$table_config" | jq -r '.partitionBy // "NONE"')
    timestamp_col=$(echo "$table_config" | jq -r '.timestamp // empty')
    
    log::info "Creating table: $name"
    
    # Build CREATE TABLE query
    local create_query="CREATE TABLE IF NOT EXISTS ${name} (${schema})"
    
    # Add timestamp designation if specified
    if [[ -n "$timestamp_col" ]]; then
        create_query="${create_query} TIMESTAMP(${timestamp_col})"
    fi
    
    # Add partitioning if specified
    if [[ "$partition_by" != "NONE" ]]; then
        create_query="${create_query} PARTITION BY ${partition_by}"
    fi
    
    # Execute create table query
    if questdb_inject::execute_query "$create_query"; then
        log::success "Created table: $name"
        
        # Add rollback action
        questdb_inject::add_rollback_action \
            "Drop table: $name" \
            "questdb_inject::execute_query 'DROP TABLE IF EXISTS ${name}'"
        
        return 0
    else
        log::error "Failed to create table: $name"
        return 1
    fi
}

#######################################
# Import CSV data
# Arguments:
#   $1 - data configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
questdb_inject::import_csv() {
    local data_config="$1"
    
    local table file format
    table=$(echo "$data_config" | jq -r '.table')
    file=$(echo "$data_config" | jq -r '.file')
    # shellcheck disable=SC2034
    format=$(echo "$data_config" | jq -r '.format // empty')
    
    # Resolve file path
    local file_path="$VROOLI_PROJECT_ROOT/$file"
    
    log::info "Importing CSV data into table '$table' from: $file"
    
    # Use curl to upload CSV file via HTTP API
    local response
    if response=$(curl -s -F "data=@${file_path}" \
        "http://${QUESTDB_HOST}:${QUESTDB_HTTP_PORT}/imp?name=${table}" 2>&1); then
        # Check for error in response
        if echo "$response" | grep -q '"error"'; then
            log::error "Import failed: $(echo "$response" | jq -r '.error // "Unknown error"')"
            return 1
        else
            log::success "Imported CSV data into table: $table"
            return 0
        fi
    else
        log::error "Failed to import CSV data into table: $table"
        return 1
    fi
}

#######################################
# Import InfluxDB Line Protocol data
# Arguments:
#   $1 - data configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
questdb_inject::import_influx() {
    local data_config="$1"
    
    local table data_file
    table=$(echo "$data_config" | jq -r '.table')
    data_file=$(echo "$data_config" | jq -r '.data')
    
    # Resolve file path
    local file_path="$VROOLI_PROJECT_ROOT/$data_file"
    
    log::info "Importing InfluxDB Line Protocol data into table '$table' from: $data_file"
    
    # Read data and send via InfluxDB Line Protocol endpoint
    local data_content
    data_content=$(cat "$file_path")
    
    local response
    if response=$(curl -s -X POST "http://${QUESTDB_HOST}:9009/write?db=${table}" \
        --data-binary "$data_content" 2>&1); then
        log::success "Imported InfluxDB data into table: $table"
        return 0
    else
        log::error "Failed to import InfluxDB data into table: $table"
        return 1
    fi
}

#######################################
# Import JSON data
# Arguments:
#   $1 - data configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
questdb_inject::import_json() {
    local data_config="$1"
    
    local table file
    table=$(echo "$data_config" | jq -r '.table')
    file=$(echo "$data_config" | jq -r '.file')
    
    # Resolve file path
    local file_path="$VROOLI_PROJECT_ROOT/$file"
    
    log::info "Importing JSON data into table '$table' from: $file"
    
    # Read JSON data
    local json_data
    json_data=$(cat "$file_path")
    
    # Convert JSON to INSERT statements
    # Note: This is a simplified implementation
    local row_count
    row_count=$(echo "$json_data" | jq '. | length')
    
    for ((i=0; i<row_count; i++)); do
        local row
        row=$(echo "$json_data" | jq -c ".[$i]")
        
        # Build INSERT statement from JSON row
        local columns values
        columns=$(echo "$row" | jq -r 'keys | join(",")')
        values=$(echo "$row" | jq -r '[.[] | tostring] | map("'\''" + . + "'\''") | join(",")')
        
        local insert_query="INSERT INTO ${table} (${columns}) VALUES (${values})"
        
        if ! questdb_inject::execute_query "$insert_query"; then
            log::error "Failed to insert row $i into table: $table"
            return 1
        fi
    done
    
    log::success "Imported JSON data into table: $table"
    return 0
}

#######################################
# Inject tables
# Arguments:
#   $1 - tables configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
questdb_inject::inject_tables() {
    local tables_config="$1"
    
    log::info "Creating QuestDB tables..."
    
    local table_count
    table_count=$(echo "$tables_config" | jq 'length')
    
    if [[ "$table_count" -eq 0 ]]; then
        log::info "No tables to create"
        return 0
    fi
    
    local failed_tables=()
    
    for ((i=0; i<table_count; i++)); do
        local table
        table=$(echo "$tables_config" | jq -c ".[$i]")
        
        local table_name
        table_name=$(echo "$table" | jq -r '.name')
        
        if ! questdb_inject::create_table "$table"; then
            failed_tables+=("$table_name")
        fi
    done
    
    if [[ ${#failed_tables[@]} -eq 0 ]]; then
        log::success "All tables created successfully"
        return 0
    else
        log::error "Failed to create tables: ${failed_tables[*]}"
        return 1
    fi
}

#######################################
# Inject data
# Arguments:
#   $1 - data configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
questdb_inject::inject_data_items() {
    local data_config="$1"
    
    log::info "Importing data into QuestDB..."
    
    local data_count
    data_count=$(echo "$data_config" | jq 'length')
    
    if [[ "$data_count" -eq 0 ]]; then
        log::info "No data to import"
        return 0
    fi
    
    local failed_imports=()
    
    for ((i=0; i<data_count; i++)); do
        local data_item
        data_item=$(echo "$data_config" | jq -c ".[$i]")
        
        local type table
        type=$(echo "$data_item" | jq -r '.type')
        table=$(echo "$data_item" | jq -r '.table')
        
        case "$type" in
            csv)
                if ! questdb_inject::import_csv "$data_item"; then
                    failed_imports+=("csv:$table")
                fi
                ;;
            influx)
                if ! questdb_inject::import_influx "$data_item"; then
                    failed_imports+=("influx:$table")
                fi
                ;;
            json)
                if ! questdb_inject::import_json "$data_item"; then
                    failed_imports+=("json:$table")
                fi
                ;;
        esac
    done
    
    if [[ ${#failed_imports[@]} -eq 0 ]]; then
        log::success "All data imported successfully"
        return 0
    else
        log::error "Failed to import data: ${failed_imports[*]}"
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
questdb_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into QuestDB"
    
    # Check QuestDB accessibility
    if ! questdb_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    QUESTDB_ROLLBACK_ACTIONS=()
    
    # Inject tables if present
    local has_tables
    has_tables=$(echo "$config" | jq -e '.tables' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_tables" == "true" ]]; then
        local tables
        tables=$(echo "$config" | jq -c '.tables')
        
        if ! questdb_inject::inject_tables "$tables"; then
            log::error "Failed to create tables"
            questdb_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject data if present
    local has_data
    has_data=$(echo "$config" | jq -e '.data' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_data" == "true" ]]; then
        local data
        data=$(echo "$config" | jq -c '.data')
        
        if ! questdb_inject::inject_data_items "$data"; then
            log::error "Failed to import data"
            questdb_inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ QuestDB data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
questdb_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking QuestDB injection status"
    
    # Check QuestDB accessibility
    if ! questdb_inject::check_accessibility; then
        return 1
    fi
    
    # Check table status
    local has_tables
    has_tables=$(echo "$config" | jq -e '.tables' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_tables" == "true" ]]; then
        local tables
        tables=$(echo "$config" | jq -c '.tables')
        
        log::info "Checking table status..."
        
        local table_count
        table_count=$(echo "$tables" | jq 'length')
        
        for ((i=0; i<table_count; i++)); do
            local table
            table=$(echo "$tables" | jq -c ".[$i]")
            
            local name
            name=$(echo "$table" | jq -r '.name')
            
            # Check if table exists and get row count
            local count_query="SELECT COUNT(*) FROM ${name}"
            local response
            response=$(curl -s -G "http://${QUESTDB_HOST}:${QUESTDB_HTTP_PORT}/exec" \
                --data-urlencode "query=${count_query}" 2>/dev/null)
            
            if echo "$response" | grep -q '"error"'; then
                log::error "❌ Table '$name' not found"
            else
                local row_count
                row_count=$(echo "$response" | jq -r '.dataset[0][0] // 0')
                log::success "✅ Table '$name' exists (rows: $row_count)"
            fi
        done
    fi
    
    # Get overall database info
    log::info "Fetching database information..."
    
    local tables_query="SELECT table_name FROM tables()"
    local response
    response=$(curl -s -G "http://${QUESTDB_HOST}:${QUESTDB_HTTP_PORT}/exec" \
        --data-urlencode "query=${tables_query}" 2>/dev/null)
    
    if ! echo "$response" | grep -q '"error"'; then
        local total_tables
        total_tables=$(echo "$response" | jq '.dataset | length')
        log::info "Total tables in database: $total_tables"
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
questdb_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        questdb_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            questdb_inject::validate_config "$config"
            ;;
        "--inject")
            questdb_inject::inject_data "$config"
            ;;
        "--status")
            questdb_inject::check_status "$config"
            ;;
        "--rollback")
            questdb_inject::execute_rollback
            ;;
        "--help")
            questdb_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            questdb_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        questdb_inject::usage
        exit 1
    fi
    
    questdb_inject::main "$@"
fi