#!/usr/bin/env bash
# Cleanup Manager - Enhanced cleanup with automatic registration and aggressive cleanup
# This provides automatic cleanup registration, trap handlers, and periodic cleanup

set -euo pipefail

# Script directory
if [[ -z "${CLEANUP_MANAGER_SCRIPT_DIR:-}" ]]; then
    CLEANUP_MANAGER_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    readonly CLEANUP_MANAGER_SCRIPT_DIR
fi

# Load the base cleanup functions
source "$CLEANUP_MANAGER_SCRIPT_DIR/cleanup.bash"

# Cleanup configuration
readonly CLEANUP_AGE_HOURS="${CLEANUP_AGE_HOURS:-24}"  # Clean resources older than X hours
readonly CLEANUP_PATTERNS=(
    "vrooli_test_*"
    "vrooli-test-*"
    "bats-run-*"
    "test_*_$$"
    "parallel-test-*"
    "*_test_namespace_*"
)

# Track cleanup registration
CLEANUP_REGISTERED=false
CLEANUP_EXIT_CODE=0

#######################################
# Register automatic cleanup on script exit
# Sets up trap handlers for various signals
# Arguments: None
# Returns: 0
#######################################
vrooli_register_cleanup() {
    if [[ "$CLEANUP_REGISTERED" == "true" ]]; then
        echo "[CLEANUP-MANAGER] Cleanup already registered"
        return 0
    fi
    
    # Skip automatic trap registration in BATS context
    # BATS manages its own test lifecycle and cleanup
    if [[ -n "${BATS_VERSION:-}" || -n "${BATS_TEST_FILENAME:-}" ]]; then
        echo "[CLEANUP-MANAGER] Skipping trap registration in BATS context"
        CLEANUP_REGISTERED=true
        return 0
    fi
    
    echo "[CLEANUP-MANAGER] Registering automatic cleanup handlers"
    
    # Create a cleanup handler function
    _cleanup_handler() {
        local exit_code=$?
        echo "[CLEANUP-MANAGER] Cleanup handler triggered (exit code: $exit_code)"
        
        # Run cleanup
        vrooli_aggressive_cleanup
        
        # Validate cleanup
        vrooli_validate_cleanup || true
        
        # Exit with original code
        exit $exit_code
    }
    
    # Register trap handlers for various signals
    trap _cleanup_handler EXIT
    trap _cleanup_handler INT
    trap _cleanup_handler TERM
    trap _cleanup_handler HUP
    
    CLEANUP_REGISTERED=true
    echo "[CLEANUP-MANAGER] Cleanup handlers registered successfully"
    return 0
}

#######################################
# Aggressive cleanup - removes all test resources
# Including old resources without namespaces
# Arguments: None
# Returns: 0
#######################################
vrooli_aggressive_cleanup() {
    echo "[CLEANUP-MANAGER] Starting aggressive cleanup"
    
    local cleaned_count=0
    
    # First run normal cleanup
    vrooli_cleanup_test || true
    
    # Clean up /tmp directories matching patterns
    echo "[CLEANUP-MANAGER] Cleaning /tmp directories"
    for pattern in "${CLEANUP_PATTERNS[@]}"; do
        local dirs
        dirs=$(find /tmp -maxdepth 2 -name "$pattern" -type d 2>/dev/null || true)
        
        if [[ -n "$dirs" ]]; then
            echo "$dirs" | while IFS= read -r dir; do
                if [[ -d "$dir" ]]; then
                    if rm -rf "$dir" 2>/dev/null; then
                        echo "[CLEANUP-MANAGER] Removed: $dir"
                        ((cleaned_count++))
                    fi
                fi
            done
        fi
        
        # Also clean files
        local files
        files=$(find /tmp -maxdepth 2 -name "$pattern" -type f 2>/dev/null || true)
        
        if [[ -n "$files" ]]; then
            echo "$files" | while IFS= read -r file; do
                if [[ -f "$file" ]]; then
                    if rm "$file" 2>/dev/null; then
                        echo "[CLEANUP-MANAGER] Removed: $file"
                        ((cleaned_count++))
                    fi
                fi
            done
        fi
    done
    
    # Clean up /dev/shm
    if [[ -d "/dev/shm" ]]; then
        echo "[CLEANUP-MANAGER] Cleaning /dev/shm"
        for pattern in "${CLEANUP_PATTERNS[@]}"; do
            local items
            items=$(find /dev/shm -maxdepth 2 -name "$pattern" 2>/dev/null || true)
            
            if [[ -n "$items" ]]; then
                echo "$items" | while IFS= read -r item; do
                    if [[ -e "$item" ]]; then
                        if rm -rf "$item" 2>/dev/null; then
                            echo "[CLEANUP-MANAGER] Removed: $item"
                            ((cleaned_count++))
                        fi
                    fi
                done
            fi
        done
    fi
    
    # Clean up specific known problematic directories
    local known_dirs=(
        "/tmp/vrooli-parallel-results"
        "/tmp/vrooli-parallel-locks"
        "/tmp/vrooli-test-cache"
        "/tmp/bats"
        "/dev/shm/vrooli-tests"
    )
    
    for dir in "${known_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            if rm -rf "$dir" 2>/dev/null; then
                echo "[CLEANUP-MANAGER] Removed known directory: $dir"
                ((cleaned_count++))
            fi
        fi
    done
    
    # Clean up orphaned named pipes and sockets
    echo "[CLEANUP-MANAGER] Cleaning orphaned pipes and sockets"
    find /tmp -type p -name "*vrooli*" -delete 2>/dev/null || true
    find /tmp -type s -name "*vrooli*" -delete 2>/dev/null || true
    
    echo "[CLEANUP-MANAGER] Aggressive cleanup complete (cleaned $cleaned_count items)"
    return 0
}

#######################################
# Clean up old test resources based on age
# Removes resources older than CLEANUP_AGE_HOURS
# Arguments: None
# Returns: 0
#######################################
vrooli_cleanup_old_resources() {
    echo "[CLEANUP-MANAGER] Cleaning resources older than ${CLEANUP_AGE_HOURS} hours"
    
    local cleaned_count=0
    local age_minutes=$((CLEANUP_AGE_HOURS * 60))
    
    # Clean old resources in /tmp
    for pattern in "${CLEANUP_PATTERNS[@]}"; do
        local old_items
        old_items=$(find /tmp -maxdepth 2 -name "$pattern" -mmin +$age_minutes 2>/dev/null || true)
        
        if [[ -n "$old_items" ]]; then
            echo "$old_items" | while IFS= read -r item; do
                if [[ -e "$item" ]]; then
                    if rm -rf "$item" 2>/dev/null; then
                        echo "[CLEANUP-MANAGER] Removed old resource: $item"
                        ((cleaned_count++))
                    fi
                fi
            done
        fi
    done
    
    # Clean old resources in /dev/shm
    if [[ -d "/dev/shm" ]]; then
        for pattern in "${CLEANUP_PATTERNS[@]}"; do
            local old_items
            old_items=$(find /dev/shm -maxdepth 2 -name "$pattern" -mmin +$age_minutes 2>/dev/null || true)
            
            if [[ -n "$old_items" ]]; then
                echo "$old_items" | while IFS= read -r item; do
                    if [[ -e "$item" ]]; then
                        if rm -rf "$item" 2>/dev/null; then
                            echo "[CLEANUP-MANAGER] Removed old resource: $item"
                            ((cleaned_count++))
                        fi
                    fi
                done
            fi
        done
    fi
    
    echo "[CLEANUP-MANAGER] Old resource cleanup complete (cleaned $cleaned_count items)"
    return 0
}

#######################################
# Full system cleanup - most aggressive
# Cleans everything including Docker containers
# Arguments: None
# Returns: 0
#######################################
vrooli_full_system_cleanup() {
    echo "[CLEANUP-MANAGER] Starting full system cleanup"
    
    # Run all cleanup methods
    vrooli_aggressive_cleanup
    vrooli_cleanup_old_resources
    
    # Clean up Docker test containers if Docker is available
    if command -v docker >/dev/null 2>&1; then
        echo "[CLEANUP-MANAGER] Cleaning Docker test containers"
        
        # Remove test containers
        docker ps -a --filter "name=vrooli_test" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null || true
        docker ps -a --filter "name=test_" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null || true
        docker ps -a --filter "label=vrooli.test=true" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null || true
        
        # Clean up test networks
        docker network ls --filter "name=vrooli_test" --format "{{.ID}}" | xargs -r docker network rm 2>/dev/null || true
        
        # Clean up test volumes
        docker volume ls --filter "name=vrooli_test" --format "{{.Name}}" | xargs -r docker volume rm 2>/dev/null || true
    fi
    
    # Kill any remaining test processes
    echo "[CLEANUP-MANAGER] Killing remaining test processes"
    pkill -f "vrooli.*test" 2>/dev/null || true
    pkill -f "bats.*run" 2>/dev/null || true
    
    echo "[CLEANUP-MANAGER] Full system cleanup complete"
    return 0
}

#######################################
# Smart cleanup - decides what level of cleanup to use
# Based on available resources and test state
# Arguments: None
# Returns: 0
#######################################
vrooli_smart_cleanup() {
    echo "[CLEANUP-MANAGER] Running smart cleanup"
    
    # Check disk usage in /tmp
    local tmp_usage
    tmp_usage=$(df /tmp | awk 'NR==2 {print $5}' | sed 's/%//')
    
    # Check disk usage in /dev/shm if it exists
    local shm_usage=0
    if [[ -d "/dev/shm" ]]; then
        shm_usage=$(df /dev/shm | awk 'NR==2 {print $5}' | sed 's/%//')
    fi
    
    # Determine cleanup level based on usage
    if [[ "$tmp_usage" -gt 80 || "$shm_usage" -gt 80 ]]; then
        echo "[CLEANUP-MANAGER] High disk usage detected (tmp: ${tmp_usage}%, shm: ${shm_usage}%)"
        vrooli_full_system_cleanup
    elif [[ "$tmp_usage" -gt 50 || "$shm_usage" -gt 50 ]]; then
        echo "[CLEANUP-MANAGER] Moderate disk usage detected"
        vrooli_aggressive_cleanup
    else
        echo "[CLEANUP-MANAGER] Normal disk usage"
        vrooli_cleanup_test
    fi
    
    # Always clean old resources
    vrooli_cleanup_old_resources
    
    echo "[CLEANUP-MANAGER] Smart cleanup complete"
    return 0
}

