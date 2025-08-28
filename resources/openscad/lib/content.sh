#!/usr/bin/env bash
# OpenSCAD Content Management Functions
# Business functionality for 3D modeling and rendering

# Add OpenSCAD script content
openscad::content::add() {
    local file=""
    local name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                # If no flag, assume it's the file
                if [[ -z "$file" ]]; then
                    file="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        log::error "File required"
        echo "Usage: resource-openscad content add --file <script.scad> [--name <name>]"
        return 1
    fi
    
    # Initialize directories if needed
    openscad::ensure_dirs
    
    # Handle shared resources
    if [[ "$file" =~ ^shared: ]]; then
        local shared_path="${file#shared:}"
        file="${VROOLI_SHARED_DIR:-/opt/vrooli/shared}/${shared_path}"
        
        if [[ ! -f "$file" ]]; then
            log::error "Shared resource not found: $shared_path"
            return 1
        fi
    fi
    
    # Check if file exists
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Validate OpenSCAD file
    if [[ ! "$file" =~ \.scad$ ]]; then
        log::warn "File should have .scad extension"
    fi
    
    # Determine name if not provided
    if [[ -z "$name" ]]; then
        name=$(basename "$file")
    fi
    
    # Copy file to scripts and injected directories
    local target_path="${OPENSCAD_SCRIPTS_DIR}/${name}"
    local injected_path="${OPENSCAD_INJECTED_DIR}/${name}"
    
    log::info "Adding OpenSCAD script: $name"
    
    cp "$file" "$target_path"
    # Only copy to injected if it's not already there
    if [[ "$file" != "$injected_path" ]]; then
        cp "$file" "$injected_path"
    fi
    chmod 644 "$target_path"
    [[ -f "$injected_path" ]] && chmod 644 "$injected_path"
    
    log::success "OpenSCAD script added: $name"
    
    # Show script info
    echo ""
    echo "Script: $name"
    echo "Size: $(stat -c%s "$target_path" 2>/dev/null || stat -f%z "$target_path" 2>/dev/null) bytes"
    echo "Location: $target_path"
    echo ""
    echo "Execute with: resource-openscad content execute --name $name"
}

# List all OpenSCAD content
openscad::content::list() {
    local format="${1:-text}"
    
    if [[ "$format" == "json" ]]; then
        # JSON output
        local scripts=()
        local outputs=()
        
        # List scripts
        if [[ -d "$OPENSCAD_SCRIPTS_DIR" ]]; then
            while IFS= read -r file; do
                [[ -f "$file" ]] && scripts+=("\"$(basename "$file")\"")
            done < <(find "$OPENSCAD_SCRIPTS_DIR" -name "*.scad" 2>/dev/null)
        fi
        
        # List outputs
        if [[ -d "$OPENSCAD_OUTPUT_DIR" ]]; then
            while IFS= read -r file; do
                [[ -f "$file" ]] && outputs+=("\"$(basename "$file")\"")
            done < <(find "$OPENSCAD_OUTPUT_DIR" -type f 2>/dev/null)
        fi
        
        # Build JSON
        echo "{"
        echo "  \"scripts\": [$(IFS=,; echo "${scripts[*]:-}")],"
        echo "  \"outputs\": [$(IFS=,; echo "${outputs[*]:-}")]"
        echo "}"
    else
        # Text output
        echo "OpenSCAD Content:"
        echo "================"
        
        echo ""
        echo "Scripts:"
        if [[ -d "$OPENSCAD_SCRIPTS_DIR" ]]; then
            local script_count=0
            while IFS= read -r file; do
                if [[ -f "$file" ]]; then
                    echo "  - $(basename "$file")"
                    ((script_count++))
                fi
            done < <(find "$OPENSCAD_SCRIPTS_DIR" -name "*.scad" 2>/dev/null | sort)
            [[ $script_count -eq 0 ]] && echo "  No scripts found"
        else
            echo "  Scripts directory not found"
        fi
        
        echo ""
        echo "Rendered Outputs:"
        if [[ -d "$OPENSCAD_OUTPUT_DIR" ]]; then
            local output_count=0
            while IFS= read -r file; do
                if [[ -f "$file" ]]; then
                    echo "  - $(basename "$file")"
                    ((output_count++))
                fi
            done < <(find "$OPENSCAD_OUTPUT_DIR" -type f 2>/dev/null | sort)
            [[ $output_count -eq 0 ]] && echo "  No outputs found"
        else
            echo "  Output directory not found"
        fi
        
        echo ""
        echo "Directories:"
        echo "  Scripts: ${OPENSCAD_SCRIPTS_DIR}"
        echo "  Outputs: ${OPENSCAD_OUTPUT_DIR}"
        echo "  Injected: ${OPENSCAD_INJECTED_DIR}"
    fi
}

