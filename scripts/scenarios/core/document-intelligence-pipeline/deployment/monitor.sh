#!/bin/bash
# Monitoring script for Document Intelligence Pipeline
# This script provides continuous health monitoring and alerting

set -euo pipefail

# Configuration
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCENARIO_ID="document-intelligence-pipeline"
SCENARIO_NAME="Document Intelligence Pipeline"
MONITOR_LOG="/tmp/vrooli-${SCENARIO_ID}-monitor.log"
PID_FILE="/tmp/vrooli-${SCENARIO_ID}-monitor.pid"

# Monitoring intervals (seconds)
HEALTH_CHECK_INTERVAL=30
PERFORMANCE_CHECK_INTERVAL=60
RESOURCE_CHECK_INTERVAL=120

# Alert thresholds
RESPONSE_TIME_THRESHOLD_MS=5000
ERROR_RATE_THRESHOLD_PERCENT=5
RESOURCE_USAGE_THRESHOLD_PERCENT=80

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Monitoring state
MONITORING_ACTIVE=true
CONSECUTIVE_FAILURES=0
MAX_CONSECUTIVE_FAILURES=3

# Logging functions
log_monitor() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [MONITOR] $1" | tee -a "$MONITOR_LOG"
}

log_health() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [HEALTH] $1" | tee -a "$MONITOR_LOG"
}

log_alert() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] ${RED}[ALERT]${NC} $1" | tee -a "$MONITOR_LOG"
}

log_performance() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [PERF] $1" | tee -a "$MONITOR_LOG"
}

log_recovery() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] ${GREEN}[RECOVERY]${NC} $1" | tee -a "$MONITOR_LOG"
}

# Signal handlers
cleanup_and_exit() {
    log_monitor "Monitoring stopped for $SCENARIO_NAME"
    rm -f "$PID_FILE"
    exit 0
}

trap cleanup_and_exit SIGTERM SIGINT

# Helper functions
get_required_resources() {
    if [[ -f "$SCENARIO_DIR/metadata.yaml" ]]; then
        grep -A 10 "required:" "$SCENARIO_DIR/metadata.yaml" | grep "^[[:space:]]*-" | sed 's/^[[:space:]]*-[[:space:]]*//' | tr '\n' ' '
    fi
}

check_service_health() {
    local service_name="$1"
    local health_url="$2"
    local timeout="${3:-5}"
    
    local start_time=$(date +%s%3N)
    local response_code
    
    if response_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time "$timeout" "$health_url" 2>/dev/null); then
        local end_time=$(date +%s%3N)
        local response_time=$((end_time - start_time))
        
        if [[ "$response_code" -ge 200 && "$response_code" -lt 400 ]]; then
            log_health "✓ $service_name healthy (${response_time}ms, HTTP $response_code)"
            return 0
        else
            log_alert "✗ $service_name unhealthy (HTTP $response_code)"
            return 1
        fi
    else
        log_alert "✗ $service_name unreachable"
        return 1
    fi
}

check_database_health() {
    local db_name="${SCENARIO_ID//-/_}"
    
    if command -v pg_isready >/dev/null 2>&1; then
        if pg_isready -h localhost -p 5433 -d "$db_name" >/dev/null 2>&1; then
            # Test actual query performance
            local start_time=$(date +%s%3N)
            if psql -h localhost -p 5433 -U postgres -d "$db_name" -c "SELECT 1;" >/dev/null 2>&1; then
                local end_time=$(date +%s%3N)
                local query_time=$((end_time - start_time))
                log_health "✓ Database healthy (${query_time}ms)"
                return 0
            else
                log_alert "✗ Database query failed"
                return 1
            fi
        else
            log_alert "✗ Database connection failed"
            return 1
        fi
    else
        log_health "⚠ pg_isready not available, skipping database health check"
        return 0
    fi
}

test_webhook_performance() {
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" ]]; then
        local webhook_url="http://localhost:5678/webhook/${SCENARIO_ID}-webhook"
        local test_payload='{"test": true, "monitoring": true, "timestamp": "'$(date -Iseconds)'"}'
        
        local start_time=$(date +%s%3N)
        local response_code
        
        if response_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 \
            -X POST -H "Content-Type: application/json" -d "$test_payload" "$webhook_url" 2>/dev/null); then
            
            local end_time=$(date +%s%3N)
            local response_time=$((end_time - start_time))
            
            if [[ "$response_code" -ge 200 && "$response_code" -lt 400 ]]; then
                log_performance "Webhook response: ${response_time}ms (HTTP $response_code)"
                
                if [[ $response_time -gt $RESPONSE_TIME_THRESHOLD_MS ]]; then
                    log_alert "Webhook response time exceeded threshold: ${response_time}ms > ${RESPONSE_TIME_THRESHOLD_MS}ms"
                fi
                
                return 0
            else
                log_alert "Webhook test failed (HTTP $response_code)"
                return 1
            fi
        else
            log_alert "Webhook test failed (connection error)"
            return 1
        fi
    fi
}

