#!/usr/bin/env bash
# Test Isolation - Namespace management and environment isolation
# Provides test isolation, cleanup registration, and resource tracking

# Prevent duplicate loading
if [[ "${VROOLI_TEST_ISOLATION_LOADED:-}" == "true" ]]; then
    return 0
fi
export VROOLI_TEST_ISOLATION_LOADED="true"

# Load dependencies
SHARED_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHARED_DIR/logging.bash"
source "$SHARED_DIR/utils.bash"

# Test isolation configuration
readonly ISOLATION_BASE_DIR="${VROOLI_ISOLATION_BASE_DIR:-/tmp/vrooli-test-isolation}"
readonly ISOLATION_NAMESPACE_PREFIX="${VROOLI_ISOLATION_NAMESPACE_PREFIX:-vrooli_test}"
readonly ISOLATION_CLEANUP_AGE="${VROOLI_ISOLATION_CLEANUP_AGE:-3600}" # 1 hour in seconds

# Tracking arrays
declare -a CLEANUP_FUNCTIONS=()
declare -a CLEANUP_DIRECTORIES=()
declare -a CLEANUP_FILES=()
declare -a CLEANUP_PROCESSES=()
declare -a CLEANUP_CONTAINERS=()

# Current test namespace
export TEST_NAMESPACE="${TEST_NAMESPACE:-}"

#######################################
# Initialize test isolation
# Creates namespace and sets up environment
# Arguments:
#   $1 - Test name (optional)
# Returns: 0 on success
# Outputs: Namespace to stdout
#######################################
vrooli_isolation_init() {
    local test_name="${1:-test}"
    
    vrooli_log_debug "ISOLATION" "Initializing test isolation for: $test_name"
    
    # Create isolation base directory
    if ! mkdir -p "$ISOLATION_BASE_DIR" 2>/dev/null; then
        vrooli_log_error "ISOLATION" "Failed to create isolation base directory: $ISOLATION_BASE_DIR"
        return 1
    fi
    
    # Generate unique namespace
    local timestamp
    timestamp=$(date +%s%N)
    local random
    random=$(vrooli_random_string 6)
    TEST_NAMESPACE="${ISOLATION_NAMESPACE_PREFIX}_${test_name}_${timestamp}_${random}"
    
    # Create namespace directory
    local namespace_dir="$ISOLATION_BASE_DIR/$TEST_NAMESPACE"
    if ! mkdir -p "$namespace_dir" 2>/dev/null; then
        vrooli_log_error "ISOLATION" "Failed to create namespace directory: $namespace_dir"
        return 1
    fi
    
    # Set environment variables
    export TEST_NAMESPACE
    export VROOLI_TEST_NAMESPACE="$TEST_NAMESPACE"
    export VROOLI_TEST_TMPDIR="$namespace_dir/tmp"
    export VROOLI_TEST_DATA_DIR="$namespace_dir/data"
    export VROOLI_TEST_LOG_DIR="$namespace_dir/logs"
    
    # Create subdirectories
    mkdir -p "$VROOLI_TEST_TMPDIR" "$VROOLI_TEST_DATA_DIR" "$VROOLI_TEST_LOG_DIR"
    
    # Track for cleanup
    CLEANUP_DIRECTORIES+=("$namespace_dir")
    
    # Clean up old namespaces
    vrooli_isolation_cleanup_old
    
    vrooli_log_info "ISOLATION" "Test isolation initialized with namespace: $TEST_NAMESPACE"
    echo "$TEST_NAMESPACE"
    return 0
}

#######################################
# Register a cleanup function to be called on exit
# Arguments:
#   $1 - Function name or command to execute
# Returns: 0
#######################################
vrooli_isolation_register_cleanup() {
    local cleanup_item="$1"
    
    CLEANUP_FUNCTIONS+=("$cleanup_item")
    vrooli_log_debug "ISOLATION" "Registered cleanup function: $cleanup_item"
    
    return 0
}

#######################################
# Register a directory for cleanup
# Arguments:
#   $1 - Directory path
# Returns: 0
#######################################
vrooli_isolation_register_directory() {
    local dir="$1"
    
    CLEANUP_DIRECTORIES+=("$dir")
    vrooli_log_debug "ISOLATION" "Registered directory for cleanup: $dir"
    
    return 0
}

#######################################
# Register a file for cleanup
# Arguments:
#   $1 - File path
# Returns: 0
#######################################
vrooli_isolation_register_file() {
    local file="$1"
    
    CLEANUP_FILES+=("$file")
    vrooli_log_debug "ISOLATION" "Registered file for cleanup: $file"
    
    return 0
}

#######################################
# Register a process for cleanup
# Arguments:
#   $1 - Process ID or name pattern
# Returns: 0
#######################################
vrooli_isolation_register_process() {
    local process="$1"
    
    CLEANUP_PROCESSES+=("$process")
    vrooli_log_debug "ISOLATION" "Registered process for cleanup: $process"
    
    return 0
}

