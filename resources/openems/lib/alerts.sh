#!/bin/bash
# OpenEMS Alert Automation - P2 Feature
# Implements alert automation through Pushover/Twilio integration

set -uo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENEMS_DIR="${APP_ROOT}/resources/openems"
OPENEMS_LIB_DIR="${OPENEMS_DIR}/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${OPENEMS_LIB_DIR}/core.sh"

# Alert configuration
OPENEMS_ALERTS_CONFIG="${OPENEMS_DATA_DIR}/alerts-config.json"
OPENEMS_ALERT_HISTORY="${OPENEMS_DATA_DIR}/alert-history.json"

################################################################################
# Alert Management Functions
################################################################################

# Initialize alert system
openems::alerts::init() {
    log::info "Initializing OpenEMS alert system..."
    
    # Create default config if not exists
    if [[ ! -f "$OPENEMS_ALERTS_CONFIG" ]]; then
        cat > "$OPENEMS_ALERTS_CONFIG" << 'EOF'
{
    "enabled": true,
    "channels": {
        "pushover": {
            "enabled": false,
            "priority": "normal"
        },
        "twilio": {
            "enabled": false,
            "sms": true,
            "voice": false
        }
    },
    "rules": [
        {
            "id": "grid-outage",
            "name": "Grid Outage Detection",
            "condition": "grid_status == 'disconnected'",
            "severity": "critical",
            "message": "Grid power lost. System running on battery/solar.",
            "channels": ["pushover", "twilio"],
            "cooldown": 300
        },
        {
            "id": "battery-low",
            "name": "Battery Low",
            "condition": "battery_soc < 20",
            "severity": "warning",
            "message": "Battery state of charge below 20%",
            "channels": ["pushover"],
            "cooldown": 600
        },
        {
            "id": "solar-offline",
            "name": "Solar Generation Offline",
            "condition": "solar_power == 0 && daylight == true",
            "severity": "warning",
            "message": "Solar panels not generating power during daylight",
            "channels": ["pushover"],
            "cooldown": 900
        },
        {
            "id": "peak-demand",
            "name": "Peak Demand Alert",
            "condition": "grid_import > peak_threshold",
            "severity": "info",
            "message": "Grid import exceeding peak threshold",
            "channels": ["pushover"],
            "cooldown": 1800
        },
        {
            "id": "battery-fault",
            "name": "Battery Fault",
            "condition": "battery_fault == true",
            "severity": "critical",
            "message": "Battery system fault detected",
            "channels": ["pushover", "twilio"],
            "cooldown": 60
        }
    ],
    "thresholds": {
        "peak_threshold": 10000,
        "battery_min_soc": 20,
        "battery_max_temp": 45
    }
}
EOF
        log::info "Created default alert configuration"
    fi
    
    # Initialize alert history
    if [[ ! -f "$OPENEMS_ALERT_HISTORY" ]]; then
        echo "[]" > "$OPENEMS_ALERT_HISTORY"
    fi
    
    return 0
}

# Check if alert automation is enabled
openems::alerts::is_enabled() {
    [[ -f "$OPENEMS_ALERTS_CONFIG" ]] && \
    jq -r '.enabled' "$OPENEMS_ALERTS_CONFIG" 2>/dev/null | grep -q "true"
}

# Configure alert channels
openems::alerts::configure() {
    local channel="${1:-}"
    local enabled="${2:-true}"
    
    if [[ -z "$channel" ]]; then
        log::error "Channel name required"
        return 1
    fi
    
    # Update configuration
    local temp_config=$(mktemp)
    jq --arg channel "$channel" --argjson enabled "$enabled" \
        '.channels[$channel].enabled = $enabled' \
        "$OPENEMS_ALERTS_CONFIG" > "$temp_config"
    
    mv "$temp_config" "$OPENEMS_ALERTS_CONFIG"
    log::info "Alert channel '$channel' configured: enabled=$enabled"
    
    return 0
}

# Send alert through Pushover
openems::alerts::send_pushover() {
    local title="$1"
    local message="$2"
    local priority="${3:-0}"
    
    # Check if Pushover is available and configured
    if ! command -v resource-pushover &>/dev/null; then
        log::debug "Pushover resource not available"
        return 1
    fi
    
    # Check if Pushover is enabled in config
    local pushover_enabled=$(jq -r '.channels.pushover.enabled' "$OPENEMS_ALERTS_CONFIG" 2>/dev/null)
    if [[ "$pushover_enabled" != "true" ]]; then
        log::debug "Pushover alerts disabled"
        return 0
    fi
    
    # Send notification
    log::info "Sending Pushover alert: $title"
    resource-pushover send \
        --title "$title" \
        --message "$message" \
        --priority "$priority" 2>/dev/null || {
        log::warn "Failed to send Pushover notification"
        return 1
    }
    
    return 0
}

