#!/usr/bin/env bash
# Unified Resource Status Display Engine
# Provides consistent status display across all Vrooli resources

# Source guard to prevent multiple sourcing
[[ -n "${_STATUS_ENGINE_SOURCED:-}" ]] && return 0
_STATUS_ENGINE_SOURCED=1

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/http-utils.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/wait-utils.sh"

#######################################
# Display unified resource status with consistent formatting
# Args: $1 - status_config (JSON configuration)
#       $2 - custom_sections_func (optional function for resource-specific sections)
# Returns: 0 always
#
# Status Config Schema:
# {
#   "resource": {
#     "name": "resource-name",
#     "category": "automation|ai|storage|etc",
#     "description": "Brief description",
#     "port": 1234,
#     "container_name": "container-name",
#     "data_dir": "/path/to/data" (optional)
#   },
#   "endpoints": {
#     "ui": "http://localhost:1234",
#     "api": "http://localhost:1234/api" (optional),
#     "health": "http://localhost:1234/health" (optional)
#   },
#   "health_tiers": {
#     "healthy": "Success message",
#     "degraded": "Warning with fix instructions",
#     "unhealthy": "Error with fix instructions"
#   },
#   "auth": {
#     "type": "basic|api-key|none",
#     "status_func": "function_to_check_auth" (optional)
#   }
# }
#######################################
status::display_unified_status() {
    local status_config="$1"
    local custom_sections_func="${2:-}"
    
    # Validate config
    if ! status::_validate_config "$status_config"; then
        log::error "Invalid status configuration provided"
        return 1
    fi
    
    # Extract basic resource info
    local resource_name
    local category
    local description
    local container_name
    
    resource_name=$(echo "$status_config" | jq -r '.resource.name // empty')
    category=$(echo "$status_config" | jq -r '.resource.category // "unknown"')
    description=$(echo "$status_config" | jq -r '.resource.description // empty')
    container_name=$(echo "$status_config" | jq -r '.resource.container_name // empty')
    
    # Display header
    status::_display_header "$resource_name" "$category" "$description"
    echo
    
    # Display basic status (installed, running, healthy)
    status::_display_basic_status "$status_config"
    echo
    
    # Only show detailed info if resource is installed
    if status::_is_resource_installed "$container_name"; then
        # Display container information
        status::_display_container_info "$status_config"
        echo
        
        # Display service details (endpoints)
        status::_display_service_details "$status_config"
        echo
        
        # Display authentication information
        status::_display_authentication_info "$status_config"
        echo
        
        # Display health feedback with actionable advice
        status::_display_health_feedback "$status_config"
        
        # Display custom resource-specific sections if provided
        if [[ -n "$custom_sections_func" ]] && command -v "$custom_sections_func" &>/dev/null; then
            echo
            "$custom_sections_func" "$status_config"
        fi
    else
        # Show installation guidance
        status::_display_installation_guidance "$status_config"
    fi
    
    return 0
}

#######################################
# Display tiered health status with actionable feedback
# Args: $1 - health_tier (HEALTHY|DEGRADED|UNHEALTHY)
#       $2 - status_config (JSON configuration)
# Returns: 0 always
#######################################
status::display_health_tier() {
    local health_tier="$1"
    local status_config="$2"
    
    local healthy_msg
    local degraded_msg  
    local unhealthy_msg
    
    healthy_msg=$(echo "$status_config" | jq -r '.health_tiers.healthy // "All systems operational"')
    degraded_msg=$(echo "$status_config" | jq -r '.health_tiers.degraded // "Service degraded"')
    unhealthy_msg=$(echo "$status_config" | jq -r '.health_tiers.unhealthy // "Service unhealthy"')
    
    case "$health_tier" in
        "HEALTHY")
            log::success "✅ Health: HEALTHY - $healthy_msg"
            ;;
        "DEGRADED")
            log::warn "⚠️  Health: DEGRADED - $degraded_msg"
            ;;
        "UNHEALTHY")
            log::error "❌ Health: UNHEALTHY - $unhealthy_msg"
            ;;
        *)
            log::warn "❓ Health: UNKNOWN - Unable to determine health status"
            ;;
    esac
}

#######################################
# Check if resource supports tiered health checking
# Args: $1 - resource_name
# Returns: 0 if supports tiered health, 1 otherwise
#######################################
status::has_tiered_health() {
    local resource_name="$1"
    
    # Check if resource has a tiered health function
    if command -v "${resource_name}::tiered_health_check" &>/dev/null; then
        return 0
    fi
    return 1
}

#######################################
# Get resource health tier
# Args: $1 - status_config (JSON configuration)
# Returns: Health tier via stdout (HEALTHY|DEGRADED|UNHEALTHY|UNKNOWN)
#######################################
status::get_health_tier() {
    local status_config="$1"
    
    local resource_name
    local container_name
    
    resource_name=$(echo "$status_config" | jq -r '.resource.name // empty')
    container_name=$(echo "$status_config" | jq -r '.resource.container_name // empty')
    
    # Try resource-specific tiered health function first
    if status::has_tiered_health "$resource_name"; then
        local tier_result
        tier_result=$("${resource_name}::tiered_health_check" 2>/dev/null || echo "UNKNOWN")
        echo "$tier_result"
        return 0
    fi
    
    # Fallback to basic health checking
    if [[ -n "$container_name" ]]; then
        if ! docker::is_running "$container_name"; then
            echo "UNHEALTHY"
            return 0
        fi
        
        # Check if container health check exists
        case $(docker::check_health "$container_name") in
            0)
                echo "HEALTHY"
                ;;
            1)
                echo "UNHEALTHY"
                ;;
            *)
                # No health check defined, try HTTP endpoint if available
                local health_endpoint
                health_endpoint=$(echo "$status_config" | jq -r '.endpoints.health // empty')
                if [[ -n "$health_endpoint" ]]; then
                    if http::request "GET" "$health_endpoint" "" "" >/dev/null 2>&1; then
                        echo "HEALTHY"
                    else
                        echo "UNHEALTHY"
                    fi
                else
                    # Running but no health check available
                    echo "UNKNOWN"
                fi
                ;;
        esac
    else
        echo "UNKNOWN"
    fi
}

#######################################
# Internal: Validate status configuration
# Args: $1 - status_config (JSON)
# Returns: 0 if valid, 1 if invalid
#######################################
status::_validate_config() {
    local config="$1"
    
    # Check if it's valid JSON
    if ! echo "$config" | jq empty 2>/dev/null; then
        return 1
    fi
    
    # Check required fields
    local required_fields=(".resource.name" ".resource.category")
    for field in "${required_fields[@]}"; do
        if ! echo "$config" | jq -e "$field" >/dev/null 2>&1; then
            log::error "Missing required field in status config: $field"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Internal: Display resource header
# Args: $1 - resource_name, $2 - category, $3 - description
#######################################
status::_display_header() {
    local resource_name="$1"
    local category="$2"
    local description="$3"
    
    log::header "📊 ${resource_name^^} Status"
    if [[ -n "$description" ]]; then
        log::info "📝 Description: $description"
    fi
    log::info "📂 Category: $category"
}

#######################################
# Internal: Display basic status (installed, running, healthy)
# Args: $1 - status_config (JSON)
#######################################
status::_display_basic_status() {
    local status_config="$1"
    
    local container_name
    container_name=$(echo "$status_config" | jq -r '.resource.container_name // empty')
    
    log::info "📊 Basic Status:"
    
    # Installation status
    if status::_is_resource_installed "$container_name"; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        return 0
    fi
    
    # Running status
    if [[ -n "$container_name" ]] && docker::is_running "$container_name"; then
        log::success "   ✅ Running: Yes"
        
        # Health status
        local health_tier
        health_tier=$(status::get_health_tier "$status_config")
        case "$health_tier" in
            "HEALTHY")
                log::success "   ✅ Health: Healthy"
                ;;
            "DEGRADED")
                log::warn "   ⚠️  Health: Degraded"
                ;;
            "UNHEALTHY")
                log::error "   ❌ Health: Unhealthy"
                ;;
            *)
                log::info "   ❓ Health: Unknown"
                ;;
        esac
    else
        log::warn "   ⚠️  Running: No"
        log::warn "   ⚠️  Health: Cannot determine (not running)"
    fi
}

#######################################
# Internal: Display container information
# Args: $1 - status_config (JSON)
#######################################
status::_display_container_info() {
    local status_config="$1"
    
    local container_name
    container_name=$(echo "$status_config" | jq -r '.resource.container_name // empty')
    
    if [[ -z "$container_name" ]]; then
        return 0
    fi
    
    log::info "🐳 Container Info:"
    
    if docker::container_exists "$container_name"; then
        if docker::is_running "$container_name"; then
            # Get container stats
            local stats
            stats=$(docker stats "$container_name" --no-stream --format "{{.CPUPerc}}|{{.MemUsage}}" 2>/dev/null || echo "")
            
            if [[ -n "$stats" ]]; then
                local cpu_mem
                cpu_mem=$(echo "$stats" | tr '|' ' ')
                log::success "   ✅ Container: Running ($container_name) [$cpu_mem]"
            else
                log::success "   ✅ Container: Running ($container_name)"
            fi
            
            # Show image and creation time
            local image
            local created
            image=$(docker::get_image "$container_name")
            created=$(docker::get_created "$container_name")
            
            if [[ -n "$image" ]]; then
                log::info "   🖼️  Image: $image"
            fi
            
            if [[ -n "$created" ]]; then
                local created_date
                created_date=$(date -d "$created" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "$created")
                log::info "   📅 Created: $created_date"
            fi
        else
            log::error "   ❌ Container: Exists but not running ($container_name)"
        fi
    else
        log::error "   ❌ Container: Not found ($container_name)"
    fi
}

#######################################
# Internal: Display service details (endpoints)
# Args: $1 - status_config (JSON)  
#######################################
status::_display_service_details() {
    local status_config="$1"
    
    local ui_endpoint
    local api_endpoint
    local health_endpoint
    
    ui_endpoint=$(echo "$status_config" | jq -r '.endpoints.ui // empty')
    api_endpoint=$(echo "$status_config" | jq -r '.endpoints.api // empty')
    health_endpoint=$(echo "$status_config" | jq -r '.endpoints.health // empty')
    
    # Only show if at least one endpoint is defined
    if [[ -n "$ui_endpoint" ]] || [[ -n "$api_endpoint" ]] || [[ -n "$health_endpoint" ]]; then
        log::info "🌐 Service Endpoints:"
        
        if [[ -n "$ui_endpoint" ]]; then
            log::info "   🎨 UI: $ui_endpoint"
        fi
        
        if [[ -n "$api_endpoint" ]]; then
            log::info "   🔌 API: $api_endpoint"
        fi
        
        if [[ -n "$health_endpoint" ]]; then
            log::info "   🏥 Health: $health_endpoint"
        fi
    fi
}

#######################################
# Internal: Display authentication information
# Args: $1 - status_config (JSON)
#######################################
status::_display_authentication_info() {
    local status_config="$1"
    
    local auth_type
    local auth_status_func
    local container_name
    
    auth_type=$(echo "$status_config" | jq -r '.auth.type // "none"')
    auth_status_func=$(echo "$status_config" | jq -r '.auth.status_func // empty')
    container_name=$(echo "$status_config" | jq -r '.resource.container_name // empty')
    
    if [[ "$auth_type" == "none" ]]; then
        return 0
    fi
    
    log::info "🔐 Authentication:"
    
    case "$auth_type" in
        "basic")
            # Check basic auth status
            if [[ -n "$container_name" ]] && docker::is_running "$container_name"; then
                local auth_active
                auth_active=$(docker::extract_env "$container_name" "N8N_BASIC_AUTH_ACTIVE" 2>/dev/null || echo "false")
                if [[ "$auth_active" == "true" ]]; then
                    local auth_user
                    auth_user=$(docker::extract_env "$container_name" "N8N_BASIC_AUTH_USER" 2>/dev/null || echo "admin")
                    log::info "   ✅ Basic Auth: Enabled (user: $auth_user)"
                else
                    log::warn "   ⚠️  Basic Auth: Disabled"
                fi
            else
                log::warn "   ❓ Basic Auth: Cannot determine (container not running)"
            fi
            ;;
        "api-key")
            # Use custom status function if provided
            if [[ -n "$auth_status_func" ]] && command -v "$auth_status_func" &>/dev/null; then
                "$auth_status_func"
            else
                log::info "   🔑 API Key: Check implementation for status"
            fi
            ;;
        *)
            log::info "   ❓ Auth Type: $auth_type (check implementation for details)"
            ;;
    esac
}

#######################################
# Internal: Display health feedback with actionable advice
# Args: $1 - status_config (JSON)
#######################################
status::_display_health_feedback() {
    local status_config="$1"
    
    local health_tier
    health_tier=$(status::get_health_tier "$status_config")
    
    log::info "🏥 Health Assessment:"
    status::display_health_tier "$health_tier" "$status_config"
}

#######################################
# Internal: Display installation guidance
# Args: $1 - status_config (JSON)
#######################################
status::_display_installation_guidance() {
    local status_config="$1"
    
    local resource_name
    resource_name=$(echo "$status_config" | jq -r '.resource.name // empty')
    
    echo
    log::info "💡 Installation Required:"
    log::info "   To install $resource_name, run:"
    log::info "   resource-${resource_name} manage install"
    echo
    log::info "   For installation options, run:"
    log::info "   resource-${resource_name} help"
}

#######################################
# Internal: Check if resource is installed
# Args: $1 - container_name
# Returns: 0 if installed, 1 if not installed
#######################################
status::_is_resource_installed() {
    local container_name="$1"
    
    if [[ -z "$container_name" ]]; then
        return 1
    fi
    
    docker::container_exists "$container_name"
}
