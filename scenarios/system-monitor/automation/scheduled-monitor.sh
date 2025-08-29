#!/bin/bash
# System Monitor - Scheduled Automation Script
# This script demonstrates automated monitoring and reporting integration

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
LOG_FILE="/tmp/system-monitor-automation.log"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Check system health
check_health() {
    log "Checking system health..."
    RESPONSE=$(curl -s "$API_BASE_URL/health")
    if echo "$RESPONSE" | grep -q "healthy"; then
        log "✅ System is healthy"
        return 0
    else
        log "❌ System health check failed: $RESPONSE"
        return 1
    fi
}

# Get current metrics
get_metrics() {
    log "Fetching current metrics..."
    METRICS=$(curl -s "$API_BASE_URL/api/metrics/current")
    if [ $? -eq 0 ]; then
        CPU=$(echo "$METRICS" | grep -o '"cpu_usage":[0-9.]*' | cut -d':' -f2)
        MEMORY=$(echo "$METRICS" | grep -o '"memory_usage":[0-9.]*' | cut -d':' -f2)
        TCP=$(echo "$METRICS" | grep -o '"tcp_connections":[0-9]*' | cut -d':' -f2)
        
        log "📊 Current metrics: CPU=${CPU}%, Memory=${MEMORY}%, TCP=${TCP}"
        
        # Check thresholds
        if (( $(echo "$CPU > 80" | bc -l) )); then
            log "⚠️  HIGH CPU ALERT: ${CPU}%"
            trigger_investigation "High CPU usage detected"
        fi
        
        if (( $(echo "$MEMORY > 85" | bc -l) )); then
            log "⚠️  HIGH MEMORY ALERT: ${MEMORY}%"
            trigger_investigation "High memory usage detected"
        fi
        
        return 0
    else
        log "❌ Failed to fetch metrics"
        return 1
    fi
}

# Trigger investigation
trigger_investigation() {
    local reason="$1"
    log "🔍 Triggering investigation: $reason"
    
    RESULT=$(curl -s -X POST "$API_BASE_URL/api/investigations/trigger" -H "Content-Type: application/json")
    INVESTIGATION_ID=$(echo "$RESULT" | grep -o '"investigation_id":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$INVESTIGATION_ID" ]; then
        log "✅ Investigation triggered: $INVESTIGATION_ID"
        monitor_investigation "$INVESTIGATION_ID"
    else
        log "❌ Failed to trigger investigation: $RESULT"
    fi
}

# Monitor investigation progress
monitor_investigation() {
    local inv_id="$1"
    log "📈 Monitoring investigation progress: $inv_id"
    
    for i in {1..30}; do
        sleep 2
        STATUS=$(curl -s "$API_BASE_URL/api/investigations/latest" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
        
        case "$STATUS" in
            "completed")
                log "✅ Investigation completed successfully"
                return 0
                ;;
            "failed")
                log "❌ Investigation failed"
                return 1
                ;;
            "in_progress")
                log "⏳ Investigation in progress..."
                ;;
        esac
    done
    
    log "⌛ Investigation monitoring timeout"
    return 1
}

# Generate daily report
generate_report() {
    log "📄 Generating daily report..."
    RESULT=$(curl -s -X POST "$API_BASE_URL/api/reports/generate" -H "Content-Type: application/json" -d '{"type":"daily"}')
    REPORT_ID=$(echo "$RESULT" | grep -o '"report_id":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$REPORT_ID" ]; then
        log "✅ Daily report generated: $REPORT_ID"
        return 0
    else
        log "❌ Failed to generate report: $RESULT"
        return 1
    fi
}

# Main automation routine
main() {
    log "🚀 Starting scheduled monitoring automation..."
    
    # Health check
    if ! check_health; then
        log "❌ System unhealthy, skipping monitoring"
        exit 1
    fi
    
    # Get metrics and check thresholds
    get_metrics
    
    # Generate report (if it's the top of the hour)
    HOUR=$(date '+%M')
    if [ "$HOUR" = "00" ]; then
        generate_report
    fi
    
    log "✅ Scheduled monitoring automation completed"
}

# Command-line interface
case "${1:-main}" in
    "health")
        check_health
        ;;
    "metrics")
        get_metrics
        ;;
    "investigate")
        trigger_investigation "${2:-Manual trigger}"
        ;;
    "report")
        generate_report
        ;;
    "main"|"")
        main
        ;;
    *)
        echo "Usage: $0 {main|health|metrics|investigate|report}"
        exit 1
        ;;
esac