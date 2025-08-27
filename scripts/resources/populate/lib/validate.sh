#!/usr/bin/env bash
################################################################################
# Population System v2.0 - Validation
# Validation functions for scenarios and content
################################################################################
set -euo pipefail

#######################################
# Validate scenario configuration
# Arguments:
#   $1 - scenario configuration (JSON)
# Returns:
#   0 if valid, 1 if invalid
#######################################
validate::scenario() {
    local config="$1"
    local is_valid=true
    
    log::info "Validating scenario structure..."
    
    # Check basic structure
    if ! echo "$config" | jq -e . >/dev/null 2>&1; then
        log::error "Invalid JSON format"
        return 1
    fi
    
    # Check for required fields (support both old .name and new .service.name paths)
    local name
    name=$(echo "$config" | jq -r '.service.name // .name // ""')
    if [[ -z "$name" ]]; then
        log::error "Missing required field: name (expected at .service.name or .name)"
        is_valid=false
    fi
    
    # Check resources section
    local resources
    resources=$(echo "$config" | jq -r '.resources // {}')
    if [[ "$resources" == "{}" ]]; then
        log::warn "No resources defined in scenario"
    fi
    
    # Validate each resource configuration
    local resource_names
    resource_names=$(echo "$config" | jq -r '.resources | keys[]' 2>/dev/null || true)
    
    for resource in $resource_names; do
        if ! validate::resource_config "$resource" "$(echo "$config" | jq -c ".resources[\"$resource\"]")"; then
            is_valid=false
        fi
    done
    
    if [[ "$is_valid" == "true" ]]; then
        log::info "✅ Scenario structure is valid"
        return 0
    else
        return 1
    fi
}

#######################################
# Validate resource configuration
# Arguments:
#   $1 - resource name
#   $2 - resource configuration (JSON)
# Returns:
#   0 if valid, 1 if invalid
#######################################
validate::resource_config() {
    local resource="$1"
    local config="$2"
    local is_valid=true
    
    log::debug "  Validating $resource configuration..."
    
    # Check if resource has content array
    local content
    content=$(echo "$config" | jq -c '.content[]' 2>/dev/null || echo "")
    
    if [[ -z "$content" ]]; then
        log::debug "    No content defined for $resource"
        return 0
    fi
    
    # Validate each content item
    local item_index=0
    while IFS= read -r item; do
        ((item_index++))
        
        if ! validate::content_item "$resource" "$item" "$item_index"; then
            is_valid=false
        fi
    done <<< "$content"
    
    if [[ "$is_valid" == "true" ]]; then
        log::debug "    ✅ $resource configuration is valid"
        return 0
    else
        log::error "    ❌ $resource configuration has errors"
        return 1
    fi
}

#######################################
# Validate single content item
# Arguments:
#   $1 - resource name
#   $2 - content item (JSON)
#   $3 - item index
# Returns:
#   0 if valid, 1 if invalid
#######################################
validate::content_item() {
    local resource="$1"
    local item="$2"
    local index="$3"
    local is_valid=true
    
    # Extract fields
    local type
    local file
    local name
    
    type=$(echo "$item" | jq -r '.type // ""')
    file=$(echo "$item" | jq -r '.file // ""')
    name=$(echo "$item" | jq -r '.name // ""')
    
    # Type is required
    if [[ -z "$type" ]]; then
        log::error "      Item $index: Missing required field 'type'"
        is_valid=false
    fi
    
    # Validate based on resource type
    case "$resource" in
        n8n)
            if [[ "$type" != "workflow" ]] && [[ "$type" != "credential" ]]; then
                log::error "      Item $index: Invalid type '$type' for n8n (expected: workflow, credential)"
                is_valid=false
            fi
            ;;
        postgres)
            if [[ "$type" != "schema" ]] && [[ "$type" != "migration" ]] && [[ "$type" != "seed" ]]; then
                log::error "      Item $index: Invalid type '$type' for postgres (expected: schema, migration, seed)"
                is_valid=false
            fi
            ;;
        qdrant)
            if [[ "$type" != "collection" ]] && [[ "$type" != "vectors" ]]; then
                log::error "      Item $index: Invalid type '$type' for qdrant (expected: collection, vectors)"
                is_valid=false
            fi
            ;;
    esac
    
    # If file is specified, check it exists (with path resolution)
    if [[ -n "$file" ]]; then
        local resolved_file="$file"
        
        # Try to resolve relative paths
        if [[ ! -f "$resolved_file" ]]; then
            # Try relative to scenarios directory
            resolved_file="${var_SCENARIOS_DIR}/${file}"
        fi
        
        if [[ ! -f "$resolved_file" ]]; then
            log::warn "      Item $index: File not found: $file"
            # This is a warning, not an error - file might be created later
        fi
    fi
    
    if [[ "$is_valid" == "true" ]]; then
        return 0
    else
        return 1
    fi
}