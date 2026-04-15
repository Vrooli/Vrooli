#!/usr/bin/env bash
# Kokoro Status Management - Standardized Format
# Functions for checking and displaying Kokoro status information

# Source format utilities and config
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh"
KOKORO_STATUS_DIR="${var_RESOURCES_DIR}/kokoro/lib"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/resources/lib/status-args.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_DIR}/kokoro/config/defaults.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_DIR}/kokoro/config/messages.sh"
# shellcheck disable=SC1091
source "${KOKORO_STATUS_DIR}/common.sh"

# Ensure configuration is exported
if command -v defaults::export_config &>/dev/null; then
    defaults::export_config 2>/dev/null || true
fi

#######################################
# Collect Kokoro status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
kokoro::status::collect_data() {
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
        container_status=$(docker inspect --format='{{.State.Status}}' "$KOKORO_CONTAINER_NAME" 2>/dev/null || echo "unknown")

        if common::is_running; then
            running="true"

            if kokoro::is_healthy; then
                healthy="true"
                health_message="Healthy - TTS synthesis service ready"
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
    status_data+=("name" "kokoro")
    status_data+=("category" "ai")
    status_data+=("description" "Kokoro text-to-speech synthesis service (82M model)")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$KOKORO_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$KOKORO_PORT")

    # Service endpoints
    status_data+=("base_url" "$KOKORO_BASE_URL")
    status_data+=("api_url" "$KOKORO_BASE_URL/v1/audio/speech")
    status_data+=("voices_url" "$KOKORO_BASE_URL/v1/audio/voices")

    # Configuration details
    local image
    image=$(kokoro::get_docker_image)
    status_data+=("image" "$image")
    status_data+=("default_voice" "$KOKORO_DEFAULT_VOICE")
    status_data+=("data_dir" "$KOKORO_DATA_DIR")
    status_data+=("voices_dir" "$KOKORO_VOICES_DIR")
    status_data+=("gpu_enabled" "$KOKORO_GPU_ENABLED")

    # Runtime information (only if running and healthy)
    if [[ "$running" == "true" ]]; then
        # GPU availability
        if [[ "$KOKORO_GPU_ENABLED" == "yes" ]]; then
            local gpu_available="false"
            if kokoro::is_gpu_available; then
                gpu_available="true"
                if command -v nvidia-smi &>/dev/null; then
                    local gpu_info
                    gpu_info=$(nvidia-smi --query-gpu=name,memory.used,memory.total --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "unknown")
                    status_data+=("gpu_info" "$gpu_info")
                fi
            fi
            status_data+=("gpu_available" "$gpu_available")
        fi

        # Voice count (if healthy)
        if [[ "$healthy" == "true" ]]; then
            local voice_count
            if [[ "$fast_mode" == "true" ]]; then
                voice_count="N/A"
            else
                voice_count=$(curl -s "$KOKORO_BASE_URL/v1/audio/voices" --max-time "$KOKORO_API_TIMEOUT" 2>/dev/null | jq 'length' 2>/dev/null || echo "unknown")
            fi
            status_data+=("voice_count" "$voice_count")
        fi

        # Storage information (skip expensive operations in fast mode)
        if [[ -d "$KOKORO_DATA_DIR" ]]; then
            local data_size

            if [[ "$fast_mode" == "true" ]]; then
                data_size="N/A"
            else
                data_size=$(du -sh "$KOKORO_DATA_DIR" 2>/dev/null | cut -f1 || echo "unknown")
            fi

            status_data+=("data_size" "$data_size")
        fi

        # Container resource usage (skip in fast mode)
        local stats
        if [[ "$fast_mode" == "true" ]]; then
            stats="N/A;N/A"
        else
            stats=$(timeout 2s docker stats --no-stream --format "{{.CPUPerc}};{{.MemUsage}}" "$KOKORO_CONTAINER_NAME" 2>/dev/null || echo "N/A;N/A")
        fi

        if [[ "$stats" != "N/A;N/A" ]]; then
            local cpu_usage memory_usage
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
# Show Kokoro status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
kokoro::status::show() {
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
    data_string=$(kokoro::status::collect_data $collect_args 2>/dev/null)

    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect Kokoro status data"
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
        kokoro::status::display_text "${data_array[@]}"
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
kokoro::status::display_text() {
    local -A data

    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done

    # Header
    log::header "🔊 Kokoro Status"
    echo

    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Kokoro, run: resource-kokoro manage install"
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
    log::info "   🔊 Speech API: ${data[api_url]:-unknown}"
    log::info "   🎭 Voices: ${data[voices_url]:-unknown}"
    echo

    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   🎭 Default Voice: ${data[default_voice]:-unknown}"
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
            log::warn "   ⚠️  GPU Available: No (will use CPU)"
        fi
        echo
    fi

    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        if [[ -n "${data[voice_count]:-}" ]]; then
            log::info "   🎭 Available Voices: ${data[voice_count]}"
        fi

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
        log::info "   🔊 Synthesize text: resource-kokoro content synthesize --text 'Hello world'"
        log::info "   🎭 List voices: resource-kokoro content voices"
        log::info "   📄 View logs: resource-kokoro logs"
        log::info "   🛑 Stop service: resource-kokoro manage stop"
    fi
}

#######################################
# Main status function for CLI registration
#######################################
kokoro::status() {
    status::run_standard "kokoro" "kokoro::status::collect_data" "kokoro::status::display_text" "$@"
}

# Legacy compatibility

#######################################
# Show comprehensive Kokoro status
# Returns: 0 if successful, 1 otherwise
#######################################
status::show_status() {
    kokoro::status::show "$@"
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

    if kokoro::is_healthy; then
        echo "running"
        return 0
    else
        echo "unhealthy"
        return 1
    fi
}

#######################################
# Check if Kokoro is ready for synthesis
# Returns: 0 if ready, 1 otherwise
#######################################
status::is_ready() {
    if ! common::is_running; then
        return 1
    fi

    if ! kokoro::is_healthy; then
        return 1
    fi

    # Additional check: ensure voices endpoint responds
    local attempts=0
    local max_attempts=5

    while [[ $attempts -lt $max_attempts ]]; do
        local response
        response=$(curl -s -o /dev/null -w "%{http_code}" \
            --max-time "$KOKORO_API_TIMEOUT" \
            "$KOKORO_BASE_URL/v1/audio/voices" 2>/dev/null)

        if [[ "$response" == "200" ]]; then
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
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$KOKORO_CONTAINER_NAME"
}

# Export functions for subshell availability
export -f kokoro::status::collect_data
export -f kokoro::status::show
export -f kokoro::status::display_text
export -f kokoro::status
export -f status::show_status
export -f status::quick_status
export -f status::is_ready
export -f status::show_resource_usage
