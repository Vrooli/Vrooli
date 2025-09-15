#!/usr/bin/env bash
################################################################################
# Keycloak Performance Monitoring Functions
################################################################################

set -euo pipefail

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/common.sh"

# Get Keycloak port from registry
KEYCLOAK_PORT=$(source "${APP_ROOT}/scripts/resources/port_registry.sh" && port_registry::get keycloak)

################################################################################
# Monitoring Functions
################################################################################

monitor::metrics() {
    log::info "🔍 Keycloak Performance Metrics"
    
    # Get metrics from Keycloak metrics endpoint
    local metrics_response
    if metrics_response=$(curl -sf "http://localhost:${KEYCLOAK_PORT}/metrics" 2>/dev/null); then
        log::info "📊 JVM Metrics:"
        echo "$metrics_response" | grep -E "jvm_memory|jvm_threads|jvm_gc" | head -10
        
        log::info "📈 HTTP Metrics:"
        echo "$metrics_response" | grep -E "http_server_requests|http_server_connections" | head -10
        
        log::info "🔐 Keycloak Metrics:"
        echo "$metrics_response" | grep -E "keycloak_logins|keycloak_failed_login|keycloak_registrations" | head -10
    else
        log::warning "Metrics endpoint not available - using alternative metrics"
    fi
    
    # Get container stats
    log::info "🐳 Container Statistics:"
    docker stats --no-stream "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null || log::error "Container not running"
    
    # Get memory usage details
    log::info "💾 Memory Details:"
    local mem_usage=$(docker exec "${KEYCLOAK_CONTAINER_NAME}" sh -c 'cat /proc/meminfo' 2>/dev/null | grep -E "MemTotal|MemFree|MemAvailable" || echo "N/A")
    echo "$mem_usage"
    
    # Get connection statistics
    log::info "🔌 Connection Statistics:"
    local connections=$(docker exec "${KEYCLOAK_CONTAINER_NAME}" sh -c 'netstat -an 2>/dev/null | grep -c ESTABLISHED' || echo "0")
    log::info "  Active connections: $connections"
    
    # Get database pool stats (if using PostgreSQL)
    if [[ "${KEYCLOAK_DB_VENDOR:-h2}" == "postgres" ]]; then
        log::info "🗄️ Database Pool:"
        local db_pool=$(docker exec "${KEYCLOAK_CONTAINER_NAME}" sh -c 'jstat -gc $(pgrep java) 2>/dev/null | tail -1' || echo "N/A")
        echo "  $db_pool"
    fi
}

monitor::health() {
    log::info "🏥 Keycloak Health Check"
    
    # Check container status
    if docker ps --format "{{.Names}}" | grep -q "^${KEYCLOAK_CONTAINER_NAME}$"; then
        log::success "Container Status: Running"
    else
        log::error "Container Status: Not Running"
        return 1
    fi
    
    # Check health endpoint
    local start_time=$(date +%s%N)
    if timeout 5 curl -sf "http://localhost:${KEYCLOAK_PORT}/health" &>/dev/null; then
        local end_time=$(date +%s%N)
        local response_time=$(( ($end_time - $start_time) / 1000000 ))
        log::success "Health Endpoint: OK (${response_time}ms)"
    else
        log::error "Health Endpoint: Failed"
    fi
    
    # Check OIDC discovery
    start_time=$(date +%s%N)
    if timeout 5 curl -sf "http://localhost:${KEYCLOAK_PORT}/realms/master/.well-known/openid-configuration" &>/dev/null; then
        local end_time=$(date +%s%N)
        local response_time=$(( ($end_time - $start_time) / 1000000 ))
        log::success "OIDC Discovery: OK (${response_time}ms)"
    else
        log::error "OIDC Discovery: Failed"
    fi
    
    # Check admin console
    start_time=$(date +%s%N)
    if timeout 5 curl -sf -o /dev/null -w "%{http_code}" "http://localhost:${KEYCLOAK_PORT}/admin" | grep -q "302"; then
        local end_time=$(date +%s%N)
        local response_time=$(( ($end_time - $start_time) / 1000000 ))
        log::success "Admin Console: OK (${response_time}ms)"
    else
        log::error "Admin Console: Failed"
    fi
}

monitor::performance() {
    log::info "⚡ Performance Analysis"
    
    # Test token generation performance
    log::info "Testing token generation performance..."
    
    local total_time=0
    local iterations=5
    
    for i in $(seq 1 $iterations); do
        local start_time=$(date +%s%N)
        
        # Get admin token
        if keycloak::get_admin_token &>/dev/null; then
            local end_time=$(date +%s%N)
            local response_time=$(( ($end_time - $start_time) / 1000000 ))
            total_time=$((total_time + response_time))
            log::info "  Token generation $i: ${response_time}ms"
        else
            log::error "  Token generation $i: Failed"
        fi
    done
    
    if [[ $total_time -gt 0 ]]; then
        local avg_time=$((total_time / iterations))
        log::success "Average token generation time: ${avg_time}ms"
        
        # Performance evaluation
        if [[ $avg_time -lt 100 ]]; then
            log::success "✅ Performance: Excellent (<100ms)"
        elif [[ $avg_time -lt 300 ]]; then
            log::success "✅ Performance: Good (<300ms)"
        elif [[ $avg_time -lt 500 ]]; then
            log::warning "⚠️ Performance: Acceptable (<500ms)"
        else
            log::error "❌ Performance: Poor (>${avg_time}ms)"
        fi
    fi
}

monitor::realms() {
    log::info "🏢 Realm Statistics"
    
    # Get admin token
    local access_token
    access_token=$(keycloak::get_admin_token) || {
        log::error "Failed to get admin token"
        return 1
    }
    
    # Get realm list
    local realms_response
    if realms_response=$(curl -sf \
        -H "Authorization: Bearer $access_token" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms" 2>/dev/null); then
        
        local realm_count=$(echo "$realms_response" | jq '. | length' 2>/dev/null || echo "0")
        log::info "Total Realms: $realm_count"
        
        # Get details for each realm
        echo "$realms_response" | jq -r '.[].realm' 2>/dev/null | while read -r realm; do
            if [[ -n "$realm" ]]; then
                # Get realm statistics
                local realm_stats
                if realm_stats=$(curl -sf \
                    -H "Authorization: Bearer $access_token" \
                    "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}" 2>/dev/null); then
                    
                    local enabled=$(echo "$realm_stats" | jq -r '.enabled' 2>/dev/null)
                    log::info "  📁 Realm: $realm (Enabled: $enabled)"
                    
                    # Get user count if possible
                    local user_count
                    if user_count=$(curl -sf \
                        -H "Authorization: Bearer $access_token" \
                        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/users/count" 2>/dev/null); then
                        log::info "     Users: $user_count"
                    fi
                    
                    # Get client count
                    local client_count
                    if client_count=$(curl -sf \
                        -H "Authorization: Bearer $access_token" \
                        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/clients" 2>/dev/null | jq '. | length' 2>/dev/null); then
                        log::info "     Clients: $client_count"
                    fi
                fi
            fi
        done
    else
        log::error "Failed to retrieve realm statistics"
    fi
}

monitor::dashboard() {
    log::header "🎯 Keycloak Monitoring Dashboard"
    echo ""
    
    monitor::health
    echo ""
    
    monitor::performance
    echo ""
    
    monitor::realms
    echo ""
    
    monitor::metrics
    echo ""
    
    monitor::history
    echo ""
    
    log::success "Dashboard refresh complete"
}

monitor::history() {
    log::info "📈 Metrics History"
    
    local history_file="${RESOURCE_DIR}/logs/metrics-history.json"
    local timestamp=$(date -Iseconds)
    
    # Ensure logs directory exists
    mkdir -p "${RESOURCE_DIR}/logs"
    
    # Collect current metrics
    local cpu_usage=$(docker stats --no-stream --format "{{.CPUPerc}}" "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null | tr -d '%')
    local mem_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null | awk '{print $1}' | tr -d 'MiB')
    local connections=$(docker exec "${KEYCLOAK_CONTAINER_NAME}" sh -c 'netstat -an 2>/dev/null | grep -c ESTABLISHED' || echo "0")
    
    # Append to history
    if [[ ! -f "$history_file" ]]; then
        echo "[]" > "$history_file"
    fi
    
    # Add new entry
    local new_entry=$(jq -n \
        --arg ts "$timestamp" \
        --arg cpu "$cpu_usage" \
        --arg mem "$mem_usage" \
        --arg conn "$connections" \
        '{timestamp: $ts, cpu: $cpu, memory: $mem, connections: $conn}')
    
    # Keep only last 100 entries
    jq ". += [$new_entry] | .[-100:]" "$history_file" > "${history_file}.tmp" && mv "${history_file}.tmp" "$history_file"
    
    # Display recent history
    log::info "Recent metrics (last 5 entries):"
    jq -r '.[-5:] | .[] | "  \(.timestamp): CPU=\(.cpu)%, Mem=\(.memory)MB, Conn=\(.connections)"' "$history_file" 2>/dev/null || log::warning "No history available"
    
    # Calculate averages
    if [[ -f "$history_file" ]]; then
        local avg_cpu=$(jq -r '[.[] | .cpu | tonumber] | add/length' "$history_file" 2>/dev/null || echo "N/A")
        local avg_mem=$(jq -r '[.[] | .memory | tonumber] | add/length' "$history_file" 2>/dev/null || echo "N/A")
        local avg_conn=$(jq -r '[.[] | .connections | tonumber] | add/length' "$history_file" 2>/dev/null || echo "N/A")
        
        log::info "Averages:"
        log::info "  CPU: ${avg_cpu}%"
        log::info "  Memory: ${avg_mem}MB"
        log::info "  Connections: ${avg_conn}"
    fi
}