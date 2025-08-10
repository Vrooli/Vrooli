#!/usr/bin/env bash
# Judge0 Common Utilities
# Shared functions used across Judge0 management modules

# Source required utilities if not already loaded
if ! command -v trash::safe_remove >/dev/null 2>&1; then
    SCRIPT_DIR_TRASH=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR_TRASH}/../../../lib/utils/var.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
fi

# Load dependencies if not already loaded (but preserve test environment)
if [[ -z "${JUDGE0_API_KEY_LENGTH:-}" ]]; then
    # Try to load config from relative path
    SCRIPT_DIR_COMMON=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
    if [[ -f "${SCRIPT_DIR_COMMON}/config/defaults.sh" ]]; then
        # Store test environment variables
        _test_config_dir="${JUDGE0_CONFIG_DIR:-}"
        _test_data_dir="${JUDGE0_DATA_DIR:-}"
        
        # shellcheck disable=SC1091
        source "${SCRIPT_DIR_COMMON}/config/defaults.sh"
        judge0::export_config
        
        # Restore test environment if it was set
        if [[ -n "$_test_config_dir" ]]; then
            export JUDGE0_CONFIG_DIR="$_test_config_dir"
        fi
        if [[ -n "$_test_data_dir" ]]; then
            export JUDGE0_DATA_DIR="$_test_data_dir"
        fi
        
        # Clean up temporary variables
        unset _test_config_dir _test_data_dir
    fi
fi

# Load logging utilities if not available
if ! declare -f log::info >/dev/null 2>&1; then
    # Try to source log utilities
    RESOURCES_DIR_COMMON="${SCRIPT_DIR_COMMON}/../.."
    if [[ -f "${RESOURCES_DIR_COMMON}/../lib/utils/log.sh" ]]; then
        # shellcheck disable=SC1091
        source "${RESOURCES_DIR_COMMON}/../lib/utils/log.sh"
    else
        # Fallback logging functions
        log::info() { echo "[INFO] $*"; }
        log::success() { echo "[SUCCESS] $*"; }
        log::warning() { echo "[WARNING] $*"; }
        log::error() { echo "[ERROR] $*"; }
    fi
fi

#######################################
# Check if Judge0 is installed
# Returns:
#   0 if installed, 1 if not
#######################################
judge0::is_installed() {
    judge0::docker::container_exists
}

#######################################
# Check if Judge0 is running
# Returns:
#   0 if running, 1 if not
#######################################
judge0::is_running() {
    judge0::docker::is_running
}

#######################################
# Get Judge0 API key from config
# Returns:
#   API key or empty string
#######################################
judge0::get_api_key() {
    local config_file="${JUDGE0_CONFIG_DIR}/api_key"
    
    if [[ -f "$config_file" ]]; then
        cat "$config_file"
    else
        echo ""
    fi
}

#######################################
# Save Judge0 API key to config
# Arguments:
#   $1 - API key
#######################################
judge0::save_api_key() {
    local api_key="$1"
    local config_file="${JUDGE0_CONFIG_DIR}/api_key"
    
    # Create config directory if it doesn't exist
    mkdir -p "$JUDGE0_CONFIG_DIR"
    
    # Save API key with restricted permissions
    echo "$api_key" > "$config_file"
    chmod 600 "$config_file"
}

#######################################
# Generate secure API key
# Returns:
#   Generated API key
#######################################
judge0::generate_api_key() {
    # Generate hex string of the specified length (not bytes)
    local hex_length=$((JUDGE0_API_KEY_LENGTH / 2))
    openssl rand -hex "$hex_length" 2>/dev/null || \
        cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w "$JUDGE0_API_KEY_LENGTH" | head -n 1
}

#######################################
# Wait for Judge0 to be healthy
# Arguments:
#   $1 - Timeout in seconds (default: 30)
# Returns:
#   0 if healthy, 1 if timeout
#######################################
judge0::wait_for_health() {
    local timeout="${1:-$JUDGE0_STARTUP_WAIT}"
    local elapsed=0
    
    log::info "Waiting for Judge0 to become healthy..."
    
    while [[ $elapsed -lt $timeout ]]; do
        if judge0::api::health_check >/dev/null 2>&1; then
            log::success "Judge0 is healthy!"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        printf "."
    done
    
    echo
    log::error "Judge0 health check timed out after ${timeout}s"
    return 1
}

