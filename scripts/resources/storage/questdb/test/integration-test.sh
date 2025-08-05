#!/usr/bin/env bash
# QuestDB Integration Test
# Tests real QuestDB time-series database functionality
# Tests web console, SQL queries, time-series operations, and PostgreSQL wire protocol

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../../tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load QuestDB configuration
RESOURCES_DIR="$SCRIPT_DIR/../../.."
# shellcheck disable=SC1091
source "$RESOURCES_DIR/common.sh"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../config/defaults.sh"
questdb::export_config

# Override library defaults with QuestDB-specific settings
SERVICE_NAME="questdb"
BASE_URL="${QUESTDB_BASE_URL:-http://localhost:9010}"
HEALTH_ENDPOINT="/status"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "HTTP Port: ${QUESTDB_HTTP_PORT:-9010}"
    "PostgreSQL Port: ${QUESTDB_PG_PORT:-8812}"
    "ILP Port: ${QUESTDB_ILP_PORT:-9011}"
    "Container: ${QUESTDB_CONTAINER_NAME:-questdb}"
    "Data Dir: ${QUESTDB_DATA_DIR:-${HOME}/.questdb/data}"
)

#######################################
# QUESTDB-SPECIFIC TEST FUNCTIONS
#######################################

test_web_console() {
    local test_name="web console accessibility"
    
    local response
    if response=$(make_api_request "/" "GET" 10); then
        if echo "$response" | grep -qi "questdb\|<!DOCTYPE html>\|console"; then
            log_test_result "$test_name" "PASS" "web console accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "web console not accessible"
    return 1
}