#######################################
# Initialize cleanup for test runner
# Call this at the start of test execution
# Arguments:
#   $1 - Test runner name (optional)
# Returns: 0
#######################################
vrooli_init_cleanup() {
    local runner_name="${1:-test-runner}"
    
    echo "[CLEANUP-MANAGER] Initializing cleanup for: $runner_name"
    
    # Register cleanup handlers
    vrooli_register_cleanup
    
    # Clean up any old resources before starting
    vrooli_cleanup_old_resources
    
    # Set up test namespace if not already set
    if [[ -z "${TEST_NAMESPACE:-}" ]]; then
        export TEST_NAMESPACE="vrooli_test_$$_$(date +%s)"
        echo "[CLEANUP-MANAGER] Created test namespace: $TEST_NAMESPACE"
    fi
    
    # Create test temp directory if not exists
    if [[ -z "${VROOLI_TEST_TMPDIR:-}" ]]; then
        export VROOLI_TEST_TMPDIR="/tmp/$TEST_NAMESPACE"
        mkdir -p "$VROOLI_TEST_TMPDIR"
        echo "[CLEANUP-MANAGER] Created test temp directory: $VROOLI_TEST_TMPDIR"
    fi
    
    echo "[CLEANUP-MANAGER] Cleanup initialization complete"
    return 0
}

#######################################
# Cleanup status report
# Shows current resource usage and cleanup recommendations
# Arguments: None
# Returns: 0
#######################################
vrooli_cleanup_status() {
    echo "======================================="
    echo "     Cleanup Status Report"
    echo "======================================="
    
    # Count test resources in /tmp
    local tmp_count=0
    for pattern in "${CLEANUP_PATTERNS[@]}"; do
        local count
        count=$(find /tmp -maxdepth 2 -name "$pattern" 2>/dev/null | wc -l)
        tmp_count=$((tmp_count + count))
    done
    echo "/tmp test resources: $tmp_count"
    
    # Count test resources in /dev/shm
    local shm_count=0
    if [[ -d "/dev/shm" ]]; then
        for pattern in "${CLEANUP_PATTERNS[@]}"; do
            local count
            count=$(find /dev/shm -maxdepth 2 -name "$pattern" 2>/dev/null | wc -l)
            shm_count=$((shm_count + count))
        done
    fi
    echo "/dev/shm test resources: $shm_count"
    
    # Count old resources
    local old_count=0
    local age_minutes=$((CLEANUP_AGE_HOURS * 60))
    for pattern in "${CLEANUP_PATTERNS[@]}"; do
        local count
        count=$(find /tmp /dev/shm -maxdepth 2 -name "$pattern" -mmin +$age_minutes 2>/dev/null | wc -l)
        old_count=$((old_count + count))
    done
    echo "Old resources (>${CLEANUP_AGE_HOURS}h): $old_count"
    
    # Disk usage
    echo ""
    echo "Disk Usage:"
    df -h /tmp | awk 'NR==1 || NR==2'
    [[ -d "/dev/shm" ]] && df -h /dev/shm | awk 'NR==2'
    
    # Recommendations
    echo ""
    echo "Recommendations:"
    if [[ $old_count -gt 10 ]]; then
        echo "- Run: vrooli_cleanup_old_resources"
    fi
    if [[ $tmp_count -gt 50 || $shm_count -gt 50 ]]; then
        echo "- Run: vrooli_aggressive_cleanup"
    fi
    if [[ $tmp_count -gt 100 || $shm_count -gt 100 ]]; then
        echo "- Run: vrooli_full_system_cleanup"
    fi
    
    echo "======================================="
    return 0
}

# Export all cleanup manager functions
export -f vrooli_register_cleanup vrooli_aggressive_cleanup
export -f vrooli_cleanup_old_resources vrooli_full_system_cleanup
export -f vrooli_smart_cleanup vrooli_init_cleanup
export -f vrooli_cleanup_status

# Auto-register cleanup if sourced from a test runner
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    echo "[CLEANUP-MANAGER] Cleanup manager loaded (source mode)"
    # Don't auto-register in source mode - let the caller decide
else
    echo "[CLEANUP-MANAGER] Cleanup manager loaded (script mode)"
    # If run directly, show status
    vrooli_cleanup_status
fi