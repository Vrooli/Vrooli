#!/usr/bin/env bash
# Generic Wait/Retry Utility Functions
# Provides reusable wait and retry operations for all resource managers

# Source guard to prevent multiple sourcing
[[ -n "${_WAIT_UTILS_SOURCED:-}" ]] && return 0
_WAIT_UTILS_SOURCED=1

# Source required utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/log.sh" 2>/dev/null || true

#######################################
# Wait for a condition to be true with timeout and progress indication
# Args: $1 - test_command, $2 - timeout_seconds, $3 - description
# Returns: 0 if condition met, 1 if timeout
#######################################
wait::for_condition() {
    local test_command="$1"
    local timeout_seconds="${2:-30}"
    local description="${3:-condition}"
    
    log::info "Waiting for $description..."
    
    local elapsed=0
    while [ $elapsed -lt $timeout_seconds ]; do
        if eval "$test_command" >/dev/null 2>&1; then
            echo
            log::success "$description is ready!"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "$description failed to be ready after $timeout_seconds seconds"
    return 1
}

#######################################
# Wait for a service to respond on a port
# Args: $1 - host, $2 - port, $3 - timeout (optional, default 60)
# Returns: 0 if port is open, 1 if timeout
#######################################
wait::for_port() {
    local host="$1"
    local port="$2"
    local timeout="${3:-60}"
    
    log::info "Waiting for $host:$port to be available..."
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if command -v nc >/dev/null && nc -z "$host" "$port" 2>/dev/null; then
            echo
            log::success "Port $host:$port is available!"
            return 0
        elif command -v timeout >/dev/null && timeout 1 bash -c "</dev/tcp/$host/$port" 2>/dev/null; then
            echo
            log::success "Port $host:$port is available!"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "Port $host:$port failed to become available after $timeout seconds"
    return 1
}

#######################################
# Wait for HTTP endpoint to respond
# Args: $1 - url, $2 - expected_code, $3 - timeout (optional), $4 - headers (optional)
# Returns: 0 if endpoint responds, 1 if timeout
#######################################
wait::for_http() {
    local url="$1"
    local expected_code="${2:-200}"
    local timeout="${3:-60}"
    local headers="${4:-}"
    
    log::info "Waiting for HTTP endpoint: $url"
    
    local elapsed=0
    local curl_args=("-s" "-w" "%{http_code}" "-o" "/dev/null" "--max-time" "5")
    
    # Add headers if provided
    if [[ -n "$headers" ]]; then
        while IFS= read -r header; do
            [[ -n "$header" ]] && curl_args+=("-H" "$header")
        done <<< "$headers"
    fi
    
    while [ $elapsed -lt $timeout ]; do
        local http_code
        http_code=$(curl "${curl_args[@]}" "$url" 2>/dev/null || echo "000")
        
        if [ "$http_code" = "$expected_code" ]; then
            echo
            log::success "HTTP endpoint is responding: $url"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "HTTP endpoint failed to respond after $timeout seconds: $url"
    return 1
}

#######################################
# Wait for file to exist
# Args: $1 - file_path, $2 - timeout (optional, default 30)
# Returns: 0 if file exists, 1 if timeout
#######################################
wait::for_file() {
    local file_path="$1"
    local timeout="${2:-30}"
    
    log::info "Waiting for file: $file_path"
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if [[ -f "$file_path" ]]; then
            echo
            log::success "File exists: $file_path"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "File failed to appear after $timeout seconds: $file_path"
    return 1
}

#######################################
# Wait for directory to exist
# Args: $1 - dir_path, $2 - timeout (optional, default 30)
# Returns: 0 if directory exists, 1 if timeout
#######################################
wait::for_directory() {
    local dir_path="$1"
    local timeout="${2:-30}"
    
    log::info "Waiting for directory: $dir_path"
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if [[ -d "$dir_path" ]]; then
            echo
            log::success "Directory exists: $dir_path"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "Directory failed to appear after $timeout seconds: $dir_path"
    return 1
}

