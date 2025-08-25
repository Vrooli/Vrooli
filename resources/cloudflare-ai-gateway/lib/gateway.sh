#!/bin/bash
# Cloudflare AI Gateway core functions

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/cloudflare-ai-gateway/lib"
RESOURCE_DIR="${APP_ROOT}/resources/cloudflare-ai-gateway"
DATA_DIR="${HOME}/Vrooli/data/cloudflare-ai-gateway"
CONFIG_FILE="${DATA_DIR}/config.json"
STATE_FILE="${DATA_DIR}/state.json"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/resources/lib/credentials-utils.sh"

# Initialize data directory
initialize_data_dir() {
    mkdir -p "${DATA_DIR}"
    mkdir -p "${DATA_DIR}/configs"
    mkdir -p "${DATA_DIR}/logs"
    
    # Initialize config if not exists
    if [[ ! -f "${CONFIG_FILE}" ]]; then
        echo '{"gateway_id": null, "active": false, "providers": []}' > "${CONFIG_FILE}"
    fi
    
    # Initialize state if not exists  
    if [[ ! -f "${STATE_FILE}" ]]; then
        echo '{"status": "inactive", "last_check": null}' > "${STATE_FILE}"
    fi
}

# Get Cloudflare API credentials from Vault
get_cloudflare_credentials() {
    local account_id=""
    local api_token=""
    
    # Try to get from Vault first
    if command -v resource-vault &>/dev/null; then
        account_id=$(resource-vault get cloudflare_account_id 2>/dev/null || echo "")
        api_token=$(resource-vault get cloudflare_api_token 2>/dev/null || echo "")
    fi
    
    # Fall back to environment variables
    if [[ -z "${account_id}" ]]; then
        account_id="${CLOUDFLARE_ACCOUNT_ID:-}"
    fi
    if [[ -z "${api_token}" ]]; then
        api_token="${CLOUDFLARE_API_TOKEN:-}"
    fi
    
    if [[ -z "${account_id}" ]] || [[ -z "${api_token}" ]]; then
        return 1
    fi
    
    echo "${account_id}:${api_token}"
}

# Make Cloudflare API request
cloudflare_api_request() {
    local method="${1}"
    local endpoint="${2}"
    local data="${3:-}"
    
    local creds
    if ! creds=$(get_cloudflare_credentials); then
        echo "Error: Cloudflare credentials not configured" >&2
        return 1
    fi
    
    local account_id="${creds%%:*}"
    local api_token="${creds##*:}"
    
    local url="https://api.cloudflare.com/client/v4/accounts/${account_id}/ai-gateway${endpoint}"
    
    local response
    if [[ -n "${data}" ]]; then
        response=$(curl -s -X "${method}" "${url}" \
            -H "Authorization: Bearer ${api_token}" \
            -H "Content-Type: application/json" \
            -d "${data}")
    else
        response=$(curl -s -X "${method}" "${url}" \
            -H "Authorization: Bearer ${api_token}")
    fi
    
    echo "${response}"
}

# Start/activate gateway
gateway_start() {
    initialize_data_dir
    
    local format="${1:-text}"
    
    # Check if already active
    local current_status
    current_status=$(jq -r '.status' "${STATE_FILE}")
    
    if [[ "${current_status}" == "active" ]]; then
        format_output "${format}" \
            '{"status": "already_active", "message": "Gateway is already active"}' \
            "Gateway is already active"
        return 0
    fi
    
    # Create or update gateway configuration
    local gateway_id
    gateway_id=$(jq -r '.gateway_id' "${CONFIG_FILE}")
    
    if [[ "${gateway_id}" == "null" ]]; then
        # Create new gateway
        local response
        response=$(cloudflare_api_request "POST" "/gateways" '{
            "name": "vrooli-ai-gateway",
            "slug": "vrooli",
            "rate_limiting": {
                "enabled": true,
                "requests_per_minute": 1000
            },
            "caching": {
                "enabled": true,
                "ttl": 3600
            }
        }')
        
        if [[ $? -ne 0 ]]; then
            format_output "${format}" \
                '{"status": "error", "message": "Failed to create gateway"}' \
                "Error: Failed to create gateway"
            return 1
        fi
        
        gateway_id=$(echo "${response}" | jq -r '.result.id')
        jq ".gateway_id = \"${gateway_id}\"" "${CONFIG_FILE}" > "${CONFIG_FILE}.tmp" && mv "${CONFIG_FILE}.tmp" "${CONFIG_FILE}"
    fi
    
    # Update state
    jq '.status = "active" | .last_check = now' "${STATE_FILE}" > "${STATE_FILE}.tmp" && mv "${STATE_FILE}.tmp" "${STATE_FILE}"
    
    format_output "${format}" \
        '{"status": "active", "gateway_id": "'"${gateway_id}"'", "message": "Gateway activated successfully"}' \
        "Gateway activated successfully (ID: ${gateway_id})"
}

