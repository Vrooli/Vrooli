#!/usr/bin/env bash
set -euo pipefail

# Shared Secrets Management Library
# Provides centralized secret storage, retrieval, and template substitution
# Following Vrooli's 3-layer secret resolution strategy:
# 1. HashiCorp Vault (production)
# 2. Project .vrooli/secrets.json (development)  
# 3. Environment variables (fallback)

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../utils/var.sh"

# Detect project root dynamically
secrets::get_project_root() {
    local current_dir
    current_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    # Walk up directory tree looking for .vrooli directory
    while [[ "$current_dir" != "/" ]]; do
        if [[ -d "$current_dir/.vrooli" ]]; then
            echo "$current_dir"
            return 0
        fi
        current_dir="$(dirname "$current_dir")"
    done
    
    # Fallback: use var_ROOT_DIR if available
    echo "${var_ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)}"
}

# Get paths to secret files
secrets::get_secrets_file() {
    echo "${var_VROOLI_CONFIG_DIR:-$(secrets::get_project_root)/.vrooli}/secrets.json"
}

secrets::get_service_file() {
    echo "${var_SERVICE_JSON_FILE:-$(secrets::get_project_root)/.vrooli/service.json}"
}

#######################################
# Resolve secret using 3-layer fallback strategy
# Arguments:
#   $1 - secret key name
#   $2 - vault path (optional, defaults to "vrooli")
# Returns:
#   0 - success, secret printed to stdout
#   1 - secret not found in any layer
#######################################
secrets::resolve() {
    local key="$1"
    local vault_path="${2:-vrooli}"
    local vault_manager="${var_SCRIPTS_RESOURCES_DIR:-$(secrets::get_project_root)/scripts/resources}/storage/vault/manage.sh"
    
    # Layer 1: Vault (if available and healthy)
    if [[ -x "$vault_manager" ]]; then
        local vault_secret
        if vault_secret=$("$vault_manager" --action get-secret --path "$vault_path" --key "$key" --format raw 2>/dev/null); then
            if [[ -n "$vault_secret" ]]; then
                echo "$vault_secret"
                return 0
            fi
        fi
    fi
    
    # Layer 2: Project .vrooli/secrets.json
    local secrets_file
    secrets_file="$(secrets::get_secrets_file)"
    if [[ -f "$secrets_file" ]]; then
        local json_secret
        if json_secret=$(jq -r ".$key // empty" "$secrets_file" 2>/dev/null); then
            if [[ -n "$json_secret" && "$json_secret" != "null" ]]; then
                echo "$json_secret"
                return 0
            fi
        fi
    fi
    
    # Layer 3: Environment variables
    local env_secret
    if env_secret=$(printenv "$key" 2>/dev/null); then
        if [[ -n "$env_secret" ]]; then
            echo "$env_secret"
            return 0
        fi
    fi
    
    return 1
}

#######################################
# SOURCE RESOURCE MANAGEMENT UTILITIES
#######################################

# Source the port registry directly for service resolution
secrets::source_port_registry() {
    local port_registry="${var_SCRIPTS_RESOURCES_DIR:-$(secrets::get_project_root)/scripts/resources}/port_registry.sh"
    
    # Check if RESOURCE_PORTS array is already populated
    if [[ "${#RESOURCE_PORTS[@]}" -gt 0 ]] 2>/dev/null; then
        return 0
    fi
    
    # Source port registry if it exists
    if [[ -f "$port_registry" ]]; then
        # shellcheck disable=SC1090
        source "$port_registry"
    fi
}

