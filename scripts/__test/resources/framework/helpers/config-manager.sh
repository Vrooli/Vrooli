#!/bin/bash
# ====================================================================
# Configuration Management for Resource Tests
# ====================================================================
#
# Provides robust configuration management including validation,
# repair, backup, and restoration functionality.
#
# Functions:
#   - validate_config()      - Validate configuration file
#   - repair_config()        - Repair corrupted configuration
#   - backup_config()        - Create timestamped backup
#   - restore_config()       - Restore from backup
#   - merge_configs()        - Merge configurations
#   - ensure_config_exists() - Ensure valid config exists
#
# ====================================================================

# Configuration file paths
CONFIG_FILE="${CONFIG_FILE:-$HOME/.vrooli/resources.local.json}"
CONFIG_DIR="$(dirname "$CONFIG_FILE")"

# Validate configuration file
validate_config() {
    local config_file="${1:-$CONFIG_FILE}"
    
    # Check if file exists and is readable
    if [[ ! -r "$config_file" ]]; then
        return 1
    fi
    
    # Check if file is not empty
    if [[ ! -s "$config_file" ]]; then
        return 1
    fi
    
    # Check if file is valid JSON
    if ! jq empty "$config_file" 2>/dev/null; then
        return 1
    fi
    
    # Check required structure
    if ! jq -e '.services | type == "object"' "$config_file" >/dev/null 2>&1; then
        return 1
    fi
    
    # Check for at least one service category
    local categories
    categories=$(jq -r '.services | keys[]' "$config_file" 2>/dev/null | wc -l)
    if [[ "$categories" -eq 0 ]]; then
        return 1
    fi
    
    return 0
}

# Create a backup of the configuration
backup_config() {
    local config_file="${1:-$CONFIG_FILE}"
    local backup_file="${config_file}.backup.$(date +%Y-%m-%dT%H-%M-%S)"
    
    if [[ -f "$config_file" ]]; then
        cp "$config_file" "$backup_file"
        echo "$backup_file"
    fi
}

# Repair corrupted configuration
repair_config() {
    local config_file="${1:-$CONFIG_FILE}"
    
    echo "🔧 Repairing configuration file: $config_file"
    
    # Backup current file if it exists
    if [[ -f "$config_file" ]]; then
        local backup_file
        backup_file=$(backup_config "$config_file")
        echo "📦 Created backup: $backup_file"
    fi
    
    # Try to find a valid backup
    local latest_valid_backup=""
    for backup in "$config_file".backup.*; do
        if [[ -f "$backup" ]] && validate_config "$backup"; then
            latest_valid_backup="$backup"
        fi
    done
    
    if [[ -n "$latest_valid_backup" ]]; then
        echo "✅ Found valid backup: $latest_valid_backup"
        cp "$latest_valid_backup" "$config_file"
        chmod 600 "$config_file"
        return 0
    fi
    
    # Generate new configuration from discovered resources
    echo "🔍 No valid backup found, generating from discovered resources..."
    generate_config_from_discovery "$config_file"
}

# Generate configuration from resource discovery
generate_config_from_discovery() {
    local config_file="${1:-$CONFIG_FILE}"
    local resources_dir="${RESOURCES_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)}"
    
    # Ensure directory exists
    mkdir -p "$(dirname "$config_file")"
    
    # Discover resources
    local discovery_output
    discovery_output=$("$resources_dir/index.sh" --action discover 2>&1)
    
    # Initialize configuration structure
    local config='{
        "services": {
            "ai": {},
            "automation": {},
            "agents": {},
            "search": {},
            "storage": {}
        }
    }'
    
    # Parse discovered resources
    while IFS= read -r line; do
        if [[ "$line" =~ ✅[[:space:]]+([^[:space:]]+)[[:space:]]+is[[:space:]]+running[[:space:]]+on[[:space:]]+port[[:space:]]+([0-9]+) ]]; then
            local resource="${BASH_REMATCH[1]}"
            local port="${BASH_REMATCH[2]}"
            local category=$(get_resource_category "$resource")
            
            config=$(echo "$config" | jq \
                --arg cat "$category" \
                --arg res "$resource" \
                --arg url "http://localhost:$port" \
                '.services[$cat][$res] = {"enabled": true, "baseUrl": $url}')
        fi
    done <<< "$discovery_output"
    
    # Write configuration
    echo "$config" | jq . > "$config_file"
    chmod 600 "$config_file"
    
    echo "✅ Generated configuration with discovered resources"
}

# Get resource category
get_resource_category() {
    local resource="$1"
    
    case "$resource" in
        ollama|whisper|unstructured-io|comfyui)
            echo "ai" ;;
        n8n|node-red|windmill|huginn)
            echo "automation" ;;
        agent-s2|browserless|claude-code)
            echo "agents" ;;
        searxng)
            echo "search" ;;
        minio|vault|qdrant|questdb|postgres)
            echo "storage" ;;
        *)
            echo "other" ;;
    esac
}

# Merge two configurations
merge_configs() {
    local base_config="$1"
    local overlay_config="$2"
    
    jq -s '.[0] * .[1]' "$base_config" "$overlay_config" 2>/dev/null
}

# Ensure valid configuration exists
ensure_config_exists() {
    local config_file="${1:-$CONFIG_FILE}"
    
    if validate_config "$config_file"; then
        echo "✅ Configuration is valid"
        return 0
    fi
    
    echo "⚠️  Configuration is invalid or missing"
    repair_config "$config_file"
    
    if validate_config "$config_file"; then
        echo "✅ Configuration repaired successfully"
        return 0
    else
        echo "❌ Failed to repair configuration"
        return 1
    fi
}

# Update resource status in configuration
update_resource_status() {
    local resource="$1"
    local enabled="$2"
    local config_file="${3:-$CONFIG_FILE}"
    
    if ! validate_config "$config_file"; then
        echo "❌ Invalid configuration file"
        return 1
    fi
    
    local category
    category=$(get_resource_category "$resource")
    
    # Update the resource status
    local updated_config
    updated_config=$(jq \
        --arg cat "$category" \
        --arg res "$resource" \
        --argjson enabled "$enabled" \
        '.services[$cat][$res].enabled = $enabled' \
        "$config_file")
    
    # Write back to file
    echo "$updated_config" > "$config_file"
    echo "✅ Updated $resource enabled=$enabled"
}

# Clean old backups (keep last N backups)
clean_old_backups() {
    local keep_count="${1:-10}"
    local config_file="${2:-$CONFIG_FILE}"
    
    # Find and sort backup files
    local backups=()
    while IFS= read -r backup; do
        backups+=("$backup")
    done < <(ls -t "${config_file}.backup."* 2>/dev/null || true)
    
    # Remove old backups
    local removed=0
    for ((i=$keep_count; i<${#backups[@]}; i++)); do
        rm -f "${backups[$i]}"
        removed=$((removed + 1))
    done
    
    if [[ $removed -gt 0 ]]; then
        echo "🧹 Removed $removed old backup(s)"
    fi
}

# Show configuration status
show_config_status() {
    local config_file="${1:-$CONFIG_FILE}"
    
    echo "Configuration Status:"
    echo "===================="
    echo "File: $config_file"
    
    if [[ ! -f "$config_file" ]]; then
        echo "Status: ❌ Not found"
        return
    fi
    
    local size
    size=$(stat -f%z "$config_file" 2>/dev/null || stat -c%s "$config_file" 2>/dev/null || echo "0")
    echo "Size: $size bytes"
    
    if validate_config "$config_file"; then
        echo "Status: ✅ Valid"
        
        # Count resources
        local total_resources
        total_resources=$(jq '[.services[][] | keys[]] | length' "$config_file" 2>/dev/null || echo "0")
        echo "Total resources: $total_resources"
        
        local enabled_resources
        enabled_resources=$(jq '[.services | to_entries[] | .value | to_entries[] | select(.value.enabled == true)] | length' "$config_file" 2>/dev/null || echo "0")
        echo "Enabled resources: $enabled_resources"
        
        # Show categories
        echo -e "\nCategories:"
        jq -r '.services | to_entries[] | "  - \(.key): \(.value | length) resources"' "$config_file" 2>/dev/null
    else
        echo "Status: ❌ Invalid"
    fi
    
    # Show backups
    local backup_count
    backup_count=$(ls "${config_file}.backup."* 2>/dev/null | wc -l || echo "0")
    echo -e "\nBackups: $backup_count"
}

# If sourced, don't execute anything
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Script is being executed directly
    case "${1:-}" in
        validate)
            validate_config "${2:-}"
            ;;
        repair)
            repair_config "${2:-}"
            ;;
        backup)
            backup_config "${2:-}"
            ;;
        ensure)
            ensure_config_exists "${2:-}"
            ;;
        status)
            show_config_status "${2:-}"
            ;;
        clean)
            clean_old_backups "${2:-10}" "${3:-}"
            ;;
        *)
            echo "Usage: $0 {validate|repair|backup|ensure|status|clean} [config_file]"
            exit 1
            ;;
    esac
fi