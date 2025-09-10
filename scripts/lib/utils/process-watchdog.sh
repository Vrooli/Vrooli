#!/usr/bin/env bash
################################################################################
# Process Watchdog - Automatically detect and kill stuck processes
# 
# Prevents system lag from runaway processes like lsof, npm, resource status checks
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true

################################################################################
# WATCHDOG CONFIGURATION
################################################################################

# Process patterns that are known to hang
WATCHDOG_PATTERNS=(
    "lsof.*-iTCP"
    "npm view.*@.*latest"
    "resource-.*status.*--format json"
    "timeout.*resource-"
    "scenario.*status.*--json"  # Add app-monitor scenario status calls
)

# Maximum CPU percentage before considering process stuck (over 5 seconds)
MAX_CPU_THRESHOLD=80

# Maximum runtime for these processes (seconds)
# Reduced from 30s to 10s for lsof to catch stuck processes faster
MAX_RUNTIME_THRESHOLD=10

################################################################################
# WATCHDOG FUNCTIONS
################################################################################

#######################################
# Check for stuck processes and kill them
# Returns:
#   Number of processes killed
#######################################
watchdog::kill_stuck_processes() {
    local killed_count=0
    local current_time=$(date +%s)
    
    # Check each pattern
    for pattern in "${WATCHDOG_PATTERNS[@]}"; do
        # Find processes matching pattern
        local pids
        pids=$(ps aux | grep -E "$pattern" | grep -v grep | awk '{print $2}' 2>/dev/null || true)
        
        for pid in $pids; do
            # Skip if not a valid PID
            [[ "$pid" =~ ^[0-9]+$ ]] || continue
            
            # Get process info
            local process_info
            process_info=$(ps -p "$pid" -o pid,etime,pcpu,comm 2>/dev/null || continue)
            
            # Parse runtime (convert etime to seconds)
            local etime
            etime=$(echo "$process_info" | tail -1 | awk '{print $2}')
            local runtime_seconds=0
            
            # Parse etime format (could be MM:SS or HH:MM:SS or DD-HH:MM:SS)
            if [[ "$etime" =~ ^([0-9]+)-([0-9]+):([0-9]+):([0-9]+)$ ]]; then
                # DD-HH:MM:SS
                runtime_seconds=$((${BASH_REMATCH[1]} * 86400 + ${BASH_REMATCH[2]} * 3600 + ${BASH_REMATCH[3]} * 60 + ${BASH_REMATCH[4]}))
            elif [[ "$etime" =~ ^([0-9]+):([0-9]+):([0-9]+)$ ]]; then
                # HH:MM:SS
                runtime_seconds=$((${BASH_REMATCH[1]} * 3600 + ${BASH_REMATCH[2]} * 60 + ${BASH_REMATCH[3]}))
            elif [[ "$etime" =~ ^([0-9]+):([0-9]+)$ ]]; then
                # MM:SS
                runtime_seconds=$((${BASH_REMATCH[1]} * 60 + ${BASH_REMATCH[2]}))
            fi
            
            # Parse CPU usage
            local cpu_usage
            cpu_usage=$(echo "$process_info" | tail -1 | awk '{print $3}' | cut -d. -f1)
            
            # Check if process should be killed
            local should_kill=false
            local kill_reason=""
            
            # Special handling for lsof - more aggressive thresholds
            local process_cmd
            process_cmd=$(ps -p "$pid" -o comm= 2>/dev/null || echo "")
            
            if [[ "$process_cmd" == "lsof" ]]; then
                # Kill lsof after just 5 seconds (they should complete in <1s normally)
                if [[ $runtime_seconds -gt 5 ]]; then
                    should_kill=true
                    kill_reason="lsof exceeded 5s runtime (${runtime_seconds}s)"
                fi
                # Or if using >50% CPU for more than 2 seconds
                if [[ $runtime_seconds -gt 2 ]] && [[ ${cpu_usage:-0} -gt 50 ]]; then
                    should_kill=true
                    kill_reason="lsof high CPU usage (${cpu_usage}%) for ${runtime_seconds}s"
                fi
            else
                # Normal thresholds for other processes
                # Check runtime threshold
                if [[ $runtime_seconds -gt $MAX_RUNTIME_THRESHOLD ]]; then
                    should_kill=true
                    kill_reason="exceeded runtime threshold (${runtime_seconds}s > ${MAX_RUNTIME_THRESHOLD}s)"
                fi
                
                # Check CPU threshold (only if running for more than 5 seconds)
                if [[ $runtime_seconds -gt 5 ]] && [[ ${cpu_usage:-0} -gt $MAX_CPU_THRESHOLD ]]; then
                    should_kill=true
                    kill_reason="high CPU usage (${cpu_usage}% > ${MAX_CPU_THRESHOLD}%) for ${runtime_seconds}s"
                fi
            fi
            
            # Kill the process if needed
            if [[ "$should_kill" == "true" ]]; then
                local cmd
                cmd=$(ps -p "$pid" -o args= 2>/dev/null | head -c 100 || echo "unknown")
                
                log::warn "Killing stuck process: PID=$pid, Reason=$kill_reason"
                log::debug "Command: $cmd"
                
                # Try graceful termination first
                kill -TERM "$pid" 2>/dev/null || true
                sleep 0.5
                
                # Force kill if still running
                if kill -0 "$pid" 2>/dev/null; then
                    kill -KILL "$pid" 2>/dev/null || true
                fi
                
                ((killed_count++))
            fi
        done
    done
    
    echo "$killed_count"
}

#######################################
# Monitor system and clean stuck processes
# Arguments:
#   $1 - Run once (true) or continuous (false)
#######################################
watchdog::monitor() {
    local run_once="${1:-true}"
    
    log::info "Starting process watchdog (mode: $([ "$run_once" == "true" ] && echo "single run" || echo "continuous"))"
    
    while true; do
        local killed
        killed=$(watchdog::kill_stuck_processes)
        
        if [[ $killed -gt 0 ]]; then
            log::info "Watchdog killed $killed stuck process(es)"
        fi
        
        # Break if running once
        [[ "$run_once" == "true" ]] && break
        
        # Sleep before next check (30 seconds for continuous mode)
        sleep 30
    done
}

#######################################
# Add protection wrapper around commands
# Arguments:
#   $@ - Command to run with protection
#######################################
watchdog::protected_run() {
    local timeout_seconds="${WATCHDOG_TIMEOUT:-10}"
    
    # Run command with timeout
    timeout "$timeout_seconds" "$@" 2>&1
    local exit_code=$?
    
    # Check if command timed out
    if [[ $exit_code -eq 124 ]]; then
        log::warn "Command timed out after ${timeout_seconds}s: $*"
        
        # Try to find and kill any related processes
        local cmd_pattern="${1##*/}"  # Get base command name
        local pids
        pids=$(pgrep -f "$cmd_pattern" 2>/dev/null || true)
        
        for pid in $pids; do
            kill -TERM "$pid" 2>/dev/null || true
        done
    fi
    
    return $exit_code
}

# Export functions for use by other scripts
export -f watchdog::kill_stuck_processes
export -f watchdog::monitor
export -f watchdog::protected_run

# If script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Parse arguments
    case "${1:-monitor}" in
        monitor)
            watchdog::monitor "${2:-true}"
            ;;
        continuous)
            watchdog::monitor false
            ;;
        kill)
            killed=$(watchdog::kill_stuck_processes)
            echo "Killed $killed stuck process(es)"
            ;;
        *)
            echo "Usage: $0 [monitor|continuous|kill]"
            echo "  monitor    - Run once and exit (default)"
            echo "  continuous - Run continuously"
            echo "  kill       - Kill stuck processes immediately"
            exit 1
            ;;
    esac
fi