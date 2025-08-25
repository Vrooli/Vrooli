#!/usr/bin/env bash
# Local credential registry for tracking n8n credentials created by auto-credentials
# Since n8n API doesn't support listing credentials, we maintain a local registry

set -euo pipefail

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_CREDENTIAL_REGISTRY_SOURCED:-}" ]] && return 0
export _N8N_CREDENTIAL_REGISTRY_SOURCED=1

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
REGISTRY_DIR="${APP_ROOT}/resources/n8n/lib"

# shellcheck disable=SC1091
source "${REGISTRY_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${REGISTRY_DIR}/../../../../lib/utils/log.sh}"

# Registry file path
CREDENTIAL_REGISTRY_FILE="${var_ROOT_DIR}/data/resources/n8n/n8n-credentials-registry.json"

#######################################
# Initialize credential registry
#######################################
credential_registry::init() {
    local registry_dir
    registry_dir=${CREDENTIAL_REGISTRY_FILE%/*
    
    # Create directory if it doesn't exist
    mkdir -p "$registry_dir"
    
    # Create registry file if it doesn't exist
    if [[ ! -f "$CREDENTIAL_REGISTRY_FILE" ]]; then
        echo '{"credentials": []}' > "$CREDENTIAL_REGISTRY_FILE"
        log::debug "Created credential registry: $CREDENTIAL_REGISTRY_FILE"
    fi
}

#######################################
# Add credential to registry
# Args: $1 - credential name, $2 - credential id, $3 - type, $4 - resource
#######################################
credential_registry::add() {
    local name="$1"
    local id="$2"
    local type="$3"
    local resource="$4"
    
    credential_registry::init
    
    # Remove any existing entry with the same name first
    credential_registry::remove_by_name "$name"
    
    # Create new entry
    local new_entry
    new_entry=$(jq -n \
        --arg name "$name" \
        --arg id "$id" \
        --arg type "$type" \
        --arg resource "$resource" \
        --arg created "$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")" \
        '{
            name: $name,
            id: $id,
            type: $type,
            resource: $resource,
            created: $created
        }')
    
    # Add to registry
    local updated_registry
    updated_registry=$(jq --argjson entry "$new_entry" '.credentials += [$entry]' "$CREDENTIAL_REGISTRY_FILE")
    echo "$updated_registry" > "$CREDENTIAL_REGISTRY_FILE"
    
    log::debug "Added credential to registry: $name (ID: $id)"
}

#######################################
# Check if credential exists in registry
# Args: $1 - credential name
# Returns: 0 if exists, 1 if not
#######################################
credential_registry::exists() {
    local name="$1"
    
    credential_registry::init
    
    if jq -e --arg name "$name" '.credentials[] | select(.name == $name)' "$CREDENTIAL_REGISTRY_FILE" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

#######################################
# Get credential ID by name
# Args: $1 - credential name
# Returns: credential ID or empty string
#######################################
credential_registry::get_id() {
    local name="$1"
    
    credential_registry::init
    
    jq -r --arg name "$name" '.credentials[] | select(.name == $name) | .id' "$CREDENTIAL_REGISTRY_FILE" 2>/dev/null || echo ""
}

#######################################
# Remove credential from registry by name
# Args: $1 - credential name
#######################################
credential_registry::remove_by_name() {
    local name="$1"
    
    credential_registry::init
    
    local updated_registry
    updated_registry=$(jq --arg name "$name" '.credentials = (.credentials | map(select(.name != $name)))' "$CREDENTIAL_REGISTRY_FILE")
    echo "$updated_registry" > "$CREDENTIAL_REGISTRY_FILE"
    
    log::debug "Removed credential from registry: $name"
}

#######################################
# Remove credential from registry by ID
# Args: $1 - credential ID
#######################################
credential_registry::remove_by_id() {
    local id="$1"
    
    credential_registry::init
    
    local updated_registry
    updated_registry=$(jq --arg id "$id" '.credentials = (.credentials | map(select(.id != $id)))' "$CREDENTIAL_REGISTRY_FILE")
    echo "$updated_registry" > "$CREDENTIAL_REGISTRY_FILE"
    
    log::debug "Removed credential from registry: ID $id"
}

#######################################
# List all credentials in registry
#######################################
credential_registry::list() {
    credential_registry::init
    
    jq '.credentials[]' "$CREDENTIAL_REGISTRY_FILE" 2>/dev/null || echo ""
}

#######################################
# Count credentials in registry
#######################################
credential_registry::count() {
    credential_registry::init
    
    jq '.credentials | length' "$CREDENTIAL_REGISTRY_FILE" 2>/dev/null || echo "0"
}

#######################################
# Clean up registry - remove entries that no longer exist in n8n
# This would require testing each credential ID, but since we can't list credentials
# in n8n, we'll need to implement this differently or skip it for now
#######################################
credential_registry::cleanup() {
    log::debug "Registry cleanup not implemented - n8n API doesn't support credential listing"
    # TODO: Implement when n8n API supports credential verification
}

#######################################
# Get registry file path (for debugging)
#######################################
credential_registry::get_file_path() {
    echo "$CREDENTIAL_REGISTRY_FILE"
}

#######################################
# Backup registry
#######################################
credential_registry::backup() {
    credential_registry::init
    
    local backup_file="${CREDENTIAL_REGISTRY_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$CREDENTIAL_REGISTRY_FILE" "$backup_file"
    log::info "Registry backed up to: $backup_file"
}

#######################################
# Show registry statistics
#######################################
credential_registry::stats() {
    credential_registry::init
    
    local total_count type_counts
    total_count=$(credential_registry::count)
    
    echo "📊 Credential Registry Statistics:"
    echo "   Total credentials: $total_count"
    echo "   Registry file: $CREDENTIAL_REGISTRY_FILE"
    
    if [[ "$total_count" -gt 0 ]]; then
        echo "   By type:"
        type_counts=$(jq -r '.credentials | group_by(.type) | map("\(.length) \(.[0].type)") | .[]' "$CREDENTIAL_REGISTRY_FILE" 2>/dev/null || echo "")
        if [[ -n "$type_counts" ]]; then
            while IFS= read -r line; do
                echo "     • $line"
            done <<< "$type_counts"
        fi
        
        echo "   By resource:"
        jq -r '.credentials | group_by(.resource) | map("\(.length) \(.[0].resource)") | .[]' "$CREDENTIAL_REGISTRY_FILE" 2>/dev/null | while IFS= read -r line; do
            echo "     • $line"
        done
    fi
}