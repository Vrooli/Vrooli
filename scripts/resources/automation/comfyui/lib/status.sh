#!/usr/bin/env bash
# ComfyUI Status and Information
# Handles status checks, health monitoring, and information display

#######################################
# Display ComfyUI status
#######################################
comfyui::status() {
    log::header "📊 ComfyUI Status"
    
    # Show access URL prominently if running
    if comfyui::is_running; then
        echo
        log::success "🌐 Access ComfyUI at: http://localhost:$COMFYUI_DIRECT_PORT"
        echo
    fi
    
    # Container status
    echo
    log::info "=== Container Status ==="
    
    if comfyui::container_exists; then
        log::success "✅ Container exists"
        
        if comfyui::is_running; then
            log::success "✅ Container is running"
            
            # Show container details
            local container_info
            container_info=$(docker::run ps --filter "name=^${COMFYUI_CONTAINER_NAME}$" --format "table {{.Status}}\t{{.Ports}}" --no-trunc | tail -n +2)
            if [[ -n "$container_info" ]]; then
                echo "   Status: $(echo "$container_info" | awk '{print $1, $2, $3}')"
                echo "   Ports: $(echo "$container_info" | awk '{$1=$2=$3=""; print $0}' | xargs)"
            fi
            
            # Check API health
            if comfyui::is_healthy; then
                log::success "✅ API is healthy"
            else
                log::error "❌ API is not responding"
            fi
        else
            log::error "❌ Container is not running"
            log::info "   Start with: $0 --action start"
        fi
    else
        log::error "❌ Container does not exist"
        log::info "   Install with: $0 --action install"
    fi
    
    # GPU status
    echo
    log::info "=== GPU Configuration ==="
    
    local gpu_type
    gpu_type=$(comfyui::detect_gpu_silent)
    
    case "$gpu_type" in
        nvidia)
            log::info "GPU Type: NVIDIA"
            
            # Check if container has GPU access
            if comfyui::is_running; then
                if docker::run exec "$COMFYUI_CONTAINER_NAME" nvidia-smi >/dev/null 2>&1; then
                    log::success "✅ GPU is accessible in container"
                    
                    # Show GPU info from container
                    local gpu_info
                    gpu_info=$(docker::run exec "$COMFYUI_CONTAINER_NAME" nvidia-smi --query-gpu=name,memory.used,memory.total --format=csv,noheader 2>/dev/null | head -1)
                    if [[ -n "$gpu_info" ]]; then
                        echo "   GPU: $gpu_info"
                    fi
                else
                    log::error "❌ GPU not accessible in container"
                    log::info "   Container may be running in CPU mode"
                fi
            fi
            ;;
        amd)
            log::info "GPU Type: AMD"
            log::warn "AMD GPU support may require additional configuration"
            ;;
        cpu)
            log::info "GPU Type: None (CPU mode)"
            log::warn "Running in CPU mode - performance will be limited"
            ;;
    esac
    
    # Resource usage
    if comfyui::is_running; then
        echo
        log::info "=== Resource Usage ==="
        
        # Get container stats
        local stats
        stats=$(docker::run stats --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}" "$COMFYUI_CONTAINER_NAME" | tail -n +2)
        if [[ -n "$stats" ]]; then
            echo "   CPU: $(echo "$stats" | awk '{print $1}')"
            echo "   Memory: $(echo "$stats" | awk '{print $2, $3, $4}')"
        fi
    fi
    
    # Storage usage
    echo
    log::info "=== Storage Usage ==="
    
    if [[ -d "$COMFYUI_DATA_DIR" ]]; then
        local total_size
        total_size=$(du -sh "$COMFYUI_DATA_DIR" 2>/dev/null | awk '{print $1}')
        echo "   Data directory: $COMFYUI_DATA_DIR"
        echo "   Total size: ${total_size:-unknown}"
        
        # Show breakdown by category
        for dir in models outputs workflows custom_nodes; do
            if [[ -d "${COMFYUI_DATA_DIR}/$dir" ]]; then
                local dir_size
                dir_size=$(du -sh "${COMFYUI_DATA_DIR}/$dir" 2>/dev/null | awk '{print $1}')
                echo "   - $dir: ${dir_size:-0}"
            fi
        done
    else
        log::warn "Data directory does not exist"
    fi
    
    # Configuration status
    echo
    log::info "=== Configuration ==="
    
    local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
    echo "   Port: $port"
    echo "   Image: ${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
    
    # Check Vrooli configuration
    if [[ -f "$VROOLI_RESOURCES_CONFIG" ]] && system::is_command "jq"; then
        local config_exists
        config_exists=$(jq -r ".services.automation.comfyui // empty" "$VROOLI_RESOURCES_CONFIG" 2>/dev/null)
        
        if [[ -n "$config_exists" ]]; then
            log::success "✅ Registered in Vrooli configuration"
        else
            log::warn "⚠️  Not registered in Vrooli configuration"
        fi
    fi
    
    # Quick actions
    if comfyui::container_exists; then
        echo
        log::header "🎯 Quick Actions"
        
        if comfyui::is_running; then
            echo "  • View logs: $0 --action logs"
            echo "  • Stop service: $0 --action stop"
            echo "  • Access ComfyUI: http://localhost:$COMFYUI_DIRECT_PORT"
            echo "  • Access AI-Dock Portal: http://localhost:$port"
        else
            echo "  • Start service: $0 --action start"
        fi
        
        echo "  • List models: $0 --action list-models"
        echo "  • GPU info: $0 --action gpu-info"
    fi
    
    # Exit with appropriate code
    if comfyui::is_running && comfyui::is_healthy; then
        return 0
    else
        return 1
    fi
}

