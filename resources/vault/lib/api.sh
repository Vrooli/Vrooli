#!/usr/bin/env bash
# Vault API Functions
# Secret management and API operations

#######################################
# Store a secret in Vault
# Arguments:
#   $1 - secret path
#   $2 - secret value
#   $3 - secret key (optional, defaults to 'value')
#######################################
vault::put_secret() {
    local path="" value="" key="value"
    
    # Support both named and positional arguments
    if [[ "${1:-}" == --* ]]; then
        # Parse named arguments
        while [[ $# -gt 0 ]]; do
            case "$1" in
                --path)
                    path="$2"
                    shift 2
                    ;;
                --value)
                    value="$2"
                    shift 2
                    ;;
                --key)
                    key="$2"
                    shift 2
                    ;;
                *)
                    log::error "Unknown argument: $1"
                    log::error "Usage: vault::put_secret --path <path> --value <value> [--key <key>]"
                    return 1
                    ;;
            esac
        done
    else
        # Traditional positional arguments - check if arguments exist
        path="${1:-}"
        value="${2:-}"
        key="${3:-value}"
    fi
    
    if [[ -z "$path" ]] || [[ -z "$value" ]]; then
        log::error "Usage: vault::put_secret <path> <value> [key]"
        log::error "   or: vault::put_secret --path <path> --value <value> [--key <key>]"
        return 1
    fi
    
    if ! vault::is_healthy || vault::is_sealed; then
        log::error "Vault is not ready (not running, unhealthy, or sealed)"
        return 1
    fi
    
    local full_path
    full_path=$(vault::construct_secret_path "$path") || return 1
    
    # Create JSON payload
    local payload
    payload=$(jq -n --arg key "$key" --arg value "$value" '{
        data: {
            ($key): $value
        }
    }')
    
    # Store secret
    local response
    response=$(vault::api_request "POST" "/v1/${full_path}" "$payload" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        vault::message "success" "MSG_VAULT_SECRET_PUT_SUCCESS"
        log::info "Stored secret at: $path"
        return 0
    else
        vault::message "error" "MSG_VAULT_SECRET_PUT_FAILED"
        log::error "Failed to store secret at: $path"
        return 1
    fi
}

#######################################
# Retrieve a secret from Vault
# Arguments:
#   $1 - secret path
#   $2 - secret key (optional, defaults to 'value')
#   $3 - output format (optional: 'json', 'raw', defaults to 'raw')
#######################################
vault::get_secret() {
    local path="" key="value" format="raw"
    
    # Support both named and positional arguments
    if [[ "${1:-}" == --* ]]; then
        # Parse named arguments
        while [[ $# -gt 0 ]]; do
            case "$1" in
                --path)
                    path="$2"
                    shift 2
                    ;;
                --key)
                    key="$2"
                    shift 2
                    ;;
                --format)
                    format="$2"
                    shift 2
                    ;;
                *)
                    log::error "Unknown argument: $1"
                    log::error "Usage: vault::get_secret --path <path> [--key <key>] [--format <format>]"
                    return 1
                    ;;
            esac
        done
    else
        # Traditional positional arguments - check if arguments exist
        path="${1:-}"
        key="${2:-value}"
        format="${3:-raw}"
    fi
    
    if [[ -z "$path" ]]; then
        log::error "Usage: vault::get_secret <path> [key] [format]"
        log::error "   or: vault::get_secret --path <path> [--key <key>] [--format <format>]"
        return 1
    fi
    
    if ! vault::is_healthy || vault::is_sealed; then
        log::error "Vault is not ready (not running, unhealthy, or sealed)"
        return 1
    fi
    
    local full_path
    full_path=$(vault::construct_secret_path "$path") || return 1
    
    # Retrieve secret
    local response
    response=$(vault::api_request "GET" "/v1/${full_path}" 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "$response" | jq -e '.data.data' >/dev/null 2>&1; then
        case "$format" in
            "json")
                echo "$response" | jq '.data.data'
                ;;
            "raw")
                echo "$response" | jq -r ".data.data.\"$key\" // empty"
                ;;
            *)
                log::error "Invalid format: $format (use 'json' or 'raw')"
                return 1
                ;;
        esac
        return 0
    else
        vault::message "error" "MSG_VAULT_SECRET_GET_FAILED"
        log::error "Failed to retrieve secret at: $path"
        return 1
    fi
}