# Stop/deactivate gateway
gateway_stop() {
    initialize_data_dir
    
    local format="${1:-text}"
    
    # Update state
    jq '.status = "inactive" | .last_check = now' "${STATE_FILE}" > "${STATE_FILE}.tmp" && mv "${STATE_FILE}.tmp" "${STATE_FILE}"
    
    format_output "${format}" \
        '{"status": "inactive", "message": "Gateway deactivated"}' \
        "Gateway deactivated"
}

# Check gateway status
gateway_status() {
    initialize_data_dir
    
    local format="${1:-text}"
    
    # Check credentials
    if ! get_cloudflare_credentials >/dev/null 2>&1; then
        format_output "${format}" \
            '{"status": "not_configured", "message": "Cloudflare credentials not configured"}' \
            "Status: Not Configured (missing Cloudflare credentials)"
        return 0
    fi
    
    # Check state
    local current_status
    current_status=$(jq -r '.status' "${STATE_FILE}")
    
    if [[ "${current_status}" != "active" ]]; then
        format_output "${format}" \
            '{"status": "inactive", "message": "Gateway is not active"}' \
            "Status: Inactive"
        return 0
    fi
    
    # Get gateway metrics
    local gateway_id
    gateway_id=$(jq -r '.gateway_id' "${CONFIG_FILE}")
    
    if [[ "${gateway_id}" != "null" ]]; then
        local response
        response=$(cloudflare_api_request "GET" "/gateways/${gateway_id}/analytics")
        
        if [[ $? -eq 0 ]]; then
            local total_requests=$(echo "${response}" | jq -r '.result.total_requests // 0')
            local cache_hits=$(echo "${response}" | jq -r '.result.cache_hits // 0')
            
            format_output "${format}" \
                "{\"status\": \"healthy\", \"gateway_id\": \"${gateway_id}\", \"total_requests\": ${total_requests}, \"cache_hits\": ${cache_hits}}" \
                "Status: Healthy
Gateway ID: ${gateway_id}
Total Requests: ${total_requests}
Cache Hits: ${cache_hits}"
        else
            format_output "${format}" \
                '{"status": "unknown", "message": "Failed to get gateway metrics"}' \
                "Status: Unknown (failed to get metrics)"
        fi
    else
        format_output "${format}" \
            '{"status": "not_initialized", "message": "Gateway not initialized"}' \
            "Status: Not Initialized"
    fi
}

# Get gateway info
gateway_info() {
    initialize_data_dir
    
    local format="${1:-text}"
    
    if [[ ! -f "${CONFIG_FILE}" ]]; then
        format_output "${format}" \
            '{"error": "No configuration found"}' \
            "No configuration found"
        return 1
    fi
    
    local config
    config=$(cat "${CONFIG_FILE}")
    
    format_output "${format}" \
        "${config}" \
        "Gateway Configuration:
$(echo "${config}" | jq .)"
}

# View gateway logs
gateway_logs() {
    initialize_data_dir
    
    local tail_lines="${1:-50}"
    local log_file="${DATA_DIR}/logs/gateway.log"
    
    if [[ ! -f "${log_file}" ]]; then
        echo "No logs available yet"
        return 0
    fi
    
    tail -n "${tail_lines}" "${log_file}"
}

