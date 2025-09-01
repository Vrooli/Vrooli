#!/usr/bin/env bash
################################################################################
# Qdrant Health Monitor - Comprehensive health checking and auto-recovery
# 
# Monitors Qdrant health, performs diagnostics, and automatically recovers
# from common failure modes
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${BASH_SOURCE[0]%/*}"
LOG_FILE="$HOME/.qdrant/health.log"
mkdir -p "$HOME/.qdrant"

# Source logging utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Configuration
QDRANT_URL="${QDRANT_URL:-http://localhost:6333}"
QDRANT_CONTAINER="${QDRANT_CONTAINER:-qdrant}"
MAX_RESTART_ATTEMPTS=3
HEALTH_CHECK_TIMEOUT=10
MEMORY_THRESHOLD_PERCENT=90
RESPONSE_TIME_THRESHOLD_MS=5000

# Logging setup
exec 1> >(tee -a "$LOG_FILE")
exec 2> >(tee -a "$LOG_FILE" >&2)

#######################################
# Log with timestamp
#######################################
log_with_time() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

#######################################
# Check if Qdrant container is running
# Returns: 0 if running, 1 if not
#######################################
check_container() {
    if docker ps --format "table {{.Names}}" | grep -q "^${QDRANT_CONTAINER}$"; then
        return 0
    else
        return 1
    fi
}

#######################################
# Check Qdrant API health
# Returns: 0 if healthy, 1 if not
#######################################
check_api_health() {
    local start_time=$(date +%s%3N)
    
    # Test basic connectivity
    local response
    response=$(timeout "$HEALTH_CHECK_TIMEOUT" curl -s "$QDRANT_URL/" 2>/dev/null || echo "")
    
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    # Check response
    if [[ -z "$response" ]]; then
        log_with_time "❌ API not responding"
        return 1
    fi
    
    # Check if response contains expected fields (Qdrant returns version info, not status)
    local title
    title=$(echo "$response" | jq -r '.title // "unknown"' 2>/dev/null || echo "unknown")
    
    if [[ "$title" != "qdrant - vector search engine" ]]; then
        log_with_time "❌ API unhealthy: unexpected response"
        return 1
    fi
    
    # Check response time
    if [[ $response_time -gt $RESPONSE_TIME_THRESHOLD_MS ]]; then
        log_with_time "⚠️ Slow response: ${response_time}ms (threshold: ${RESPONSE_TIME_THRESHOLD_MS}ms)"
    fi
    
    log_with_time "✅ API healthy (${response_time}ms)"
    return 0
}

#######################################
# Check collections health
# Returns: 0 if healthy, 1 if not
#######################################
check_collections() {
    local collections_response
    collections_response=$(timeout 10 curl -s "$QDRANT_URL/collections" 2>/dev/null || echo "")
    
    if [[ -z "$collections_response" ]]; then
        log_with_time "❌ Cannot list collections"
        return 1
    fi
    
    local collection_count
    collection_count=$(echo "$collections_response" | jq '.result.collections | length' 2>/dev/null || echo "0")
    
    log_with_time "📊 Found $collection_count collections"
    
    # Check if main collections exist
    local missing_collections=()
    for collection in "vrooli-main-docs" "vrooli-main-code" "vrooli-main-workflows"; do
        if ! echo "$collections_response" | jq -e ".result.collections[] | select(.name == \"$collection\")" >/dev/null 2>&1; then
            missing_collections+=("$collection")
        fi
    done
    
    if [[ ${#missing_collections[@]} -gt 0 ]]; then
        log_with_time "⚠️ Missing collections: ${missing_collections[*]}"
    fi
    
    return 0
}

#######################################
# Check container resource usage
# Returns: 0 if healthy, 1 if critical
#######################################
check_resources() {
    local stats
    stats=$(docker stats "$QDRANT_CONTAINER" --no-stream --format "table {{.MemPerc}}\t{{.CPUPerc}}" 2>/dev/null | tail -n1)
    
    if [[ -z "$stats" ]]; then
        log_with_time "❌ Cannot get container stats"
        return 1
    fi
    
    local mem_percent=$(echo "$stats" | awk '{print $1}' | sed 's/%//')
    local cpu_percent=$(echo "$stats" | awk '{print $2}' | sed 's/%//')
    
    # Check memory usage
    if (( $(echo "$mem_percent > $MEMORY_THRESHOLD_PERCENT" | bc -l) )); then
        log_with_time "🚨 High memory usage: ${mem_percent}% (threshold: ${MEMORY_THRESHOLD_PERCENT}%)"
        return 1
    fi
    
    log_with_time "📊 Resources: Memory=${mem_percent}%, CPU=${cpu_percent}%"
    return 0
}

#######################################
# Restart Qdrant container
# Returns: 0 if successful, 1 if failed
#######################################
restart_container() {
    log_with_time "🔄 Attempting to restart Qdrant container..."
    
    if docker restart "$QDRANT_CONTAINER" >/dev/null 2>&1; then
        # Wait for container to be ready
        local attempts=0
        while [[ $attempts -lt 30 ]]; do
            if check_api_health >/dev/null 2>&1; then
                log_with_time "✅ Container restarted and healthy"
                return 0
            fi
            sleep 2
            ((attempts++))
        done
        
        log_with_time "❌ Container restarted but not responding"
        return 1
    else
        log_with_time "❌ Failed to restart container"
        return 1
    fi
}

#######################################
# Perform comprehensive health check
# Returns: 0 if healthy, 1 if critical issues
#######################################
perform_health_check() {
    log_with_time "🔍 Starting Qdrant health check..."
    
    local issues=0
    
    # Check container status
    if ! check_container; then
        log_with_time "❌ Container not running"
        return 1
    fi
    
    # Check API health
    if ! check_api_health; then
        ((issues++))
    fi
    
    # Check collections
    if ! check_collections; then
        ((issues++))
    fi
    
    # Check resources
    if ! check_resources; then
        ((issues++))
    fi
    
    if [[ $issues -eq 0 ]]; then
        log_with_time "✅ All health checks passed"
        return 0
    else
        log_with_time "⚠️ $issues health issues detected"
        return 1
    fi
}

#######################################
# Main health monitoring with auto-recovery
#######################################
main() {
    local restart_attempts=0
    
    # Initial health check
    if perform_health_check; then
        exit 0
    fi
    
    # Auto-recovery attempts
    while [[ $restart_attempts -lt $MAX_RESTART_ATTEMPTS ]]; do
        log_with_time "🚨 Health check failed, attempting recovery (attempt $((restart_attempts + 1))/$MAX_RESTART_ATTEMPTS)..."
        
        if restart_container; then
            if perform_health_check; then
                log_with_time "✅ Recovery successful"
                exit 0
            fi
        fi
        
        ((restart_attempts++))
        sleep 5
    done
    
    log_with_time "🚨 CRITICAL: Unable to recover Qdrant after $MAX_RESTART_ATTEMPTS attempts"
    
    # Send alert notification (if notification system available)
    if command -v notify-send >/dev/null 2>&1; then
        notify-send "Qdrant Critical" "Service down, manual intervention required"
    fi
    
    exit 1
}

# Run health check
main "$@"