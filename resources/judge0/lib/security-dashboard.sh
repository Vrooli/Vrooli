#!/usr/bin/env bash
# Judge0 Advanced Security Dashboard
# Enhanced sandbox validation and security monitoring system

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/judge0/config/defaults.sh"

# Security configuration
SECURITY_REPORT_DIR="${JUDGE0_LOGS_DIR}/security-reports"
SECURITY_METRICS_FILE="${SECURITY_REPORT_DIR}/metrics.json"
SECURITY_EVENTS_FILE="${SECURITY_REPORT_DIR}/events.log"
VULNERABILITY_SCAN_FILE="${SECURITY_REPORT_DIR}/vulnerability-scan.json"

#######################################
# Initialize security dashboard
#######################################
judge0::security::dashboard::init() {
    mkdir -p "$SECURITY_REPORT_DIR"
    
    # Initialize metrics file if not exists
    if [[ ! -f "$SECURITY_METRICS_FILE" ]]; then
        echo '{"submissions":{},"threats":{},"sandboxes":{},"audit_log":[]}' > "$SECURITY_METRICS_FILE"
    fi
    
    log::info "Security dashboard initialized"
}

#######################################
# Display comprehensive security dashboard
#######################################
judge0::security::dashboard::show() {
    log::header "Judge0 Advanced Security Dashboard"
    
    echo
    log::info "🛡️  Security Overview"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Sandbox validation status
    judge0::security::dashboard::sandbox_status
    
    # Threat detection summary
    judge0::security::dashboard::threat_summary
    
    # Resource isolation metrics
    judge0::security::dashboard::isolation_metrics
    
    # Vulnerability scan results
    judge0::security::dashboard::vulnerability_scan
    
    # Security events
    judge0::security::dashboard::recent_events
    
    # Recommendations
    judge0::security::dashboard::recommendations
}

#######################################
# Check sandbox validation status
#######################################
judge0::security::dashboard::sandbox_status() {
    log::info "📦 Sandbox Validation Status"
    
    local workers_running=0
    local sandbox_secure=true
    local isolation_active=true
    
    # Check worker containers
    for container in $(docker ps --filter "name=judge0-workers" --format "{{.Names}}"); do
        ((workers_running++))
        
        # Check if running with proper privileges
        local privileged=$(docker inspect "$container" --format '{{.HostConfig.Privileged}}')
        if [[ "$privileged" != "true" ]]; then
            sandbox_secure=false
            log::warn "  ⚠️  Worker $container not running in privileged mode"
        fi
        
        # Check isolation
        if docker exec "$container" which isolate >/dev/null 2>&1; then
            log::success "  ✅ Isolate available in $container"
        else
            isolation_active=false
            log::error "  ❌ Isolate missing in $container"
        fi
        
        # Check cgroup mounts
        local cgroup_mounted=$(docker exec "$container" mount | grep -c cgroup || echo "0")
        if [[ $cgroup_mounted -gt 0 ]]; then
            log::success "  ✅ Cgroups mounted in $container"
        else
            log::warn "  ⚠️  Cgroups not properly mounted in $container"
        fi
    done
    
    echo
    log::info "  Workers Running: $workers_running"
    log::info "  Sandbox Secure: $([ "$sandbox_secure" = true ] && echo "✅ Yes" || echo "❌ No")"
    log::info "  Isolation Active: $([ "$isolation_active" = true ] && echo "✅ Yes" || echo "⚠️  Limited")"
    echo
}

