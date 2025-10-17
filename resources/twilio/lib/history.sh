#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_DIR="${APP_ROOT}/resources/twilio"
TWILIO_LIB_DIR="${TWILIO_DIR}/lib"
TWILIO_DATA_DIR="${TWILIO_DIR}/data"
TWILIO_HISTORY_FILE="${TWILIO_DATA_DIR}/message_history.json"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Ensure data directory exists
twilio::history::ensure_data_dir() {
    if [[ ! -d "$TWILIO_DATA_DIR" ]]; then
        mkdir -p "$TWILIO_DATA_DIR"
    fi
    
    # Initialize history file if it doesn't exist
    if [[ ! -f "$TWILIO_HISTORY_FILE" ]]; then
        echo '{"messages": []}' > "$TWILIO_HISTORY_FILE"
    fi
}

# Log a message to history
twilio::history::log_message() {
    local sid="${1:-}"
    local from="${2:-}"
    local to="${3:-}"
    local body="${4:-}"
    local status="${5:-sent}"
    local timestamp="${6:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
    
    twilio::history::ensure_data_dir
    
    # Create message entry
    local message_entry
    message_entry=$(jq -n \
        --arg sid "$sid" \
        --arg from "$from" \
        --arg to "$to" \
        --arg body "$body" \
        --arg status "$status" \
        --arg timestamp "$timestamp" \
        '{
            sid: $sid,
            from: $from,
            to: $to,
            body: $body,
            status: $status,
            timestamp: $timestamp
        }')
    
    # Append to history file
    jq ".messages += [$message_entry]" "$TWILIO_HISTORY_FILE" > "${TWILIO_HISTORY_FILE}.tmp" && \
        mv "${TWILIO_HISTORY_FILE}.tmp" "$TWILIO_HISTORY_FILE"
    
    log::debug "Message logged to history: $sid"
}

# Get message history
twilio::history::list() {
    local limit="${1:-50}"
    local filter_to="${2:-}"
    local filter_status="${3:-}"
    
    twilio::history::ensure_data_dir
    
    local jq_filter=".messages"
    
    # Apply filters
    if [[ -n "$filter_to" ]]; then
        jq_filter="$jq_filter | map(select(.to == \"$filter_to\"))"
    fi
    
    if [[ -n "$filter_status" ]]; then
        jq_filter="$jq_filter | map(select(.status == \"$filter_status\"))"
    fi
    
    # Sort by timestamp (newest first) and limit
    jq_filter="$jq_filter | sort_by(.timestamp) | reverse | .[:$limit]"
    
    jq "$jq_filter" "$TWILIO_HISTORY_FILE"
}

# Get message by SID
twilio::history::get_by_sid() {
    local sid="${1:-}"
    
    if [[ -z "$sid" ]]; then
        log::error "Message SID is required"
        return 1
    fi
    
    twilio::history::ensure_data_dir
    
    local message
    message=$(jq ".messages[] | select(.sid == \"$sid\")" "$TWILIO_HISTORY_FILE")
    
    if [[ -z "$message" ]]; then
        log::error "Message not found: $sid"
        return 1
    fi
    
    echo "$message"
}

# Update message status
twilio::history::update_status() {
    local sid="${1:-}"
    local new_status="${2:-}"
    
    if [[ -z "$sid" ]] || [[ -z "$new_status" ]]; then
        log::error "Usage: update_status <sid> <status>"
        return 1
    fi
    
    twilio::history::ensure_data_dir
    
    # Update the status in the history file
    jq "(.messages[] | select(.sid == \"$sid\") | .status) = \"$new_status\"" \
        "$TWILIO_HISTORY_FILE" > "${TWILIO_HISTORY_FILE}.tmp" && \
        mv "${TWILIO_HISTORY_FILE}.tmp" "$TWILIO_HISTORY_FILE"
    
    log::debug "Updated status for $sid to $new_status"
}

