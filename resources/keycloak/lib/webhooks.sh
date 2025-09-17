#!/bin/bash
# Keycloak webhook management functionality

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=/dev/null
source "${SCRIPT_DIR}/common.sh" 2>/dev/null || true

# Set strict mode after sourcing libraries
set -euo pipefail

# Default webhook settings
WEBHOOK_DEFAULT_TIMEOUT="${WEBHOOK_DEFAULT_TIMEOUT:-5000}"
WEBHOOK_DEFAULT_RETRIES="${WEBHOOK_DEFAULT_RETRIES:-3}"

# ========== Webhook Event Configuration ==========

webhook::list_events() {
    cat <<EOF
Available Webhook Events:
- LOGIN: User login events
- LOGOUT: User logout events  
- REGISTER: New user registration
- UPDATE_PROFILE: User profile updates
- UPDATE_PASSWORD: Password changes
- VERIFY_EMAIL: Email verification
- RESET_PASSWORD: Password reset requests
- CODE_TO_TOKEN: Authorization code exchange
- REFRESH_TOKEN: Token refresh
- INTROSPECT_TOKEN: Token introspection
- CLIENT_LOGIN: Client/Service account login
- CLIENT_REGISTER: New client registration
- CLIENT_UPDATE: Client configuration changes
- CLIENT_DELETE: Client deletion
- ADMIN_EVENT: Administrative actions
- REALM_EVENT: Realm-level changes
- GROUP_MEMBERSHIP: Group membership changes
- ROLE_MAPPING: Role assignment changes
EOF
}

# ========== Webhook Registration ==========

webhook::register() {
    local realm="${1:-master}"
    local event_type="${2:-}"
    local webhook_url="${3:-}"
    local secret="${4:-}"
    
    if [[ -z "$event_type" || -z "$webhook_url" ]]; then
        echo "[ERROR] Usage: webhook register <realm> <event_type> <webhook_url> [secret]"
        return 1
    fi
    
    echo "[INFO] Registering webhook for $event_type events in realm $realm..."
    
    # Get admin token
    local token
    token="$(get_admin_token)"
    if [[ -z "$token" ]]; then
        echo "[ERROR] Failed to get admin token"
        return 1
    fi
    
    # Create event listener configuration
    local listener_config
    listener_config=$(cat <<EOF
{
    "name": "webhook-${event_type,,}-$(date +%s)",
    "type": "webhook",
    "enabled": true,
    "config": {
        "url": "$webhook_url",
        "eventTypes": ["$event_type"],
        "includeRepresentation": "true",
        "timeout": "$WEBHOOK_DEFAULT_TIMEOUT",
        "retries": "$WEBHOOK_DEFAULT_RETRIES"
EOF
    )
    
    # Add secret if provided
    if [[ -n "$secret" ]]; then
        listener_config+=",\"secret\": \"$secret\""
    fi
    
    listener_config+="    }
}"
    
    # Configure event listener via admin API
    local response
    response=$(curl -s -X POST \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$listener_config" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/events/config" \
        2>&1) || {
        echo "[ERROR] Failed to register webhook: $response"
        return 1
    }
    
    echo "[SUCCESS] Webhook registered for $event_type events"
    echo "[INFO] Webhook URL: $webhook_url"
    [[ -n "$secret" ]] && echo "[INFO] Secret configured for signature validation"
    return 0
}

# ========== Webhook Management ==========

webhook::list() {
    local realm="${1:-master}"
    
    echo "[INFO] Listing webhooks for realm $realm..."
    
    # Get admin token
    local token
    token="$(get_admin_token)"
    if [[ -z "$token" ]]; then
        echo "[ERROR] Failed to get admin token"
        return 1
    fi
    
    # Get event configuration
    local response
    response=$(curl -s -X GET \
        -H "Authorization: Bearer $token" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/events/config" \
        2>&1) || {
        echo "[ERROR] Failed to list webhooks: $response"
        return 1
    }
    
    if [[ "$response" == "[]" || -z "$response" ]]; then
        echo "[INFO] No webhooks configured"
    else
        echo "[INFO] Configured webhooks:"
        echo "$response" | jq -r '.[] | "\(.name) - \(.config.url) [\(.config.eventTypes | join(","))]"' 2>/dev/null || echo "$response"
    fi
    
    return 0
}

webhook::remove() {
    local realm="${1:-master}"
    local webhook_name="${2:-}"
    
    if [[ -z "$webhook_name" ]]; then
        echo "[ERROR] Usage: webhook remove <realm> <webhook_name>"
        return 1
    fi
    
    echo "[INFO] Removing webhook $webhook_name from realm $realm..."
    
    # Get admin token
    local token
    token="$(get_admin_token)"
    if [[ -z "$token" ]]; then
        echo "[ERROR] Failed to get admin token"
        return 1
    fi
    
    # Remove webhook
    local response
    response=$(curl -s -X DELETE \
        -H "Authorization: Bearer $token" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/events/config/${webhook_name}" \
        2>&1) || {
        echo "[ERROR] Failed to remove webhook: $response"
        return 1
    }
    
    echo "[SUCCESS] Webhook $webhook_name removed"
    return 0
}

# ========== Webhook Testing ==========

webhook::test() {
    local realm="${1:-master}"
    local webhook_url="${2:-}"
    
    if [[ -z "$webhook_url" ]]; then
        echo "[ERROR] Usage: webhook test <realm> <webhook_url>"
        return 1
    fi
    
    echo "[INFO] Testing webhook connectivity to $webhook_url..."
    
    # Test webhook endpoint is reachable
    local test_payload
    test_payload='{"event":"TEST","timestamp":"'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'"}'
    
    local response
    response=$(timeout 5 curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$webhook_url" \
        -w "\n%{http_code}" \
        2>&1) || {
        echo "[ERROR] Webhook endpoint not reachable: $webhook_url"
        return 1
    }
    
    local http_code
    http_code=$(echo "$response" | tail -n1)
    
    if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]]; then
        echo "[SUCCESS] Webhook endpoint responded with HTTP $http_code"
    else
        echo "[WARNING] Webhook endpoint returned HTTP $http_code"
        echo "[INFO] Response: $(echo "$response" | head -n-1)"
    fi
    
    return 0
}

