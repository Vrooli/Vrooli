#!/usr/bin/env bash
# Judge0 Security Monitoring Script
# Monitors and alerts on suspicious container activity

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
JUDGE0_LIB_DIR="${APP_ROOT}/resources/judge0/lib"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${JUDGE0_LIB_DIR}/../../../../lib/utils/var.sh"

# Source logging functions
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common/logging.sh" || {
    echo "ERROR: Failed to source logging functions" >&2
    exit 1
}

# Load Judge0 config for proper data directory
# shellcheck disable=SC1091
source "${JUDGE0_LIB_DIR}/../config/defaults.sh"

# Configuration using proper variables instead of hardcoded paths
ALERT_LOG="${JUDGE0_LOGS_DIR}/security-alerts.log"
MONITOR_INTERVAL=5  # seconds
CPU_ALERT_THRESHOLD=80  # percent
MEMORY_ALERT_THRESHOLD=90  # percent
PROCESS_ALERT_THRESHOLD=150  # number of processes

#######################################
# Initialize monitoring
#######################################
monitor::init() {
    local log_dir=${ALERT_LOG%/*
    mkdir -p "$log_dir"
    
    log::info "Starting Judge0 security monitoring..."
    log::info "CPU threshold: ${CPU_ALERT_THRESHOLD}%"
    log::info "Memory threshold: ${MEMORY_ALERT_THRESHOLD}%"
    log::info "Process threshold: ${PROCESS_ALERT_THRESHOLD}"
}

#######################################
# Log security alert
#######################################
monitor::alert() {
    local severity="$1"
    local message="$2"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo "[$timestamp] [$severity] $message" >> "$ALERT_LOG"
    
    case "$severity" in
        "CRITICAL")
            log::error "SECURITY ALERT: $message"
            # In production, send notification (email, webhook, etc)
            ;;
        "WARNING")
            log::warn "Security Warning: $message"
            ;;
        "INFO")
            log::info "Security Notice: $message"
            ;;
    esac
}

#######################################
# Check container resource usage
#######################################
monitor::check_resources() {
    local container_name="$1"
    
    # Get container stats
    local stats=$(docker stats "$container_name" --no-stream --format "{{json .}}" 2>/dev/null || echo "{}")
    
    if [[ "$stats" == "{}" ]]; then
        return 1
    fi
    
    # Parse CPU and memory usage
    local cpu_percent=$(echo "$stats" | jq -r '.CPUPerc' | sed 's/%//')
    local mem_percent=$(echo "$stats" | jq -r '.MemPerc' | sed 's/%//')
    
    # Check thresholds
    if (( $(echo "$cpu_percent > $CPU_ALERT_THRESHOLD" | bc -l) )); then
        monitor::alert "WARNING" "Container $container_name CPU usage high: ${cpu_percent}%"
    fi
    
    if (( $(echo "$mem_percent > $MEMORY_ALERT_THRESHOLD" | bc -l) )); then
        monitor::alert "CRITICAL" "Container $container_name memory usage critical: ${mem_percent}%"
    fi
}

#######################################
# Check for suspicious processes
#######################################
monitor::check_processes() {
    local container_name="$1"
    
    # Count processes in container
    local process_count=$(docker exec "$container_name" sh -c 'ps aux | wc -l' 2>/dev/null || echo "0")
    
    if [[ "$process_count" -gt "$PROCESS_ALERT_THRESHOLD" ]]; then
        monitor::alert "WARNING" "Container $container_name has high process count: $process_count"
        
        # Log top processes
        local top_processes=$(docker exec "$container_name" sh -c 'ps aux --sort=-%cpu | head -5' 2>/dev/null || echo "Failed to get processes")
        monitor::alert "INFO" "Top processes: $top_processes"
    fi
}

#######################################
# Check for network activity
#######################################
monitor::check_network() {
    local container_name="$1"
    
    # Check for any network connections (should be none with our isolation)
    local connections=$(docker exec "$container_name" sh -c 'netstat -an 2>/dev/null | grep -E "ESTABLISHED|LISTEN" | wc -l' 2>/dev/null || echo "0")
    
    if [[ "$connections" -gt "0" ]]; then
        monitor::alert "CRITICAL" "Container $container_name has unexpected network connections: $connections"
        
        # Log connection details
        local conn_details=$(docker exec "$container_name" sh -c 'netstat -an 2>/dev/null | grep -E "ESTABLISHED|LISTEN"' || echo "Failed to get details")
        monitor::alert "INFO" "Connection details: $conn_details"
    fi
}

#######################################
# Check file system changes
#######################################
monitor::check_filesystem() {
    local container_name="$1"
    
    # Check for writes to unexpected locations
    local writes=$(docker exec "$container_name" sh -c 'find / -type f -mmin -1 2>/dev/null | grep -v -E "^/(tmp|var/tmp|judge0/tmp)" | wc -l' || echo "0")
    
    if [[ "$writes" -gt "0" ]]; then
        monitor::alert "WARNING" "Container $container_name has unexpected file writes"
        
        # Log file details
        local file_list=$(docker exec "$container_name" sh -c 'find / -type f -mmin -1 2>/dev/null | grep -v -E "^/(tmp|var/tmp|judge0/tmp)" | head -10' || echo "Failed to get files")
        monitor::alert "INFO" "Modified files: $file_list"
    fi
}

#######################################
# Monitor container logs
#######################################
monitor::check_logs() {
    local container_name="$1"
    
    # Check for suspicious patterns in logs
    local suspicious_patterns=(
        "privilege escalation"
        "root shell"
        "/etc/passwd"
        "/etc/shadow"
        "kernel exploit"
        "CVE-"
        "chmod 777"
        "rm -rf /"
    )
    
    for pattern in "${suspicious_patterns[@]}"; do
        if docker logs "$container_name" --tail 100 2>&1 | grep -qi "$pattern"; then
            monitor::alert "CRITICAL" "Suspicious pattern found in $container_name logs: $pattern"
        fi
    done
}

#######################################
# Main monitoring loop
#######################################
monitor::run() {
    monitor::init
    
    while true; do
        # Monitor main server
        monitor::check_resources "vrooli-judge0-server" || true
        monitor::check_logs "vrooli-judge0-server" || true
        
        # Monitor workers
        for worker in $(docker ps --filter "name=vrooli-judge0-workers" --format "{{.Names}}"); do
            monitor::check_resources "$worker" || true
            monitor::check_processes "$worker" || true
            monitor::check_network "$worker" || true
            monitor::check_filesystem "$worker" || true
            monitor::check_logs "$worker" || true
        done
        
        sleep "$MONITOR_INTERVAL"
    done
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    monitor::run
fi