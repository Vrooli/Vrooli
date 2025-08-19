#!/usr/bin/env bash
# Agent S2 Status Management - Standardized Format
# Functions for checking and displaying Agent S2 status information

# Source format utilities and config
AGENTS2_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${AGENTS2_STATUS_DIR}/../../../../lib/utils/format.sh"
# shellcheck disable=SC1091
source "${AGENTS2_STATUS_DIR}/../../../lib/status-args.sh"
# shellcheck disable=SC1091
source "${AGENTS2_STATUS_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${AGENTS2_STATUS_DIR}/common.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v agents2::export_config &>/dev/null; then
    agents2::export_config 2>/dev/null || true
fi

#######################################
# Collect Agent S2 status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
agent_s2::status::collect_data() {
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
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    if agents2::container_exists; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "${AGENTS2_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
        
        if agents2::is_running; then
            running="true"
            
            if agents2::is_healthy; then
                healthy="true"
                health_message="Healthy - AI automation agent ready"
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
    status_data+=("name" "agent-s2")
    status_data+=("category" "agents")
    status_data+=("description" "Autonomous computer interaction service with GUI automation")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$AGENTS2_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$AGENTS2_PORT")
    
    # Service endpoints
    status_data+=("api_url" "$AGENTS2_BASE_URL")
    status_data+=("vnc_url" "$AGENTS2_VNC_URL")
    status_data+=("api_docs_url" "${AGENTS2_BASE_URL}/docs")
    status_data+=("health_url" "${AGENTS2_BASE_URL}/health")
    
    # Configuration details
    status_data+=("image" "$AGENTS2_IMAGE_NAME")
    status_data+=("data_dir" "$AGENTS2_DATA_DIR")
    status_data+=("vnc_port" "$AGENTS2_VNC_PORT")
    status_data+=("vnc_password" "$AGENTS2_VNC_PASSWORD")
    status_data+=("llm_provider" "$AGENTS2_LLM_PROVIDER")
    status_data+=("llm_model" "$AGENTS2_LLM_MODEL")
    status_data+=("display" "${AGENTS2_DISPLAY:-:98}")
    status_data+=("screen_resolution" "$AGENTS2_SCREEN_RESOLUTION")
    status_data+=("enable_host_display" "$AGENTS2_ENABLE_HOST_DISPLAY")
    status_data+=("memory_limit" "$AGENTS2_MEMORY_LIMIT")
    status_data+=("cpu_limit" "$AGENTS2_CPU_LIMIT")
    status_data+=("shm_size" "$AGENTS2_SHM_SIZE")
    
    # Runtime information (only if running)
    if [[ "$running" == "true" ]]; then
        # Container stats (optimized with smart skipping)
        local stats cpu_usage memory_usage
        
        # Skip expensive operations in fast mode
        local skip_stats="$fast_mode"
        
        if [[ "$skip_stats" == "true" ]]; then
            cpu_usage="N/A"
            memory_usage="N/A"
        else
            stats=$(timeout 2s docker stats --no-stream --format "{{.CPUPerc}};{{.MemUsage}}" "$AGENTS2_CONTAINER_NAME" 2>/dev/null || echo "N/A;N/A")
            
            if [[ "$stats" != "N/A;N/A" ]]; then
                cpu_usage=$(echo "$stats" | cut -d';' -f1)
                memory_usage=$(echo "$stats" | cut -d';' -f2)
            else
                cpu_usage="N/A"
                memory_usage="N/A"
            fi
        fi
        
        status_data+=("cpu_usage" "$cpu_usage")
        status_data+=("memory_usage" "$memory_usage")
        
        # Get detailed health info if available (optimized)
        if [[ "$healthy" == "true" ]] && [[ "$skip_stats" == "false" ]]; then
            local health_info
            health_info=$(timeout 2s curl -s --max-time 2 "${AGENTS2_BASE_URL}/health" 2>/dev/null)
            if [[ -n "$health_info" ]] && echo "$health_info" | jq . >/dev/null 2>&1; then
                local api_status
                api_status=$(echo "$health_info" | jq -r '.status // "unknown"' 2>/dev/null)
                status_data+=("api_health_status" "$api_status")
            fi
        elif [[ "$healthy" == "true" ]]; then
            status_data+=("api_health_status" "N/A")
        fi
        
        # Storage usage (optimized)
        if [[ -d "$AGENTS2_DATA_DIR" ]]; then
            if [[ "$skip_stats" == "false" ]]; then
                local storage_size
                storage_size=$(timeout 2s du -sh "$AGENTS2_DATA_DIR" 2>/dev/null | awk '{print $1}' || echo "N/A")
                status_data+=("storage_size" "$storage_size")
            else
                status_data+=("storage_size" "N/A")
            fi
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show Agent S2 status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
agent_s2::status::show() {
    local format="text"
    local verbose="false"
    local fast="false"
    
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
            --fast)
                fast="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Collect status data (pass fast flag if set)
    local data_string
    local collect_args=""
    if [[ "$fast" == "true" ]]; then
        collect_args="--fast"
    fi
    data_string=$(agent_s2::status::collect_data $collect_args 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect Agent S2 status data"
        fi
        return 1
    fi
    
    # Convert string to array
    local data_array
    mapfile -t data_array <<< "$data_string"
    
    # Output based on format
    if [[ "$format" == "json" ]]; then
        format::output "json" "kv" "${data_array[@]}"
    else
        # Text format with standardized structure
        agent_s2::status::display_text "${data_array[@]}"
    fi
    
    # Return appropriate exit code
    local healthy="false"
    local running="false"
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        case "${data_array[i]}" in
            "healthy") healthy="${data_array[i+1]}" ;;
            "running") running="${data_array[i+1]}" ;;
        esac
    done
    
    if [[ "$healthy" == "true" ]]; then
        return 0
    elif [[ "$running" == "true" ]]; then
        return 1
    else
        return 2
    fi
}

