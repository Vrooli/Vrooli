#!/usr/bin/env bash
################################################################################
# PostgreSQL Resource CLI - v2.0 Universal Contract Compliant
# 
# PostgreSQL relational database with multi-instance support
#
# Usage:
#   resource-postgres <command> [options]
#   resource-postgres <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    POSTGRES_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${POSTGRES_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
POSTGRES_CLI_DIR="${APP_ROOT}/resources/postgres"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${POSTGRES_CLI_DIR}/config/defaults.sh"

# Source PostgreSQL libraries
for lib in core common docker install status backup database instance multi_instance migration test; do
    lib_file="${POSTGRES_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "postgres" "PostgreSQL database with multi-instance support" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="postgres::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="postgres::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="postgres::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="postgres::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="postgres::docker::restart"

# Test handlers - delegate to test scripts per v2.0 contract
CLI_COMMAND_HANDLERS["test::smoke"]="postgres::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="postgres::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="postgres::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="postgres::test::all"

# Content handlers for database functionality
CLI_COMMAND_HANDLERS["content::add"]="postgres::content::add"
CLI_COMMAND_HANDLERS["content::list"]="postgres::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="postgres::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="postgres::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="postgres::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "info" "Show resource information from runtime.json" "postgres::info"
cli::register_command "help" "Show comprehensive help with examples" "postgres::help"
cli::register_command "status" "Show detailed PostgreSQL status" "postgres::status::show"
cli::register_command "logs" "Show PostgreSQL logs" "postgres::docker::logs"

# ==============================================================================
# POSTGRES-SPECIFIC COMMANDS
# ==============================================================================
# Database management commands for PostgreSQL
cli::register_command "credentials" "Show PostgreSQL credentials for integration" "postgres::credentials"

# Custom content subcommands for PostgreSQL-specific operations
cli::register_subcommand "content" "backup" "Create database backup" "postgres::backup::create" "modifies-system"
cli::register_subcommand "content" "restore" "Restore from backup" "postgres::backup::restore" "modifies-system"
cli::register_subcommand "content" "migrate" "Run database migrations" "postgres::migration::run" "modifies-system"

# Custom content subcommands for database operations
cli::register_subcommand "content" "create-database" "Create new database" "postgres::database::create" "modifies-system"
cli::register_subcommand "content" "create-instance" "Create new instance" "postgres::instance::create" "modifies-system"
cli::register_subcommand "content" "list-instances" "List all instances" "postgres::instance::list"

# Custom test subcommands for PostgreSQL health validation
cli::register_subcommand "test" "performance" "Run performance tests on PostgreSQL" "postgres::test::performance"

# PostgreSQL-specific wrapper functions for content operations
postgres::content::add() {
    local file="${1:-}"
    shift || true
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            *)
                if [[ -z "$file" ]]; then
                    file="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        log::error "File path required"
        echo "Usage: resource-postgres content add --file <sql-file>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${APP_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    postgres::database::execute_file "main" "$file"
}

postgres::content::list() {
    echo "PostgreSQL Databases and Instances:"
    postgres::instance::list
}

postgres::content::get() {
    local name="${1:-}"
    if [[ -z "$name" ]]; then
        log::error "Database or instance name required"
        echo "Usage: resource-postgres content get <instance-or-database>"
        return 1
    fi
    
    # Try as instance first, then as database
    if postgres::common::container_exists "$name"; then
        postgres::status::show_instance "$name"
    else
        postgres::database::list "main" | grep -i "$name" || {
            log::error "Instance or database not found: $name"
            return 1
        }
    fi
}

postgres::content::remove() {
    local name="${1:-}"
    if [[ -z "$name" ]]; then
        log::error "Database or instance name required"
        echo "Usage: resource-postgres content remove <instance-or-database>"
        return 1
    fi
    
    log::warn "This will permanently remove PostgreSQL content: $name"
    if ! flow::confirm "Are you sure?"; then
        return 1
    fi
    
    # Try as instance first, then as database
    if postgres::common::container_exists "$name"; then
        postgres::instance::destroy "$name"
    else
        postgres::database::drop "main" "$name"
    fi
}

postgres::content::execute() {
    local sql="${1:-}"
    local instance="${2:-main}"
    
    if [[ -z "$sql" ]]; then
        log::error "SQL query required"
        echo "Usage: resource-postgres content execute '<sql-query>' [instance]"
        return 1
    fi
    
    postgres::database::execute "$instance" "$sql"
}

