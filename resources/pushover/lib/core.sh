#!/bin/bash
# Pushover core functionality

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
PUSHOVER_CORE_DIR="${APP_ROOT}/resources/pushover/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Configuration
PUSHOVER_API_URL="https://api.pushover.net/1"
PUSHOVER_DATA_DIR="${var_DATA_DIR:-${HOME}/.vrooli}/resources/pushover"
PUSHOVER_CONFIG_FILE="${PUSHOVER_DATA_DIR}/config.json"
PUSHOVER_CREDENTIALS_FILE="${var_DATA_DIR:-${HOME}/.vrooli}/resources/pushover-credentials.json"
PUSHOVER_TEMPLATES_DIR="${PUSHOVER_DATA_DIR}/templates"

# Initialize Pushover
pushover::init() {
    local verbose="${1:-false}"
    
    # Create data directory
    mkdir -p "$PUSHOVER_DATA_DIR"
    mkdir -p "$PUSHOVER_TEMPLATES_DIR"
    
    # Initialize credential variables
    export PUSHOVER_APP_TOKEN="${PUSHOVER_APP_TOKEN:-}"
    export PUSHOVER_USER_KEY="${PUSHOVER_USER_KEY:-}"
    export PUSHOVER_DEMO_MODE="${PUSHOVER_DEMO_MODE:-false}"
    
    # Check for demo mode flag file
    if [[ -f "${PUSHOVER_DATA_DIR}/demo-mode.flag" ]]; then
        export PUSHOVER_DEMO_MODE="true"
        export PUSHOVER_APP_TOKEN="DEMO_TOKEN"
        export PUSHOVER_USER_KEY="DEMO_USER"
        if [[ "$verbose" == "true" ]]; then
            log::info "Running in demo mode"
        fi
        return 0
    fi
    
    # Load credentials if available
    if [[ -f "$PUSHOVER_CREDENTIALS_FILE" ]]; then
        export PUSHOVER_APP_TOKEN=$(jq -r '.app_token // empty' "$PUSHOVER_CREDENTIALS_FILE" 2>/dev/null || echo "")
        export PUSHOVER_USER_KEY=$(jq -r '.user_key // empty' "$PUSHOVER_CREDENTIALS_FILE" 2>/dev/null || echo "")
    fi
    
    # Check Vault for credentials if not in file
    if [[ -z "$PUSHOVER_APP_TOKEN" ]] || [[ -z "$PUSHOVER_USER_KEY" ]]; then
        pushover::load_vault_credentials "$verbose"
    fi
}

# Load credentials from Vault
pushover::load_vault_credentials() {
    local verbose="${1:-false}"
    
    # Check if vault is available
    if command -v resource-vault >/dev/null 2>&1; then
        local vault_status
        vault_status=$(resource-vault status --format json 2>/dev/null)
        
        if echo "$vault_status" | jq -e '.unsealed == true' >/dev/null 2>&1; then
            # Try to get credentials from vault
            local vault_data
            vault_data=$(resource-vault get pushover 2>/dev/null)
            
            if [[ -n "$vault_data" ]]; then
                export PUSHOVER_APP_TOKEN=$(echo "$vault_data" | jq -r '.app_token // empty' 2>/dev/null || echo "")
                export PUSHOVER_USER_KEY=$(echo "$vault_data" | jq -r '.user_key // empty' 2>/dev/null || echo "")
                
                if [[ "$verbose" == "true" ]]; then
                    log::info "Loaded credentials from Vault"
                fi
            fi
        fi
    fi
}

# Check if Pushover is installed
pushover::is_installed() {
    # Check if Python requests is available
    python3 -c "import requests" 2>/dev/null
}

# Check if Pushover is configured
pushover::is_configured() {
    [[ -n "${PUSHOVER_APP_TOKEN:-}" ]] && [[ -n "${PUSHOVER_USER_KEY:-}" ]]
}

# Check if Pushover is "running" (service is active)
pushover::is_running() {
    # Pushover is a cloud service, so "running" means configured and reachable
    pushover::is_configured && pushover::health_check >/dev/null 2>&1
}

# Health check
pushover::health_check() {
    local verbose="${1:-false}"
    
    if ! pushover::is_configured; then
        if [[ "$verbose" == "true" ]]; then
            log::error "Pushover is not configured (missing credentials)"
        fi
        return 1
    fi
    
    # In demo mode, simulate successful health check
    if [[ "$PUSHOVER_DEMO_MODE" == "true" ]]; then
        if [[ "$verbose" == "true" ]]; then
            log::success "Pushover is running in demo mode (simulated health check)"
        fi
        return 0
    fi
    
    # Verify credentials with API
    local response
    response=$(curl -s -X POST \
        -F "token=${PUSHOVER_APP_TOKEN}" \
        -F "user=${PUSHOVER_USER_KEY}" \
        "${PUSHOVER_API_URL}/users/validate.json" 2>/dev/null)
    
    if echo "$response" | jq -e '.status == 1' >/dev/null 2>&1; then
        if [[ "$verbose" == "true" ]]; then
            log::success "Pushover API is accessible and credentials are valid"
        fi
        return 0
    else
        if [[ "$verbose" == "true" ]]; then
            log::error "Pushover API validation failed"
            echo "$response" | jq '.' 2>/dev/null
        fi
        return 1
    fi
}

