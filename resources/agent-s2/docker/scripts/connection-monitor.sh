#!/bin/bash
# Connection monitoring and cleanup for Agent-S2
# This script monitors TCP connections and takes action if they exceed limits

MAX_CONNECTIONS=500
CRITICAL_CONNECTIONS=1000
CHECK_INTERVAL=60
LOG_FILE="/tmp/connection-monitor.log"

# Function to log with timestamp
log_message() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Function to get connection count
get_connection_count() {
    ss -tn state established 2>/dev/null | wc -l
}

# Function to cleanup connections
cleanup_connections() {
    log_message "Attempting connection cleanup..."
    
    # Try to clear TIME_WAIT connections (requires root)
    if sudo -n ss -K state time-wait 2>/dev/null; then
        log_message "Cleaned TIME_WAIT connections"
    fi
    
    # Kill old FIN_WAIT connections
    if sudo -n ss -K state fin-wait-1 2>/dev/null; then
        log_message "Cleaned FIN_WAIT1 connections"
    fi
    
    if sudo -n ss -K state fin-wait-2 2>/dev/null; then
        log_message "Cleaned FIN_WAIT2 connections"
    fi
}

# Main monitoring loop
log_message "Starting Agent-S2 connection monitor"
log_message "Max connections: $MAX_CONNECTIONS"
log_message "Critical threshold: $CRITICAL_CONNECTIONS"
log_message "Check interval: ${CHECK_INTERVAL}s"

while true; do
    CONN_COUNT=$(get_connection_count)
    
    # Log current status
    if [ "$CONN_COUNT" -lt "$MAX_CONNECTIONS" ]; then
        log_message "OK: $CONN_COUNT connections (below $MAX_CONNECTIONS limit)"
    elif [ "$CONN_COUNT" -lt "$CRITICAL_CONNECTIONS" ]; then
        log_message "WARNING: $CONN_COUNT connections (exceeds $MAX_CONNECTIONS limit)"
        cleanup_connections
    else
        log_message "CRITICAL: $CONN_COUNT connections (exceeds $CRITICAL_CONNECTIONS critical limit)"
        cleanup_connections
        
        # If still critical after cleanup, restart the proxy service
        CONN_COUNT_AFTER=$(get_connection_count)
        if [ "$CONN_COUNT_AFTER" -gt "$CRITICAL_CONNECTIONS" ]; then
            log_message "CRITICAL: Restarting security-proxy due to connection leak"
            supervisorctl restart security-proxy 2>/dev/null || log_message "Failed to restart proxy (may need to restart container)"
        fi
    fi
    
    sleep "$CHECK_INTERVAL"
done