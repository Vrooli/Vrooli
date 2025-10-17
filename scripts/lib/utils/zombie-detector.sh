#!/usr/bin/env bash
################################################################################
# Zombie/Orphan Process Detector
# 
# Reusable script for detecting zombie processes and orphaned Vrooli processes
# Extracted from status-command.sh to avoid duplication
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true

################################################################################
# ZOMBIE DETECTION
################################################################################

#######################################
# Count zombie processes (ALL defunct processes)
# Returns:
#   Number of zombie processes
#######################################
zombie::count_zombies() {
    ps aux | grep '<defunct>' | grep -v grep | wc -l 2>/dev/null || echo "0"
}

#######################################
# Check for zombie processes for a specific port
# Arguments:
#   $1 - Port number
# Returns:
#   0 if zombies found for port, 1 if not
#######################################
zombie::check_port_zombies() {
    local port="$1"
    local zombie_count
    # Stricter timeout (2s) and nowait flag to prevent lsof from hanging
    zombie_count=$(timeout 2s lsof -iTCP:"$port" -sTCP:LISTEN -P -n 2>/dev/null | grep '<defunct>' | wc -l || echo "0")
    [[ $zombie_count -gt 0 ]]
}

################################################################################
# ORPHAN DETECTION
################################################################################

#######################################
# Count orphaned Vrooli processes
# Uses the same logic as status-command.sh but reusable
# Returns:
#   Number of orphaned processes
#######################################
zombie::count_orphans() {
    bash -c '
        processes_dir="$HOME/.vrooli/processes/scenarios"
        orphan_count=0
        
        # First, build a list of all tracked PIDs
        tracked_pids=""
        if [[ -d "$processes_dir" ]]; then
            for json_file in "$processes_dir"/*/*.json; do
                if [[ -f "$json_file" ]]; then
                    pid=$(jq -r ".pid // empty" "$json_file" 2>/dev/null)
                    [[ -n "$pid" ]] && tracked_pids="$tracked_pids $pid"
                    
                    # Also get PGID if available
                    pgid=$(jq -r ".pgid // empty" "$json_file" 2>/dev/null)
                    [[ -n "$pgid" ]] && tracked_pids="$tracked_pids $pgid"
                fi
            done
        fi
        
        # Function to check if PID or any ancestor is tracked
        is_process_or_ancestor_tracked() {
            local check_pid="$1"
            local max_depth=10  # Prevent infinite loops
            local depth=0
            
            while [[ "$check_pid" != "0" ]] && [[ "$check_pid" != "1" ]] && [[ $depth -lt $max_depth ]]; do
                # Check if this PID is in our tracked list
                for tracked in $tracked_pids; do
                    if [[ "$tracked" == "$check_pid" ]]; then
                        return 0  # Found a tracked ancestor
                    fi
                done
                
                # Get parent PID
                local ppid=$(ps -o ppid= -p "$check_pid" 2>/dev/null | tr -d " ")
                if [[ -z "$ppid" ]] || [[ "$ppid" == "$check_pid" ]]; then
                    break  # No parent or same as current (shouldn'\''t happen)
                fi
                
                check_pid="$ppid"
                ((depth++))
            done
            
            return 1  # No tracked ancestor found
        }
        
        # Get Vrooli-related processes into a temporary file
        temp_file=$(mktemp)
        ps aux | grep -E "(vrooli|/scenarios/.*/(api|ui)|node_modules/.bin/vite|ecosystem-manager|picker-wheel)" | grep -v grep | grep -v "bash -c" > "$temp_file"
        
        while IFS= read -r line; do
            if [[ -z "$line" ]]; then continue; fi
            
            # Extract PID (second field)
            pid=$(echo "$line" | awk "{print \$2}")
            if [[ ! "$pid" =~ ^[0-9]+$ ]]; then continue; fi
            
            # Skip own processes
            if echo "$line" | grep -q "vrooli-api\|status-command\|zombie-detector"; then continue; fi
            
            # Check if this process or any ancestor is tracked
            if ! is_process_or_ancestor_tracked "$pid"; then
                # No tracked ancestor found, this is a true orphan
                ((orphan_count++))
            fi
        done < "$temp_file"
        
        rm -f "$temp_file"
        echo $orphan_count
    ' 2>/dev/null || echo "0"
}

################################################################################
# STALE LOCK DETECTION
################################################################################