#######################################
# Display status in text format
#######################################
agent_s2::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🤖 Agent S2 Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Agent S2, run: ./manage.sh --action install"
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
    log::info "   🔌 API URL: ${data[api_url]:-unknown}"
    log::info "   🖥️  VNC URL: ${data[vnc_url]:-unknown}"
    log::info "   📚 API Docs: ${data[api_docs_url]:-unknown}"
    log::info "   💓 Health Check: ${data[health_url]:-unknown}"
    echo
    
    # VNC Configuration
    log::info "🖥️  VNC Configuration:"
    log::info "   📶 Port: ${data[vnc_port]:-unknown}"
    log::info "   🔑 Password: ${data[vnc_password]:-unknown}"
    log::info "   📺 Display: ${data[display]:-unknown}"
    log::info "   📐 Resolution: ${data[screen_resolution]:-unknown}"
    log::info "   🏠 Host Display Access: ${data[enable_host_display]:-unknown}"
    echo
    
    # LLM Configuration
    log::info "🧠 LLM Configuration:"
    log::info "   🏢 Provider: ${data[llm_provider]:-unknown}"
    log::info "   🤖 Model: ${data[llm_model]:-unknown}"
    echo
    
    # Resource Configuration
    log::info "⚙️  Resource Configuration:"
    log::info "   💾 Memory Limit: ${data[memory_limit]:-unknown}"
    log::info "   🖥️  CPU Limit: ${data[cpu_limit]:-unknown}"
    log::info "   🧠 Shared Memory: ${data[shm_size]:-unknown}"
    echo
    
    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        if [[ -n "${data[cpu_usage]:-}" ]]; then
            log::info "   🖥️  CPU Usage: ${data[cpu_usage]}"
        fi
        if [[ -n "${data[memory_usage]:-}" ]]; then
            log::info "   🧠 Memory Usage: ${data[memory_usage]}"
        fi
        if [[ -n "${data[storage_size]:-}" ]]; then
            log::info "   💾 Storage Size: ${data[storage_size]}"
        fi
        if [[ -n "${data[api_health_status]:-}" ]]; then
            log::info "   💓 API Status: ${data[api_health_status]}"
        fi
        echo
        
        # Quick access info
        log::info "🎯 Quick Actions:"
        log::info "   🌐 Access API: ${data[api_url]:-http://localhost:4113}"
        log::info "   🖥️  Connect VNC: vncviewer localhost:${data[vnc_port]:-5900}"
        log::info "   📄 View logs: ./manage.sh --action logs"
        log::info "   🛑 Stop service: ./manage.sh --action stop"
    fi
}

#######################################
# Main status function for CLI registration
#######################################
agent_s2::status() {
    status::run_standard "agent-s2" "agent_s2::status::collect_data" "agent_s2::status::display_text" "$@"
}

# CLI framework expects hyphenated function name
agent-s2::status() {
    status::run_standard "agent-s2" "agent_s2::status::collect_data" "agent_s2::status::display_text" "$@"
}

#######################################
# Legacy function compatibility
#######################################
agents2::show_status() {
    agent_s2::status::show "$@"
}

