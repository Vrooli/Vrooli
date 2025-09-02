#!/bin/bash
# TARS-desktop core functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TARS_DESKTOP_CORE_DIR="${APP_ROOT}/resources/tars-desktop/lib"
TARS_DESKTOP_RESOURCE_DIR="${APP_ROOT}/resources/tars-desktop"

# Source dependencies (disable strict error handling that var.sh sets)
source "${APP_ROOT}/scripts/lib/utils/var.sh"
set +e  # Disable strict error handling after var.sh
source "${TARS_DESKTOP_RESOURCE_DIR}/config/defaults.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# Resource registry not currently used - removed to avoid sourcing non-existent file
# source "${TARS_DESKTOP_RESOURCE_DIR}/../../lib/resource-registry.sh"

# Initialize TARS-desktop
tars_desktop::init() {
    local verbose="${1:-false}"
    
    # Resource registration commented out - registry not currently implemented
    # resource_registry::register "$TARS_DESKTOP_NAME" "$TARS_DESKTOP_CATEGORY"
    
    # Register port
    if [[ -f "${var_ROOT_DIR}/scripts/resources/port_registry.sh" ]]; then
        source "${var_ROOT_DIR}/scripts/resources/port_registry.sh"
        port_registry::register "tars-desktop" "$TARS_DESKTOP_PORT" "TARS Desktop UI Automation API"
    fi
    
    # Create necessary directories
    mkdir -p "$TARS_DESKTOP_INSTALL_DIR"
    mkdir -p "$TARS_DESKTOP_CACHE_DIR"
    
    # Check for API key from environment or Vault
    if [[ -z "$TARS_DESKTOP_API_KEY" ]]; then
        # Try to get OpenRouter key since TARS can use it
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
            local vault_key
            vault_key=$(docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv get -field=api_key secret/vrooli/openrouter 2>/dev/null" || true)
            if [[ -n "$vault_key" && "$vault_key" != "No value found at secret/vrooli/openrouter" && "$vault_key" != "sk-placeholder-key" ]]; then
                export TARS_DESKTOP_API_KEY="$vault_key"
                [[ "$verbose" == "true" ]] && log::info "Using OpenRouter API key from Vault for TARS"
            fi
        fi
    fi
    
    return 0
}

# Check if TARS-desktop is installed
tars_desktop::is_installed() {
    [[ -d "$TARS_DESKTOP_INSTALL_DIR" && -f "${TARS_DESKTOP_INSTALL_DIR}/server.py" ]]
}

# Check if TARS-desktop is running
tars_desktop::is_running() {
    # Check if the Python process is running
    if pgrep -f "tars_desktop.*server" >/dev/null 2>&1; then
        return 0
    fi
    
    # Also check if API is responding
    if curl -s -f "${TARS_DESKTOP_API_BASE}/health" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

# Get TARS-desktop health status
tars_desktop::health_check() {
    local timeout="${1:-$TARS_DESKTOP_HEALTH_CHECK_TIMEOUT}"
    
    if ! tars_desktop::is_running; then
        return 1
    fi
    
    # Check API health endpoint
    local response
    response=$(timeout "$timeout" curl -s "${TARS_DESKTOP_API_BASE}/health" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # Check for success in response
    if echo "$response" | grep -q '"status".*"ok"'; then
        return 0
    fi
    
    return 1
}

# Get TARS-desktop capabilities
tars_desktop::get_capabilities() {
    local timeout="${1:-$TARS_DESKTOP_TIMEOUT}"
    
    timeout "$timeout" curl -s "${TARS_DESKTOP_API_BASE}/capabilities" 2>/dev/null | \
        jq '.' 2>/dev/null || echo '{"error": "Failed to get capabilities"}'
}

# Execute UI action
tars_desktop::execute_action() {
    local action="$1"
    local target="$2"
    local timeout="${3:-$TARS_DESKTOP_TIMEOUT}"
    
    local payload
    payload=$(jq -n \
        --arg action "$action" \
        --arg target "$target" \
        '{action: $action, target: $target}')
    
    timeout "$timeout" curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${TARS_DESKTOP_API_KEY}" \
        -d "$payload" \
        "${TARS_DESKTOP_API_BASE}/execute" 2>/dev/null
}

# Take screenshot
tars_desktop::screenshot() {
    local output_path="${1:-/tmp/tars-screenshot.png}"
    local timeout="${2:-$TARS_DESKTOP_TIMEOUT}"
    
    timeout "$timeout" curl -s -X GET \
        -H "Authorization: Bearer ${TARS_DESKTOP_API_KEY}" \
        "${TARS_DESKTOP_API_BASE}/screenshot" \
        -o "$output_path" 2>/dev/null
    
    if [[ -f "$output_path" ]]; then
        echo "Screenshot saved to: $output_path"
        return 0
    else
        echo "Failed to capture screenshot"
        return 1
    fi
}

# Export functions
export -f tars_desktop::init
export -f tars_desktop::is_installed
export -f tars_desktop::is_running
export -f tars_desktop::health_check
export -f tars_desktop::get_capabilities
export -f tars_desktop::execute_action
export -f tars_desktop::screenshot