#######################################
# Register a Docker container for cleanup
# Arguments:
#   $1 - Container name or ID
# Returns: 0
#######################################
vrooli_isolation_register_container() {
    local container="$1"
    
    CLEANUP_CONTAINERS+=("$container")
    vrooli_log_debug "ISOLATION" "Registered container for cleanup: $container"
    
    return 0
}

#######################################
# Create an isolated temporary directory
# Arguments:
#   $1 - Directory name (optional)
# Returns: 0
# Outputs: Directory path to stdout
#######################################
vrooli_isolation_create_tmpdir() {
    local name="${1:-tmp}"
    
    if [[ -z "$TEST_NAMESPACE" ]]; then
        vrooli_log_error "ISOLATION" "Test isolation not initialized"
        return 1
    fi
    
    local tmpdir="$VROOLI_TEST_TMPDIR/${name}_$(vrooli_random_string 8)"
    
    if mkdir -p "$tmpdir"; then
        vrooli_isolation_register_directory "$tmpdir"
        echo "$tmpdir"
        return 0
    else
        vrooli_log_error "ISOLATION" "Failed to create temporary directory: $tmpdir"
        return 1
    fi
}

#######################################
# Create an isolated file
# Arguments:
#   $1 - File name (optional)
# Returns: 0
# Outputs: File path to stdout
#######################################
vrooli_isolation_create_file() {
    local name="${1:-file}"
    
    if [[ -z "$TEST_NAMESPACE" ]]; then
        vrooli_log_error "ISOLATION" "Test isolation not initialized"
        return 1
    fi
    
    local file="$VROOLI_TEST_TMPDIR/${name}_$(vrooli_random_string 8)"
    
    if touch "$file"; then
        vrooli_isolation_register_file "$file"
        echo "$file"
        return 0
    else
        vrooli_log_error "ISOLATION" "Failed to create file: $file"
        return 1
    fi
}

#######################################
# Execute cleanup for current test
# Arguments: None
# Returns: 0
#######################################
vrooli_isolation_cleanup() {
    vrooli_log_info "ISOLATION" "Starting test isolation cleanup for namespace: $TEST_NAMESPACE"
    
    local errors=0
    
    # Execute registered cleanup functions
    for func in "${CLEANUP_FUNCTIONS[@]}"; do
        vrooli_log_debug "ISOLATION" "Executing cleanup function: $func"
        if ! eval "$func" 2>/dev/null; then
            vrooli_log_warn "ISOLATION" "Cleanup function failed: $func"
            ((errors++))
        fi
    done
    
    # Kill registered processes
    for process in "${CLEANUP_PROCESSES[@]}"; do
        if [[ "$process" =~ ^[0-9]+$ ]]; then
            # It's a PID
            if kill -0 "$process" 2>/dev/null; then
                kill "$process" 2>/dev/null || true
                sleep 1
                kill -9 "$process" 2>/dev/null || true
                vrooli_log_debug "ISOLATION" "Killed process: $process"
            fi
        else
            # It's a pattern
            pkill -f "$process" 2>/dev/null || true
            vrooli_log_debug "ISOLATION" "Killed processes matching: $process"
        fi
    done
    
    # Remove Docker containers
    if command -v docker >/dev/null 2>&1; then
        for container in "${CLEANUP_CONTAINERS[@]}"; do
            if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
                docker stop "$container" 2>/dev/null || true
                docker rm "$container" 2>/dev/null || true
                vrooli_log_debug "ISOLATION" "Removed container: $container"
            fi
        done
        
        # Also clean up any containers with our namespace
        if [[ -n "$TEST_NAMESPACE" ]]; then
            docker ps -a --filter "name=$TEST_NAMESPACE" --format "{{.ID}}" | \
                xargs -r docker rm -f 2>/dev/null || true
        fi
    fi
    
    # Remove registered files
    for file in "${CLEANUP_FILES[@]}"; do
        if [[ -f "$file" ]]; then
            rm -f "$file" 2>/dev/null || true
            vrooli_log_debug "ISOLATION" "Removed file: $file"
        fi
    done
    
    # Remove registered directories
    for dir in "${CLEANUP_DIRECTORIES[@]}"; do
        if [[ -d "$dir" ]]; then
            rm -rf "$dir" 2>/dev/null || true
            vrooli_log_debug "Removed directory: $dir"
        fi
    done
    
    # Clean up namespace-based resources
    if [[ -n "$TEST_NAMESPACE" ]]; then
        # Clean /tmp
        find /tmp -maxdepth 2 -name "*${TEST_NAMESPACE}*" -exec rm -rf {} + 2>/dev/null || true
        
        # Clean /dev/shm
        if [[ -d "/dev/shm" ]]; then
            find /dev/shm -maxdepth 2 -name "*${TEST_NAMESPACE}*" -exec rm -rf {} + 2>/dev/null || true
        fi
    fi
    
    # Clear tracking arrays
    CLEANUP_FUNCTIONS=()
    CLEANUP_DIRECTORIES=()
    CLEANUP_FILES=()
    CLEANUP_PROCESSES=()
    CLEANUP_CONTAINERS=()
    
    vrooli_log_info "ISOLATION" "Test isolation cleanup complete (errors: $errors)"
    return 0
}

