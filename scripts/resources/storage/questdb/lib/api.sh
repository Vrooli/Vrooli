#!/usr/bin/env bash
# QuestDB API Functions
# HTTP REST, PostgreSQL, and InfluxDB Line Protocol interactions

#######################################
# Execute SQL query via HTTP API
# Arguments:
#   $1 - SQL query
#   $2 - Limit (optional)
# Returns:
#   Query results or error
#######################################
questdb::api::query() {
    local query="$1"
    local limit="${2:-100}"
    
    if ! questdb::docker::is_running; then
        echo_error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    if ! questdb::validate_query "$query"; then
        return 1
    fi
    
    echo_info "${QUESTDB_API_MESSAGES["executing_query"]}"
    
    # Execute query
    local response
    response=$(curl -s -G "${QUESTDB_BASE_URL}/exec" \
        --data-urlencode "query=${query}" \
        --data-urlencode "limit=${limit}" \
        --data-urlencode "fmt=json" \
        -w "\n%{http_code}" 2>/dev/null)
    
    local http_code
    http_code=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | head -n-1)
    
    if [[ "$http_code" != "200" ]]; then
        echo_error "${QUESTDB_API_MESSAGES["query_failed"]}"
        echo "$body" | jq -r '.message // .' 2>/dev/null || echo "$body"
        return 1
    fi
    
    echo_success "${QUESTDB_API_MESSAGES["query_success"]}"
    echo "$body" | jq -r '.' 2>/dev/null || echo "$body"
}

#######################################
# Get server status
# Returns:
#   Server status JSON
#######################################
questdb::api::status() {
    if ! questdb::docker::is_running; then
        echo_error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    curl -s "${QUESTDB_BASE_URL}/status" | jq -r '.' 2>/dev/null
}

#######################################
# List all tables
# Returns:
#   List of tables
#######################################
questdb::api::list_tables() {
    questdb::api::query "SELECT table_name FROM tables()" 1000
}

#######################################
# Get table schema
# Arguments:
#   $1 - Table name
# Returns:
#   Table schema
#######################################
questdb::api::table_schema() {
    local table="$1"
    
    if [[ -z "$table" ]]; then
        echo_error "Table name required"
        return 1
    fi
    
    questdb::api::query "SHOW COLUMNS FROM ${table}"
}

#######################################
# Send metrics via InfluxDB Line Protocol
# Arguments:
#   $1 - Metrics data (InfluxDB line format)
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::ilp::send() {
    local data="$1"
    
    if ! questdb::docker::is_running; then
        echo_error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    if [[ -z "$data" ]]; then
        echo_error "No data provided"
        return 1
    fi
    
    echo_info "${QUESTDB_API_MESSAGES["inserting_data"]}"
    
    # Send data via TCP
    echo "$data" | nc -w 5 localhost "${QUESTDB_ILP_PORT}" || {
        echo_error "Failed to send data via InfluxDB Line Protocol"
        return 1
    }
    
    echo_success "Data sent successfully"
}

#######################################
# Execute query via PostgreSQL protocol
# Arguments:
#   $1 - SQL query
# Returns:
#   Query results
#######################################
questdb::pg::query() {
    local query="$1"
    
    if ! questdb::docker::is_running; then
        echo_error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    # Use psql if available
    if command -v psql &> /dev/null; then
        PGPASSWORD="${QUESTDB_PG_PASSWORD}" psql \
            -h localhost \
            -p "${QUESTDB_PG_PORT}" \
            -U "${QUESTDB_PG_USER}" \
            -d qdb \
            -c "$query" \
            2>/dev/null
    else
        echo_warning "psql not installed, using HTTP API instead"
        questdb::api::query "$query"
    fi
}

#######################################
# Bulk insert CSV data
# Arguments:
#   $1 - Table name
#   $2 - CSV file path
# Returns:
#   0 on success, 1 on failure
#######################################
questdb::api::bulk_insert_csv() {
    local table="$1"
    local csv_file="$2"
    
    if [[ ! -f "$csv_file" ]]; then
        echo_error "CSV file not found: $csv_file"
        return 1
    fi
    
    echo_info "Bulk inserting data into table: $table"
    
    # Upload CSV via HTTP API
    local response
    response=$(curl -s -F "data=@${csv_file}" \
        "${QUESTDB_BASE_URL}/imp?name=${table}" \
        -w "\n%{http_code}" 2>/dev/null)
    
    local http_code
    http_code=$(echo "$response" | tail -n1)
    
    if [[ "$http_code" != "200" ]]; then
        echo_error "Bulk insert failed"
        echo "$response" | head -n-1
        return 1
    fi
    
    echo_success "Bulk insert completed"
}

#######################################
# Get table row count
# Arguments:
#   $1 - Table name
# Returns:
#   Row count
#######################################
questdb::api::table_count() {
    local table="$1"
    
    if [[ -z "$table" ]]; then
        echo_error "Table name required"
        return 1
    fi
    
    local result
    result=$(questdb::api::query "SELECT COUNT(*) as count FROM ${table}" 1)
    
    echo "$result" | jq -r '.dataset[0][0] // 0' 2>/dev/null || echo "0"
}

#######################################
# Health check endpoint
# Returns:
#   0 if healthy, 1 otherwise
#######################################
questdb::api::health_check() {
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" "${QUESTDB_BASE_URL}/status" 2>/dev/null || echo "000")
    
    [[ "$response" == "200" ]]
}

#######################################
# Export API functions
#######################################
export -f questdb::api::query
export -f questdb::api::status
export -f questdb::api::list_tables
export -f questdb::api::table_schema
export -f questdb::ilp::send
export -f questdb::pg::query
export -f questdb::api::bulk_insert_csv
export -f questdb::api::table_count
export -f questdb::api::health_check