#######################################
# Check for stale port locks
# Arguments:
#   $1 - Port number
#   $2 - Scenario name (optional, for context)
# Returns:
#   0 if lock is stale, 1 if lock is valid or no lock exists
#######################################
zombie::check_stale_port_lock() {
    local port="$1"
    local scenario_name="${2:-unknown}"
    local lock_file="$HOME/.vrooli/state/scenarios/.port_${port}.lock"
    
    # Check if lock file exists
    [[ -f "$lock_file" ]] || return 1
    
    # Parse lock file format: scenario_name:pid:timestamp
    local lock_info
    lock_info=$(cat "$lock_file" 2>/dev/null || echo "")
    [[ -n "$lock_info" ]] || return 1
    
    local lock_scenario="${lock_info%%:*}"
    local lock_pid_part="${lock_info#*:}"
    local lock_pid="${lock_pid_part%%:*}"
    
    # Check if PID is valid and running
    if [[ "$lock_pid" =~ ^[0-9]+$ ]] && kill -0 "$lock_pid" 2>/dev/null; then
        return 1  # Lock is valid, process is running
    fi
    
    # Lock is stale
    return 0
}

#######################################
# Clean stale port locks
# Arguments:
#   $1 - Port number
# Returns:
#   0 on success, 1 on failure
#######################################
zombie::clean_stale_port_lock() {
    local port="$1"
    local lock_file="$HOME/.vrooli/state/scenarios/.port_${port}.lock"
    
    if zombie::check_stale_port_lock "$port"; then
        log::info "Cleaning stale port lock for port $port"
        rm -f "$lock_file" 2>/dev/null
        return 0
    fi
    
    return 1  # Lock is not stale or doesn't exist
}

################################################################################
# DIAGNOSTIC FUNCTIONS
################################################################################

#######################################
# Comprehensive port diagnostic
# Provides detailed information about why a port is unavailable
# Arguments:
#   $1 - Port number
#   $2 - Scenario name
# Outputs:
#   Detailed diagnostic information to stderr
#######################################
zombie::diagnose_port_failure() {
    local port="$1"
    local scenario_name="$2"
    
    echo "" >&2
    echo "🚨 PORT BINDING FAILURE DIAGNOSTICS" >&2
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
    echo "Scenario: $scenario_name" >&2
    echo "Port: $port" >&2
    echo "" >&2
    
    # Check 1: Basic port availability
    echo "🔍 Checking port availability..." >&2
    local pids
    # Stricter timeout (2s) and flags to prevent lsof from hanging
    pids=$(timeout 2s lsof -tiTCP:"$port" -sTCP:LISTEN -P -n 2>/dev/null || echo "")
    
    if [[ -n "$pids" ]]; then
        echo "❌ Port $port is in use by process(es): $pids" >&2
        echo "   Process details:" >&2
        while IFS= read -r pid; do
            [[ -n "$pid" ]] || continue
            local process_info
            process_info=$(ps -p "$pid" -o pid,ppid,user,comm,args --no-headers 2>/dev/null || echo "$pid: <process not found>")
            echo "     PID $process_info" >&2
        done <<< "$pids"
    else
        echo "✅ No processes listening on port $port" >&2
    fi
    
    echo "" >&2
    
    # Check 2: Stale lock file detection
    echo "🔍 Checking for stale lock files..." >&2
    local lock_file="$HOME/.vrooli/state/scenarios/.port_${port}.lock"
    
    if [[ -f "$lock_file" ]]; then
        local lock_info
        lock_info=$(cat "$lock_file" 2>/dev/null || echo "")
        if [[ -n "$lock_info" ]]; then
            local lock_scenario="${lock_info%%:*}"
            local lock_pid_part="${lock_info#*:}"
            local lock_pid="${lock_pid_part%%:*}"
            local lock_timestamp="${lock_info##*:}"
            
            echo "⚠️  Found lock file: $lock_file" >&2
            echo "   Lock held by: scenario '$lock_scenario', PID $lock_pid" >&2
            echo "   Lock timestamp: $lock_timestamp" >&2
            
            if [[ "$lock_pid" =~ ^[0-9]+$ ]] && kill -0 "$lock_pid" 2>/dev/null; then
                echo "✅ Lock is VALID - process $lock_pid is running" >&2
            else
                echo "❌ Lock is STALE - process $lock_pid is not running (zombie/orphan lock)" >&2
                echo "   🛠️  Fix: rm '$lock_file'" >&2
            fi
        else
            echo "⚠️  Found empty lock file: $lock_file" >&2
            echo "   🛠️  Fix: rm '$lock_file'" >&2
        fi
    else
        echo "✅ No lock file found for port $port" >&2
    fi
    
    echo "" >&2
    
    # Check 3: Zombie process detection (if port has processes)
    if [[ -n "$pids" ]]; then
        echo "🔍 Checking for zombie processes on port $port..." >&2
        local zombie_found=false
        while IFS= read -r pid; do
            [[ -n "$pid" ]] || continue
            local state
            state=$(ps -o state= -p "$pid" 2>/dev/null | tr -d ' ')
            if [[ "$state" == "Z" ]]; then
                echo "💀 Found zombie process: PID $pid" >&2
                zombie_found=true
            fi
        done <<< "$pids"
        
        if [[ "$zombie_found" == "false" ]]; then
            echo "✅ No zombie processes found on port $port" >&2
        fi
    fi
    
    echo "" >&2
    
    # Check 4: System-wide orphan detection
    echo "🔍 Checking for system-wide Vrooli orphans..." >&2
    local orphan_count
    orphan_count=$(zombie::count_orphans)
    
    if [[ $orphan_count -gt 0 ]]; then
        echo "⚠️  Found $orphan_count orphaned Vrooli processes" >&2
        echo "   These may be interfering with port allocation" >&2
        echo "   🛠️  Fix: vrooli status --verbose (to identify), then kill orphaned processes" >&2
    else
        echo "✅ No orphaned Vrooli processes found" >&2
    fi
    
    echo "" >&2
    echo "💡 RECOMMENDED ACTIONS:" >&2
    
    # Provide specific recommendations
    local has_recommendations=false
    
    if zombie::check_stale_port_lock "$port"; then
        echo "   1. Clean stale lock: rm '$HOME/.vrooli/state/scenarios/.port_${port}.lock'" >&2
        has_recommendations=true
    fi
    
    if [[ -n "$pids" ]]; then
        echo "   2. Kill processes on port: kill $pids" >&2
        has_recommendations=true
    fi
    
    if [[ $orphan_count -gt 0 ]]; then
        echo "   3. Clean up orphans: vrooli status --verbose # then kill orphaned PIDs" >&2
        has_recommendations=true
    fi
    
    if [[ "$has_recommendations" == "false" ]]; then
        echo "   • Port appears to be available - there may be a different issue" >&2
        echo "   • Try: vrooli resource restart" >&2
        echo "   • Check logs: vrooli scenario logs $scenario_name" >&2
    fi
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" >&2
    echo "" >&2
}