# Send alert through Twilio
openems::alerts::send_twilio() {
    local title="$1"
    local message="$2"
    local type="${3:-sms}"
    
    # Check if Twilio is available and configured
    if ! command -v resource-twilio &>/dev/null; then
        log::debug "Twilio resource not available"
        return 1
    fi
    
    # Check if Twilio is enabled in config
    local twilio_enabled=$(jq -r '.channels.twilio.enabled' "$OPENEMS_ALERTS_CONFIG" 2>/dev/null)
    if [[ "$twilio_enabled" != "true" ]]; then
        log::debug "Twilio alerts disabled"
        return 0
    fi
    
    # Combine title and message
    local full_message="OpenEMS Alert: $title - $message"
    
    # Send notification based on type
    if [[ "$type" == "sms" ]]; then
        local sms_enabled=$(jq -r '.channels.twilio.sms' "$OPENEMS_ALERTS_CONFIG" 2>/dev/null)
        if [[ "$sms_enabled" == "true" ]]; then
            log::info "Sending SMS alert via Twilio"
            resource-twilio sms send --message "$full_message" 2>/dev/null || {
                log::warn "Failed to send SMS notification"
                return 1
            }
        fi
    elif [[ "$type" == "voice" ]]; then
        local voice_enabled=$(jq -r '.channels.twilio.voice' "$OPENEMS_ALERTS_CONFIG" 2>/dev/null)
        if [[ "$voice_enabled" == "true" ]]; then
            log::info "Initiating voice alert via Twilio"
            resource-twilio voice call --message "$full_message" 2>/dev/null || {
                log::warn "Failed to initiate voice call"
                return 1
            }
        fi
    fi
    
    return 0
}

# Process alert rule
openems::alerts::process_rule() {
    local rule_json="$1"
    local telemetry_json="$2"
    
    local rule_id=$(echo "$rule_json" | jq -r '.id')
    local rule_name=$(echo "$rule_json" | jq -r '.name')
    local condition=$(echo "$rule_json" | jq -r '.condition')
    local severity=$(echo "$rule_json" | jq -r '.severity')
    local message=$(echo "$rule_json" | jq -r '.message')
    local cooldown=$(echo "$rule_json" | jq -r '.cooldown // 300')
    
    # Check cooldown
    if openems::alerts::in_cooldown "$rule_id" "$cooldown"; then
        log::debug "Rule $rule_id in cooldown period"
        return 0
    fi
    
    # Evaluate condition (simplified evaluation for demo)
    # In production, would use proper expression evaluator
    local should_alert=false
    
    case "$rule_id" in
        "grid-outage")
            local grid_status=$(echo "$telemetry_json" | jq -r '.grid_status // "connected"')
            [[ "$grid_status" == "disconnected" ]] && should_alert=true
            ;;
        "battery-low")
            local battery_soc=$(echo "$telemetry_json" | jq -r '.battery_soc // 100')
            [[ "$battery_soc" -lt 20 ]] && should_alert=true
            ;;
        "solar-offline")
            local solar_power=$(echo "$telemetry_json" | jq -r '.solar_power // 0')
            local daylight=$(echo "$telemetry_json" | jq -r '.daylight // false')
            [[ "$solar_power" -eq 0 && "$daylight" == "true" ]] && should_alert=true
            ;;
        "peak-demand")
            local grid_import=$(echo "$telemetry_json" | jq -r '.grid_import // 0')
            local threshold=$(jq -r '.thresholds.peak_threshold // 10000' "$OPENEMS_ALERTS_CONFIG")
            [[ "$grid_import" -gt "$threshold" ]] && should_alert=true
            ;;
        "battery-fault")
            local battery_fault=$(echo "$telemetry_json" | jq -r '.battery_fault // false')
            [[ "$battery_fault" == "true" ]] && should_alert=true
            ;;
    esac
    
    # Send alerts if condition met
    if [[ "$should_alert" == "true" ]]; then
        log::info "Alert triggered: $rule_name"
        
        # Determine priority based on severity
        local priority=0
        case "$severity" in
            "critical") priority=1 ;;
            "warning") priority=0 ;;
            "info") priority=-1 ;;
        esac
        
        # Send through configured channels
        local channels=$(echo "$rule_json" | jq -r '.channels[]')
        while IFS= read -r channel; do
            case "$channel" in
                "pushover")
                    openems::alerts::send_pushover "$rule_name" "$message" "$priority"
                    ;;
                "twilio")
                    openems::alerts::send_twilio "$rule_name" "$message" "sms"
                    [[ "$severity" == "critical" ]] && \
                        openems::alerts::send_twilio "$rule_name" "$message" "voice"
                    ;;
            esac
        done <<< "$channels"
        
        # Record alert in history
        openems::alerts::record "$rule_id" "$rule_name" "$message" "$severity"
    fi
    
    return 0
}