#######################################
# Create Judge0 data directories
#######################################
judge0::create_directories() {
    local dirs=(
        "$JUDGE0_DATA_DIR"
        "$JUDGE0_CONFIG_DIR"
        "$JUDGE0_LOGS_DIR"
        "$JUDGE0_SUBMISSIONS_DIR"
    )
    
    for dir in "${dirs[@]}"; do
        if ! mkdir -p "$dir"; then
            log::error "Failed to create directory: $dir"
            return 1
        fi
    done
    
    # Set appropriate permissions
    chmod 700 "$JUDGE0_CONFIG_DIR"
    chmod 755 "$JUDGE0_DATA_DIR" "$JUDGE0_LOGS_DIR" "$JUDGE0_SUBMISSIONS_DIR"
}

#######################################
# Clean up Judge0 data
# Arguments:
#   $1 - Force cleanup (yes/no)
#######################################
judge0::cleanup_data() {
    local force="${1:-no}"
    
    if [[ "$force" != "yes" ]]; then
        log::warning "This will delete all Judge0 data including submissions and logs"
        read -p "Are you sure? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Cleanup cancelled"
            return 0
        fi
    fi
    
    log::info "Cleaning up Judge0 data..."
    
    # Remove data directories
    if [[ -d "$JUDGE0_DATA_DIR" ]]; then
        if command -v trash::safe_remove >/dev/null 2>&1; then
            trash::safe_remove "$JUDGE0_DATA_DIR" --no-confirm
        else
            rm -rf "$JUDGE0_DATA_DIR"
        fi
        log::success "Removed Judge0 data directory"
    fi
}

#######################################
# Get Judge0 container stats
# Returns:
#   JSON object with container stats
#######################################
judge0::get_container_stats() {
    if ! judge0::is_running; then
        echo "{}"
        return
    fi
    
    docker stats "$JUDGE0_CONTAINER_NAME" --no-stream --format "{{json .}}" 2>/dev/null || echo "{}"
}

#######################################
# Get Judge0 worker stats
# Returns:
#   JSON array with worker stats
#######################################
judge0::get_worker_stats() {
    local workers=()
    
    for i in $(seq 1 "$JUDGE0_WORKERS_COUNT"); do
        local worker_name="${JUDGE0_WORKERS_NAME}-${i}"
        if docker::is_running "$worker_name"; then
            local stats=$(docker stats "$worker_name" --no-stream --format "{{json .}}" 2>/dev/null || echo "{}")
            workers+=("$stats")
        fi
    done
    
    # Construct JSON array
    local json="["
    local first=true
    for worker in "${workers[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            json+=","
        fi
        json+="$worker"
    done
    json+="]"
    
    echo "$json"
}

#######################################
# Format bytes to human readable
# Arguments:
#   $1 - Bytes
# Returns:
#   Formatted string
#######################################
judge0::format_bytes() {
    local bytes="$1"
    local units=("B" "KB" "MB" "GB" "TB")
    local unit=0
    
    # Use bc for floating point arithmetic
    while (( $(echo "$bytes >= 1024" | bc -l) )) && (( unit < ${#units[@]} - 1 )); do
        bytes=$(echo "scale=2; $bytes / 1024" | bc -l)
        ((unit++))
    done
    
    printf "%.2f %s" "$bytes" "${units[$unit]}"
}

#######################################
# Validate programming language
# Arguments:
#   $1 - Language name
# Returns:
#   0 if valid, 1 if not
#######################################
judge0::validate_language() {
    local language="$1"
    
    # Check default languages first
    if judge0::get_language_id "$language" >/dev/null; then
        return 0
    fi
    
    # If API is available, check full language list
    if judge0::is_running && judge0::api::get_languages | grep -q "\"name\":\"$language\""; then
        return 0
    fi
    
    return 1
}

#######################################
# Get Judge0 version from running container
# Returns:
#   Version string or "unknown"
#######################################
judge0::get_version() {
    if judge0::is_running; then
        local info=$(judge0::api::system_info 2>/dev/null || echo "{}")
        echo "$info" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown"
    else
        echo "$JUDGE0_VERSION"
    fi
}