#######################################
# Resolve service reference to actual URL/port
# Arguments:
#   $1 - resource name (e.g., "ollama", "n8n")
#   $2 - property type (e.g., "url", "port", "host")
# Returns:
#   0 - success, value printed to stdout
#   1 - resource/property not found
#######################################
secrets::resolve_service_reference() {
    local resource="$1"
    local property="${2:-url}"
    
    # Validate inputs
    if [[ -z "$resource" ]]; then
        return 1
    fi
    
    # Ensure port registry is sourced
    secrets::source_port_registry
    
    # Check if RESOURCE_PORTS array is available and populated
    if [[ "${#RESOURCE_PORTS[@]}" -eq 0 ]] 2>/dev/null; then
        return 1
    fi
    
    case "$property" in
        "url")
            # Get port directly from RESOURCE_PORTS array
            local port="${RESOURCE_PORTS[$resource]:-}"
            if [[ -n "$port" ]]; then
                echo "http://localhost:$port"
                return 0
            fi
            ;;
        "port")
            # Get port directly from RESOURCE_PORTS array
            local port="${RESOURCE_PORTS[$resource]:-}"
            if [[ -n "$port" ]]; then
                echo "$port"
                return 0
            fi
            ;;
        "host")
            echo "localhost"
            return 0
            ;;
        *)
            return 1
            ;;
    esac
    
    return 1
}

#######################################
# Substitute service references in JSON configuration
# Arguments:
#   $1 - JSON string with ${service.RESOURCE.PROPERTY} placeholders
# Returns:
#   JSON with service references resolved
#######################################
secrets::substitute_service_references() {
    # Temporarily disable unbound variable check for this function
    local old_set_u="$(set +o | grep nounset || true)"
    set +u
    
    local json_input
    # Handle both parameter and piped input
    if [[ $# -gt 0 ]]; then
        json_input="$1"
    else
        json_input="$(cat)"
    fi
    local json_output="$json_input"
    
    # Ensure port registry is sourced
    secrets::source_port_registry
    
    # Check if RESOURCE_PORTS array is available and populated
    if [[ "${#RESOURCE_PORTS[@]}" -eq 0 ]]; then
        echo "WARNING: RESOURCE_PORTS not available, service references will not be resolved" >&2
        echo "$json_output"
        # Restore original set options
        [[ "$old_set_u" =~ "nounset" ]] && set -u
        return 0
    fi
    
    # Find all ${service.RESOURCE.PROPERTY} patterns
    local service_patterns=()
    while IFS= read -r pattern; do
        [[ -n "$pattern" ]] && service_patterns+=("$pattern")
    done < <(echo "$json_input" | grep -o '\${service\.[^}]*}' | sort -u || true)
    
    for pattern in "${service_patterns[@]}"; do
        # Extract service reference by removing ${ and }
        local service_ref="${pattern#\$\{}"
        service_ref="${service_ref%\}}"
        
        # Parse service.RESOURCE.PROPERTY format
        if [[ "$service_ref" =~ ^service\.([^.]+)\.([^.]+)$ ]]; then
            local resource="${BASH_REMATCH[1]}"
            local property="${BASH_REMATCH[2]}"
            local service_value=""
            
            # Resolve service reference directly
            case "$property" in
                "url")
                    local port="${RESOURCE_PORTS[$resource]:-}"
                    if [[ -n "$port" ]]; then
                        service_value="http://localhost:$port"
                    fi
                    ;;
                "port")
                    service_value="${RESOURCE_PORTS[$resource]:-}"
                    ;;
                "host")
                    service_value="localhost"
                    ;;
            esac
            
            if [[ -n "$service_value" ]]; then
                # Replace the pattern in JSON using | as delimiter to avoid issues with /
                json_output="${json_output//$pattern/$service_value}"
            else
                # Source logging utility if available
                if command -v log::warn >/dev/null 2>&1; then
                    log::warn "Service reference not found: $service_ref (pattern: $pattern)"
                else
                    echo "WARNING: Service reference not found: $service_ref (pattern: $pattern)" >&2
                fi
                # Keep the placeholder - don't fail the entire operation
            fi
        else
            # Invalid service reference format
            if command -v log::warn >/dev/null 2>&1; then
                log::warn "Invalid service reference format: $service_ref (pattern: $pattern)"
            else
                echo "WARNING: Invalid service reference format: $service_ref (pattern: $pattern)" >&2
            fi
        fi
    done
    
    # Restore original set options
    [[ "$old_set_u" =~ "nounset" ]] && set -u
    
    echo "$json_output"
}

