#!/usr/bin/env bash
################################################################################
# Vrooli Process Manager
# 
# Simple, reliable process management with singleton enforcement.
# Uses file locks (flock) to guarantee only one instance per process.
#
# Features:
# - Singleton enforcement via flock
# - Crash-safe (kernel releases locks automatically)
# - Zero background processes
# - Simple PID-based management
# - ~300 lines instead of 3,400+
#
# Usage:
#   source process-manager.sh
#   pm::start "name" "command" "/working/dir"
#   pm::stop "name"
#   pm::status "name"
#   pm::logs "name"
#
################################################################################

set -o nounset  # Error on undefined variables
set -o errtrace # Inherit trap ERR

# Configuration
PM_HOME="${PM_HOME:-$HOME/.vrooli/processes}"
PM_LOG_DIR="${PM_LOG_DIR:-$HOME/.vrooli/logs}"

# Colors for output (only if terminal)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

################################################################################
# Core Functions
################################################################################

# Start a process with singleton enforcement
pm::start() {
    local name="${1:-}"
    local command="${2:-}"
    local working_dir="${3:-$(pwd)}"
    
    # Validate arguments
    if [[ -z "$name" ]] || [[ -z "$command" ]]; then
        echo -e "${RED}Error: pm::start requires name and command${NC}" >&2
        return 1
    fi
    
    # Ensure directories exist
    local process_dir="$PM_HOME/$name"
    mkdir -p "$process_dir"
    mkdir -p "$PM_LOG_DIR"
    
    local lock_file="$process_dir/lock"
    local pid_file="$process_dir/pid"
    local info_file="$process_dir/info"
    local log_file="$PM_LOG_DIR/$name.log"
    
    # Try to acquire exclusive lock (non-blocking)
    exec 200>"$lock_file"
    if ! flock -n 200; then
        # Lock is held - check if process is actually running
        if [[ -f "$pid_file" ]]; then
            local existing_pid
            existing_pid=$(cat "$pid_file" 2>/dev/null || echo "")
            
            if [[ -n "$existing_pid" ]] && kill -0 "$existing_pid" 2>/dev/null; then
                echo -e "${YELLOW}Already running: $name (PID: $existing_pid)${NC}"
                return 0
            else
                # Process is dead but lock is held - this shouldn't happen
                # but we'll handle it gracefully
                echo -e "${YELLOW}Found stale lock for $name, cleaning up...${NC}"
                exec 200>&-  # Close our file descriptor
                rm -f "$lock_file" "$pid_file" "$info_file"
                
                # Try again
                exec 200>"$lock_file"
                if ! flock -n 200; then
                    echo -e "${RED}Failed to acquire lock after cleanup${NC}" >&2
                    return 1
                fi
            fi
        else
            # Lock is held but no PID file - another process is starting
            echo -e "${YELLOW}Process $name is starting...${NC}"
            return 0
        fi
    fi
    
    # We have the lock - proceed with starting the process
    echo -e "${BLUE}Starting: $name${NC}"
    
    # Save process info
    cat > "$info_file" <<EOF
command=$command
working_dir=$working_dir
started=$(date -Iseconds)
EOF
    
    # Start the process with lock held
    # Use a wrapper that holds the lock and monitors the actual process
    (
        # Acquire the lock for this wrapper
        exec 200>"$lock_file"
        flock 200
        
        # Change to working directory
        if ! cd "$working_dir"; then
            echo "Failed to change to directory: $working_dir" >&2
            exit 1
        fi
        
        # Start the actual command in background
        bash -c "$command" >> "$log_file" 2>&1 &
        local cmd_pid=$!
        
        # Save the actual command's PID
        echo "$cmd_pid" > "$pid_file"
        
        # Wait for the command to complete
        # This wrapper holds the lock until the command exits
        wait "$cmd_pid"
        local exit_code=$?
        
        # Clean up PID file when command exits
        rm -f "$pid_file"
        
        # Exit with same code as command
        exit $exit_code
    ) &
    
    local launcher_pid=$!
    
    # Wait briefly to ensure process started and PID is written
    sleep 0.5
    
    # Check if the launcher is still running
    if kill -0 "$launcher_pid" 2>/dev/null; then
        # Get the actual PID from the file
        if [[ -f "$pid_file" ]]; then
            local actual_pid
            actual_pid=$(cat "$pid_file" 2>/dev/null)
            
            # Verify the actual process is running
            if [[ -n "$actual_pid" ]] && kill -0 "$actual_pid" 2>/dev/null; then
                echo -e "${GREEN}✓ Started: $name (PID: $actual_pid)${NC}"
                return 0
            else
                echo -e "${RED}✗ Process failed to start: $name${NC}" >&2
                rm -f "$pid_file" "$info_file"
                return 1
            fi
        else
            # PID file not yet created, but launcher is running
            echo -e "${GREEN}✓ Started: $name (starting...)${NC}"
            return 0
        fi
    else
        # Launcher exited - check if it was successful
        if [[ -f "$pid_file" ]]; then
            local actual_pid
            actual_pid=$(cat "$pid_file" 2>/dev/null)
            if [[ -n "$actual_pid" ]] && kill -0 "$actual_pid" 2>/dev/null; then
                echo -e "${GREEN}✓ Started: $name (PID: $actual_pid)${NC}"
                return 0
            fi
        fi
        
        # Process failed to start
        echo -e "${RED}✗ Failed to start: $name${NC}" >&2
        rm -f "$pid_file" "$info_file"
        return 1
    fi
}

# Stop a process
pm::stop() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo -e "${RED}Error: pm::stop requires name${NC}" >&2
        return 1
    fi
    
    local process_dir="$PM_HOME/$name"
    local pid_file="$process_dir/pid"
    local info_file="$process_dir/info"
    local lock_file="$process_dir/lock"
    
    # Check if process directory exists
    if [[ ! -d "$process_dir" ]]; then
        echo -e "${YELLOW}Not running: $name${NC}"
        return 0
    fi
    
    # Check if PID file exists
    if [[ ! -f "$pid_file" ]]; then
        echo -e "${YELLOW}Not running: $name${NC}"
        rm -rf "$process_dir"  # Clean up empty directory
        return 0
    fi
    
    local pid
    pid=$(cat "$pid_file" 2>/dev/null || echo "")
    
    if [[ -z "$pid" ]]; then
        echo -e "${YELLOW}No PID found for: $name${NC}"
        rm -rf "$process_dir"
        return 0
    fi
    
    echo -e "${BLUE}Stopping: $name (PID: $pid)${NC}"
    
    # Send SIGTERM for graceful shutdown
    if kill -0 "$pid" 2>/dev/null; then
        kill -TERM "$pid" 2>/dev/null || true
        
        # Wait up to 5 seconds for graceful shutdown
        local count=0
        while kill -0 "$pid" 2>/dev/null && [[ $count -lt 5 ]]; do
            sleep 1
            count=$((count + 1))
        done
        
        # Force kill if still running
        if kill -0 "$pid" 2>/dev/null; then
            echo -e "${YELLOW}Force killing $name${NC}"
            kill -KILL "$pid" 2>/dev/null || true
            sleep 0.5
        fi
    fi
    
    # Clean up files (lock will be automatically released when process dies)
    rm -f "$pid_file" "$info_file" "$lock_file"
    rmdir "$process_dir" 2>/dev/null || true  # Remove directory if empty
    
    echo -e "${GREEN}✓ Stopped: $name${NC}"
    return 0
}

