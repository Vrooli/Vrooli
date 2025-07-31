#!/bin/bash
# Firefox monitoring and recovery script for Agent S2

# Configuration
FIREFOX_CMD="firefox-esr --profile /home/agents2/.mozilla/firefox/agent-s2 --new-window about:blank"
LOG_FILE="/var/log/supervisor/firefox.log"
CRASH_LOG="/var/log/firefox-crashes.log"
MAX_RESTARTS=5
RESTART_DELAY=5
MEMORY_LIMIT_MB=1500

# Initialize
RESTART_COUNT=0
LAST_RESTART_TIME=0

log_message() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$CRASH_LOG"
}

check_memory_usage() {
    local pid=$1
    if [ -z "$pid" ]; then
        return 1
    fi
    
    # Get memory usage in MB
    local mem_kb=$(ps -p "$pid" -o rss= 2>/dev/null | tr -d ' ')
    if [ -z "$mem_kb" ]; then
        return 1
    fi
    
    local mem_mb=$((mem_kb / 1024))
    
    if [ $mem_mb -gt $MEMORY_LIMIT_MB ]; then
        log_message "WARNING: Firefox memory usage high: ${mem_mb}MB > ${MEMORY_LIMIT_MB}MB limit"
        return 2
    fi
    
    return 0
}

validate_profile() {
    local profile_dir="/home/agents2/.mozilla/firefox/agent-s2"
    local profiles_ini="/home/agents2/.mozilla/firefox/profiles.ini"
    local user_js="$profile_dir/user.js"
    
    log_message "Validating Firefox profile integrity..."
    
    # Check if profile directory exists
    if [ ! -d "$profile_dir" ]; then
        log_message "Profile directory missing, will be recreated by startup script"
        return 1
    fi
    
    # Check if profiles.ini exists
    if [ ! -f "$profiles_ini" ]; then
        log_message "profiles.ini missing, will be recreated by startup script"
        return 1
    fi
    
    # Check if user.js exists and is readable
    if [ ! -f "$user_js" ] || [ ! -r "$user_js" ]; then
        log_message "user.js missing or unreadable, will be recreated by startup script"
        return 1
    fi
    
    log_message "Firefox profile validation passed"
    return 0
}

graceful_firefox_shutdown() {
    local firefox_pid=$(pgrep -f "firefox-esr" | head -1)
    if [ -n "$firefox_pid" ]; then
        log_message "Gracefully shutting down Firefox (PID: $firefox_pid)"
        # Send TERM signal for graceful shutdown
        kill -TERM "$firefox_pid" 2>/dev/null
        
        # Wait up to 5 seconds for graceful shutdown
        local wait_count=0
        while [ $wait_count -lt 5 ] && ps -p "$firefox_pid" > /dev/null 2>&1; do
            sleep 1
            wait_count=$((wait_count + 1))
        done
        
        # Force kill if still running
        if ps -p "$firefox_pid" > /dev/null 2>&1; then
            log_message "Firefox not responding to TERM, force killing..."
            kill -KILL "$firefox_pid" 2>/dev/null
            sleep 1
        else
            log_message "Firefox shut down gracefully"
        fi
    fi
}

start_firefox() {
    log_message "Starting Firefox..."
    
    # Validate Firefox profile before starting
    if ! validate_profile; then
        log_message "Profile validation failed, running startup script to recreate profile..."
        /opt/agent-s2/startup.sh
        if ! validate_profile; then
            log_message "ERROR: Profile recreation failed, Firefox may show profile dialog"
        fi
    fi
    
    # Clean up any existing Firefox processes gracefully
    graceful_firefox_shutdown
    
    # Clean up Firefox profile lock files to prevent profile dialog
    local profile_dir="/home/agents2/.mozilla/firefox/agent-s2"
    rm -f "$profile_dir/.parentlock" 2>/dev/null
    rm -f "$profile_dir/lock" 2>/dev/null
    rm -f "$profile_dir/.lock" 2>/dev/null
    log_message "Cleaned up Firefox profile lock files"
    
    # Clear Firefox cache to prevent memory bloat
    rm -rf /home/agents2/.cache/mozilla/firefox/*/cache2/* 2>/dev/null
    
    # Set memory-conscious environment variables
    export MOZ_DISABLE_CONTENT_SANDBOX=1
    export MOZ_DISABLE_GMP_SANDBOX=1
    export MOZ_DISABLE_NPAPI_SANDBOX=1
    export MOZ_DISABLE_GPU_SANDBOX=1
    
    # Start Firefox with memory limits
    DISPLAY=:99 $FIREFOX_CMD >> "$LOG_FILE" 2>&1 &
    local pid=$!
    
    # Wait for Firefox to stabilize
    sleep 5
    
    # Check if process started successfully
    if ps -p $pid > /dev/null 2>&1; then
        log_message "Firefox started successfully (PID: $pid)"
        echo $pid
        return 0
    else
        log_message "ERROR: Firefox failed to start"
        return 1
    fi
}

monitor_firefox() {
    local firefox_pid=""
    
    while true; do
        # Check if Firefox is running
        firefox_pid=$(pgrep -f "firefox-esr" | head -1)
        
        if [ -z "$firefox_pid" ]; then
            log_message "Firefox not running, attempting restart..."
            
            # Check restart limits
            current_time=$(date +%s)
            time_since_last_restart=$((current_time - LAST_RESTART_TIME))
            
            # Reset counter if enough time has passed
            if [ $time_since_last_restart -gt 300 ]; then  # 5 minutes
                RESTART_COUNT=0
            fi
            
            if [ $RESTART_COUNT -ge $MAX_RESTARTS ]; then
                log_message "ERROR: Maximum restart attempts reached ($MAX_RESTARTS)"
                sleep 60  # Wait longer before trying again
                RESTART_COUNT=0  # Reset counter
            else
                firefox_pid=$(start_firefox)
                if [ $? -eq 0 ]; then
                    RESTART_COUNT=$((RESTART_COUNT + 1))
                    LAST_RESTART_TIME=$current_time
                else
                    log_message "ERROR: Failed to restart Firefox"
                fi
            fi
        else
            # Firefox is running, check its health
            check_memory_usage "$firefox_pid"
            local mem_status=$?
            
            if [ $mem_status -eq 2 ]; then
                # Memory usage too high, restart Firefox
                log_message "Restarting Firefox due to high memory usage"
                graceful_firefox_shutdown
                firefox_pid=$(start_firefox)
            fi
        fi
        
        # Check for segfaults in system logs
        if dmesg 2>/dev/null | tail -10 | grep -q "firefox.*segfault"; then
            log_message "WARNING: Firefox segfault detected in system logs"
            # Clear dmesg if possible (requires privileges)
            dmesg -c 2>/dev/null || true
        fi
        
        sleep 10  # Check every 10 seconds
    done
}

# Main execution
log_message "Firefox monitor started"

# Initial Firefox start
firefox_pid=$(start_firefox)

# Start monitoring loop
monitor_firefox