#######################################
# List secrets at a path
# Arguments:
#   $1 - secret path (directory)
#   $2 - output format (optional: 'json', 'list', defaults to 'list')
#######################################
vault::list_secrets() {
    local path="" format="list"
    
    # Support both named and positional arguments
    if [[ "${1:-}" == --* ]]; then
        # Parse named arguments
        while [[ $# -gt 0 ]]; do
            case "$1" in
                --path)
                    path="$2"
                    shift 2
                    ;;
                --format)
                    format="$2"
                    shift 2
                    ;;
                *)
                    log::error "Unknown argument: $1"
                    log::error "Usage: vault::list_secrets --path <path> [--format <format>]"
                    return 1
                    ;;
            esac
        done
    else
        # Traditional positional arguments - check if arguments exist
        path="${1:-}"
        format="${2:-list}"
    fi
    
    # Default to root path if not specified (for v2.0 content list compatibility)
    if [[ -z "$path" ]]; then
        path="/"
    fi
    
    if ! vault::is_healthy || vault::is_sealed; then
        log::error "Vault is not ready (not running, unhealthy, or sealed)"
        return 1
    fi
    
    # Construct metadata path for listing
    local list_path="${VAULT_SECRET_ENGINE}/metadata"
    if [[ -n "$VAULT_NAMESPACE_PREFIX" ]]; then
        list_path="${list_path}/${VAULT_NAMESPACE_PREFIX}"
    fi
    list_path="${list_path}/${path%/}"  # Remove trailing slash
    
    # List secrets
    local response
    response=$(vault::api_request "LIST" "/v1/${list_path}" 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "$response" | jq -e '.data.keys' >/dev/null 2>&1; then
        case "$format" in
            "json")
                echo "$response" | jq '.data.keys'
                ;;
            "list")
                echo "$response" | jq -r '.data.keys[]'
                ;;
            *)
                log::error "Invalid format: $format (use 'json' or 'list')"
                return 1
                ;;
        esac
        return 0
    else
        log::error "Failed to list secrets at: $path"
        return 1
    fi
}

