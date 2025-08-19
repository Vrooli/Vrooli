#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
KEYCLOAK_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "${KEYCLOAK_LIB_DIR}/../../../../lib/utils/format.sh"
source "${KEYCLOAK_LIB_DIR}/../../../../lib/utils/log.sh"

# Dependencies are expected to be sourced by caller

# Get Keycloak status
keycloak::status() {
    local format="text"
    local verbose="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="${2:-text}"
                shift 2
                ;;
            --verbose)
                verbose="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status="stopped"
    local message="Keycloak is not running"
    local health="unhealthy"
    local details=""
    
    if keycloak::is_running; then
        status="running"
        local port
        port=$(keycloak::get_port)
        
        # Check health endpoint - try multiple methods
        if docker exec "${KEYCLOAK_CONTAINER_NAME}" /bin/sh -c "test -f /tmp/ready" 2>/dev/null; then
            # Keycloak has a ready marker file in newer versions
            health="healthy"
            message="Keycloak is running and healthy on port ${port}"
        elif docker run --rm --network vrooli-network alpine/curl -sf "http://vrooli-keycloak:8080/realms/master" >/dev/null 2>&1; then
            # Try accessing through Docker network
            health="healthy"
            message="Keycloak is running and healthy on port ${port}"
        elif curl -sf "http://localhost:${port}/realms/master" >/dev/null 2>&1; then
            # Try the realms endpoint instead of health endpoint
            health="healthy"
            message="Keycloak is running and healthy on port ${port}"
            
            # Get additional details if verbose
            if [[ "${verbose}" == "true" ]]; then
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
            health="unhealthy"
            message="Keycloak is running but not responding on port ${port}"
        fi
    elif keycloak::container_exists; then
        message="Keycloak container exists but is not running"
    elif keycloak::is_installed; then
        message="Keycloak is installed but not running"
    else
        message="Keycloak is not installed (Docker required)"
    fi
    
    # Convert status to boolean for consistency
    local running_bool="false"
    [[ "${status}" == "running" ]] && running_bool="true"
    
    # Use standard log utilities
    log::header "📊 Keycloak Status"
    log::info "📝 Description: Enterprise identity and access management"
    log::info "📂 Category: execution"
    echo ""
    
    log::info "📊 Basic Status:"
    
    # Check if installed
    if keycloak::is_installed; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
    fi
    
    # Check if running
    if [[ "${status}" == "running" ]]; then
        log::success "   ✅ Running: Yes"
        log::success "   ✅ Health: ${health^}"
    else
        log::error "   ❌ Running: No"
        log::warning "   ⚠️  Health: Not available"
    fi
    
    echo ""
    
    # Container info if running
    if [[ "${status}" == "running" ]]; then
        log::info "🐳 Container Info:"
        log::info "   📦 Name: ${KEYCLOAK_CONTAINER_NAME}"
        log::info "   📊 Status: running"
        log::info "   🖼️  Image: ${KEYCLOAK_IMAGE}"
        
        # Get container stats
        local cpu_usage memory_usage
        cpu_usage=$(docker stats --no-stream --format "{{.CPUPerc}}" "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
        memory_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
        
        log::info "   🔥 CPU: ${cpu_usage}"
        log::info "   💾 Memory: ${memory_usage}"
        
        echo ""
        log::info "🌐 Service Endpoints:"
        local port
        port=$(keycloak::get_port)
        log::info "   🔗 Admin Console: http://localhost:${port}"
        log::info "   🔌 API: http://localhost:${port}/realms"
        log::info "   🏥 Health: http://localhost:${port}/health"
        log::info "   📚 Docs: http://localhost:${port}/metrics"
        
        echo ""
        log::info "⚙️  Configuration:"
        log::info "   📶 Port: ${port}"
        log::info "   👤 Admin User: admin"
        log::info "   🗄️  Database: H2 (embedded)"
        log::info "   📁 Data Dir: ${KEYCLOAK_DATA_DIR}"
        
        if [[ "${verbose}" == "true" ]] && [[ -n "${details}" ]]; then
            echo ""
            log::info "📊 Additional Details:"
            log::info "   ${details}"
        fi
    fi
    
    echo ""
    log::info "📋 Status Message:"
    if [[ "${health}" == "healthy" ]]; then
        log::success "   ✅ ${message}"
    elif [[ "${status}" == "running" ]]; then
        log::warning "   ⚠️  ${message}"
    else
        log::error "   ❌ ${message}"
    fi
}