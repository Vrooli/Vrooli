#!/usr/bin/env bash
# Qdrant Credentials Management
# Functions for generating n8n credentials and connection information

set -euo pipefail

# Get directory of this script
QDRANT_CREDENTIALS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required utilities
# shellcheck disable=SC1091
source "${QDRANT_CREDENTIALS_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"

# Source configuration
# shellcheck disable=SC1091
source "${QDRANT_CREDENTIALS_DIR}/../config/defaults.sh" 2>/dev/null || true

#######################################
# Generate n8n credentials for Qdrant
# Arguments:
#   $1 - Resource status (running/stopped/unknown)
# Outputs: JSON credentials array
# Returns: 0 on success, 1 on failure
#######################################
qdrant::credentials::generate() {
    local status="${1:-unknown}"
    
    local connections_array="[]"
    
    if [[ "$status" == "running" ]]; then
        local connection_json
        
        if [[ -n "${QDRANT_API_KEY:-}" ]]; then
            # With API key authentication
            connection_json=$(jq -n \
                --arg id "api" \
                --arg name "Qdrant REST API" \
                --arg n8n_credential_type "httpHeaderAuth" \
                --arg host "localhost" \
                --arg port "${QDRANT_PORT:-6333}" \
                --arg path "/collections" \
                --arg description "Qdrant vector database REST API" \
                --arg version "${QDRANT_VERSION:-latest}" \
                --arg header_name "api-key" \
                --arg header_value "${QDRANT_API_KEY}" \
                '{
                    id: $id,
                    name: $name,
                    n8n_credential_type: $n8n_credential_type,
                    connection: {
                        host: $host,
                        port: ($port | tonumber),
                        path: $path,
                        ssl: false
                    },
                    auth: {
                        header_name: $header_name,
                        header_value: $header_value
                    },
                    metadata: {
                        description: $description,
                        capabilities: ["vectors", "search", "clustering", "embeddings"],
                        version: $version
                    }
                }')
        else
            # Without authentication
            connection_json=$(jq -n \
                --arg id "api" \
                --arg name "Qdrant REST API" \
                --arg n8n_credential_type "httpRequest" \
                --arg host "localhost" \
                --arg port "${QDRANT_PORT:-6333}" \
                --arg path "/collections" \
                --arg description "Qdrant vector database REST API" \
                --arg version "${QDRANT_VERSION:-latest}" \
                '{
                    id: $id,
                    name: $name,
                    n8n_credential_type: $n8n_credential_type,
                    connection: {
                        host: $host,
                        port: ($port | tonumber),
                        path: $path,
                        ssl: false
                    },
                    metadata: {
                        description: $description,
                        capabilities: ["vectors", "search", "clustering", "embeddings"],
                        version: $version
                    }
                }')
        fi
        
        connections_array=$(echo "$connection_json" | jq -s '.')
    fi
    
    echo "$connections_array"
}

#######################################
# Get formatted credentials response
# Arguments:
#   $1 - Resource status (running/stopped/unknown)
# Outputs: Complete credentials response JSON
# Returns: 0 on success, 1 on failure
#######################################
qdrant::credentials::get() {
    local status="${1:-unknown}"
    
    # Source credentials utilities
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    local connections_array
    connections_array=$(qdrant::credentials::generate "$status")
    
    local response
    response=$(credentials::build_response "qdrant" "$status" "$connections_array")
    
    echo "$response"
}

#######################################
# Show formatted credentials output
# Arguments:
#   $1 - Resource status (running/stopped/unknown)
# Outputs: Formatted credentials display
# Returns: 0 on success, 1 on failure
#######################################
qdrant::credentials::show() {
    local status="${1:-unknown}"
    
    local response
    response=$(qdrant::credentials::get "$status")
    
    # Source credentials utilities for formatting
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    credentials::format_output "$response"
}