check_resource_usage() {
    # CPU usage
    if command -v top >/dev/null 2>&1; then
        local cpu_usage
        cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | awk -F'%' '{print $1}' | tr -d ' ')
        if [[ -n "$cpu_usage" ]] && (( $(echo "$cpu_usage > $RESOURCE_USAGE_THRESHOLD_PERCENT" | bc -l 2>/dev/null || echo 0) )); then
            log_alert "High CPU usage: ${cpu_usage}%"
        else
            log_performance "CPU usage: ${cpu_usage}%"
        fi
    fi
    
    # Memory usage
    if command -v free >/dev/null 2>&1; then
        local mem_usage
        mem_usage=$(free | grep Mem | awk '{printf("%.1f", $3/$2 * 100.0)}')
        if [[ -n "$mem_usage" ]] && (( $(echo "$mem_usage > $RESOURCE_USAGE_THRESHOLD_PERCENT" | bc -l 2>/dev/null || echo 0) )); then
            log_alert "High memory usage: ${mem_usage}%"
        else
            log_performance "Memory usage: ${mem_usage}%"
        fi
    fi
    
    # Disk usage
    local disk_usage
    disk_usage=$(df "$SCENARIO_DIR" | awk 'NR==2 {print $5}' | tr -d '%')
    if [[ -n "$disk_usage" ]] && [[ $disk_usage -gt $RESOURCE_USAGE_THRESHOLD_PERCENT ]]; then
        log_alert "High disk usage: ${disk_usage}%"
    else
        log_performance "Disk usage: ${disk_usage}%"
    fi
}

collect_metrics() {
    # If QuestDB is available, store metrics
    if [[ "$REQUIRED_RESOURCES" =~ "questdb" ]] && command -v curl >/dev/null 2>&1; then
        local timestamp=$(date +%s%9N)
        local metrics_data="scenario_metrics,scenario_id=${SCENARIO_ID} "
        
        # Add health status
        metrics_data+="health_checks_passed=${HEALTH_CHECKS_PASSED:-0},"
        metrics_data+="health_checks_failed=${HEALTH_CHECKS_FAILED:-0},"
        metrics_data+="consecutive_failures=${CONSECUTIVE_FAILURES} "
        metrics_data+="${timestamp}"
        
        # Send to QuestDB via InfluxDB line protocol
        if echo "$metrics_data" | nc -w 1 localhost 9011 >/dev/null 2>&1; then
            log_performance "Metrics stored to QuestDB"
        else
            log_health "⚠ Could not store metrics to QuestDB"
        fi
    fi
}

# Main monitoring functions
health_monitoring_loop() {
    log_monitor "Starting health monitoring loop (interval: ${HEALTH_CHECK_INTERVAL}s)"
    
    while $MONITORING_ACTIVE; do
        local health_checks_passed=0
        local health_checks_failed=0
        
        log_health "=== Health Check Round ==="
        
        # Check required resources
        for resource in $REQUIRED_RESOURCES; do
            case "$resource" in
                "ollama")
                    if check_service_health "Ollama" "http://localhost:11434/api/tags"; then
                        ((health_checks_passed++))
                    else
                        ((health_checks_failed++))
                    fi
                    ;;
                "n8n")
                    if check_service_health "n8n" "http://localhost:5678/healthz"; then
                        ((health_checks_passed++))
                    else
                        ((health_checks_failed++))
                    fi
                    ;;
                "postgres")
                    if check_database_health; then
                        ((health_checks_passed++))
                    else
                        ((health_checks_failed++))
                    fi
                    ;;
                "windmill")
                    if check_service_health "Windmill" "http://localhost:5681/api/version"; then
                        ((health_checks_passed++))
                    else
                        ((health_checks_failed++))
                    fi
                    ;;
                "whisper")
                    if check_service_health "Whisper" "http://localhost:8090/"; then
                        ((health_checks_passed++))
                    else
                        ((health_checks_failed++))
                    fi
                    ;;
                "comfyui")
                    if check_service_health "ComfyUI" "http://localhost:8188/"; then
                        ((health_checks_passed++))
                    else
                        ((health_checks_failed++))
                    fi
                    ;;
                "minio")
                    if check_service_health "MinIO" "http://localhost:9000/minio/health/live"; then
                        ((health_checks_passed++))
                    else
                        ((health_checks_failed++))
                    fi
                    ;;
                "qdrant")
                    if check_service_health "Qdrant" "http://localhost:6333/"; then
                        ((health_checks_passed++))
                    else
                        ((health_checks_failed++))
                    fi
                    ;;
                "questdb")
                    if check_service_health "QuestDB" "http://localhost:9010/"; then
                        ((health_checks_passed++))
                    else
                        ((health_checks_failed++))
                    fi
                    ;;
            esac
        done
        
        # Update failure counter
        if [[ $health_checks_failed -gt 0 ]]; then
            ((CONSECUTIVE_FAILURES++))
            log_alert "Health check round failed: $health_checks_failed failures"
            
            if [[ $CONSECUTIVE_FAILURES -ge $MAX_CONSECUTIVE_FAILURES ]]; then
                log_alert "CRITICAL: $CONSECUTIVE_FAILURES consecutive failures detected!"
                # In a production environment, you might want to trigger recovery actions here
            fi
        else
            if [[ $CONSECUTIVE_FAILURES -gt 0 ]]; then
                log_recovery "System recovered after $CONSECUTIVE_FAILURES failures"
            fi
            CONSECUTIVE_FAILURES=0
            log_health "All health checks passed ($health_checks_passed/$((health_checks_passed + health_checks_failed)))"
        fi
        
        # Export for metrics collection
        export HEALTH_CHECKS_PASSED=$health_checks_passed
        export HEALTH_CHECKS_FAILED=$health_checks_failed
        
        sleep $HEALTH_CHECK_INTERVAL
    done
}