#######################################
# Display ComfyUI resource information
#######################################
comfyui::info() {
    log::header "ℹ️ ComfyUI Resource Information"
    
    echo
    echo "ComfyUI is a powerful and modular AI image generation workflow platform."
    echo
    
    # Basic information
    log::info "=== Resource Details ==="
    echo "  Name: comfyui"
    echo "  Category: automation"
    echo "  Type: Docker container"
    echo "  Image: ${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
    echo
    
    # Features
    log::info "=== Features ==="
    echo "  • Node-based workflow editor"
    echo "  • Support for SD, SDXL, and custom models"
    echo "  • GPU acceleration (NVIDIA/AMD)"
    echo "  • REST API for automation"
    echo "  • Custom node support"
    echo "  • Batch processing"
    echo
    
    # Access information
    log::info "=== Access Information ==="
    local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
    echo "  ComfyUI Interface: http://localhost:$COMFYUI_DIRECT_PORT"
    echo "  AI-Dock Portal: http://localhost:$port"
    echo "  API Endpoint: http://localhost:$COMFYUI_DIRECT_PORT"
    echo
    
    # API endpoints
    log::info "=== API Endpoints ==="
    echo "  POST /prompt              - Execute workflow"
    echo "  GET  /history/{id}        - Get execution status"
    echo "  GET  /view                - View generated images"
    echo "  POST /upload/image        - Upload input images"
    echo "  GET  /system_stats        - System information"
    echo "  WS   ws://localhost:$port/ws - Real-time updates"
    echo
    
    # Directory structure
    log::info "=== Directory Structure ==="
    echo "  ${COMFYUI_DATA_DIR}/"
    echo "  ├── models/              # AI models"
    echo "  │   ├── checkpoints/     # Main models"
    echo "  │   ├── vae/            # VAE models"
    echo "  │   ├── loras/          # LoRA models"
    echo "  │   └── controlnet/     # ControlNet models"
    echo "  ├── outputs/            # Generated images"
    echo "  ├── workflows/          # Saved workflows"
    echo "  ├── custom_nodes/       # Custom nodes"
    echo "  └── input/             # Input images"
    echo
    
    # Resource requirements
    log::info "=== Resource Requirements ==="
    echo "  RAM: ${COMFYUI_MIN_RAM_GB}GB minimum, ${COMFYUI_RECOMMENDED_RAM_GB}GB recommended"
    echo "  Disk: ${COMFYUI_MIN_DISK_GB}GB minimum, ${COMFYUI_RECOMMENDED_DISK_GB}GB recommended"
    echo "  GPU: NVIDIA/AMD recommended, CPU mode available"
    echo
    
    # Integration
    log::info "=== Vrooli Integration ==="
    echo "  ComfyUI integrates with Vrooli's automation system,"
    echo "  allowing AI-powered image generation in workflows."
    echo "  Use with n8n, Node-RED, or other automation tools."
    echo
    
    # Documentation
    log::info "=== Documentation ==="
    echo "  ComfyUI Docs: https://docs.comfy.org/"
    echo "  GitHub: https://github.com/comfyanonymous/ComfyUI"
    echo "  Examples: https://comfyanonymous.github.io/ComfyUI_examples/"
    echo
    
    # Current status
    if comfyui::container_exists; then
        log::info "=== Current Status ==="
        if comfyui::is_running; then
            if comfyui::is_healthy; then
                log::success "✅ Running and healthy"
            else
                log::warn "⚠️  Running but API not responding"
            fi
        else
            log::info "⏸️  Installed but not running"
        fi
    else
        log::info "📦 Not installed"
    fi
}

