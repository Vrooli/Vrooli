#!/bin/bash

# Background health monitor for Core Debugger
# Runs periodic health checks and detects issues

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SCENARIO_ROOT="$(dirname "$SCRIPT_DIR")"
DATA_DIR="$SCENARIO_ROOT/data"
CLI="$SCENARIO_ROOT/cli/core-debugger"

# Check interval (seconds)
CHECK_INTERVAL=${HEALTH_CHECK_INTERVAL:-60}

# Log file
LOG_FILE="$DATA_DIR/logs/health-monitor.log"
mkdir -p "$(dirname "$LOG_FILE")"

log() {
    echo "[$(date -Iseconds)] $1" >> "$LOG_FILE"
}

# Main monitoring loop
log "Health monitor started (interval: ${CHECK_INTERVAL}s)"

while true; do
    # Run health check
    if ! "$CLI" check-health --json > "$DATA_DIR/health/current.json" 2>/dev/null; then
        log "ERROR: Health check failed"
        
        # Try to detect specific issues
        if ! vrooli help > /dev/null 2>&1; then
            "$CLI" report-issue --component cli --description "CLI not responding to help command" --severity high
            log "Reported CLI issue"
        fi
        
        if ! vrooli status > /dev/null 2>&1; then
            "$CLI" report-issue --component orchestrator --description "Orchestrator status check failed" --severity high
            log "Reported orchestrator issue"
        fi
    else
        log "Health check completed successfully"
    fi
    
    # Check for pattern-based issues in logs
    if [ -f "$LOG_FILE" ]; then
        # Check for repeated errors
        local error_count=$(tail -n 100 "$LOG_FILE" | grep -c "ERROR" || true)
        if [ "$error_count" -gt 10 ]; then
            "$CLI" report-issue --component "health-monitor" --description "High error rate detected: $error_count errors in last 100 log lines" --severity medium
            log "High error rate reported"
        fi
    fi
    
    # Sleep before next check
    sleep "$CHECK_INTERVAL"
done