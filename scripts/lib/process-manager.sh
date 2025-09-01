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
    
    # Use consistent locking mechanism
    local lock_fd
    if ! exec {lock_fd}<>"$lock_file" 2>/dev/null; then
        echo -e "${RED}✗${NC} Error: $name (cannot access lock file)"
        return 1
    fi
    
    if flock -n "$lock_fd" 2>/dev/null; then
        # Got lock - process is NOT running
        flock -u "$lock_fd" 2>/dev/null || true
        exec {lock_fd}>&- 2>/dev/null || true
        echo -e "${YELLOW}○${NC} Stopped: $name"
        
        # Clean up stale files
        rm -rf "$process_dir" 2>/dev/null || true
    else
        # Lock is held - process might be running
        exec {lock_fd}>&- 2>/dev/null || true
        
        if [[ -f "$pid_file" ]]; then
            local pid
            pid=$(cat "$pid_file" 2>/dev/null || echo "unknown")
            
            # Triple-check the PID is valid and process is actually running
            if [[ "$pid" != "unknown" ]] && [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
                # Get process info for additional verification
                local process_info
                process_info=$(ps -p "$pid" -o pid,ppid,comm,etime --no-headers 2>/dev/null || echo "")
                
                if [[ -n "$process_info" ]]; then
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
                    # PID exists but process not found - cleanup needed
                    echo -e "${RED}✗${NC} Stale: $name (PID $pid not found, cleaning up)"
                    pm::force_cleanup "$name" >/dev/null 2>&1 || true
                fi
            else
                # Invalid PID but lock is held - might be starting or corrupt
                if [[ "$pid" == "unknown" ]] || [[ ! "$pid" =~ ^[0-9]+$ ]]; then
                    echo -e "${BLUE}●${NC} Starting: $name (acquiring PID...)"
                else
                    echo -e "${RED}✗${NC} Error: $name (PID $pid invalid, lock held)"
                fi
            fi
        else
            echo -e "${BLUE}●${NC} Starting: $name (no PID file yet)"
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
    
    local process_dir="$PM_HOME/$name"
    local lock_file="$process_dir/lock"
    local pid_file="$process_dir/pid"
    
    # If process directory doesn't exist, process isn't running
    if [[ ! -d "$process_dir" ]]; then
        return 1
    fi
    
    # If lock file doesn't exist, process isn't running
    if [[ ! -f "$lock_file" ]]; then
        # Clean up stale directory
        rm -rf "$process_dir" 2>/dev/null || true
        return 1
    fi
    
    # Try to acquire lock non-blocking
    local lock_fd
    exec {lock_fd}<>"$lock_file" 2>/dev/null || return 1
    
    if flock -n "$lock_fd" 2>/dev/null; then
        # Got the lock - process is NOT running
        # Check if there's a PID file with a dead process
        if [[ -f "$pid_file" ]]; then
            local pid
            pid=$(cat "$pid_file" 2>/dev/null || echo "")
            if [[ -n "$pid" ]] && ! kill -0 "$pid" 2>/dev/null; then
                # Process is dead, clean up
                rm -rf "$process_dir" 2>/dev/null || true
            fi
        fi
        
        flock -u "$lock_fd" 2>/dev/null || true
        exec {lock_fd}>&- 2>/dev/null || true
        return 1
    else
        # Could not get lock - verify the process is actually running
        exec {lock_fd}>&- 2>/dev/null || true
        
        # Double-check with PID if available
        if [[ -f "$pid_file" ]]; then
            local pid
            pid=$(cat "$pid_file" 2>/dev/null || echo "")
            if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                return 0  # Process is definitely running
            else
                # PID is dead but lock is held - possible race condition
                # Give it a moment and try again
                sleep 0.1
                if [[ -f "$pid_file" ]]; then
                    pid=$(cat "$pid_file" 2>/dev/null || echo "")
                    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                        return 0
                    fi
                fi
                # Process appears to be dead despite lock
                return 1
            fi
        fi
        
        # Lock is held but no PID file - process might be starting
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
    
    # Only list processes that are actually running or have valid state
    for process_dir in "$PM_HOME"/*/; do
        if [[ -d "$process_dir" ]]; then
            local process_name
            process_name=$(basename "$process_dir")
            local lock_file="$process_dir/lock"
            local pid_file="$process_dir/pid"
            
            # Only list if there's evidence this process exists
            if [[ -f "$lock_file" ]]; then
                # Check if lock is held or if there's a valid PID
                local lock_fd
                if exec {lock_fd}<>"$lock_file" 2>/dev/null; then
                    if ! flock -n "$lock_fd" 2>/dev/null; then
                        # Lock is held - process is likely running
                        echo "$process_name"
                        exec {lock_fd}>&- 2>/dev/null || true
                    elif [[ -f "$pid_file" ]]; then
                        local pid
                        pid=$(cat "$pid_file" 2>/dev/null || echo "")
                        if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                            # Process is running but lock not held - transitional state
                            echo "$process_name"
                        else
                            # Stale process - cleanup
                            rm -rf "$process_dir" 2>/dev/null || true
                        fi
                        exec {lock_fd}>&- 2>/dev/null || true
                    else
                        exec {lock_fd}>&- 2>/dev/null || true
                    fi
                else
                    # Cannot access lock file - might be permissions issue
                    if [[ -f "$pid_file" ]]; then
                        local pid
                        pid=$(cat "$pid_file" 2>/dev/null || echo "")
                        if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                            echo "$process_name"
                        fi
                    fi
                fi
            else
                # No lock file - clean up directory
                rm -rf "$process_dir" 2>/dev/null || true
            fi
        fi
    done
}

# Get detailed status information for external integration
pm::get_status_json() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo '{"error": "process name required"}'
        return 1
    fi
    
    local process_dir="$PM_HOME/$name"
    local pid_file="$process_dir/pid"
    local info_file="$process_dir/info"
    local lock_file="$process_dir/lock"
    
    # Default JSON structure
    local status="stopped"
    local pid="null"
    local started="null"
    local command="null"
    local working_dir="null"
    
    # Check if process directory exists
    if [[ ! -d "$process_dir" ]]; then
        echo "{\"name\": \"$name\", \"status\": \"stopped\", \"pid\": null, \"started\": null, \"command\": null, \"working_dir\": null}"
        return 0
    fi
    
    # Check process status using consistent logic
    if pm::is_running "$name"; then
        status="running"
        
        if [[ -f "$pid_file" ]]; then
            local raw_pid
            raw_pid=$(cat "$pid_file" 2>/dev/null || echo "")
            if [[ -n "$raw_pid" ]] && [[ "$raw_pid" =~ ^[0-9]+$ ]]; then
                pid="$raw_pid"
            fi
        fi
        
        if [[ -f "$info_file" ]]; then
            started=$(grep "^started=" "$info_file" 2>/dev/null | cut -d= -f2 | sed 's/"/\\"/g' || echo "")
            command=$(grep "^command=" "$info_file" 2>/dev/null | cut -d= -f2- | sed 's/"/\\"/g' || echo "")
            working_dir=$(grep "^working_dir=" "$info_file" 2>/dev/null | cut -d= -f2 | sed 's/"/\\"/g' || echo "")
        fi
    else
        # Check if there's a stale PID file
        if [[ -f "$pid_file" ]]; then
            local raw_pid
            raw_pid=$(cat "$pid_file" 2>/dev/null || echo "")
            if [[ -n "$raw_pid" ]] && ! kill -0 "$raw_pid" 2>/dev/null; then
                status="failed"
            fi
        fi
    fi
    
    # Build JSON response - escape strings properly
    printf '{\n'
    printf '  "name": "%s",\n' "$name"
    printf '  "status": "%s",\n' "$status"
    printf '  "pid": %s,\n' "${pid:-null}"
    printf '  "started": %s,\n' "${started:+\"$started\"}${started:-null}"
    printf '  "command": %s,\n' "${command:+\"$command\"}${command:-null}"
    printf '  "working_dir": %s\n' "${working_dir:+\"$working_dir\"}${working_dir:-null}"
    printf '}'
    
    return 0
}

# Reconcile process manager status with external status checks
pm::reconcile_status() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        return 1
    fi
    
    # Get our internal status
    local pm_running=false
    if pm::is_running "$name"; then
        pm_running=true
    fi
    
    # If we think it's not running, do a final cleanup check
    if [[ "$pm_running" == "false" ]]; then
        local process_dir="$PM_HOME/$name"
        if [[ -d "$process_dir" ]]; then
            # Clean up stale directory
            rm -rf "$process_dir" 2>/dev/null || true
        fi
    fi
    
    # Return boolean result
    [[ "$pm_running" == "true" ]]
}

# Export functions for use in other scripts
export -f pm::start
export -f pm::stop
export -f pm::status
export -f pm::logs
export -f pm::is_running
export -f pm::force_cleanup
export -f pm::list
export -f pm::get_status_json
export -f pm::reconcile_status
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