#######################################
# Substitute secrets in JSON configuration
# Arguments:
#   $1 - JSON string with {{SECRET_NAME}} placeholders
# Returns:
#   JSON with secrets resolved
#######################################
secrets::substitute_json() {
    local json_input
    # Handle both parameter and piped input
    if [[ $# -gt 0 ]]; then
        json_input="$1"
    else
        json_input="$(cat)"
    fi
    local json_output="$json_input"
    
    # Find all {{SECRET_NAME}} patterns using a simpler approach
    local secret_patterns
    mapfile -t secret_patterns < <(echo "$json_input" | grep -o '{{[^}]*}}' | sort -u)
    
    for pattern in "${secret_patterns[@]}"; do
        # Extract secret name by removing {{ and }}
        local secret_name="${pattern#\{\{}"
        secret_name="${secret_name%\}\}}"
        local secret_value
        
        if secret_value=$(secrets::resolve "$secret_name"); then
            # Escape the secret value for JSON
            local escaped_value
            escaped_value=$(echo "$secret_value" | jq -R .)
            # Remove the quotes added by jq -R
            escaped_value="${escaped_value#\"}"
            escaped_value="${escaped_value%\"}"
            
            # Replace the pattern in JSON
            json_output="${json_output//$pattern/$escaped_value}"
        else
            # Source logging utility if available
            if command -v log::warn >/dev/null 2>&1; then
                log::warn "Secret not found: $secret_name (pattern: $pattern)"
            else
                echo "WARNING: Secret not found: $secret_name (pattern: $pattern)" >&2
            fi
            # Keep the placeholder - don't fail the entire operation
        fi
    done
    
    echo "$json_output"
}

#######################################
# Comprehensive template substitution for all patterns
# Handles both secrets ({{SECRET}}) and service references (${service.resource.property})
# Arguments:
#   $1 - JSON string with templates
# Returns:
#   JSON with all templates resolved
#######################################
secrets::substitute_all_templates() {
    local json_input
    # Handle both parameter and piped input
    if [[ $# -gt 0 ]]; then
        json_input="$1"
    else
        json_input="$(cat)"
    fi
    
    # Process service references first (${service.X.Y})
    local processed_json
    processed_json=$(echo "$json_input" | secrets::substitute_service_references)
    
    # Then process secrets ({{SECRET_NAME}})
    processed_json=$(echo "$processed_json" | secrets::substitute_json)
    
    echo "$processed_json"
}

#######################################
# Save API key to project secrets.json
# Arguments:
#   $1 - secret key name (e.g., "N8N_API_KEY") 
#   $2 - secret value
# Returns:
#   0 - success
#   1 - error
#######################################
secrets::save_key() {
    local key="$1"
    local value="$2"
    local secrets_file
    secrets_file="$(secrets::get_secrets_file)"
    
    # Validate inputs
    if [[ -z "$key" ]]; then
        echo "ERROR: Secret key name is required" >&2
        return 1
    fi
    
    if [[ -z "$value" ]]; then
        echo "ERROR: Secret value is required" >&2
        return 1
    fi
    
    # Ensure secrets directory exists
    local secrets_dir
    secrets_dir="$(dirname "$secrets_file")"
    mkdir -p "$secrets_dir"
    
    # Load existing secrets or create new structure
    local secrets_json
    if [[ -f "$secrets_file" ]]; then
        # Backup existing file
        cp "$secrets_file" "${secrets_file}.backup" 2>/dev/null || true
        secrets_json=$(cat "$secrets_file")
    else
        # Create default structure
        secrets_json='{
  "_metadata": {
    "environment": "development",
    "last_updated": "",
    "notes": "Development secrets - production uses Vault"
  }
}'
    fi
    
    # Update timestamp and add/update the secret
    local updated_json
    updated_json=$(echo "$secrets_json" | jq --arg key "$key" --arg value "$value" --arg timestamp "$(date -Iseconds)" '
        ._metadata.last_updated = $timestamp |
        .[$key] = $value
    ')
    
    # Write updated secrets with secure permissions
    echo "$updated_json" > "$secrets_file"
    chmod 600 "$secrets_file"
    
    return 0
}

#######################################
# Update service.json to use template substitution
# Arguments:
#   $1 - service path (e.g., "automation.n8n")
#   $2 - key to update (e.g., "apiKey")
#   $3 - template variable name (e.g., "N8N_API_KEY")
# Returns:
#   0 - success
#   1 - error
#######################################
secrets::set_template() {
    local service_path="$1"
    local key="$2"
    local template_var="$3"
    local service_file
    service_file="$(secrets::get_service_file)"
    
    # Validate inputs
    if [[ -z "$service_path" ]] || [[ -z "$key" ]] || [[ -z "$template_var" ]]; then
        echo "ERROR: All parameters required: service_path, key, template_var" >&2
        return 1
    fi
    
    # Check if service.json exists
    if [[ ! -f "$service_file" ]]; then
        echo "ERROR: Service configuration file not found: $service_file" >&2
        return 1
    fi
    
    # Backup existing file
    cp "$service_file" "${service_file}.backup" 2>/dev/null || true
    
    # Update service.json with template substitution
    local updated_json
    local jq_path
    jq_path=".resources.${service_path}.${key}"
    local template_value="{{$template_var}}"
    
    updated_json=$(jq --arg path "$jq_path" --arg value "$template_value" '
        setpath($path | split("."); $value)
    ' "$service_file")
    
    # Write updated service configuration
    echo "$updated_json" > "$service_file"
    
    return 0
}

#######################################
# Get API key using 3-layer resolution
# Arguments:
#   $1 - secret key name
# Returns:
#   0 - success, key printed to stdout
#   1 - key not found
#######################################
secrets::get_key() {
    secrets::resolve "$1"
}

#######################################
# Validate secret storage locations and permissions
# Returns:
#   0 - validation passed
#   1 - validation failed
#######################################
secrets::validate_storage() {
    local secrets_file
    secrets_file="$(secrets::get_secrets_file)"
    local issues=0
    
    # Check if secrets.json exists and has proper permissions
    if [[ -f "$secrets_file" ]]; then
        local perms
        perms=$(stat -c "%a" "$secrets_file" 2>/dev/null || echo "000")
        if [[ "$perms" != "600" ]]; then
            echo "WARNING: Secrets file has incorrect permissions: $perms (should be 600)" >&2
            echo "Fix with: chmod 600 $secrets_file" >&2
            ((issues++))
        fi
    fi
    
    # Check for secrets in wrong locations (user home vs project root)
    local user_service_file="$HOME/.vrooli/service.json"
    if [[ -f "$user_service_file" ]]; then
        # Check if it contains actual secrets (not templates)
        if grep -q '"apiKey":\s*"[^{]' "$user_service_file" 2>/dev/null; then
            echo "WARNING: Found API keys in wrong location: $user_service_file" >&2
            echo "These should be moved to: $secrets_file" >&2
            ((issues++))
        fi
    fi
    
    return $issues
}

#######################################
# Clean up secrets from wrong locations
# Arguments:
#   $1 - "dry-run" for preview only, anything else for actual cleanup
# Returns:
#   0 - success
#   1 - error
#######################################
secrets::cleanup_wrong_locations() {
    local mode="${1:-}"
    local user_service_file="$HOME/.vrooli/service.json"
    local cleaned=0
    
    if [[ -f "$user_service_file" ]]; then
        echo "Checking for secrets in wrong location: $user_service_file"
        
        # Extract any API keys that look like actual secrets (not templates)
        local secrets_found
        secrets_found=$(jq -r '
            .. | 
            objects | 
            to_entries[] | 
            select(.key == "apiKey" and (.value | type) == "string" and (.value | startswith("{{") | not)) |
            "\(.key)=\(.value)"
        ' "$user_service_file" 2>/dev/null || echo "")
        
        if [[ -n "$secrets_found" ]]; then
            echo "Found secrets that should be moved:"
            echo "$secrets_found"
            
            if [[ "$mode" != "dry-run" ]]; then
                echo "Moving secrets would require manual intervention to determine correct variable names"
                echo "Please run migration script instead"
            fi
        fi
    fi
    
    return 0
}

#######################################
# CONFIGURATION MANAGEMENT FUNCTIONS
# Functions for handling project configuration files
#######################################

#######################################
# Get the project configuration directory path
# Returns: path to project root .vrooli directory
#######################################
secrets::get_project_config_dir() {
    echo "${var_VROOLI_CONFIG_DIR:-$(secrets::get_project_root)/.vrooli}"
}

#######################################
# Get the correct project configuration file path
# Returns: path to project root service.json
#######################################
secrets::get_project_config_file() {
    echo "${var_SERVICE_JSON_FILE:-$(secrets::get_project_root)/.vrooli/service.json}"
}

#######################################
# Get the incorrect user configuration file path (for migration)
# Returns: path to user home service.json
#######################################
secrets::get_user_config_file() {
    echo "$HOME/.vrooli/service.json"
}

#######################################
# Check if a resource is using wrong config location
# Arguments:
#   $1 - resource script path or function context
# Returns:
#   0 - using correct location
#   1 - using wrong location
#######################################
secrets::validate_config_location() {
    local context="${1:-unknown}"
    local user_config_file
    user_config_file="$(secrets::get_user_config_file)"
    local project_config_file  
    project_config_file="$(secrets::get_project_config_file)"
    
    # Check if resource is still referencing user home config
    if [[ -f "$user_config_file" ]]; then
        echo "WARNING: $context - Found config in wrong location: $user_config_file" >&2
        echo "Should be in project location: $project_config_file" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Migrate configuration from user home to project root
# Arguments:
#   $1 - resource name (for logging)
#   $2 - dry-run mode ("dry-run" or anything else for actual migration)
# Returns:
#   0 - success or no migration needed
#   1 - error during migration
#######################################
secrets::migrate_config() {
    local resource_name="${1:-unknown}"
    local dry_run="${2:-}"
    local user_config_file
    user_config_file="$(secrets::get_user_config_file)"
    local project_config_file
    project_config_file="$(secrets::get_project_config_file)"
    local project_config_dir
    project_config_dir="$(dirname "$project_config_file")"
    
    # Check if migration is needed
    if [[ ! -f "$user_config_file" ]]; then
        if command -v log::info >/dev/null 2>&1; then
            log::info "$resource_name: No user config found, migration not needed"
        else
            echo "INFO: $resource_name: No user config found, migration not needed"
        fi
        return 0
    fi
    
    if [[ "$dry_run" == "dry-run" ]]; then
        echo "DRY RUN: Would migrate $resource_name config:"
        echo "  From: $user_config_file"
        echo "  To: $project_config_file"
        return 0
    fi
    
    # Ensure project config directory exists
    mkdir -p "$project_config_dir"
    
    # If project config already exists, merge instead of overwrite
    if [[ -f "$project_config_file" ]]; then
        if command -v log::warn >/dev/null 2>&1; then
            log::warn "$resource_name: Project config exists, manual merge may be needed"
        else
            echo "WARNING: $resource_name: Project config exists, manual merge may be needed" >&2
        fi
        
        # Create backup of both configs
        cp "$project_config_file" "${project_config_file}.backup.$(date +%s)" 2>/dev/null || true
        cp "$user_config_file" "${user_config_file}.backup.$(date +%s)" 2>/dev/null || true
        
        return 1  # Let user handle merge manually
    fi
    
    # Move config from user to project location
    if cp "$user_config_file" "$project_config_file"; then
        # Set proper permissions
        chmod 644 "$project_config_file"
        
        # Remove old config after successful copy
        rm -f "$user_config_file"
        
        if command -v log::success >/dev/null 2>&1; then
            log::success "$resource_name: Config migrated successfully"
        else
            echo "SUCCESS: $resource_name: Config migrated successfully"
        fi
        
        return 0
    else
        if command -v log::error >/dev/null 2>&1; then
            log::error "$resource_name: Failed to migrate config"
        else
            echo "ERROR: $resource_name: Failed to migrate config" >&2
        fi
        return 1
    fi
}

#######################################
# Update a resource configuration to use project root location
# Arguments:
#   $1 - resource name
#   $2 - config section path (e.g., "storage.redis")
#   $3 - config JSON object
# Returns:
#   0 - success
#   1 - error
#######################################
secrets::update_project_config() {
    local resource_name="$1"
    local config_path="$2"
    local config_json="$3"
    local project_config_file
    project_config_file="$(secrets::get_project_config_file)"
    local project_config_dir
    project_config_dir="$(dirname "$project_config_file")"
    
    # Validate inputs
    if [[ -z "$resource_name" ]] || [[ -z "$config_path" ]] || [[ -z "$config_json" ]]; then
        echo "ERROR: All parameters required: resource_name, config_path, config_json" >&2
        return 1
    fi
    
    # Ensure project config directory exists
    mkdir -p "$project_config_dir"
    
    # Load existing config or create new structure
    local existing_config
    if [[ -f "$project_config_file" ]]; then
        existing_config=$(cat "$project_config_file")
    else
        # Create default project config structure
        existing_config='{
  "$schema": "./schemas/service.schema.json",
  "version": "1.0.0",
  "enabled": true,
  "resources": {
    "ai": {},
    "automation": {},
    "storage": {},
    "agents": {},
    "execution": {}
  }
}'
    fi
    
    # Update the config with the new resource configuration
    local updated_config
    if updated_config=$(echo "$existing_config" | jq --arg path "$config_path" --argjson config "$config_json" '
        setpath(["resources"] + ($path | split(".")); $config)
    '); then
        # Write updated config
        echo "$updated_config" > "$project_config_file"
        chmod 644 "$project_config_file"
        
        if command -v log::success >/dev/null 2>&1; then
            log::success "$resource_name: Project configuration updated"
        else
            echo "SUCCESS: $resource_name: Project configuration updated"
        fi
        return 0
    else
        if command -v log::error >/dev/null 2>&1; then
            log::error "$resource_name: Failed to update project configuration"
        else
            echo "ERROR: $resource_name: Failed to update project configuration" >&2
        fi
        return 1
    fi
}

#######################################
# Comprehensive validation of all configuration locations
# Returns:
#   0 - all configs in correct locations
#   1 - some configs in wrong locations
#######################################
secrets::validate_all_configs() {
    local issues=0
    local user_config_file
    user_config_file="$(secrets::get_user_config_file)"
    local project_config_file
    project_config_file="$(secrets::get_project_config_file)"
    
    # Check for any remaining user home configs
    if [[ -f "$user_config_file" ]]; then
        echo "WARNING: Found configuration in wrong location: $user_config_file" >&2
        echo "Should be in: $project_config_file" >&2
        ((issues++))
    fi
    
    # Check project config permissions if it exists
    if [[ -f "$project_config_file" ]]; then
        local perms
        perms=$(stat -c "%a" "$project_config_file" 2>/dev/null || echo "000")
        if [[ "$perms" != "644" ]] && [[ "$perms" != "664" ]]; then
            echo "WARNING: Project config has unusual permissions: $perms" >&2
            echo "Consider: chmod 644 $project_config_file" >&2
            ((issues++))
        fi
    fi
    
    # Validate secrets file (existing function)
    secrets::validate_storage
    local secrets_issues=$?
    ((issues += secrets_issues))
    
    if [[ $issues -eq 0 ]]; then
        if command -v log::success >/dev/null 2>&1; then
            log::success "✅ All configurations in correct locations"
        else
            echo "SUCCESS: All configurations in correct locations"
        fi
    else
        if command -v log::warn >/dev/null 2>&1; then
            log::warn "⚠️  Found $issues configuration issues"
        else
            echo "WARNING: Found $issues configuration issues" >&2
        fi
    fi
    
    return $issues
}