#!/usr/bin/env bash
################################################################################
# SQLite Resource - Replication Library
#
# Basic replication functionality for SQLite databases
################################################################################

set -euo pipefail

# Set default data path if not defined
SQLITE_DATA_PATH="${SQLITE_DATA_PATH:-${VROOLI_DATA:-${HOME}/.vrooli/data}/sqlite}"

# Initialize replication configuration
sqlite::replication::init() {
    local config_dir="${SQLITE_DATA_PATH}/replication"
    
    if [[ ! -d "$config_dir" ]]; then
        mkdir -p "$config_dir"
        chmod 700 "$config_dir"
    fi
    
    # Create replication state file if not exists
    local state_file="${config_dir}/state.json"
    if [[ ! -f "$state_file" ]]; then
        echo '{"replicas": [], "last_sync": null}' > "$state_file"
        chmod 600 "$state_file"
    fi
    
    return 0
}

# Add a replica target
sqlite::replication::add_replica() {
    local db_name="${1:-}"
    local target_path="${2:-}"
    local sync_interval="${3:-60}"  # Default 60 seconds
    
    # Validate inputs
    if ! sqlite::validate_name "$db_name"; then
        return 1
    fi
    
    if [[ -z "$target_path" ]]; then
        log::error "Target path required for replica"
        return 1
    fi
    
    # Ensure target directory exists
    local target_dir="$(dirname "$target_path")"
    if [[ ! -d "$target_dir" ]]; then
        log::error "Target directory does not exist: $target_dir"
        return 1
    fi
    
    # Check if target is writable
    if [[ ! -w "$target_dir" ]]; then
        log::error "Target directory is not writable: $target_dir"
        return 1
    fi
    
    # Initialize if needed
    sqlite::replication::init
    
    # Add replica to configuration
    local config_file="${SQLITE_DATA_PATH}/replication/state.json"
    local temp_file="${config_file}.tmp"
    
    # Use jq to add replica if available, otherwise use sed
    if command -v jq &> /dev/null; then
        jq --arg db "$db_name" --arg path "$target_path" --argjson interval "$sync_interval" \
           '.replicas += [{"database": $db, "target": $path, "interval": $interval, "enabled": true}]' \
           "$config_file" > "$temp_file" && mv "$temp_file" "$config_file"
    else
        # Simple append for systems without jq
        local replica_json='{"database":"'$db_name'","target":"'$target_path'","interval":'$sync_interval',"enabled":true}'
        sed -i 's/\("replicas": \[\)/\1'$replica_json',/' "$config_file"
    fi
    
    log::info "Replica added: $db_name -> $target_path (sync every ${sync_interval}s)"
    return 0
}

# Remove a replica
sqlite::replication::remove_replica() {
    local db_name="${1:-}"
    local target_path="${2:-}"
    
    if ! sqlite::validate_name "$db_name"; then
        return 1
    fi
    
    local config_file="${SQLITE_DATA_PATH}/replication/state.json"
    if [[ ! -f "$config_file" ]]; then
        log::error "No replication configuration found"
        return 1
    fi
    
    # Use jq to remove replica if available
    if command -v jq &> /dev/null; then
        local temp_file="${config_file}.tmp"
        jq --arg db "$db_name" --arg path "$target_path" \
           '.replicas = [.replicas[] | select(.database != $db or .target != $path)]' \
           "$config_file" > "$temp_file" && mv "$temp_file" "$config_file"
        log::info "Replica removed: $db_name -> $target_path"
    else
        log::error "jq required for removing replicas"
        return 1
    fi
    
    return 0
}

# List all configured replicas
sqlite::replication::list() {
    local config_file="${SQLITE_DATA_PATH}/replication/state.json"
    
    if [[ ! -f "$config_file" ]]; then
        log::info "No replicas configured"
        return 0
    fi
    
    if command -v jq &> /dev/null; then
        echo "Configured replicas:"
        jq -r '.replicas[] | "  [\(.database)] -> \(.target) (every \(.interval)s, \(if .enabled then "enabled" else "disabled" end))"' "$config_file"
    else
        log::info "Replica configuration exists at: $config_file"
        cat "$config_file"
    fi
    
    return 0
}

