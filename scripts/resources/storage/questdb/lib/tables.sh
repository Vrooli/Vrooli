#!/usr/bin/env bash
# QuestDB Table Management Functions

#######################################
# List all tables with details
# Returns:
#   Table information
#######################################
questdb::tables::list() {
    if ! questdb::docker::is_running; then
        echo_error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    echo_header "QuestDB Tables"
    
    # Get table list with row counts
    local query="
        SELECT 
            t.table_name,
            t.designatedTimestamp,
            t.partitionBy,
            t.maxUncommittedRows,
            (SELECT COUNT(*) FROM t.table_name) as row_count
        FROM tables() t
    "
    
    # For now, use simpler query as the above won't work directly
    query="SELECT table_name FROM tables()"
    
    local result
    result=$(questdb::api::query "$query" 1000)
    
    if [[ $? -eq 0 ]]; then
        echo "$result" | jq -r '.dataset[] | .[0]' 2>/dev/null | while read -r table; do
            if [[ -n "$table" ]]; then
                local count
                count=$(questdb::api::table_count "$table" 2>/dev/null || echo "?")
                echo "📊 $table (rows: $count)"
            fi
        done
    else
        echo_error "Failed to list tables"
        return 1
    fi
}

#######################################
# Create table from schema file
# Arguments:
#   $1 - Table name
#   $2 - Schema file path
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::tables::create_from_file() {
    local table_name="$1"
    local schema_file="$2"
    
    if [[ -z "$table_name" ]] || [[ -z "$schema_file" ]]; then
        echo_error "Table name and schema file required"
        return 1
    fi
    
    if [[ ! -f "$schema_file" ]]; then
        echo_error "Schema file not found: $schema_file"
        return 1
    fi
    
    echo_info "${QUESTDB_API_MESSAGES["creating_table"]} $table_name"
    
    # Read schema
    local sql
    sql=$(<"$schema_file")
    
    # Execute create table statement
    if questdb::api::query "$sql" 1; then
        echo_success "${QUESTDB_API_MESSAGES["table_created"]}"
        return 0
    else
        return 1
    fi
}

#######################################
# Create default monitoring tables
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::tables::create_defaults() {
    echo_header "Creating Default Tables"
    
    # System metrics table
    local system_metrics_sql="
CREATE TABLE IF NOT EXISTS system_metrics (
    timestamp TIMESTAMP,
    host SYMBOL,
    metric_name SYMBOL,
    metric_value DOUBLE,
    tags STRING
) timestamp(timestamp) PARTITION BY DAY;
"
    
    # AI inference metrics
    local ai_metrics_sql="
CREATE TABLE IF NOT EXISTS ai_inference (
    timestamp TIMESTAMP,
    model SYMBOL,
    task_type SYMBOL,
    user_id SYMBOL,
    response_time_ms DOUBLE,
    tokens_input INT,
    tokens_output INT,
    cost DOUBLE,
    success BOOLEAN,
    error_type SYMBOL
) timestamp(timestamp) PARTITION BY DAY;
"
    
    # Resource health metrics
    local resource_health_sql="
CREATE TABLE IF NOT EXISTS resource_health (
    timestamp TIMESTAMP,
    resource_name SYMBOL,
    resource_type SYMBOL,
    status SYMBOL,
    response_time_ms DOUBLE,
    cpu_percent DOUBLE,
    memory_mb DOUBLE,
    error_count INT
) timestamp(timestamp) PARTITION BY DAY;
"
    
    # Workflow execution metrics
    local workflow_metrics_sql="
CREATE TABLE IF NOT EXISTS workflow_metrics (
    timestamp TIMESTAMP,
    workflow_id SYMBOL,
    workflow_name SYMBOL,
    platform SYMBOL,
    execution_time_ms DOUBLE,
    steps_total INT,
    steps_completed INT,
    status SYMBOL,
    error_message STRING
) timestamp(timestamp) PARTITION BY DAY;
"
    
    # Create each table
    local tables=(
        "system_metrics:$system_metrics_sql"
        "ai_inference:$ai_metrics_sql"
        "resource_health:$resource_health_sql"
        "workflow_metrics:$workflow_metrics_sql"
    )
    
    local failed=0
    for table_def in "${tables[@]}"; do
        local table_name="${table_def%%:*}"
        local table_sql="${table_def#*:}"
        
        echo_info "Creating table: $table_name"
        if questdb::api::query "$table_sql" 1 &>/dev/null; then
            echo_success "✓ $table_name created"
        else
            echo_warning "⚠ $table_name may already exist or failed to create"
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        echo_success "All default tables created successfully"
        return 0
    else
        echo_warning "Some tables may have failed to create"
        return 1
    fi
}

#######################################
# Drop table
# Arguments:
#   $1 - Table name
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::tables::drop() {
    local table="$1"
    
    if [[ -z "$table" ]]; then
        echo_error "Table name required"
        return 1
    fi
    
    echo_warning "⚠️  Dropping table: $table"
    if ! args::prompt_yes_no "Are you sure you want to drop table '$table'?" "n"; then
        echo_info "Drop cancelled"
        return 0
    fi
    
    if questdb::api::query "DROP TABLE IF EXISTS $table" 1; then
        echo_success "Table dropped successfully"
        return 0
    else
        echo_error "Failed to drop table"
        return 1
    fi
}

#######################################
# Show table information
# Arguments:
#   $1 - Table name
# Returns:
#   Table details
#######################################
questdb::tables::info() {
    local table="$1"
    
    if [[ -z "$table" ]]; then
        echo_error "Table name required"
        return 1
    fi
    
    echo_header "Table Information: $table"
    
    # Get schema
    echo ""
    echo "Schema:"
    questdb::api::table_schema "$table"
    
    # Get row count
    echo ""
    echo "Statistics:"
    local count
    count=$(questdb::api::table_count "$table")
    echo "  Row Count: $count"
    
    # Get sample data
    echo ""
    echo "Sample Data (first 5 rows):"
    questdb::api::query "SELECT * FROM $table LIMIT 5" 5
}

#######################################
# Truncate table
# Arguments:
#   $1 - Table name
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::tables::truncate() {
    local table="$1"
    
    if [[ -z "$table" ]]; then
        echo_error "Table name required"
        return 1
    fi
    
    echo_warning "⚠️  Truncating table: $table"
    if ! args::prompt_yes_no "Are you sure you want to remove all data from '$table'?" "n"; then
        echo_info "Truncate cancelled"
        return 0
    fi
    
    if questdb::api::query "TRUNCATE TABLE $table" 1; then
        echo_success "Table truncated successfully"
        return 0
    else
        echo_error "Failed to truncate table"
        return 1
    fi
}

#######################################
# Export table functions
#######################################
export -f questdb::tables::list
export -f questdb::tables::create_from_file
export -f questdb::tables::create_defaults
export -f questdb::tables::drop
export -f questdb::tables::info
export -f questdb::tables::truncate