#!/usr/bin/env bash
set -euo pipefail

# Vrooli Injection Validation Utilities
# Common validation patterns for all resource injection adapters
# Provides standardized validation functions to eliminate duplication
# Version: 1.0.0

# Prevent multiple sourcing
[[ -n "${VROOLI_VALIDATION_UTILS_LOADED:-}" ]] && return 0
readonly VROOLI_VALIDATION_UTILS_LOADED=1

readonly VALIDATION_UTILS_VERSION="1.0.0"

# Source required utilities (relative to framework location)
VALIDATION_UTILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${VALIDATION_UTILS_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"

#######################################
# Validate semantic version format
# Arguments:
#   $1 - version string
# Returns:
#   0 if valid semver, 1 otherwise
#######################################
validation::is_valid_semver() {
    local version="$1"
    
    if [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Validate URL format
# Arguments:
#   $1 - URL string
# Returns:
#   0 if valid URL, 1 otherwise
#######################################
validation::is_valid_url() {
    local url="$1"
    
    # Basic URL validation - starts with http(s)://
    if [[ "$url" =~ ^https?://[a-zA-Z0-9.-]+(:[0-9]+)?(/.*)?$ ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Validate port number
# Arguments:
#   $1 - port number
# Returns:
#   0 if valid port, 1 otherwise
#######################################
validation::is_valid_port() {
    local port="$1"
    
    if [[ "$port" =~ ^[0-9]+$ ]] && [[ "$port" -ge 1 ]] && [[ "$port" -le 65535 ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Validate file extension
# Arguments:
#   $1 - file path
#   $2 - allowed extensions (space-separated)
# Returns:
#   0 if extension is allowed, 1 otherwise
#######################################
validation::has_allowed_extension() {
    local file_path="$1"
    local allowed_extensions="$2"
    
    local file_ext="${file_path##*.}"
    file_ext="${file_ext,,}"  # Convert to lowercase
    
    for ext in $allowed_extensions; do
        ext="${ext,,}"  # Convert to lowercase
        if [[ "$file_ext" == "$ext" ]]; then
            return 0
        fi
    done
    
    return 1
}

#######################################
# Validate model name against allowed list
# Arguments:
#   $1 - model name
#   $2 - allowed models (space-separated)
# Returns:
#   0 if model is allowed, 1 otherwise
#######################################
validation::is_allowed_model() {
    local model_name="$1"
    local allowed_models="$2"
    
    for allowed in $allowed_models; do
        if [[ "$model_name" == "$allowed" ]]; then
            return 0
        fi
    done
    
    return 1
}

#######################################
# Validate JSON object has required fields
# Arguments:
#   $1 - JSON object string
#   $2 - required fields (space-separated)
#   $3 - object description (for error messages)
# Returns:
#   0 if all required fields present, 1 otherwise
#######################################
validation::has_required_fields() {
    local json_object="$1"
    local required_fields="$2"
    local object_description="${3:-object}"
    
    for field in $required_fields; do
        local field_value
        field_value=$(echo "$json_object" | jq -r ".$field // empty")
        
        if [[ -z "$field_value" ]]; then
            log::error "$object_description missing required field: '$field'"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Validate boolean field value
# Arguments:
#   $1 - JSON object string
#   $2 - field name
# Returns:
#   0 if field is valid boolean, 1 otherwise
#######################################
validation::is_valid_boolean_field() {
    local json_object="$1"
    local field_name="$2"
    
    local field_value
    field_value=$(echo "$json_object" | jq -r ".$field_name // empty")
    
    if [[ -n "$field_value" && "$field_value" != "true" && "$field_value" != "false" ]]; then
        log::error "Field '$field_name' must be a boolean (true/false), got: $field_value"
        return 1
    fi
    
    return 0
}

#######################################
# Validate string field is not empty
# Arguments:
#   $1 - JSON object string
#   $2 - field name
#   $3 - field description (for error messages)
# Returns:
#   0 if field is non-empty string, 1 otherwise
#######################################
validation::is_non_empty_string() {
    local json_object="$1"
    local field_name="$2"
    local field_description="${3:-$field_name}"
    
    local field_value
    field_value=$(echo "$json_object" | jq -r ".$field_name // empty")
    
    if [[ -z "$field_value" ]]; then
        log::error "$field_description cannot be empty"
        return 1
    fi
    
    return 0
}

#######################################
# Validate array field contains items
# Arguments:
#   $1 - JSON object string
#   $2 - field name
#   $3 - minimum item count (default: 1)
# Returns:
#   0 if array has sufficient items, 1 otherwise
#######################################
validation::array_has_min_items() {
    local json_object="$1"
    local field_name="$2"
    local min_count="${3:-1}"
    
    local array_length
    array_length=$(echo "$json_object" | jq ".$field_name | length")
    
    if [[ "$array_length" -lt "$min_count" ]]; then
        log::error "Field '$field_name' must have at least $min_count items, got: $array_length"
        return 1
    fi
    
    return 0
}

#######################################
# Validate workflow/script name format
# Arguments:
#   $1 - name string
# Returns:
#   0 if name format is valid, 1 otherwise
#######################################
validation::is_valid_name() {
    local name="$1"
    
    # Allow alphanumeric, hyphens, underscores, dots (no spaces at start/end)
    if [[ "$name" =~ ^[a-zA-Z0-9][a-zA-Z0-9._-]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$ ]]; then
        return 0
    else
        log::error "Invalid name format: '$name' (must be alphanumeric with ._- allowed, no spaces)"
        return 1
    fi
}

#######################################
# Validate file exists and is readable
# Arguments:
#   $1 - file path (can be relative to project root)
#   $2 - file description (for error messages)
# Returns:
#   0 if file exists and readable, 1 otherwise
#######################################
validation::file_exists_and_readable() {
    local file_path="$1"
    local file_description="${2:-file}"
    
    # Try absolute path first, then relative to project root
    local full_path="$file_path"
    if [[ ! -f "$full_path" ]]; then
        full_path="${var_ROOT_DIR}/$file_path"
    fi
    
    if [[ ! -f "$full_path" ]]; then
        log::error "$file_description not found: $file_path"
        return 1
    fi
    
    if [[ ! -r "$full_path" ]]; then
        log::error "$file_description is not readable: $full_path"
        return 1
    fi
    
    return 0
}

#######################################
# Validate configuration against common patterns
# Arguments:
#   $1 - configuration type (workflows, scripts, models, etc.)
#   $2 - array configuration JSON
#   $3 - allowed extensions (optional, space-separated)
# Returns:
#   0 if configuration is valid, 1 otherwise
#######################################
validation::validate_common_config_pattern() {
    local config_type="$1"
    local array_config="$2"
    local allowed_extensions="${3:-}"
    
    log::debug "Validating $config_type configuration..."
    
    # Basic array validation
    if ! inject_framework::validate_array "$array_config" "$config_type" "name file"; then
        return 1
    fi
    
    local item_count
    item_count=$(echo "$array_config" | jq 'length')
    
    # Validate each item
    for ((i=0; i<item_count; i++)); do
        local item
        item=$(echo "$array_config" | jq -c ".[$i]")
        
        # Validate name format
        local name
        name=$(echo "$item" | jq -r '.name')
        if ! validation::is_valid_name "$name"; then
            log::error "$config_type item at index $i has invalid name"
            return 1
        fi
        
        # Validate file exists
        local file
        file=$(echo "$item" | jq -r '.file')
        if ! validation::file_exists_and_readable "$file" "$config_type file"; then
            return 1
        fi
        
        # Validate file extension if specified
        if [[ -n "$allowed_extensions" ]]; then
            if ! validation::has_allowed_extension "$file" "$allowed_extensions"; then
                log::error "$config_type file '$file' has invalid extension. Allowed: $allowed_extensions"
                return 1
            fi
        fi
        
        # Validate enabled field if present
        if echo "$item" | jq -e '.enabled' >/dev/null 2>&1; then
            if ! validation::is_valid_boolean_field "$item" "enabled"; then
                log::error "$config_type item at index $i has invalid 'enabled' field"
                return 1
            fi
        fi
    done
    
    log::success "$config_type configuration is valid ($item_count items)"
    return 0
}

#######################################
# Validate model configuration (common across AI services)
# Arguments:
#   $1 - models array configuration JSON
#   $2 - allowed model names (space-separated)
# Returns:
#   0 if valid, 1 otherwise
#######################################
validation::validate_models_config() {
    local models_config="$1"
    local allowed_models="$2"
    
    log::debug "Validating models configuration..."
    
    # Basic array validation
    if ! inject_framework::validate_array "$models_config" "models" "name"; then
        return 1
    fi
    
    local model_count
    model_count=$(echo "$models_config" | jq 'length')
    
    # Validate each model
    for ((i=0; i<model_count; i++)); do
        local model
        model=$(echo "$models_config" | jq -c ".[$i]")
        
        # Check required fields
        local name
        name=$(echo "$model" | jq -r '.name // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Model at index $i missing required 'name' field"
            return 1
        fi
        
        # Validate model name against allowed list
        if [[ -n "$allowed_models" ]]; then
            if ! validation::is_allowed_model "$name" "$allowed_models"; then
                log::error "Invalid model name: $name. Allowed models: $allowed_models"
                return 1
            fi
        fi
        
        # Validate boolean fields if present
        for bool_field in download preload enabled; do
            if echo "$model" | jq -e ".$bool_field" >/dev/null 2>&1; then
                if ! validation::is_valid_boolean_field "$model" "$bool_field"; then
                    return 1
                fi
            fi
        done
        
        log::debug "Model '$name' configuration is valid"
    done
    
    log::success "Models configuration is valid ($model_count models)"
    return 0
}

#######################################
# Validate credentials configuration (common across automation services)
# Arguments:
#   $1 - credentials array configuration JSON
# Returns:
#   0 if valid, 1 otherwise
#######################################
validation::validate_credentials_config() {
    local credentials_config="$1"
    
    log::debug "Validating credentials configuration..."
    
    # Basic array validation
    if ! inject_framework::validate_array "$credentials_config" "credentials" "name type"; then
        return 1
    fi
    
    local cred_count
    cred_count=$(echo "$credentials_config" | jq 'length')
    
    # Validate each credential
    for ((i=0; i<cred_count; i++)); do
        local credential
        credential=$(echo "$credentials_config" | jq -c ".[$i]")
        
        # Validate required fields
        if ! validation::has_required_fields "$credential" "name type" "credential"; then
            return 1
        fi
        
        # Validate name format
        local name
        name=$(echo "$credential" | jq -r '.name')
        if ! validation::is_valid_name "$name"; then
            log::error "Credential at index $i has invalid name format"
            return 1
        fi
        
        log::debug "Credential '$name' configuration is valid"
    done
    
    log::success "Credentials configuration is valid ($cred_count credentials)"
    return 0
}

#######################################
# Get validation summary for logging
# Returns formatted validation summary
#######################################
validation::get_summary() {
    cat << EOF
Validation utilities v${VALIDATION_UTILS_VERSION} available:
  - Semantic version validation
  - URL and port validation  
  - File existence and extension validation
  - JSON field validation (required, boolean, non-empty)
  - Common configuration patterns (workflows, models, credentials)
  - Name format validation
EOF
}

log::debug "Validation utilities v${VALIDATION_UTILS_VERSION} loaded"