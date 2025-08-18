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
    local silent="${3:-false}"
    
    if ! questdb::docker::is_running; then
        [[ "$silent" != "true" ]] && log::error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    if ! questdb::validate_query "$query"; then
        return 1
    fi
    
    [[ "$silent" != "true" ]] && log::info "Executing query..."
    
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
        [[ "$silent" != "true" ]] && log::error "❌ Query failed"
        echo "$body" | jq -r '.message // .' 2>/dev/null || echo "$body"
        return 1
    fi
    
    [[ "$silent" != "true" ]] && log::success "✅ Query executed successfully"
    echo "$body" | jq -r '.' 2>/dev/null || echo "$body"
}

#######################################
# Get server status
# Returns:
#   Server status JSON
#######################################
questdb::api::status() {
    if ! questdb::docker::is_running; then
        log::error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    # Since QuestDB doesn't have a /status endpoint, we'll create a status response
    # by checking if the database is responding and getting basic metrics
    local health_check
    if questdb::api::health_check; then
        health_check="healthy"
    else
        health_check="unhealthy"
    fi
    
    # Get table count and version info
    local table_count version
    table_count=$(questdb::api::query "SELECT COUNT(*) FROM tables()" 1 2>/dev/null | \
        jq -r '.dataset[0][0] // 0' 2>/dev/null || echo "0")
    
    # Construct status JSON
    cat <<EOF | jq -r '.'
{
    "status": "$health_check",
    "http_port": ${QUESTDB_HTTP_PORT},
    "pg_port": ${QUESTDB_PG_PORT},
    "ilp_port": ${QUESTDB_ILP_PORT},
    "tables": $table_count
}
EOF
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
        log::error "Table name required"
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
        log::error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    if [[ -z "$data" ]]; then
        log::error "No data provided"
        return 1
    fi
    
    log::info "Inserting data..."
    
    # Send data via TCP
    echo "$data" | nc -w 5 localhost "${QUESTDB_ILP_PORT}" || {
        log::error "Failed to send data via InfluxDB Line Protocol"
        return 1
    }
    
    log::success "Data sent successfully"
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
        log::error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
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
        log::warning "psql not installed, using HTTP API instead"
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
        log::error "CSV file not found: $csv_file"
        return 1
    fi
    
    log::info "Bulk inserting data into table: $table"
    
    # Upload CSV via HTTP API
    local response
    response=$(curl -s -F "data=@${csv_file}" \
        "${QUESTDB_BASE_URL}/imp?name=${table}" \
        -w "\n%{http_code}" 2>/dev/null)
    
    local http_code
    http_code=$(echo "$response" | tail -n1)
    
    if [[ "$http_code" != "200" ]]; then
        log::error "Bulk insert failed"
        echo "$response" | head -n-1
        return 1
    fi
    
    log::success "Bulk insert completed"
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
        log::error "Table name required"
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
    # Use SELECT 1 query to check if database is responding
    local response
    response=$(curl -s -G --max-time 5 "${QUESTDB_BASE_URL}/exec" \
        --data-urlencode "query=SELECT 1" 2>/dev/null)
    
    # Check if response contains expected result
    echo "$response" | grep -q '"count":1' 2>/dev/null
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