# ========== Webhook Event Configuration ==========

webhook::configure_events() {
    local realm="${1:-master}"
    local events="${2:-}"
    
    if [[ -z "$events" ]]; then
        echo "[ERROR] Usage: webhook configure-events <realm> <comma-separated-events>"
        echo "[INFO] Example: webhook configure-events master LOGIN,LOGOUT,REGISTER"
        return 1
    fi
    
    echo "[INFO] Configuring events for realm $realm..."
    
    # Get admin token
    local token
    token="$(get_admin_token)"
    if [[ -z "$token" ]]; then
        echo "[ERROR] Failed to get admin token"
        return 1
    fi
    
    # Enable event logging
    local event_config
    event_config=$(cat <<EOF
{
    "eventsEnabled": true,
    "eventsListeners": ["webhook"],
    "enabledEventTypes": [$(echo "$events" | sed 's/,/","/g' | sed 's/^/"/;s/$/"/')],
    "adminEventsEnabled": true,
    "adminEventsDetailsEnabled": true
}
EOF
    )
    
    # Update event configuration
    local response
    response=$(curl -s -X PUT \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$event_config" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/events/config" \
        2>&1) || {
        echo "[ERROR] Failed to configure events: $response"
        return 1
    }
    
    echo "[SUCCESS] Events configured for realm $realm"
    echo "[INFO] Enabled events: $events"
    return 0
}

# ========== Webhook History ==========

webhook::history() {
    local realm="${1:-master}"
    local limit="${2:-10}"
    
    echo "[INFO] Retrieving webhook event history for realm $realm..."
    
    # Get admin token
    local token
    token="$(get_admin_token)"
    if [[ -z "$token" ]]; then
        echo "[ERROR] Failed to get admin token"
        return 1
    fi
    
    # Get event history
    local response
    response=$(curl -s -X GET \
        -H "Authorization: Bearer $token" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/events?max=${limit}" \
        2>&1) || {
        echo "[ERROR] Failed to get event history: $response"
        return 1
    }
    
    if [[ "$response" == "[]" || -z "$response" ]]; then
        echo "[INFO] No events in history"
    else
        echo "[INFO] Recent events:"
        echo "$response" | jq -r '.[] | "\(.time | todate) - \(.type) - User: \(.userId // "N/A")"' 2>/dev/null || echo "$response"
    fi
    
    return 0
}

# ========== Webhook Retry Configuration ==========

webhook::configure_retry() {
    local realm="${1:-master}"
    local webhook_name="${2:-}"
    local max_retries="${3:-3}"
    local retry_interval="${4:-1000}"
    
    if [[ -z "$webhook_name" ]]; then
        echo "[ERROR] Usage: webhook configure-retry <realm> <webhook_name> [max_retries] [retry_interval_ms]"
        return 1
    fi
    
    echo "[INFO] Configuring retry policy for webhook $webhook_name..."
    
    # Get admin token
    local token
    token="$(get_admin_token)"
    if [[ -z "$token" ]]; then
        echo "[ERROR] Failed to get admin token"
        return 1
    fi
    
    # Update retry configuration
    local retry_config
    retry_config=$(cat <<EOF
{
    "maxRetries": $max_retries,
    "retryInterval": $retry_interval,
    "backoffMultiplier": 2,
    "maxRetryInterval": 60000
}
EOF
    )
    
    local response
    response=$(curl -s -X PATCH \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$retry_config" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/events/config/${webhook_name}/retry" \
        2>&1) || {
        echo "[ERROR] Failed to configure retry policy: $response"
        return 1
    }
    
    echo "[SUCCESS] Retry policy configured"
    echo "[INFO] Max retries: $max_retries"
    echo "[INFO] Initial retry interval: ${retry_interval}ms"
    return 0
}

# ========== CLI Interface ==========

# Only process CLI arguments if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        "list-events")
            webhook::list_events
            ;;
        "register")
            webhook::register "${2:-master}" "${3:-}" "${4:-}" "${5:-}"
            ;;
        "list")
            webhook::list "${2:-master}"
            ;;
        "remove")
            webhook::remove "${2:-master}" "${3:-}"
            ;;
        "test")
            webhook::test "${2:-master}" "${3:-}"
            ;;
        "configure-events")
            webhook::configure_events "${2:-master}" "${3:-}"
            ;;
        "history")
            webhook::history "${2:-master}" "${3:-10}"
            ;;
        "configure-retry")
            webhook::configure_retry "${2:-master}" "${3:-}" "${4:-3}" "${5:-1000}"
            ;;
        *)
            cat <<EOF
Webhook Management Commands:
  list-events                    List available webhook event types
  register <realm> <event> <url> [secret]  Register a webhook
  list [realm]                   List configured webhooks
  remove <realm> <name>          Remove a webhook
  test <realm> <url>             Test webhook connectivity
  configure-events <realm> <events>  Configure realm events
  history [realm] [limit]        Show event history
  configure-retry <realm> <name> [retries] [interval]  Configure retry policy

Examples:
  webhook list-events
  webhook register master LOGIN https://example.com/webhook mysecret
  webhook list master
  webhook test master https://example.com/webhook
  webhook configure-events master LOGIN,LOGOUT,REGISTER
  webhook history master 20
EOF
            exit 0
            ;;
    esac
fi