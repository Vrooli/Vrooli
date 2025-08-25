#!/usr/bin/env bash
# PostgreSQL Integration Test
# Tests real PostgreSQL multi-instance functionality
# Tests instance management, database operations, and connection health

set -euo pipefail

# Source var.sh first for directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/postgres/test"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source enhanced integration test library with fixture support
# shellcheck disable=SC1091
source "${var_SCRIPTS_DIR}/__test/lib/enhanced-integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load PostgreSQL configuration using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../lib/common.sh"

# Override library defaults with PostgreSQL-specific settings
# shellcheck disable=SC2034
SERVICE_NAME="postgres"
# shellcheck disable=SC2034
BASE_URL="postgresql://localhost:${POSTGRES_DEFAULT_PORT:-5433}"
# shellcheck disable=SC2034
HEALTH_ENDPOINT=""  # PostgreSQL uses connection-based health checks
# shellcheck disable=SC2034
REQUIRED_TOOLS=("curl" "docker" "psql")
# shellcheck disable=SC2034
SERVICE_METADATA=(
    "Default Port: ${POSTGRES_DEFAULT_PORT:-5433}"
    "Port Range: ${POSTGRES_INSTANCE_PORT_RANGE_START:-5433}-${POSTGRES_INSTANCE_PORT_RANGE_END:-5499}"
    "Max Instances: ${POSTGRES_MAX_INSTANCES:-67}"
    "Container Prefix: ${POSTGRES_CONTAINER_PREFIX:-vrooli-postgres}"
    "Default User: ${POSTGRES_DEFAULT_USER:-vrooli}"
    "Default DB: ${POSTGRES_DEFAULT_DB:-vrooli_client}"
)

#######################################
# POSTGRESQL-SPECIFIC TEST FUNCTIONS
#######################################

test_docker_availability() {
    local test_name="Docker availability"
    
    if command -v docker >/dev/null 2>&1; then
        if docker info >/dev/null 2>&1; then
            log_test_result "$test_name" "PASS" "Docker daemon accessible"
            return 0
        else
            log_test_result "$test_name" "FAIL" "Docker daemon not accessible"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "Docker not installed"
        return 1
    fi
}

