#!/usr/bin/env bash
# Vault Audit and Access Control - Enhanced audit logging and access management
# Provides comprehensive audit capabilities and access control management

# Source required libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VAULT_AUDIT_DIR="${APP_ROOT}/resources/vault/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/vault/config/defaults.sh"
# shellcheck disable=SC1091
source "${VAULT_AUDIT_DIR}/common.sh"
# shellcheck disable=SC1091
source "${VAULT_AUDIT_DIR}/api.sh"

# Audit log directory
VAULT_AUDIT_LOG_DIR="${VAULT_DATA_DIR}/audit"

#######################################
# Enable audit logging
# Arguments:
#   $1 - audit device name (optional, default: "file")
#   $2 - log path (optional, default: /vault/audit/audit.log)
#######################################
vault::audit::enable() {
    local device_name="${1:-file}"
    local log_path="${2:-/vault/audit/audit.log}"
    
    log::info "Enabling audit logging..."
    
    # Check if already enabled
    if vault::audit::is_enabled "$device_name"; then
        log::warn "Audit device '$device_name' is already enabled"
        return 0
    fi
    
    # Create audit directory if needed
    mkdir -p "$VAULT_AUDIT_LOG_DIR"
    
    # Enable audit device
    local payload
    payload=$(jq -n \
        --arg path "$log_path" \
        '{
            type: "file",
            options: {
                file_path: $path,
                log_raw: "false",
                hmac_accessor: "true",
                mode: "0600",
                format: "json"
            }
        }')
    
    if vault::api_request "PUT" "/v1/sys/audit/${device_name}" "$payload" >/dev/null 2>&1; then
        log::success "✅ Audit logging enabled: $device_name"
        return 0
    else
        log::error "Failed to enable audit logging"
        return 1
    fi
}

#######################################
# Disable audit logging
# Arguments:
#   $1 - audit device name (optional, default: "file")
#######################################
vault::audit::disable() {
    local device_name="${1:-file}"
    
    log::info "Disabling audit logging..."
    
    if ! vault::audit::is_enabled "$device_name"; then
        log::warn "Audit device '$device_name' is not enabled"
        return 0
    fi
    
    if vault::api_request "DELETE" "/v1/sys/audit/${device_name}" >/dev/null 2>&1; then
        log::success "✅ Audit logging disabled: $device_name"
        return 0
    else
        log::error "Failed to disable audit logging"
        return 1
    fi
}

#######################################
# Check if audit device is enabled
# Arguments:
#   $1 - audit device name
# Returns: 0 if enabled, 1 otherwise
#######################################
vault::audit::is_enabled() {
    local device_name="${1:-file}"
    
    local response
    response=$(vault::api_request "GET" "/v1/sys/audit" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        return 1
    fi
    
    echo "$response" | jq -e --arg name "$device_name" '.[$name]' >/dev/null 2>&1
}

#######################################
# List audit devices
#######################################
vault::audit::list() {
    log::header "Audit Devices"
    
    local response
    response=$(vault::api_request "GET" "/v1/sys/audit" 2>/dev/null)
    
    if [[ -z "$response" ]] || [[ "$response" == "{}" ]] || [[ "$response" == "null" ]]; then
        log::info "No audit devices configured"
        return 0
    fi
    
    # Check if response is an error string
    if ! echo "$response" | jq -e . >/dev/null 2>&1; then
        log::info "No audit devices configured"
        return 0
    fi
    
    # Parse and display audit devices
    local devices
    devices=$(echo "$response" | jq -r 'to_entries[] | "• \(.key): \(.value.type) (Path: \(.value.options.file_path // .value.path // "N/A"))"' 2>/dev/null)
    
    if [[ -n "$devices" ]]; then
        echo "$devices"
    else
        log::info "No audit devices configured"
    fi
}

#######################################
# Create access control policy
# Arguments:
#   $1 - policy name
#   $2 - policy HCL content or file path
#######################################
vault::access::create_policy() {
    local policy_name="${1:-}"
    local policy_content="${2:-}"
    
    if [[ -z "$policy_name" ]] || [[ -z "$policy_content" ]]; then
        log::error "Usage: vault::access::create_policy <name> <content|file>"
        return 1
    fi
    
    log::info "Creating access policy: $policy_name"
    
    # Check if content is a file
    if [[ -f "$policy_content" ]]; then
        policy_content=$(cat "$policy_content")
    fi
    
    # Create policy payload
    local payload
    payload=$(jq -n --arg policy "$policy_content" '{ policy: $policy }')
    
    if vault::api_request "PUT" "/v1/sys/policies/acl/${policy_name}" "$payload" >/dev/null 2>&1; then
        log::success "✅ Policy created: $policy_name"
        return 0
    else
        log::error "Failed to create policy: $policy_name"
        return 1
    fi
}

#######################################
# Delete access control policy
# Arguments:
#   $1 - policy name
#######################################
vault::access::delete_policy() {
    local policy_name="${1:-}"
    
    if [[ -z "$policy_name" ]]; then
        log::error "Usage: vault::access::delete_policy <name>"
        return 1
    fi
    
    log::info "Deleting access policy: $policy_name"
    
    if vault::api_request "DELETE" "/v1/sys/policies/acl/${policy_name}" >/dev/null 2>&1; then
        log::success "✅ Policy deleted: $policy_name"
        return 0
    else
        log::error "Failed to delete policy: $policy_name"
        return 1
    fi
}

#######################################
# List access control policies
#######################################
vault::access::list_policies() {
    log::header "Access Control Policies"
    
    local response
    response=$(vault::api_request "LIST" "/v1/sys/policies/acl" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        log::info "No policies configured"
        return 0
    fi
    
    local policies
    policies=$(echo "$response" | jq -r '.data.keys[]' 2>/dev/null)
    
    if [[ -z "$policies" ]]; then
        log::info "No policies configured"
    else
        echo "$policies" | while read -r policy; do
            echo "• $policy"
        done
    fi
}

#######################################
# Get policy details
# Arguments:
#   $1 - policy name
#######################################
vault::access::get_policy() {
    local policy_name="${1:-}"
    
    if [[ -z "$policy_name" ]]; then
        log::error "Usage: vault::access::get_policy <name>"
        return 1
    fi
    
    local response
    response=$(vault::api_request "GET" "/v1/sys/policies/acl/${policy_name}" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        log::error "Policy not found: $policy_name"
        return 1
    fi
    
    echo "$response" | jq -r '.data.policy'
}

#######################################
# Create a token with specific policy
# Arguments:
#   $1 - policy name
#   $2 - TTL (optional, default: 1h)
#######################################
vault::access::create_token() {
    local policy_name="${1:-}"
    local ttl="${2:-1h}"
    
    if [[ -z "$policy_name" ]]; then
        log::error "Usage: vault::access::create_token <policy> [ttl]"
        return 1
    fi
    
    log::info "Creating token with policy: $policy_name"
    
    local payload
    payload=$(jq -n \
        --arg policy "$policy_name" \
        --arg ttl "$ttl" \
        '{
            policies: [$policy],
            ttl: $ttl,
            renewable: true
        }')
    
    local response
    response=$(vault::api_request "POST" "/v1/auth/token/create" "$payload" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        local token
        token=$(echo "$response" | jq -r '.auth.client_token')
        log::success "✅ Token created"
        echo "Token: $token"
        echo "TTL: $ttl"
        echo "Policies: $policy_name"
        return 0
    else
        log::error "Failed to create token"
        return 1
    fi
}

#######################################
# Analyze audit logs for security events
# Arguments:
#   $1 - time range (optional, default: last hour)
#######################################
vault::audit::analyze() {
    local time_range="${1:-1h}"
    
    log::header "Audit Log Analysis"
    
    # Check if audit logs exist
    local audit_file="${VAULT_AUDIT_LOG_DIR}/audit.log"
    if [[ ! -f "$audit_file" ]]; then
        # Try to get from container
        docker exec "$VAULT_CONTAINER_NAME" test -f /vault/audit/audit.log 2>/dev/null
        if [[ $? -ne 0 ]]; then
            log::warn "No audit logs found. Enable audit logging first."
            return 1
        fi
        # Copy from container
        docker cp "$VAULT_CONTAINER_NAME:/vault/audit/audit.log" "$audit_file" 2>/dev/null
    fi
    
    if [[ ! -f "$audit_file" ]]; then
        log::warn "No audit logs available"
        return 1
    fi
    
    # Analyze recent events
    echo "Recent Security Events (last $time_range):"
    echo "----------------------------------------"
    
    # Count event types
    local auth_events secret_events policy_events error_events
    auth_events=$(grep -c '"type":"request".*"path":"auth/' "$audit_file" 2>/dev/null || echo 0)
    secret_events=$(grep -c '"type":"request".*"path":"secret/' "$audit_file" 2>/dev/null || echo 0)
    policy_events=$(grep -c '"type":"request".*"path":"sys/policies' "$audit_file" 2>/dev/null || echo 0)
    error_events=$(grep -c '"error":' "$audit_file" 2>/dev/null || echo 0)
    
    echo "• Authentication Events: $auth_events"
    echo "• Secret Operations: $secret_events"
    echo "• Policy Changes: $policy_events"
    echo "• Errors: $error_events"
    
    # Show recent errors if any
    if [[ $error_events -gt 0 ]]; then
        echo ""
        echo "Recent Errors:"
        tail -n 100 "$audit_file" | grep '"error":' | tail -n 5 | jq -r '.error' 2>/dev/null || echo "Unable to parse errors"
    fi
    
    # Show top accessed paths
    echo ""
    echo "Top Accessed Paths:"
    tail -n 1000 "$audit_file" 2>/dev/null | \
        grep -o '"path":"[^"]*"' | \
        cut -d'"' -f4 | \
        sort | uniq -c | sort -rn | head -5 | \
        awk '{printf "• %s: %d accesses\n", $2, $1}'
}

#######################################
# Setup standard Vrooli security policies
#######################################
vault::access::setup_standard_policies() {
    log::header "Setting up standard Vrooli policies"
    
    # Create read-only policy
    local readonly_policy='
    # Read-only access to secrets
    path "secret/data/vrooli/*" {
        capabilities = ["read", "list"]
    }
    path "secret/metadata/vrooli/*" {
        capabilities = ["read", "list"]
    }'
    
    vault::access::create_policy "vrooli-readonly" "$readonly_policy"
    
    # Create developer policy
    local developer_policy='
    # Developer access to secrets
    path "secret/data/vrooli/*" {
        capabilities = ["create", "read", "update", "delete", "list"]
    }
    path "secret/metadata/vrooli/*" {
        capabilities = ["read", "list", "delete"]
    }'
    
    vault::access::create_policy "vrooli-developer" "$developer_policy"
    
    # Create scenario policy
    local scenario_policy='
    # Scenario access to specific paths
    path "secret/data/vrooli/scenarios/{{identity.entity.aliases.auth_token.metadata.scenario}}/*" {
        capabilities = ["read", "list"]
    }
    path "secret/metadata/vrooli/scenarios/{{identity.entity.aliases.auth_token.metadata.scenario}}/*" {
        capabilities = ["read", "list"]
    }'
    
    vault::access::create_policy "vrooli-scenario" "$scenario_policy"
    
    log::success "✅ Standard policies created"
}

# Export functions for use by other scripts
export -f vault::audit::enable
export -f vault::audit::disable
export -f vault::audit::list
export -f vault::audit::analyze
export -f vault::access::create_policy
export -f vault::access::delete_policy
export -f vault::access::list_policies
export -f vault::access::create_token
export -f vault::access::setup_standard_policies