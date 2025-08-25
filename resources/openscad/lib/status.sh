#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
OPENSCAD_STATUS_DIR="${APP_ROOT}/resources/openscad/lib"

# Source dependencies
source "${OPENSCAD_STATUS_DIR}/../../../../lib/utils/format.sh"
source "${OPENSCAD_STATUS_DIR}/../../../lib/status-args.sh"

# Collect OpenSCAD status data in format-agnostic structure
openscad::status::collect_data() {
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
    
    local status="stopped"
    local installed="false"
    local running="false"
    local healthy="false"
    local message="OpenSCAD is not installed"
    local version="unknown"
    local scripts_count=0
    local outputs_count=0
    
    if openscad::is_installed; then
        installed="true"
        
        if openscad::is_running; then
            status="running"
            running="true"
            
            # Check if we can execute commands in the container (skip in fast mode)
            if [[ "$fast_mode" == "true" ]]; then
                healthy="true"
                version="N/A"
                message="OpenSCAD is running (fast mode)"
                scripts_count="N/A"
                outputs_count="N/A"
            elif docker exec "${OPENSCAD_CONTAINER_NAME}" sh -c "openscad --version 2>&1" 2>/dev/null | grep -q "OpenSCAD"; then
                healthy="true"
                version=$(docker exec "${OPENSCAD_CONTAINER_NAME}" sh -c "openscad --version 2>&1" 2>/dev/null | head -1 | sed 's/OpenSCAD version //' || echo "unknown")
                message="OpenSCAD is running and healthy"
                
                # Count scripts and outputs
                scripts_count=$(find "${OPENSCAD_SCRIPTS_DIR}" -name "*.scad" 2>/dev/null | wc -l || echo 0)
                outputs_count=$(find "${OPENSCAD_OUTPUT_DIR}" -name "*.stl" -o -name "*.off" -o -name "*.png" 2>/dev/null | wc -l || echo 0)
            else
                message="OpenSCAD container is running but not responding"
            fi
        elif openscad::container_exists; then
            message="OpenSCAD container exists but is not running"
        else
            message="OpenSCAD is installed but not running"
        fi
    elif command -v docker >/dev/null 2>&1; then
        message="OpenSCAD is not installed (run 'resource-openscad install')"
    else
        message="Docker is required but not installed"
    fi
    
    # Build status data array
    local status_data=(
        "name" "openscad"
        "category" "execution"
        "description" "Programmatic 3D CAD modeler"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$healthy"
        "health_message" "$message"
        "version" "$version"
        "scripts" "$scripts_count"
        "outputs" "$outputs_count"
        "scripts_dir" "${OPENSCAD_SCRIPTS_DIR:-unknown}"
        "output_dir" "${OPENSCAD_OUTPUT_DIR:-unknown}"
    )
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
openscad::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "🔧 OpenSCAD Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install OpenSCAD, run: resource-openscad install"
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
    log::info "   📦 Version: ${data[version]:-unknown}"
    log::info "   📄 Scripts: ${data[scripts]:-0} in ${data[scripts_dir]:-unknown}"
    log::info "   📐 Outputs: ${data[outputs]:-0} in ${data[output_dir]:-unknown}"
    echo
    
    log::info "📋 Status Message:"
    log::info "   ${data[health_message]:-No status message}"
    echo
}

# Main status function using standard wrapper
openscad::status() {
    status::run_standard "openscad" "openscad::status::collect_data" "openscad::status::display_text" "$@"
}