#######################################
# Delete a secret from Vault
# Arguments:
#   $1 - secret path
#######################################
vault::delete_secret() {
    local path=""
    
    # Support both named and positional arguments
    if [[ "${1:-}" == --* ]]; then
        # Parse named arguments
        while [[ $# -gt 0 ]]; do
            case "$1" in
                --path)
                    path="$2"
                    shift 2
                    ;;
                *)
                    log::error "Unknown argument: $1"
                    log::error "Usage: vault::delete_secret --path <path>"
                    return 1
                    ;;
            esac
        done
    else
        # Traditional positional arguments - check if arguments exist
        path="${1:-}"
    fi
    
    if [[ -z "$path" ]]; then
        log::error "Usage: vault::delete_secret <path>"
        log::error "   or: vault::delete_secret --path <path>"
        return 1
    fi
    
    if ! vault::is_healthy || vault::is_sealed; then
        log::error "Vault is not ready (not running, unhealthy, or sealed)"
        return 1
    fi
    
    local full_path
    full_path=$(vault::construct_secret_path "$path") || return 1
    
    # Delete secret
    local response
    response=$(vault::api_request "DELETE" "/v1/${full_path}" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        vault::message "success" "MSG_VAULT_SECRET_DELETE_SUCCESS"
        log::info "Deleted secret at: $path"
        return 0
    else
        vault::message "error" "MSG_VAULT_SECRET_DELETE_FAILED"
        log::error "Failed to delete secret at: $path"
        return 1
    fi
}

#######################################
# Check if a secret exists
# Arguments:
#   $1 - secret path
#######################################
vault::secret_exists() {
    local path="$1"
    
    if [[ -z "$path" ]]; then
        return 1
    fi
    
    vault::get_secret "$path" "value" "raw" >/dev/null 2>&1
}

#######################################
# Get secret metadata
# Arguments:
#   $1 - secret path
#######################################
vault::get_secret_metadata() {
    local path="${1:-}"
    
    if [[ -z "$path" ]]; then
        log::error "Usage: vault::get_secret_metadata <path>"
        return 1
    fi
    
    if ! vault::is_healthy || vault::is_sealed; then
        log::error "Vault is not ready (not running, unhealthy, or sealed)"
        return 1
    fi
    
    # Construct metadata path
    local metadata_path="${VAULT_SECRET_ENGINE}/metadata"
    if [[ -n "$VAULT_NAMESPACE_PREFIX" ]]; then
        metadata_path="${metadata_path}/${VAULT_NAMESPACE_PREFIX}"
    fi
    metadata_path="${metadata_path}/${path}"
    
    # Get metadata
    local response
    response=$(vault::api_request "GET" "/v1/${metadata_path}" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq '.data'
        return 0
    else
        log::error "Failed to get metadata for: $path"
        return 1
    fi
}

#######################################
# Create or update multiple secrets from JSON
# Arguments:
#   $1 - JSON string or file path containing secrets
#   $2 - base path prefix
#######################################
vault::bulk_put_secrets() {
    local json_input="$1"
    local base_path="${2:-}"
    
    if [[ -z "$json_input" ]]; then
        log::error "Usage: vault::bulk_put_secrets <json_input> [base_path]"
        return 1
    fi
    
    # Check if input is a file path
    local json_data
    if [[ -f "$json_input" ]]; then
        json_data=$(cat "$json_input")
    else
        json_data="$json_input"
    fi
    
    # Validate JSON
    if ! echo "$json_data" | jq empty 2>/dev/null; then
        log::error "Invalid JSON input"
        return 1
    fi
    
    local success_count=0
    local failed_count=0
    
    # Process each key-value pair
    while IFS=$'\t' read -r key value; do
        local path="$key"
        if [[ -n "$base_path" ]]; then
            path="${base_path}/${key}"
        fi
        
        if vault::put_secret "$path" "$value"; then
            ((success_count++))
        else
            ((failed_count++))
        fi
    done < <(echo "$json_data" | jq -r 'to_entries[] | [.key, .value] | @tsv')
    
    log::info "Bulk operation completed: $success_count successful, $failed_count failed"
    
    if [[ $failed_count -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Export secrets to JSON format
# Arguments:
#   $1 - base path to export
#   $2 - output file (optional, defaults to stdout)
#######################################
vault::export_secrets() {
    local base_path="$1"
    local output_file="${2:-}"
    
    if [[ -z "$base_path" ]]; then
        log::error "Usage: vault::export_secrets <base_path> [output_file]"
        return 1
    fi
    
    # Get list of secrets
    local secrets
    mapfile -t secrets < <(vault::list_secrets "$base_path" "list" 2>/dev/null)
    
    if [[ ${#secrets[@]} -eq 0 ]]; then
        log::warn "No secrets found at: $base_path"
        return 1
    fi
    
    # Create JSON object
    local json_output="{}"
    
    for secret in "${secrets[@]}"; do
        local secret_path="${base_path}/${secret}"
        local secret_data
        
        secret_data=$(vault::get_secret "$secret_path" "value" "json" 2>/dev/null)
        if [[ $? -eq 0 ]]; then
            json_output=$(echo "$json_output" | jq --arg key "$secret" --argjson value "$secret_data" '. + {($key): $value}')
        fi
    done
    
    # Output to file or stdout
    if [[ -n "$output_file" ]]; then
        echo "$json_output" | jq . > "$output_file"
        log::info "Exported secrets to: $output_file"
    else
        echo "$json_output" | jq .
    fi
}

#######################################
# Generate secure API key and store in Vault
# Arguments:
#   $1 - secret path
#   $2 - key length (optional, default 32)
#######################################
vault::generate_api_key() {
    local path="$1"
    local length="${2:-32}"
    
    if [[ -z "$path" ]]; then
        log::error "Usage: vault::generate_api_key <path> [length]"
        return 1
    fi
    
    local api_key
    api_key=$(vault::generate_random_string "$length")
    
    if vault::put_secret "$path" "$api_key"; then
        log::info "Generated and stored API key at: $path"
        return 0
    else
        log::error "Failed to store generated API key"
        return 1
    fi
}

#######################################
# Rotate a secret (generate new value and store)
# Arguments:
#   $1 - secret path
#   $2 - rotation type (random, password, api-key)
#   $3 - length (optional, default 32)
#######################################
vault::rotate_secret() {
    local path="$1"
    local rotation_type="$2"
    local length="${3:-32}"
    
    if [[ -z "$path" ]] || [[ -z "$rotation_type" ]]; then
        log::error "Usage: vault::rotate_secret <path> <type> [length]"
        log::error "Types: random, password, api-key"
        return 1
    fi
    
    local new_value
    case "$rotation_type" in
        "random")
            new_value=$(vault::generate_random_string "$length")
            ;;
        "password")
            new_value=$(openssl rand -base64 "$length" | tr -d "=+/" | head -c "$length")
            ;;
        "api-key")
            new_value="$(vault::generate_random_string 8)_$(vault::generate_random_string 24)"
            ;;
        *)
            log::error "Unknown rotation type: $rotation_type"
            return 1
            ;;
    esac
    
    if vault::put_secret "$path" "$new_value"; then
        log::info "Rotated secret at: $path"
        return 0
    else
        log::error "Failed to rotate secret at: $path"
        return 1
    fi
}