# Get delivery status from Twilio API
twilio::history::check_delivery_status() {
    local sid="${1:-}"
    
    if [[ -z "$sid" ]]; then
        log::error "Message SID is required"
        return 1
    fi
    
    # Check if installed and authenticated
    if ! twilio::is_installed; then
        log::error "$MSG_TWILIO_NOT_INSTALLED"
        return 1
    fi
    
    if ! twilio::setup_auth; then
        log::error "Authentication failed"
        return 1
    fi
    
    local cmd
    cmd=$(twilio::get_command)
    
    # Fetch message status from API
    local status
    status=$("$cmd" api:core:messages:get --sid "$sid" --properties status 2>/dev/null | jq -r '.status // "unknown"')
    
    if [[ "$status" != "unknown" ]]; then
        # Update local history
        twilio::history::update_status "$sid" "$status"
        echo "$status"
    else
        log::error "Could not fetch status for message: $sid"
        return 1
    fi
}

# Check and update all pending messages
twilio::history::update_pending_statuses() {
    twilio::history::ensure_data_dir
    
    # Get all messages with status 'sent' or 'pending'
    local pending_sids
    pending_sids=$(jq -r '.messages[] | select(.status == "sent" or .status == "pending") | .sid' "$TWILIO_HISTORY_FILE")
    
    if [[ -z "$pending_sids" ]]; then
        log::info "No pending messages to update"
        return 0
    fi
    
    log::info "Updating delivery status for pending messages..."
    
    local updated_count=0
    while IFS= read -r sid; do
        if [[ -n "$sid" ]]; then
            if twilio::history::check_delivery_status "$sid" &>/dev/null; then
                ((updated_count++))
            fi
            # Rate limiting
            sleep 0.1
        fi
    done <<< "$pending_sids"
    
    log::success "Updated $updated_count message statuses"
}

# Get statistics
twilio::history::stats() {
    twilio::history::ensure_data_dir
    
    local total=$(jq '.messages | length' "$TWILIO_HISTORY_FILE")
    local delivered=$(jq '.messages | map(select(.status == "delivered")) | length' "$TWILIO_HISTORY_FILE")
    local failed=$(jq '.messages | map(select(.status == "failed" or .status == "undelivered")) | length' "$TWILIO_HISTORY_FILE")
    local pending=$(jq '.messages | map(select(.status == "sent" or .status == "pending")) | length' "$TWILIO_HISTORY_FILE")
    
    # Get date range
    local first_date=$(jq -r '.messages | sort_by(.timestamp) | first | .timestamp // "N/A"' "$TWILIO_HISTORY_FILE")
    local last_date=$(jq -r '.messages | sort_by(.timestamp) | last | .timestamp // "N/A"' "$TWILIO_HISTORY_FILE")
    
    # Calculate delivery rate without bc
    local delivery_rate=0
    if [[ $total -gt 0 ]]; then
        delivery_rate=$(( delivered * 100 / total ))
    fi
    
    echo "{"
    echo "  \"total_messages\": $total,"
    echo "  \"delivered\": $delivered,"
    echo "  \"failed\": $failed,"
    echo "  \"pending\": $pending,"
    echo "  \"delivery_rate\": $delivery_rate,"
    echo "  \"first_message\": \"$first_date\","
    echo "  \"last_message\": \"$last_date\""
    echo "}"
}

# Clear history (with confirmation)
twilio::history::clear() {
    local force="${1:-}"
    
    if [[ "$force" != "--force" ]]; then
        log::warn "This will delete all message history. Use --force to confirm."
        return 1
    fi
    
    twilio::history::ensure_data_dir
    echo '{"messages": []}' > "$TWILIO_HISTORY_FILE"
    log::success "Message history cleared"
}

# Export history to CSV
twilio::history::export_csv() {
    local output_file="${1:-twilio_history.csv}"
    
    twilio::history::ensure_data_dir
    
    # Create CSV header
    echo "timestamp,sid,from,to,status,message" > "$output_file"
    
    # Export data
    jq -r '.messages[] | [.timestamp, .sid, .from, .to, .status, .body] | @csv' "$TWILIO_HISTORY_FILE" >> "$output_file"
    
    local count=$(jq '.messages | length' "$TWILIO_HISTORY_FILE")
    log::success "Exported $count messages to $output_file"
}