# Check if alert is in cooldown
openems::alerts::in_cooldown() {
    local rule_id="$1"
    local cooldown_seconds="$2"
    
    # Check last alert time from history
    local last_alert_time=$(jq -r \
        --arg id "$rule_id" \
        '.[] | select(.rule_id == $id) | .timestamp' \
        "$OPENEMS_ALERT_HISTORY" 2>/dev/null | tail -1)
    
    if [[ -z "$last_alert_time" ]]; then
        return 1  # No previous alert, not in cooldown
    fi
    
    # Convert to epoch and check cooldown
    local last_epoch=$(date -d "$last_alert_time" +%s 2>/dev/null || echo 0)
    local current_epoch=$(date +%s)
    local elapsed=$((current_epoch - last_epoch))
    
    [[ "$elapsed" -lt "$cooldown_seconds" ]]
}

# Record alert in history
openems::alerts::record() {
    local rule_id="$1"
    local rule_name="$2"
    local message="$3"
    local severity="$4"
    
    local timestamp=$(date -Iseconds)
    
    # Add to history
    local temp_history=$(mktemp)
    jq --arg id "$rule_id" \
       --arg name "$rule_name" \
       --arg msg "$message" \
       --arg sev "$severity" \
       --arg ts "$timestamp" \
       '. + [{
           "rule_id": $id,
           "rule_name": $name,
           "message": $msg,
           "severity": $sev,
           "timestamp": $ts
       }] | .[-100:]' \
       "$OPENEMS_ALERT_HISTORY" > "$temp_history"
    
    mv "$temp_history" "$OPENEMS_ALERT_HISTORY"
    log::debug "Alert recorded: $rule_name"
    
    return 0
}

# Process telemetry for alerts
openems::alerts::process_telemetry() {
    local telemetry_json="${1:-}"
    
    if [[ -z "$telemetry_json" ]]; then
        # Try to fetch current telemetry
        telemetry_json=$(openems::core::get_telemetry 2>/dev/null || echo '{}')
    fi
    
    if [[ "$telemetry_json" == "{}" ]]; then
        log::debug "No telemetry data available"
        return 0
    fi
    
    # Process each alert rule
    local rules=$(jq -c '.rules[]' "$OPENEMS_ALERTS_CONFIG" 2>/dev/null)
    while IFS= read -r rule; do
        openems::alerts::process_rule "$rule" "$telemetry_json"
    done <<< "$rules"
    
    return 0
}

# Test alert system
openems::alerts::test() {
    log::info "Testing OpenEMS alert system..."
    
    # Initialize if needed
    openems::alerts::init
    
    # Test with simulated critical event
    local test_telemetry='{
        "grid_status": "disconnected",
        "battery_soc": 15,
        "solar_power": 0,
        "daylight": true,
        "grid_import": 12000,
        "battery_fault": false
    }'
    
    log::info "Simulating grid outage and low battery..."
    openems::alerts::process_telemetry "$test_telemetry"
    
    # Check history
    local alerts_sent=$(jq '. | length' "$OPENEMS_ALERT_HISTORY")
    log::info "Alert test complete. Alerts recorded: $alerts_sent"
    
    return 0
}

# List alert history
openems::alerts::history() {
    local limit="${1:-10}"
    
    if [[ ! -f "$OPENEMS_ALERT_HISTORY" ]]; then
        log::info "No alert history available"
        return 0
    fi
    
    echo "Recent OpenEMS Alerts:"
    echo "======================"
    
    jq -r --argjson limit "$limit" \
        '.[-$limit:] | reverse | .[] | 
        "\(.timestamp | split("T")[0] + " " + split("T")[1][:8]) | \(.severity | ascii_upcase) | \(.rule_name): \(.message)"' \
        "$OPENEMS_ALERT_HISTORY"
    
    return 0
}

# Clear alert history
openems::alerts::clear_history() {
    log::info "Clearing alert history..."
    echo "[]" > "$OPENEMS_ALERT_HISTORY"
    log::info "Alert history cleared"
    return 0
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Handle direct script execution
    case "${1:-}" in
        "init")
            openems::alerts::init
            ;;
        "test")
            openems::alerts::test
            ;;
        "configure")
            openems::alerts::configure "${2:-}" "${3:-}"
            ;;
        "history")
            openems::alerts::history "${2:-10}"
            ;;
        "clear")
            openems::alerts::clear_history
            ;;
        *)
            echo "Usage: $0 {init|test|configure|history|clear}"
            exit 1
            ;;
    esac
fi