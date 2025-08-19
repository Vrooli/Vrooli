#!/bin/bash
# Blender status functionality

# Get script directory
BLENDER_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Only source core.sh if functions aren't already defined
if ! type blender::is_running &>/dev/null; then
    source "${BLENDER_STATUS_DIR}/core.sh"
fi

# Source status-args library
source "${BLENDER_STATUS_DIR}/../../../lib/status-args.sh"

# Check Blender health
blender::health_check() {
    if ! blender::is_running; then
        return 1
    fi
    
    # Try to run a simple Blender command as health check
    if docker exec "$BLENDER_CONTAINER_NAME" blender --version &>/dev/null; then
        return 0
    fi
    
    return 1
}

# Get Blender version
blender::get_version() {
    if blender::is_running; then
        docker exec "$BLENDER_CONTAINER_NAME" blender --version 2>/dev/null | grep -oP 'Blender \K[0-9.]+' | head -1
    else
        echo "${BLENDER_VERSION}"
    fi
}

# Collect Blender status data in format-agnostic structure
blender::status::collect_data() {
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
    blender::init >/dev/null 2>&1
    
    local status="unknown"
    local message=""
    local installed="false"
    local running="false"
    local healthy="false"
    local version=""
    
    # Check installation
    if blender::is_installed; then
        installed="true"
        
        # Check if running
        if blender::is_running; then
            running="true"
            status="running"
            
            # Check health (skip in fast mode)
            if [[ "$fast_mode" == "true" ]]; then
                healthy="true"
                status="healthy"
                message="Blender is running (fast mode)"
                version="N/A"
            elif blender::health_check; then
                healthy="true"
                status="healthy"
                message="Blender is running and healthy"
                # Get version
                version=$(blender::get_version)
            else
                message="Blender is running but not responding"
                version="N/A"
            fi
        else
            status="stopped"
            message="Blender is installed but not running"
            version="${BLENDER_VERSION}"
        fi
    else
        status="not_installed"
        message="Blender is not installed"
        version="${BLENDER_VERSION}"
    fi
    
    # Count scripts and outputs (skip in fast mode)
    local script_count=0
    local output_count=0
    
    if [[ "$fast_mode" == "true" ]]; then
        script_count="N/A"
        output_count="N/A"
    else
        if [[ -d "$BLENDER_SCRIPTS_DIR" ]]; then
            script_count=$(timeout 3s find "$BLENDER_SCRIPTS_DIR" -name "*.py" -type f 2>/dev/null | wc -l)
        fi
        
        if [[ -d "$BLENDER_OUTPUT_DIR" ]]; then
            output_count=$(timeout 3s find "$BLENDER_OUTPUT_DIR" -type f 2>/dev/null | wc -l)
        fi
    fi
    
    # Build status data array
    local status_data=(
        "name" "blender"
        "category" "execution"
        "description" "Professional 3D creation suite with Python API"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$healthy"
        "health_message" "$message"
        "status" "$status"
        "version" "$version"
        "port" "$BLENDER_PORT"
        "scripts" "$script_count"
        "outputs" "$output_count"
        "scripts_dir" "$BLENDER_SCRIPTS_DIR"
        "output_dir" "$BLENDER_OUTPUT_DIR"
    )
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
blender::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Get verbose mode from environment or default to false
    local verbose="${STATUS_VERBOSE:-false}"
    
    # Source format utilities for proper logging functions
    source "${BLENDER_STATUS_DIR}/../../../../lib/utils/format.sh"
    
    # Header
    log::header "🎨 Blender Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Blender, run: ./manage.sh --action install"
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
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   🔢 Version: ${data[version]:-unknown}"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   📁 Scripts: ${data[scripts]:-0} files in ${data[scripts_dir]:-unknown}"
    log::info "   📊 Outputs: ${data[outputs]:-0} files in ${data[output_dir]:-unknown}"
    echo
    
    if [[ "$verbose" == "true" && "${data[running]:-false}" == "true" ]]; then
        log::info "🐳 Container Details:"
        if [[ -n "${data[container_id]:-}" ]]; then
            log::info "   📦 ID: ${data[container_id]}"
            log::info "   🖼️  Image: ${data[container_image]}"
            log::info "   📊 State: ${data[container_state]}"
        else
            log::info "   Run with --verbose for container details"
        fi
        echo
    fi
}

# Main status function using standard wrapper
blender::status() {
    status::run_standard "blender" "blender::status::collect_data" "blender::status::display_text" "$@"
}

# Export functions
export -f blender::health_check
export -f blender::get_version
export -f blender::status