agents2::show_info() {
    cat << EOF
=== Agent S2 Resource Information ===

ID: agent-s2
Category: agents
Display Name: Agent S2
Description: Autonomous computer interaction service with GUI automation

Service Details:
- Container Name: $AGENTS2_CONTAINER_NAME
- Image Name: $AGENTS2_IMAGE_NAME
- API Port: $AGENTS2_PORT
- VNC Port: $AGENTS2_VNC_PORT
- API URL: $AGENTS2_BASE_URL
- VNC URL: $AGENTS2_VNC_URL
- Data Directory: $AGENTS2_DATA_DIR

Display Configuration:
- Virtual Display: $AGENTS2_DISPLAY
- Screen Resolution: $AGENTS2_SCREEN_RESOLUTION
- VNC Access: Enabled (password protected)
- Host Display Access: $AGENTS2_ENABLE_HOST_DISPLAY

LLM Configuration:
- Provider: $([[ -z "${AGENTS2_OPENAI_API_KEY}" && -z "${AGENTS2_ANTHROPIC_API_KEY}" ]] && echo "ollama (auto-detected)" || echo "$AGENTS2_LLM_PROVIDER")
- Model: $AGENTS2_LLM_MODEL
- Ollama URL: $AGENTS2_OLLAMA_BASE_URL
- API Keys: $([[ -n "${AGENTS2_OPENAI_API_KEY}" || -n "${AGENTS2_ANTHROPIC_API_KEY}" ]] && echo "Configured" || echo "None (using Ollama)")

Resource Limits:
- Memory: $AGENTS2_MEMORY_LIMIT
- CPUs: $AGENTS2_CPU_LIMIT
- Shared Memory: $AGENTS2_SHM_SIZE

Capabilities:
- Screenshot capture
- Mouse control (click, move, drag)
- Keyboard input (type, key press)
- GUI automation sequences
- Multi-step task planning
- Cross-platform support

Security Features:
- Runs as non-root user
- Isolated virtual display
- Sandboxed environment
- Restricted application access
- VNC password protection

API Endpoints:
- GET  /health          - Health check
- GET  /capabilities    - List capabilities
- POST /screenshot      - Capture screenshot
- POST /execute         - Execute automation task
- GET  /tasks/{id}      - Get task status
- GET  /mouse/position  - Get mouse position

Management Commands:
$0 --action status      # Check status
$0 --action logs        # View logs
$0 --action usage       # Run examples
$0 --action restart     # Restart service

For more information: https://www.simular.ai/agent-s
EOF
}

#######################################
# Show access information (legacy compatibility)
#######################################
agents2::show_access_info() {
    log::info "Access Information:"
    log::info "  API URL: $AGENTS2_BASE_URL"
    log::info "  VNC URL: $AGENTS2_VNC_URL"
    log::info "  API Docs: ${AGENTS2_BASE_URL}/docs"
    log::info "  Container: $AGENTS2_CONTAINER_NAME"
    
    # Show VNC connection command
    echo
    log::info "Connect via VNC:"
    log::info "  Using viewer: vncviewer localhost:$AGENTS2_VNC_PORT"
    log::info "  Using browser: Open VNC client and connect to localhost:$AGENTS2_VNC_PORT"
    log::info "  Password: $AGENTS2_VNC_PASSWORD"
}

#######################################
# Check port availability (legacy compatibility)
#######################################
agents2::check_ports() {
    log::info "Port Status:"
    
    # Check API port
    if command -v resources::is_service_running >/dev/null 2>&1; then
        if resources::is_service_running "$AGENTS2_PORT"; then
            log::success "Port $AGENTS2_PORT (API) is accessible"
        else
            log::warn "Port $AGENTS2_PORT (API) is not accessible"
        fi
        
        # Check VNC port
        if resources::is_service_running "$AGENTS2_VNC_PORT"; then
            log::success "Port $AGENTS2_VNC_PORT (VNC) is accessible"
        else
            log::warn "Port $AGENTS2_VNC_PORT (VNC) is not accessible"
        fi
    else
        log::info "  API Port: $AGENTS2_PORT"
        log::info "  VNC Port: $AGENTS2_VNC_PORT"
    fi
}

#######################################
# Get quick status (for scripts) - legacy compatibility
# Returns: JSON status object
#######################################
agents2::get_status_json() {
    local status="not_installed"
    local health="unknown"
    local api_responsive=false
    local vnc_accessible=false
    
    if agents2::container_exists; then
        if agents2::is_running; then
            status="running"
            health=$(agents2::get_health_status)
            
            # Check API
            if agents2::is_healthy; then
                api_responsive=true
            fi
            
            # Check VNC port (simplified check)
            local vnc_check
            vnc_check=$(ss -tln 2>/dev/null | grep ":${AGENTS2_VNC_PORT} " || netstat -tln 2>/dev/null | grep ":${AGENTS2_VNC_PORT} " || echo "")
            if [[ -n "$vnc_check" ]]; then
                vnc_accessible=true
            fi
        else
            status="stopped"
        fi
    fi
    
    cat << EOF
{
    "status": "$status",
    "health": "$health",
    "api_responsive": $api_responsive,
    "vnc_accessible": $vnc_accessible,
    "ports": {
        "api": $AGENTS2_PORT,
        "vnc": $AGENTS2_VNC_PORT
    },
    "urls": {
        "api": "$AGENTS2_BASE_URL",
        "vnc": "$AGENTS2_VNC_URL"
    }
}
EOF
}

# Export functions for subshell availability
export -f agent_s2::status::collect_data
export -f agent_s2::status::show
export -f agent_s2::status::display_text
export -f agent_s2::status
export -f agents2::show_status
export -f agents2::show_access_info
export -f agents2::check_ports
export -f agents2::show_info
export -f agents2::get_status_json