# Configure gateway settings
gateway_configure() {
    initialize_data_dir
    
    local provider=""
    local cache_ttl=""
    local rate_limit=""
    
    while [[ $# -gt 0 ]]; do
        case "${1}" in
            --provider)
                provider="${2}"
                shift 2
                ;;
            --cache-ttl)
                cache_ttl="${2}"
                shift 2
                ;;
            --rate-limit)
                rate_limit="${2}"
                shift 2
                ;;
            *)
                echo "Unknown option: ${1}" >&2
                return 1
                ;;
        esac
    done
    
    # Update configuration
    local config
    config=$(cat "${CONFIG_FILE}")
    
    if [[ -n "${provider}" ]]; then
        # Add provider to list if not exists
        config=$(echo "${config}" | jq ".providers += [\"${provider}\"] | .providers |= unique")
    fi
    
    if [[ -n "${cache_ttl}" ]]; then
        config=$(echo "${config}" | jq ".cache_ttl = ${cache_ttl}")
    fi
    
    if [[ -n "${rate_limit}" ]]; then
        config=$(echo "${config}" | jq ".rate_limit = ${rate_limit}")
    fi
    
    echo "${config}" > "${CONFIG_FILE}"
    echo "Configuration updated successfully"
}

# Content management functions
handle_content_add() {
    initialize_data_dir
    
    local file=""
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "${1}" in
            --file)
                file="${2}"
                shift 2
                ;;
            --name)
                name="${2}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "${file}" ]]; then
        echo "Error: --file is required" >&2
        return 1
    fi
    
    if [[ ! -f "${file}" ]]; then
        echo "Error: File not found: ${file}" >&2
        return 1
    fi
    
    if [[ -z "${name}" ]]; then
        name=$(basename "${file}" .json)
    fi
    
    cp "${file}" "${DATA_DIR}/configs/${name}.json"
    echo "Configuration '${name}' added successfully"
}

handle_content_list() {
    initialize_data_dir
    
    local format="${1:-text}"
    
    local configs=()
    for config_file in "${DATA_DIR}/configs"/*.json; do
        if [[ -f "${config_file}" ]]; then
            configs+=("$(basename "${config_file}" .json)")
        fi
    done
    
    if [[ ${#configs[@]} -eq 0 ]]; then
        format_output "${format}" \
            '{"configs": []}' \
            "No configurations found"
    else
        local json_array=$(printf '"%s",' "${configs[@]}" | sed 's/,$//')
        format_output "${format}" \
            "{\"configs\": [${json_array}]}" \
            "Configurations:
$(printf '%s\n' "${configs[@]}")"
    fi
}

handle_content_get() {
    initialize_data_dir
    
    local name="${1}"
    
    if [[ -z "${name}" ]]; then
        echo "Error: Configuration name required" >&2
        return 1
    fi
    
    local config_file="${DATA_DIR}/configs/${name}.json"
    
    if [[ ! -f "${config_file}" ]]; then
        echo "Error: Configuration '${name}' not found" >&2
        return 1
    fi
    
    cat "${config_file}"
}

handle_content_remove() {
    initialize_data_dir
    
    local name="${1}"
    
    if [[ -z "${name}" ]]; then
        echo "Error: Configuration name required" >&2
        return 1
    fi
    
    local config_file="${DATA_DIR}/configs/${name}.json"
    
    if [[ ! -f "${config_file}" ]]; then
        echo "Error: Configuration '${name}' not found" >&2
        return 1
    fi
    
    rm "${config_file}"
    echo "Configuration '${name}' removed successfully"
}

handle_content_execute() {
    initialize_data_dir
    
    local name="${1}"
    
    if [[ -z "${name}" ]]; then
        echo "Error: Configuration name required" >&2
        return 1
    fi
    
    local config_file="${DATA_DIR}/configs/${name}.json"
    
    if [[ ! -f "${config_file}" ]]; then
        echo "Error: Configuration '${name}' not found" >&2
        return 1
    fi
    
    # Apply configuration to gateway
    local gateway_id
    gateway_id=$(jq -r '.gateway_id' "${CONFIG_FILE}")
    
    if [[ "${gateway_id}" == "null" ]]; then
        echo "Error: Gateway not initialized. Run 'start' first." >&2
        return 1
    fi
    
    local response
    response=$(cloudflare_api_request "PATCH" "/gateways/${gateway_id}" "$(cat "${config_file}")")
    
    if [[ $? -eq 0 ]]; then
        echo "Configuration '${name}' applied successfully"
    else
        echo "Error: Failed to apply configuration" >&2
        return 1
    fi
}