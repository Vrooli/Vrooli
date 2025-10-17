#!/usr/bin/env bash
# Judge0 Webhook Management Module
# Handles webhook configuration and callback notifications for async code execution

# Source shared utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/judge0/lib/common.sh"

#######################################
# Configure webhook for a submission
# Arguments:
#   $1 - Webhook URL
#   $2 - Events to listen for (comma-separated)
# Outputs:
#   Webhook configuration status
#######################################
judge0::webhooks::configure() {
    local webhook_url="$1"
    local events="${2:-completed,failed}"
    
    if [[ -z "$webhook_url" ]]; then
        log::error "Webhook URL is required"
        return 1
    fi
    
    # Validate URL format
    if ! [[ "$webhook_url" =~ ^https?:// ]]; then
        log::error "Invalid webhook URL format"
        return 1
    fi
    
    # Store webhook configuration
    local config_file="${JUDGE0_CONFIG_DIR}/webhooks.json"
    
    # Create or update webhook configuration
    cat > "$config_file" << EOF
{
  "enabled": true,
  "url": "$webhook_url",
  "events": ["${events//,/\",\"}"],
  "retry_attempts": 3,
  "retry_delay": 1000,
  "timeout": 5000,
  "headers": {
    "Content-Type": "application/json",
    "X-Judge0-Webhook": "true"
  }
}
EOF
    
    log::success "Webhook configured: $webhook_url"
    return 0
}

#######################################
# Submit code with webhook callback
# Arguments:
#   $1 - Source code
#   $2 - Language
#   $3 - Webhook URL
#   $4 - Input data (optional)
# Outputs:
#   Submission token
#######################################
judge0::webhooks::submit_with_callback() {
    local code="$1"
    local language="$2"
    local webhook_url="$3"
    local input="${4:-}"
    
    if [[ -z "$code" ]] || [[ -z "$language" ]] || [[ -z "$webhook_url" ]]; then
        log::error "Code, language, and webhook URL are required"
        return 1
    fi
    
    # Get language ID
    local lang_id=$(judge0::get_language_id "$language")
    if [[ -z "$lang_id" ]]; then
        log::error "Unsupported language: $language"
        return 1
    fi
    
    # Prepare submission with callback
    local submission_data=$(jq -n \
        --arg code "$code" \
        --arg lang_id "$lang_id" \
        --arg input "$input" \
        --arg webhook "$webhook_url" \
        '{
            source_code: $code,
            language_id: ($lang_id | tonumber),
            stdin: $input,
            callback_url: $webhook,
            wait: false
        }')
    
    # Submit to Judge0
    local response=$(curl -sf -X POST "${JUDGE0_BASE_URL}/submissions" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Token: $(judge0::get_api_key)" \
        -d "$submission_data" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        log::error "Failed to submit code with webhook"
        return 1
    fi
    
    local token=$(echo "$response" | jq -r '.token')
    if [[ -z "$token" ]] || [[ "$token" == "null" ]]; then
        log::error "Failed to get submission token"
        return 1
    fi
    
    echo "$token"
    log::info "Submission created with webhook: $token"
    return 0
}

#######################################
# Register webhook endpoint
# Arguments:
#   $1 - Endpoint name
#   $2 - Webhook URL
# Outputs:
#   Registration status
#######################################
judge0::webhooks::register() {
    local name="$1"
    local url="$2"
    
    if [[ -z "$name" ]] || [[ -z "$url" ]]; then
        log::error "Endpoint name and URL are required"
        return 1
    fi
    
    # Store in webhook registry
    local registry_file="${JUDGE0_CONFIG_DIR}/webhook_registry.json"
    
    # Load existing registry or create new
    local registry="{}"
    if [[ -f "$registry_file" ]]; then
        registry=$(cat "$registry_file")
    fi
    
    # Add new endpoint
    registry=$(echo "$registry" | jq \
        --arg name "$name" \
        --arg url "$url" \
        --arg created "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        '.[$name] = {
            url: $url,
            created: $created,
            active: true,
            submissions: 0,
            last_used: null
        }')
    
    echo "$registry" > "$registry_file"
    
    log::success "Webhook endpoint registered: $name"
    return 0
}

#######################################
# List registered webhooks
# Outputs:
#   List of registered webhook endpoints
#######################################
judge0::webhooks::list() {
    local registry_file="${JUDGE0_CONFIG_DIR}/webhook_registry.json"
    
    if [[ ! -f "$registry_file" ]]; then
        log::info "No webhooks registered"
        return 0
    fi
    
    log::header "Registered Webhooks"
    
    # Parse and display webhooks
    jq -r 'to_entries[] | "\(.key):\n  URL: \(.value.url)\n  Active: \(.value.active)\n  Submissions: \(.value.submissions)\n  Created: \(.value.created)\n  Last Used: \(.value.last_used // "Never")\n"' "$registry_file"
    
    return 0
}

#######################################
# Remove webhook endpoint
# Arguments:
#   $1 - Endpoint name
# Outputs:
#   Removal status
#######################################
judge0::webhooks::remove() {
    local name="$1"
    
    if [[ -z "$name" ]]; then
        log::error "Endpoint name is required"
        return 1
    fi
    
    local registry_file="${JUDGE0_CONFIG_DIR}/webhook_registry.json"
    
    if [[ ! -f "$registry_file" ]]; then
        log::error "No webhooks registered"
        return 1
    fi
    
    # Remove endpoint from registry
    local registry=$(cat "$registry_file")
    registry=$(echo "$registry" | jq "del(.\"$name\")")
    
    echo "$registry" > "$registry_file"
    
    log::success "Webhook endpoint removed: $name"
    return 0
}

#######################################
# Test webhook endpoint
# Arguments:
#   $1 - Webhook URL
# Outputs:
#   Test result
#######################################
judge0::webhooks::test() {
    local url="$1"
    
    if [[ -z "$url" ]]; then
        log::error "Webhook URL is required"
        return 1
    fi
    
    log::info "Testing webhook: $url"
    
    # Send test payload
    local test_payload=$(jq -n \
        --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        '{
            event: "test",
            timestamp: $timestamp,
            submission: {
                token: "test-token",
                status: "completed",
                stdout: "Hello from Judge0 webhook test",
                stderr: null,
                compile_output: null,
                exit_code: 0
            }
        }')
    
    local response=$(curl -sf -X POST "$url" \
        -H "Content-Type: application/json" \
        -H "X-Judge0-Webhook: test" \
        -d "$test_payload" \
        -w "\nHTTP_STATUS:%{http_code}" \
        2>/dev/null)
    
    local http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    
    if [[ "$http_status" == "200" ]] || [[ "$http_status" == "201" ]] || [[ "$http_status" == "204" ]]; then
        log::success "Webhook test successful (HTTP $http_status)"
        return 0
    else
        log::error "Webhook test failed (HTTP ${http_status:-000})"
        return 1
    fi
}

#######################################
# Monitor webhook activity
# Outputs:
#   Real-time webhook activity
#######################################
judge0::webhooks::monitor() {
    log::header "Monitoring Webhook Activity"
    log::info "Press Ctrl+C to stop monitoring"
    
    local activity_log="${JUDGE0_LOGS_DIR}/webhook_activity.log"
    
    # Create activity log if it doesn't exist
    touch "$activity_log"
    
    # Monitor activity log
    tail -f "$activity_log" | while read -r line; do
        echo "[$(date +'%H:%M:%S')] $line"
    done
}

#######################################
# Process webhook callback (internal)
# Arguments:
#   $1 - Submission token
#   $2 - Status
#   $3 - Result data (JSON)
#######################################
judge0::webhooks::process_callback() {
    local token="$1"
    local status="$2"
    local result="$3"
    
    # Load webhook configuration
    local config_file="${JUDGE0_CONFIG_DIR}/webhooks.json"
    if [[ ! -f "$config_file" ]]; then
        return 0  # No webhook configured
    fi
    
    local config=$(cat "$config_file")
    local enabled=$(echo "$config" | jq -r '.enabled')
    
    if [[ "$enabled" != "true" ]]; then
        return 0  # Webhooks disabled
    fi
    
    local webhook_url=$(echo "$config" | jq -r '.url')
    local events=$(echo "$config" | jq -r '.events[]')
    
    # Check if this event should trigger webhook
    local should_trigger=false
    for event in $events; do
        if [[ "$status" == "$event" ]]; then
            should_trigger=true
            break
        fi
    done
    
    if [[ "$should_trigger" != "true" ]]; then
        return 0
    fi
    
    # Prepare webhook payload
    local payload=$(jq -n \
        --arg token "$token" \
        --arg status "$status" \
        --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        --argjson result "$result" \
        '{
            event: "submission_\($status)",
            timestamp: $timestamp,
            submission: ({token: $token, status: $status} + $result)
        }')
    
    # Send webhook
    local retry_attempts=$(echo "$config" | jq -r '.retry_attempts // 3')
    local retry_delay=$(echo "$config" | jq -r '.retry_delay // 1000')
    
    for ((i=1; i<=retry_attempts; i++)); do
        local response=$(curl -sf -X POST "$webhook_url" \
            -H "Content-Type: application/json" \
            -H "X-Judge0-Webhook: true" \
            -d "$payload" \
            -w "\nHTTP_STATUS:%{http_code}" \
            2>/dev/null)
        
        local http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
        
        if [[ "$http_status" == "200" ]] || [[ "$http_status" == "201" ]] || [[ "$http_status" == "204" ]]; then
            # Log successful webhook
            echo "Webhook delivered: token=$token, status=$status, url=$webhook_url" >> "${JUDGE0_LOGS_DIR}/webhook_activity.log"
            return 0
        fi
        
        # Wait before retry
        if [[ $i -lt $retry_attempts ]]; then
            sleep $((retry_delay / 1000))
        fi
    done
    
    # Log failed webhook
    echo "Webhook failed: token=$token, status=$status, url=$webhook_url, attempts=$retry_attempts" >> "${JUDGE0_LOGS_DIR}/webhook_activity.log"
    return 1
}