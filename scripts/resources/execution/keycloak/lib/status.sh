#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
KEYCLOAK_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "${KEYCLOAK_LIB_DIR}/../../../../lib/utils/format.sh"
source "${KEYCLOAK_LIB_DIR}/../../../../lib/utils/log.sh"

# Source status-args library
source "${KEYCLOAK_LIB_DIR}/../../../lib/status-args.sh"

# Dependencies are expected to be sourced by caller

# Collect Keycloak status data in format-agnostic structure
keycloak::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status="stopped"
    local message="Keycloak is not running"
    local health="false"
    local details=""
    local installed="false"
    local running="false"
    
    if keycloak::is_installed; then
        installed="true"
    fi
    
    if keycloak::is_running; then
        running="true"
        status="running"
        local port
        port=$(keycloak::get_port)
        
        # Check health endpoint - try multiple methods (skip in fast mode)
        if [[ "$fast_mode" == "true" ]]; then
            health="true"
            message="Keycloak is running (fast mode)"
        elif docker exec "${KEYCLOAK_CONTAINER_NAME}" /bin/sh -c "test -f /tmp/ready" 2>/dev/null; then
            # Keycloak has a ready marker file in newer versions
            health="true"
            message="Keycloak is running and healthy on port ${port}"
        elif docker run --rm --network vrooli-network alpine/curl -sf "http://vrooli-keycloak:8080/realms/master" >/dev/null 2>&1; then
            # Try accessing through Docker network
            health="true"
            message="Keycloak is running and healthy on port ${port}"
        elif curl -sf "http://localhost:${port}/realms/master" >/dev/null 2>&1; then
            # Try the realms endpoint instead of health endpoint
            health="true"
            message="Keycloak is running and healthy on port ${port}"
            
            # Get additional details if verbose
            if [[ "${STATUS_VERBOSE:-false}" == "true" && "$fast_mode" == "false" ]]; then
                local container_ip
                container_ip=$(keycloak::get_container_ip)
                local uptime
                uptime=$(docker inspect --format='{{.State.StartedAt}}' "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
                
                details="Container IP: ${container_ip}, Started: ${uptime}"
                
                # Try to get realm count
                local token
                token=$(keycloak::get_admin_token)
                if [[ -n "${token}" ]]; then
                    local realm_count
                    realm_count=$(curl -sf -H "Authorization: Bearer ${token}" \
                        "http://localhost:${port}/admin/realms" 2>/dev/null | \
                        jq '. | length' 2>/dev/null || echo "unknown")
                    details="${details}, Realms: ${realm_count}"
                fi
            fi
        else
            message="Keycloak is running but not responding on port ${port}"
        fi
    elif keycloak::container_exists; then
        message="Keycloak container exists but is not running"
    elif keycloak::is_installed; then
        message="Keycloak is installed but not running"
    else
        message="Keycloak is not installed (Docker required)"
    fi
    
    # Get container stats (skip in fast mode)
    local cpu_usage="N/A"
    local memory_usage="N/A"
    if [[ "$running" == "true" && "$fast_mode" == "false" ]]; then
        cpu_usage=$(timeout 2s docker stats --no-stream --format "{{.CPUPerc}}" "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
        memory_usage=$(timeout 2s docker stats --no-stream --format "{{.MemUsage}}" "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
    fi
    
    # Build status data array
    local status_data=(
        "name" "keycloak"
        "category" "execution"
        "description" "Enterprise identity and access management"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$health"
        "health_message" "$message"
        "status" "$status"
        "port" "$(keycloak::get_port)"
        "container_name" "${KEYCLOAK_CONTAINER_NAME:-unknown}"
        "image" "${KEYCLOAK_IMAGE:-unknown}"
        "cpu_usage" "$cpu_usage"
        "memory_usage" "$memory_usage"
        "data_dir" "${KEYCLOAK_DATA_DIR:-unknown}"
    )
    
    if [[ -n "$details" ]]; then
        status_data+=("details" "$details")
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
keycloak::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Get verbose mode from environment or default to false
    local verbose="${STATUS_VERBOSE:-false}"
    
    # Header
    log::header "🔐 Keycloak Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   Keycloak requires Docker. Run: resource-keycloak install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Container info if running
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::info "🐳 Container Info:"
        log::info "   📦 Name: ${data[container_name]:-unknown}"
        log::info "   🖼️  Image: ${data[image]:-unknown}"
        log::info "   🔥 CPU: ${data[cpu_usage]:-unknown}"
        log::info "   💾 Memory: ${data[memory_usage]:-unknown}"
        echo
        
        log::info "🌐 Service Endpoints:"
        log::info "   🔗 Admin Console: http://localhost:${data[port]:-unknown}"
        log::info "   🔌 API: http://localhost:${data[port]:-unknown}/realms"
        log::info "   🏥 Health: http://localhost:${data[port]:-unknown}/health"
        log::info "   📚 Docs: http://localhost:${data[port]:-unknown}/metrics"
        echo
        
        log::info "⚙️  Configuration:"
        log::info "   📶 Port: ${data[port]:-unknown}"
        log::info "   👤 Admin User: admin"
        log::info "   🗄️  Database: H2 (embedded)"
        log::info "   📁 Data Dir: ${data[data_dir]:-unknown}"
        echo
        
        if [[ "$verbose" == "true" && -n "${data[details]:-}" ]]; then
            log::info "📊 Additional Details:"
            log::info "   ${data[details]}"
            echo
        fi
    fi
    
    log::info "📋 Status Message:"
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ ${data[health_message]:-Healthy}"
    elif [[ "${data[running]:-false}" == "true" ]]; then
        log::warn "   ⚠️  ${data[health_message]:-Running but unhealthy}"
    else
        log::error "   ❌ ${data[health_message]:-Not running}"
    fi
    echo
}

# Main status function using standard wrapper
keycloak::status() {
    status::run_standard "keycloak" "keycloak::status::collect_data" "keycloak::status::display_text" "$@"
}