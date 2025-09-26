#!/bin/bash
# Airbyte Webhook Notification Library
# Provides webhook notifications for sync events

set -euo pipefail

# Webhook storage directory
WEBHOOK_DIR="${DATA_DIR}/webhooks"
WEBHOOK_REGISTRY="${WEBHOOK_DIR}/registry.json"
WEBHOOK_LOG="${WEBHOOK_DIR}/webhook.log"

# Initialize webhook storage
init_webhooks() {
    if [[ ! -d "$WEBHOOK_DIR" ]]; then
        mkdir -p "$WEBHOOK_DIR"
        chmod 755 "$WEBHOOK_DIR"
    fi
    
    if [[ ! -f "$WEBHOOK_REGISTRY" ]]; then
        echo "[]" > "$WEBHOOK_REGISTRY"
    fi
    
    if [[ ! -f "$WEBHOOK_LOG" ]]; then
        touch "$WEBHOOK_LOG"
    fi
}

# Register a webhook
register_webhook() {
    local name="$1"
    local url="$2"
    local events="$3"  # comma-separated: sync_started,sync_completed,sync_failed
    local enabled="${4:-true}"
    local auth_type="${5:-none}"  # none, basic, bearer, api_key
    local auth_value="${6:-}"
    
    init_webhooks
    
    # Validate URL
    if ! validate_url "$url"; then
        log_error "Invalid webhook URL: $url"
        return 1
    fi
    
    # Parse events
    local event_array=$(echo "$events" | jq -Rs 'split(",")')
    
    # Create webhook entry
    local webhook_entry=$(jq -n \
        --arg name "$name" \
        --arg url "$url" \
        --argjson events "$event_array" \
        --arg enabled "$enabled" \
        --arg auth_type "$auth_type" \
        --arg created "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        '{
            name: $name,
            url: $url,
            events: $events,
            enabled: ($enabled == "true"),
            auth: {
                type: $auth_type,
                value: null
            },
            created: $created,
            last_triggered: null,
            trigger_count: 0,
            last_status: null
        }')
    
    # Store auth value securely if provided
    if [[ -n "$auth_value" ]]; then
        store_webhook_auth "$name" "$auth_type" "$auth_value"
    fi
    
    # Update registry
    local updated_registry=$(cat "$WEBHOOK_REGISTRY" | jq \
        --argjson entry "$webhook_entry" \
        'map(select(.name != $entry.name)) + [$entry]')
    
    echo "$updated_registry" > "$WEBHOOK_REGISTRY"
    
    log_info "Webhook registered: $name"
    echo "$webhook_entry"
}

# Validate webhook URL
validate_url() {
    local url="$1"
    
    # Check if URL is well-formed
    if [[ ! "$url" =~ ^https?:// ]]; then
        return 1
    fi
    
    # Optional: Test connectivity (commented out to avoid external calls)
    # timeout 5 curl -sf -o /dev/null "$url" 2>/dev/null
    
    return 0
}

# Store webhook authentication
store_webhook_auth() {
    local name="$1"
    local auth_type="$2"
    local auth_value="$3"
    
    # Use credential storage for secure auth storage
    local auth_data=$(jq -n \
        --arg type "$auth_type" \
        --arg value "$auth_value" \
        '{type: $type, value: $value}')
    
    echo "$auth_data" | store_credential "webhook_${name}" "api_key" "$auth_data"
}

# Get webhook by name
get_webhook() {
    local name="$1"
    
    init_webhooks
    
    cat "$WEBHOOK_REGISTRY" | jq --arg name "$name" '.[] | select(.name == $name)'
}

# List all webhooks
list_webhooks() {
    init_webhooks
    
    cat "$WEBHOOK_REGISTRY" | jq '.'
}

# Enable webhook
enable_webhook() {
    local name="$1"
    
    init_webhooks
    
    local updated=$(cat "$WEBHOOK_REGISTRY" | jq \
        --arg name "$name" \
        'map(if .name == $name then .enabled = true else . end)')
    
    echo "$updated" > "$WEBHOOK_REGISTRY"
    
    log_info "Webhook enabled: $name"
}

# Disable webhook
disable_webhook() {
    local name="$1"
    
    init_webhooks
    
    local updated=$(cat "$WEBHOOK_REGISTRY" | jq \
        --arg name "$name" \
        'map(if .name == $name then .enabled = false else . end)')
    
    echo "$updated" > "$WEBHOOK_REGISTRY"
    
    log_info "Webhook disabled: $name"
}

# Delete webhook
delete_webhook() {
    local name="$1"
    
    init_webhooks
    
    # Remove from registry
    local updated=$(cat "$WEBHOOK_REGISTRY" | jq \
        --arg name "$name" \
        'map(select(.name != $name))')
    
    echo "$updated" > "$WEBHOOK_REGISTRY"
    
    # Remove auth if exists
    remove_credential "webhook_${name}" 2>/dev/null || true
    
    log_info "Webhook deleted: $name"
}

# Trigger webhook for event
trigger_webhook() {
    local event_type="$1"  # sync_started, sync_completed, sync_failed
    local payload="$2"     # JSON payload with sync details
    
    init_webhooks
    
    # Get all enabled webhooks for this event
    local webhooks=$(cat "$WEBHOOK_REGISTRY" | jq \
        --arg event "$event_type" \
        '.[] | select(.enabled == true and (.events | contains([$event])))')
    
    if [[ -z "$webhooks" ]]; then
        log_debug "No webhooks registered for event: $event_type"
        return 0
    fi
    
    # Process each webhook
    echo "$webhooks" | jq -c '.' | while IFS= read -r webhook; do
        local name=$(echo "$webhook" | jq -r '.name')
        local url=$(echo "$webhook" | jq -r '.url')
        local auth_type=$(echo "$webhook" | jq -r '.auth.type')
        
        log_info "Triggering webhook: $name for event: $event_type"
        
        # Build curl command
        local curl_cmd="curl -X POST"
        curl_cmd="$curl_cmd -H 'Content-Type: application/json'"
        
        # Add authentication
        case "$auth_type" in
            basic)
                local auth_value=$(get_credential "webhook_${name}" | jq -r '.value')
                curl_cmd="$curl_cmd -u '$auth_value'"
                ;;
            bearer)
                local token=$(get_credential "webhook_${name}" | jq -r '.value')
                curl_cmd="$curl_cmd -H 'Authorization: Bearer $token'"
                ;;
            api_key)
                local api_key=$(get_credential "webhook_${name}" | jq -r '.value')
                curl_cmd="$curl_cmd -H 'X-API-Key: $api_key'"
                ;;
        esac
        
        # Add webhook metadata to payload
        local enriched_payload=$(echo "$payload" | jq \
            --arg event "$event_type" \
            --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            --arg webhook "$name" \
            '. + {webhook_event: $event, webhook_timestamp: $timestamp, webhook_name: $webhook}')
        
        # Execute webhook
        local response
        local status_code
        
        response=$(echo "$enriched_payload" | eval "$curl_cmd -d @- -w '\n%{http_code}' '$url'" 2>&1)
        status_code=$(echo "$response" | tail -n1)
        
        # Log the webhook call
        log_webhook_call "$name" "$event_type" "$status_code"
        
        # Update webhook statistics
        update_webhook_stats "$name" "$status_code"
        
        # Check for success
        if [[ "$status_code" =~ ^2[0-9][0-9]$ ]]; then
            log_info "Webhook triggered successfully: $name (status: $status_code)"
        else
            log_error "Webhook failed: $name (status: $status_code)"
        fi
    done
}

# Log webhook call
log_webhook_call() {
    local name="$1"
    local event="$2"
    local status="$3"
    local timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    
    echo "[$timestamp] Webhook: $name, Event: $event, Status: $status" >> "$WEBHOOK_LOG"
}