#######################################
# Display threat detection summary
#######################################
judge0::security::dashboard::threat_summary() {
    log::info "🔍 Threat Detection Summary"
    
    local threats_today=0
    local threats_blocked=0
    local suspicious_patterns=0
    
    if [[ -f "$SECURITY_EVENTS_FILE" ]]; then
        threats_today=$(grep -c "$(date +%Y-%m-%d)" "$SECURITY_EVENTS_FILE" 2>/dev/null || echo "0")
        threats_blocked=$(grep -c "BLOCKED" "$SECURITY_EVENTS_FILE" 2>/dev/null || echo "0")
        suspicious_patterns=$(grep -c "SUSPICIOUS" "$SECURITY_EVENTS_FILE" 2>/dev/null || echo "0")
    fi
    
    echo "  Threats Detected Today: $threats_today"
    echo "  Threats Blocked: $threats_blocked"
    echo "  Suspicious Patterns: $suspicious_patterns"
    
    # Check for common attack patterns
    local sql_injection_attempts=$(docker logs vrooli-judge0-server 2>&1 | grep -ci "sql injection\|union select\|drop table" || echo "0")
    local command_injection_attempts=$(docker logs vrooli-judge0-server 2>&1 | grep -ci ";\s*rm\|&&\s*cat\||\s*wget" || echo "0")
    
    if [[ $sql_injection_attempts -gt 0 ]]; then
        log::warn "  ⚠️  SQL Injection attempts detected: $sql_injection_attempts"
    fi
    
    if [[ $command_injection_attempts -gt 0 ]]; then
        log::warn "  ⚠️  Command Injection attempts detected: $command_injection_attempts"
    fi
    
    echo
}

#######################################
# Display resource isolation metrics
#######################################
judge0::security::dashboard::isolation_metrics() {
    log::info "🔒 Resource Isolation Metrics"
    
    # Check resource limits
    local cpu_limit="${JUDGE0_CPU_TIME_LIMIT}s"
    local memory_limit="$((JUDGE0_MEMORY_LIMIT / 1024))MB"
    local process_limit="${JUDGE0_MAX_PROCESSES}"
    local network_isolated="${JUDGE0_ENABLE_NETWORK}"
    
    echo "  CPU Time Limit: $cpu_limit per execution"
    echo "  Memory Limit: $memory_limit per execution"
    echo "  Process Limit: $process_limit per execution"
    echo "  Network Access: $([ "$network_isolated" = "false" ] && echo "🔒 Disabled" || echo "⚠️  Enabled")"
    
    # Check actual container resource usage
    for container in vrooli-judge0-server config-judge0-workers-1 config-judge0-workers-2; do
        if docker ps --filter "name=$container" --format "{{.Names}}" | grep -q "$container"; then
            local stats=$(docker stats "$container" --no-stream --format "{{.CPUPerc}} {{.MemUsage}}" 2>/dev/null || echo "N/A N/A")
            echo "  $container: CPU: $(echo "$stats" | awk '{print $1}'), Memory: $(echo "$stats" | awk '{print $2}')"
        fi
    done
    
    echo
}

