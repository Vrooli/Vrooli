#!/usr/bin/env bash
################################################################################
# QuestDB Resource CLI
# 
# Lightweight CLI interface for QuestDB that delegates to existing lib functions.
#
# Usage:
#   resource-questdb <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory and resolve VROOLI_ROOT
QUESTDB_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Handle both direct execution and symlink execution
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this script is a symlink, resolve to the original location
    REAL_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    QUESTDB_CLI_DIR="$(cd "$(dirname "$REAL_SCRIPT")" && pwd)"
fi

VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$QUESTDB_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$QUESTDB_CLI_DIR"
export QUESTDB_SCRIPT_DIR="$QUESTDB_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Source questdb configuration
# shellcheck disable=SC1091
source "${QUESTDB_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
questdb::export_config 2>/dev/null || true

# Source questdb libraries
for lib in common docker api status tables; do
    lib_file="${QUESTDB_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "questdb"

################################################################################
# Delegate to existing questdb functions
################################################################################

# Inject data into questdb
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-questdb inject <file.sql|file.csv|file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would inject: $file"
        return 0
    fi
    
    # Use existing injection function based on file type
    local extension="${file##*.}"
    case "$extension" in
        sql)
            if command -v questdb::api::query_file &>/dev/null; then
                questdb::api::query_file "$file"
            else
                log::error "SQL injection not available"
                return 1
            fi
            ;;
        csv|json)
            if command -v questdb::inject_data &>/dev/null; then
                questdb::inject_data "$file"
            else
                log::error "Data injection not available"
                return 1
            fi
            ;;
        *)
            log::error "Unsupported file type: $extension"
            return 1
            ;;
    esac
}

# Validate questdb configuration
resource_cli::validate() {
    if command -v questdb::validate &>/dev/null; then
        questdb::validate
    elif command -v questdb::status::health_check &>/dev/null; then
        questdb::status::health_check
    else
        # Basic validation
        log::header "Validating QuestDB"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "questdb" || {
            log::error "QuestDB container not running"
            return 1
        }
        log::success "QuestDB is running"
    fi
}

# Show questdb status
resource_cli::status() {
    if command -v questdb::status::check &>/dev/null; then
        questdb::status::check
    else
        # Basic status
        log::header "QuestDB Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "questdb"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=questdb" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start questdb
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start questdb"
        return 0
    fi
    
    if command -v questdb::docker::start &>/dev/null; then
        questdb::docker::start
    else
        docker start questdb || log::error "Failed to start questdb"
    fi
}

# Stop questdb
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop questdb"
        return 0
    fi
    
    if command -v questdb::docker::stop &>/dev/null; then
        questdb::docker::stop
    else
        docker stop questdb || log::error "Failed to stop questdb"
    fi
}

# Install questdb
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install questdb"
        return 0
    fi
    
    if command -v questdb::install::run &>/dev/null; then
        questdb::install::run
    else
        log::error "questdb::install::run not available"
        return 1
    fi
}

# Uninstall questdb
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove questdb and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall questdb"
        return 0
    fi
    
    if command -v questdb::uninstall::run &>/dev/null; then
        questdb::uninstall::run
    else
        docker stop questdb 2>/dev/null || true
        docker rm questdb 2>/dev/null || true
        log::success "questdb uninstalled"
    fi
}

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "questdb"; return 0; }
        return 1
    fi
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "$QUESTDB_CONTAINER_NAME")
    
    # Build connections array
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local connections=()
        
        # PostgreSQL-compatible connection
        local pg_connection_obj
        pg_connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "$QUESTDB_PG_PORT" \
            --arg database "qdb" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                database: $database,
                ssl: $ssl
            }')
        
        local pg_auth_obj
        pg_auth_obj=$(jq -n \
            --arg username "$QUESTDB_PG_USER" \
            --arg password "$QUESTDB_PG_PASSWORD" \
            '{
                username: $username,
                password: $password
            }')
        
        local pg_metadata_obj
        pg_metadata_obj=$(jq -n \
            --arg description "QuestDB PostgreSQL-compatible interface" \
            --argjson default_tables "$(printf '%s\n' "${QUESTDB_DEFAULT_TABLES[@]}" | jq -R . | jq -s .)" \
            '{
                description: $description,
                protocol: "postgresql",
                default_tables: $default_tables
            }')
        
        local pg_connection
        pg_connection=$(credentials::build_connection \
            "postgres" \
            "QuestDB PostgreSQL Interface" \
            "postgres" \
            "$pg_connection_obj" \
            "$pg_auth_obj" \
            "$pg_metadata_obj")
        
        connections+=("$pg_connection")
        
        # HTTP API connection (for time-series data ingestion)
        local http_connection_obj
        http_connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "$QUESTDB_HTTP_PORT" \
            --arg path "/exec" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local http_metadata_obj
        http_metadata_obj=$(jq -n \
            --arg description "QuestDB HTTP REST API for queries and data ingestion" \
            --arg console_url "$QUESTDB_BASE_URL" \
            --argjson ilp_port "$QUESTDB_ILP_PORT" \
            '{
                description: $description,
                console_url: $console_url,
                ilp_port: $ilp_port,
                protocol: "http"
            }')
        
        local http_connection
        http_connection=$(credentials::build_connection \
            "http" \
            "QuestDB HTTP API" \
            "httpRequest" \
            "$http_connection_obj" \
            "{}" \
            "$http_metadata_obj")
        
        connections+=("$http_connection")
        
        # Build array from connections
        local connections_json
        connections_json=$(printf '%s\n' "${connections[@]}" | jq -s '.')
        connections_array="$connections_json"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "questdb" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