# Send notification
pushover::send_notification() {
    local message="$1"
    local title="${2:-Vrooli Notification}"
    local priority="${3:-0}"
    local sound="${4:-pushover}"
    
    if ! pushover::is_configured; then
        log::error "Pushover is not configured"
        return 1
    fi
    
    # In demo mode, simulate successful send
    if [[ "$PUSHOVER_DEMO_MODE" == "true" ]]; then
        log::info "[DEMO MODE] Would send notification:"
        log::info "  Title: $title"
        log::info "  Message: $message"
        log::info "  Priority: $priority"
        log::info "  Sound: $sound"
        return 0
    fi
    
    local response
    response=$(curl -s -X POST \
        -F "token=${PUSHOVER_APP_TOKEN}" \
        -F "user=${PUSHOVER_USER_KEY}" \
        -F "message=${message}" \
        -F "title=${title}" \
        -F "priority=${priority}" \
        -F "sound=${sound}" \
        "${PUSHOVER_API_URL}/messages.json" 2>/dev/null)
    
    if echo "$response" | jq -e '.status == 1' >/dev/null 2>&1; then
        return 0
    else
        log::error "Failed to send notification"
        echo "$response" | jq '.' 2>/dev/null
        return 1
    fi
}

# Restart Pushover service (stop and start)
pushover::restart() {
    log::info "Restarting Pushover service..."
    pushover::stop
    sleep 1
    pushover::start
}

# Show logs (simulated for cloud service)
pushover::show_logs() {
    local lines="${1:-50}"
    log::info "Pushover is a cloud service - showing recent activity..."
    
    # If there's a local log file, show it
    local log_file="${PUSHOVER_DATA_DIR}/activity.log"
    if [[ -f "$log_file" ]]; then
        tail -n "$lines" "$log_file"
    else
        log::info "No local activity logs found"
        log::info "API activity can be viewed at https://pushover.net"
    fi
}

# List templates
pushover::list_templates() {
    pushover::init false
    
    if [[ ! -d "$PUSHOVER_TEMPLATES_DIR" ]]; then
        log::info "No templates directory found"
        return 1
    fi
    
    log::header "Available Templates"
    for template in "$PUSHOVER_TEMPLATES_DIR"/*.json; do
        if [[ -f "$template" ]]; then
            local name=$(basename "$template" .json)
            local title=$(jq -r '.title // "N/A"' "$template" 2>/dev/null)
            echo "- $name: $title"
        fi
    done
}

# Get template details
pushover::get_template() {
    local template_name="${1:-}"
    
    if [[ -z "$template_name" ]]; then
        log::error "Template name required"
        return 1
    fi
    
    pushover::init false
    local template_file="${PUSHOVER_TEMPLATES_DIR}/${template_name}.json"
    
    if [[ ! -f "$template_file" ]]; then
        log::error "Template not found: $template_name"
        return 1
    fi
    
    log::header "Template: $template_name"
    jq '.' "$template_file"
}

# Remove template
pushover::remove_template() {
    local template_name="${1:-}"
    
    if [[ -z "$template_name" ]]; then
        log::error "Template name required"
        return 1
    fi
    
    pushover::init false
    local template_file="${PUSHOVER_TEMPLATES_DIR}/${template_name}.json"
    
    if [[ ! -f "$template_file" ]]; then
        log::error "Template not found: $template_name"
        return 1
    fi
    
    rm -f "$template_file"
    log::success "Template removed: $template_name"
}

# Test functions
pushover::test::smoke() {
    log::header "Running Pushover smoke test"
    
    pushover::init false
    
    # Check if installed
    if ! pushover::is_installed; then
        log::error "Pushover is not installed"
        return 1
    fi
    
    # Check if configured
    if ! pushover::is_configured; then
        log::warning "Pushover is not configured - running in limited mode"
    fi
    
    log::success "Smoke test passed"
    return 0
}

pushover::test::integration() {
    log::header "Running Pushover integration test"
    
    pushover::init false
    
    # Check installation
    if ! pushover::is_installed; then
        log::error "Pushover is not installed"
        return 1
    fi
    
    # Check configuration
    if ! pushover::is_configured; then
        log::error "Pushover is not configured - cannot run integration test"
        log::info "Configure with: resource-pushover configure"
        return 1
    fi
    
    # Health check
    if ! pushover::health_check true; then
        log::error "Health check failed"
        return 1
    fi
    
    # Test send in demo mode (non-destructive)
    export PUSHOVER_DEMO_MODE="true"
    if pushover::send_notification "Integration test message" "Test" "0" "pushover"; then
        log::success "Test notification sent (demo mode)"
    else
        log::error "Failed to send test notification"
        return 1
    fi
    export PUSHOVER_DEMO_MODE="false"
    
    log::success "Integration test passed"
    return 0
}