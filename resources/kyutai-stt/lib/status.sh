#!/usr/bin/env bash
# Kyutai STT Status Management - Standardized Format
# Functions for checking and displaying Kyutai STT status information

# Source format utilities and config
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh"
KYUTAI_STT_STATUS_DIR="${var_RESOURCES_DIR}/kyutai-stt/lib"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/resources/lib/status-args.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_DIR}/kyutai-stt/config/defaults.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_DIR}/kyutai-stt/config/messages.sh"
# shellcheck disable=SC1091
source "${KYUTAI_STT_STATUS_DIR}/common.sh"

# Ensure configuration is exported
if command -v defaults::export_config &>/dev/null; then
    defaults::export_config 2>/dev/null || true
fi

#######################################
# Collect Kyutai STT status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
kyutai_stt::status::collect_data() {
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

    if common::container_exists; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "$KYUTAI_STT_CONTAINER_NAME" 2>/dev/null || echo "unknown")

        if common::is_running; then
            running="true"

            if kyutai_stt::is_healthy; then
                healthy="true"
                if kyutai_stt::model_loaded; then
                    health_message="Healthy - streaming STT model loaded"
                else
                    health_message="Healthy - server up, model still loading"
                fi
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
    status_data+=("name" "kyutai-stt")
    status_data+=("category" "ai")
    status_data+=("description" "Kyutai streaming speech-to-text service")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$KYUTAI_STT_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$KYUTAI_STT_PORT")

    # Service endpoints
    status_data+=("base_url" "$KYUTAI_STT_BASE_URL")
    status_data+=("health_url" "$KYUTAI_STT_BASE_URL/health")
    status_data+=("info_url" "$KYUTAI_STT_BASE_URL/v1/info")
    status_data+=("stream_url" "$KYUTAI_STT_WS_URL")

    # Configuration details
    status_data+=("image" "$KYUTAI_STT_IMAGE")
    status_data+=("model" "$KYUTAI_STT_HF_REPO")
    status_data+=("device" "$KYUTAI_STT_DEVICE")
    status_data+=("data_dir" "$KYUTAI_STT_DATA_DIR")
    status_data+=("models_dir" "$KYUTAI_STT_MODELS_DIR")
    status_data+=("gpu_enabled" "$KYUTAI_STT_GPU_ENABLED")

    # Runtime information (only if running)
    if [[ "$running" == "true" ]]; then
        # GPU availability
        if [[ "$KYUTAI_STT_GPU_ENABLED" == "yes" ]]; then
            local gpu_available="false"
            if kyutai_stt::is_gpu_available; then
                gpu_available="true"
                local gpu_info
                gpu_info=$(system::host_inventory_field "first_gpu_summary" 2>/dev/null || echo "unknown")
                status_data+=("gpu_info" "${gpu_info:-unknown}")
            fi
            status_data+=("gpu_available" "$gpu_available")
        fi

        # Storage information (skip expensive operations in fast mode)
        if [[ -d "$KYUTAI_STT_DATA_DIR" ]]; then
            local data_size
            if [[ "$fast_mode" == "true" ]]; then
                data_size="N/A"
            else
                data_size=$(du -sh "$KYUTAI_STT_DATA_DIR" 2>/dev/null | cut -f1 || echo "unknown")
            fi
            status_data+=("data_size" "$data_size")
        fi

        # Container resource usage (skip in fast mode)
        local stats
        if [[ "$fast_mode" == "true" ]]; then
            stats="N/A;N/A"
        else
            stats=$(timeout 2s docker stats --no-stream --format "{{.CPUPerc}};{{.MemUsage}}" "$KYUTAI_STT_CONTAINER_NAME" 2>/dev/null || echo "N/A;N/A")
        fi

        local cpu_usage memory_usage
        if [[ "$stats" != "N/A;N/A" ]]; then
            cpu_usage=$(echo "$stats" | cut -d';' -f1)
            memory_usage=$(echo "$stats" | cut -d';' -f2)
        else
            cpu_usage="N/A"
            memory_usage="N/A"
        fi
        status_data+=("cpu_usage" "$cpu_usage")
        status_data+=("memory_usage" "$memory_usage")
    fi

    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show Kyutai STT status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
