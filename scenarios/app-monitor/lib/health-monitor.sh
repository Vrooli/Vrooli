#!/usr/bin/env bash
################################################################################
# App Monitor Health Monitor
# 
# Monitors app-monitor for issues and auto-recovers when needed
# Prevents CPU spikes from runaway processes
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true

################################################################################
# CONFIGURATION
################################################################################

# Health check thresholds
MAX_CPU_PER_PROCESS=50      # Max CPU % for a single process
MAX_TOTAL_CPU=70            # Max total CPU % for all app-monitor processes
MAX_SUBPROCESS_COUNT=20     # Max number of subprocesses
HEALTH_CHECK_INTERVAL=30    # Seconds between health checks
RECOVERY_COOLDOWN=60        # Seconds to wait after recovery action

################################################################################
# HEALTH CHECK FUNCTIONS
################################################################################

#######################################
# Check if app-monitor is healthy
# Returns:
#   0 if healthy, 1 if unhealthy
#######################################
health::check_app_monitor() {
    local unhealthy=false
    local issues=()
    
    # Find app-monitor main process
    local main_pid
    main_pid=$(pgrep -f "app-monitor-api" | head -1 2>/dev/null || echo "")
    
    if [[ -z "$main_pid" ]]; then
        log::info "App-monitor not running"
        return 0  # Not running is not unhealthy
    fi
    
    # Check CPU usage of main process
    local cpu_usage
    cpu_usage=$(ps -p "$main_pid" -o pcpu= 2>/dev/null | tr -d ' ' | sed 's/^$/0/' || echo "0")
    
    # Ensure cpu_usage is a valid number
    if [[ -z "$cpu_usage" ]] || [[ "$cpu_usage" == "" ]]; then
        cpu_usage="0"
    fi
    
    if (( $(echo "$cpu_usage > $MAX_CPU_PER_PROCESS" | bc -l 2>/dev/null || echo "0") )); then
        issues+=("Main process CPU too high: ${cpu_usage}%")
        unhealthy=true
    fi
    
    # Count subprocesses
    local subprocess_count
    subprocess_count=$(pstree -p "$main_pid" 2>/dev/null | grep -o '([0-9]*)' | wc -l || echo "0")
    
    if [[ $subprocess_count -gt $MAX_SUBPROCESS_COUNT ]]; then
        issues+=("Too many subprocesses: $subprocess_count")
        unhealthy=true
    fi
    
    # Check for stuck lsof processes spawned by app-monitor
    local stuck_lsof
    stuck_lsof=$(pgrep -P "$main_pid" -f "lsof" 2>/dev/null | wc -l || echo "0")
    
    # Ensure stuck_lsof is a valid number
    stuck_lsof=$(echo "$stuck_lsof" | tr -d ' ')
    if [[ -z "$stuck_lsof" ]] || [[ "$stuck_lsof" == "" ]]; then
        stuck_lsof="0"
    fi
    
    if [[ "$stuck_lsof" -gt "0" ]]; then
        issues+=("Found $stuck_lsof stuck lsof processes")
        unhealthy=true
    fi
    
    # Check total CPU usage of all app-monitor related processes
    local total_cpu
    total_cpu=$(ps aux | grep -E "(app-monitor|scenario.*status|resource.*status)" | grep -v grep | \
                awk '{sum+=$3} END {if(NR==0) print "0"; else print sum}')
    
    # Ensure total_cpu is a valid number
    if [[ -z "$total_cpu" ]] || [[ "$total_cpu" == "" ]]; then
        total_cpu="0"
    fi
    
    if (( $(echo "$total_cpu > $MAX_TOTAL_CPU" | bc -l 2>/dev/null || echo "0") )); then
        issues+=("Total CPU usage too high: ${total_cpu}%")
        unhealthy=true
    fi
    
    # Report issues
    if [[ "$unhealthy" == "true" ]]; then
        log::warn "App-monitor health issues detected:"
        for issue in "${issues[@]}"; do
            log::warn "  - $issue"
        done
        return 1
    fi
    
    return 0
}