#######################################
# Clean up old test namespaces
# Removes namespaces older than ISOLATION_CLEANUP_AGE
# Arguments: None
# Returns: 0
#######################################
vrooli_isolation_cleanup_old() {
    vrooli_log_debug "Cleaning up old test namespaces"
    
    local current_time
    current_time=$(date +%s)
    local cleaned=0
    
    # Clean up old namespace directories
    if [[ -d "$ISOLATION_BASE_DIR" ]]; then
        for namespace_dir in "$ISOLATION_BASE_DIR"/*; do
            [[ -d "$namespace_dir" ]] || continue
            
            # Get directory age
            local dir_time
            dir_time=$(stat -c %Y "$namespace_dir" 2>/dev/null || stat -f %m "$namespace_dir" 2>/dev/null || echo 0)
            local age=$((current_time - dir_time))
            
            if [[ $age -gt $ISOLATION_CLEANUP_AGE ]]; then
                rm -rf "$namespace_dir" 2>/dev/null || true
                ((cleaned++))
                vrooli_log_debug "Removed old namespace: $(basename "$namespace_dir")"
            fi
        done
    fi
    
    # Clean up old files in /tmp
    find /tmp -maxdepth 2 -name "${ISOLATION_NAMESPACE_PREFIX}_*" -mmin +$((ISOLATION_CLEANUP_AGE / 60)) \
        -exec rm -rf {} + 2>/dev/null || true
    
    # Clean up old files in /dev/shm
    if [[ -d "/dev/shm" ]]; then
        find /dev/shm -maxdepth 2 -name "${ISOLATION_NAMESPACE_PREFIX}_*" -mmin +$((ISOLATION_CLEANUP_AGE / 60)) \
            -exec rm -rf {} + 2>/dev/null || true
    fi
    
    vrooli_log_debug "Cleaned up $cleaned old namespaces"
    return 0
}

#######################################
# Set up trap handlers for automatic cleanup
# Call this at the start of your test
# Arguments: None
# Returns: 0
#######################################
vrooli_isolation_setup_traps() {
    vrooli_log_debug "Setting up cleanup trap handlers"
    
    # Define cleanup handler
    _isolation_cleanup_handler() {
        local exit_code=$?
        vrooli_log_debug "Cleanup trap triggered (exit code: $exit_code)"
        vrooli_isolation_cleanup
        exit $exit_code
    }
    
    # Set up traps for various signals
    trap _isolation_cleanup_handler EXIT
    trap _isolation_cleanup_handler INT
    trap _isolation_cleanup_handler TERM
    trap _isolation_cleanup_handler HUP
    
    vrooli_log_info "Cleanup trap handlers installed"
    return 0
}

#######################################
# Get current test namespace
# Arguments: None
# Returns: 0
# Outputs: Namespace to stdout
#######################################
vrooli_isolation_get_namespace() {
    if [[ -z "$TEST_NAMESPACE" ]]; then
        vrooli_log_error "ISOLATION" "Test isolation not initialized"
        return 1
    fi
    
    echo "$TEST_NAMESPACE"
    return 0
}

#######################################
# Check if test isolation is active
# Arguments: None
# Returns: 0 if active, 1 if not
#######################################
vrooli_isolation_is_active() {
    [[ -n "$TEST_NAMESPACE" ]]
}

#######################################
# Get isolation statistics
# Useful for debugging and monitoring
# Arguments: None
# Returns: 0
# Outputs: Statistics to stdout
#######################################
vrooli_isolation_stats() {
    echo "Test Isolation Statistics:"
    echo "  Namespace: ${TEST_NAMESPACE:-not initialized}"
    echo "  Cleanup functions: ${#CLEANUP_FUNCTIONS[@]}"
    echo "  Tracked directories: ${#CLEANUP_DIRECTORIES[@]}"
    echo "  Tracked files: ${#CLEANUP_FILES[@]}"
    echo "  Tracked processes: ${#CLEANUP_PROCESSES[@]}"
    echo "  Tracked containers: ${#CLEANUP_CONTAINERS[@]}"
    
    if [[ -n "$TEST_NAMESPACE" && -d "$ISOLATION_BASE_DIR/$TEST_NAMESPACE" ]]; then
        local size
        size=$(du -sh "$ISOLATION_BASE_DIR/$TEST_NAMESPACE" 2>/dev/null | cut -f1)
        echo "  Namespace disk usage: ${size:-unknown}"
    fi
    
    return 0
}

#######################################
# Export functions for use in tests
#######################################
export -f vrooli_isolation_init vrooli_isolation_register_cleanup
export -f vrooli_isolation_register_directory vrooli_isolation_register_file
export -f vrooli_isolation_register_process vrooli_isolation_register_container
export -f vrooli_isolation_create_tmpdir vrooli_isolation_create_file
export -f vrooli_isolation_cleanup vrooli_isolation_cleanup_old
export -f vrooli_isolation_setup_traps vrooli_isolation_get_namespace
export -f vrooli_isolation_is_active vrooli_isolation_stats

vrooli_log_info "ISOLATION" "Test isolation module loaded successfully"