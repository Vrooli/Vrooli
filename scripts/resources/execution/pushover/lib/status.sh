#!/bin/bash
# Pushover status functionality

# Get script directory
PUSHOVER_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${PUSHOVER_STATUS_DIR}/core.sh"

# Status check
pushover::status() {
    local verbose="${1:-false}"
    local format="${2:-text}"
    
    # Initialize
    pushover::init "$verbose"
    
    # Gather status information
    local installed=$(pushover::is_installed && echo "true" || echo "false")
    local configured=$(pushover::is_configured && echo "true" || echo "false")
    local running=$(pushover::is_running && echo "true" || echo "false")
    local health="unhealthy"
    
    if [[ "$running" == "true" ]]; then
        health="healthy"
    elif [[ "$configured" == "true" ]]; then
        health="partial"
    elif [[ "$installed" == "true" ]]; then
        health="installed"
    fi
    
    # Prepare details
    local details=""
    if [[ "$configured" == "true" ]]; then
        details="API configured, ready to send notifications"
    elif [[ "$installed" == "true" ]]; then
        details="Dependencies installed, awaiting API credentials"
    else
        details="Not installed"
    fi
    
    # Format output using format.sh
    local status_data=$(cat <<EOF
{
    "name": "pushover",
    "installed": $installed,
    "running": $running,
    "health": "$health",
    "configured": $configured,
    "api_url": "$PUSHOVER_API_URL",
    "details": "$details"
}
EOF
    )
    
    # Use format.sh for consistent output
    if [[ "$format" == "json" ]]; then
        echo "$status_data"
    else
        format::status "📱 Pushover Status" "$status_data"
        
        if [[ "$verbose" == "true" ]]; then
            echo ""
            log::info "Configuration:"
            log::info "   📁 Data Directory: $PUSHOVER_DATA_DIR"
            log::info "   🔑 Credentials: ${PUSHOVER_CREDENTIALS_FILE}"
            
            if [[ "$configured" == "true" ]]; then
                log::info "   🔐 App Token: ${PUSHOVER_APP_TOKEN:0:10}..."
                log::info "   👤 User Key: ${PUSHOVER_USER_KEY:0:10}..."
            fi
            
            echo ""
            log::info "Notification Features:"
            log::info "   📨 Message Types: text, HTML, monospace"
            log::info "   🔔 Priority Levels: -2 to 2 (silent to emergency)"
            log::info "   🎵 Sounds: 21 notification sounds available"
            log::info "   📎 Attachments: Images up to 2.5MB"
            log::info "   🔗 URL Support: Clickable links in messages"
        fi
    fi
}