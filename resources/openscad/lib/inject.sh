#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENSCAD_INJECT_DIR="${APP_ROOT}/resources/openscad/lib"

# Dependencies are expected to be sourced by caller

# Inject OpenSCAD script file
openscad::inject() {
    local file_path="${1:-}"
    
    if [[ -z "$file_path" ]]; then
        log::error "Usage: resource-openscad inject <file.scad>"
        return 1
    fi
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    if [[ ! "$file_path" =~ \.scad$ ]]; then
        log::warning "File does not have .scad extension: $file_path"
    fi
    
    # Ensure OpenSCAD is running
    if ! openscad::is_running; then
        log::warning "OpenSCAD is not running, starting it..."
        openscad::start || return 1
    fi
    
    # Copy file to injected directory
    local filename=$(basename "$file_path")
    local dest_path="${OPENSCAD_INJECTED_DIR}/${filename}"
    
    log::info "Injecting OpenSCAD script: ${filename}"
    
    if cp "$file_path" "$dest_path"; then
        # Also copy to scripts directory for processing
        cp "$file_path" "${OPENSCAD_SCRIPTS_DIR}/${filename}"
        
        # Try to render it to STL as a test
        log::info "Testing script by rendering to STL..."
        local output_file="${OPENSCAD_OUTPUT_DIR}/${filename%.scad}.stl"
        
        if docker exec "${OPENSCAD_CONTAINER_NAME}" openscad -o "/output/$(basename "$output_file")" "/scripts/${filename}" 2>/dev/null; then
            log::success "Script injected and rendered successfully: ${filename}"
            log::info "Output saved to: ${output_file}"
        else
            log::warning "Script injected but rendering failed (script may require parameters)"
            log::success "Script injected: ${filename}"
        fi
        
        return 0
    else
        log::error "Failed to inject script: ${filename}"
        return 1
    fi
}

# List injected OpenSCAD scripts
openscad::list_injected() {
    log::info "Injected OpenSCAD scripts:"
    
    if [[ -d "${OPENSCAD_INJECTED_DIR}" ]]; then
        local count=$(find "${OPENSCAD_INJECTED_DIR}" -name "*.scad" 2>/dev/null | wc -l)
        
        if [[ $count -eq 0 ]]; then
            log::info "No scripts injected yet"
        else
            find "${OPENSCAD_INJECTED_DIR}" -name "*.scad" -exec ls -la {} \; 2>/dev/null
            
            # Also show outputs if any
            local output_count=$(find "${OPENSCAD_OUTPUT_DIR}" -type f 2>/dev/null | wc -l)
            if [[ $output_count -gt 0 ]]; then
                echo ""
                log::info "Generated outputs:"
                find "${OPENSCAD_OUTPUT_DIR}" -type f -exec ls -la {} \; 2>/dev/null | head -10
            fi
        fi
    else
        log::info "Injected directory does not exist yet"
    fi
}

# Render an OpenSCAD script to various formats
openscad::render() {
    local script_path="${1:-}"
    local output_format="${2:-stl}"
    local output_name="${3:-}"
    
    if [[ -z "$script_path" ]]; then
        log::error "Usage: resource-openscad render <script.scad> [format] [output_name]"
        log::info "Formats: stl, off, amf, 3mf, dxf, svg, png"
        return 1
    fi
    
    if [[ ! -f "$script_path" ]]; then
        log::error "Script not found: $script_path"
        return 1
    fi
    
    # Ensure OpenSCAD is running
    if ! openscad::is_running; then
        log::warning "OpenSCAD is not running, starting it..."
        openscad::start || return 1
    fi
    
    local filename=$(basename "$script_path")
    local script_name="${filename%.scad}"
    
    if [[ -z "$output_name" ]]; then
        output_name="${script_name}"
    fi
    
    # Copy script to container-accessible location
    cp "$script_path" "${OPENSCAD_SCRIPTS_DIR}/${filename}"
    
    local output_file="${OPENSCAD_OUTPUT_DIR}/${output_name}.${output_format}"
    
    log::info "Rendering ${filename} to ${output_format}..."
    
    if docker exec "${OPENSCAD_CONTAINER_NAME}" openscad -o "/output/${output_name}.${output_format}" "/scripts/${filename}"; then
        log::success "Rendered successfully: ${output_file}"
        return 0
    else
        log::error "Failed to render ${filename}"
        return 1
    fi
}

# Clear all OpenSCAD data
openscad::clear_data() {
    log::warning "Clearing all OpenSCAD data..."
    
    if [[ -d "${OPENSCAD_SCRIPTS_DIR}" ]]; then
        rm -rf "${OPENSCAD_SCRIPTS_DIR}"/*
        log::info "Cleared scripts directory"
    fi
    
    if [[ -d "${OPENSCAD_OUTPUT_DIR}" ]]; then
        rm -rf "${OPENSCAD_OUTPUT_DIR}"/*
        log::info "Cleared output directory"
    fi
    
    if [[ -d "${OPENSCAD_INJECTED_DIR}" ]]; then
        rm -rf "${OPENSCAD_INJECTED_DIR}"/*
        log::info "Cleared injected directory"
    fi
    
    log::success "All OpenSCAD data cleared"
}