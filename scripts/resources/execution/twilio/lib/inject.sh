#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
TWILIO_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# Inject phone numbers, workflows, templates, or TwiML scripts
inject_data() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "Please provide a file to inject"
        echo "Usage: resource-twilio inject <file>"
        echo "Supported formats: .json (numbers/workflow/credentials/templates), .xml (TwiML), .txt (message template)"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    log::header "💉 Injecting Twilio Data"
    
    twilio::ensure_dirs
    
    # Create additional directories for templates and scripts
    local templates_dir="${TWILIO_CONFIG_DIR}/templates"
    local twiml_dir="${TWILIO_CONFIG_DIR}/twiml"
    mkdir -p "$templates_dir" "$twiml_dir"
    
    # Get file extension
    local ext="${file##*.}"
    local basename
    basename=$(basename "$file" ".$ext")
    
    # Handle different file types
    if [[ "$ext" == "json" ]]; then
        # Check JSON content type
        if jq -e '.numbers' "$file" >/dev/null 2>&1; then
            # Phone numbers file
            log::info "Injecting phone numbers..."
            cp "$file" "$TWILIO_PHONE_NUMBERS_FILE"
            local count
            count=$(jq '.numbers | length' "$file")
            log::success "Injected $count phone numbers"
            
        elif jq -e '.workflow' "$file" >/dev/null 2>&1; then
            # Workflow file
            log::info "Injecting workflow..."
            local name
            name=$(jq -r '.workflow.name // "workflow"' "$file")
            cp "$file" "$TWILIO_WORKFLOWS_DIR/${name}.json"
            log::success "Injected workflow: $name"
            
        elif jq -e '.templates' "$file" >/dev/null 2>&1; then
            # Message templates file
            log::info "Injecting message templates..."
            cp "$file" "$templates_dir/${basename}.json"
            local template_count
            template_count=$(jq '.templates | length' "$file")
            log::success "Injected $template_count message templates"
            
        elif jq -e '.account_sid' "$file" >/dev/null 2>&1; then
            # Credentials file
            log::info "Injecting credentials..."
            cp "$file" "$TWILIO_CREDENTIALS_FILE"
            chmod 600 "$TWILIO_CREDENTIALS_FILE"
            log::success "Injected Twilio credentials"
            
        else
            log::error "Unknown JSON format"
            return 1
        fi
        
    elif [[ "$ext" == "xml" ]]; then
        # TwiML script
        log::info "Injecting TwiML script: $basename"
        cp "$file" "$twiml_dir/${basename}.xml"
        log::success "Injected TwiML script: ${basename}.xml"
        
    elif [[ "$ext" == "txt" ]]; then
        # Plain text message template
        log::info "Injecting message template: $basename"
        cp "$file" "$templates_dir/${basename}.txt"
        log::success "Injected message template: ${basename}.txt"
        
    else
        log::error "Unsupported file format: .$ext"
        echo "Supported formats: .json, .xml (TwiML), .txt (template)"
        return 1
    fi
    
    return 0
}

# List injected templates and scripts
list_injected() {
    log::header "📋 Injected Twilio Data"
    
    local templates_dir="${TWILIO_CONFIG_DIR}/templates"
    local twiml_dir="${TWILIO_CONFIG_DIR}/twiml"
    local workflows_dir="${TWILIO_WORKFLOWS_DIR}"
    
    if [[ -d "$templates_dir" ]] && ls -A "$templates_dir" >/dev/null 2>&1; then
        log::info "Message Templates:"
        for file in "$templates_dir"/*; do
            [[ -f "$file" ]] && echo "  - $(basename "$file")"
        done
    fi
    
    if [[ -d "$twiml_dir" ]] && ls -A "$twiml_dir" >/dev/null 2>&1; then
        log::info "TwiML Scripts:"
        for file in "$twiml_dir"/*; do
            [[ -f "$file" ]] && echo "  - $(basename "$file")"
        done
    fi
    
    if [[ -d "$workflows_dir" ]] && ls -A "$workflows_dir" >/dev/null 2>&1; then
        log::info "Workflows:"
        for file in "$workflows_dir"/*.json; do
            [[ -f "$file" ]] && echo "  - $(basename "$file")"
        done
    fi
    
    if [[ -f "$TWILIO_PHONE_NUMBERS_FILE" ]]; then
        local count
        count=$(jq '.numbers | length' "$TWILIO_PHONE_NUMBERS_FILE" 2>/dev/null || echo "0")
        log::info "Phone Numbers: $count configured"
    fi
}

twilio::inject() {
    inject_data "$@"
}

twilio::list_injected() {
    list_injected "$@"
}

