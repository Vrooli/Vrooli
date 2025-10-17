#!/bin/bash
# PostGIS Cleanup Functions - Test Data Management

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POSTGIS_CLEANUP_LIB_DIR="${APP_ROOT}/resources/postgis/lib"

# Source common functions
source "${POSTGIS_CLEANUP_LIB_DIR}/common.sh"

#######################################
# Clean up test tables from database
# Arguments:
#   $1 - Database name (optional, defaults to 'spatial')
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::cleanup::test_tables() {
    local database="${1:-spatial}"
    local container="${POSTGIS_CONTAINER:-postgis-main}"

    log::info "Cleaning up test tables from database: $database"

    # Check if container is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::warning "PostGIS container not running, skipping cleanup"
        return 0
    fi

    # Get list of test tables
    local test_tables
    test_tables=$(docker exec "$container" psql -U vrooli -d "$database" -t -c \
        "SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename LIKE 'test_%';" 2>/dev/null | xargs)

    if [[ -z "$test_tables" ]]; then
        log::info "No test tables found"
        return 0
    fi

    log::info "Found test tables: $test_tables"

    # Drop each test table
    local dropped=0
    local failed=0
    for table in $test_tables; do
        if docker exec "$container" psql -U vrooli -d "$database" -c "DROP TABLE IF EXISTS $table CASCADE;" &>/dev/null; then
            log::success "Dropped table: $table"
            ((dropped++))
        else
            log::error "Failed to drop table: $table"
            ((failed++))
        fi
    done

    log::info "Cleanup complete: $dropped tables dropped, $failed failed"
    return 0
}

#######################################
# Clean up test databases
# Removes databases matching test_ pattern
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::cleanup::test_databases() {
    local container="${POSTGIS_CONTAINER:-postgis-main}"

    log::info "Cleaning up test databases"

    # Check if container is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::warning "PostGIS container not running, skipping cleanup"
        return 0
    fi

    # Get list of test databases
    local test_dbs
    test_dbs=$(docker exec "$container" psql -U vrooli -d postgres -t -c \
        "SELECT datname FROM pg_database WHERE datname LIKE 'test_%';" 2>/dev/null | xargs)

    if [[ -z "$test_dbs" ]]; then
        log::info "No test databases found"
        return 0
    fi

    log::info "Found test databases: $test_dbs"

    # Drop each test database
    local dropped=0
    local failed=0
    for db in $test_dbs; do
        # Terminate connections first
        docker exec "$container" psql -U vrooli -d postgres -c \
            "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$db';" &>/dev/null

        if docker exec "$container" psql -U vrooli -d postgres -c "DROP DATABASE IF EXISTS $db;" &>/dev/null; then
            log::success "Dropped database: $db"
            ((dropped++))
        else
            log::error "Failed to drop database: $db"
            ((failed++))
        fi
    done

    log::info "Cleanup complete: $dropped databases dropped, $failed failed"
    return 0
}

#######################################
# Clean up geocoding cache
# Removes cached geocoding results
# Arguments:
#   $1 - Database name (optional, defaults to 'spatial')
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::cleanup::geocoding_cache() {
    local database="${1:-spatial}"
    local container="${POSTGIS_CONTAINER:-postgis-main}"

    log::info "Cleaning up geocoding cache in database: $database"

    # Check if container is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::warning "PostGIS container not running, skipping cleanup"
        return 0
    fi

    # Check if geocoding_cache table exists
    local table_exists
    table_exists=$(docker exec "$container" psql -U vrooli -d "$database" -t -c \
        "SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'geocoding_cache');" 2>/dev/null | xargs)

    if [[ "$table_exists" != "t" ]]; then
        log::info "Geocoding cache table does not exist"
        return 0
    fi

    # Truncate the cache table
    if docker exec "$container" psql -U vrooli -d "$database" -c "TRUNCATE TABLE geocoding_cache;" &>/dev/null; then
        log::success "Geocoding cache cleared"
        return 0
    else
        log::error "Failed to clear geocoding cache"
        return 1
    fi
}

#######################################
# Full cleanup - all test data
# Cleans test tables, test databases, and caches
# Returns:
#   0 on success, 1 on failure
#######################################
postgis::cleanup::all() {
    log::header "Running full PostGIS cleanup"

    local failed=0

    # Clean up test tables in main database
    postgis::cleanup::test_tables "spatial" || ((failed++))

    # Clean up test databases
    postgis::cleanup::test_databases || ((failed++))

    # Clean up geocoding cache
    postgis::cleanup::geocoding_cache "spatial" || ((failed++))

    if [[ $failed -eq 0 ]]; then
        log::success "Full cleanup completed successfully"
        return 0
    else
        log::warning "Cleanup completed with $failed issues"
        return 1
    fi
}

# Export functions for CLI usage
export -f postgis::cleanup::test_tables
export -f postgis::cleanup::test_databases
export -f postgis::cleanup::geocoding_cache
export -f postgis::cleanup::all
