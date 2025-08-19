#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
TWILIO_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# Inject phone numbers or workflows
inject_data() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "Please provide a file to inject"
        echo "Usage: resource-twilio inject <file.json>"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    log::header "💉 Injecting Twilio Data"
    
    twilio::ensure_dirs
    
    # Determine file type based on content
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
        
    elif jq -e '.account_sid' "$file" >/dev/null 2>&1; then
        # Credentials file
        log::info "Injecting credentials..."
        cp "$file" "$TWILIO_CREDENTIALS_FILE"
        chmod 600 "$TWILIO_CREDENTIALS_FILE"
        log::success "Injected Twilio credentials"
        
    else
        log::error "Unknown file format"
        return 1
    fi
    
    return 0
}

twilio::inject() {
    inject_data "$@"
}

