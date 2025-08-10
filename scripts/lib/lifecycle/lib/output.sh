#!/usr/bin/env bash
# Lifecycle Engine - Output Management Module
# Handles step outputs, variable management, and inter-step communication

set -euo pipefail

# Determine script directory
_HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${_HERE}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

# Guard against re-sourcing
[[ -n "${_OUTPUT_MODULE_LOADED:-}" ]] && return 0
declare -gr _OUTPUT_MODULE_LOADED=1

# Global storage for step outputs
declare -gA STEP_OUTPUTS=()
declare -gA STEP_METADATA=()
declare -gA STEP_CACHE=()

# Output capture temporary directory
OUTPUT_TEMP_DIR=""

#######################################
# Initialize output management
# Creates temporary directory for output capture
#######################################
output::init() {
    if [[ -z "$OUTPUT_TEMP_DIR" ]]; then
        OUTPUT_TEMP_DIR=$(mktemp -d -t lifecycle-outputs.XXXXXX)
        export OUTPUT_TEMP_DIR
    fi
}

#######################################
# Cleanup output management
# Removes temporary files and directories
#######################################
output::cleanup() {
    if [[ -n "$OUTPUT_TEMP_DIR" ]] && [[ -d "$OUTPUT_TEMP_DIR" ]]; then
        trash::safe_remove "$OUTPUT_TEMP_DIR" --no-confirm
    fi
}

#######################################
# Capture command output
# Arguments:
#   $1 - Step name
#   $2 - Output variable name
#   $3 - Command to execute
# Returns:
#   Exit code of command
#######################################
output::capture() {
    local step_name="$1"
    local var_name="$2"
    local command="$3"
    
    output::init
    
    local output_file="${OUTPUT_TEMP_DIR}/${step_name}_${var_name}.out"
    local error_file="${OUTPUT_TEMP_DIR}/${step_name}_${var_name}.err"
    local exit_code=0
    
    # Execute command and capture output
    if eval "$command" > "$output_file" 2> "$error_file"; then
        exit_code=0
    else
        exit_code=$?
    fi
    
    # Store output
    local output_value
    output_value=$(cat "$output_file" 2>/dev/null || echo "")
    
    output::set "$step_name" "$var_name" "$output_value"
    
    # Store error if any
    if [[ -s "$error_file" ]]; then
        local error_value
        error_value=$(cat "$error_file")
        output::set "$step_name" "${var_name}_error" "$error_value"
    fi
    
    # Store exit code
    output::set "$step_name" "${var_name}_exit_code" "$exit_code"
    
    return $exit_code
}

#######################################
# Set output variable
# Arguments:
#   $1 - Step name
#   $2 - Variable name
#   $3 - Value
#######################################
output::set() {
    local step_name="$1"
    local var_name="$2"
    local value="$3"
    
    local key="${step_name}.${var_name}"
    STEP_OUTPUTS["$key"]="$value"
    
    # Also export as environment variable
    local env_var="OUTPUT_${step_name^^}_${var_name^^}"
    env_var="${env_var//-/_}"
    env_var="${env_var//./_}"
    export "$env_var=$value"
}

#######################################
# Get output variable
# Arguments:
#   $1 - Step name
#   $2 - Variable name
# Returns:
#   Variable value or empty string
#######################################
output::get() {
    local step_name="$1"
    local var_name="$2"
    
    local key="${step_name}.${var_name}"
    echo "${STEP_OUTPUTS[$key]:-}"
}

#######################################
# Check if output exists
# Arguments:
#   $1 - Step name
#   $2 - Variable name
# Returns:
#   0 if exists, 1 if not
#######################################
output::exists() {
    local step_name="$1"
    local var_name="$2"
    
    local key="${step_name}.${var_name}"
    [[ -n "${STEP_OUTPUTS[$key]:-}" ]]
}

#######################################
# Process step output definitions
# Arguments:
#   $1 - Step name
#   $2 - JSON output configuration
#   $3 - Command output (stdout)
# Sets:
#   Output variables based on configuration
#######################################
output::process_definitions() {
    local step_name="$1"
    local output_config="$2"
    local command_output="$3"
    
    if [[ "$output_config" == "{}" ]] || [[ "$output_config" == "null" ]]; then
        return 0
    fi
    
    # Process each output definition
    local keys
    keys=$(echo "$output_config" | jq -r 'keys[]')
    
    while IFS= read -r key; do
        local definition
        definition=$(echo "$output_config" | jq ".\"$key\"")
        
        local type
        type=$(echo "$definition" | jq -r '.type // "string"')
        
        case "$type" in
            string)
                output::process_string "$step_name" "$key" "$definition" "$command_output"
                ;;
            json)
                output::process_json "$step_name" "$key" "$definition" "$command_output"
                ;;
            regex)
                output::process_regex "$step_name" "$key" "$definition" "$command_output"
                ;;
            file)
                output::process_file "$step_name" "$key" "$definition"
                ;;
            env)
                output::process_env "$step_name" "$key" "$definition"
                ;;
            *)
                echo "Warning: Unknown output type: $type" >&2
                ;;
        esac
    done <<< "$keys"
}