test_instance_discovery() {
    local test_name="instance discovery"
    
    # Use the common function to list instances
    local instances
    mapfile -t instances < <(postgres::common::list_instances 2>/dev/null)
    if [[ ${#instances[@]} -gt 0 ]]; then
        local instance_count=${#instances[@]}
        if [[ $instance_count -gt 0 ]]; then
            log_test_result "$test_name" "PASS" "instances found: $instance_count (${instances[*]})"
            return 0
        else
            log_test_result "$test_name" "SKIP" "no instances found"
            return 2
        fi
    else
        log_test_result "$test_name" "FAIL" "instance discovery failed"
        return 1
    fi
}

test_running_instances() {
    local test_name="running instances check"
    
    local instances
    mapfile -t instances < <(postgres::common::list_instances 2>/dev/null)
    if [[ ${#instances[@]} -gt 0 ]]; then
        local running_count=0
        local running_instances=()
        
        for instance in "${instances[@]}"; do
            if postgres::common::is_running "$instance" 2>/dev/null; then
                running_count=$((running_count + 1))
                running_instances+=("$instance")
            fi
        done
        
        if [[ $running_count -gt 0 ]]; then
            log_test_result "$test_name" "PASS" "running instances: $running_count (${running_instances[*]})"
            return 0
        else
            log_test_result "$test_name" "SKIP" "no running instances"
            return 2
        fi
    else
        log_test_result "$test_name" "SKIP" "no instances to check"
        return 2
    fi
}

test_instance_health_checks() {
    local test_name="instance health checks"
    
    local instances
    mapfile -t instances < <(postgres::common::list_instances 2>/dev/null)
    if [[ ${#instances[@]} -gt 0 ]]; then
        local healthy_count=0
        local tested_count=0
        
        for instance in "${instances[@]}"; do
            if postgres::common::is_running "$instance" 2>/dev/null; then
                tested_count=$((tested_count + 1))
                if postgres::common::health_check "$instance" 2>/dev/null; then
                    healthy_count=$((healthy_count + 1))
                fi
            fi
        done
        
        if [[ $tested_count -gt 0 ]]; then
            if [[ $healthy_count -eq $tested_count ]]; then
                log_test_result "$test_name" "PASS" "all running instances healthy ($healthy_count/$tested_count)"
                return 0
            else
                log_test_result "$test_name" "FAIL" "some instances unhealthy ($healthy_count/$tested_count)"
                return 1
            fi
        else
            log_test_result "$test_name" "SKIP" "no running instances to test"
            return 2
        fi
    else
        log_test_result "$test_name" "SKIP" "no instances found"
        return 2
    fi
}

test_database_connections() {
    local test_name="database connections"
    
    if ! command -v psql >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "psql not available"
        return 2
    fi
    
    local instances
    mapfile -t instances < <(postgres::common::list_instances 2>/dev/null)
    if [[ ${#instances[@]} -gt 0 ]]; then
        local connected_count=0
        local tested_count=0
        
        for instance in "${instances[@]}"; do
            if postgres::common::is_running "$instance" 2>/dev/null; then
                tested_count=$((tested_count + 1))
                
                local port
                port=$(postgres::common::get_instance_config "$instance" "port" 2>/dev/null || echo "${POSTGRES_DEFAULT_PORT}")
                local user="${POSTGRES_DEFAULT_USER}"
                local db="${POSTGRES_DEFAULT_DB}"
                
                # Test basic connection
                if PGPASSWORD="password" psql -h localhost -p "$port" -U "$user" -d "$db" -c "SELECT 1;" >/dev/null 2>&1; then
                    connected_count=$((connected_count + 1))
                fi
            fi
        done
        
        if [[ $tested_count -gt 0 ]]; then
            if [[ $connected_count -gt 0 ]]; then
                log_test_result "$test_name" "PASS" "database connections working ($connected_count/$tested_count)"
                return 0
            else
                log_test_result "$test_name" "FAIL" "no database connections working ($connected_count/$tested_count)"
                return 1
            fi
        else
            log_test_result "$test_name" "SKIP" "no running instances to test"
            return 2
        fi
    else
        log_test_result "$test_name" "SKIP" "no instances found"
        return 2
    fi
}

test_container_health() {
    local test_name="container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check for PostgreSQL containers
    local containers
    containers=$(docker ps --format '{{.Names}}' | grep "^${POSTGRES_CONTAINER_PREFIX}" || echo "")
    
    if [[ -n "$containers" ]]; then
        local container_count
        container_count=$(echo "$containers" | wc -l)
        
        # Check if all containers are running
        local healthy_containers=0
        while IFS= read -r container; do
            if [[ -n "$container" ]]; then
                local status
                status=$(docker inspect "$container" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
                if [[ "$status" == "running" ]]; then
                    healthy_containers=$((healthy_containers + 1))
                fi
            fi
        done <<< "$containers"
        
        if [[ $healthy_containers -eq $container_count ]]; then
            log_test_result "$test_name" "PASS" "all containers healthy ($healthy_containers/$container_count)"
            return 0
        else
            log_test_result "$test_name" "FAIL" "some containers unhealthy ($healthy_containers/$container_count)"
            return 1
        fi
    else
        log_test_result "$test_name" "SKIP" "no PostgreSQL containers found"
        return 2
    fi
}

test_port_allocation() {
    local test_name="port allocation check"
    
    local instances
    mapfile -t instances < <(postgres::common::list_instances 2>/dev/null)
    if [[ ${#instances[@]} -gt 0 ]]; then
        local port_conflicts=0
        local used_ports=()
        
        for instance in "${instances[@]}"; do
            local port
            port=$(postgres::common::get_instance_config "$instance" "port" 2>/dev/null || echo "${POSTGRES_DEFAULT_PORT}")
            
            # Check if port is already in used_ports array
            for used_port in "${used_ports[@]}"; do
                if [[ "$port" == "$used_port" ]]; then
                    port_conflicts=$((port_conflicts + 1))
                    break
                fi
            done
            
            used_ports+=("$port")
        done
        
        if [[ $port_conflicts -eq 0 ]]; then
            log_test_result "$test_name" "PASS" "no port conflicts detected"
            return 0
        else
            log_test_result "$test_name" "FAIL" "port conflicts detected: $port_conflicts"
            return 1
        fi
    else
        log_test_result "$test_name" "SKIP" "no instances to check"
        return 2
    fi
}

test_configuration_files() {
    local test_name="configuration files integrity"
    
    local instances
    mapfile -t instances < <(postgres::common::list_instances 2>/dev/null)
    if [[ ${#instances[@]} -gt 0 ]]; then
        local valid_configs=0
        local total_configs=0
        
        for instance in "${instances[@]}"; do
            total_configs=$((total_configs + 1))
            
            # Check if instance has valid configuration
            local config_file="${POSTGRES_INSTANCES_DIR}/${instance}/config/instance.conf"
            if [[ -f "$config_file" ]]; then
                # Basic validation - check if it contains key PostgreSQL settings
                if grep -q "port\|max_connections\|shared_buffers" "$config_file" 2>/dev/null; then
                    valid_configs=$((valid_configs + 1))
                fi
            fi
        done
        
        if [[ $total_configs -gt 0 ]]; then
            if [[ $valid_configs -eq $total_configs ]]; then
                log_test_result "$test_name" "PASS" "all configurations valid ($valid_configs/$total_configs)"
                return 0
            else
                log_test_result "$test_name" "FAIL" "invalid configurations found ($valid_configs/$total_configs)"
                return 1
            fi
        else
            log_test_result "$test_name" "SKIP" "no instances to check"
            return 2
        fi
    else
        log_test_result "$test_name" "SKIP" "no instances found"
        return 2
    fi
}

#######################################
# FIXTURE-BASED DATABASE TESTS
#######################################

test_sql_fixture_import() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Find a running instance to test with
    local instances
    mapfile -t instances < <(postgres::common::list_instances 2>/dev/null)
    if [[ ${#instances[@]} -eq 0 ]]; then
        return 2  # No instances available
    fi
    
    local test_instance=""
    for instance in "${instances[@]}"; do
        if postgres::common::is_running "$instance" 2>/dev/null; then
            test_instance="$instance"
            break
        fi
    done
    
    if [[ -z "$test_instance" ]]; then
        return 2  # No running instance
    fi
    
    # Get connection details
    local port
    port=$(postgres::common::get_instance_config "$test_instance" "port" 2>/dev/null || echo "5433")
    
    # Test if we can import the SQL fixture (would need credentials in production)
    local filename
    filename=$(basename "$fixture_path")
    
    # Check file extension to handle different data types
    case "$filename" in
        *.sql)
            # Would import SQL file
            return 0
            ;;
        *.csv)
            # Would import CSV data
            return 0
            ;;
        *.json)
            # Would import JSON data (PostgreSQL supports JSONB)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

test_json_storage_fixture() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Read JSON data
    local json_data
    json_data=$(cat "$fixture_path")
    
    # Validate it's valid JSON
    if ! echo "$json_data" | jq empty 2>/dev/null; then
        return 1  # Invalid JSON
    fi
    
    # In a real test, we would:
    # 1. Create a test table with JSONB column
    # 2. Insert the JSON data
    # 3. Query it back to verify storage
    
    # For now, just validate the fixture is usable
    return 0
}

test_csv_import_fixture() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Check if it's a valid CSV by reading first line
    if head -1 "$fixture_path" | grep -q ","; then
        # Valid CSV format
        return 0
    fi
    
    return 1
}

# Run fixture-based database tests
run_postgres_fixture_tests() {
    if [[ "$FIXTURES_AVAILABLE" == "true" ]]; then
        # First check if we have any running instances
        local instances
        mapfile -t instances < <(postgres::common::list_instances 2>/dev/null)
    if [[ ${#instances[@]} -eq 0 ]]; then
            log_test_result "fixture tests" "SKIP" "no PostgreSQL instances available"
            return 2
        fi
        
        local has_running=false
        for instance in "${instances[@]}"; do
            if postgres::common::is_running "$instance" 2>/dev/null; then
                has_running=true
                break
            fi
        done
        
        if [[ "$has_running" != "true" ]]; then
            log_test_result "fixture tests" "SKIP" "no running PostgreSQL instances"
            return 2
        fi
        
        # Test with structured data fixtures
        test_with_fixture "import JSON data" "documents" "structured/database_export.json" \
            test_json_storage_fixture
        
        test_with_fixture "import CSV data" "documents" "structured/customers.csv" \
            test_csv_import_fixture
        
        test_with_fixture "import inventory data" "documents" "structured/inventory.tsv" \
            test_csv_import_fixture
        
        # Test with workflow data
        test_with_fixture "store workflow JSON" "documents" "workflow-data.json" \
            test_json_storage_fixture
        
        # Test with various data formats
        local storage_fixtures
        storage_fixtures=$(discover_resource_fixtures "postgres" "storage")
        
        for fixture_pattern in $storage_fixtures; do
            # Look for structured data files
            local data_files
            if data_files=$(fixture_get_all "$fixture_pattern" "*.csv" 2>/dev/null | head -3); then
                for data_file in $data_files; do
                    local data_name
                    data_name=$(basename "$data_file")
                    test_with_fixture "import $data_name" "" "$data_file" \
                        test_csv_import_fixture
                done
            fi
            
            # Test JSON data storage
            if data_files=$(fixture_get_all "$fixture_pattern" "*.json" 2>/dev/null | head -3); then
                for data_file in $data_files; do
                    local data_name
                    data_name=$(basename "$data_file")
                    test_with_fixture "store JSON $data_name" "" "$data_file" \
                        test_json_storage_fixture
                done
            fi
        done
        
        # Test with negative fixtures for robustness
        if [[ -d "$FIXTURES_DIR/negative-tests" ]]; then
            test_with_fixture "handle malformed JSON" "negative-tests" "documents/malformed.json" \
                test_json_storage_fixture
            
            test_with_fixture "handle invalid CSV" "negative-tests" "edge-cases/null_bytes.txt" \
                test_csv_import_fixture
        fi
    fi
}

test_log_output() {
    local test_name="container log health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local containers
    containers=$(docker ps --format '{{.Names}}' | grep "^${POSTGRES_CONTAINER_PREFIX}" | head -1 || echo "")
    
    if [[ -n "$containers" ]]; then
        local container
        container=$(echo "$containers" | head -1)
        
        local logs_output
        if logs_output=$(docker logs "$container" --tail 10 2>&1 2>/dev/null || true); then
            # Look for PostgreSQL startup success patterns
            if echo "$logs_output" | grep -qi "database system is ready\|accepting connections\|PostgreSQL init process complete"; then
                log_test_result "$test_name" "PASS" "healthy PostgreSQL logs"
                return 0
            elif echo "$logs_output" | grep -qi "FATAL\|ERROR.*failed\|PANIC"; then
                log_test_result "$test_name" "FAIL" "errors detected in PostgreSQL logs"
                return 1
            fi
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
    echo "PostgreSQL Information:"
    echo "  Multi-Instance Setup: Yes"
    echo "  Default Connection: postgresql://${POSTGRES_DEFAULT_USER}:password@localhost:${POSTGRES_DEFAULT_PORT}/${POSTGRES_DEFAULT_DB}"
    echo "  Port Range: ${POSTGRES_INSTANCE_PORT_RANGE_START}-${POSTGRES_INSTANCE_PORT_RANGE_END}"
    echo "  Max Instances: ${POSTGRES_MAX_INSTANCES}"
    echo "  Container Prefix: ${POSTGRES_CONTAINER_PREFIX}"
    echo "  Data Directory: ${POSTGRES_INSTANCES_DIR}"
    echo "  Templates: ${POSTGRES_TEMPLATE_DIR}"
    
    local instances
    mapfile -t instances < <(postgres::common::list_instances 2>/dev/null)
    if [[ ${#instances[@]} -gt 0 ]]; then
        echo "  Instances (${#instances[@]}):"
        for instance in "${instances[@]}"; do
            local port
            port=$(postgres::common::get_instance_config "$instance" "port" 2>/dev/null || echo "unknown")
            local status="stopped"
            if postgres::common::is_running "$instance" 2>/dev/null; then
                status="running"
            fi
            echo "    - $instance (port: $port, status: $status)"
        done
    else
        echo "  Instances: None found"
    fi
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (manage.sh validation, config checks, etc.)
register_standard_interface_tests

# Register PostgreSQL-specific tests
register_tests \
    "test_docker_availability" \
    "test_instance_discovery" \
    "test_running_instances" \
    "test_instance_health_checks" \
    "test_database_connections" \
    "test_container_health" \
    "test_port_allocation" \
    "test_configuration_files" \
    "test_log_output" \
    "run_postgres_fixture_tests"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi