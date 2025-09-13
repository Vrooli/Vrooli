#!/usr/bin/env bash
################################################################################
# Twilio Audit Logging Library
# 
# Comprehensive audit logging for all Twilio operations
# Tracks: who, what, when, where, why, and result
################################################################################

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_DIR="${APP_ROOT}/resources/twilio"
TWILIO_DATA_DIR="${TWILIO_DIR}/data"
TWILIO_AUDIT_DIR="${TWILIO_DATA_DIR}/audit"
TWILIO_AUDIT_FILE="${TWILIO_AUDIT_DIR}/audit_$(date +%Y%m).log"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Audit Directory Management
################################################################################

# Ensure audit directory exists with proper permissions
twilio::audit::ensure_dir() {
    if [[ ! -d "$TWILIO_AUDIT_DIR" ]]; then
        mkdir -p "$TWILIO_AUDIT_DIR"
        # Restrict permissions for audit logs
        chmod 700 "$TWILIO_AUDIT_DIR"
    fi
    
    # Create monthly audit file if it doesn't exist
    if [[ ! -f "$TWILIO_AUDIT_FILE" ]]; then
        touch "$TWILIO_AUDIT_FILE"
        chmod 600 "$TWILIO_AUDIT_FILE"
    fi
}

################################################################################
# Audit Event Types
################################################################################

# Define audit event types
export AUDIT_SMS_SEND="SMS_SEND"
export AUDIT_SMS_BULK="SMS_BULK"
export AUDIT_SMS_TEMPLATE="SMS_TEMPLATE"
export AUDIT_AUTH_SUCCESS="AUTH_SUCCESS"
export AUDIT_AUTH_FAILURE="AUTH_FAILURE"
export AUDIT_CONFIG_CHANGE="CONFIG_CHANGE"
export AUDIT_RATE_LIMIT="RATE_LIMIT"
export AUDIT_ERROR="ERROR"
export AUDIT_ACCESS="ACCESS"

################################################################################
# Core Audit Functions
################################################################################

# Get caller information
twilio::audit::get_caller_info() {
    local caller_user="${USER:-unknown}"
    local caller_pid="$$"
    local caller_ppid="${PPID:-0}"
    local caller_tty="$(tty 2>/dev/null || echo 'notty')"
    local caller_ip="${SSH_CLIENT%% *}" # Get IP from SSH_CLIENT if available
    caller_ip="${caller_ip:-local}"
    
    echo "${caller_user}|${caller_pid}|${caller_ppid}|${caller_tty}|${caller_ip}"
}

# Generate audit entry
twilio::audit::create_entry() {
    local event_type="${1:-UNKNOWN}"
    local action="${2:-}"
    local details="${3:-}"
    local result="${4:-SUCCESS}"
    local metadata="${5:-}"
    
    local timestamp="$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")"
    local caller_info="$(twilio::audit::get_caller_info)"
    local session_id="${TWILIO_SESSION_ID:-$(uuidgen 2>/dev/null || echo "SESSION_$(date +%s)")}"
    
    # Create structured audit entry
    local audit_entry=$(cat <<EOF
{
  "timestamp": "${timestamp}",
  "session_id": "${session_id}",
  "event_type": "${event_type}",
  "action": "${action}",
  "result": "${result}",
  "caller": {
    "user": "$(echo $caller_info | cut -d'|' -f1)",
    "pid": "$(echo $caller_info | cut -d'|' -f2)",
    "ppid": "$(echo $caller_info | cut -d'|' -f3)",
    "tty": "$(echo $caller_info | cut -d'|' -f4)",
    "ip": "$(echo $caller_info | cut -d'|' -f5)"
  },
  "details": ${details:-"{}"},
  "metadata": ${metadata:-"{}"}
}
EOF
)
    
    echo "$audit_entry"
}

# Log audit event
twilio::audit::log() {
    local event_type="${1:-}"
    local action="${2:-}"
    local details="${3:-}"
    local result="${4:-SUCCESS}"
    local metadata="${5:-}"
    
    twilio::audit::ensure_dir
    
    # Create audit entry
    local audit_entry
    audit_entry=$(twilio::audit::create_entry "$event_type" "$action" "$details" "$result" "$metadata")
    
    # Write to audit log (append-only)
    echo "$audit_entry" >> "$TWILIO_AUDIT_FILE"
    
    # Also log to syslog if available
    if command -v logger &> /dev/null; then
        logger -t "twilio-audit" -p user.info "$event_type: $action - $result"
    fi
    
    # Log debug message
    log::debug "Audit logged: $event_type - $action"
}

################################################################################
# Specific Audit Functions
################################################################################

# Audit SMS send
twilio::audit::log_sms() {
    local from="${1:-}"
    local to="${2:-}"
    local message_length="${3:-0}"
    local sid="${4:-}"
    local result="${5:-SUCCESS}"
    local error="${6:-}"
    
    local details=$(jq -n \
        --arg from "$from" \
        --arg to "$to" \
        --arg len "$message_length" \
        --arg sid "$sid" \
        --arg error "$error" \
        '{
            from: $from,
            to: $to,
            message_length: $len,
            sid: $sid,
            error: $error
        }')
    
    twilio::audit::log "$AUDIT_SMS_SEND" "send_sms" "$details" "$result"
}

# Audit bulk SMS
twilio::audit::log_bulk_sms() {
    local recipient_count="${1:-0}"
    local successful="${2:-0}"
    local failed="${3:-0}"
    local from="${4:-}"
    
    local details=$(jq -n \
        --arg count "$recipient_count" \
        --arg success "$successful" \
        --arg fail "$failed" \
        --arg from "$from" \
        '{
            recipient_count: $count,
            successful: $success,
            failed: $fail,
            from: $from
        }')
    
    twilio::audit::log "$AUDIT_SMS_BULK" "send_bulk_sms" "$details" \
        "$([[ $failed -eq 0 ]] && echo 'SUCCESS' || echo 'PARTIAL')"
}

# Audit authentication
twilio::audit::log_auth() {
    local action="${1:-login}"
    local result="${2:-SUCCESS}"
    local method="${3:-api_key}"
    local error="${4:-}"
    
    local details=$(jq -n \
        --arg method "$method" \
        --arg error "$error" \
        '{
            method: $method,
            error: $error
        }')
    
    local event_type="$([[ "$result" == "SUCCESS" ]] && echo "$AUDIT_AUTH_SUCCESS" || echo "$AUDIT_AUTH_FAILURE")"
    twilio::audit::log "$event_type" "$action" "$details" "$result"
}

# Audit configuration changes
twilio::audit::log_config() {
    local setting="${1:-}"
    local old_value="${2:-}"
    local new_value="${3:-}"
    
    local details=$(jq -n \
        --arg setting "$setting" \
        --arg old "$old_value" \
        --arg new "$new_value" \
        '{
            setting: $setting,
            old_value: $old,
            new_value: $new
        }')
    
    twilio::audit::log "$AUDIT_CONFIG_CHANGE" "update_config" "$details" "SUCCESS"
}

# Audit rate limiting
twilio::audit::log_rate_limit() {
    local operation="${1:-}"
    local current_rate="${2:-}"
    local limit="${3:-}"
    
    local details=$(jq -n \
        --arg op "$operation" \
        --arg rate "$current_rate" \
        --arg limit "$limit" \
        '{
            operation: $op,
            current_rate: $rate,
            limit: $limit
        }')
    
    twilio::audit::log "$AUDIT_RATE_LIMIT" "rate_limit_hit" "$details" "WARNING"
}

################################################################################
# Audit Query Functions
################################################################################

# Search audit logs
twilio::audit::search() {
    local event_type="${1:-}"
    local date_from="${2:-}"
    local date_to="${3:-}"
    local user="${4:-}"
    
    twilio::audit::ensure_dir
    
    local search_cmd="cat $TWILIO_AUDIT_DIR/*.log 2>/dev/null"
    
    # Apply filters
    if [[ -n "$event_type" ]]; then
        search_cmd="$search_cmd | jq 'select(.event_type == \"$event_type\")'"
    fi
    
    if [[ -n "$user" ]]; then
        search_cmd="$search_cmd | jq 'select(.caller.user == \"$user\")'"
    fi
    
    if [[ -n "$date_from" ]]; then
        search_cmd="$search_cmd | jq 'select(.timestamp >= \"$date_from\")'"
    fi
    
    if [[ -n "$date_to" ]]; then
        search_cmd="$search_cmd | jq 'select(.timestamp <= \"$date_to\")'"
    fi
    
    eval "$search_cmd"
}

# Get audit statistics
twilio::audit::stats() {
    local month="${1:-$(date +%Y%m)}"
    local audit_file="${TWILIO_AUDIT_DIR}/audit_${month}.log"
    
    if [[ ! -f "$audit_file" ]]; then
        log::warn "No audit log for month: $month"
        return 1
    fi
    
    echo "=== Audit Statistics for $month ==="
    echo ""
    echo "Event Type Summary:"
    jq -s 'group_by(.event_type) | map({type: .[0].event_type, count: length})' "$audit_file" 2>/dev/null || echo "No data"
    
    echo ""
    echo "Result Summary:"
    jq -s 'group_by(.result) | map({result: .[0].result, count: length})' "$audit_file" 2>/dev/null || echo "No data"
    
    echo ""
    echo "User Activity:"
    jq -s 'group_by(.caller.user) | map({user: .[0].caller.user, count: length}) | sort_by(.count) | reverse | .[0:10]' "$audit_file" 2>/dev/null || echo "No data"
}

# Archive old audit logs
twilio::audit::archive() {
    local months_to_keep="${1:-6}"
    
    twilio::audit::ensure_dir
    
    # Find and compress old audit logs
    local cutoff_date=$(date -d "$months_to_keep months ago" +%Y%m)
    
    for audit_file in "$TWILIO_AUDIT_DIR"/audit_*.log; do
        if [[ -f "$audit_file" ]]; then
            local file_month=$(basename "$audit_file" | grep -oE '[0-9]{6}')
            if [[ "$file_month" -lt "$cutoff_date" ]]; then
                log::info "Archiving audit log: $audit_file"
                gzip "$audit_file"
            fi
        fi
    done
}

################################################################################
# Export Functions
################################################################################

export -f twilio::audit::ensure_dir
export -f twilio::audit::get_caller_info
export -f twilio::audit::create_entry
export -f twilio::audit::log
export -f twilio::audit::log_sms
export -f twilio::audit::log_bulk_sms
export -f twilio::audit::log_auth
export -f twilio::audit::log_config
export -f twilio::audit::log_rate_limit
export -f twilio::audit::search
export -f twilio::audit::stats
export -f twilio::audit::archive

# Set session ID if not already set
export TWILIO_SESSION_ID="${TWILIO_SESSION_ID:-$(uuidgen 2>/dev/null || echo "SESSION_$(date +%s)")}"