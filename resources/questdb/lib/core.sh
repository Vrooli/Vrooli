#!/usr/bin/env bash
# QuestDB Core Functions

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Show credentials for n8n integration
questdb::core::credentials() {
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    # Ensure QuestDB configuration is loaded
    questdb::export_config
    
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