performance_monitoring_loop() {
    log_monitor "Starting performance monitoring loop (interval: ${PERFORMANCE_CHECK_INTERVAL}s)"
    
    while $MONITORING_ACTIVE; do
        log_performance "=== Performance Check Round ==="
        
        # Test webhook performance
        test_webhook_performance
        
        # Collect and store metrics
        collect_metrics
        
        sleep $PERFORMANCE_CHECK_INTERVAL
    done
}

resource_monitoring_loop() {
    log_monitor "Starting resource monitoring loop (interval: ${RESOURCE_CHECK_INTERVAL}s)"
    
    while $MONITORING_ACTIVE; do
        log_performance "=== Resource Usage Check ==="
        
        check_resource_usage
        
        sleep $RESOURCE_CHECK_INTERVAL
    done
}

# Control functions
start_monitoring() {
    if [[ -f "$PID_FILE" ]]; then
        local existing_pid
        existing_pid=$(cat "$PID_FILE")
        if kill -0 "$existing_pid" 2>/dev/null; then
            echo "Monitoring is already running (PID: $existing_pid)"
            exit 1
        else
            rm -f "$PID_FILE"
        fi
    fi
    
    echo $$ > "$PID_FILE"
    log_monitor "Starting monitoring for $SCENARIO_NAME ($SCENARIO_ID)"
    log_monitor "Monitor log: $MONITOR_LOG"
    log_monitor "PID file: $PID_FILE"
    
    # Load required resources
    REQUIRED_RESOURCES=$(get_required_resources)
    log_monitor "Monitoring resources: $REQUIRED_RESOURCES"
    
    # Start monitoring loops in background
    health_monitoring_loop &
    local health_pid=$!
    
    performance_monitoring_loop &
    local perf_pid=$!
    
    resource_monitoring_loop &
    local resource_pid=$!
    
    log_monitor "Monitoring loops started (Health: $health_pid, Performance: $perf_pid, Resource: $resource_pid)"
    
    # Wait for any loop to exit
    wait
}

stop_monitoring() {
    if [[ -f "$PID_FILE" ]]; then
        local monitor_pid
        monitor_pid=$(cat "$PID_FILE")
        if kill -0 "$monitor_pid" 2>/dev/null; then
            log_monitor "Stopping monitoring (PID: $monitor_pid)"
            kill -TERM "$monitor_pid"
            rm -f "$PID_FILE"
            echo "Monitoring stopped"
        else
            echo "Monitoring is not running"
            rm -f "$PID_FILE"
        fi
    else
        echo "No monitoring PID file found"
    fi
}

status_monitoring() {
    if [[ -f "$PID_FILE" ]]; then
        local monitor_pid
        monitor_pid=$(cat "$PID_FILE")
        if kill -0 "$monitor_pid" 2>/dev/null; then
            echo "Monitoring is running (PID: $monitor_pid)"
            echo "Log file: $MONITOR_LOG"
            return 0
        else
            echo "Monitoring PID file exists but process is not running"
            rm -f "$PID_FILE"
            return 1
        fi
    else
        echo "Monitoring is not running"
        return 1
    fi
}

show_logs() {
    if [[ -f "$MONITOR_LOG" ]]; then
        tail -f "$MONITOR_LOG"
    else
        echo "No monitor log file found at $MONITOR_LOG"
        exit 1
    fi
}

# Main function
main() {
    case "${1:-start}" in
        "start")
            start_monitoring
            ;;
        "stop")
            stop_monitoring
            ;;
        "restart")
            stop_monitoring
            sleep 2
            start_monitoring
            ;;
        "status")
            status_monitoring
            ;;
        "logs")
            show_logs
            ;;
        "help"|"-h"|"--help")
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  start    - Start monitoring (default)"
            echo "  stop     - Stop monitoring"
            echo "  restart  - Restart monitoring"
            echo "  status   - Show monitoring status"
            echo "  logs     - Follow monitor logs"
            echo "  help     - Show this help message"
            echo ""
            echo "Configuration:"
            echo "  Health check interval: ${HEALTH_CHECK_INTERVAL}s"
            echo "  Performance check interval: ${PERFORMANCE_CHECK_INTERVAL}s"
            echo "  Resource check interval: ${RESOURCE_CHECK_INTERVAL}s"
            echo "  Response time threshold: ${RESPONSE_TIME_THRESHOLD_MS}ms"
            echo "  Error rate threshold: ${ERROR_RATE_THRESHOLD_PERCENT}%"
            echo ""
            ;;
        *)
            echo "Unknown command: $1"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"