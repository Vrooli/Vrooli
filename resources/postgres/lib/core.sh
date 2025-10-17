#!/usr/bin/env bash
# PostgreSQL Resource Core Library
# Provides core functionality required by v2.0 contract

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POSTGRES_DIR="${APP_ROOT}/resources/postgres"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "${POSTGRES_DIR}/config/defaults.sh" 2>/dev/null || true
source "${POSTGRES_DIR}/lib/common.sh" 2>/dev/null || true
source "${POSTGRES_DIR}/lib/test.sh" 2>/dev/null || true

#######################################
# Core test delegation functions for v2.0 compliance
#######################################

postgres::test::smoke() {
    log::info "Running PostgreSQL smoke tests..."
    bash "${POSTGRES_DIR}/test/phases/test-smoke.sh"
    return $?
}

postgres::test::integration() {
    log::info "Running PostgreSQL integration tests..."
    bash "${POSTGRES_DIR}/test/phases/test-integration.sh"
    return $?
}

postgres::test::unit() {
    log::info "Running PostgreSQL unit tests..."
    bash "${POSTGRES_DIR}/test/phases/test-unit.sh"
    return $?
}

postgres::test::all() {
    log::info "Running all PostgreSQL tests..."
    bash "${POSTGRES_DIR}/test/run-tests.sh" all
    return $?
}

#######################################
# Core info function for v2.0 compliance
#######################################

postgres::info() {
    local format="${1:-}"
    local runtime_json="${POSTGRES_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_json" ]]; then
        log::error "Runtime configuration not found at $runtime_json"
        return 1
    fi
    
    if [[ "$format" == "--json" ]]; then
        cat "$runtime_json"
    else
        log::header "PostgreSQL Resource Information"
        log::info "Configuration from runtime.json:"
        
        local startup_order=$(jq -r '.startup_order // "unknown"' "$runtime_json")
        local dependencies=$(jq -r '.dependencies[]? // "none"' "$runtime_json" | paste -sd, -)
        local startup_timeout=$(jq -r '.startup_timeout // "60"' "$runtime_json")
        local startup_time=$(jq -r '.startup_time_estimate // "unknown"' "$runtime_json")
        local recovery_attempts=$(jq -r '.recovery_attempts // "3"' "$runtime_json")
        local priority=$(jq -r '.priority // "medium"' "$runtime_json")
        
        log::info "  Startup Order: $startup_order"
        log::info "  Dependencies: ${dependencies:-none}"
        log::info "  Startup Timeout: ${startup_timeout}s"
        log::info "  Startup Time Estimate: $startup_time"
        log::info "  Recovery Attempts: $recovery_attempts"
        log::info "  Priority: $priority"
    fi
}

#######################################
# Core help function for v2.0 compliance
#######################################

postgres::help() {
    cat << EOF
PostgreSQL Resource - v2.0 Contract Compliant

USAGE:
    resource-postgres <command> [options]
    resource-postgres <group> <subcommand> [options]

GROUPS & COMMANDS:

  manage        Resource lifecycle management
    install     Install PostgreSQL and dependencies
    uninstall   Remove PostgreSQL completely
    start       Start PostgreSQL service
    stop        Stop PostgreSQL service
    restart     Restart PostgreSQL service

  test          Run validation tests
    smoke       Quick health check (<30s)
    integration Full functionality tests
    unit        Library function tests
    all         Run all test suites

  content       Database content management
    add         Add SQL file to database
    list        List databases/tables
    get         Export database/table
    remove      Delete database/table
    execute     Run SQL query

  info          Show resource information from runtime.json
  status        Show detailed PostgreSQL status
  logs          Show PostgreSQL logs
  credentials   Show connection credentials
  help          Show this help message

EXAMPLES:
    # Start PostgreSQL
    resource-postgres manage start --wait
    
    # Run health check
    resource-postgres test smoke
    
    # Create new database
    resource-postgres content execute "CREATE DATABASE mydb"
    
    # Show status
    resource-postgres status

DEFAULT CONFIGURATION:
    Port Range: 5433-5499
    Default User: vrooli
    Default Database: vrooli
    Container: postgres:16-alpine

For more information, see resources/postgres/README.md
EOF
}

# Export functions for CLI usage
export -f postgres::test::smoke
export -f postgres::test::integration
export -f postgres::test::unit
export -f postgres::test::all
export -f postgres::info
export -f postgres::help