#######################################
# Process string output (full stdout)
# Arguments:
#   $1 - Step name
#   $2 - Variable name
#   $3 - Definition JSON
#   $4 - Command output
#######################################
output::process_string() {
    local step_name="$1"
    local var_name="$2"
    local definition="$3"
    local output="$4"
    
    # Check for trim option
    local trim
    trim=$(echo "$definition" | jq -r '.trim // "true"')
    
    if [[ "$trim" == "true" ]]; then
        output="${output%"${output##*[![:space:]]}"}"  # Trim trailing
        output="${output#"${output%%[![:space:]]*}"}"  # Trim leading
    fi
    
    output::set "$step_name" "$var_name" "$output"
}

#######################################
# Process JSON output
# Arguments:
#   $1 - Step name
#   $2 - Variable name
#   $3 - Definition JSON
#   $4 - Command output
#######################################
output::process_json() {
    local step_name="$1"
    local var_name="$2"
    local definition="$3"
    local output="$4"
    
    # Get JSON path if specified
    local json_path
    json_path=$(echo "$definition" | jq -r '.path // "."')
    
    # Validate JSON
    if ! echo "$output" | jq empty 2>/dev/null; then
        echo "Warning: Invalid JSON output for $var_name" >&2
        return 1
    fi
    
    # Extract value at path
    local value
    value=$(echo "$output" | jq -r "$json_path")
    
    output::set "$step_name" "$var_name" "$value"
}

#######################################
# Process regex extraction
# Arguments:
#   $1 - Step name
#   $2 - Variable name
#   $3 - Definition JSON
#   $4 - Command output
#######################################
output::process_regex() {
    local step_name="$1"
    local var_name="$2"
    local definition="$3"
    local output="$4"
    
    local pattern
    pattern=$(echo "$definition" | jq -r '.pattern // ""')
    
    if [[ -z "$pattern" ]]; then
        echo "Warning: No regex pattern specified for $var_name" >&2
        return 1
    fi
    
    # Extract using grep
    local value
    if value=$(echo "$output" | grep -oP "$pattern" 2>/dev/null | head -1); then
        output::set "$step_name" "$var_name" "$value"
    else
        echo "Warning: Regex pattern not matched for $var_name" >&2
    fi
}

#######################################
# Process file output
# Arguments:
#   $1 - Step name
#   $2 - Variable name
#   $3 - Definition JSON
#######################################
output::process_file() {
    local step_name="$1"
    local var_name="$2"
    local definition="$3"
    
    local file_path
    file_path=$(echo "$definition" | jq -r '.path // ""')
    
    if [[ -z "$file_path" ]]; then
        echo "Warning: No file path specified for $var_name" >&2
        return 1
    fi
    
    # Expand path
    file_path=$(eval "echo \"$file_path\"")
    
    if [[ -f "$file_path" ]]; then
        local content
        content=$(cat "$file_path")
        output::set "$step_name" "$var_name" "$content"
    else
        echo "Warning: File not found for $var_name: $file_path" >&2
    fi
}

#######################################
# Process environment variable output
# Arguments:
#   $1 - Step name
#   $2 - Variable name
#   $3 - Definition JSON
#######################################
output::process_env() {
    local step_name="$1"
    local var_name="$2"
    local definition="$3"
    
    local env_var
    env_var=$(echo "$definition" | jq -r '.variable // ""')
    
    if [[ -z "$env_var" ]]; then
        env_var="$var_name"
    fi
    
    local value="${!env_var:-}"
    output::set "$step_name" "$var_name" "$value"
}

#######################################
# Set step metadata
# Arguments:
#   $1 - Step name
#   $2 - Metadata key
#   $3 - Value
#######################################
output::set_metadata() {
    local step_name="$1"
    local key="$2"
    local value="$3"
    
    STEP_METADATA["${step_name}.${key}"]="$value"
}

#######################################
# Get step metadata
# Arguments:
#   $1 - Step name
#   $2 - Metadata key
# Returns:
#   Metadata value or empty string
#######################################
output::get_metadata() {
    local step_name="$1"
    local key="$2"
    
    echo "${STEP_METADATA[${step_name}.${key}]:-}"
}

#######################################
# Cache step result
# Arguments:
#   $1 - Step name
#   $2 - Cache key
#   $3 - Value
#######################################
output::cache_set() {
    local step_name="$1"
    local cache_key="$2"
    local value="$3"
    
    STEP_CACHE["${step_name}.${cache_key}"]="$value"
}

#######################################
# Get cached step result
# Arguments:
#   $1 - Step name
#   $2 - Cache key
# Returns:
#   Cached value or empty string
#######################################
output::cache_get() {
    local step_name="$1"
    local cache_key="$2"
    
    echo "${STEP_CACHE[${step_name}.${cache_key}]:-}"
}

#######################################
# Check if step result is cached
# Arguments:
#   $1 - Step name
#   $2 - Cache key
# Returns:
#   0 if cached, 1 if not
#######################################
output::cache_exists() {
    local step_name="$1"
    local cache_key="$2"
    
    [[ -n "${STEP_CACHE[${step_name}.${cache_key}]:-}" ]]
}

#######################################
# Export all outputs as environment variables
#######################################
output::export_all() {
    for key in "${!STEP_OUTPUTS[@]}"; do
        local env_var="OUTPUT_${key^^}"
        env_var="${env_var//-/_}"
        env_var="${env_var//./_}"
        export "$env_var=${STEP_OUTPUTS[$key]}"
    done
}

#######################################
# List all outputs (for debugging)
#######################################
output::list() {
    echo "Step Outputs:" >&2
    for key in "${!STEP_OUTPUTS[@]}"; do
        echo "  $key = ${STEP_OUTPUTS[$key]}" >&2
    done
}

# Cleanup on exit
trap output::cleanup EXIT

# If sourced for testing, don't auto-execute
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script should be sourced, not executed directly" >&2
    exit 1
fi