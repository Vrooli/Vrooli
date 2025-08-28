#!/usr/bin/env bash
# LangChain Content Management Functions
# Handle content operations for chains, agents, and workflows

# Source required libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
LANGCHAIN_CONTENT_DIR="${APP_ROOT}/resources/langchain/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/langchain/config/defaults.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_CONTENT_DIR}/core.sh"

# Ensure configuration is exported
if command -v langchain::export_config &>/dev/null; then
    langchain::export_config 2>/dev/null || true
fi

#######################################
# Add content to LangChain workspace
#######################################
langchain::content::add() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required"
        echo "Usage: resource-langchain content add <file>"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Use existing inject functionality
    langchain::inject_file "$file"
}

#######################################
# List content in LangChain workspace
#######################################
langchain::content::list() {
    local content_type="${1:-all}"
    
    log::header "LangChain Content Listing"
    
    if [[ "$content_type" == "all" || "$content_type" == "chains" ]]; then
        echo
        log::info "📄 Chains:"
        if [[ -d "$LANGCHAIN_CHAINS_DIR" ]]; then
            find "$LANGCHAIN_CHAINS_DIR" -name "*.py" -o -name "*.json" | while read -r file; do
                local basename
                basename=$(basename "$file")
                echo "  • $basename"
            done
        else
            log::warn "  Chains directory not found"
        fi
    fi
    
    if [[ "$content_type" == "all" || "$content_type" == "agents" ]]; then
        echo
        log::info "🤖 Agents:"
        if [[ -d "$LANGCHAIN_AGENTS_DIR" ]]; then
            find "$LANGCHAIN_AGENTS_DIR" -name "*.py" -o -name "*.json" | while read -r file; do
                local basename
                basename=$(basename "$file")
                echo "  • $basename"
            done
        else
            log::warn "  Agents directory not found"
        fi
    fi
    
    if [[ "$content_type" == "all" || "$content_type" == "workspace" ]]; then
        echo
        log::info "📁 Workspace:"
        if [[ -d "$LANGCHAIN_WORKSPACE_DIR" ]]; then
            find "$LANGCHAIN_WORKSPACE_DIR" -name "*.py" -o -name "*.json" | while read -r file; do
                local basename
                basename=$(basename "$file")
                echo "  • $basename"
            done
        else
            log::warn "  Workspace directory not found"
        fi
    fi
    
    echo
}

#######################################
# Get specific content from workspace
#######################################
langchain::content::get() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Content name required"
        echo "Usage: resource-langchain content get <name>"
        return 1
    fi
    
    # Search for the file in all directories
    local found_files=()
    local search_dirs=("$LANGCHAIN_CHAINS_DIR" "$LANGCHAIN_AGENTS_DIR" "$LANGCHAIN_WORKSPACE_DIR")
    
    for dir in "${search_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            while IFS= read -r -d '' file; do
                found_files+=("$file")
            done < <(find "$dir" -name "*${name}*" -type f \( -name "*.py" -o -name "*.json" \) -print0 2>/dev/null)
        fi
    done
    
    if [[ ${#found_files[@]} -eq 0 ]]; then
        log::error "Content '$name' not found"
        return 1
    elif [[ ${#found_files[@]} -eq 1 ]]; then
        log::info "Found content: ${found_files[0]}"
        cat "${found_files[0]}"
    else
        log::info "Multiple matches found:"
        for file in "${found_files[@]}"; do
            echo "  • $file"
        done
        echo
        log::info "Please be more specific or use full filename"
        return 1
    fi
}

#######################################
# Remove content from workspace
#######################################
langchain::content::remove() {
    local name="${1:-}"
    local force="${2:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Content name required"
        echo "Usage: resource-langchain content remove <name> [--force]"
        return 1
    fi
    
    # Search for the file in all directories
    local found_files=()
    local search_dirs=("$LANGCHAIN_CHAINS_DIR" "$LANGCHAIN_AGENTS_DIR" "$LANGCHAIN_WORKSPACE_DIR")
    
    for dir in "${search_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            while IFS= read -r -d '' file; do
                found_files+=("$file")
            done < <(find "$dir" -name "*${name}*" -type f \( -name "*.py" -o -name "*.json" \) -print0 2>/dev/null)
        fi
    done
    
    if [[ ${#found_files[@]} -eq 0 ]]; then
        log::error "Content '$name' not found"
        return 1
    fi
    
    if [[ ${#found_files[@]} -gt 1 && "$force" != "--force" ]]; then
        log::warn "Multiple matches found:"
        for file in "${found_files[@]}"; do
            echo "  • $file"
        done
        log::warn "Use --force to remove all matches"
        return 1
    fi
    
    for file in "${found_files[@]}"; do
        log::info "Removing: $file"
        rm "$file"
        
        # Remove companion files (.json if .py removed, vice versa)
        local basename="${file%.*}"
        local companion_py="${basename}.py"
        local companion_json="${basename}.json"
        
        [[ -f "$companion_py" && "$file" != "$companion_py" ]] && rm "$companion_py" && log::info "Removed companion: $companion_py"
        [[ -f "$companion_json" && "$file" != "$companion_json" ]] && rm "$companion_json" && log::info "Removed companion: $companion_json"
    done
    
    log::success "Content removed successfully"
}

#######################################
# Execute content (run chains/agents)
#######################################
langchain::content::execute() {
    local name="${1:-}"
    shift
    
    if [[ -z "$name" ]]; then
        log::error "Content name required"
        echo "Usage: resource-langchain content execute <name> [args...]"
        return 1
    fi
    
    if ! langchain::is_installed; then
        log::error "LangChain is not installed"
        return 1
    fi
    
    # Search for executable Python files
    local found_files=()
    local search_dirs=("$LANGCHAIN_CHAINS_DIR" "$LANGCHAIN_AGENTS_DIR" "$LANGCHAIN_WORKSPACE_DIR")
    
    for dir in "${search_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            while IFS= read -r -d '' file; do
                if [[ -x "$file" ]] || [[ "$file" == *.py ]]; then
                    found_files+=("$file")
                fi
            done < <(find "$dir" -name "*${name}*.py" -type f -print0 2>/dev/null)
        fi
    done
    
    if [[ ${#found_files[@]} -eq 0 ]]; then
        log::error "Executable content '$name' not found"
        return 1
    elif [[ ${#found_files[@]} -gt 1 ]]; then
        log::info "Multiple matches found:"
        for file in "${found_files[@]}"; do
            echo "  • $file"
        done
        log::info "Please be more specific"
        return 1
    fi
    
    local script_file="${found_files[0]}"
    log::info "Executing: $(basename "$script_file")"
    
    # Execute the script with the LangChain Python environment
    "${LANGCHAIN_VENV_DIR}/bin/python" "$script_file" "$@"
}

# Export functions
export -f langchain::content::add
export -f langchain::content::list
export -f langchain::content::get
export -f langchain::content::remove
export -f langchain::content::execute