#######################################
# Kill stuck processes spawned by app-monitor
#######################################
health::kill_stuck_processes() {
    local killed_count=0
    
    # Find and kill stuck lsof processes
    local lsof_pids
    lsof_pids=$(pgrep -f "lsof.*-iTCP" 2>/dev/null || true)
    
    for pid in $lsof_pids; do
        [[ -z "$pid" ]] && continue
        
        # Check if process has been running for more than 3 seconds
        local etime
        etime=$(ps -p "$pid" -o etimes= 2>/dev/null | tr -d ' ' || echo "0")
        
        if [[ $etime -gt 3 ]]; then
            log::info "Killing stuck lsof process: PID $pid (runtime: ${etime}s)"
            kill -9 "$pid" 2>/dev/null || true
            ((killed_count++))
        fi
    done
    
    # Kill any timeout commands that have been running too long
    local timeout_pids
    timeout_pids=$(pgrep -f "timeout.*vrooli" 2>/dev/null || true)
    
    for pid in $timeout_pids; do
        [[ -z "$pid" ]] && continue
        
        local etime
        etime=$(ps -p "$pid" -o etimes= 2>/dev/null | tr -d ' ' || echo "0")
        
        if [[ $etime -gt 10 ]]; then
            log::info "Killing stuck timeout process: PID $pid (runtime: ${etime}s)"
            kill -9 "$pid" 2>/dev/null || true
            ((killed_count++))
        fi
    done
    
    echo "$killed_count"
}

#######################################
# Restart app-monitor gracefully
#######################################
health::restart_app_monitor() {
    log::info "Restarting app-monitor..."
    
    # Stop app-monitor
    "${APP_ROOT}/cli/vrooli" scenario stop app-monitor 2>/dev/null || true
    
    # Wait for processes to clean up
    sleep 2
    
    # Kill any remaining processes
    pkill -f "app-monitor" 2>/dev/null || true
    
    # Wait a bit more
    sleep 3
    
    # Start app-monitor
    "${APP_ROOT}/cli/vrooli" scenario run app-monitor
    
    log::success "App-monitor restarted"
}

################################################################################
# MAIN MONITORING LOOP
################################################################################

#######################################
# Run continuous health monitoring
#######################################
health::monitor_loop() {
    local last_recovery=0
    
    log::info "Starting app-monitor health monitor"
    log::info "Check interval: ${HEALTH_CHECK_INTERVAL}s"
    log::info "Recovery cooldown: ${RECOVERY_COOLDOWN}s"
    
    while true; do
        # Check health
        if ! health::check_app_monitor; then
            local current_time
            current_time=$(date +%s)
            
            # Check if we're in cooldown period
            local time_since_recovery=$((current_time - last_recovery))
            if [[ $time_since_recovery -lt $RECOVERY_COOLDOWN ]]; then
                log::info "In recovery cooldown (${time_since_recovery}s/${RECOVERY_COOLDOWN}s)"
            else
                # Try to recover
                log::warn "Attempting recovery..."
                
                # First, try killing stuck processes
                local killed
                killed=$(health::kill_stuck_processes)
                
                if [[ $killed -gt 0 ]]; then
                    log::info "Killed $killed stuck processes"
                    last_recovery=$current_time
                    
                    # Give it time to recover
                    sleep 5
                    
                    # Check health again
                    if ! health::check_app_monitor; then
                        log::warn "Still unhealthy after killing processes, restarting..."
                        health::restart_app_monitor
                    fi
                else
                    # No stuck processes found, restart the service
                    log::warn "No stuck processes found, restarting service..."
                    health::restart_app_monitor
                    last_recovery=$current_time
                fi
            fi
        else
            log::debug "App-monitor health check passed"
        fi
        
        # Wait before next check
        sleep "$HEALTH_CHECK_INTERVAL"
    done
}

################################################################################
# CLI INTERFACE
################################################################################

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-monitor}" in
        monitor)
            health::monitor_loop
            ;;
        check)
            if health::check_app_monitor; then
                log::success "App-monitor is healthy"
                exit 0
            else
                log::error "App-monitor is unhealthy"
                exit 1
            fi
            ;;
        kill-stuck)
            killed=$(health::kill_stuck_processes)
            log::info "Killed $killed stuck processes"
            ;;
        restart)
            health::restart_app_monitor
            ;;
        help|*)
            cat << 'EOF'
App Monitor Health Monitor - Prevents CPU spikes and auto-recovers from issues

USAGE:
    health-monitor.sh <command>

COMMANDS:
    monitor      Run continuous health monitoring (default)
    check        Check health once and exit
    kill-stuck   Kill stuck processes
    restart      Restart app-monitor

EXAMPLES:
    # Run continuous monitoring
    ./health-monitor.sh monitor
    
    # Check health once
    ./health-monitor.sh check
    
    # Kill stuck processes
    ./health-monitor.sh kill-stuck

EOF
            ;;
    esac
fi