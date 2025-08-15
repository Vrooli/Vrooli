#!/usr/bin/env bash
################################################################################
# QuestDB Resource CLI
# 
# Lightweight CLI interface for QuestDB using the CLI Command Framework
#
# Usage:
#   resource-questdb <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (handle symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    REAL_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    QUESTDB_CLI_DIR="$(cd "$(dirname "$REAL_SCRIPT")" && pwd)"
else
    QUESTDB_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/resources/lib/cli-command-framework.sh"

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

# Initialize CLI framework
cli::init "questdb" "QuestDB high-performance time-series database management"

# Override help to provide QuestDB-specific examples
cli::register_command "help" "Show this help message with QuestDB examples" "resource_cli::show_help"

# Register additional QuestDB-specific commands
cli::register_command "inject" "Inject SQL/CSV/JSON data into QuestDB" "resource_cli::inject" "modifies-system"
cli::register_command "query" "Execute SQL query" "resource_cli::query"
cli::register_command "tables" "List or create tables" "resource_cli::tables"
cli::register_command "api" "Make API request" "resource_cli::api"
cli::register_command "console" "Open web console in browser" "resource_cli::console"
cli::register_command "logs" "Show container logs" "resource_cli::logs"
cli::register_command "credentials" "Show n8n credentials for QuestDB" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall QuestDB (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject data into QuestDB
resource_cli::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-questdb inject <file.sql|file.csv|file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-questdb inject data.sql"
        echo "  resource-questdb inject shared:initialization/storage/questdb/sample.csv"
        echo "  resource-questdb inject time_series.json"
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
            log::error "Unsupported file type: $extension (supported: sql, csv, json)"
            return 1
            ;;
    esac
}

# Validate QuestDB configuration
resource_cli::validate() {
    if command -v questdb::validate &>/dev/null; then
        questdb::validate
    elif command -v questdb::status::health_check &>/dev/null; then
        questdb::status::health_check
    else
        # Basic validation
        log::header "Validating QuestDB"
        local container_name="${QUESTDB_CONTAINER_NAME:-questdb}"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name" || {
            log::error "QuestDB container not running"
            return 1
        }
        log::success "QuestDB is running"
    fi
}

# Show QuestDB status
resource_cli::status() {
    if command -v questdb::status::check &>/dev/null; then
        questdb::status::check
    else
        # Basic status
        log::header "QuestDB Status"
        local container_name="${QUESTDB_CONTAINER_NAME:-questdb}"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=$container_name" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start QuestDB
resource_cli::start() {
    if command -v questdb::docker::start &>/dev/null; then
        questdb::docker::start
    else
        local container_name="${QUESTDB_CONTAINER_NAME:-questdb}"
        docker start "$container_name" || log::error "Failed to start QuestDB"
    fi
}

# Stop QuestDB
resource_cli::stop() {
    if command -v questdb::docker::stop &>/dev/null; then
        questdb::docker::stop
    else
        local container_name="${QUESTDB_CONTAINER_NAME:-questdb}"
        docker stop "$container_name" || log::error "Failed to stop QuestDB"
    fi
}

# Install QuestDB
resource_cli::install() {
    if command -v questdb::install::run &>/dev/null; then
        questdb::install::run
    else
        log::error "questdb::install::run not available"
        return 1
    fi
}

# Uninstall QuestDB
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove QuestDB and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v questdb::uninstall::run &>/dev/null; then
        questdb::uninstall::run
    else
        local container_name="${QUESTDB_CONTAINER_NAME:-questdb}"
        docker stop "$container_name" 2>/dev/null || true
        docker rm "$container_name" 2>/dev/null || true
        log::success "QuestDB uninstalled"
    fi
}

# Show credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "questdb"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${QUESTDB_CONTAINER_NAME:-questdb}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local connections=()
        
        # PostgreSQL-compatible connection
        local pg_connection_obj
        pg_connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${QUESTDB_PG_PORT:-8812}" \
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
            --arg username "${QUESTDB_PG_USER:-admin}" \
            --arg password "${QUESTDB_PG_PASSWORD:-quest}" \
            '{
                username: $username,
                password: $password
            }')
        
        local pg_metadata_obj
        pg_metadata_obj=$(jq -n \
            --arg description "QuestDB PostgreSQL-compatible interface" \
            '{
                description: $description,
                protocol: "postgresql"
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
            --argjson port "${QUESTDB_HTTP_PORT:-9000}" \
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
            --arg console_url "${QUESTDB_BASE_URL:-http://localhost:9000}" \
            --argjson ilp_port "${QUESTDB_ILP_PORT:-9009}" \
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
        
        local connections_json
        connections_json=$(printf '%s\n' "${connections[@]}" | jq -s '.')
        connections_array="$connections_json"
    fi
    
    local response
    response=$(credentials::build_response "questdb" "$status" "$connections_array")
    credentials::format_output "$response"
}

