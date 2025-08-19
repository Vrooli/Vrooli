#!/bin/bash
# Pushover status functionality

# Get script directory
PUSHOVER_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${PUSHOVER_STATUS_DIR}/core.sh"
source "${PUSHOVER_STATUS_DIR}/../../../lib/status-args.sh"
source "${PUSHOVER_STATUS_DIR}/../../../../lib/utils/format.sh"

# Collect Pushover status data in format-agnostic structure
pushover::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    pushover::init false
    
    local installed="false"
    local running="false" 
    local healthy="false"
    local health_message="Pushover not configured"
    
    if pushover::is_configured; then
        installed="true"
        running="true"
        healthy="true"
        health_message="Pushover is configured and ready"
    fi
    
    # Build status data array
    local status_data=(
        "name" "pushover"
        "category" "execution"
        "description" "Real-time notifications for Android, iPhone, iPad and Desktop"
        "installed" "$installed"
        "running" "$running"
        "healthy" "$healthy"
        "health_message" "$health_message"
        "api_url" "https://api.pushover.net/1/messages.json"
    )
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display status in text format
pushover::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📱 Pushover Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Configured: Yes"
        log::success "   ✅ Running: Yes"
        log::success "   ✅ Health: ${data[health_message]:-Healthy}"
    else
        log::error "   ❌ Configured: No"
        echo
        log::info "💡 Configuration Required:"
        log::info "   To configure Pushover, run: resource-pushover install"
        return
    fi
    echo
    
    log::info "🌐 Service Info:"
    log::info "   🔌 API: ${data[api_url]:-unknown}"
    echo
}

# New main status function using standard wrapper
pushover::status::new() {
    status::run_standard "pushover" "pushover::status::collect_data" "pushover::status::display_text" "$@"
}

# Status check
pushover::status() {
    local verbose="false"
    local format="text"
    local fast="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="${2:-text}"
                shift 2
                ;;
            --verbose)
                verbose="true"
                shift
                ;;
            --fast)
                fast="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
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