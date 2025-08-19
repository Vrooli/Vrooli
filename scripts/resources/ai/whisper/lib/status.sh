#!/usr/bin/env bash
# Whisper Status Management - Standardized Format
# Functions for checking and displaying Whisper status information

# Source format utilities and config
WHISPER_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${WHISPER_STATUS_DIR}/../../../../lib/utils/format.sh"
# shellcheck disable=SC1091
source "${WHISPER_STATUS_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WHISPER_STATUS_DIR}/../config/messages.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WHISPER_STATUS_DIR}/common.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v defaults::export_config &>/dev/null; then
    defaults::export_config 2>/dev/null || true
fi

#######################################
# Collect Whisper status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
whisper::status::collect_data() {
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    if common::container_exists; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "$WHISPER_CONTAINER_NAME" 2>/dev/null || echo "unknown")
        
        if common::is_running; then
            running="true"
            
            if whisper::is_healthy; then
                healthy="true"
                health_message="Healthy - AI transcription service ready"
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
    status_data+=("name" "whisper")
    status_data+=("category" "ai")
    status_data+=("description" "OpenAI Whisper ASR (Automatic Speech Recognition) service")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$WHISPER_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$WHISPER_PORT")
    
    # Service endpoints
    status_data+=("base_url" "$WHISPER_BASE_URL")
    status_data+=("api_url" "$WHISPER_BASE_URL/asr")
    status_data+=("health_url" "$WHISPER_BASE_URL/docs")
    
    # Configuration details
    local image
    image=$(whisper::get_docker_image)
    status_data+=("image" "$image")
    status_data+=("default_model" "$WHISPER_DEFAULT_MODEL")
    status_data+=("data_dir" "$WHISPER_DATA_DIR")
    status_data+=("models_dir" "$WHISPER_MODELS_DIR")
    status_data+=("uploads_dir" "$WHISPER_UPLOADS_DIR")
    status_data+=("gpu_enabled" "$WHISPER_GPU_ENABLED")
    
    # Runtime information (only if running and healthy)
    if [[ "$running" == "true" ]]; then
        # GPU availability
        if [[ "$WHISPER_GPU_ENABLED" == "yes" ]]; then
            local gpu_available="false"
            if whisper::is_gpu_available; then
                gpu_available="true"
                if command -v nvidia-smi &>/dev/null; then
                    local gpu_info
                    gpu_info=$(nvidia-smi --query-gpu=name,memory.used,memory.total --format=csv,noheader,nounits 2>/dev/null | head -1 || echo "unknown")
                    status_data+=("gpu_info" "$gpu_info")
                fi
            fi
            status_data+=("gpu_available" "$gpu_available")
        fi
        
        # Get current model from container
        local current_model
        current_model=$(status::get_current_model 2>/dev/null || echo "$WHISPER_DEFAULT_MODEL")
        status_data+=("current_model" "$current_model")
        
        # Model size information
        local model_size
        model_size=$(common::get_model_size "$current_model")
        status_data+=("model_size_gb" "$model_size")
        
        # Storage information
        if [[ -d "$WHISPER_DATA_DIR" ]]; then
            local data_size models_count uploads_count
            data_size=$(du -sh "$WHISPER_DATA_DIR" 2>/dev/null | cut -f1 || echo "unknown")
            status_data+=("data_size" "$data_size")
            
            # Count model files
            models_count=0
            if [[ -d "$WHISPER_MODELS_DIR" ]]; then
                models_count=$(find "$WHISPER_MODELS_DIR" -type f 2>/dev/null | wc -l)
            fi
            status_data+=("models_count" "$models_count")
            
            # Count upload files
            uploads_count=0
            if [[ -d "$WHISPER_UPLOADS_DIR" ]]; then
                uploads_count=$(find "$WHISPER_UPLOADS_DIR" -type f 2>/dev/null | wc -l)
            fi
            status_data+=("uploads_count" "$uploads_count")
        fi
        
        # Container resource usage
        local stats
        stats=$(docker stats --no-stream --format "{{.CPUPerc}};{{.MemUsage}}" "$WHISPER_CONTAINER_NAME" 2>/dev/null || echo "N/A;N/A")
        
        if [[ "$stats" != "N/A;N/A" ]]; then
            local cpu_usage memory_usage
            cpu_usage=$(echo "$stats" | cut -d';' -f1)
            memory_usage=$(echo "$stats" | cut -d';' -f2)
            status_data+=("cpu_usage" "$cpu_usage")
            status_data+=("memory_usage" "$memory_usage")
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show Whisper status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
whisper::status::show() {
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
    data_string=$(whisper::status::collect_data 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect Whisper status data"
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
        whisper::status::display_text "${data_array[@]}"
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
whisper::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🎤 Whisper Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Whisper, run: ./manage.sh --action install"
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
    log::info "   🎯 ASR API: ${data[api_url]:-unknown}"
    log::info "   💊 Health: ${data[health_url]:-unknown}"
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   🧠 Default Model: ${data[default_model]:-unknown}"
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
        log::info "   🧠 Current Model: ${data[current_model]:-unknown}"
        log::info "   📏 Model Size: ${data[model_size_gb]:-unknown} GB"
        
        if [[ -n "${data[cpu_usage]:-}" ]]; then
            log::info "   💻 CPU Usage: ${data[cpu_usage]}"
        fi
        if [[ -n "${data[memory_usage]:-}" ]]; then
            log::info "   🧠 Memory Usage: ${data[memory_usage]}"
        fi
        if [[ -n "${data[data_size]:-}" ]]; then
            log::info "   💾 Data Size: ${data[data_size]}"
        fi
        if [[ -n "${data[models_count]:-}" ]]; then
            log::info "   📚 Models: ${data[models_count]} files"
        fi
        if [[ -n "${data[uploads_count]:-}" ]]; then
            log::info "   📤 Uploads: ${data[uploads_count]} files"
        fi
        echo
        
        # Quick access info
        log::info "🎯 Quick Actions:"
        log::info "   🎤 Transcribe audio: ./manage.sh --action transcribe --file <audio-file>"
        log::info "   📄 View logs: ./manage.sh --action logs"
        log::info "   🛑 Stop service: ./manage.sh --action stop"
    fi
}

#######################################
# Main status function for CLI registration
#######################################
whisper::status() {
    whisper::status::show "$@"
}

# Legacy compatibility - keep existing functions unchanged

#######################################
# Show comprehensive Whisper status
# Returns: 0 if successful, 1 otherwise
#######################################
status::show_status() {
    log::header "🎤 Whisper Status"
    
    local overall_status="healthy"
    
    # Check Docker availability
    echo "Docker Status:"
    if common::check_docker; then
        log::success "  ✅ Docker is available and configured"
    else
        log::error "  ❌ Docker is not available"
        overall_status="unhealthy"
    fi
    
    echo
    
    # Check container status
    echo "Container Status:"
    if common::container_exists; then
        if common::is_running; then
            log::success "  ✅ Whisper container is running"
            
            # Get basic container info
            local created=$(docker inspect "$WHISPER_CONTAINER_NAME" --format '{{.Created}}' 2>/dev/null)
            local image=$(docker inspect "$WHISPER_CONTAINER_NAME" --format '{{.Config.Image}}' 2>/dev/null)
            log::info "     Container: $WHISPER_CONTAINER_NAME"
            log::info "     Image: $image"
            log::info "     Created: ${created:0:19}"
        else
            log::warn "  ⚠️  Whisper container exists but is not running"
            overall_status="degraded"
        fi
    else
        log::info "  ℹ️  Whisper container does not exist"
        overall_status="not_installed"
    fi
    
    echo
    
    # Check API status
    echo "API Status:"
    if common::is_running; then
        log::info "  🔍 Testing API endpoint: $WHISPER_BASE_URL"
        
        if whisper::is_healthy; then
            log::success "  ✅ Whisper API is responding"
            
            # Get model information if available
            local model_info
            model_info=$(status::get_current_model 2>/dev/null)
            if [[ -n "$model_info" ]]; then
                log::info "     Current model: $model_info"
            fi
        else
            log::error "  ❌ Whisper API is not responding"
            overall_status="unhealthy"
        fi
    else
        log::info "  ⏸️  API not available (container not running)"
    fi
    
    echo
    
    # Check port status
    echo "Network Status:"
    log::info "  Port: $WHISPER_PORT"
    if system::is_port_in_use "$WHISPER_PORT"; then
        if common::is_running; then
            log::success "  ✅ Port $WHISPER_PORT is in use by Whisper"
        else
            log::warn "  ⚠️  Port $WHISPER_PORT is in use by another service"
            overall_status="conflict"
        fi
    else
        log::info "  ℹ️  Port $WHISPER_PORT is available"
    fi
    
    echo
    
    # Check data directories
    echo "Storage Status:"
    if [[ -d "$WHISPER_DATA_DIR" ]]; then
        log::success "  ✅ Data directory exists: $WHISPER_DATA_DIR"
        
        # Check subdirectories
        for dir in "$WHISPER_MODELS_DIR" "$WHISPER_UPLOADS_DIR"; do
            if [[ -d "$dir" ]]; then
                local size=$(du -sh "$dir" 2>/dev/null | cut -f1)
                log::info "     $(basename "$dir"): $size"
            else
                log::info "     $(basename "$dir"): not created"
            fi
        done
    else
        log::info "  ℹ️  Data directory not created: $WHISPER_DATA_DIR"
    fi
    
    echo
    
    # Check GPU status if GPU is enabled
    if [[ "$WHISPER_GPU_ENABLED" == "yes" ]]; then
        echo "GPU Status:"
        if whisper::is_gpu_available; then
            log::success "  ✅ NVIDIA GPU is available"
            if system::is_command "nvidia-smi"; then
                local gpu_info=$(nvidia-smi --query-gpu=name,memory.used,memory.total --format=csv,noheader,nounits 2>/dev/null | head -1)
                if [[ -n "$gpu_info" ]]; then
                    log::info "     GPU: $gpu_info"
                fi
            fi
        else
            log::warn "  ⚠️  GPU not available, will use CPU"
        fi
        echo
    fi
    
    # Configuration summary
    echo "Configuration:"
    log::info "  Base URL: $WHISPER_BASE_URL"
    log::info "  Container: $WHISPER_CONTAINER_NAME"
    log::info "  Default Model: $WHISPER_DEFAULT_MODEL"
    log::info "  Data Directory: $WHISPER_DATA_DIR"
    log::info "  GPU Enabled: $WHISPER_GPU_ENABLED"
    
    echo
    
    # Overall status summary
    case "$overall_status" in
        "healthy")
            log::success "🎉 Overall Status: Healthy - Whisper is running and responding"
            return 0
            ;;
        "degraded")
            log::warn "⚠️  Overall Status: Degraded - Some issues detected"
            return 0
            ;;
        "conflict")
            log::warn "⚠️  Overall Status: Port Conflict - Port is in use by another service"
            return 1
            ;;
        "unhealthy")
            log::error "❌ Overall Status: Unhealthy - Whisper is not working properly"
            return 1
            ;;
        "not_installed")
            log::info "ℹ️  Overall Status: Not Installed - Run installation first"
            return 1
            ;;
        *)
            log::error "❓ Overall Status: Unknown"
            return 1
            ;;
    esac
}

