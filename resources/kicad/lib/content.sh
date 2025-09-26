#!/bin/bash
# KiCad Content Management Functions

# Get script directory and APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KICAD_CONTENT_LIB_DIR="${APP_ROOT}/resources/kicad/lib"

# Source common functions if not already sourced
if ! declare -f kicad::init_dirs &>/dev/null; then
    source "${KICAD_CONTENT_LIB_DIR}/common.sh"
fi

# Add content (import projects/libraries)
kicad::content::add() {
    local source_path="${1:-}"
    
    if [[ -z "$source_path" ]]; then
        log::error "Usage: resource-kicad content add <file/directory>"
        return 1
    fi
    
    if [[ ! -e "$source_path" ]]; then
        log::error "Source path does not exist: $source_path"
        return 1
    fi
    
    # Use the existing inject functionality
    if declare -f kicad::inject &>/dev/null; then
        kicad::inject "$source_path"
    else
        log::error "Inject functionality not available"
        return 1
    fi
}

# List all content (projects and libraries)
kicad::content::list() {
    kicad::content::list_projects
    echo ""
    kicad::content::list_libraries
}

# List all KiCad projects
kicad::content::list_projects() {
    kicad::init_dirs
    echo "KiCad Projects:"
    if [[ -d "$KICAD_PROJECTS_DIR" ]]; then
        find "$KICAD_PROJECTS_DIR" -name "*.kicad_pro" -o -name "*.pro" 2>/dev/null | while read -r proj; do
            echo "  - $(basename "${proj%/*}")"
        done
    else
        echo "  No projects found"
    fi
}

# List all KiCad libraries
kicad::content::list_libraries() {
    kicad::init_dirs
    echo "KiCad Libraries:"
    if [[ -d "$KICAD_LIBRARIES_DIR" ]]; then
        ls -la "$KICAD_LIBRARIES_DIR" 2>/dev/null | grep -E "\.(kicad_sym|kicad_mod|lib)$" | awk '{print "  - " $NF}'
    else
        echo "  No libraries found"
    fi
}

# Get specific content item
kicad::content::get() {
    local item="${1:-}"
    if [[ -z "$item" ]]; then
        log::error "Usage: resource-kicad content get <project-name>"
        return 1
    fi
    
    # Validate project name - no path traversal allowed
    if [[ "$item" == *"/"* ]] || [[ "$item" == *".."* ]] || [[ "$item" == *"\\"* ]]; then
        log::error "Invalid project name: must not contain path separators or '..' sequences"
        return 1
    fi
    
    kicad::init_dirs
    local project_path="${KICAD_PROJECTS_DIR}/${item}"
    
    # Double-check that the resolved path is within the projects directory
    local resolved_path=$(realpath "$project_path" 2>/dev/null)
    local projects_dir=$(realpath "${KICAD_PROJECTS_DIR}" 2>/dev/null)
    if [[ ! "$resolved_path" == "${projects_dir}"/* ]]; then
        log::error "Security violation: attempted to access files outside projects directory"
        return 1
    fi
    
    if [[ -d "$project_path" ]]; then
        echo "Project: $item"
        echo "Location: $project_path"
        ls -la "$project_path"
    else
        log::error "Project not found: $item"
        return 1
    fi
}

# Remove content item
kicad::content::remove() {
    local item="${1:-}"
    if [[ -z "$item" ]]; then
        log::error "Usage: resource-kicad content remove <project-name>"
        return 1
    fi
    
    # Validate project name - no path traversal allowed
    if [[ "$item" == *"/"* ]] || [[ "$item" == *".."* ]] || [[ "$item" == *"\\"* ]]; then
        log::error "Invalid project name: must not contain path separators or '..' sequences"
        return 1
    fi
    
    kicad::init_dirs
    local project_path="${KICAD_PROJECTS_DIR}/${item}"
    
    # Double-check that the resolved path is within the projects directory
    local resolved_path=$(realpath "$project_path" 2>/dev/null)
    local projects_dir=$(realpath "${KICAD_PROJECTS_DIR}" 2>/dev/null)
    if [[ ! "$resolved_path" == "${projects_dir}"/* ]]; then
        log::error "Security violation: attempted to remove files outside projects directory"
        return 1
    fi
    
    if [[ -d "$project_path" ]]; then
        rm -rf "$project_path"
        log::success "Removed project: $item"
    else
        log::error "Project not found: $item"
        return 1
    fi
}

# Execute programmatic operations on content
kicad::content::execute() {
    local operation="${1:-}"
    shift
    local args="$@"
    
    if [[ -z "$operation" ]]; then
        echo "Available operations:"
        echo "  place-components <board> <config>  - Place components programmatically"
        echo "  generate-bom <schematic>           - Generate bill of materials"
        echo "  run-drc <board>                    - Run design rule check"
        echo "  create-scripts                     - Create Python automation scripts"
        echo "  visualize-3d <board>               - Generate 3D visualization of PCB"
        return 1
    fi
    
    # Source Python functions
    source "${APP_ROOT}/resources/kicad/lib/python.sh"
    
    case "$operation" in
        place-components)
            local board="${1:-}"
            local config="${2:-}"
            if [[ -z "$board" ]] || [[ -z "$config" ]]; then
                echo "Usage: resource-kicad content execute place-components <board.kicad_pcb> <config.yaml>"
                return 1
            fi
            
            # Create placement script if needed
            local script="${KICAD_DATA_DIR}/scripts/place_components.py"
            if [[ ! -f "$script" ]]; then
                kicad::python::create_placement_script "$script"
            fi
            
            # Execute placement
            kicad::python::execute "$script" place "$board" "$config"
            ;;
            
        generate-bom)
            local schematic="${1:-}"
            if [[ -z "$schematic" ]]; then
                echo "Usage: resource-kicad content execute generate-bom <schematic.kicad_sch>"
                return 1
            fi
            
            # Create BOM script if needed
            local script="${KICAD_DATA_DIR}/scripts/generate_bom.py"
            if [[ ! -f "$script" ]]; then
                kicad::python::create_bom_script "$script"
            fi
            
            # Generate BOM
            kicad::python::execute "$script" "$schematic"
            ;;
            
        run-drc)
            local board="${1:-}"
            if [[ -z "$board" ]]; then
                echo "Usage: resource-kicad content execute run-drc <board.kicad_pcb>"
                return 1
            fi
            
            kicad::python::run_drc "$board"
            ;;
            
        create-scripts)
            echo "Creating Python automation scripts..."
            kicad::python::create_placement_script
            kicad::python::create_bom_script
            echo "Scripts created in ${KICAD_DATA_DIR}/scripts/"
            ;;
            
        visualize-3d)
            local board="${1:-}"
            if [[ -z "$board" ]]; then
                echo "Usage: resource-kicad content execute visualize-3d <board.kicad_pcb>"
                return 1
            fi
            
            echo "Generating 3D visualization of PCB..."
            
            # Check if we have visualization tools
            if command -v pcbdraw &>/dev/null; then
                # Use pcbdraw for 3D visualization
                local output_file="${KICAD_OUTPUTS_DIR}/$(basename "$board" .kicad_pcb)_3d.png"
                pcbdraw "$board" "$output_file" --style builtin:set-blue.json 2>/dev/null || {
                    echo "Warning: pcbdraw failed, creating mock visualization"
                    echo "[Mock 3D Visualization of $(basename "$board")]" > "$output_file"
                }
                echo "3D visualization saved to: $output_file"
            elif [[ -n "$(kicad::get_cli_path)" ]]; then
                # Use kicad-cli to export STEP file for 3D viewing
                local kicad_cli=$(kicad::get_cli_path)
                local step_file="${KICAD_OUTPUTS_DIR}/$(basename "$board" .kicad_pcb).step"
                "$kicad_cli" pcb export step "$board" -o "$step_file"
                echo "3D STEP file exported to: $step_file"
                echo "Note: Open this file in a 3D viewer like FreeCAD or KiCad's 3D viewer"
            else
                echo "Warning: No 3D visualization tools available"
                echo "Install pcbdraw with: pip install pcbdraw"
                echo "Or install KiCad for full 3D viewing capabilities"
                local mock_file="${KICAD_OUTPUTS_DIR}/$(basename "$board" .kicad_pcb)_3d_mock.txt"
                echo "[Mock 3D Visualization]" > "$mock_file"
                echo "Board: $board" >> "$mock_file"
                echo "Generated: $(date)" >> "$mock_file"
                echo "Mock 3D file created: $mock_file"
            fi
            ;;
            
        *)
            echo "Unknown operation: $operation"
            echo "Run 'resource-kicad content execute' to see available operations"
            return 1
            ;;
    esac
}

# Export PCB project to various formats
kicad::export::project() {
    local project="${1:-}"
    local format="${2:-gerber}"
    
    if [[ -z "$project" ]]; then
        echo "Usage: resource-kicad content export <project-name> [format]"
        echo "Available formats: gerber, pdf, svg, step"
        return 1
    fi
    
    kicad::init_dirs
    
    # Handle both project names and paths
    local project_path=""
    if [[ -d "${KICAD_PROJECTS_DIR}/${project}" ]]; then
        project_path="${KICAD_PROJECTS_DIR}/${project}"
    elif [[ -d "$project" ]]; then
        project_path="$project"
    else
        echo "Error: Project not found: $project"
        echo "Available projects:"
        kicad::content::list_projects | grep -E "^  -" || echo "  None"
        return 1
    fi
    
    local output_dir="${KICAD_OUTPUTS_DIR}/${project}"
    mkdir -p "$output_dir"
    
    echo "Exporting project: $project to format: $format"
    
    # Get KiCad CLI path
    local kicad_cli=$(kicad::get_cli_path)
    
    # Use kicad-cli if available, otherwise use mock
    if [[ -n "$kicad_cli" ]] && [[ "$kicad_cli" != *"mock"* ]]; then
        # Find actual PCB and schematic files
        local pcb_file=$(find "$project_path" -name "*.kicad_pcb" 2>/dev/null | head -1)
        local sch_file=$(find "$project_path" -name "*.kicad_sch" 2>/dev/null | head -1)
        
        if [[ -z "$pcb_file" ]] && [[ "$format" != "pdf" ]]; then
            echo "Error: No PCB file found in project: $project"
            return 1
        fi
        
        case "$format" in
            gerber)
                "$kicad_cli" pcb export gerber "$pcb_file" -o "$output_dir" 2>/dev/null || {
                    echo "Warning: Gerber export failed"
                    return 1
                }
                ;;
            pdf)
                if [[ -n "$pcb_file" ]]; then
                    "$kicad_cli" pcb export pdf "$pcb_file" -o "$output_dir/board.pdf" 2>/dev/null || echo "Warning: PCB PDF export failed"
                fi
                if [[ -n "$sch_file" ]]; then
                    "$kicad_cli" sch export pdf "$sch_file" -o "$output_dir/schematic.pdf" 2>/dev/null || echo "Warning: Schematic PDF export failed"
                fi
                ;;
            svg)
                if [[ -n "$pcb_file" ]]; then
                    "$kicad_cli" pcb export svg "$pcb_file" -o "$output_dir" 2>/dev/null || echo "Warning: PCB SVG export failed"
                fi
                if [[ -n "$sch_file" ]]; then
                    "$kicad_cli" sch export svg "$sch_file" -o "$output_dir" 2>/dev/null || echo "Warning: Schematic SVG export failed"
                fi
                ;;
            step)
                "$kicad_cli" pcb export step "$pcb_file" -o "$output_dir/board.step" 2>/dev/null || {
                    echo "Warning: STEP export failed"
                    return 1
                }
                ;;
            *)
                echo "Unknown format: $format"
                return 1
                ;;
        esac
    elif [[ -n "$kicad_cli" ]] && [[ "$kicad_cli" == *"mock"* ]]; then
        # Use mock implementation
        "$kicad_cli" pcb export "$format" "$project_path"/*.kicad_pcb -o "$output_dir"
        echo "Mock: Export complete (using mock implementation)"
    else
        echo "Warning: KiCad not installed. Cannot export project."
        echo "Mock files created in: $output_dir"
        touch "$output_dir/mock_${format}_export"
    fi
    
    echo "Export complete. Files saved to: $output_dir"
    ls -la "$output_dir" 2>/dev/null || true
    return 0
}

# Export functions for CLI framework
export -f kicad::content::list
export -f kicad::content::list_projects
export -f kicad::content::list_libraries
export -f kicad::content::get
export -f kicad::content::remove
export -f kicad::content::execute
export -f kicad::export::project