#!/bin/bash
# KiCad Injection Functions

# Get script directory
KICAD_INJECT_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "${KICAD_INJECT_LIB_DIR}/common.sh"

# Inject KiCad files (projects, libraries, templates)
kicad::inject() {
    local source_path="${1:-}"
    local target_type="${2:-auto}"  # auto, project, library, template
    
    if [[ -z "$source_path" ]]; then
        echo "Error: Source path required"
        echo "Usage: resource-kicad inject <file/directory> [type]"
        return 1
    fi
    
    if [[ ! -e "$source_path" ]]; then
        echo "Error: Source path does not exist: $source_path"
        return 1
    fi
    
    # Initialize directories
    kicad::init_dirs
    
    # Auto-detect type if not specified
    if [[ "$target_type" == "auto" ]]; then
        if [[ -f "$source_path" ]]; then
            case "$source_path" in
                *.kicad_pro|*.kicad_pcb|*.kicad_sch)
                    target_type="project"
                    ;;
                *.kicad_sym|*.kicad_mod|*.lib|*.dcm)
                    target_type="library"
                    ;;
                *.kicad_wks)
                    target_type="template"
                    ;;
                *)
                    echo "Warning: Could not auto-detect file type, treating as project"
                    target_type="project"
                    ;;
            esac
        elif [[ -d "$source_path" ]]; then
            # Check directory contents to determine type
            if find "$source_path" -name "*.kicad_pro" -o -name "*.kicad_pcb" | head -1 | grep -q .; then
                target_type="project"
            elif find "$source_path" -name "*.kicad_sym" -o -name "*.kicad_mod" | head -1 | grep -q .; then
                target_type="library"
            else
                target_type="project"
            fi
        fi
    fi
    
    # Perform injection based on type
    case "$target_type" in
        project)
            kicad::inject::project "$source_path"
            ;;
        library)
            kicad::inject::library "$source_path"
            ;;
        template)
            kicad::inject::template "$source_path"
            ;;
        *)
            echo "Error: Unknown injection type: $target_type"
            return 1
            ;;
    esac
}

# Inject a KiCad project
kicad::inject::project() {
    local source_path="$1"
    local project_name
    
    if [[ -f "$source_path" ]]; then
        # Single file - create minimal project structure
        project_name="$(basename "$source_path" | sed 's/\.[^.]*$//')"
        local target_dir="${KICAD_PROJECTS_DIR}/${project_name}"
        
        mkdir -p "$target_dir"
        cp "$source_path" "$target_dir/"
        
        # Create basic project file if it doesn't exist
        if [[ ! -f "$target_dir/${project_name}.kicad_pro" ]]; then
            cp "${KICAD_TEMPLATES_DIR}/basic_template.kicad_pro" "$target_dir/${project_name}.kicad_pro" 2>/dev/null || true
        fi
        
    elif [[ -d "$source_path" ]]; then
        # Directory - copy entire project
        project_name="$(basename "$source_path")"
        local target_dir="${KICAD_PROJECTS_DIR}/${project_name}"
        
        if [[ -d "$target_dir" ]]; then
            echo "Warning: Project already exists: $project_name"
            echo "Backing up existing project..."
            mv "$target_dir" "${target_dir}.backup.$(date +%Y%m%d_%H%M%S)"
        fi
        
        cp -r "$source_path" "$target_dir"
    fi
    
    echo "✅ Project injected: $project_name"
    echo "   Location: ${KICAD_PROJECTS_DIR}/${project_name}"
    
    # Generate outputs if KiCad is installed
    if kicad::is_installed; then
        echo "Generating initial outputs..."
        kicad::export::project "$project_name" || true
    fi
    
    return 0
}

# Inject KiCad libraries
kicad::inject::library() {
    local source_path="$1"
    local library_name
    
    if [[ -f "$source_path" ]]; then
        # Single library file
        library_name="$(basename "$source_path")"
        cp "$source_path" "${KICAD_LIBRARIES_DIR}/"
        echo "✅ Library injected: $library_name"
        
    elif [[ -d "$source_path" ]]; then
        # Directory of libraries
        library_name="$(basename "$source_path")"
        local target_dir="${KICAD_LIBRARIES_DIR}/${library_name}"
        
        mkdir -p "$target_dir"
        cp -r "$source_path"/* "$target_dir/"
        echo "✅ Library collection injected: $library_name"
    fi
    
    echo "   Location: ${KICAD_LIBRARIES_DIR}"
    return 0
}

# Inject KiCad templates
kicad::inject::template() {
    local source_path="$1"
    local template_name
    
    if [[ -f "$source_path" ]]; then
        template_name="$(basename "$source_path")"
        cp "$source_path" "${KICAD_TEMPLATES_DIR}/"
        echo "✅ Template injected: $template_name"
        
    elif [[ -d "$source_path" ]]; then
        template_name="$(basename "$source_path")"
        cp -r "$source_path" "${KICAD_TEMPLATES_DIR}/"
        echo "✅ Template directory injected: $template_name"
    fi
    
    echo "   Location: ${KICAD_TEMPLATES_DIR}"
    return 0
}

# Export project to various formats
kicad::export::project() {
    local project_name="$1"
    local formats="${2:-$KICAD_EXPORT_FORMATS}"
    
    local project_dir="${KICAD_PROJECTS_DIR}/${project_name}"
    local output_dir="${KICAD_OUTPUTS_DIR}/${project_name}"
    
    if [[ ! -d "$project_dir" ]]; then
        echo "Error: Project not found: $project_name"
        return 1
    fi
    
    mkdir -p "$output_dir"
    
    # Find PCB file
    local pcb_file
    pcb_file=$(find "$project_dir" -name "*.kicad_pcb" | head -1)
    
    if [[ -n "$pcb_file" ]] && command -v kicad-cli &>/dev/null; then
        for format in $(echo "$formats" | tr ',' ' '); do
            case "$format" in
                gerber)
                    echo "Exporting Gerber files..."
                    kicad-cli pcb export gerbers "$pcb_file" -o "$output_dir/gerbers" 2>/dev/null || true
                    ;;
                pdf)
                    echo "Exporting PDF..."
                    kicad-cli pcb export pdf "$pcb_file" -o "$output_dir/${project_name}.pdf" 2>/dev/null || true
                    ;;
                svg)
                    echo "Exporting SVG..."
                    kicad-cli pcb export svg "$pcb_file" -o "$output_dir" 2>/dev/null || true
                    ;;
                step)
                    echo "Exporting 3D STEP model..."
                    kicad-cli pcb export step "$pcb_file" -o "$output_dir/${project_name}.step" 2>/dev/null || true
                    ;;
            esac
        done
    fi
    
    return 0
}

# Export functions
export -f kicad::inject
export -f kicad::inject::project
export -f kicad::inject::library
export -f kicad::inject::template
export -f kicad::export::project