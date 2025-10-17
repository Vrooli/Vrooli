#!/bin/bash
# Pushover start/stop functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
PUSHOVER_START_DIR="${APP_ROOT}/resources/pushover/lib"

# Source dependencies
source "${PUSHOVER_START_DIR}/core.sh"

# Start Pushover (activate service)
pushover::start() {
    local verbose="${1:-false}"
    
    log::header "Starting Pushover"
    
    # Initialize
    pushover::init "$verbose"
    
    # Check if already running
    if pushover::is_running; then
        log::success "Pushover is already active"
        return 0
    fi
    
    # Check if installed
    if ! pushover::is_installed; then
        log::error "Pushover is not installed. Run 'install' first"
        return 1
    fi
    
    # Check if configured
    if ! pushover::is_configured; then
        log::error "Pushover is not configured (missing credentials)"
        log::info "See 'resource-pushover install' for configuration instructions"
        return 1
    fi
    
    # Verify API connection
    if pushover::health_check "$verbose"; then
        log::success "Pushover service activated successfully"
        
        # Update config to mark as enabled
        if [[ -f "$PUSHOVER_CONFIG_FILE" ]]; then
            jq '.enabled = true' "$PUSHOVER_CONFIG_FILE" > "${PUSHOVER_CONFIG_FILE}.tmp"
            mv "${PUSHOVER_CONFIG_FILE}.tmp" "$PUSHOVER_CONFIG_FILE"
        fi
        
        return 0
    else
        log::error "Failed to activate Pushover service"
        return 1
    fi
}

# Stop Pushover (deactivate service)
pushover::stop() {
    local verbose="${1:-false}"
    
    log::header "Stopping Pushover"
    
    # Initialize
    pushover::init "$verbose"
    
    # Update config to mark as disabled
    if [[ -f "$PUSHOVER_CONFIG_FILE" ]]; then
        jq '.enabled = false' "$PUSHOVER_CONFIG_FILE" > "${PUSHOVER_CONFIG_FILE}.tmp"
        mv "${PUSHOVER_CONFIG_FILE}.tmp" "$PUSHOVER_CONFIG_FILE"
    fi
    
    log::success "Pushover service deactivated"
    return 0
}

# Send a notification
pushover::send() {
    local message=""
    local title="Vrooli Notification"
    local priority="0"
    local sound="pushover"
    local template=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -m|--message)
                message="$2"
                shift 2
                ;;
            -t|--title)
                title="$2"
                shift 2
                ;;
            -p|--priority)
                priority="$2"
                shift 2
                ;;
            -s|--sound)
                sound="$2"
                shift 2
                ;;
            --template)
                template="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Initialize
    pushover::init false
    
    # Check if service is running
    if ! pushover::is_running; then
        log::error "Pushover service is not active"
        return 1
    fi
    
    # Load template if specified
    if [[ -n "$template" ]]; then
        local template_file="${PUSHOVER_TEMPLATES_DIR}/${template}.json"
        if [[ -f "$template_file" ]]; then
            local template_data
            template_data=$(cat "$template_file")
            
            # Extract template fields
            title=$(echo "$template_data" | jq -r '.title // "Vrooli Notification"')
            priority=$(echo "$template_data" | jq -r '.priority // 0')
            sound=$(echo "$template_data" | jq -r '.sound // "pushover"')
            
            # Use template message if no message provided
            if [[ -z "$message" ]]; then
                message=$(echo "$template_data" | jq -r '.message // "Notification from Vrooli"')
            fi
        else
            log::warning "Template not found: $template"
        fi
    fi
    
    # Require message
    if [[ -z "$message" ]]; then
        log::error "Message is required"
        echo "Usage: resource-pushover send -m \"Your message\" [-t \"Title\"] [-p priority] [-s sound]"
        return 1
    fi
    
    # Send notification
    if pushover::send_notification "$message" "$title" "$priority" "$sound"; then
        log::success "Notification sent successfully"
        return 0
    else
        return 1
    fi
}