################################################################################
# QuestDB-specific commands (if functions exist)
################################################################################

# Execute SQL query
questdb_query() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        log::error "SQL query required"
        echo "Usage: resource-questdb query \"SELECT * FROM table\""
        return 1
    fi
    
    if command -v questdb::api::query &>/dev/null; then
        questdb::api::query "$query"
    else
        log::error "Query functionality not available"
        return 1
    fi
}

# List or create tables
questdb_tables() {
    local action="${1:-list}"
    local table_name="${2:-}"
    local schema_file="${3:-}"
    
    case "$action" in
        list)
            if command -v questdb::tables::list &>/dev/null; then
                questdb::tables::list
            else
                log::error "Table listing not available"
                return 1
            fi
            ;;
        create)
            if [[ -z "$table_name" ]]; then
                log::error "Table name required for create"
                echo "Usage: resource-questdb tables create <table_name> [schema_file]"
                return 1
            fi
            
            if command -v questdb::tables::create &>/dev/null; then
                if [[ -n "$schema_file" ]]; then
                    questdb::tables::create "$table_name" "$schema_file"
                else
                    questdb::tables::create "$table_name"
                fi
            else
                log::error "Table creation not available"
                return 1
            fi
            ;;
        *)
            log::error "Unknown table action: $action"
            echo "Usage: resource-questdb tables [list|create] [table_name] [schema_file]"
            return 1
            ;;
    esac
}

# Make API request
questdb_api() {
    local endpoint="${1:-}"
    local method="${2:-GET}"
    local data="${3:-}"
    
    if [[ -z "$endpoint" ]]; then
        log::error "API endpoint required"
        echo "Usage: resource-questdb api <endpoint> [method] [data]"
        return 1
    fi
    
    if command -v questdb::api::request &>/dev/null; then
        questdb::api::request "$method" "$endpoint" "$data"
    else
        log::error "API functionality not available"
        return 1
    fi
}

# Open web console
questdb_console() {
    if command -v questdb::open_console &>/dev/null; then
        questdb::open_console
    else
        # Basic console opening
        local console_url="${QUESTDB_BASE_URL:-http://localhost:9000}"
        echo "Opening QuestDB console at: $console_url"
        
        # Try to open in browser
        if command -v xdg-open &>/dev/null; then
            xdg-open "$console_url" 2>/dev/null &
        elif command -v open &>/dev/null; then
            open "$console_url" 2>/dev/null &
        else
            echo "Please open $console_url in your browser"
        fi
    fi
}

# Show logs
questdb_logs() {
    local lines="${1:-50}"
    local follow="${2:-false}"
    
    if [[ "$follow" == "true" ]]; then
        docker logs -f --tail "$lines" questdb 2>/dev/null || {
            log::error "Failed to show logs - is questdb running?"
            return 1
        }
    else
        docker logs --tail "$lines" questdb 2>/dev/null || {
            log::error "Failed to show logs - is questdb running?"
            return 1
        }
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 QuestDB Resource CLI

USAGE:
    resource-questdb <command> [options]

CORE COMMANDS:
    inject <file>               Inject SQL/CSV/JSON data into questdb
    validate                    Validate questdb configuration
    status                      Show questdb status
    start                       Start questdb container
    stop                        Stop questdb container
    install                     Install questdb
    uninstall                   Uninstall questdb (requires --force)
    credentials                 Get connection credentials for n8n integration
    
QUESTDB COMMANDS:
    query "<sql>"               Execute SQL query
    tables [list|create]        List or create tables
    api <endpoint> [method]     Make API request
    console                     Open web console in browser
    logs [lines] [follow]       Show container logs

OPTIONS:
    --verbose, -v               Show detailed output
    --dry-run                   Preview actions without executing
    --force                     Force operation (skip confirmations)

EXAMPLES:
    resource-questdb status
    resource-questdb query "SELECT * FROM trades LIMIT 10"
    resource-questdb tables list
    resource-questdb tables create my_table schema.sql
    resource-questdb inject shared:initialization/storage/questdb/data.sql
    resource-questdb console
    resource-questdb logs 100 true

For more information: https://docs.vrooli.com/resources/questdb
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "$@"
            ;;
            
        # QuestDB-specific commands
        query)
            questdb_query "$@"
            ;;
        tables)
            questdb_tables "$@"
            ;;
        api)
            questdb_api "$@"
            ;;
        console)
            questdb_console "$@"
            ;;
        logs)
            questdb_logs "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi