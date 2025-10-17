#!/usr/bin/env bash
set -euo pipefail

# Vrooli Injection Framework
# Common framework for all resource injection adapters
# Eliminates 80% of code duplication across inject.sh files
# Version: 1.0.0

# Prevent multiple sourcing
[[ -n "${VROOLI_INJECT_FRAMEWORK_LOADED:-}" ]] && return 0
readonly VROOLI_INJECT_FRAMEWORK_LOADED=1

# Framework initialization
readonly FRAMEWORK_VERSION="1.1.0"  # Updated: Fixed critical security vulnerabilities

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
readonly FRAMEWORK_DIR="${APP_ROOT}/scripts/resources/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

# Global framework state
declare -g FRAMEWORK_ADAPTER_NAME=""
declare -g FRAMEWORK_SERVICE_HOST=""
declare -g FRAMEWORK_DATA_DIR=""
declare -g FRAMEWORK_HEALTH_ENDPOINT=""
declare -ag FRAMEWORK_ROLLBACK_ACTIONS=()

# Adapter function registry
declare -Ag FRAMEWORK_VALIDATORS=()
declare -Ag FRAMEWORK_INJECTORS=()
declare -Ag FRAMEWORK_STATUS_CHECKERS=()
declare -Ag FRAMEWORK_HEALTH_CHECKERS=()

#######################################
# Register adapter with framework
# Arguments:
#   --name: Adapter name (required)
#   --service-host: Service host URL (required)
#   --health-endpoint: Health check endpoint (optional)
#   --data-dir: Data directory (optional)
#   --validate-func: Validation function name (required)
#   --inject-func: Injection function name (required)
#   --status-func: Status check function name (required)
#   --health-func: Health check function name (optional)
# Usage:
#   inject_framework::register "whisper" \
#     --service-host "http://localhost:9005" \
#     --health-endpoint "/docs" \
#     --data-dir "${HOME}/.whisper" \
#     --validate-func "whisper_validate_config" \
#     --inject-func "whisper_inject_data" \
#     --status-func "whisper_check_status"
#######################################
inject_framework::register() {
    local name="$1"
    shift
    
    FRAMEWORK_ADAPTER_NAME="$name"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --service-host)
                FRAMEWORK_SERVICE_HOST="$2"
                shift 2
                ;;
            --health-endpoint)
                FRAMEWORK_HEALTH_ENDPOINT="$2"
                shift 2
                ;;
            --data-dir)
                FRAMEWORK_DATA_DIR="$2"
                shift 2
                ;;
            --validate-func)
                FRAMEWORK_VALIDATORS["$name"]="$2"
                shift 2
                ;;
            --inject-func)
                FRAMEWORK_INJECTORS["$name"]="$2"
                shift 2
                ;;
            --status-func)
                FRAMEWORK_STATUS_CHECKERS["$name"]="$2"
                shift 2
                ;;
            --health-func)
                FRAMEWORK_HEALTH_CHECKERS["$name"]="$2"
                shift 2
                ;;
            *)
                log::error "Unknown registration parameter: $1"
                return 1
                ;;
        esac
    done
    
    # Validate required parameters
    if [[ -z "$FRAMEWORK_SERVICE_HOST" ]]; then
        log::error "Adapter registration requires --service-host"
        return 1
    fi
    
    if [[ -z "${FRAMEWORK_VALIDATORS[$name]:-}" ]]; then
        log::error "Adapter registration requires --validate-func"
        return 1
    fi
    
    if [[ -z "${FRAMEWORK_INJECTORS[$name]:-}" ]]; then
        log::error "Adapter registration requires --inject-func"
        return 1
    fi
    
    if [[ -z "${FRAMEWORK_STATUS_CHECKERS[$name]:-}" ]]; then
        log::error "Adapter registration requires --status-func"
        return 1
    fi
    
    log::debug "Registered adapter: $name with framework v${FRAMEWORK_VERSION}"
    return 0
}

#######################################
# Standard usage display
# Arguments:
#   $1 - adapter name
#   $2 - description
#   $3 - configuration format example (optional)
#######################################
inject_framework::usage() {
    local name="$1"
    local description="$2"
    local config_example="${3:-}"
    
    cat << EOF
${name^} Data Injection Adapter (Framework v${FRAMEWORK_VERSION})

USAGE:
    \$0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    $description
    Supports validation, injection, status checks, and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected data
    --rollback    Rollback injected data
    --help        Show this help message

EOF

    if [[ -n "$config_example" ]]; then
        cat << EOF
CONFIGURATION FORMAT:
$config_example

EOF
    fi
    
    cat << EOF
EXAMPLES:
    # Validate configuration
    \$0 --validate '{"items": [{"name": "example"}]}'
    
    # Perform injection
    \$0 --inject '{"items": [{"name": "example", "enabled": true}]}'
    
    # Check status
    \$0 --status '{"items": [{"name": "example"}]}'
    
    # Rollback changes
    \$0 --rollback

EOF
}

#######################################
# Standard accessibility check
# Uses registered service host and health endpoint
# Returns:
#   0 if accessible, 1 otherwise
#######################################
inject_framework::check_accessibility() {
    local name="${FRAMEWORK_ADAPTER_NAME}"
    local host="${FRAMEWORK_SERVICE_HOST}"
    local health_endpoint="${FRAMEWORK_HEALTH_ENDPOINT:-}"
    
    if [[ -z "$host" ]]; then
        log::error "No service host configured for $name"
        return 1
    fi
    
    # Use custom health check if provided
    if [[ -n "${FRAMEWORK_HEALTH_CHECKERS[$name]:-}" ]]; then
        local health_func="${FRAMEWORK_HEALTH_CHECKERS[$name]}"
        if declare -f "$health_func" >/dev/null; then
            "$health_func"
            return $?
        fi
    fi
    
    # Standard HTTP health check
    if ! system::is_command "curl"; then
        log::error "curl command not available for health check"
        return 1
    fi
    
    local check_url="${host}${health_endpoint}"
    log::debug "Checking $name accessibility at $check_url"
    
    if curl -s --max-time 5 "$check_url" >/dev/null 2>&1; then
        log::debug "$name is accessible at $host"
        return 0
    else
        log::error "$name is not accessible at $host"
        log::info "Ensure $name is running: resource-$name manage start"
        return 1
    fi
}

#######################################
# Standard rollback system - SAFE VERSION
# Add predefined rollback action to framework queue
# Arguments:
#   $1 - description
#   $2 - rollback type (file, directory, api_delete, custom_function)
#   $3+ - parameters for the rollback type
#######################################
inject_framework::add_rollback_action() {
    local description="$1"
    local rollback_type="$2"
    shift 2
    local params=("$@")
    
    # Store as: "description|type|param1|param2|..."
    local action_entry="$description|$rollback_type"
    for param in "${params[@]}"; do
        action_entry="$action_entry|$param"
    done
    
    FRAMEWORK_ROLLBACK_ACTIONS+=("$action_entry")
    log::debug "Added rollback action for ${FRAMEWORK_ADAPTER_NAME}: $description"
}

#######################################
# Execute all rollback actions - SAFE VERSION
# Executes in reverse order for proper cleanup
#######################################
inject_framework::execute_rollback() {
    local name="${FRAMEWORK_ADAPTER_NAME}"
    
    if [[ ${#FRAMEWORK_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No $name rollback actions to execute"
        return 0
    fi
    
    log::info "Executing $name rollback actions..."
    
    local success_count=0
    local total_count=${#FRAMEWORK_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#FRAMEWORK_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${FRAMEWORK_ROLLBACK_ACTIONS[i]}"
        
        # Parse action entry safely
        IFS='|' read -ra action_parts <<< "$action"
        
        if [[ ${#action_parts[@]} -lt 2 ]]; then
            log::error "Invalid rollback action format: $action"
            continue
        fi
        
        local description="${action_parts[0]}"
        local rollback_type="${action_parts[1]}"
        local params=("${action_parts[@]:2}")
        
        log::info "Rollback: $description"
        
        if inject_framework::execute_safe_rollback "$rollback_type" "${params[@]}"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "$name rollback completed: $success_count/$total_count actions successful"
    FRAMEWORK_ROLLBACK_ACTIONS=()
}

#######################################
# Standard JSON validation
# Arguments:
#   $1 - configuration JSON string
#   $2 - required fields array (space-separated)
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject_framework::validate_json() {
    local config="$1"
    local required_fields="${2:-}"
    local name="${FRAMEWORK_ADAPTER_NAME}"
    
    log::debug "Validating $name JSON configuration..."
    
    # Basic JSON syntax validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in $name injection configuration"
        return 1
    fi
    
    # Check for at least one of the required fields
    if [[ -n "$required_fields" ]]; then
        local has_any_field=false
        
        for field in $required_fields; do
            if echo "$config" | jq -e ".$field" >/dev/null 2>&1; then
                has_any_field=true
                break
            fi
        done
        
        if [[ "$has_any_field" == false ]]; then
            log::error "$name injection configuration must have one of: $required_fields"
            return 1
        fi
    fi
    
    log::debug "$name JSON configuration is valid"
    return 0
}

#######################################
# Standard array validation
# Arguments:
#   $1 - array JSON string
#   $2 - array name (for error messages)
#   $3 - required fields per item (space-separated)
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject_framework::validate_array() {
    local array_config="$1"
    local array_name="$2"
    local required_item_fields="${3:-}"
    
    # Check if it's actually an array
    local array_type
    array_type=$(echo "$array_config" | jq -r 'type')
    
    if [[ "$array_type" != "array" ]]; then
        log::error "$array_name configuration must be an array, got: $array_type"
        return 1
    fi
    
    # Validate each item if required fields specified
    if [[ -n "$required_item_fields" ]]; then
        local item_count
        item_count=$(echo "$array_config" | jq 'length')
        
        for ((i=0; i<item_count; i++)); do
            local item
            item=$(echo "$array_config" | jq -c ".[$i]")
            
            for field in $required_item_fields; do
                local field_value
                field_value=$(echo "$item" | jq -r ".$field // empty")
                
                if [[ -z "$field_value" ]]; then
                    log::error "$array_name item at index $i missing required '$field' field"
                    return 1
                fi
            done
        done
    fi
    
    log::debug "$array_name array configuration is valid"
    return 0
}



#######################################
# Validate array with context and custom validator
# Arguments:
#   $1 - array JSON string
#   $2 - array name (for error messages)
#   $3 - required fields per item (space-separated)
#   $4 - custom validator function (optional)
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject_framework::validate_array_with_context() {
    local array_config="$1"
    local array_name="$2"
    local required_fields="$3"
    local custom_validator="${4:-}"
    
    # First do basic array validation
    if ! inject_framework::validate_array "$array_config" "$array_name" "$required_fields"; then
        return 1
    fi
    
    # If custom validator provided, validate each item
    if [[ -n "$custom_validator" ]] && declare -f "$custom_validator" >/dev/null; then
        local item_count
        item_count=$(echo "$array_config" | jq 'length')
        
        for ((i=0; i<item_count; i++)); do
            local item
            item=$(echo "$array_config" | jq -c ".[$i]")
            
            # Get item name for context
            local item_name
            for name_field in name id file path; do
                item_name=$(echo "$item" | jq -r ".$name_field // empty")
                if [[ -n "$item_name" ]]; then
                    break
                fi
            done
            item_name="${item_name:-item_$i}"
            
            # Call custom validator with item, index, and name
            if ! "$custom_validator" "$item" "$i" "$item_name"; then
                return 1
            fi
        done
    fi
    
    return 0
}

#######################################
# Process array items with callback
# Arguments:
#   $1 - array JSON string
#   $2 - callback function name
#   $3 - callback description (for logging)
# Returns:
#   0 if all items processed successfully, 1 if any failed
#######################################
inject_framework::process_array() {
    local array_config="$1"
    local callback_func="$2"
    local description="${3:-items}"
    
    local item_count
    item_count=$(echo "$array_config" | jq 'length')
    
    if [[ "$item_count" -eq 0 ]]; then
        log::info "No $description to process"
        return 0
    fi
    
    log::info "Processing $item_count $description..."
    
    local failed_items=()
    
    for ((i=0; i<item_count; i++)); do
        local item
        item=$(echo "$array_config" | jq -c ".[$i]")
        
        # Get item name for error tracking (try common name fields)
        local item_name
        for name_field in name id file path; do
            item_name=$(echo "$item" | jq -r ".$name_field // empty")
            if [[ -n "$item_name" ]]; then
                break
            fi
        done
        item_name="${item_name:-item_$i}"
        
        if ! "$callback_func" "$item"; then
            failed_items+=("$item_name")
        fi
    done
    
    if [[ ${#failed_items[@]} -eq 0 ]]; then
        log::success "All $description processed successfully"
        return 0
    else
        log::error "Failed to process $description: ${failed_items[*]}"
        return 1
    fi
}

#######################################
# Process array items in parallel batches for improved performance
# Processes items concurrently with configurable batch size and proper error handling
# Arguments:
#   $1 - array configuration JSON
#   $2 - callback function name
#   $3 - callback description (for logging)
#   $4 - batch size (optional, default: 5)
#   $5 - enable parallel processing (optional, default: yes)
# Returns:
#   0 if all items processed successfully, 1 if any failed
#######################################
inject_framework::process_array_parallel() {
    local array_config="$1"
    local callback_func="$2"
    local description="${3:-items}"
    local batch_size="${4:-5}"
    local enable_parallel="${5:-yes}"
    
    # Fall back to sequential processing if parallel is disabled
    if [[ "$enable_parallel" != "yes" ]]; then
        log::debug "Parallel processing disabled, using sequential method"
        inject_framework::process_array "$array_config" "$callback_func" "$description"
        return $?
    fi
    
    local item_count
    item_count=$(echo "$array_config" | jq 'length')
    
    if [[ "$item_count" -eq 0 ]]; then
        log::info "No $description to process"
        return 0
    fi
    
    log::info "Processing $item_count $description in parallel batches of $batch_size..."
    
    # Create temporary directory for parallel job tracking
    local temp_dir
    temp_dir=$(mktemp -d)
    local failed_items=()
    local processed_count=0
    
    # Function to clean up background jobs and temp files
    cleanup_parallel_jobs() {
        # Kill any remaining background jobs
        local jobs_list
        jobs_list=$(jobs -p)
        if [[ -n "$jobs_list" ]]; then
            echo "$jobs_list" | xargs -r kill 2>/dev/null || true
        fi
        # Clean up temp directory
        [[ -d "$temp_dir" ]] && rm -rf "$temp_dir"
    }
    
    # Set up signal handlers for cleanup
    trap cleanup_parallel_jobs EXIT INT TERM
    
    # Process items in batches
    for ((i=0; i<item_count; i+=batch_size)); do
        local batch_start=$i
        local batch_end=$((i + batch_size - 1))
        if [[ $batch_end -ge $item_count ]]; then
            batch_end=$((item_count - 1))
        fi
        
        log::debug "Processing batch: items $((batch_start + 1))-$((batch_end + 1)) of $item_count"
        
        # Start background jobs for this batch
        local batch_jobs=()
        for ((j=batch_start; j<=batch_end; j++)); do
            local item
            item=$(echo "$array_config" | jq -c ".[$j]")
            
            # Get item name for tracking
            local item_name
            for name_field in name id file path; do
                item_name=$(echo "$item" | jq -r ".$name_field // empty")
                if [[ -n "$item_name" ]]; then
                    break
                fi
            done
            item_name="${item_name:-item_$j}"
            
            # Create unique result file for this job
            local result_file="$temp_dir/result_$j"
            
            # Start background job with output redirection
            (
                if "$callback_func" "$item" 2>&1; then
                    echo "SUCCESS:$item_name" > "$result_file"
                else
                    echo "FAILED:$item_name" > "$result_file"
                fi
            ) &
            
            local job_pid=$!
            batch_jobs+=("$job_pid:$j:$item_name")
            log::debug "Started background job for '$item_name' (PID: $job_pid)"
        done
        
        # Wait for all jobs in this batch to complete
        log::debug "Waiting for batch of ${#batch_jobs[@]} jobs to complete..."
        for job_info in "${batch_jobs[@]}"; do
            local job_pid="${job_info%%:*}"
            local job_index="${job_info#*:}"
            job_index="${job_index%:*}"
            local job_name="${job_info##*:}"
            
            if wait "$job_pid"; then
                log::debug "Job completed successfully: $job_name"
            else
                log::debug "Job completed with error: $job_name"
            fi
            
            # Read result from file
            local result_file="$temp_dir/result_$job_index"
            if [[ -f "$result_file" ]]; then
                local result_content
                result_content=$(cat "$result_file")
                if [[ "$result_content" == FAILED:* ]]; then
                    local failed_name="${result_content#FAILED:}"
                    failed_items+=("$failed_name")
                fi
                rm -f "$result_file"
            else
                log::warn "No result file found for job: $job_name"
                failed_items+=("$job_name")
            fi
            
            processed_count=$((processed_count + 1))
        done
        
        log::debug "Completed batch: $processed_count/$item_count items processed"
    done
    
    # Clean up
    cleanup_parallel_jobs
    trap - EXIT INT TERM
    
    # Report results
    if [[ ${#failed_items[@]} -eq 0 ]]; then
        log::success "All $description processed successfully using parallel batches"
        return 0
    else
        log::error "Failed to process $description in parallel: ${failed_items[*]}"
        return 1
    fi
}

#######################################
# Main command dispatcher
# Handles standard command interface and delegates to registered functions
# Arguments:
#   $1 - action (--validate, --inject, --status, --rollback, --help)
#   $2 - configuration JSON (except for --rollback and --help)
#######################################
inject_framework::main() {
    local action="$1"
    local config="${2:-}"
    local name="${FRAMEWORK_ADAPTER_NAME}"
    
    if [[ -z "$name" ]]; then
        log::error "No adapter registered with framework"
        return 1
    fi
    
    case "$action" in
        "--validate")
            if [[ -z "$config" ]]; then
                log::error "Configuration JSON required for validation"
                return 1
            fi
            
            local validate_func="${FRAMEWORK_VALIDATORS[$name]}"
            if declare -f "$validate_func" >/dev/null; then
                "$validate_func" "$config"
            else
                log::error "Validation function not found: $validate_func"
                return 1
            fi
            ;;
        "--inject")
            if [[ -z "$config" ]]; then
                log::error "Configuration JSON required for injection"
                return 1
            fi
            
            local inject_func="${FRAMEWORK_INJECTORS[$name]}"
            if declare -f "$inject_func" >/dev/null; then
                "$inject_func" "$config"
            else
                log::error "Injection function not found: $inject_func"
                return 1
            fi
            ;;
        "--status")
            if [[ -z "$config" ]]; then
                log::error "Configuration JSON required for status check"
                return 1
            fi
            
            local status_func="${FRAMEWORK_STATUS_CHECKERS[$name]}"
            if declare -f "$status_func" >/dev/null; then
                "$status_func" "$config"
            else
                log::error "Status function not found: $status_func"
                return 1
            fi
            ;;
        "--rollback")
            inject_framework::execute_rollback
            ;;
        "--help")
            # This should be handled by the adapter's usage function
            log::error "Help should be handled by adapter usage function"
            return 1
            ;;
        *)
            log::error "Unknown action: $action"
            return 1
            ;;
    esac
}

#######################################
# Utility function to resolve file paths relative to project root
# Arguments:
#   $1 - relative file path
# Returns:
#   Absolute file path
#######################################
inject_framework::resolve_file_path() {
    local file_path="$1"
    echo "${var_ROOT_DIR}/$file_path"
}

#######################################
# Utility function to create directories safely
# Arguments:
#   $1 - directory path
#   $2 - description (for rollback)
#######################################
inject_framework::ensure_directory() {
    local dir_path="$1"
    local description="${2:-directory}"
    
    if [[ ! -d "$dir_path" ]]; then
        log::info "Creating $description: $dir_path"
        mkdir -p "$dir_path"
        
        # Add rollback action
        inject_framework::add_rollback_action \
            "Remove created $description: $dir_path" \
            "rmdir '$dir_path' 2>/dev/null || true"
    fi
}

#######################################
# Utility function for file operations with rollback
# Arguments:
#   $1 - source file path
#   $2 - destination file path
#   $3 - operation description
#######################################
inject_framework::copy_file_with_rollback() {
    local src_path="$1"
    local dest_path="$2"
    local description="${3:-file}"
    
    if [[ ! -f "$src_path" ]]; then
        log::error "Source file not found: $src_path"
        return 1
    fi
    
    log::info "Copying $description: $(basename "$src_path")"
    cp "$src_path" "$dest_path"
    
    # Add rollback action
    inject_framework::add_rollback_action \
        "Remove copied $description: $dest_path" \
        "rm -f '$dest_path' 2>/dev/null || true"
    
    return 0
}

#######################################
# Validate JSON structure against expected types
# Arguments:
#   $1 - JSON data
#   $2 - structure specification (e.g., "workflows:array credentials:object")
# Returns:
#   0 if structure matches, 1 otherwise
#######################################
inject_framework::validate_json_structure() {
    local json_data="$1"
    local structure_spec="$2"
    
    for requirement in $structure_spec; do
        IFS=':' read -r field expected_type <<< "$requirement"
        local actual_type
        actual_type=$(echo "$json_data" | jq -r ".$field | type" 2>/dev/null || echo "null")
        
        if [[ "$actual_type" != "$expected_type" && "$actual_type" != "null" ]]; then
            log::error "Field '$field' must be $expected_type, got: $actual_type"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Load adapter configuration with proper error handling
# Arguments:
#   $1 - adapter name
#   $2 - script directory
#######################################
inject_framework::load_adapter_config() {
    local adapter_name="$1"
    local script_dir="$2"
    
    local config_file="${script_dir}/config/defaults.sh"
    if [[ -f "$config_file" ]]; then
        # shellcheck disable=SC1090
        source "$config_file" 2>/dev/null || true
        
        # Call export function if it exists
        local export_func="${adapter_name}::export_config"
        if declare -f "$export_func" >/dev/null 2>&1; then
            "$export_func" 2>/dev/null || true
        fi
    fi
}


#######################################
# Check if JSON has a field (simplified helper)
# Arguments:
#   $1 - JSON string
#   $2 - field name
# Returns:
#   0 if field exists, 1 otherwise
#######################################
inject_framework::has_field() {
    local json="$1"
    local field="$2"
    echo "$json" | jq -e ".$field" >/dev/null 2>&1
}

#######################################
# Get field value with default
# Arguments:
#   $1 - JSON string
#   $2 - field path (e.g., ".name" or ".config.url")
#   $3 - default value (optional)
# Returns:
#   Field value or default
#######################################
inject_framework::get_field() {
    local json="$1"
    local field="$2"
    local default="${3:-}"
    
    local value
    value=$(echo "$json" | jq -r "$field // empty" 2>/dev/null)
    
    if [[ -z "$value" ]]; then
        echo "$default"
    else
        echo "$value"
    fi
}

#######################################
# Process JSON conditionally
# Only processes array if field exists
# Arguments:
#   $1 - config JSON
#   $2 - field name
#   $3 - callback function
#   $4 - description
# Returns:
#   0 if successful or field doesn't exist, 1 if processing failed
#######################################
inject_framework::process_if_exists() {
    local config="$1"
    local field="$2"
    local callback="$3"
    local description="$4"
    
    if inject_framework::has_field "$config" "$field"; then
        local array
        array=$(echo "$config" | jq -c ".$field")
        
        if ! inject_framework::process_array "$array" "$callback" "$description"; then
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Validate and extract ID from API response
# Arguments:
#   $1 - API response JSON
#   $2 - expected field name (default: "id")
# Returns:
#   ID if found, empty string otherwise
#######################################
inject_framework::extract_id() {
    local response="$1"
    local field="${2:-id}"
    
    local id
    id=$(echo "$response" | jq -r ".$field // empty" 2>/dev/null)
    
    if [[ -n "$id" ]] && [[ "$id" != "null" ]]; then
        echo "$id"
    else
        echo ""
    fi
}

#######################################
# Make API call with proper error handling - SAFE VERSION
# Arguments:
#   $1 - HTTP method (GET, POST, PUT, DELETE)
#   $2 - URL
#   $3 - Data/file (optional, use @ prefix for files)
#   $4 - Additional curl options (optional)
# Returns:
#   0 if successful, 1 otherwise
# Outputs:
#   Response body on stdout
#######################################
inject_framework::api_call() {
    local method="$1"
    local url="$2"
    local data="${3:-}"
    local extra_opts="${4:-}"
    
    # Validate method
    case "$method" in
        GET|POST|PUT|DELETE|PATCH) ;;
        *) 
            log::error "Invalid HTTP method: $method"
            return 1
            ;;
    esac
    
    # Build curl command array safely
    local curl_args=()
    curl_args+=("-s" "-X" "$method")
    
    # Add data if provided
    if [[ -n "$data" ]]; then
        if [[ "$data" == @* ]]; then
            # File upload - validate file exists
            local file_path="${data#@}"
            if [[ ! -f "$file_path" ]]; then
                log::error "Data file not found: $file_path"
                return 1
            fi
            curl_args+=("-d" "$data")
        else
            # Raw data
            curl_args+=("-d" "$data")
        fi
        curl_args+=("-H" "Content-Type: application/json")
    fi
    
    # Add extra options (split safely)
    if [[ -n "$extra_opts" ]]; then
        # Note: This assumes extra_opts is a single option or properly quoted
        # For complex options, caller should pass them as separate calls
        read -ra extra_array <<< "$extra_opts"
        curl_args+=("${extra_array[@]}")
    fi
    
    # Add URL last
    curl_args+=("$url")
    
    # Execute curl command safely
    local response
    if response=$(curl "${curl_args[@]}" 2>/dev/null); then
        echo "$response"
        return 0
    else
        log::debug "API call failed: ${method} ${url}"
        return 1
    fi
}

#######################################
# Simplified status reporter
# Arguments:
#   $1 - item name
#   $2 - status (found/notfound/active/inactive)
#   $3 - additional info (optional)
#######################################
inject_framework::report_status() {
    local name="$1"
    local status="$2"
    local info="${3:-}"
    
    case "$status" in
        found)
            log::success "✅ $name found${info:+ ($info)}"
            ;;
        notfound)
            log::error "❌ $name not found"
            ;;
        active)
            log::info "   Status: Active${info:+ - $info}"
            ;;
        inactive)
            log::info "   Status: Inactive${info:+ - $info}"
            ;;
        *)
            log::info "   $name: $status${info:+ - $info}"
            ;;
    esac
}

#######################################
# Batch process with progress tracking
# Arguments:
#   $1 - array JSON
#   $2 - callback function
#   $3 - description
#   $4 - show progress (yes/no, default: yes)
# Returns:
#   0 if all successful, 1 if any failed
#######################################
inject_framework::process_array_with_progress() {
    local array_config="$1"
    local callback_func="$2"
    local description="${3:-items}"
    local show_progress="${4:-yes}"
    
    local item_count
    item_count=$(echo "$array_config" | jq 'length')
    
    if [[ "$item_count" -eq 0 ]]; then
        log::info "No $description to process"
        return 0
    fi
    
    log::info "Processing $item_count $description..."
    
    local failed_items=()
    local processed=0
    
    for ((i=0; i<item_count; i++)); do
        local item
        item=$(echo "$array_config" | jq -c ".[$i]")
        
        # Show progress if enabled
        if [[ "$show_progress" == "yes" ]] && [[ "$item_count" -gt 5 ]]; then
            local percent=$((processed * 100 / item_count))
            log::info "Progress: $percent% ($processed/$item_count)"
        fi
        
        # Get item identifier
        local item_name
        for name_field in name id file path; do
            item_name=$(echo "$item" | jq -r ".$name_field // empty")
            if [[ -n "$item_name" ]]; then
                break
            fi
        done
        item_name="${item_name:-item_$i}"
        
        if ! "$callback_func" "$item"; then
            failed_items+=("$item_name")
        fi
        
        processed=$((processed + 1))
    done
    
    if [[ ${#failed_items[@]} -eq 0 ]]; then
        log::success "All $description processed successfully"
        return 0
    else
        log::error "Failed to process $description: ${failed_items[*]}"
        return 1
    fi
}

#######################################
# Execute predefined rollback types safely
# Arguments:
#   $1 - rollback type
#   $2+ - parameters
# Returns:
#   0 if successful, 1 if failed
#######################################
inject_framework::execute_safe_rollback() {
    local rollback_type="$1"
    shift
    local params=("$@")
    
    case "$rollback_type" in
        "file")
            # Remove a file safely
            if [[ ${#params[@]} -ge 1 ]] && [[ -f "${params[0]}" ]]; then
                rm -f "${params[0]}" 2>/dev/null
            fi
            ;;
        "directory")
            # Remove a directory safely
            if [[ ${#params[@]} -ge 1 ]] && [[ -d "${params[0]}" ]]; then
                rmdir "${params[0]}" 2>/dev/null || rm -rf "${params[0]}" 2>/dev/null
            fi
            ;;
        "api_delete")
            # Make API DELETE request safely
            if [[ ${#params[@]} -ge 1 ]] && system::is_command "curl"; then
                curl -s -X DELETE "${params[0]}" >/dev/null 2>&1
            fi
            ;;
        "api_post")
            # Make API POST request safely (e.g., deactivation)
            if [[ ${#params[@]} -ge 1 ]] && system::is_command "curl"; then
                curl -s -X POST "${params[0]}" >/dev/null 2>&1
            fi
            ;;
        "custom_function")
            # Call a predefined function safely
            if [[ ${#params[@]} -ge 1 ]] && declare -f "${params[0]}" >/dev/null; then
                "${params[@]}"
            fi
            ;;
        "echo")
            # Simple echo for testing/info
            echo "${params[*]}"
            ;;
        *)
            log::error "Unknown rollback type: $rollback_type"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Helper functions for migrating existing rollback usage
# These provide backward compatibility while encouraging safe usage
#######################################

# Safe file removal rollback
inject_framework::add_file_rollback() {
    local description="$1"
    local file_path="$2"
    inject_framework::add_rollback_action "$description" "file" "$file_path"
}

# Safe directory removal rollback
inject_framework::add_directory_rollback() {
    local description="$1"
    local dir_path="$2"
    inject_framework::add_rollback_action "$description" "directory" "$dir_path"
}

# Safe API DELETE rollback
inject_framework::add_api_delete_rollback() {
    local description="$1"
    local api_url="$2"
    inject_framework::add_rollback_action "$description" "api_delete" "$api_url"
}

# Safe API POST rollback (e.g., deactivation)
inject_framework::add_api_post_rollback() {
    local description="$1"
    local api_url="$2"
    inject_framework::add_rollback_action "$description" "api_post" "$api_url"
}

# Safe function call rollback
inject_framework::add_function_rollback() {
    local description="$1"
    local function_name="$2"
    shift 2
    inject_framework::add_rollback_action "$description" "custom_function" "$function_name" "$@"
}

log::debug "Injection framework v${FRAMEWORK_VERSION} loaded (SECURITY HARDENED)"