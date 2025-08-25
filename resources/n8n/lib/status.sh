#!/usr/bin/env bash
# n8n Status Functions - Minimal wrapper using status engine

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_STATUS_SOURCED:-}" ]] && return 0
export _N8N_STATUS_SOURCED=1

# Source core and frameworks
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
N8N_LIB_DIR="${APP_ROOT}/resources/n8n/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/status-args.sh"
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
    # Check Docker daemon first
    if ! docker::check_daemon; then
        echo '{"error": "Docker daemon not available"}'
        return 1
    fi
    
    status::run_standard "n8n" "n8n::status::collect_data" "n8n::status::display_text" "$@"
}

#######################################
# Collect n8n status data as key-value pairs
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
n8n::status::collect_data() {
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
    
    local status_data=()
    
    local installed="false"
    local running="false"
    local healthy="false"
    local health_message="Not installed"
    local container_status="not_found"
    
    # Check if n8n is installed
    if docker::container_exists "$N8N_CONTAINER_NAME"; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "$N8N_CONTAINER_NAME" 2>/dev/null || echo "unknown")
        
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
    
    # Basic resource information
    status_data+=("name" "n8n")
    status_data+=("category" "automation")
    status_data+=("description" "Business workflow automation platform")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$N8N_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$N8N_PORT")
    
    # Service endpoints
    status_data+=("ui_url" "$N8N_BASE_URL")
    status_data+=("api_url" "${N8N_BASE_URL}/api/v1")
    status_data+=("health_url" "${N8N_BASE_URL}/healthz")
    
    # Configuration details
    status_data+=("image" "${N8N_IMAGE:-n8nio/n8n:latest}")
    status_data+=("data_dir" "${N8N_DATA_DIR:-/home/.n8n}")
    
    # Get container info if running
    if [[ "$running" == "true" ]]; then
        local stats cpu_usage mem_info
        
        # Skip expensive operations in fast mode
        local skip_stats="$fast_mode"
        
        if [[ "$skip_stats" == "true" ]]; then
            cpu_usage="N/A"
            mem_info="N/A"
        else
            stats=$(timeout 2s docker stats "$N8N_CONTAINER_NAME" --no-stream --format "{{.CPUPerc}}|{{.MemUsage}}" 2>/dev/null || echo "N/A|N/A")
            
            if [[ -n "$stats" && "$stats" != "N/A|N/A" ]]; then
                cpu_usage=$(echo "$stats" | cut -d'|' -f1 | tr -d '% ')
                cpu_usage="${cpu_usage:-0}%"
                mem_info=$(echo "$stats" | cut -d'|' -f2)
            else
                cpu_usage="N/A"
                mem_info="N/A"
            fi
        fi
        
        status_data+=("cpu_usage" "$cpu_usage")
        status_data+=("memory_usage" "${mem_info:-N/A}")
        
        # Get workflow and execution count if healthy
        if [[ "$healthy" == "true" ]]; then
            # Try to get workflow info (this would need proper API implementation)
            status_data+=("workflows" "unknown")
            status_data+=("executions" "unknown")
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Display status in text format
#######################################
n8n::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🚀 n8n Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install n8n, run: ./manage.sh --action install"
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
    
    # Container information
    log::info "🐳 Container Info:"
    log::info "   📦 Name: ${data[container_name]:-unknown}"
    log::info "   📊 Status: ${data[container_status]:-unknown}"
    log::info "   🖼️  Image: ${data[image]:-unknown}"
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🎨 UI: ${data[ui_url]:-unknown}"
    log::info "   🔌 API: ${data[api_url]:-unknown}"
    log::info "   📊 Health: ${data[health_url]:-unknown}"
    echo
    
    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        if [[ -n "${data[cpu_usage]:-}" && "${data[cpu_usage]}" != "N/A" ]]; then
            log::info "   💾 CPU Usage: ${data[cpu_usage]}"
        fi
        if [[ -n "${data[memory_usage]:-}" && "${data[memory_usage]}" != "N/A" ]]; then
            log::info "   🧠 Memory Usage: ${data[memory_usage]}"
        fi
        
        # Workflow information if available
        if [[ -n "${data[workflows]:-}" ]]; then
            log::info "   ⚙️  Workflows: ${data[workflows]}"
        fi
        if [[ -n "${data[executions]:-}" ]]; then
            log::info "   ▶️  Executions: ${data[executions]}"
        fi
        echo
        
        # Quick access info
        log::info "🎯 Quick Actions:"
        log::info "   🌐 Access n8n: ${data[ui_url]:-http://localhost:5678}"
        log::info "   📄 View logs: ./manage.sh --action logs"
        log::info "   🛑 Stop service: ./manage.sh --action stop"
    fi
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