# Update webhook statistics
update_webhook_stats() {
    local name="$1"
    local status_code="$2"
    
    local updated=$(cat "$WEBHOOK_REGISTRY" | jq \
        --arg name "$name" \
        --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --arg status "$status_code" \
        'map(if .name == $name then 
            .last_triggered = $timestamp |
            .trigger_count = (.trigger_count + 1) |
            .last_status = $status
        else . end)')
    
    echo "$updated" > "$WEBHOOK_REGISTRY"
}

# Test webhook
test_webhook() {
    local name="$1"
    
    local webhook=$(get_webhook "$name")
    if [[ -z "$webhook" ]]; then
        log_error "Webhook not found: $name"
        return 1
    fi
    
    # Create test payload
    local test_payload=$(jq -n \
        --arg connection "test-connection" \
        --arg job "test-job-id" \
        '{
            test: true,
            connection_id: $connection,
            job_id: $job,
            message: "This is a test webhook notification"
        }')
    
    # Trigger the webhook
    trigger_webhook "test" "$test_payload"
}

# Get webhook statistics
webhook_stats() {
    local name="${1:-}"
    
    init_webhooks
    
    if [[ -n "$name" ]]; then
        # Get stats for specific webhook
        local webhook=$(get_webhook "$name")
        if [[ -z "$webhook" ]]; then
            echo "Webhook not found: $name"
            return 1
        fi
        
        echo "Webhook Statistics: $name"
        echo "  URL: $(echo "$webhook" | jq -r '.url')"
        echo "  Events: $(echo "$webhook" | jq -r '.events | join(", ")')"
        echo "  Enabled: $(echo "$webhook" | jq -r '.enabled')"
        echo "  Trigger Count: $(echo "$webhook" | jq -r '.trigger_count')"
        echo "  Last Triggered: $(echo "$webhook" | jq -r '.last_triggered // "Never"')"
        echo "  Last Status: $(echo "$webhook" | jq -r '.last_status // "N/A"')"
        echo "  Created: $(echo "$webhook" | jq -r '.created')"
    else
        # Get overall stats
        local total=$(cat "$WEBHOOK_REGISTRY" | jq 'length')
        local enabled=$(cat "$WEBHOOK_REGISTRY" | jq '[.[] | select(.enabled == true)] | length')
        local triggered=$(cat "$WEBHOOK_REGISTRY" | jq '[.[] | select(.trigger_count > 0)] | length')
        
        echo "Webhook Statistics:"
        echo "  Total Webhooks: $total"
        echo "  Enabled: $enabled"
        echo "  Ever Triggered: $triggered"
        echo ""
        echo "Recent Activity:"
        tail -n 10 "$WEBHOOK_LOG" 2>/dev/null || echo "  No recent activity"
    fi
}

# Monitor sync and trigger webhooks
monitor_sync_with_webhooks() {
    local connection_id="$1"
    local job_id="$2"
    
    # Trigger sync started webhook
    local start_payload=$(jq -n \
        --arg connection "$connection_id" \
        --arg job "$job_id" \
        --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        '{
            connection_id: $connection,
            job_id: $job,
            started_at: $started
        }')
    
    trigger_webhook "sync_started" "$start_payload"
    
    # Monitor the job
    local result=$(monitor_sync_job "$job_id")
    local exit_code=$?
    
    # Prepare completion payload
    local end_payload=$(echo "$result" | jq \
        --arg connection "$connection_id" \
        --arg completed "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        '. + {
            connection_id: $connection,
            completed_at: $completed
        }')
    
    # Trigger appropriate completion webhook
    if [[ $exit_code -eq 0 ]]; then
        trigger_webhook "sync_completed" "$end_payload"
    else
        trigger_webhook "sync_failed" "$end_payload"
    fi
    
    return $exit_code
}