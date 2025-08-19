#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
CLINE_INJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$CLINE_INJECT_DIR/common.sh"

# Inject configuration or agent definition into Cline
cline::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "Usage: resource-cline inject <file>"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    log::info "Injecting configuration from: $file"
    
    # Ensure directories exist
    mkdir -p "$CLINE_DATA_DIR"
    mkdir -p "$CLINE_CONFIG_DIR"
    
    # Detect file type
    local filename=$(basename "$file")
    local extension="${filename##*.}"
    
    case "$extension" in
        json)
            # Assume it's a configuration file
            if jq empty "$file" 2>/dev/null; then
                # Valid JSON, copy to config directory
                cp "$file" "$CLINE_DATA_DIR/$(basename "$file")"
                log::success "Configuration injected: $(basename "$file")"
                
                # If it contains API settings, update provider
                if jq -e '.apiProvider' "$file" >/dev/null 2>&1; then
                    local provider=$(jq -r '.apiProvider' "$file")
                    echo "$provider" > "$CLINE_CONFIG_DIR/provider.conf"
                    log::info "Updated provider to: $provider"
                fi
            else
                log::error "Invalid JSON file"
                return 1
            fi
            ;;
        py|js|ts)
            # Code files - save as templates
            mkdir -p "$CLINE_DATA_DIR/templates"
            cp "$file" "$CLINE_DATA_DIR/templates/"
            log::success "Template injected: $(basename "$file")"
            ;;
        md|txt)
            # Documentation/prompts
            mkdir -p "$CLINE_DATA_DIR/docs"
            cp "$file" "$CLINE_DATA_DIR/docs/"
            log::success "Documentation injected: $(basename "$file")"
            ;;
        *)
            log::warn "Unknown file type: $extension"
            cp "$file" "$CLINE_DATA_DIR/"
            log::success "File injected: $(basename "$file")"
            ;;
    esac
    
    return 0
}

# Main - only run if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cline::inject "$@"
fi