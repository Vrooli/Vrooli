#!/usr/bin/env bash
# ComfyUI Status Management - Standardized Format
# Functions for checking and displaying ComfyUI status information

# Source format utilities and config
COMFYUI_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${COMFYUI_STATUS_DIR}/../../../../lib/utils/format.sh"
# shellcheck disable=SC1091
source "${COMFYUI_STATUS_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${COMFYUI_STATUS_DIR}/common.sh" 2>/dev/null || true

#######################################
# Collect ComfyUI status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
comfyui::status::collect_data() {
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    if common::container_exists; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "${COMFYUI_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
        
        if common::is_running; then
            running="true"
            
            if common::is_healthy; then
                healthy="true"
                health_message="Healthy - AI image generation ready"
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
    status_data+=("name" "comfyui")
    status_data+=("category" "automation")
    status_data+=("description" "AI-powered image generation and workflow automation")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$COMFYUI_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$COMFYUI_DIRECT_PORT")
    
    # Service endpoints
    status_data+=("ui_url" "http://localhost:${COMFYUI_DIRECT_PORT}")
    status_data+=("api_url" "http://localhost:${COMFYUI_DIRECT_PORT}/api")
    status_data+=("health_url" "http://localhost:${COMFYUI_DIRECT_PORT}/system_stats")
    
    # Configuration details
    local image="${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
    status_data+=("image" "$image")
    status_data+=("data_dir" "$COMFYUI_DATA_DIR")
    status_data+=("jupyter_port" "$COMFYUI_JUPYTER_PORT")
    
    # GPU configuration
    local gpu_type
    gpu_type=$(status::detect_gpu_silent 2>/dev/null || echo "cpu")
    status_data+=("gpu_type" "$gpu_type")
    
    # Runtime information (only if running)
    if [[ "$running" == "true" ]]; then
        # Container stats
        local stats
        stats=$(docker stats --no-stream --format "{{.CPUPerc}};{{.MemUsage}}" "$COMFYUI_CONTAINER_NAME" 2>/dev/null || echo "N/A;N/A")
        
        if [[ "$stats" != "N/A;N/A" ]]; then
            local cpu_usage memory_usage
            cpu_usage=$(echo "$stats" | cut -d';' -f1)
            memory_usage=$(echo "$stats" | cut -d';' -f2)
            status_data+=("cpu_usage" "$cpu_usage")
            status_data+=("memory_usage" "$memory_usage")
        fi
        
        # GPU status if available
        if [[ "$gpu_type" == "nvidia" ]] && common::is_running; then
            local gpu_accessible="false"
            if docker exec "$COMFYUI_CONTAINER_NAME" nvidia-smi >/dev/null 2>&1; then
                gpu_accessible="true"
                local gpu_info
                gpu_info=$(docker exec "$COMFYUI_CONTAINER_NAME" nvidia-smi --query-gpu=name,memory.used,memory.total --format=csv,noheader 2>/dev/null | head -1 || echo "unknown")
                status_data+=("gpu_info" "$gpu_info")
            fi
            status_data+=("gpu_accessible" "$gpu_accessible")
        fi
        
        # Storage usage
        if [[ -d "$COMFYUI_DATA_DIR" ]]; then
            local total_size
            total_size=$(du -sh "$COMFYUI_DATA_DIR" 2>/dev/null | awk '{print $1}' || echo "unknown")
            status_data+=("storage_size" "$total_size")
            
            # Model counts
            local models_count=0
            if [[ -d "${COMFYUI_DATA_DIR}/models" ]]; then
                models_count=$(find "${COMFYUI_DATA_DIR}/models" -type f \( -name "*.safetensors" -o -name "*.ckpt" -o -name "*.pth" \) 2>/dev/null | wc -l)
            fi
            status_data+=("models_count" "$models_count")
            
            # Workflow counts
            local workflows_count=0
            if [[ -d "${COMFYUI_DATA_DIR}/workflows" ]]; then
                workflows_count=$(find "${COMFYUI_DATA_DIR}/workflows" -name "*.json" 2>/dev/null | wc -l)
            fi
            status_data+=("workflows_count" "$workflows_count")
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show ComfyUI status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
comfyui::status::show() {
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
    
    # Collect status data
    local data_string
    data_string=$(comfyui::status::collect_data 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect ComfyUI status data"
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
        comfyui::status::display_text "${data_array[@]}"
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
comfyui::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📊 ComfyUI Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install ComfyUI, run: ./manage.sh --action install"
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
    if [[ -n "${data[jupyter_port]:-}" ]]; then
        log::info "   📓 Jupyter: http://localhost:${data[jupyter_port]}"
    fi
    echo
    
    # GPU Configuration
    log::info "🎮 GPU Configuration:"
    log::info "   🖥️  Type: ${data[gpu_type]:-unknown}"
    if [[ "${data[gpu_type]:-}" == "nvidia" ]]; then
        if [[ "${data[gpu_accessible]:-false}" == "true" ]]; then
            log::success "   ✅ GPU Accessible: Yes"
            if [[ -n "${data[gpu_info]:-}" && "${data[gpu_info]}" != "unknown" ]]; then
                log::info "   📊 GPU Info: ${data[gpu_info]}"
            fi
        else
            log::warn "   ⚠️  GPU Accessible: No (running in CPU mode)"
        fi
    elif [[ "${data[gpu_type]:-}" == "cpu" ]]; then
        log::warn "   ⚠️  Running in CPU mode - performance will be limited"
    fi
    echo
    
    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        if [[ -n "${data[cpu_usage]:-}" ]]; then
            log::info "   💾 CPU Usage: ${data[cpu_usage]}"
        fi
        if [[ -n "${data[memory_usage]:-}" ]]; then
            log::info "   🧠 Memory Usage: ${data[memory_usage]}"
        fi
        if [[ -n "${data[storage_size]:-}" ]]; then
            log::info "   💾 Storage Size: ${data[storage_size]}"
        fi
        if [[ -n "${data[models_count]:-}" ]]; then
            log::info "   🧠 AI Models: ${data[models_count]}"
        fi
        if [[ -n "${data[workflows_count]:-}" ]]; then
            log::info "   ⚡ Workflows: ${data[workflows_count]}"
        fi
        echo
        
        # Quick access info
        log::info "🎯 Quick Actions:"
        log::info "   🌐 Access ComfyUI: ${data[ui_url]:-http://localhost:8188}"
        log::info "   📄 View logs: ./manage.sh --action logs"
        log::info "   🛑 Stop service: ./manage.sh --action stop"
    fi
}

#######################################
# Legacy status function for backward compatibility
#######################################
status::status() {
    comfyui::status::show "$@"
}

#######################################
# Main status function for CLI registration
#######################################
comfyui::status() {
    comfyui::status::show "$@"
}

#######################################
# Detect GPU type (silent version for data collection)
#######################################
status::detect_gpu_silent() {
    if command -v nvidia-smi >/dev/null 2>&1; then
        echo "nvidia"
    elif command -v rocm-smi >/dev/null 2>&1; then
        echo "amd"
    else
        echo "cpu"
    fi
}

#######################################
# Additional legacy functions for backward compatibility
#######################################

status::show_model_integrity() {
    log::info "=== Model Integrity ==="
    
    local models_valid=0
    local models_invalid=0
    
    # Check default models
    for i in "${!COMFYUI_MODEL_NAMES[@]}"; do
        local model_name="${COMFYUI_MODEL_NAMES[$i]}"
        local expected_size="${COMFYUI_MODEL_SIZES[$i]}"
        
        # Determine model path
        local model_path
        if [[ "$model_name" == *"vae"* ]]; then
            model_path="${COMFYUI_DATA_DIR}/models/vae/$model_name"
        else
            model_path="${COMFYUI_DATA_DIR}/models/checkpoints/$model_name"
        fi
        
        if [[ -f "$model_path" ]]; then
            # Get actual size
            local actual_size
            actual_size=$(stat -c%s "$model_path" 2>/dev/null || stat -f%z "$model_path" 2>/dev/null)
            
            if [[ "$actual_size" == "$expected_size" ]]; then
                log::success "✅ $model_name - Valid ($(numfmt --to=iec-i --suffix=B "$actual_size" 2>/dev/null || echo "$actual_size bytes"))"
                models_valid=$((models_valid + 1))
            else
                log::error "❌ $model_name - Invalid size"
                echo "      Expected: $(numfmt --to=iec-i --suffix=B "$expected_size" 2>/dev/null || echo "$expected_size bytes")"
                echo "      Actual: $(numfmt --to=iec-i --suffix=B "$actual_size" 2>/dev/null || echo "$actual_size bytes")"
                models_invalid=$((models_invalid + 1))
            fi
        else
            log::warn "⚠️  $model_name - Not installed"
        fi
    done
    
    if [[ $models_invalid -gt 0 ]]; then
        log::info "   Run '$0 --action download-models' to fix corrupted models"
    elif [[ $models_valid -eq 0 ]]; then
        log::info "   Run '$0 --action download-models' to install default models"
    fi
}

# GPU detection function for backward compatibility
status::detect_gpu() {
    local gpu_type
    gpu_type=$(status::detect_gpu_silent)
    
    case "$gpu_type" in
        nvidia)
            log::info "🖥️  GPU Type: NVIDIA"
            ;;
        amd)
            log::info "🖥️  GPU Type: AMD"
            ;;
        cpu)
            log::info "🖥️  GPU Type: None (CPU mode)"
            ;;
    esac
    
    echo "$gpu_type"
}