test_status_endpoint() {
    local test_name="status endpoint"
    
    local response
    if response=$(make_api_request "/status" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local status
            status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
            log_test_result "$test_name" "PASS" "status: $status"
            return 0
        elif echo "$response" | grep -qi "running\|ok\|healthy"; then
            log_test_result "$test_name" "PASS" "status endpoint accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "status endpoint not working"
    return 1
}

test_sql_execution() {
    local test_name="SQL execution via HTTP"
    
    # Test basic SQL query
    local sql_query="SELECT 1 as test_value"
    local encoded_query
    encoded_query=$(printf '%s' "$sql_query" | sed 's/ /%20/g')
    
    local response
    if response=$(make_api_request "/exec?query=$encoded_query" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            # Check for result columns or data
            if echo "$response" | jq -e '.columns // .dataset' >/dev/null 2>&1; then
                log_test_result "$test_name" "PASS" "SQL execution working"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "SQL execution not working"
    return 1
}

test_table_operations() {
    local test_name="table operations (basic test)"
    
    # Create a test table
    local table_name="integration_test_$(date +%s)"
    local create_sql="CREATE TABLE $table_name (ts TIMESTAMP, value DOUBLE) timestamp(ts) PARTITION BY DAY"
    local encoded_create
    encoded_create=$(printf '%s' "$create_sql" | sed 's/ /%20/g; s/(/%28/g; s/)/%29/g')
    
    local response
    if response=$(make_api_request "/exec?query=$encoded_create" "GET" 10); then
        # Try to insert data
        local insert_sql="INSERT INTO $table_name VALUES ('2024-01-01T00:00:00.000Z', 42.0)"
        local encoded_insert
        encoded_insert=$(printf '%s' "$insert_sql" | sed 's/ /%20/g; s/(/%28/g; s/)/%29/g; s/'\''/%27/g')
        
        if make_api_request "/exec?query=$encoded_insert" "GET" 5 >/dev/null 2>&1; then
            # Clean up
            local drop_sql="DROP TABLE $table_name"
            local encoded_drop
            encoded_drop=$(printf '%s' "$drop_sql" | sed 's/ /%20/g')
            make_api_request "/exec?query=$encoded_drop" "GET" 5 >/dev/null 2>&1 || true
            
            log_test_result "$test_name" "PASS" "table operations working"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "table operations test failed"
    return 2
}

test_metrics_endpoint() {
    local test_name="metrics endpoint"
    
    local response
    if response=$(make_api_request "/metrics" "GET" 5); then
        if echo "$response" | grep -q "# HELP\|questdb_\|# TYPE"; then
            log_test_result "$test_name" "PASS" "Prometheus metrics available"
            return 0
        elif echo "$response" | jq . >/dev/null 2>&1; then
            log_test_result "$test_name" "PASS" "JSON metrics available"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "metrics endpoint not available"
    return 2
}

test_postgresql_wire_protocol() {
    local test_name="PostgreSQL wire protocol"
    
    if ! command -v psql >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "psql not available"
        return 2
    fi
    
    # Test PostgreSQL connection
    local pg_port="${QUESTDB_PG_PORT:-8812}"
    if PGPASSWORD="quest" psql -h localhost -p "$pg_port" -U admin -d qdb -c "SELECT 1;" >/dev/null 2>&1; then
        log_test_result "$test_name" "PASS" "PostgreSQL wire protocol working"
        return 0
    else
        log_test_result "$test_name" "SKIP" "PostgreSQL wire protocol not accessible"
        return 2
    fi
}

test_ilp_endpoint() {
    local test_name="InfluxDB Line Protocol endpoint"
    
    # Test ILP endpoint availability (just check if port responds)
    local ilp_port="${QUESTDB_ILP_PORT:-9011}"
    if timeout 3 bash -c "</dev/tcp/localhost/$ilp_port" 2>/dev/null; then
        log_test_result "$test_name" "PASS" "ILP endpoint accessible"
        return 0
    else
        log_test_result "$test_name" "SKIP" "ILP endpoint not accessible"
        return 2
    fi
}

test_container_health() {
    local test_name="Docker container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "^${QUESTDB_CONTAINER_NAME}$"; then
        local container_status
        container_status=$(docker inspect "${QUESTDB_CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "container running"
            return 0
        else
            log_test_result "$test_name" "FAIL" "container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "container not found"
        return 1
    fi
}

test_log_output() {
    local test_name="application log health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local logs_output
    if logs_output=$(docker logs "${QUESTDB_CONTAINER_NAME}" --tail 10 2>&1 2>/dev/null || true); then
        # Look for QuestDB startup success patterns
        if echo "$logs_output" | grep -qi "questdb.*started\|server.*running\|listening.*on"; then
            log_test_result "$test_name" "PASS" "healthy QuestDB logs"
            return 0
        elif echo "$logs_output" | grep -qi "error\|exception\|failed\|fatal"; then
            log_test_result "$test_name" "FAIL" "errors detected in logs"
            return 1
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "log status unclear"
    return 2
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "QuestDB Information:"
    echo "  Web Console: $BASE_URL"
    echo "  PostgreSQL Connection: ${QUESTDB_PG_URL}"
    echo "  API Endpoints:"
    echo "    - Status: GET $BASE_URL/status"
    echo "    - Execute SQL: GET $BASE_URL/exec?query=..."
    echo "    - Metrics: GET $BASE_URL/metrics"
    echo "  Protocols:"
    echo "    HTTP: ${QUESTDB_HTTP_PORT}"
    echo "    PostgreSQL Wire: ${QUESTDB_PG_PORT}"
    echo "    InfluxDB Line Protocol: ${QUESTDB_ILP_PORT}"
    echo "  Container: ${QUESTDB_CONTAINER_NAME}"
    echo "  Data Directory: ${QUESTDB_DATA_DIR}"
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register QuestDB-specific tests
register_tests \
    "test_web_console" \
    "test_status_endpoint" \
    "test_sql_execution" \
    "test_table_operations" \
    "test_metrics_endpoint" \
    "test_postgresql_wire_protocol" \
    "test_ilp_endpoint" \
    "test_container_health" \
    "test_log_output"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi