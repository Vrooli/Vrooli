#!/usr/bin/env bash
set -euo pipefail

# Source var.sh first with relative path
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Now source everything else using var_ variables
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_EXIT_CODES_FILE}"

# Source zombie detector for enhanced diagnostics
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/zombie-detector.sh" 2>/dev/null || true

ports::preflight() {
    if ! system::is_command "lsof"; then
        flow::exit_with_error "Required command 'lsof' not found; please install 'lsof'" "$EXIT_DEPENDENCY_ERROR"
    fi
}

# Validates that the given port argument is a non-empty decimal number between 1 and 65535.
ports::validate_port() {
    local port=$1
    if [[ -z "$port" ]]; then
        flow::exit_with_error "Invalid port: '$port'" "$EXIT_INVALID_ARGUMENT"
    fi
    if ! [[ "$port" =~ ^[0-9]+$ ]]; then
        flow::exit_with_error "Invalid port: '$port'" "$EXIT_INVALID_ARGUMENT"
    fi
    # Force decimal interpretation to avoid octal parsing on leading zeros
    if (( 10#$port < 1 || 10#$port > 65535 )); then
        flow::exit_with_error "Port out of range: $port" "$EXIT_INVALID_ARGUMENT"
    fi
}

# Returns PIDs listening on TCP port $1, or exits on error
ports::get_listening_pids() {
    ports::preflight
    local port=$1
    ports::validate_port "$port"
    local output code=0
    output=$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>&1) || code=$?
    if (( code == 1 )); then
        # No listening process
        echo ""
        return 0
    elif (( code != 0 )); then
        flow::exit_with_error "Error checking port $port: $output" "$EXIT_GENERAL_ERROR"
    fi
    echo "$output"
}

# Returns 0 if TCP port $1 has a listening process
ports::is_port_in_use() {
    ports::preflight
    local port=$1
    ports::validate_port "$port"
    local pids
    pids=$(ports::get_listening_pids "$port")
    [[ -n "$pids" ]]
}

# Kills processes listening on TCP port $1
ports::kill() {
    ports::preflight
    local port=$1
    ports::validate_port "$port"
    local pids
    pids=$(ports::get_listening_pids "$port")
    if [[ -n "$pids" ]]; then
        flow::maybe_run_sudo kill $pids
        local kill_status=$?
        if (( kill_status == 0 )); then
            log::success "Killed processes on port $port: $pids"
        else
            flow::exit_with_error "Failed to kill processes on port $port: $pids" "$EXIT_RESOURCE_ERROR"
        fi
    fi
}

# If port is in use, prompt user to kill blockers
ports::check_and_free() {
    ports::preflight
    local port=$1
    ports::validate_port "$port"
    # Fix default YES to avoid unbound-variable errors
    local yes=${2:-${YES:-}}
    local pids
    pids=$(ports::get_listening_pids "$port")
    if [[ -n "$pids" ]]; then
        log::warning "Port $port is in use by process(es): $pids"
        if flow::is_yes "$yes"; then
            ports::kill "$port"
        else
            if flow::confirm "Kill process(es) listening on port $port?"; then
                ports::kill "$port"
            else
                flow::exit_with_error "Please free port $port and retry" "$EXIT_RESOURCE_ERROR"
            fi
        fi
    fi
}

#######################################
# SCENARIO PORT MANAGEMENT
# Enhanced functionality for scenario-level port operations
#######################################

# Source port registry for conflict checking
if [[ -f "${APP_ROOT}/scripts/resources/port_registry.sh" ]]; then
    # shellcheck disable=SC1091
    source "${APP_ROOT}/scripts/resources/port_registry.sh"
fi

# State directory for scenario ports
SCENARIO_STATE_DIR="${HOME}/.vrooli/state/scenarios"

# Initialize scenario port state directory
ports::init_scenario_state() {
    mkdir -p "$SCENARIO_STATE_DIR"
}

# Fast port availability check using kernel binding test
# Replaces the old O(n²) check_port_conflicts function
ports::is_port_available() {
    local port="$1"
    local scenario_name="$2"
    
    # Quick check: is port in resource ports?
    if declare -p RESOURCE_PORTS >/dev/null 2>&1; then
        for resource in "${!RESOURCE_PORTS[@]}"; do
            if [[ "${RESOURCE_PORTS[$resource]}" == "$port" ]]; then
                return 1  # Port is reserved for resource
            fi
        done
    fi
    
    # Check for existing lock file to handle concurrent allocations
    local lock_file="$SCENARIO_STATE_DIR/.port_${port}.lock"
    if [[ -f "$lock_file" ]]; then
        local lock_info lock_scenario lock_pid lock_timestamp
        lock_info=$(cat "$lock_file" 2>/dev/null || true)
        lock_scenario=${lock_info%%:*}
        lock_pid=${lock_info#*:}; lock_pid=${lock_pid%%:*}
        lock_timestamp=${lock_info##*:}

        # If the same scenario already owns the lock and the process is alive,
        # treat the port as available so sequential lifecycle phases can reuse it.
        if [[ -n "$lock_scenario" && "$lock_scenario" == "$scenario_name" ]]; then
            if [[ -n "$lock_pid" ]] && kill -0 "$lock_pid" 2>/dev/null; then
                return 0
            fi
            # Same scenario but process exited – clean up the stale lock
            rm -f "$lock_file" 2>/dev/null || true
        else
            # Different scenario holding the lock – verify if it's still alive
            if [[ -n "$lock_pid" ]] && kill -0 "$lock_pid" 2>/dev/null; then
                return 1
            fi

            # No live process; respect recent locks to avoid rapid recycling
            local now
            now=$(date +%s)
            if [[ -n "$lock_timestamp" ]] && (( now - lock_timestamp < 300 )); then
                return 1
            fi

            # Stale lock – remove it and continue
            rm -f "$lock_file" 2>/dev/null || true
        fi
    fi
    
    # Ultimate test: can we bind to it?
    # Using timeout to prevent hanging
    if timeout 0.1 bash -c "exec 3<>/dev/tcp/127.0.0.1/$port" 2>/dev/null; then
        # Port is in use (we connected to something)
        return 1
    fi
    
    # Try to bind with nc (more reliable test)
    local test_pid
    if nc -l 127.0.0.1 "$port" 2>/dev/null & test_pid=$!; then
        # Successfully bound - port is free!
        kill $test_pid 2>/dev/null
        wait $test_pid 2>/dev/null
        return 0
    else
        # Couldn't bind - port is taken
        return 1
    fi
}

# Fast port allocation using hash-based starting point and bind testing
ports::find_available_scenario_port() {
    local range_start="$1"
    local range_end="$2" 
    local scenario_name="$3"
    local port_type="${4:-port}"  # Optional: api, ui, etc. for better distribution
    
    local range_size=$((range_end - range_start + 1))
    
    # Hash-based starting point for deterministic distribution
    local hash_input="${scenario_name}_${port_type}"
    local hash_value
    hash_value=$(echo -n "$hash_input" | cksum | cut -d' ' -f1)
    local start_offset=$((hash_value % range_size))
    local start_port=$((range_start + start_offset))
    
    local port=$start_port
    local attempts=0
    local max_attempts=$range_size
    
    while ((attempts < max_attempts)); do
        if ports::is_port_available "$port" "$scenario_name"; then
            # Try to claim it atomically
            local lock_file="$SCENARIO_STATE_DIR/.port_${port}.lock"
            if (set -C; echo "${scenario_name}:$$:$(date +%s)" > "$lock_file") 2>/dev/null; then
                # We got it!
                echo "$port"
                return 0
            fi
        fi
        
        # Move to next port with wraparound
        port=$((range_start + ((port - range_start + 1) % range_size)))
        ((attempts++))
    done
    
    # No available ports in range
    return 1
}

# Allocate ports for a scenario based on service.json
ports::allocate_scenario() {
    local scenario_name="$1"
    local service_json_path="$2"
    
    ports::init_scenario_state
    
    if [[ ! -f "$service_json_path" ]]; then
        echo "ERROR: service.json not found: $service_json_path" >&2
        return 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        echo "ERROR: jq is required for JSON processing" >&2
        return 1
    fi
    
    # Parse ports configuration from service.json
    local ports_config
    ports_config=$(jq -r '.ports // {}' "$service_json_path" 2>/dev/null)
    
    if [[ "$ports_config" == "{}" || "$ports_config" == "null" ]]; then
        # No ports to allocate
        echo "{\"success\": true, \"allocated_ports\": {}, \"env_vars\": {}}"
        return 0
    fi
    
    local -A allocated_ports=()
    local -A env_vars=()
    local errors=()
    
    # Process ALL port configurations in a SINGLE pass
    while IFS= read -r port_entry; do
        [[ -n "$port_entry" ]] || continue
        
        local port_name port_config
        port_name=$(echo "$port_entry" | jq -r '.key')
        port_config=$(echo "$port_entry" | jq -r '.value')
        
        # Check if it has a fixed port
        local fixed_port
        fixed_port=$(echo "$port_config" | jq -r '.port // empty' 2>/dev/null)
        
        if [[ -n "$fixed_port" && "$fixed_port" != "null" ]]; then
            # Fixed port allocation
            if ! ports::is_port_available "$fixed_port" "$scenario_name"; then
                # Enhanced diagnostics using zombie detector
                if command -v zombie::diagnose_port_failure >/dev/null 2>&1; then
                    # Provide comprehensive diagnostics
                    zombie::diagnose_port_failure "$fixed_port" "$scenario_name"
                    
                    # Attempt to auto-clean stale locks
                    if zombie::check_stale_port_lock "$fixed_port"; then
                        echo "🔧 Attempting to auto-clean stale lock for port $fixed_port..." >&2
                        if zombie::clean_stale_port_lock "$fixed_port"; then
                            echo "✅ Successfully cleaned stale lock - retrying port allocation" >&2
                            
                            # Retry port allocation after cleaning
                            if ports::is_port_available "$fixed_port" "$scenario_name"; then
                                echo "🎉 Port $fixed_port is now available after cleanup!" >&2
                                # Continue with normal allocation below
                            else
                                errors+=("FIXED PORT UNAVAILABLE: Port $fixed_port for '$port_name' is still unavailable after cleaning stale locks. See diagnostics above for details.")
                                continue
                            fi
                        else
                            errors+=("STALE LOCK CLEANUP FAILED: Unable to clean stale lock for port $fixed_port. Manual intervention required: rm ~/.vrooli/state/scenarios/.port_${fixed_port}.lock")
                            continue
                        fi
                    else
                        errors+=("FIXED PORT UNAVAILABLE: Port $fixed_port for '$port_name' is in use by active processes. See diagnostics above for details.")
                        continue
                    fi
                else
                    # Fallback to original error handling if zombie detector not available
                    local lock_file="$SCENARIO_STATE_DIR/.port_${fixed_port}.lock"
                    if [[ -f "$lock_file" ]]; then
                        local lock_info
                        lock_info=$(cat "$lock_file" 2>/dev/null || echo "unknown")
                        local lock_scenario="${lock_info%%:*}"
                        errors+=("PORT CONFLICT: Fixed port $fixed_port for '$port_name' is locked by scenario '$lock_scenario'. Clean stale locks with: rm ~/.vrooli/state/scenarios/.port_${fixed_port}.lock")
                    else
                        errors+=("PORT CONFLICT: Fixed port $fixed_port for '$port_name' is in use by another process")
                    fi
                    continue
                fi
            fi
            
            # Claim the port with lock file
            local lock_file="$SCENARIO_STATE_DIR/.port_${fixed_port}.lock"
            if [[ -f "$lock_file" ]]; then
                local existing_info existing_scenario
                existing_info=$(cat "$lock_file" 2>/dev/null || true)
                existing_scenario=${existing_info%%:*}
                if [[ -n "$existing_scenario" && "$existing_scenario" == "$scenario_name" ]]; then
                    printf '%s:%s:%s\n' "$scenario_name" "$$" "$(date +%s)" > "$lock_file"
                else
                    errors+=("LOCK FAILURE: Fixed port $fixed_port for '$port_name' already locked by scenario '${existing_scenario:-unknown}'")
                    continue
                fi
            elif ! (set -C; printf '%s:%s:%s\n' "$scenario_name" "$$" "$(date +%s)" > "$lock_file") 2>/dev/null; then
                errors+=("LOCK FAILURE: Could not claim fixed port $fixed_port for '$port_name' (race condition or permission issue)")
                continue
            fi
            
            allocated_ports["$port_name"]="$fixed_port"
        else
            # Check for range-based allocation
            local port_range
            port_range=$(echo "$port_config" | jq -r '.range // empty' 2>/dev/null)
            
            if [[ -n "$port_range" && "$port_range" != "null" ]] && [[ "$port_range" =~ ^([0-9]+)-([0-9]+)$ ]]; then
                local start="${BASH_REMATCH[1]}"
                local end="${BASH_REMATCH[2]}"
                
                local available_port
                # Pass port_name (api, ui, etc.) for better hash distribution
                available_port=$(ports::find_available_scenario_port "$start" "$end" "$scenario_name" "$port_name")
                
                if [[ -n "$available_port" ]]; then
                    allocated_ports["$port_name"]="$available_port"
                else
                    errors+=("RANGE EXHAUSTED: No available ports in range $port_range for '$port_name'. Try: vrooli resource restart to free up resources, or check for stale locks in ~/.vrooli/state/scenarios/")
                    continue
                fi
            else
                # No fixed port or range - skip
                continue
            fi
        fi
        
        # Set environment variable for allocated port
        local env_var
        env_var=$(echo "$port_config" | jq -r '.env_var // empty' 2>/dev/null)
        [[ -z "$env_var" ]] && env_var="${port_name^^}_PORT"
        env_vars["$env_var"]="${allocated_ports[$port_name]}"
        
    done < <(echo "$ports_config" | jq -c 'to_entries[]' 2>/dev/null)
    
    # Add resource ports to environment variables (only for required/enabled resources)
    # Parse resource requirements from service.json
    local resources_config
    resources_config=$(jq -r '.resources // {}' "$service_json_path" 2>/dev/null)
    
    if [[ "$resources_config" != "{}" && "$resources_config" != "null" ]]; then
        # Source port registry if available
        if [[ -f "${APP_ROOT}/scripts/resources/port_registry.sh" ]]; then
            source "${APP_ROOT}/scripts/resources/port_registry.sh" 2>/dev/null || true
        fi
        
        # Only export ports for resources that are required or explicitly enabled
        while IFS= read -r resource_entry; do
            [[ -n "$resource_entry" ]] || continue
            
            local resource_name resource_config
            resource_name=$(echo "$resource_entry" | jq -r '.key')
            resource_config=$(echo "$resource_entry" | jq -r '.value')
            
            # Check if resource is required or enabled
            local is_required is_enabled
            is_required=$(echo "$resource_config" | jq -r '.required // false')
            is_enabled=$(echo "$resource_config" | jq -r '.enabled // false')
            
            # Only include if explicitly required or enabled
            if [[ "$is_required" == "true" || "$is_enabled" == "true" ]]; then
                # Get port from RESOURCE_PORTS array if it exists
                if [[ -n "${RESOURCE_PORTS[$resource_name]:-}" ]]; then
                    local resource_env_var="${resource_name^^}_PORT"
                    # Convert hyphens to underscores for valid environment variable names
                    resource_env_var="${resource_env_var//-/_}"
                    env_vars["$resource_env_var"]="${RESOURCE_PORTS[$resource_name]}"
                    
                    # Log what we're exporting for transparency (only in debug mode)
                    [[ "${DEBUG:-}" == "true" ]] && echo "# Exporting $resource_env_var=${RESOURCE_PORTS[$resource_name]} for $resource_name (required=$is_required, enabled=$is_enabled)" >&2
                fi
            fi
        done < <(echo "$resources_config" | jq -c 'to_entries[]' 2>/dev/null)
    fi
    
    # Check for errors
    if [[ ${#errors[@]} -gt 0 ]]; then
        local error_msg=""
        for error in "${errors[@]}"; do
            error_msg="${error_msg}${error}; "
        done
        echo "{\"success\": false, \"error\": \"${error_msg%%; }\"}"
        return 1
    fi
    
    # Build JSON response for allocated ports and env vars
    # (No longer saving state files - using lock files instead)
    
    # Build allocated_ports JSON
    local allocated_ports_json="{}"
    for port_name in "${!allocated_ports[@]}"; do
        allocated_ports_json=$(echo "$allocated_ports_json" | jq --arg key "$port_name" --argjson value "${allocated_ports[$port_name]}" '. + {($key): $value}')
    done
    
    # Build env_vars JSON
    local env_vars_json="{}"
    for env_var in "${!env_vars[@]}"; do
        env_vars_json=$(echo "$env_vars_json" | jq --arg key "$env_var" --arg value "${env_vars[$env_var]}" '. + {($key): $value}')
    done
    
    # Return success response
    jq -n \
        --argjson allocated_ports "$allocated_ports_json" \
        --argjson env_vars "$env_vars_json" \
        '{
            success: true,
            allocated_ports: $allocated_ports,
            env_vars: $env_vars
        }'
    
    return 0
}

# Deprecated functions removed - using lock-based allocation now

#######################################
# UNIFIED ENVIRONMENT MANAGEMENT
# Single function that handles all environment complexity
#######################################

# Check if scenario processes are currently running (ultra-fast one-liner)
_check_scenario_processes_running() {
    local scenario_name="$1"
    
    # First try: Check process tracking files (most reliable)
    local process_dir="$HOME/.vrooli/processes/scenarios/$scenario_name"
    if [[ -d "$process_dir" ]]; then
        for pid_file in "$process_dir"/*.pid; do
            [[ -f "$pid_file" ]] || continue
            local pid
            pid=$(cat "$pid_file" 2>/dev/null)
            [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null && return 0
        done
    fi
    
    # Fallback: Check for processes with various scenario name patterns
    pgrep -f "\.${scenario_name}\." >/dev/null 2>&1 || pgrep -f "\.${scenario_name}-" >/dev/null 2>&1 || pgrep -f "${scenario_name}-api" >/dev/null 2>&1
}

# Discover actual ports being used by running scenario processes
_discover_scenario_ports() {
    local scenario_name="$1"
    local service_json_path="$2"
    
    # Get PIDs of scenario processes (prefer process tracking files, fallback to pattern matching)
    local pids=()
    local process_dir="$HOME/.vrooli/processes/scenarios/$scenario_name"
    
    # First: Get PIDs from process tracking files (most reliable)
    if [[ -d "$process_dir" ]]; then
        for pid_file in "$process_dir"/*.pid; do
            [[ -f "$pid_file" ]] || continue
            local pid
            pid=$(cat "$pid_file" 2>/dev/null)
            if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                pids+=("$pid")
            fi
        done
    fi
    
    # Fallback: Pattern matching if no tracking files found
    if [[ ${#pids[@]} -eq 0 ]]; then
        local pattern_pids
        pattern_pids=$({ pgrep -f "\.${scenario_name}\." 2>/dev/null || true; pgrep -f "\.${scenario_name}-" 2>/dev/null || true; pgrep -f "${scenario_name}-api" 2>/dev/null || true; } | sort -u)
        [[ -n "$pattern_pids" ]] && pids=($(echo "$pattern_pids" | xargs))
    fi
    
    local pids_str
    pids_str=$(printf "%s " "${pids[@]}" | xargs)
    
    [[ -z "$pids_str" ]] && echo "{}" && return 0
    
    # Get all listening ports for these PIDs
    local ports_data="{}"
    
    # Use lsof to find listening ports for our PIDs
    local lsof_output
    lsof_output=$(lsof -iTCP -sTCP:LISTEN -P 2>/dev/null | grep -E "$(echo $pids_str | tr ' ' '|')" 2>/dev/null || true)
    
    if [[ -n "$lsof_output" ]]; then
        # Parse service.json to understand expected ports
        local port_config
        port_config=$(jq -r '.ports // {}' "$service_json_path" 2>/dev/null)
        
        # Extract port numbers from lsof output
        local discovered_ports
        discovered_ports=$(echo "$lsof_output" | awk -F':' '{print $NF}' | awk '{print $1}' | sort -u)
        
        # Try to match discovered ports to service.json definitions
        # Strategy: Match by port ranges or exact values
        while IFS= read -r port; do
            [[ -z "$port" ]] && continue
            
            # Try to identify which service this port belongs to
            local matched=false
            
            # Check each port definition in service.json
            while IFS= read -r port_entry; do
                [[ -z "$port_entry" ]] && continue
                
                local port_name port_def
                port_name=$(echo "$port_entry" | jq -r '.key')
                port_def=$(echo "$port_entry" | jq -r '.value')
                
                # Check fixed port
                local fixed_port
                fixed_port=$(echo "$port_def" | jq -r '.port // empty' 2>/dev/null)
                if [[ "$fixed_port" == "$port" ]]; then
                    # Found exact match
                    local env_var
                    env_var=$(echo "$port_def" | jq -r '.env_var // empty' 2>/dev/null)
                    [[ -z "$env_var" ]] && env_var="${port_name^^}_PORT"
                    ports_data=$(echo "$ports_data" | jq --arg key "$env_var" --arg value "$port" '. + {($key): $value}')
                    matched=true
                    break
                fi
                
                # Check port range
                local port_range
                port_range=$(echo "$port_def" | jq -r '.range // empty' 2>/dev/null)
                if [[ -n "$port_range" && "$port_range" != "null" ]] && [[ "$port_range" =~ ^([0-9]+)-([0-9]+)$ ]]; then
                    local start="${BASH_REMATCH[1]}"
                    local end="${BASH_REMATCH[2]}"
                    if (( port >= start && port <= end )); then
                        # Port is in expected range
                        local env_var
                        env_var=$(echo "$port_def" | jq -r '.env_var // empty' 2>/dev/null)
                        [[ -z "$env_var" ]] && env_var="${port_name^^}_PORT"
                        ports_data=$(echo "$ports_data" | jq --arg key "$env_var" --arg value "$port" '. + {($key): $value}')
                        matched=true
                        break
                    fi
                fi
            done < <(echo "$port_config" | jq -c 'to_entries[]' 2>/dev/null)
            
            # If no match found, try to guess based on process name from lsof
            if [[ "$matched" == "false" ]]; then
                # Get the process name that owns this port
                local process_info
                process_info=$(echo "$lsof_output" | grep ":$port" | head -1)
                
                # Try to extract step name from process command
                # Pattern: vrooli.develop.scenario.step_name
                if [[ "$process_info" =~ vrooli\.[^.]+\.${scenario_name}\.([^[:space:]]+) ]]; then
                    local step_name="${BASH_REMATCH[1]}"
                    # Common patterns: api, ui, server, etc.
                    if [[ "$step_name" =~ (api|server|backend) ]]; then
                        ports_data=$(echo "$ports_data" | jq --arg key "API_PORT" --arg value "$port" '. + {($key): $value}')
                    elif [[ "$step_name" =~ (ui|frontend|web) ]]; then
                        ports_data=$(echo "$ports_data" | jq --arg key "UI_PORT" --arg value "$port" '. + {($key): $value}')
                    else
                        # Generic port assignment based on step name
                        local env_var="${step_name^^}_PORT"
                        ports_data=$(echo "$ports_data" | jq --arg key "$env_var" --arg value "$port" '. + {($key): $value}')
                    fi
                fi
            fi
        done <<< "$discovered_ports"
    fi
    
    echo "$ports_data"
}

# Load environment variables from enabled resources
_load_resource_environment() {
    local service_json_path="$1"
    local combined_env="{}"
    
    if [[ ! -f "$service_json_path" ]] || ! command -v jq >/dev/null 2>&1; then
        echo "$combined_env"
        return 0
    fi
    
    # Parse enabled resources from service.json
    local enabled_resources
    enabled_resources=$(jq -r '.resources | to_entries[] | select(.value.enabled == true) | .key' "$service_json_path" 2>/dev/null || echo "")
    
    while IFS= read -r resource_name; do
        [[ -z "$resource_name" ]] && continue
        
        local exports_file="${APP_ROOT}/resources/${resource_name}/config/exports.sh"
        
        if [[ -f "$exports_file" ]]; then
            # Source the exports file in subshell and capture environment
            local resource_env
            resource_env=$(
                # Temporarily suppress debug output
                export DEBUG="${DEBUG:-false}"
                
                # Source the exports file in clean environment
                (
                    # Source the exports file
                    source "$exports_file" 2>/dev/null || exit 1
                    
                    # Export all variables that match common patterns
                    env | grep -E "^(${resource_name^^}_|$(echo "$resource_name" | tr '[:lower:]' '[:upper:]' | tr '-' '_')_)" | while IFS='=' read -r key value; do
                        # Escape value for JSON (handle empty values correctly)
                        local escaped_value
                        if [[ -z "$value" ]]; then
                            escaped_value='""'  # Empty string in JSON
                        else
                            escaped_value=$(printf '%s' "$value" | jq -R .)
                        fi
                        printf '"%s": %s,' "$key" "$escaped_value"
                    done
                ) 2>/dev/null | sed '$ s/,$//'  # Remove last comma
            )
            
            if [[ -n "$resource_env" ]]; then
                # Merge into combined environment
                combined_env=$(echo "$combined_env" | jq ". + {$resource_env}")
            fi
        fi
        
    done <<< "$enabled_resources"
    
    echo "$combined_env"
}

# Get complete scenario environment - THE UNIFIED FUNCTION
# This handles everything: allocation, state, resources, process checking
ports::get_scenario_environment() {
    local scenario_name="$1"
    local service_json_path="$2"
    
    if [[ -z "$scenario_name" ]] || [[ -z "$service_json_path" ]]; then
        echo "{\"success\": false, \"error\": \"Missing required parameters: scenario_name and service_json_path\"}"
        return 1
    fi
    
    if [[ ! -f "$service_json_path" ]]; then
        echo "{\"success\": false, \"error\": \"Service JSON not found: $service_json_path\"}"
        return 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        echo "{\"success\": false, \"error\": \"jq is required for JSON processing\"}"
        return 1
    fi
    
    ports::init_scenario_state
    
    local state_file="$SCENARIO_STATE_DIR/${scenario_name}.json"
    local is_running=false
    local env_vars="{}"
    local allocated_ports="{}"
    local message="Environment loaded successfully"
    
    # Check if scenario processes are running
    if _check_scenario_processes_running "$scenario_name"; then
        is_running=true
    fi
    
    # Phase 1: Process Check and Port Discovery
    if [[ "$is_running" == "true" ]]; then
        # Scenario is running - discover actual ports in use
        local discovered_ports
        discovered_ports=$(_discover_scenario_ports "$scenario_name" "$service_json_path")
        
        if [[ "$discovered_ports" != "{}" ]]; then
            # Use discovered ports as environment variables
            env_vars="$discovered_ports"
            
            # Also populate allocated_ports for API compatibility
            # Extract port numbers only for allocated_ports
            allocated_ports="{}"
            while IFS= read -r entry; do
                [[ -z "$entry" ]] && continue
                local key value
                key=$(echo "$entry" | jq -r '.key')
                value=$(echo "$entry" | jq -r '.value')
                
                # Simplify key for allocated_ports (remove _PORT suffix)
                local port_name
                port_name=$(echo "$key" | sed 's/_PORT$//' | tr '[:upper:]' '[:lower:]')
                allocated_ports=$(echo "$allocated_ports" | jq --arg key "$port_name" --argjson value "$value" '. + {($key): $value}')
            done < <(echo "$discovered_ports" | jq -c 'to_entries[]' 2>/dev/null)
            
            message="Discovered ports for running scenario"
        else
            # Scenario running but no ports discovered - treat as fresh allocation
            is_running=false
            message="Running scenario has no discoverable ports, allocating fresh"
        fi
    fi
    
    # Phase 2: Fresh Allocation (if needed)
    if [[ "$is_running" == "false" ]]; then
        local allocation_result
        allocation_result=$(ports::allocate_scenario "$scenario_name" "$service_json_path")
        
        if [[ $(echo "$allocation_result" | jq -r '.success') != "true" ]]; then
            # Pass through allocation error
            echo "$allocation_result"
            return 1
        fi
        
        env_vars=$(echo "$allocation_result" | jq -r '.env_vars // {}')
        allocated_ports=$(echo "$allocation_result" | jq -r '.allocated_ports // {}')
        message="Created fresh port allocation"
    fi
    
    # Phase 3: Resource Environment Loading (THE KEY ENHANCEMENT!)
    local resource_env
    resource_env=$(_load_resource_environment "$service_json_path")
    
    if [[ "$resource_env" != "{}" ]]; then
        env_vars=$(echo "$env_vars" | jq ". + $resource_env")
        message="$message with resource environment"
    fi
    
    # Phase 4: Return Complete Environment
    jq -n \
        --argjson env_vars "$env_vars" \
        --argjson allocated_ports "$allocated_ports" \
        --arg is_running "$is_running" \
        --arg message "$message" \
        '{
            success: true,
            is_running: ($is_running == "true"),
            allocated_ports: $allocated_ports,
            env_vars: $env_vars,
            message: $message
        }'
    
    return 0
}

# CLI interface for scenario port management
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        allocate_scenario)
            ports::allocate_scenario "$2" "$3"
            ;;
        get_scenario_environment)
            ports::get_scenario_environment "$2" "$3"
            ;;
        *)
            # Keep existing behavior for other operations
            ;;
    esac
fi 
