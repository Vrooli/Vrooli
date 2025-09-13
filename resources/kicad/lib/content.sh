#!/bin/bash
# KiCad Content Management Functions

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
    
    kicad::init_dirs
    local project_path="${KICAD_PROJECTS_DIR}/${item}"
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
    
    kicad::init_dirs
    local project_path="${KICAD_PROJECTS_DIR}/${item}"
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
            
        *)
            echo "Unknown operation: $operation"
            echo "Run 'resource-kicad content execute' to see available operations"
            return 1
            ;;
    esac
}