# Check process status
pm::status() {
    local name="${1:-}"
    
    if [[ -n "$name" ]]; then
        # Single process status
        _pm_status_single "$name"
    else
        # All processes status
        echo "Process Status:"
        echo "────────────────────────────────────────────────────────"
        
        if [[ ! -d "$PM_HOME" ]]; then
            echo "No processes registered"
            return 0
        fi
        
        local found=false
        for process_dir in "$PM_HOME"/*/; do
            if [[ -d "$process_dir" ]]; then
                found=true
                local process_name
                process_name=$(basename "$process_dir")
                _pm_status_single "$process_name"
            fi
        done
        
        if [[ "$found" == "false" ]]; then
            echo "No processes registered"
        fi
    fi
}

# Internal: Check single process status
_pm_status_single() {
    local name="$1"
    local process_dir="$PM_HOME/$name"
    local pid_file="$process_dir/pid"
    local info_file="$process_dir/info"
    local lock_file="$process_dir/lock"
    
    # Check if process directory exists
    if [[ ! -d "$process_dir" ]]; then
        echo -e "${YELLOW}○${NC} Stopped: $name"
        return 0
    fi
    
    # Try to acquire lock to test if process is running
    exec 201>"$lock_file" 2>/dev/null
    if flock -n 201 2>/dev/null; then
        # Got lock - process is NOT running
        flock -u 201 2>/dev/null
        exec 201>&-
        echo -e "${YELLOW}○${NC} Stopped: $name"
        
        # Clean up stale files
        rm -rf "$process_dir" 2>/dev/null
    else
        # Lock is held - process IS running
        exec 201>&-
        
        if [[ -f "$pid_file" ]]; then
            local pid
            pid=$(cat "$pid_file" 2>/dev/null || echo "unknown")
            
            # Double-check the PID is valid
            if [[ "$pid" != "unknown" ]] && kill -0 "$pid" 2>/dev/null; then
                local started=""
                if [[ -f "$info_file" ]]; then
                    started=$(grep "^started=" "$info_file" 2>/dev/null | cut -d= -f2 || echo "")
                    if [[ -n "$started" ]]; then
                        # Format the date nicely
                        started=$(date -d "$started" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "$started")
                    fi
                fi
                
                echo -e "${GREEN}●${NC} Running: $name (PID: $pid${started:+, Started: $started})"
            else
                # PID is invalid but lock is held - shouldn't happen
                echo -e "${RED}✗${NC} Error: $name (lock held but PID $pid invalid)"
            fi
        else
            echo -e "${BLUE}●${NC} Starting: $name (acquiring PID...)"
        fi
    fi
}

# Show process logs
pm::logs() {
    local name="${1:-}"
    local lines="${2:-50}"
    local follow="${3:-false}"
    
    if [[ -z "$name" ]]; then
        echo -e "${RED}Error: pm::logs requires name${NC}" >&2
        return 1
    fi
    
    local log_file="$PM_LOG_DIR/$name.log"
    
    if [[ ! -f "$log_file" ]]; then
        echo -e "${YELLOW}No logs found for: $name${NC}"
        return 1
    fi
    
    if [[ "$follow" == "true" || "$follow" == "--follow" || "$follow" == "-f" ]]; then
        echo -e "${BLUE}Following logs for: $name (Ctrl+C to stop)${NC}"
        echo "────────────────────────────────────────────────────────"
        tail -f "$log_file"
    else
        echo -e "${BLUE}Logs for: $name (last $lines lines)${NC}"
        echo "────────────────────────────────────────────────────────"
        tail -n "$lines" "$log_file"
    fi
}

# Check if process is running (for external scripts)
pm::is_running() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        return 1
    fi
    
    local lock_file="$PM_HOME/$name/lock"
    
    # If lock file doesn't exist, process isn't running
    if [[ ! -f "$lock_file" ]]; then
        return 1
    fi
    
    # Try to acquire lock non-blocking
    exec 202>"$lock_file" 2>/dev/null
    if flock -n 202 2>/dev/null; then
        # Got the lock - process is NOT running
        flock -u 202 2>/dev/null
        exec 202>&-
        return 1
    else
        # Could not get lock - process IS running
        exec 202>&-
        return 0
    fi
}

# Force cleanup (emergency use only)
pm::force_cleanup() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo -e "${RED}Error: pm::force_cleanup requires name${NC}" >&2
        return 1
    fi
    
    echo -e "${RED}⚠ Force cleaning: $name${NC}"
    
    local process_dir="$PM_HOME/$name"
    local pid_file="$process_dir/pid"
    
    # Try to kill by PID if it exists
    if [[ -f "$pid_file" ]]; then
        local pid
        pid=$(cat "$pid_file" 2>/dev/null || echo "")
        if [[ -n "$pid" ]]; then
            echo "  Killing PID: $pid"
            kill -KILL "$pid" 2>/dev/null || true
        fi
    fi
    
    # Remove all files
    echo "  Removing files..."
    rm -rf "$process_dir"
    
    echo -e "${GREEN}✓ Force cleaned: $name${NC}"
}

# List all processes (simple format for scripts)
pm::list() {
    if [[ ! -d "$PM_HOME" ]]; then
        return 0
    fi
    
    for process_dir in "$PM_HOME"/*/; do
        if [[ -d "$process_dir" ]]; then
            basename "$process_dir"
        fi
    done
}

# Export functions for use in other scripts
export -f pm::start
export -f pm::stop
export -f pm::status
export -f pm::logs
export -f pm::is_running
export -f pm::force_cleanup
export -f pm::list
export -f _pm_status_single

# Indicate library is loaded
export PM_LOADED=true

# If script is executed directly, show usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "Vrooli Process Manager"
    echo ""
    echo "This script should be sourced, not executed directly:"
    echo "  source process-manager.sh"
    echo ""
    echo "Available functions:"
    echo "  pm::start <name> <command> [working_dir]"
    echo "  pm::stop <name>"
    echo "  pm::status [name]"
    echo "  pm::logs <name> [lines] [--follow]"
    echo "  pm::is_running <name>"
    echo "  pm::force_cleanup <name>"
    echo "  pm::list"
    exit 1
fi