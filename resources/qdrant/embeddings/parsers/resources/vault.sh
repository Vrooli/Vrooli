#!/usr/bin/env bash
# HashiCorp Vault Parser for Qdrant Embeddings
# Extracts semantic information from Vault configuration and policy files
#
# Handles:
# - Secret engine configurations
# - Policy definitions and permissions
# - Authentication method configurations
# - Key-value store structures
# - Dynamic secret configurations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract Vault configuration
# 
# Gets basic Vault instance configuration
#
# Arguments:
#   $1 - Path to Vault config file (JSON/HCL)
# Returns: JSON with Vault configuration
#######################################
extractor::lib::vault::extract_config() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local file_ext="${file##*.}"
    local config_data="{}"
    
    # Handle different file formats
    case "$file_ext" in
        json)
            if ! jq empty "$file" 2>/dev/null; then
                log::debug "Invalid JSON format in Vault config: $file" >&2
                return 1
            fi
            config_data=$(jq '.' "$file" 2>/dev/null)
            ;;
        hcl|tf)
            # Basic HCL parsing - extract key configuration blocks
            config_data="{}"
            ;;
        *)
            # Try to parse as JSON first, then give up
            if jq empty "$file" 2>/dev/null; then
                config_data=$(jq '.' "$file" 2>/dev/null)
            else
                return 1
            fi
            ;;
    esac
    
    # Extract key configuration elements
    local vault_addr=$(echo "$config_data" | jq -r '.vault_addr // .address // ""' 2>/dev/null)
    local ui_enabled=$(echo "$config_data" | jq -r '.ui // false' 2>/dev/null)
    local api_addr=$(echo "$config_data" | jq -r '.api_addr // ""' 2>/dev/null)
    local cluster_addr=$(echo "$config_data" | jq -r '.cluster_addr // ""' 2>/dev/null)
    
    # Extract storage backend
    local storage_type=""
    local storage_config="{}"
    if echo "$config_data" | jq -e '.storage' >/dev/null 2>/dev/null; then
        storage_type=$(echo "$config_data" | jq -r '.storage | keys[0] // "unknown"' 2>/dev/null)
        storage_config=$(echo "$config_data" | jq -c '.storage' 2>/dev/null)
    fi
    
    # Extract listener configuration
    local listeners=0
    local tls_enabled="false"
    if echo "$config_data" | jq -e '.listener' >/dev/null 2>/dev/null; then
        listeners=$(echo "$config_data" | jq '.listener | length' 2>/dev/null || echo "0")
        if echo "$config_data" | jq -e '.listener[] | select(has("tls_cert_file") or has("tls_key_file"))' >/dev/null 2>/dev/null; then
            tls_enabled="true"
        fi
    fi
    
    jq -n \
        --arg addr "$vault_addr" \
        --arg api_addr "$api_addr" \
        --arg cluster_addr "$cluster_addr" \
        --arg ui "$ui_enabled" \
        --arg storage_type "$storage_type" \
        --argjson storage_config "$storage_config" \
        --arg listeners "$listeners" \
        --arg tls "$tls_enabled" \
        '{
            vault_address: $addr,
            api_address: $api_addr,
            cluster_address: $cluster_addr,
            ui_enabled: ($ui == "true"),
            storage_backend: $storage_type,
            storage_config: $storage_config,
            listener_count: ($listeners | tonumber),
            tls_enabled: ($tls == "true")
        }'
}