#######################################
# Run vulnerability scan
#######################################
judge0::security::dashboard::vulnerability_scan() {
    log::info "🔐 Vulnerability Scan Results"
    
    local scan_date=$(date +%Y-%m-%d)
    local vulnerabilities=()
    
    # Check for outdated Judge0 version
    local current_version="${JUDGE0_VERSION}"
    local latest_version="1.13.1"  # Should be fetched from API in production
    
    if [[ "$current_version" != "$latest_version" ]]; then
        vulnerabilities+=("Judge0 version outdated: $current_version (latest: $latest_version)")
    fi
    
    # Check for exposed ports
    local exposed_ports=$(docker inspect vrooli-judge0-server --format '{{range $p, $conf := .NetworkSettings.Ports}}{{$p}} {{end}}' 2>/dev/null || echo "")
    if [[ -n "$exposed_ports" ]]; then
        echo "  Exposed Ports: $exposed_ports"
    fi
    
    # Check for API authentication
    if [[ "${JUDGE0_ENABLE_AUTHENTICATION}" != "true" ]]; then
        vulnerabilities+=("API authentication disabled")
        log::warn "  ⚠️  API authentication is disabled"
    else
        log::success "  ✅ API authentication enabled"
    fi
    
    # Check for secure configuration
    if [[ -f "${JUDGE0_CONFIG_DIR}/api_key" ]]; then
        local api_key_perms=$(stat -c %a "${JUDGE0_CONFIG_DIR}/api_key" 2>/dev/null || echo "000")
        if [[ "$api_key_perms" != "600" ]]; then
            vulnerabilities+=("API key file permissions too open: $api_key_perms")
            log::warn "  ⚠️  API key file permissions: $api_key_perms (should be 600)"
        else
            log::success "  ✅ API key file properly secured"
        fi
    fi
    
    # Save vulnerability scan results
    echo "{\"scan_date\":\"$scan_date\",\"vulnerabilities\":$(printf '%s\n' "${vulnerabilities[@]}" | jq -R . | jq -s .)}" > "$VULNERABILITY_SCAN_FILE"
    
    if [[ ${#vulnerabilities[@]} -eq 0 ]]; then
        log::success "  ✅ No critical vulnerabilities detected"
    else
        log::warn "  ⚠️  Found ${#vulnerabilities[@]} potential vulnerabilities"
        for vuln in "${vulnerabilities[@]}"; do
            echo "     - $vuln"
        done
    fi
    
    echo
}

#######################################
# Display recent security events
#######################################
judge0::security::dashboard::recent_events() {
    log::info "📊 Recent Security Events (Last 24 Hours)"
    
    if [[ -f "$SECURITY_EVENTS_FILE" ]]; then
        local events_count=$(tail -100 "$SECURITY_EVENTS_FILE" | grep -c "$(date +%Y-%m-%d)" || echo "0")
        echo "  Total Events: $events_count"
        
        if [[ $events_count -gt 0 ]]; then
            echo "  Recent Events:"
            tail -5 "$SECURITY_EVENTS_FILE" | while read -r line; do
                echo "    $line"
            done
        fi
    else
        echo "  No security events recorded"
    fi
    
    echo
}

#######################################
# Display security recommendations
#######################################
judge0::security::dashboard::recommendations() {
    log::info "💡 Security Recommendations"
    
    local recommendations=()
    
    # Check if running in production
    if [[ "${JUDGE0_ENABLE_NETWORK}" == "true" ]]; then
        recommendations+=("Disable network access for submissions (JUDGE0_ENABLE_NETWORK=false)")
    fi
    
    # Check resource limits
    if [[ ${JUDGE0_CPU_TIME_LIMIT} -gt 10 ]]; then
        recommendations+=("Reduce CPU time limit to prevent resource exhaustion")
    fi
    
    if [[ ${JUDGE0_MEMORY_LIMIT} -gt 262144 ]]; then
        recommendations+=("Reduce memory limit to prevent memory exhaustion")
    fi
    
    # Check worker count
    local worker_count=$(docker ps --filter "name=judge0-workers" --format "{{.Names}}" | wc -l)
    if [[ $worker_count -lt 2 ]]; then
        recommendations+=("Increase worker count for better isolation and performance")
    fi
    
    if [[ ${#recommendations[@]} -eq 0 ]]; then
        log::success "  ✅ Security configuration optimal"
    else
        for rec in "${recommendations[@]}"; do
            echo "  • $rec"
        done
    fi
    
    echo
}

#######################################
# Start continuous security monitoring
#######################################
judge0::security::dashboard::monitor() {
    log::info "Starting continuous security monitoring..."
    log::info "Press Ctrl+C to stop"
    
    judge0::security::dashboard::init
    
    while true; do
        clear
        judge0::security::dashboard::show
        
        # Log security metrics
        local timestamp=$(date +%Y-%m-%dT%H:%M:%S)
        local metrics=$(judge0::security::dashboard::collect_metrics)
        echo "[$timestamp] $metrics" >> "$SECURITY_EVENTS_FILE"
        
        sleep 30
    done
}

#######################################
# Collect security metrics
#######################################
judge0::security::dashboard::collect_metrics() {
    local active_submissions=$(curl -s http://localhost:${JUDGE0_PORT}/submissions?wait=false | jq length 2>/dev/null || echo "0")
    local worker_status="OK"
    
    # Check worker health
    for container in $(docker ps --filter "name=judge0-workers" --format "{{.Names}}"); do
        if ! docker exec "$container" ps aux >/dev/null 2>&1; then
            worker_status="DEGRADED"
        fi
    done
    
    echo "submissions=$active_submissions,workers=$worker_status"
}

# Export functions for use in other scripts
export -f judge0::security::dashboard::init
export -f judge0::security::dashboard::show
export -f judge0::security::dashboard::monitor

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-show}" in
        show)
            judge0::security::dashboard::init
            judge0::security::dashboard::show
            ;;
        monitor)
            judge0::security::dashboard::monitor
            ;;
        *)
            log::error "Usage: $0 [show|monitor]"
            exit 1
            ;;
    esac
fi