#######################################
# Retry a command with exponential backoff
# Args: $1 - max_attempts, $2 - initial_delay, $3 - command, $4+ - command args
# Returns: Last command exit code
#######################################
wait::retry_with_backoff() {
    local max_attempts="$1"
    local initial_delay="$2"
    shift 2
    
    local attempt=1
    local delay="$initial_delay"
    
    while [ $attempt -le $max_attempts ]; do
        if "$@"; then
            return 0
        fi
        
        if [ $attempt -lt $max_attempts ]; then
            log::warn "Command failed (attempt $attempt/$max_attempts), retrying in ${delay}s..."
            sleep "$delay"
            delay=$((delay * 2))  # Exponential backoff
        fi
        
        attempt=$((attempt + 1))
    done
    
    log::error "Command failed after $max_attempts attempts"
    return 1
}

#######################################
# Simple retry with fixed delay
# Args: $1 - max_attempts, $2 - delay, $3 - command, $4+ - command args  
# Returns: Last command exit code
#######################################
wait::retry() {
    local max_attempts="$1"
    local delay="$2"
    shift 2
    
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if "$@"; then
            return 0
        fi
        
        if [ $attempt -lt $max_attempts ]; then
            log::warn "Command failed (attempt $attempt/$max_attempts), retrying in ${delay}s..."
            sleep "$delay"
        fi
        
        attempt=$((attempt + 1))
    done
    
    log::error "Command failed after $max_attempts attempts"
    return 1
}

#######################################
# Wait for process to be running
# Args: $1 - process_name, $2 - timeout (optional, default 30)
# Returns: 0 if process found, 1 if timeout
#######################################
wait::for_process() {
    local process_name="$1"
    local timeout="${2:-30}"
    
    log::info "Waiting for process: $process_name"
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if pgrep -f "$process_name" >/dev/null; then
            echo
            log::success "Process is running: $process_name"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "Process failed to start after $timeout seconds: $process_name"
    return 1
}

#######################################
# Wait for process to stop
# Args: $1 - process_name, $2 - timeout (optional, default 30)
# Returns: 0 if process stopped, 1 if timeout
#######################################
wait::for_process_stop() {
    local process_name="$1"
    local timeout="${2:-30}"
    
    log::info "Waiting for process to stop: $process_name"
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if ! pgrep -f "$process_name" >/dev/null; then
            echo
            log::success "Process has stopped: $process_name"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "Process failed to stop after $timeout seconds: $process_name"
    return 1
}

#######################################
# Wait for service with health check
# Args: $1 - service_name, $2 - health_check_command, $3 - timeout (optional)
# Returns: 0 if service healthy, 1 if timeout
#######################################
wait::for_service() {
    local service_name="$1"
    local health_check_command="$2"
    local timeout="${3:-60}"
    
    log::info "Waiting for service to be healthy: $service_name"
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if eval "$health_check_command" >/dev/null 2>&1; then
            echo
            log::success "Service is healthy: $service_name"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "Service failed to become healthy after $timeout seconds: $service_name"
    return 1
}

#######################################
# Wait for multiple conditions with parallel checking
# Args: $1 - timeout, $2+ - condition commands
# Returns: 0 if all conditions met, 1 if timeout
#######################################
wait::for_all_conditions() {
    local timeout="$1"
    shift
    local conditions=("$@")
    
    log::info "Waiting for ${#conditions[@]} conditions to be met..."
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        local all_met=true
        
        for condition in "${conditions[@]}"; do
            if ! eval "$condition" >/dev/null 2>&1; then
                all_met=false
                break
            fi
        done
        
        if [[ "$all_met" == "true" ]]; then
            echo
            log::success "All conditions are met!"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "Not all conditions were met after $timeout seconds"
    return 1
}

#######################################
# Wait with custom progress indicator
# Args: $1 - test_command, $2 - timeout, $3 - progress_message, $4 - success_message
# Returns: 0 if condition met, 1 if timeout
#######################################
wait::with_progress() {
    local test_command="$1"
    local timeout="$2"
    local progress_message="$3"
    local success_message="$4"
    
    echo -n "$progress_message"
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if eval "$test_command" >/dev/null 2>&1; then
            echo
            log::success "$success_message"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    return 1
}