################################################################################
# CONVENIENCE FUNCTIONS
################################################################################

#######################################
# Clean all stale port locks for a scenario
# Arguments:
#   $1 - Scenario name
# Returns:
#   Number of cleaned locks
#######################################
zombie::clean_scenario_locks() {
    local scenario_name="$1"
    local cleaned_count=0
    local lock_dir="$HOME/.vrooli/state/scenarios"
    
    [[ -d "$lock_dir" ]] || return 0
    
    for lock_file in "$lock_dir"/.port_*.lock; do
        [[ -f "$lock_file" ]] || continue
        
        local port="${lock_file##*/}"  # Get filename
        port="${port%.lock}"           # Remove .lock
        port="${port#.port_}"          # Remove .port_ prefix
        
        # Check if this lock is for our scenario and is stale
        local lock_info
        lock_info=$(cat "$lock_file" 2>/dev/null || echo "")
        local lock_scenario="${lock_info%%:*}"
        
        if [[ "$lock_scenario" == "$scenario_name" ]] && zombie::check_stale_port_lock "$port"; then
            rm -f "$lock_file"
            ((cleaned_count++))
            log::info "Cleaned stale lock for scenario '$scenario_name' port $port"
        fi
    done
    
    echo $cleaned_count
}

################################################################################
# CLI INTERFACE
################################################################################

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-help}" in
        count-zombies)
            zombie::count_zombies
            ;;
        count-orphans)
            zombie::count_orphans
            ;;
        check-port-lock)
            zombie::check_stale_port_lock "${2:-}" "${3:-}"
            ;;
        clean-port-lock)
            zombie::clean_stale_port_lock "${2:-}"
            ;;
        diagnose-port)
            zombie::diagnose_port_failure "${2:-}" "${3:-unknown}"
            ;;
        clean-scenario-locks)
            zombie::clean_scenario_locks "${2:-}"
            ;;
        help|*)
            cat << 'EOF'
Zombie/Orphan Process Detector - Reusable utilities for process management

USAGE:
    zombie-detector.sh <command> [args...]

COMMANDS:
    count-zombies              Count all zombie (defunct) processes
    count-orphans              Count orphaned Vrooli processes
    check-port-lock <port>     Check if port lock is stale (exit 0 if stale)
    clean-port-lock <port>     Clean stale lock for specific port
    diagnose-port <port> <scenario>  Show comprehensive port failure diagnostics
    clean-scenario-locks <scenario>  Clean all stale locks for a scenario

EXAMPLES:
    # Check for orphans
    ./zombie-detector.sh count-orphans

    # Diagnose why port 21600 failed for app-monitor
    ./zombie-detector.sh diagnose-port 21600 app-monitor
    
    # Clean stale locks for app-monitor scenario
    ./zombie-detector.sh clean-scenario-locks app-monitor

EOF
            ;;
    esac
fi