# Sync a database to its replicas
sqlite::replication::sync() {
    local db_name="${1:-}"
    local force="${2:-false}"
    
    if ! sqlite::validate_name "$db_name"; then
        return 1
    fi
    
    local source_db="${SQLITE_DATA_PATH}/databases/${db_name}"
    if [[ ! -f "$source_db" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    local config_file="${SQLITE_DATA_PATH}/replication/state.json"
    if [[ ! -f "$config_file" ]]; then
        log::info "No replicas configured"
        return 0
    fi
    
    # Get replicas for this database
    if command -v jq &> /dev/null; then
        local replicas=$(jq -r --arg db "$db_name" '.replicas[] | select(.database == $db and .enabled) | .target' "$config_file")
        
        if [[ -z "$replicas" ]]; then
            log::info "No active replicas for database: $db_name"
            return 0
        fi
        
        local sync_count=0
        local fail_count=0
        
        while IFS= read -r target; do
            if [[ -n "$target" ]]; then
                log::info "Syncing to: $target"
                
                # Use SQLite's backup API for consistency
                if sqlite3 "$source_db" ".backup '$target'"; then
                    sync_count=$((sync_count + 1))
                    log::info "Synced successfully to: $target"
                else
                    fail_count=$((fail_count + 1))
                    log::error "Failed to sync to: $target"
                fi
            fi
        done <<< "$replicas"
        
        # Update last sync timestamp
        local temp_file="${config_file}.tmp"
        jq --arg db "$db_name" --arg timestamp "$(date -Iseconds)" \
           '.last_sync = {database: $db, timestamp: $timestamp}' \
           "$config_file" > "$temp_file" && mv "$temp_file" "$config_file"
        
        log::info "Replication complete: $sync_count succeeded, $fail_count failed"
        return $([[ $fail_count -eq 0 ]] && echo 0 || echo 1)
    else
        log::error "jq required for replication sync"
        return 1
    fi
}

# Monitor and auto-sync replicas
sqlite::replication::monitor() {
    local interval="${1:-60}"  # Default check every 60 seconds
    
    log::info "Starting replication monitor (checking every ${interval}s)"
    log::info "Press Ctrl+C to stop"
    
    # Trap to handle clean shutdown
    trap 'log::info "Replication monitor stopped"; exit 0' INT TERM
    
    while true; do
        local config_file="${SQLITE_DATA_PATH}/replication/state.json"
        
        if [[ -f "$config_file" ]] && command -v jq &> /dev/null; then
            # Get unique database names that have replicas
            local databases=$(jq -r '.replicas[] | select(.enabled) | .database' "$config_file" | sort -u)
            
            while IFS= read -r db_name; do
                if [[ -n "$db_name" ]]; then
                    log::info "Auto-syncing database: $db_name"
                    sqlite::replication::sync "$db_name" || true
                fi
            done <<< "$databases"
        fi
        
        sleep "$interval"
    done
}

# Enable/disable a replica
sqlite::replication::toggle() {
    local db_name="${1:-}"
    local target_path="${2:-}"
    local enabled="${3:-true}"
    
    if ! sqlite::validate_name "$db_name"; then
        return 1
    fi
    
    local config_file="${SQLITE_DATA_PATH}/replication/state.json"
    if [[ ! -f "$config_file" ]]; then
        log::error "No replication configuration found"
        return 1
    fi
    
    if command -v jq &> /dev/null; then
        local temp_file="${config_file}.tmp"
        local enabled_bool=$([[ "$enabled" == "true" ]] && echo "true" || echo "false")
        
        jq --arg db "$db_name" --arg path "$target_path" --argjson enabled "$enabled_bool" \
           '(.replicas[] | select(.database == $db and .target == $path)) |= . + {enabled: $enabled}' \
           "$config_file" > "$temp_file" && mv "$temp_file" "$config_file"
        
        local action=$([[ "$enabled" == "true" ]] && echo "enabled" || echo "disabled")
        log::info "Replica $action: $db_name -> $target_path"
    else
        log::error "jq required for toggling replicas"
        return 1
    fi
    
    return 0
}

# Verify replica consistency
sqlite::replication::verify() {
    local db_name="${1:-}"
    
    if ! sqlite::validate_name "$db_name"; then
        return 1
    fi
    
    local source_db="${SQLITE_DATA_PATH}/databases/${db_name}"
    if [[ ! -f "$source_db" ]]; then
        log::error "Database not found: $db_name"
        return 1
    fi
    
    local config_file="${SQLITE_DATA_PATH}/replication/state.json"
    if [[ ! -f "$config_file" ]]; then
        log::info "No replicas configured"
        return 0
    fi
    
    if command -v jq &> /dev/null; then
        local replicas=$(jq -r --arg db "$db_name" '.replicas[] | select(.database == $db and .enabled) | .target' "$config_file")
        
        if [[ -z "$replicas" ]]; then
            log::info "No active replicas for database: $db_name"
            return 0
        fi
        
        # Compare table counts and sizes
        local source_stats=$(sqlite3 "$source_db" "SELECT COUNT(*) FROM sqlite_master WHERE type='table';" 2>/dev/null)
        
        log::info "Source has $source_stats tables"
        
        local consistent=true
        while IFS= read -r target; do
            if [[ -n "$target" ]] && [[ -f "$target" ]]; then
                local target_stats=$(sqlite3 "$target" "SELECT COUNT(*) FROM sqlite_master WHERE type='table';" 2>/dev/null)
                
                if [[ "$source_stats" == "$target_stats" ]]; then
                    # Also check integrity
                    local target_integrity=$(sqlite3 "$target" "PRAGMA integrity_check;" 2>/dev/null)
                    if [[ "$target_integrity" == "ok" ]]; then
                        log::info "✓ Replica consistent: $target ($target_stats tables)"
                    else
                        log::error "✗ Replica corrupt: $target (integrity: $target_integrity)"
                        consistent=false
                    fi
                else
                    log::error "✗ Replica inconsistent: $target (tables: source=$source_stats, target=$target_stats)"
                    consistent=false
                fi
            else
                log::warning "Replica not found: $target"
                consistent=false
            fi
        done <<< "$replicas"
        
        if $consistent; then
            log::success "All replicas are consistent"
            return 0
        else
            log::error "Some replicas are inconsistent"
            return 1
        fi
    else
        log::error "jq required for replica verification"
        return 1
    fi
}

# Export functions for CLI use
export -f sqlite::replication::init
export -f sqlite::replication::add_replica
export -f sqlite::replication::remove_replica
export -f sqlite::replication::list
export -f sqlite::replication::sync
export -f sqlite::replication::monitor
export -f sqlite::replication::toggle
export -f sqlite::replication::verify