# Performance test for PostgreSQL - validates the resource itself, not business use
postgres::test::performance() {
    log::info "Running PostgreSQL performance validation tests..."
    
    local instances=($(postgres::common::list_instances))
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::error "No PostgreSQL instances found for performance testing"
        return 1
    fi
    
    local test_instance="${instances[0]}"
    log::info "Testing performance of instance: $test_instance"
    
    # Test basic connection performance
    local start_time=$(date +%s%3N)
    if postgres::common::health_check "$test_instance"; then
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        log::info "Connection test: ${duration}ms"
        
        if [[ $duration -lt 1000 ]]; then
            log::success "Performance test passed: Connection under 1s"
            return 0
        else
            log::warn "Performance test warning: Connection took ${duration}ms"
            return 1
        fi
    else
        log::error "Performance test failed: Could not connect to PostgreSQL"
        return 1
    fi
}

# Credentials function using existing implementation
postgres::credentials() {
    # Delegate to existing implementation from backup CLI
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "postgres"; return 0; }
        return 1
    fi
    
    # Get all PostgreSQL containers (instances)
    local connections=()
    local overall_status="stopped"
    
    # Check for PostgreSQL containers
    if command -v docker &>/dev/null; then
        local container_prefix="${POSTGRES_CONTAINER_PREFIX:-vrooli-postgres}"
        local containers
        containers=$(docker ps -a --filter "name=^/${container_prefix}-" --format "{{.Names}}:{{.Status}}")
        
        if [[ -n "$containers" ]]; then
            while IFS=: read -r container_name container_status; do
                # Skip if container is not running
                if [[ ! "$container_status" =~ ^Up ]]; then
                    continue
                fi
                
                overall_status="running"
                
                # Extract instance info from container name
                local instance_name="${container_name#$container_prefix-}"
                
                # Get container port mapping
                local port_mapping
                port_mapping=$(docker port "$container_name" 5432 2>/dev/null | head -1)
                local host_port="${port_mapping##*:}"
                host_port="${host_port:-5432}"
                
                # Get environment variables for database info
                local db_name db_user db_password
                db_name=$(docker inspect "$container_name" --format '{{range .Config.Env}}{{if (index (split . "=") 0 | eq "POSTGRES_DB")}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                db_name="${db_name:-${POSTGRES_DEFAULT_DB:-postgres}}"
                
                db_user=$(docker inspect "$container_name" --format '{{range .Config.Env}}{{if (index (split . "=") 0 | eq "POSTGRES_USER")}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                db_user="${db_user:-${POSTGRES_DEFAULT_USER:-postgres}}"
                
                db_password=$(docker inspect "$container_name" --format '{{range .Config.Env}}{{if (index (split . "=") 0 | eq "POSTGRES_PASSWORD")}}{{index (split . "=") 1}}{{end}}{{end}}' 2>/dev/null)
                db_password="${db_password:-postgres}"
                
                # Create connection object
                local connection_obj
                connection_obj=$(jq -n -c \
                    --arg host "localhost" \
                    --arg port "$host_port" \
                    --arg database "$db_name" \
                    '{host: $host, port: ($port | tonumber), database: $database, ssl: false}')
                
                local auth_obj
                auth_obj=$(jq -n -c \
                    --arg username "$db_user" \
                    --arg password "$db_password" \
                    '{username: $username, password: $password}')
                
                local metadata_obj
                metadata_obj=$(jq -n -c \
                    --arg description "PostgreSQL instance: $instance_name" \
                    --argjson capabilities '["sql", "transactions", "acid"]' \
                    --arg version "16" \
                    '{description: $description, capabilities: $capabilities, version: $version}')
                
                local connection
                connection=$(credentials::build_connection \
                    "$instance_name" \
                    "PostgreSQL ($instance_name)" \
                    "postgres" \
                    "$connection_obj" \
                    "$auth_obj" \
                    "$metadata_obj")
                
                connections+=("$connection")
                
            done <<< "$containers"
        fi
    fi
    
    # Convert connections array to JSON
    local connections_array="[]"
    if [[ ${#connections[@]} -gt 0 ]]; then
        connections_array=$(printf '%s\n' "${connections[@]}" | jq -s '.')
    fi
    
    local response
    response=$(credentials::build_response "postgres" "$overall_status" "$connections_array")
    credentials::format_output "$response"
}

# Docker logs wrapper
postgres::docker::logs() {
    local instance="${1:-main}"
    local container_name="${POSTGRES_CONTAINER_PREFIX:-vrooli-postgres}-${instance}"
    
    if ! docker ps -a --format "{{.Names}}" | grep -q "^${container_name}$"; then
        log::error "PostgreSQL instance not found: $instance"
        return 1
    fi
    
    docker logs "$container_name" "${@:2}"
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi