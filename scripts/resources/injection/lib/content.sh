#!/usr/bin/env bash
################################################################################
# Injection System v2.0 - Content Management
# Functions for adding content to resources using v2.0 contract
################################################################################
set -euo pipefail

#######################################
# Add content to a specific resource
# Arguments:
#   $1 - resource name
#   $2 - resource configuration (JSON)
# Returns:
#   0 on success, 1 on failure
#######################################
content::add_to_resource() {
    local resource="$1"
    local config="$2"
    
    # Check if resource CLI is available
    local resource_cli="resource-${resource}"
    if ! command -v "$resource_cli" >/dev/null 2>&1; then
        # Try direct path
        resource_cli="${APP_ROOT}/resources/${resource}/cli.sh"
        if [[ ! -f "$resource_cli" ]]; then
            log::warn "Resource CLI not found: $resource"
            return 1
        fi
    fi
    
    # Check if resource is running
    if ! "$resource_cli" status >/dev/null 2>&1; then
        log::warn "Resource not running: $resource"
        return 1
    fi
    
    # Extract content array from config
    local content_items
    content_items=$(echo "$config" | jq -c '.content[]' 2>/dev/null || echo "")
    
    if [[ -z "$content_items" ]]; then
        log::debug "No content defined for $resource"
        return 0
    fi
    
    # Process each content item
    local item_count=0
    local success_count=0
    
    while IFS= read -r item; do
        ((item_count++))
        
        if [[ "$INJECTION_DRY_RUN" == "true" ]]; then
            log::info "  [DRY RUN] Would add content to $resource"
            content::show_item_details "$item"
            ((success_count++))
        else
            if content::add_single_item "$resource" "$resource_cli" "$item"; then
                ((success_count++))
            fi
        fi
    done <<< "$content_items"
    
    if [[ $success_count -eq $item_count ]]; then
        log::info "  Added $success_count items to $resource"
        return 0
    else
        log::warn "  Added $success_count of $item_count items to $resource"
        return 1
    fi
}

#######################################
# Add single content item to resource
# Arguments:
#   $1 - resource name
#   $2 - resource CLI command
#   $3 - content item (JSON)
#######################################
content::add_single_item() {
    local resource="$1"
    local resource_cli="$2"
    local item="$3"
    
    # Extract item properties
    local type
    local file
    local name
    
    type=$(echo "$item" | jq -r '.type // "unknown"')
    file=$(echo "$item" | jq -r '.file // ""')
    name=$(echo "$item" | jq -r '.name // ""')
    
    # Resolve file path (handle relative paths from scenarios)
    if [[ -n "$file" ]] && [[ ! -f "$file" ]]; then
        # Try relative to scenarios directory
        local alt_file="${var_SCENARIOS_DIR}/${file}"
        if [[ -f "$alt_file" ]]; then
            file="$alt_file"
        fi
    fi
    
    # Validate file exists
    if [[ -n "$file" ]] && [[ ! -f "$file" ]]; then
        log::error "  File not found: $file"
        return 1
    fi
    
    # Use v2.0 content add pattern
    local add_cmd="$resource_cli content add"
    
    # Build command based on resource type and content
    case "$resource" in
        n8n)
            # N8n workflows
            add_cmd="$add_cmd workflows --file \"$file\""
            ;;
        postgres)
            # Database schemas/migrations
            add_cmd="$add_cmd schema --file \"$file\""
            ;;
        qdrant)
            # Vector collections
            add_cmd="$add_cmd collection --config \"$item\""
            ;;
        minio)
            # Storage buckets/files
            add_cmd="$add_cmd bucket --name \"$name\""
            ;;
        *)
            # Generic content add
            add_cmd="$add_cmd --type \"$type\" --file \"$file\""
            ;;
    esac
    
    if [[ "$INJECTION_VERBOSE" == "true" ]]; then
        log::info "  Executing: $add_cmd"
    fi
    
    # Execute the command
    if eval "$add_cmd" >/dev/null 2>&1; then
        log::debug "  ✅ Added: $name ($type)"
        return 0
    else
        log::error "  ❌ Failed to add: $name ($type)"
        return 1
    fi
}

#######################################
# Show details of content item (for dry-run)
# Arguments:
#   $1 - content item (JSON)
#######################################
content::show_item_details() {
    local item="$1"
    
    local type
    local file
    local name
    
    type=$(echo "$item" | jq -r '.type // "unknown"')
    file=$(echo "$item" | jq -r '.file // ""')
    name=$(echo "$item" | jq -r '.name // ""')
    
    echo "    Type: $type"
    [[ -n "$name" ]] && echo "    Name: $name"
    [[ -n "$file" ]] && echo "    File: $file"
}