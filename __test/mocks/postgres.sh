#!/usr/bin/env bash
# Simplified PostgreSQL Mock for Resource Testing
# 
# Provides basic PostgreSQL command mocking for testing resource functionality
# Focus: Connection testing, basic queries, health checks
#
# NAMING CONVENTIONS: This mock follows the standard naming pattern expected
# by the convention-based resource testing framework:
#
#   test_postgres_connection() : Test basic database connectivity
#   test_postgres_health()     : Test database health/status  
#   test_postgres_basic()      : Test basic database operations
#
# This allows the testing framework to automatically discover and run these
# tests without requiring hardcoded knowledge of PostgreSQL specifics.

# Mock psql command
psql() {
    case "$*" in
        # Version check
        *"--version"*)
            echo "psql (PostgreSQL) 16.3"
            return 0
            ;;
        # Connection test with SELECT 1
        *"-c SELECT 1"*|*"-c 'SELECT 1'"*)
            echo " ?column? "
            echo "----------"
            echo "        1"
            echo "(1 row)"
            return 0
            ;;
        # List databases
        *"-l"*|*"--list"*)
            echo "                                  List of databases"
            echo "   Name    |  Owner   | Encoding |   Collate   |    Ctype    |   Access privileges"
            echo "-----------+----------+----------+-------------+-------------+----------------------"
            echo " postgres  | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |"
            echo " template0 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres         +"
            echo " template1 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres         +"
            echo " vrooli    | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |"
            echo "(4 rows)"
            return 0
            ;;
        # Any other query
        *"-c"*)
            echo "Query executed successfully"
            return 0
            ;;
        # Interactive mode (should not happen in tests)
        *)
            echo "psql: mock interactive mode not supported"
            return 0
            ;;
    esac
}

# Mock createdb command
createdb() {
    case "$*" in
        *"vrooli"*|*"test"*|*"issue_tracker"*)
            echo "CREATE DATABASE"
            return 0
            ;;
        *)
            echo "CREATE DATABASE"
            return 0
            ;;
    esac
}

# Mock dropdb command
dropdb() {
    echo "DROP DATABASE"
    return 0
}

# Mock pg_dump command
pg_dump() {
    case "$*" in
        *"--version"*)
            echo "pg_dump (PostgreSQL) 16.3"
            ;;
        *)
            echo "-- PostgreSQL database dump"
            echo "-- Mock dump output"
            ;;
    esac
    return 0
}

# Mock pg_restore command
pg_restore() {
    case "$*" in
        *"--version"*)
            echo "pg_restore (PostgreSQL) 16.3"
            ;;
        *)
            echo "pg_restore: mock restore completed"
            ;;
    esac
    return 0
}

# Test functions for resource validation
test_postgres_connection() {
    # Simulate connection test
    psql -h localhost -p 5432 -U postgres -c "SELECT 1" >/dev/null 2>&1
    return $?
}

test_postgres_basic() {
    # Simulate basic database operations test
    psql -h localhost -p 5432 -U postgres -c "SELECT version();" >/dev/null 2>&1
    return $?
}

test_postgres_advanced() {
    # Simulate advanced database operations test
    psql -h localhost -p 5432 -U postgres -c "SELECT current_timestamp;" >/dev/null 2>&1
    return $?
}

test_postgres_health() {
    # Simulate health check
    psql -h localhost -p 5432 -U postgres -c "SELECT 1" >/dev/null 2>&1
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        log_test_pass "postgres health check"
    else
        log_test_fail "postgres health check" "Connection failed"
    fi
    
    return $exit_code
}

# Export mock functions
export -f psql createdb dropdb pg_dump pg_restore
export -f test_postgres_connection test_postgres_basic test_postgres_advanced test_postgres_health