# Execute SQL query
resource_cli::query() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        log::error "SQL query required"
        echo "Usage: resource-questdb query \"SELECT * FROM table\""
        echo ""
        echo "Examples:"
        echo "  resource-questdb query \"SELECT * FROM trades LIMIT 10\""
        echo "  resource-questdb query \"SHOW TABLES\""
        echo "  resource-questdb query \"SELECT count(*) FROM sensors WHERE timestamp > now() - 1h\""
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
resource_cli::tables() {
    local action="${1:-list}"
    local table_name="${2:-}"
    local schema_file="${3:-}"
    
    case "$action" in
        list)
            if command -v questdb::tables::list &>/dev/null; then
                questdb::tables::list
            else
                # Fallback to SHOW TABLES query
                resource_cli::query "SHOW TABLES"
            fi
            ;;
        create)
            if [[ -z "$table_name" ]]; then
                log::error "Table name required for create"
                echo "Usage: resource-questdb tables create <table_name> [schema_file]"
                echo ""
                echo "Examples:"
                echo "  resource-questdb tables create sensors"
                echo "  resource-questdb tables create trades schema.sql"
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
resource_cli::api() {
    local endpoint="${1:-}"
    local method="${2:-GET}"
    local data="${3:-}"
    
    if [[ -z "$endpoint" ]]; then
        log::error "API endpoint required"
        echo "Usage: resource-questdb api <endpoint> [method] [data]"
        echo ""
        echo "Examples:"
        echo "  resource-questdb api /exec GET"
        echo "  resource-questdb api /exec POST '{\"query\":\"SELECT * FROM trades\"}'"
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
resource_cli::console() {
    if command -v questdb::open_console &>/dev/null; then
        questdb::open_console
    else
        local console_url="${QUESTDB_BASE_URL:-http://localhost:9000}"
        echo "Opening QuestDB console at: $console_url"
        
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
resource_cli::logs() {
    local lines="${1:-50}"
    local follow="${2:-false}"
    local container_name="${QUESTDB_CONTAINER_NAME:-questdb}"
    
    if [[ "$follow" == "true" ]]; then
        docker logs -f --tail "$lines" "$container_name" 2>/dev/null || {
            log::error "Failed to show logs - is QuestDB running?"
            return 1
        }
    else
        docker logs --tail "$lines" "$container_name" 2>/dev/null || {
            log::error "Failed to show logs - is QuestDB running?"
            return 1
        }
    fi
}

# Custom help function with QuestDB-specific examples
resource_cli::show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add QuestDB-specific examples
    echo ""
    echo "📊 QuestDB Time-Series Database Examples:"
    echo ""
    echo "Data Management:"
    echo "  resource-questdb inject data.sql                    # Import SQL schema"
    echo "  resource-questdb inject shared:init/questdb/sample.csv  # Import CSV data"
    echo "  resource-questdb inject time_series.json           # Import JSON data"
    echo ""
    echo "SQL Queries:"
    echo "  resource-questdb query \"SHOW TABLES\""
    echo "  resource-questdb query \"SELECT * FROM trades LIMIT 10\""
    echo "  resource-questdb query \"SELECT count(*) FROM sensors WHERE timestamp > now() - 1h\""
    echo ""
    echo "Table Operations:"
    echo "  resource-questdb tables list                       # List all tables"
    echo "  resource-questdb tables create sensors schema.sql # Create table with schema"
    echo ""
    echo "API & Console:"
    echo "  resource-questdb api /exec GET                     # Make API request"
    echo "  resource-questdb console                           # Open web UI"
    echo "  resource-questdb logs 100 true                     # Follow logs"
    echo ""
    echo "Time-Series Features:"
    echo "  • High-performance ingestion (millions of rows/sec)"
    echo "  • PostgreSQL wire protocol compatibility"
    echo "  • ANSI SQL with time-series extensions"
    echo "  • Real-time analytics and dashboards"
    echo ""
    echo "Default Ports: HTTP 9000, PostgreSQL 8812, ILP 9009"
    echo "Web Console: http://localhost:9000"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi