#!/usr/bin/env bash
# Lifecycle Engine - Step Executor Module
# Handles execution of individual steps with retry, timeout, and error handling

set -euo pipefail

# Get script directory
LIB_LIFECYCLE_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "$LIB_LIFECYCLE_LIB_DIR/../../utils/var.sh"

# Guard against re-sourcing
[[ -n "${_EXECUTOR_MODULE_LOADED:-}" ]] && return 0
declare -gr _EXECUTOR_MODULE_LOADED=1

# Source dependencies
# shellcheck source=./condition.sh
source "$var_LIB_LIFECYCLE_DIR/lib/condition.sh"
# shellcheck source=./output.sh
source "$var_LIB_LIFECYCLE_DIR/lib/output.sh"

# Step execution statistics
declare -gA STEP_STATS=()

#######################################
# Execute a single step
# Arguments:
#   $1 - Step JSON object
# Returns:
#   Exit code of the step
#######################################
executor::run_step() {
    local step_json="$1"
    
    # Parse step configuration
    local step_name step_run step_type
    step_name=$(echo "$step_json" | jq -r '.name // "unnamed"')
    step_run=$(echo "$step_json" | jq -r '.run // empty')
    step_type=$(echo "$step_json" | jq -r '.type // "bash"')
    
    # Check if step should be skipped
    if condition::should_skip_step "$step_name"; then
        executor::log_info "Skipping step: $step_name"
        return 0
    fi
    
    # Evaluate condition
    local step_condition
    step_condition=$(echo "$step_json" | jq -r '.condition // empty')
    if [[ -n "$step_condition" ]]; then
        if ! condition::evaluate "$step_condition"; then
            executor::log_info "Skipping step (condition false): $step_name"
            return 0
        fi
    fi
    
    # Check for parallel execution (delegate to parallel module)
    local step_parallel
    step_parallel=$(echo "$step_json" | jq -r '.parallel // empty')
    if [[ -n "$step_parallel" ]] && [[ "$step_parallel" != "null" ]]; then
        # This would be handled by parallel module
        executor::log_warning "Parallel execution not handled in executor module"
        return 1
    fi
    
    # Execute the step
    executor::log_info "Executing step: $step_name"
    
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        executor::dry_run_step "$step_json"
        return 0
    fi
    
    # Record start time
    local start_time
    start_time=$(date +%s)
    STEP_STATS["${step_name}.start_time"]="$start_time"
    
    # Execute based on type
    local exit_code=0
    case "$step_type" in
        bash|shell)
            executor::run_bash_step "$step_json" || exit_code=$?
            ;;
        script)
            executor::run_script_step "$step_json" || exit_code=$?
            ;;
        function)
            executor::run_function_step "$step_json" || exit_code=$?
            ;;
        *)
            executor::log_error "Unknown step type: $step_type"
            exit_code=1
            ;;
    esac
    
    # Record end time and duration
    local end_time duration
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    STEP_STATS["${step_name}.end_time"]="$end_time"
    STEP_STATS["${step_name}.duration"]="$duration"
    STEP_STATS["${step_name}.exit_code"]="$exit_code"
    
    # Handle step failure
    if [[ $exit_code -ne 0 ]]; then
        executor::handle_failure "$step_json" $exit_code
        return $?
    fi
    
    executor::log_success "Step completed: $step_name (${duration}s)"
    return 0
}

#######################################
# Execute bash/shell step
# Arguments:
#   $1 - Step JSON object
# Returns:
#   Exit code of command
#######################################
executor::run_bash_step() {
    local step_json="$1"
    
    local step_name step_run step_workdir step_env step_timeout step_outputs
    step_name=$(echo "$step_json" | jq -r '.name // "unnamed"')
    step_run=$(echo "$step_json" | jq -r '.run // empty')
    step_workdir=$(echo "$step_json" | jq -r '.workdir // empty')
    step_env=$(echo "$step_json" | jq -r '.env // {}')
    step_timeout=$(echo "$step_json" | jq -r '.timeout // empty')
    step_outputs=$(echo "$step_json" | jq -r '.outputs // {}')
    
    # Build environment variables
    local env_vars=()
    if [[ "$step_env" != "{}" ]] && [[ "$step_env" != "null" ]]; then
        while IFS= read -r key && IFS= read -r value; do
            env_vars+=("$key=$value")
        done < <(echo "$step_env" | jq -r 'to_entries[] | .key, .value')
    fi
    
    # Build command with timeout
    local cmd="$step_run"
    if [[ -n "$step_timeout" ]]; then
        local timeout_seconds
        timeout_seconds=$(executor::parse_duration "$step_timeout")
        cmd="timeout $timeout_seconds $cmd"
    fi
    
    # Execute command
    local exit_code=0
    local output=""
    
    # Default to service.json directory if no workdir specified
    if [[ -z "$step_workdir" ]] && [[ -n "${SERVICE_JSON_PATH:-}" ]]; then
        step_workdir=$(dirname "${SERVICE_JSON_PATH}")
    fi
    
    if [[ "$step_outputs" != "{}" ]] && [[ "$step_outputs" != "null" ]]; then
        # Capture output for processing
        if [[ -n "$step_workdir" ]]; then
            output=$(cd "$step_workdir" && env "${env_vars[@]}" bash -c "$cmd" 2>&1) || exit_code=$?
        else
            output=$(env "${env_vars[@]}" bash -c "$cmd" 2>&1) || exit_code=$?
        fi
        
        # Process outputs
        output::process_definitions "$step_name" "$step_outputs" "$output"
    else
        # Run without capturing output
        if [[ -n "$step_workdir" ]]; then
            (cd "$step_workdir" && env "${env_vars[@]}" bash -c "$cmd") || exit_code=$?
        else
            env "${env_vars[@]}" bash -c "$cmd" || exit_code=$?
        fi
    fi
    
    return $exit_code
}

#######################################
# Execute script step
# Arguments:
#   $1 - Step JSON object
# Returns:
#   Exit code of script
#######################################
executor::run_script_step() {
    local step_json="$1"
    
    local script_path
    script_path=$(echo "$step_json" | jq -r '.script // empty')
    
    if [[ -z "$script_path" ]] || [[ ! -f "$script_path" ]]; then
        executor::log_error "Script not found: $script_path"
        return 1
    fi
    
    # Make script executable
    chmod +x "$script_path"
    
    # Convert to bash step
    local bash_step
    bash_step=$(echo "$step_json" | jq --arg cmd "$script_path" '.run = $cmd')
    
    executor::run_bash_step "$bash_step"
}

#######################################
# Execute function step
# Arguments:
#   $1 - Step JSON object
# Returns:
#   Exit code of function
#######################################
executor::run_function_step() {
    local step_json="$1"
    
    local function_name
    function_name=$(echo "$step_json" | jq -r '.function // empty')
    
    if [[ -z "$function_name" ]]; then
        executor::log_error "No function specified"
        return 1
    fi
    
    # Check if function exists
    if ! declare -f "$function_name" >/dev/null 2>&1; then
        executor::log_error "Function not found: $function_name"
        return 1
    fi
    
    # Get function arguments
    local args
    args=$(echo "$step_json" | jq -r '.args[]? // empty' 2>/dev/null)
    
    # Execute function
    if [[ -n "$args" ]]; then
        "$function_name" $args
    else
        "$function_name"
    fi
}

#######################################
# Handle step failure
# Arguments:
#   $1 - Step JSON object
#   $2 - Exit code
# Returns:
#   Exit code based on error strategy
#######################################
executor::handle_failure() {
    local step_json="$1"
    local exit_code="$2"
    
    local step_name error_strategy
    step_name=$(echo "$step_json" | jq -r '.name // "unnamed"')
    error_strategy=$(echo "$step_json" | jq -r '.error // "stop"')
    
    executor::log_error "Step failed: $step_name (exit code: $exit_code)"
    
    case "$error_strategy" in
        continue|warn)
            executor::log_warning "Continuing despite error (strategy: $error_strategy)"
            return 0
            ;;
        retry)
            executor::retry_step "$step_json"
            return $?
            ;;
        ignore)
            executor::log_info "Ignoring error (strategy: ignore)"
            return 0
            ;;
        stop|fail|*)
            return $exit_code
            ;;
    esac
}

#######################################
# Retry failed step
# Arguments:
#   $1 - Step JSON object
# Returns:
#   Exit code of retry attempt
#######################################
executor::retry_step() {
    local step_json="$1"
    
    local step_name retry_config
    step_name=$(echo "$step_json" | jq -r '.name // "unnamed"')
    retry_config=$(echo "$step_json" | jq -r '.retry // {}')
    
    local max_attempts delay backoff
    max_attempts=$(echo "$retry_config" | jq -r '.attempts // 3')
    delay=$(echo "$retry_config" | jq -r '.delay // "5s"')
    backoff=$(echo "$retry_config" | jq -r '.backoff // "linear"')
    
    local attempt=1
    local current_delay
    current_delay=$(executor::parse_duration "$delay")
    
    while [[ $attempt -le $max_attempts ]]; do
        executor::log_info "Retry attempt $attempt/$max_attempts for: $step_name"
        
        if [[ $attempt -gt 1 ]]; then
            executor::log_info "Waiting ${current_delay}s before retry..."
            sleep "$current_delay"
        fi
        
        # Clear previous failure
        unset "STEP_STATS[${step_name}.exit_code]"
        
        # Retry execution
        if executor::run_step "$step_json"; then
            executor::log_success "Retry successful for: $step_name"
            return 0
        fi
        
        # Calculate next delay based on backoff strategy
        case "$backoff" in
            exponential)
                current_delay=$((current_delay * 2))
                ;;
            linear)
                current_delay=$((current_delay + $(executor::parse_duration "$delay")))
                ;;
            constant|*)
                # Keep same delay
                ;;
        esac
        
        ((attempt++))
    done
    
    executor::log_error "All retry attempts failed for: $step_name"
    return 1
}

#######################################
# Dry run step (show what would be executed)
# Arguments:
#   $1 - Step JSON object
#######################################
executor::dry_run_step() {
    local step_json="$1"
    
    local step_name step_run step_type step_workdir step_env step_timeout
    step_name=$(echo "$step_json" | jq -r '.name // "unnamed"')
    step_run=$(echo "$step_json" | jq -r '.run // empty')
    step_type=$(echo "$step_json" | jq -r '.type // "bash"')
    step_workdir=$(echo "$step_json" | jq -r '.workdir // empty')
    step_env=$(echo "$step_json" | jq -r '.env // {}')
    step_timeout=$(echo "$step_json" | jq -r '.timeout // empty')
    
    echo "[DRY RUN] Step: $step_name"
    echo "  Type: $step_type"
    [[ -n "$step_run" ]] && echo "  Command: $step_run"
    [[ -n "$step_workdir" ]] && echo "  Working directory: $step_workdir"
    [[ "$step_env" != "{}" ]] && echo "  Environment: $step_env"
    [[ -n "$step_timeout" ]] && echo "  Timeout: $step_timeout"
}

#######################################
# Parse duration string to seconds
# Arguments:
#   $1 - Duration string (e.g., "5s", "2m", "1h")
# Returns:
#   Duration in seconds
#######################################
executor::parse_duration() {
    local duration="$1"
    
    case "$duration" in
        *h)
            echo $((${duration%h} * 3600))
            ;;
        *m)
            echo $((${duration%m} * 60))
            ;;
        *s)
            echo "${duration%s}"
            ;;
        *)
            echo "$duration"
            ;;
    esac
}

#######################################
# Get step statistics
# Arguments:
#   $1 - Step name
#   $2 - Stat key (optional)
# Returns:
#   Stat value or all stats
#######################################
executor::get_stats() {
    local step_name="$1"
    local key="${2:-}"
    
    if [[ -n "$key" ]]; then
        echo "${STEP_STATS[${step_name}.${key}]:-}"
    else
        # Return all stats for step
        for stat_key in "${!STEP_STATS[@]}"; do
            if [[ "$stat_key" == "${step_name}."* ]]; then
                echo "${stat_key#${step_name}.}: ${STEP_STATS[$stat_key]}"
            fi
        done
    fi
}

#######################################
# Logging functions
#######################################
executor::log_info() {
    echo "[INFO] $*" >&2
}

executor::log_success() {
    echo "[SUCCESS] $*" >&2
}

executor::log_warning() {
    echo "[WARNING] $*" >&2
}

executor::log_error() {
    echo "[ERROR] $*" >&2
}

# If sourced for testing, don't auto-execute
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script should be sourced, not executed directly" >&2
    exit 1
fi