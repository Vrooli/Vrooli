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