#######################################
# Extract secret engines
# 
# Analyzes configured secret engines and mounts
#
# Arguments:
#   $1 - Path to Vault secrets config file
# Returns: JSON with secret engine information
#######################################
extractor::lib::vault::extract_secret_engines() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local engines=()
    local kv_stores=0
    local database_engines=0
    local pki_engines=0
    local transit_engines=0
    local aws_engines=0
    
    # Extract secret engine configurations
    local content=$(cat "$file" 2>/dev/null)
    
    # Parse JSON if it's JSON format
    if jq empty "$file" 2>/dev/null; then
        # Extract mount configurations
        if jq -e '.secret_engines // .mounts // .mount' "$file" >/dev/null 2>/dev/null; then
            local mounts=$(jq -c '.secret_engines // .mounts // .mount' "$file" 2>/dev/null)
            
            echo "$mounts" | jq -c 'to_entries[]' 2>/dev/null | while IFS= read -r mount; do
                local mount_path=$(echo "$mount" | jq -r '.key // ""')
                local mount_type=$(echo "$mount" | jq -r '.value.type // ""')
                
                engines+=("$mount_type")
                
                case "$mount_type" in
                    "kv"|"kv-v1"|"kv-v2")
                        ((kv_stores++))
                        ;;
                    "database")
                        ((database_engines++))
                        ;;
                    "pki")
                        ((pki_engines++))
                        ;;
                    "transit")
                        ((transit_engines++))
                        ;;
                    "aws")
                        ((aws_engines++))
                        ;;
                esac
            done
        fi
    else
        # Basic text parsing for HCL/other formats
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"kv'; then
            ((kv_stores++))
            engines+=("kv")
        fi
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"database'; then
            ((database_engines++))
            engines+=("database")
        fi
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"pki'; then
            ((pki_engines++))
            engines+=("pki")
        fi
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"transit'; then
            ((transit_engines++))
            engines+=("transit")
        fi
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"aws'; then
            ((aws_engines++))
            engines+=("aws")
        fi
    fi
    
    local engines_json="[]"
    if [[ ${#engines[@]} -gt 0 ]]; then
        engines_json=$(printf '%s\n' "${engines[@]}" | sort -u | jq -R . | jq -s '.')
    fi
    
    jq -n \
        --argjson engines "$engines_json" \
        --arg kv_count "$kv_stores" \
        --arg db_count "$database_engines" \
        --arg pki_count "$pki_engines" \
        --arg transit_count "$transit_engines" \
        --arg aws_count "$aws_engines" \
        '{
            secret_engines: $engines,
            kv_stores: ($kv_count | tonumber),
            database_engines: ($db_count | tonumber),
            pki_engines: ($pki_count | tonumber),
            transit_engines: ($transit_count | tonumber),
            aws_engines: ($aws_count | tonumber),
            total_engines: (($engines | length))
        }'
}

#######################################
# Extract policies
# 
# Analyzes Vault policies and permissions
#
# Arguments:
#   $1 - Path to Vault policy file
# Returns: JSON with policy information
#######################################
extractor::lib::vault::extract_policies() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local policy_name=$(basename "$file" | sed 's/\.[^.]*$//')
    local content=$(cat "$file" 2>/dev/null)
    
    # Count policy rules/paths
    local path_count=0
    local capabilities=()
    local paths=()
    
    # Parse JSON policy format
    if jq empty "$file" 2>/dev/null; then
        if jq -e '.path // .rule' "$file" >/dev/null 2>/dev/null; then
            path_count=$(jq '.path // .rule | keys | length' "$file" 2>/dev/null || echo "0")
            
            # Extract capabilities
            while IFS= read -r cap; do
                [[ -z "$cap" ]] && continue
                capabilities+=("$cap")
            done < <(jq -r '.path // .rule | .. | .capabilities[]? // empty' "$file" 2>/dev/null | sort -u)
            
            # Extract paths
            while IFS= read -r path; do
                [[ -z "$path" ]] && continue
                paths+=("$path")
            done < <(jq -r '.path // .rule | keys[]' "$file" 2>/dev/null)
        fi
    else
        # Parse HCL policy format
        path_count=$(echo "$content" | grep -c '^path ' 2>/dev/null || echo "0")
        
        # Extract capabilities from HCL
        while IFS= read -r cap_line; do
            [[ -z "$cap_line" ]] && continue
            local caps=$(echo "$cap_line" | sed -E 's/.*capabilities[[:space:]]*=[[:space:]]*\[([^\]]+)\].*/\1/' | tr ',' '\n' | sed 's/[" ]//g')
            while IFS= read -r cap; do
                [[ -n "$cap" ]] && capabilities+=("$cap")
            done <<< "$caps"
        done < <(echo "$content" | grep -E 'capabilities[[:space:]]*=' 2>/dev/null)
        
        # Extract paths from HCL
        while IFS= read -r path_line; do
            local path=$(echo "$path_line" | sed -E 's/^path[[:space:]]*"([^"]+)".*/\1/')
            [[ -n "$path" ]] && paths+=("$path")
        done < <(echo "$content" | grep -E '^path ' 2>/dev/null)
    fi
    
    # Analyze policy scope
    local has_admin_access="false"
    local has_read_only="false"
    local has_secret_access="false"
    local has_auth_access="false"
    
    for cap in "${capabilities[@]}"; do
        case "$cap" in
            "sudo"|"root")
                has_admin_access="true"
                ;;
            "read"|"list")
                has_read_only="true"
                ;;
        esac
    done
    
    for path in "${paths[@]}"; do
        if [[ "$path" == *"secret/"* ]] || [[ "$path" == *"kv/"* ]]; then
            has_secret_access="true"
        elif [[ "$path" == *"auth/"* ]]; then
            has_auth_access="true"
        fi
    done
    
    local capabilities_json="[]"
    local paths_json="[]"
    [[ ${#capabilities[@]} -gt 0 ]] && capabilities_json=$(printf '%s\n' "${capabilities[@]}" | sort -u | jq -R . | jq -s '.')
    [[ ${#paths[@]} -gt 0 ]] && paths_json=$(printf '%s\n' "${paths[@]}" | jq -R . | jq -s '.')
    
    jq -n \
        --arg name "$policy_name" \
        --arg path_count "$path_count" \
        --argjson capabilities "$capabilities_json" \
        --argjson paths "$paths_json" \
        --arg admin "$has_admin_access" \
        --arg readonly "$has_read_only" \
        --arg secrets "$has_secret_access" \
        --arg auth "$has_auth_access" \
        '{
            policy_name: $name,
            path_count: ($path_count | tonumber),
            capabilities: $capabilities,
            paths: $paths,
            has_admin_access: ($admin == "true"),
            has_read_only_access: ($readonly == "true"),
            has_secret_access: ($secrets == "true"),
            has_auth_access: ($auth == "true")
        }'
}

#######################################
# Extract authentication methods
# 
# Analyzes configured auth methods
#
# Arguments:
#   $1 - Path to Vault auth config file
# Returns: JSON with authentication information
#######################################
extractor::lib::vault::extract_auth_methods() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local auth_methods=()
    local content=$(cat "$file" 2>/dev/null)
    
    # Parse JSON format
    if jq empty "$file" 2>/dev/null; then
        if jq -e '.auth // .auth_methods' "$file" >/dev/null 2>/dev/null; then
            while IFS= read -r method; do
                [[ -z "$method" ]] && continue
                auth_methods+=("$method")
            done < <(jq -r '.auth // .auth_methods | to_entries[] | .value.type // .key' "$file" 2>/dev/null)
        fi
    else
        # Parse HCL/text format
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"userpass'; then
            auth_methods+=("userpass")
        fi
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"ldap'; then
            auth_methods+=("ldap")
        fi
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"jwt'; then
            auth_methods+=("jwt")
        fi
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"kubernetes'; then
            auth_methods+=("kubernetes")
        fi
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"aws'; then
            auth_methods+=("aws")
        fi
        if echo "$content" | grep -qE 'type[[:space:]]*=[[:space:]]*"github'; then
            auth_methods+=("github")
        fi
    fi
    
    local methods_json="[]"
    if [[ ${#auth_methods[@]} -gt 0 ]]; then
        methods_json=$(printf '%s\n' "${auth_methods[@]}" | sort -u | jq -R . | jq -s '.')
    fi
    
    jq -n \
        --argjson methods "$methods_json" \
        '{
            auth_methods: $methods,
            method_count: ($methods | length)
        }'
}

#######################################
# Analyze Vault purpose
# 
# Determines Vault usage based on configuration
#
# Arguments:
#   $1 - Path to Vault config file
# Returns: JSON with purpose analysis
#######################################
extractor::lib::vault::analyze_purpose() {
    local file="$1"
    local purposes=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"purposes": [], "primary_purpose": "unknown"}'
        return
    fi
    
    local filename=$(basename "$file" | tr '[:upper:]' '[:lower:]')
    local content=$(cat "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]')
    
    # Analyze filename for hints
    if [[ "$filename" == *"config"* ]]; then
        purposes+=("vault_configuration")
    elif [[ "$filename" == *"policy"* ]]; then
        purposes+=("access_control")
    elif [[ "$filename" == *"secret"* ]]; then
        purposes+=("secret_management")
    elif [[ "$filename" == *"auth"* ]]; then
        purposes+=("authentication")
    fi
    
    # Analyze content patterns
    if echo "$content" | grep -qE 'type.*=.*"kv'; then
        purposes+=("key_value_storage")
    fi
    
    if echo "$content" | grep -qE 'type.*=.*"database'; then
        purposes+=("dynamic_database_credentials")
    fi
    
    if echo "$content" | grep -qE 'type.*=.*"pki'; then
        purposes+=("certificate_authority")
    fi
    
    if echo "$content" | grep -qE 'type.*=.*"transit'; then
        purposes+=("encryption_as_service")
    fi
    
    if echo "$content" | grep -qE 'type.*=.*"aws'; then
        purposes+=("aws_credential_management")
    fi
    
    if echo "$content" | grep -qE 'capabilities.*=.*\["sudo"'; then
        purposes+=("admin_policy")
    fi
    
    if echo "$content" | grep -qE 'path.*"auth/'; then
        purposes+=("authentication_management")
    fi
    
    if echo "$content" | grep -qE 'seal.*"auto'; then
        purposes+=("auto_unsealing")
    fi
    
    if echo "$content" | grep -qE 'ha_enabled|cluster'; then
        purposes+=("high_availability")
    fi
    
    # Determine primary purpose
    local primary_purpose="secret_management"
    if [[ ${#purposes[@]} -gt 0 ]]; then
        primary_purpose="${purposes[0]}"
    fi
    
    local purposes_json=$(printf '%s\n' "${purposes[@]}" | sort -u | jq -R . | jq -s '.')
    
    jq -n \
        --argjson purposes "$purposes_json" \
        --arg primary "$primary_purpose" \
        '{
            purposes: $purposes,
            primary_purpose: $primary
        }'
}

#######################################
# Extract all Vault information
# 
# Main extraction function that combines all analyses
#
# Arguments:
#   $1 - Vault config file path or directory
#   $2 - Component type (config, policy, auth, etc.)
#   $3 - Resource name
# Returns: JSON lines with all Vault information
#######################################
extractor::lib::vault::extract_all() {
    local path="$1"
    local component_type="${2:-config}"
    local resource_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        local file_ext="${filename##*.}"
        
        # Check if it's a supported file type
        case "$file_ext" in
            json|hcl|conf|policy|tf)
                ;;
            *)
                return 1
                ;;
        esac
        
        # Get file statistics
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        
        # Determine what type of Vault config this is
        local config_type="unknown"
        if [[ "$filename" == *"policy"* ]]; then
            config_type="policy"
        elif [[ "$filename" == *"auth"* ]]; then
            config_type="auth"
        elif [[ "$filename" == *"secret"* ]] || [[ "$filename" == *"engine"* ]]; then
            config_type="secret_engine"
        else
            config_type="main_config"
        fi
        
        # Extract appropriate components based on config type
        local config="{}"
        local secret_engines="{}"
        local policies="{}"
        local auth_methods="{}"
        local purpose=""
        
        case "$config_type" in
            "policy")
                policies=$(extractor::lib::vault::extract_policies "$file")
                ;;
            "auth")
                auth_methods=$(extractor::lib::vault::extract_auth_methods "$file")
                ;;
            "secret_engine")
                secret_engines=$(extractor::lib::vault::extract_secret_engines "$file")
                ;;
            *)
                config=$(extractor::lib::vault::extract_config "$file")
                secret_engines=$(extractor::lib::vault::extract_secret_engines "$file")
                auth_methods=$(extractor::lib::vault::extract_auth_methods "$file")
                ;;
        esac
        
        purpose=$(extractor::lib::vault::analyze_purpose "$file")
        
        # Get key metrics
        local primary_purpose=$(echo "$purpose" | jq -r '.primary_purpose')
        
        # Build content summary based on config type
        local content="Vault $config_type: $filename | Type: $component_type | Resource: $resource_name"
        content="$content | Purpose: $primary_purpose"
        
        case "$config_type" in
            "policy")
                local path_count=$(echo "$policies" | jq -r '.path_count // 0')
                [[ $path_count -gt 0 ]] && content="$content | Paths: $path_count"
                ;;
            "secret_engine")
                local engine_count=$(echo "$secret_engines" | jq -r '.total_engines // 0')
                [[ $engine_count -gt 0 ]] && content="$content | Engines: $engine_count"
                ;;
            "auth")
                local method_count=$(echo "$auth_methods" | jq -r '.method_count // 0')
                [[ $method_count -gt 0 ]] && content="$content | Auth Methods: $method_count"
                ;;
            *)
                local tls_enabled=$(echo "$config" | jq -r '.tls_enabled // false')
                [[ "$tls_enabled" == "true" ]] && content="$content | TLS Enabled"
                ;;
        esac
        
        # Output comprehensive Vault analysis
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg config_type "$config_type" \
            --arg file_size "$file_size" \
            --argjson config "$config" \
            --argjson secret_engines "$secret_engines" \
            --argjson policies "$policies" \
            --argjson auth_methods "$auth_methods" \
            --argjson purpose "$purpose" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    filename: $filename,
                    component_type: $component_type,
                    config_type: $config_type,
                    vault_type: "hashicorp_vault",
                    file_size: ($file_size | tonumber),
                    config: $config,
                    secret_engines: $secret_engines,
                    policies: $policies,
                    auth_methods: $auth_methods,
                    purpose: $purpose,
                    content_type: "vault_config",
                    extraction_method: "vault_parser"
                }
            }' | jq -c
            
    elif [[ -d "$path" ]]; then
        # Directory - find all Vault config files
        local config_files=()
        while IFS= read -r file; do
            config_files+=("$file")
        done < <(find "$path" -type f \( -name "*.json" -o -name "*.hcl" -o -name "*.conf" -o -name "*.policy" \) 2>/dev/null)
        
        if [[ ${#config_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        for file in "${config_files[@]}"; do
            extractor::lib::vault::extract_all "$file" "$component_type" "$resource_name"
        done
    fi
}

#######################################
# Check if file is a Vault configuration
# 
# Validates if file is a Vault config/policy file
#
# Arguments:
#   $1 - File path
# Returns: 0 if Vault config, 1 otherwise
#######################################
extractor::lib::vault::is_vault_config() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local filename=$(basename "$file")
    local content=$(cat "$file" 2>/dev/null)
    
    # Check filename patterns
    if [[ "$filename" == *"vault"* ]] || [[ "$filename" == *.policy ]]; then
        return 0
    fi
    
    # Check content patterns
    if echo "$content" | grep -qE '(vault|secret_engine|auth_method|policy)'; then
        return 0
    fi
    
    # Check for Vault-specific configuration keys
    if echo "$content" | grep -qE '(storage|listener|seal|api_addr|cluster_addr)'; then
        return 0
    fi
    
    return 1
}

# Export all functions
export -f extractor::lib::vault::extract_config
export -f extractor::lib::vault::extract_secret_engines
export -f extractor::lib::vault::extract_policies
export -f extractor::lib::vault::extract_auth_methods
export -f extractor::lib::vault::analyze_purpose
export -f extractor::lib::vault::extract_all
export -f extractor::lib::vault::is_vault_config