# Get specific OpenSCAD content
openscad::content::get() {
    local name=""
    local type="script"  # Default to script
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            *)
                # If no flag, assume it's the name
                if [[ -z "$name" ]]; then
                    name="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Name required"
        echo "Usage: resource-openscad content get --name <name> [--type script|output]"
        return 1
    fi
    
    # Determine source path based on type
    local source_path=""
    case "$type" in
        script)
            source_path="$OPENSCAD_SCRIPTS_DIR/$name"
            [[ ! "$name" =~ \.scad$ ]] && source_path="${source_path}.scad"
            ;;
        output)
            source_path="$OPENSCAD_OUTPUT_DIR/$name"
            ;;
        *)
            log::error "Invalid type: $type (use 'script' or 'output')"
            return 1
            ;;
    esac
    
    # Check if file exists
    if [[ ! -f "$source_path" ]]; then
        log::error "$type not found: $name"
        return 1
    fi
    
    # Output the content (for binary outputs, just show info)
    if [[ "$type" == "output" && ! "$name" =~ \.(scad|dxf|svg)$ ]]; then
        # Binary file - show info instead of content
        echo "File: $name"
        echo "Type: Binary output"
        echo "Size: $(stat -c%s "$source_path" 2>/dev/null || stat -f%z "$source_path" 2>/dev/null) bytes"
        echo "Path: $source_path"
    else
        # Text file - show content
        cat "$source_path"
    fi
}

# Remove OpenSCAD content
openscad::content::remove() {
    local name=""
    local type="script"  # Default to script
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            *)
                # If no flag, assume it's the name
                if [[ -z "$name" ]]; then
                    name="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Name required"
        echo "Usage: resource-openscad content remove --name <name> [--type script|output]"
        return 1
    fi
    
    # Determine target paths based on type
    case "$type" in
        script)
            local script_path="$OPENSCAD_SCRIPTS_DIR/$name"
            [[ ! "$name" =~ \.scad$ ]] && script_path="${script_path}.scad"
            local injected_path="$OPENSCAD_INJECTED_DIR/$name"
            [[ ! "$name" =~ \.scad$ ]] && injected_path="${injected_path}.scad"
            
            if [[ ! -f "$script_path" ]]; then
                log::error "Script not found: $name"
                return 1
            fi
            
            # Remove from both locations
            rm -f "$script_path" "$injected_path"
            log::success "Script removed: $name"
            ;;
        output)
            local output_path="$OPENSCAD_OUTPUT_DIR/$name"
            
            if [[ ! -f "$output_path" ]]; then
                log::error "Output not found: $name"
                return 1
            fi
            
            rm -f "$output_path"
            log::success "Output removed: $name"
            ;;
        *)
            log::error "Invalid type: $type (use 'script' or 'output')"
            return 1
            ;;
    esac
}

# Execute OpenSCAD content - render a script to various formats
openscad::content::execute() {
    local name=""
    local format="stl"
    local output_name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            --output)
                output_name="$2"
                shift 2
                ;;
            *)
                # If no flag, assume it's the name
                if [[ -z "$name" ]]; then
                    name="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Script name required"
        echo "Usage: resource-openscad content execute --name <script> [--format <format>] [--output <name>]"
        echo "Formats: stl, off, amf, 3mf, dxf, svg, png"
        return 1
    fi
    
    # Add .scad extension if not present
    [[ ! "$name" =~ \.scad$ ]] && name="${name}.scad"
    
    local script_path="$OPENSCAD_SCRIPTS_DIR/$name"
    
    # Check if script exists
    if [[ ! -f "$script_path" ]]; then
        log::error "Script not found: $name"
        echo "Available scripts:"
        openscad::content::list | grep -A 20 "Scripts:" | grep "^  -"
        return 1
    fi
    
    # Ensure OpenSCAD is running
    if ! openscad::is_running; then
        log::warning "OpenSCAD is not running, starting it..."
        openscad::start || return 1
    fi
    
    local script_name="${name%.scad}"
    
    if [[ -z "$output_name" ]]; then
        output_name="${script_name}"
    fi
    
    local output_file="${OPENSCAD_OUTPUT_DIR}/${output_name}.${format}"
    
    log::info "Rendering ${name} to ${format}..."
    
    if docker exec "${OPENSCAD_CONTAINER_NAME}" openscad -o "/output/${output_name}.${format}" "/scripts/${name}"; then
        log::success "Rendered successfully: ${output_file}"
        
        # Show output info
        echo ""
        echo "Output: ${output_name}.${format}"
        echo "Size: $(stat -c%s "$output_file" 2>/dev/null || stat -f%z "$output_file" 2>/dev/null) bytes"
        echo "Location: $output_file"
        
        return 0
    else
        log::error "Failed to render ${name}"
        return 1
    fi
}