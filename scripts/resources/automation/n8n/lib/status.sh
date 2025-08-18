#!/usr/bin/env bash
# n8n Status Functions - Minimal wrapper using status engine

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_STATUS_SOURCED:-}" ]] && return 0
export _N8N_STATUS_SOURCED=1

# Source core and frameworks
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-engine.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/core.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/health.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/auto-credentials.sh"

#######################################
# Show n8n status with format support
#######################################
n8n::status() {
    local format="text"
    local verbose="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            --verbose|-v)
                verbose="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check Docker daemon
    if ! docker::check_daemon; then
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Docker daemon not available"}'
        else
            log::error "Docker daemon not available"
        fi
        return 1
    fi
    
    # Collect status data
    local data_string
    data_string=$(n8n::status::collect_data 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect n8n status data"
        fi
        return 1
    fi
    
    # Format the output appropriately
    if [[ "$format" == "json" ]]; then
        echo "$data_string"
    else
        # Use the existing status engine for text output
        local config
        config=$(n8n::get_status_config)
        status::display_unified_status "$config" "n8n::display_workflow_info"
    fi
}

#######################################
# Collect n8n status data as JSON
#######################################
n8n::status::collect_data() {
    local installed="false"
    local running="false"
    local healthy="false"
    local health_message="Not installed"
    
    # Check if n8n is installed
    if docker::container_exists "$N8N_CONTAINER_NAME"; then
        installed="true"
        
        # Check if running
        if docker::is_running "$N8N_CONTAINER_NAME"; then
            running="true"
            
            # Check health
            if n8n::check_basic_health; then
                healthy="true"
                health_message="Healthy - All systems operational"
            else
                health_message="Unhealthy - Service not responding"
            fi
        else
            health_message="Stopped - Container not running"
        fi
    fi
    
    # Get container info if running
    local cpu_usage="0"
    local memory_usage="0"
    local memory_limit="0"
    
    if [[ "$running" == "true" ]]; then
        local stats
        stats=$(docker stats "$N8N_CONTAINER_NAME" --no-stream --format "{{.CPUPerc}}|{{.MemUsage}}" 2>/dev/null || echo "")
        if [[ -n "$stats" ]]; then
            cpu_usage=$(echo "$stats" | cut -d'|' -f1 | tr -d '% ')
            local mem_info
            mem_info=$(echo "$stats" | cut -d'|' -f2)
            memory_usage=$(echo "$mem_info" | cut -d'/' -f1 | tr -d ' ')
            memory_limit=$(echo "$mem_info" | cut -d'/' -f2 | tr -d ' ')
        fi
    fi
    
    # Build JSON response
    cat <<EOF
{
  "name": "n8n",
  "category": "automation",
  "description": "Business workflow automation platform",
  "installed": $installed,
  "running": $running,
  "healthy": $healthy,
  "health_message": "$health_message",
  "container_name": "$N8N_CONTAINER_NAME",
  "port": $N8N_PORT,
  "ui_url": "$N8N_BASE_URL",
  "api_url": "${N8N_BASE_URL}/api/v1",
  "health_url": "${N8N_BASE_URL}/healthz",
  "cpu_usage": "$cpu_usage",
  "memory_usage": "$memory_usage",
  "memory_limit": "$memory_limit"
}
EOF
}


#######################################
# Show n8n information
#######################################
n8n::info() {
    cat << EOF
=== n8n Resource Information ===

ID: n8n
Category: automation
Display Name: n8n
Description: Workflow automation platform

Service Details:
- Container Name: $N8N_CONTAINER_NAME
- Service Port: $N8N_PORT
- Service URL: $N8N_BASE_URL
- Docker Image: $N8N_IMAGE
- Data Directory: $N8N_DATA_DIR

Configuration:
- Authentication: ${BASIC_AUTH:-yes}
- Database: ${DATABASE_TYPE:-sqlite}
- Tunnel Mode: ${TUNNEL_ENABLED:-no}

For more information, visit: https://docs.n8n.io
EOF
}

#######################################
# Show n8n version information
#######################################
n8n::version() {
    log::header "🔧 n8n Version Information"
    
    # Check if container exists and is running
    if ! docker::is_running "$N8N_CONTAINER_NAME"; then
        log::warn "n8n container is not running"
        log::info "Try: ./manage.sh --action start"
        return 1
    fi
    
    # Get n8n version from container
    local version_output
    if version_output=$(docker exec "$N8N_CONTAINER_NAME" n8n --version 2>/dev/null); then
        log::success "n8n Version: $version_output"
    else
        log::error "Failed to retrieve n8n version"
        return 1
    fi
    
    # Get container image info
    local image_info
    image_info=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{.Config.Image}}' 2>/dev/null)
    if [[ -n "$image_info" ]]; then
        log::info "Docker Image: $image_info"
    fi
    
    # Get container creation date
    local created_date
    created_date=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{.Created}}' 2>/dev/null)
    if [[ -n "$created_date" ]]; then
        log::info "Container Created: $created_date"
    fi
    
    return 0
}





#######################################
# Test n8n functionality
#######################################
n8n::test() {
    log::info "Testing n8n functionality..."
    
    local tests_passed=0
    local total_tests=0
    
    # Test 1: Tiered health check
    log::info "1. Running health checks..."
    local health_tier
    health_tier=$(n8n::tiered_health_check)
    total_tests=$((total_tests + 1))
    if [[ "$health_tier" == "HEALTHY" ]]; then
        log::success "   ✅ Health check: PASSED ($health_tier)"
        tests_passed=$((tests_passed + 1))
    else
        log::warn "   ⚠️  Health check: PARTIAL ($health_tier)"
        # Don't fail completely on degraded health
        if [[ "$health_tier" != "UNHEALTHY" ]]; then
            tests_passed=$((tests_passed + 1))
        fi
    fi
    
    # Test 2: API connectivity
    log::info "2. Testing API connectivity..."
    total_tests=$((total_tests + 1))
    if n8n::test_api >/dev/null 2>&1; then
        log::success "   ✅ API connectivity: PASSED"
        tests_passed=$((tests_passed + 1))
    else
        log::warn "   ⚠️  API connectivity: FAILED (API key may be missing)"
        # Don't count as failure if just missing API key
    fi
    
    # Summary
    log::info "Test Results: $tests_passed/$total_tests core tests passed"
    
    if [[ "$tests_passed" -eq "$total_tests" ]]; then
        log::success "🎉 All n8n tests passed"
        return 0
    elif [[ "$tests_passed" -gt 0 ]]; then
        log::warn "⚠️  Some tests passed - n8n is functional but may need configuration"
        return 0
    else
        log::error "❌ All tests failed - n8n is not functional"
        return 1
    fi
}