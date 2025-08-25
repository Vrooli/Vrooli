#!/bin/bash
# Pushover injection functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
PUSHOVER_INJECT_DIR="${APP_ROOT}/resources/pushover/lib"

# Source dependencies
source "${PUSHOVER_INJECT_DIR}/core.sh"

# Inject notification templates or configurations
pushover::inject() {
    local file_path="${1:-}"
    local verbose="${2:-false}"
    
    log::header "Injecting Pushover Template"
    
    # Initialize
    pushover::init "$verbose"
    
    # Check if file provided
    if [[ -z "$file_path" ]]; then
        log::error "No file path provided"
        echo "Usage: resource-pushover inject <template.json>"
        return 1
    fi
    
    # Check if file exists
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    # Get filename without path
    local filename=$(basename "$file_path")
    local template_name="${filename%.json}"
    
    # Validate JSON
    if ! jq '.' "$file_path" >/dev/null 2>&1; then
        log::error "Invalid JSON in file: $file_path"
        return 1
    fi
    
    # Copy to templates directory
    local target_path="${PUSHOVER_TEMPLATES_DIR}/${filename}"
    cp "$file_path" "$target_path"
    
    if [[ $? -eq 0 ]]; then
        log::success "Template injected: $template_name"
        
        if [[ "$verbose" == "true" ]]; then
            log::info "Template details:"
            jq '.' "$target_path"
        fi
        
        return 0
    else
        log::error "Failed to inject template"
        return 1
    fi
}

# List available templates
pushover::list_templates() {
    local verbose="${1:-false}"
    
    log::header "Available Pushover Templates"
    
    # Initialize
    pushover::init "$verbose"
    
    if [[ ! -d "$PUSHOVER_TEMPLATES_DIR" ]]; then
        log::info "No templates directory found"
        return 0
    fi
    
    local template_count=0
    for template_file in "${PUSHOVER_TEMPLATES_DIR}"/*.json; do
        if [[ -f "$template_file" ]]; then
            local template_name=$(basename "$template_file" .json)
            local title=$(jq -r '.title // "N/A"' "$template_file" 2>/dev/null)
            local priority=$(jq -r '.priority // 0' "$template_file" 2>/dev/null)
            local sound=$(jq -r '.sound // "pushover"' "$template_file" 2>/dev/null)
            
            echo ""
            log::info "Template: $template_name"
            echo "   Title: $title"
            echo "   Priority: $priority"
            echo "   Sound: $sound"
            
            if [[ "$verbose" == "true" ]]; then
                echo "   Message:"
                jq -r '.message' "$template_file" 2>/dev/null | sed 's/^/      /'
            fi
            
            ((template_count++))
        fi
    done
    
    if [[ $template_count -eq 0 ]]; then
        log::info "No templates found"
    else
        echo ""
        log::success "Found $template_count template(s)"
    fi
    
    return 0
}