#!/bin/bash
# TARS-desktop content functionality - business operations

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TARS_DESKTOP_CONTENT_DIR="${APP_ROOT}/resources/tars-desktop/lib"

# Source dependencies
source "${TARS_DESKTOP_CONTENT_DIR}/core.sh"

# Add automation script/workflow to TARS-desktop
tars_desktop::content::add() {
    local script_name="${1:-}"
    local script_content="${2:-}"
    
    if [[ -z "$script_name" || -z "$script_content" ]]; then
        log::error "Usage: add <script_name> <script_content>"
        return 1
    fi
    
    local scripts_dir="${TARS_DESKTOP_INSTALL_DIR}/scripts"
    mkdir -p "$scripts_dir"
    
    echo "$script_content" > "${scripts_dir}/${script_name}.py"
    log::success "Automation script '$script_name' added to TARS-desktop"
}

# List automation scripts
tars_desktop::content::list() {
    local scripts_dir="${TARS_DESKTOP_INSTALL_DIR}/scripts"
    
    if [[ -d "$scripts_dir" ]]; then
        find "$scripts_dir" -name "*.py" -type f -exec basename {} \; 2>/dev/null | sort
    else
        echo "No scripts directory found"
    fi
}

# Get automation script content
tars_desktop::content::get() {
    local script_name="${1:-}"
    
    if [[ -z "$script_name" ]]; then
        log::error "Usage: get <script_name>"
        return 1
    fi
    
    local scripts_dir="${TARS_DESKTOP_INSTALL_DIR}/scripts"
    local script_file="${scripts_dir}/${script_name}.py"
    
    if [[ -f "$script_file" ]]; then
        cat "$script_file"
    else
        log::error "Script '$script_name' not found"
        return 1
    fi
}

# Remove automation script
tars_desktop::content::remove() {
    local script_name="${1:-}"
    
    if [[ -z "$script_name" ]]; then
        log::error "Usage: remove <script_name>"
        return 1
    fi
    
    local scripts_dir="${TARS_DESKTOP_INSTALL_DIR}/scripts"
    local script_file="${scripts_dir}/${script_name}.py"
    
    if [[ -f "$script_file" ]]; then
        rm "$script_file"
        log::success "Script '$script_name' removed"
    else
        log::error "Script '$script_name' not found"
        return 1
    fi
}

# Execute UI automation action (business functionality)
tars_desktop::content::execute() {
    local action="${1:-}"
    local target="${2:-}"
    local data="${3:-}"
    
    if [[ -z "$action" ]]; then
        log::error "Usage: execute <action> [target] [data]"
        return 1
    fi
    
    # Check if TARS-desktop is running
    if ! tars_desktop::is_running; then
        log::error "TARS-desktop is not running. Start it first with 'manage start'"
        return 1
    fi
    
    # Execute the action through the API
    case "$action" in
        click|doubleclick|rightclick|moveto|dragto|scroll|typewrite|hotkey|screenshot)
            tars_desktop::execute_action "$action" "$target" "$data"
            ;;
        *)
            log::error "Unknown action: $action"
            log::info "Available actions: click, doubleclick, rightclick, moveto, dragto, scroll, typewrite, hotkey, screenshot"
            return 1
            ;;
    esac
}

# Export functions
export -f tars_desktop::content::add
export -f tars_desktop::content::list
export -f tars_desktop::content::get
export -f tars_desktop::content::remove
export -f tars_desktop::content::execute