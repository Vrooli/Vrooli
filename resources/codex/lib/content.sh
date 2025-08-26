#!/usr/bin/env bash
# Codex Content Management Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_CONTENT_DIR="${APP_ROOT}/resources/codex/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CODEX_CONTENT_DIR}/common.sh"

#######################################
# Add content (Python script) to Codex
# Arguments:
#   file_path - Path to the Python script to add
#   [name] - Optional name for the script
# Returns:
#   0 on success, 1 on failure
#######################################
codex::content::add() {
    local file_path="${1:-}"
    local name="${2:-}"
    
    if [[ -z "${file_path}" ]]; then
        log::error "No file specified"
        echo "Usage: content add <file_path> [name]"
        return 1
    fi
    
    if [[ ! -f "${file_path}" ]]; then
        log::error "File not found: ${file_path}"
        return 1
    fi
    
    # Create directories if needed
    mkdir -p "${CODEX_SCRIPTS_DIR}" "${CODEX_INJECTED_DIR}"
    
    # Determine target name
    local basename
    basename=$(basename "${file_path}")
    if [[ -n "${name}" ]]; then
        # Use provided name but keep .py extension
        basename="${name}"
        if [[ ! "${basename}" =~ \.py$ ]]; then
            basename="${basename}.py"
        fi
    fi
    
    local target="${CODEX_SCRIPTS_DIR}/${basename}"
    
    log::info "Adding script: ${basename}"
    
    # Copy to scripts directory
    if cp "${file_path}" "${target}"; then
        # Also copy to injected directory for tracking
        cp "${file_path}" "${CODEX_INJECTED_DIR}/${basename}"
        
        log::success "Script added successfully: ${basename}"
        log::info "Script location: ${target}"
    else
        log::error "Failed to add script"
        return 1
    fi
    
    return 0
}

#######################################
# List all content (Python scripts)
# Arguments:
#   --format [text|json] - Output format
# Returns:
#   0 on success
#######################################
codex::content::list() {
    local format="text"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="${2:-text}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ ! -d "${CODEX_SCRIPTS_DIR}" ]]; then
        if [[ "$format" == "json" ]]; then
            echo "[]"
        else
            log::info "No scripts directory found"
        fi
        return 0
    fi
    
    local scripts
    scripts=$(find "${CODEX_SCRIPTS_DIR}" -name "*.py" -type f 2>/dev/null | sort)
    
    if [[ "$format" == "json" ]]; then
        echo "["
        local first=true
        while IFS= read -r script; do
            [[ -z "$script" ]] && continue
            if [[ "$first" == "true" ]]; then
                first=false
            else
                echo ","
            fi
            local basename size modified
            basename=$(basename "${script}")
            size=$(stat -c%s "${script}" 2>/dev/null || echo "0")
            modified=$(stat -c%y "${script}" 2>/dev/null | cut -d. -f1 || echo "unknown")
            echo -n "  {\"name\": \"${basename}\", \"size\": ${size}, \"modified\": \"${modified}\"}"
        done <<< "${scripts}"
        echo ""
        echo "]"
    else
        log::header "Codex Scripts"
        
        local count=0
        while IFS= read -r script; do
            [[ -z "$script" ]] && continue
            ((count++))
            local basename size modified
            basename=$(basename "${script}")
            size=$(stat -c%s "${script}" 2>/dev/null || echo "0")
            modified=$(stat -c%y "${script}" 2>/dev/null | cut -d. -f1 || echo "unknown")
            printf "  %2d. %-30s %6s bytes  %s\n" "${count}" "${basename}" "${size}" "${modified}"
        done <<< "${scripts}"
        
        if [[ ${count} -eq 0 ]]; then
            log::info "No scripts found"
        else
            echo ""
            log::info "Total: ${count} script(s)"
        fi
    fi
    
    return 0
}

#######################################
# Get content (show script details)
# Arguments:
#   name - Name of the script to get
# Returns:
#   0 on success, 1 if not found
#######################################
codex::content::get() {
    local name="${1:-}"
    
    if [[ -z "${name}" ]]; then
        log::error "No script name specified"
        echo "Usage: content get <script_name>"
        return 1
    fi
    
    # Add .py extension if not present
    if [[ ! "${name}" =~ \.py$ ]]; then
        name="${name}.py"
    fi
    
    local script_path="${CODEX_SCRIPTS_DIR}/${name}"
    
    if [[ ! -f "${script_path}" ]]; then
        log::error "Script not found: ${name}"
        return 1
    fi
    
    log::header "Script: ${name}"
    
    # Show metadata
    local size modified
    size=$(stat -c%s "${script_path}" 2>/dev/null || echo "0")
    modified=$(stat -c%y "${script_path}" 2>/dev/null | cut -d. -f1 || echo "unknown")
    
    echo "Location: ${script_path}"
    echo "Size: ${size} bytes"
    echo "Modified: ${modified}"
    echo ""
    echo "Content:"
    echo "--------"
    cat "${script_path}"
    
    return 0
}

#######################################
# Remove content (delete script)
# Arguments:
#   name - Name of the script to remove
# Returns:
#   0 on success, 1 if not found
#######################################
codex::content::remove() {
    local name="${1:-}"
    
    if [[ -z "${name}" ]]; then
        log::error "No script name specified"
        echo "Usage: content remove <script_name>"
        return 1
    fi
    
    # Add .py extension if not present
    if [[ ! "${name}" =~ \.py$ ]]; then
        name="${name}.py"
    fi
    
    local script_path="${CODEX_SCRIPTS_DIR}/${name}"
    local injected_path="${CODEX_INJECTED_DIR}/${name}"
    
    if [[ ! -f "${script_path}" ]]; then
        log::error "Script not found: ${name}"
        return 1
    fi
    
    log::info "Removing script: ${name}"
    
    # Remove from both locations
    rm -f "${script_path}"
    rm -f "${injected_path}"
    
    log::success "Script removed successfully: ${name}"
    return 0
}

#######################################
# Execute content (run script with Codex)
# Arguments:
#   name - Name of the script to execute
# Returns:
#   0 on success, 1 on failure
#######################################
codex::content::execute() {
    local name="${1:-}"
    
    if [[ -z "${name}" ]]; then
        log::error "No script name specified"
        echo "Usage: content execute <script_name>"
        return 1
    fi
    
    # Add .py extension if not present
    if [[ ! "${name}" =~ \.py$ ]]; then
        name="${name}.py"
    fi
    
    local script_path="${CODEX_SCRIPTS_DIR}/${name}"
    
    if [[ ! -f "${script_path}" ]]; then
        log::error "Script not found: ${name}"
        return 1
    fi
    
    # Use the existing run function from inject.sh
    if type -t codex::run &>/dev/null; then
        codex::run "${name}"
    else
        log::error "Execute function not available"
        return 1
    fi
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "$1" in
        add)
            shift
            codex::content::add "$@"
            ;;
        list)
            shift
            codex::content::list "$@"
            ;;
        get)
            shift
            codex::content::get "$@"
            ;;
        remove)
            shift
            codex::content::remove "$@"
            ;;
        execute)
            shift
            codex::content::execute "$@"
            ;;
        *)
            echo "Usage: $0 {add|list|get|remove|execute} [args...]"
            exit 1
            ;;
    esac
fi