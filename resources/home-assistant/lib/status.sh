#!/bin/bash
# Home Assistant Status Functions

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
HOME_ASSISTANT_STATUS_DIR="${APP_ROOT}/resources/home-assistant/lib"

# Source dependencies
source "${HOME_ASSISTANT_STATUS_DIR}/core.sh"
source "${HOME_ASSISTANT_STATUS_DIR}/health.sh"
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-args.sh"

#######################################
# Collect Home Assistant status data
# Arguments:
#   --fast: Skip expensive operations
# Returns: Key-value pairs (one per line)
#######################################
home_assistant::status::collect_data() {
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
    
    # Initialize
    home_assistant::init >/dev/null 2>&1
    
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "$HOME_ASSISTANT_CONTAINER_NAME" 2>/dev/null || echo "unknown")
        
        if docker::is_running "$HOME_ASSISTANT_CONTAINER_NAME"; then
            running="true"
            
            if home_assistant::health::is_healthy; then
                healthy="true"
                health_message="Healthy - Home automation platform is ready"
            else
                health_message="Unhealthy - Service not responding"
            fi
        else
            health_message="Stopped - Container not running"
        fi
    else
        health_message="Not installed - Container does not exist"
    fi
    
    # Basic resource information
    status_data+=("name" "home-assistant")
    status_data+=("category" "execution")
    status_data+=("description" "Open source home automation platform")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$HOME_ASSISTANT_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$HOME_ASSISTANT_PORT")
    
    # Service endpoints
    status_data+=("web_url" "$HOME_ASSISTANT_BASE_URL")
    status_data+=("api_url" "$HOME_ASSISTANT_BASE_URL/api/")
    status_data+=("auth_url" "$HOME_ASSISTANT_BASE_URL/auth/token")
    
    # Configuration details
    status_data+=("image" "$HOME_ASSISTANT_IMAGE")
    status_data+=("config_dir" "$HOME_ASSISTANT_CONFIG_DIR")
    status_data+=("timezone" "$HOME_ASSISTANT_TIME_ZONE")
    
    # Runtime information (only if running and not in fast mode)
    if [[ "$running" == "true" && "$fast_mode" == "false" ]]; then
        # Get memory and CPU usage
        local stats_output memory_usage cpu_usage
        stats_output=$(timeout 2s docker stats "$HOME_ASSISTANT_CONTAINER_NAME" --no-stream --format "{{.MemUsage}}\t{{.CPUPerc}}" 2>/dev/null || echo "N/A\tN/A")
        
        if [[ "$stats_output" != "N/A"*"N/A" ]]; then
            memory_usage=$(echo "$stats_output" | cut -f1 | cut -d' ' -f1)
            cpu_usage=$(echo "$stats_output" | cut -f2)
        else
            memory_usage="N/A"
            cpu_usage="N/A"
        fi
        
        status_data+=("memory_usage" "${memory_usage:-N/A}")
        status_data+=("cpu_usage" "${cpu_usage:-N/A}")
        
        # Calculate uptime
        local created_time
        created_time=$(docker inspect --format='{{.State.StartedAt}}' "$HOME_ASSISTANT_CONTAINER_NAME" 2>/dev/null)
        
        if [[ -n "$created_time" ]]; then
            local uptime
            uptime=$(docker::calculate_uptime "$created_time")
            status_data+=("uptime" "${uptime:-N/A}")
        else
            status_data+=("uptime" "N/A")
        fi
        
        # Get version info (if available)
        local version
        version=$(docker exec "$HOME_ASSISTANT_CONTAINER_NAME" python3 -c "import homeassistant; print(homeassistant.__version__)" 2>/dev/null || echo "unknown")
        status_data+=("version" "$version")
    else
        status_data+=("memory_usage" "N/A")
        status_data+=("cpu_usage" "N/A")
        status_data+=("uptime" "N/A")
        status_data+=("version" "N/A")
    fi
    
    # Output data as key-value pairs
    local i
    for ((i=0; i<${#status_data[@]}; i+=2)); do
        echo "${status_data[i]}"
        echo "${status_data[i+1]}"
    done
}

#######################################
# Display Home Assistant status in text format
# Arguments:
#   data_array: Array of key-value pairs
#######################################
home_assistant::status::display_text() {
    local -a data_array=("$@")
    
    # Convert array to associative array for easier access
    local -A data
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        data["${data_array[i]}"]="${data_array[i+1]}"
    done
    
    # Display formatted status
    log::header "🏠 Home Assistant Status"
    
    log::info "📊 Basic Status:"
    if [[ "${data[installed]}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
    fi
    
    if [[ "${data[running]}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::error "   ❌ Running: No"
    fi
    
    if [[ "${data[healthy]}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::error "   ❌ Health: Unhealthy"
    fi
    
    if [[ "${data[installed]}" == "true" ]]; then
        log::info "🐳 Container Info:"
        log::info "   📦 Name: ${data[container_name]}"
        log::info "   📊 Status: ${data[container_status]}"
        log::info "   🖼️  Image: ${data[image]}"
        
        if [[ "${data[memory_usage]}" != "N/A" ]]; then
            log::info "   💾 Memory: ${data[memory_usage]}"
        fi
        
        if [[ "${data[cpu_usage]}" != "N/A" ]]; then
            log::info "   🔥 CPU: ${data[cpu_usage]}"
        fi
        
        if [[ "${data[uptime]}" != "N/A" ]]; then
            log::info "   ⏱️  Uptime: ${data[uptime]}"
        fi
        
        if [[ "${data[version]}" != "N/A" && "${data[version]}" != "unknown" ]]; then
            log::info "   🔖 Version: ${data[version]}"
        fi
    fi
    
    log::info "🌐 Service Endpoints:"
    log::info "   🏠 Web UI: ${data[web_url]}"
    log::info "   🔌 API: ${data[api_url]}"
    log::info "   🔐 Auth: ${data[auth_url]}"
    
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]}"
    log::info "   📁 Config Directory: ${data[config_dir]}"
    log::info "   🕐 Timezone: ${data[timezone]}"
    
    log::info "📋 Status Message:"
    log::info "   ${data[health_message]}"
}

#######################################
# Check Home Assistant status
# Arguments:
#   --format: Output format (text/json)
#   --fast: Skip expensive operations
# Returns: 0 if healthy, 1 otherwise
#######################################
home_assistant::status() {
    status::run_standard "home_assistant" "home_assistant::status::collect_data" "home_assistant::status::display_text" "$@"
}

# Export function
export -f home_assistant::status