kyutai_stt::status::show() {
    local format="text"
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
    data_string=$(kyutai_stt::status::collect_data $collect_args 2>/dev/null)

    if [[ -z "$data_string" ]]; then
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect Kyutai STT status data"
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
        kyutai_stt::status::display_text "${data_array[@]}"
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
kyutai_stt::status::display_text() {
    local -A data

    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done

    # Header
    log::header "🎤 Kyutai STT Status"
    echo

    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Kyutai STT, run: resource-kyutai-stt manage install"
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
    log::info "   🔗 Base URL: ${data[base_url]:-unknown}"
    log::info "   🏥 Health: ${data[health_url]:-unknown}"
    log::info "   ℹ️  Info: ${data[info_url]:-unknown}"
    log::info "   🎙️  Stream (WS): ${data[stream_url]:-unknown}"
    echo

    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   🧠 Model: ${data[model]:-unknown}"
    log::info "   🖥️  Device: ${data[device]:-unknown}"
    log::info "   📁 Data Directory: ${data[data_dir]:-unknown}"
    log::info "   🎮 GPU Enabled: ${data[gpu_enabled]:-unknown}"
    echo

    # GPU information
    if [[ "${data[gpu_enabled]:-}" == "yes" ]]; then
        log::info "🎮 GPU Status:"
        if [[ "${data[gpu_available]:-false}" == "true" ]]; then
            log::success "   ✅ GPU Available: Yes"
            if [[ -n "${data[gpu_info]:-}" && "${data[gpu_info]}" != "unknown" ]]; then
                log::info "   📊 GPU Info: ${data[gpu_info]}"
            fi
        else
            log::warn "   ⚠️  GPU Available: No (Kyutai STT requires CUDA)"
        fi
        echo
    fi

    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        if [[ -n "${data[cpu_usage]:-}" ]]; then
            log::info "   💻 CPU Usage: ${data[cpu_usage]}"
        fi
        if [[ -n "${data[memory_usage]:-}" ]]; then
            log::info "   🧠 Memory Usage: ${data[memory_usage]}"
        fi
        if [[ -n "${data[data_size]:-}" ]]; then
            log::info "   💾 Data Size: ${data[data_size]}"
        fi
        echo

        # Quick access info
        log::info "🎯 Quick Actions:"
        log::info "   ℹ️  Server info: resource-kyutai-stt content info"
        log::info "   📄 View logs: resource-kyutai-stt logs"
        log::info "   🛑 Stop service: resource-kyutai-stt manage stop"
    fi
}

#######################################
# Main status function for CLI registration
#######################################
kyutai_stt::status() {
    status::run_standard "kyutai-stt" "kyutai_stt::status::collect_data" "kyutai_stt::status::display_text" "$@"
}

# Legacy compatibility

#######################################
# Show comprehensive Kyutai STT status
# Returns: 0 if successful, 1 otherwise
#######################################
status::show_status() {
    kyutai_stt::status::show "$@"
}

#######################################
# Show quick status (for use in other scripts)
# Outputs: status string
#######################################
status::quick_status() {
    if ! common::is_running; then
        echo "stopped"
        return 1
    fi

    if kyutai_stt::is_healthy; then
        echo "running"
        return 0
    else
        echo "unhealthy"
        return 1
    fi
}

#######################################
# Check if Kyutai STT is ready for transcription
# Returns: 0 if ready, 1 otherwise
#######################################
status::is_ready() {
    if ! common::is_running; then
        return 1
    fi

    if ! kyutai_stt::is_healthy; then
        return 1
    fi

    # Additional check: ensure the model finished loading
    local attempts=0
    local max_attempts=5

    while [[ $attempts -lt $max_attempts ]]; do
        if kyutai_stt::model_loaded; then
            return 0
        fi
        ((attempts++))
        sleep 2
    done

    return 1
}

#######################################
# Show resource usage
# Returns: 0 if successful, 1 otherwise
#######################################
status::show_resource_usage() {
    if ! common::is_running; then
        log::error "${MSG_NOT_RUNNING}"
        return 1
    fi

    echo "Resource Usage:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$KYUTAI_STT_CONTAINER_NAME"
}

# Export functions for subshell availability
export -f kyutai_stt::status::collect_data
export -f kyutai_stt::status::show
export -f kyutai_stt::status::display_text
export -f kyutai_stt::status
export -f status::show_status
export -f status::quick_status
export -f status::is_ready
export -f status::show_resource_usage