#######################################
# Check if ready for operations
#######################################
comfyui::verify_ready() {
    # This function is called by check_ready in common.sh
    # Additional ComfyUI-specific checks can go here
    
    # Check if we have at least one model
    if [[ -d "${COMFYUI_DATA_DIR}/models/checkpoints" ]]; then
        local model_count
        model_count=$(find "${COMFYUI_DATA_DIR}/models/checkpoints" -name "*.safetensors" -o -name "*.ckpt" | wc -l)
        
        if [[ $model_count -eq 0 ]]; then
            log::warn "No checkpoint models found"
            log::info "Download models with: $0 --action download-models"
        fi
    fi
    
    return 0
}

#######################################
# Show system requirements
#######################################
comfyui::show_requirements() {
    log::header "📋 ComfyUI System Requirements"
    
    echo
    log::info "=== Minimum Requirements ==="
    echo "  • RAM: ${COMFYUI_MIN_RAM_GB}GB"
    echo "  • Disk Space: ${COMFYUI_MIN_DISK_GB}GB"
    echo "  • Docker installed and running"
    echo "  • CPU: 4+ cores recommended"
    echo
    
    log::info "=== Recommended Requirements ==="
    echo "  • RAM: ${COMFYUI_RECOMMENDED_RAM_GB}GB+"
    echo "  • Disk Space: ${COMFYUI_RECOMMENDED_DISK_GB}GB+"
    echo "  • GPU: NVIDIA RTX 3060+ or AMD RX 6600+"
    echo "  • VRAM: 12GB+ for SDXL models"
    echo "  • CPU: 8+ cores"
    echo
    
    log::info "=== GPU Requirements ==="
    echo "  NVIDIA:"
    echo "  • Driver version ${COMFYUI_NVIDIA_MIN_DRIVER}+"
    echo "  • CUDA 11.8+"
    echo "  • NVIDIA Container Runtime"
    echo
    echo "  AMD:"
    echo "  • ROCm ${COMFYUI_AMD_MIN_ROCM}+"
    echo "  • Compatible GPU (RX 6000 series+)"
    echo
    
    log::info "=== Current System ==="
    
    # Check RAM
    local total_ram_gb
    total_ram_gb=$(free -g | awk '/^Mem:/{print $2}')
    echo "  RAM: ${total_ram_gb}GB"
    
    # Check disk space
    local available_gb
    available_gb=$(df -BG "$(dirname "$COMFYUI_DATA_DIR")" | awk 'NR==2 {print $4}' | sed 's/G//')
    echo "  Available disk: ${available_gb}GB"
    
    # Check CPU
    local cpu_cores
    cpu_cores=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo "unknown")
    echo "  CPU cores: $cpu_cores"
    
    # Check GPU
    local gpu_type
    gpu_type=$(comfyui::detect_gpu_silent)
    echo "  GPU type: $gpu_type"
}