#######################################
# Get current model from running container
# Outputs: model name
#######################################
status::get_current_model() {
    if ! common::is_running; then
        return 1
    fi
    
    # Try to get model from environment variable
    docker inspect "$WHISPER_CONTAINER_NAME" --format '{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null | \
        grep "ASR_MODEL=" | \
        cut -d= -f2
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
    
    if whisper::is_healthy; then
        echo "running"
        return 0
    else
        echo "unhealthy"
        return 1
    fi
}

#######################################
# Check if Whisper is ready for transcription
# Returns: 0 if ready, 1 otherwise
#######################################
status::is_ready() {
    if ! common::is_running; then
        return 1
    fi
    
    if ! whisper::is_healthy; then
        return 1
    fi
    
    # Additional check: ensure model is loaded
    # This might take time on first startup
    local attempts=0
    local max_attempts=5
    
    while [[ $attempts -lt $max_attempts ]]; do
        local response
        response=$(curl -s -o /dev/null -w "%{http_code}" \
            --max-time "$WHISPER_API_TIMEOUT" \
            "$WHISPER_BASE_URL/docs" 2>/dev/null)
        
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
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$WHISPER_CONTAINER_NAME"
}

#######################################
# Legacy compatibility functions - keep existing behavior unchanged
#######################################

status::show_status() {
    # Legacy function - redirect to new standardized function
    whisper::status::show "$@"
}

# Export functions for subshell availability
export -f whisper::status::collect_data
export -f whisper::status::show
export -f whisper::status::display_text
export -f whisper::status
export -f status::show_status
export -f status::get_current_model
export -f status::quick_status
export -f status::is_ready
export -f status::show_resource_usage