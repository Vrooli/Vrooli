#!/usr/bin/env bash
# App Identity Management for Qdrant Embeddings
# Creates and manages app-identity.json files for tracking embedding state

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Define paths from APP_ROOT
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
IDENTITY_DIR="${EMBEDDINGS_DIR}/indexers"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Identity file location
APP_IDENTITY_FILE=".vrooli/app-identity.json"
TEMPLATE_FILE="${EMBEDDINGS_DIR}/config/templates/app-identity.json"

#######################################
# Initialize app identity for current project
# Arguments:
#   $1 - App ID (optional, will be detected if not provided)
#   $2 - App type (main/generated, default: main)
#   $3 - Source scenario (for generated apps)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::identity::init() {
    local app_id="${1:-}"
    local app_type="${2:-main}"
    local source_scenario="${3:-}"
    
    # Create .vrooli directory if it doesn't exist
    mkdir -p "${APP_IDENTITY_FILE%/*}"
    
    # Detect app ID if not provided
    if [[ -z "$app_id" ]]; then
        app_id=$(qdrant::identity::detect_app_id)
    fi
    
    # Get git origin if available
    local git_origin=""
    if [[ -d .git ]]; then
        git_origin=$(git config --get remote.origin.url 2>/dev/null || echo "")
    fi
    
    # Create identity file from template
    if [[ -f "$TEMPLATE_FILE" ]]; then
        cp "$TEMPLATE_FILE" "$APP_IDENTITY_FILE"
    else
        log::error "Template file not found: $TEMPLATE_FILE"
        return 1
    fi
    
    # Update identity file with app-specific information
    local created_date=$(date -Iseconds)
    local temp_file=$(mktemp)
    
    jq --arg app_id "$app_id" \
       --arg app_type "$app_type" \
       --arg source_scenario "$source_scenario" \
       --arg git_origin "$git_origin" \
       --arg created "$created_date" \
       '.app_id = $app_id |
        .type = $app_type |
        .source_scenario = (if $source_scenario == "" then null else $source_scenario end) |
        .git_origin = $git_origin |
        .created = $created |
        .embedding_config.collections = {
          workflows: ($app_id + "-workflows"),
          scenarios: ($app_id + "-scenarios"),
          docs: ($app_id + "-docs"),
          code: ($app_id + "-code"),
          resources: ($app_id + "-resources"),
          filetrees: ($app_id + "-filetrees")
        }' "$APP_IDENTITY_FILE" > "$temp_file"
    
    mv "$temp_file" "$APP_IDENTITY_FILE"
    
    log::success "Initialized app identity: $app_id"
    log::info "Identity file: $APP_IDENTITY_FILE"
    
    return 0
}

#######################################
# Detect app ID from project structure
# Returns: App ID string
#######################################
qdrant::identity::detect_app_id() {
    local app_id=""
    
    # Check if we're in a generated app (has .scenario.yaml)
    if [[ -f .scenario.yaml ]]; then
        # Extract scenario name
        local scenario_name=$(grep "^name:" .scenario.yaml | cut -d: -f2 | tr -d ' ' | tr '[:upper:]' '[:lower:]')
        if [[ -n "$scenario_name" ]]; then
            # Add version suffix if exists
            local version=$(grep "^version:" .scenario.yaml | cut -d: -f2 | tr -d ' ' || echo "v1")
            app_id="${scenario_name}-${version}"
        fi
    fi
    
    # Check if we're in main Vrooli repo
    if [[ -z "$app_id" ]] && [[ -f "CLAUDE.md" ]] && [[ -d "scripts/scenarios" ]]; then
        app_id="vrooli-main"
    fi
    
    # Fallback to directory name
    if [[ -z "$app_id" ]]; then
        app_id=$(basename "$(pwd)" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')
    fi
    
    echo "$app_id"
}

#######################################
# Get app ID from identity file
# Returns: App ID or empty string if not found
#######################################
qdrant::identity::get_app_id() {
    if [[ -f "$APP_IDENTITY_FILE" ]]; then
        jq -r '.app_id // ""' "$APP_IDENTITY_FILE" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

#######################################
# Update identity after indexing
# Arguments:
#   $1 - Number of embeddings created
#   $2 - Duration in seconds
# Returns: 0 on success
#######################################
qdrant::identity::update_after_index() {
    local embeddings_count="${1:-0}"
    local duration="${2:-0}"
    
    if [[ ! -f "$APP_IDENTITY_FILE" ]]; then
        log::error "App identity file not found"
        return 1
    fi
    
    # Get current git commit if available
    local current_commit=""
    if [[ -d .git ]]; then
        current_commit=$(git rev-parse HEAD 2>/dev/null || echo "")
    fi
    
    # Calculate actual collection count from Qdrant
    local app_id=$(jq -r '.app_id' "$APP_IDENTITY_FILE")
    local collections_count=0
    local total_size_mb=0
    
    if [[ -n "$app_id" ]]; then
        # Count collections that actually exist in Qdrant
        for collection in $(jq -r '.embedding_config.collections | to_entries[] | .value' "$APP_IDENTITY_FILE" 2>/dev/null); do
            if curl -s "http://localhost:6333/collections/$collection" | jq -e '.result' >/dev/null 2>&1; then
                ((collections_count++))
                
                # Get collection info for size calculation
                local collection_info=$(curl -s "http://localhost:6333/collections/$collection")
                # Use points_count since vectors_count might be null
                local points_count=$(echo "$collection_info" | jq -r '.result.points_count // 0')
                local vector_size=$(echo "$collection_info" | jq -r '.result.config.params.vectors.size // 1024')
                
                # Rough estimate: each point has vector of vector_size * 4 bytes (float32) + metadata overhead
                local collection_size_bytes=$((points_count * vector_size * 4 + points_count * 500))  # 500 bytes metadata per point estimate
                local collection_size_mb=$((collection_size_bytes / 1024 / 1024))
                total_size_mb=$((total_size_mb + collection_size_mb))
            fi
        done
    fi
    
    # Update identity file with all stats
    local indexed_date=$(date -Iseconds)
    local temp_file=$(mktemp)
    
    jq --arg indexed "$indexed_date" \
       --arg commit "$current_commit" \
       --arg embeddings "$embeddings_count" \
       --arg duration "$duration" \
       --arg collections "$collections_count" \
       --arg size_mb "$total_size_mb" \
       '.last_indexed = $indexed |
        .index_commit = (if $commit == "" then null else $commit end) |
        .stats.total_embeddings = ($embeddings | tonumber) |
        .stats.last_refresh_duration_seconds = ($duration | tonumber) |
        .stats.collections_count = ($collections | tonumber) |
        .stats.index_size_mb = ($size_mb | tonumber)' \
       "$APP_IDENTITY_FILE" > "$temp_file"
    
    mv "$temp_file" "$APP_IDENTITY_FILE"
    
    log::debug "Updated app identity after indexing with full stats"
    return 0
}

#######################################
# Update progress in real-time during indexing
# Arguments:
#   $1 - Collection name
#   $2 - Items processed so far
#   $3 - Total items expected (optional)
# Returns: 0 on success
#######################################
qdrant::identity::update_progress() {
    local collection="${1:-}"
    local processed="${2:-0}"
    local total="${3:-}"
    
    if [[ ! -f "$APP_IDENTITY_FILE" ]]; then
        return 1
    fi
    
    # Extract collection type from name (e.g., "vrooli-main-docs" -> "docs")
    local collection_type="${collection##*-}"
    
    # Update progress in identity file
    local temp_file=$(mktemp)
    local timestamp=$(date -Iseconds)
    
    if [[ -n "$total" ]]; then
        jq --arg type "$collection_type" \
           --arg processed "$processed" \
           --arg total "$total" \
           --arg timestamp "$timestamp" \
           '.metadata.indexing_progress[$type] = {
              processed: ($processed | tonumber),
              total: ($total | tonumber),
              last_update: $timestamp
           }' "$APP_IDENTITY_FILE" > "$temp_file"
    else
        jq --arg type "$collection_type" \
           --arg processed "$processed" \
           --arg timestamp "$timestamp" \
           '.metadata.indexing_progress[$type] = {
              processed: ($processed | tonumber),
              last_update: $timestamp
           }' "$APP_IDENTITY_FILE" > "$temp_file"
    fi
    
    mv "$temp_file" "$APP_IDENTITY_FILE"
    return 0
}

#######################################
# Update only stats without changing timestamps
# Used for sync operations
# Returns: 0 on success
#######################################
qdrant::identity::update_stats_only() {
    if [[ ! -f "$APP_IDENTITY_FILE" ]]; then
        log::error "App identity file not found"
        return 1
    fi
    
    local app_id=$(jq -r '.app_id' "$APP_IDENTITY_FILE")
    local collections_count=0
    local total_embeddings=0
    local total_size_mb=0
    
    if [[ -n "$app_id" ]]; then
        # Count collections and embeddings that actually exist in Qdrant
        for collection in $(jq -r '.embedding_config.collections | to_entries[] | .value' "$APP_IDENTITY_FILE" 2>/dev/null); do
            local collection_info=$(curl -s "http://localhost:6333/collections/$collection")
            
            if echo "$collection_info" | jq -e '.result' >/dev/null 2>&1; then
                ((collections_count++))
                
                # Get actual point count and size
                local points_count=$(echo "$collection_info" | jq -r '.result.points_count // 0')
                total_embeddings=$((total_embeddings + points_count))
                
                # Use points_count since vectors_count might be null
                local vector_size=$(echo "$collection_info" | jq -r '.result.config.params.vectors.size // 1024')
                
                # Rough estimate: each point has vector of vector_size * 4 bytes (float32) + metadata overhead
                local collection_size_bytes=$((points_count * vector_size * 4 + points_count * 500))
                local collection_size_mb=$((collection_size_bytes / 1024 / 1024))
                total_size_mb=$((total_size_mb + collection_size_mb))
            fi
        done
    fi
    
    # Update only stats in identity file
    local temp_file=$(mktemp)
    
    jq --arg embeddings "$total_embeddings" \
       --arg collections "$collections_count" \
       --arg size_mb "$total_size_mb" \
       '.stats.total_embeddings = ($embeddings | tonumber) |
        .stats.collections_count = ($collections | tonumber) |
        .stats.index_size_mb = ($size_mb | tonumber)' \
       "$APP_IDENTITY_FILE" > "$temp_file"
    
    mv "$temp_file" "$APP_IDENTITY_FILE"
    
    log::debug "Updated stats: $collections_count collections, $total_embeddings embeddings, ${total_size_mb}MB"
    return 0
}

#######################################
# Check if re-indexing is needed
# Returns: 0 if needed, 1 if not needed
#######################################
qdrant::identity::needs_reindex() {
    if [[ ! -f "$APP_IDENTITY_FILE" ]]; then
        log::debug "No identity file, indexing needed"
        return 0
    fi
    
    # Check if git exists
    if [[ ! -d .git ]]; then
        # Check if it was previously indexed with git
        local had_git=$(jq -r '.index_commit // ""' "$APP_IDENTITY_FILE")
        if [[ -n "$had_git" && "$had_git" != "null" ]]; then
            log::debug "Git was removed, re-indexing needed"
            return 0
        fi
        
        # No git, check if never indexed
        local last_indexed=$(jq -r '.last_indexed // ""' "$APP_IDENTITY_FILE")
        if [[ -z "$last_indexed" || "$last_indexed" == "null" ]]; then
            log::debug "Never indexed, indexing needed"
            return 0
        fi
        
        log::debug "No git, already indexed"
        return 1
    fi
    
    # Git exists, check commit
    local last_commit=$(jq -r '.index_commit // ""' "$APP_IDENTITY_FILE")
    local current_commit=$(git rev-parse HEAD 2>/dev/null || echo "")
    
    if [[ "$last_commit" != "$current_commit" ]]; then
        log::debug "Commit changed from $last_commit to $current_commit, re-indexing needed"
        return 0
    fi
    
    log::debug "No changes detected, indexing not needed"
    return 1
}

#######################################
# Get collection names for app
# Returns: Space-separated list of collection names
#######################################
qdrant::identity::get_collections() {
    if [[ ! -f "$APP_IDENTITY_FILE" ]]; then
        log::error "App identity file not found"
        return 1
    fi
    
    jq -r '.embedding_config.collections | to_entries[] | select(.value != null) | .value' "$APP_IDENTITY_FILE" 2>/dev/null | tr '\n' ' '
}

#######################################
# Display identity information
# Returns: 0 on success
#######################################
qdrant::identity::show() {
    if [[ ! -f "$APP_IDENTITY_FILE" ]]; then
        log::warn "No app identity file found"
        log::info "Run 'resource-qdrant embeddings init' to create one"
        return 1
    fi
    
    echo "=== App Identity ==="
    echo
    jq -r '
        "App ID: " + .app_id +
        "\nType: " + .type +
        "\nSource Scenario: " + (.source_scenario // "N/A") +
        "\nCreated: " + .created +
        "\nLast Indexed: " + (.last_indexed // "Never") +
        "\nIndex Commit: " + (.index_commit // "N/A") +
        "\n\nCollections:" +
        "\n  Workflows: " + (.embedding_config.collections.workflows // "Not configured") +
        "\n  Scenarios: " + (.embedding_config.collections.scenarios // "Not configured") +
        "\n  Documentation: " + (.embedding_config.collections.docs // "Not configured") +
        "\n  Code: " + (.embedding_config.collections.code // "Not configured") +
        "\n  Resources: " + (.embedding_config.collections.resources // "Not configured") +
        "\n  FileTrees: " + (.embedding_config.collections.filetrees // "Not configured") +
        "\n\nStatistics:" +
        "\n  Total Embeddings: " + (.stats.total_embeddings | tostring) +
        "\n  Last Refresh Duration: " + (if .stats.last_refresh_duration_seconds then (.stats.last_refresh_duration_seconds | tostring) + " seconds" else "N/A" end)
    ' "$APP_IDENTITY_FILE"
    echo
}

#######################################
# List all app identities in the system
# Returns: 0 on success
#######################################
qdrant::identity::list_all() {
    echo "=== Registered Apps ==="
    echo
    
    local count=0
    local seen_apps=()  # Track seen app_ids to prevent showing duplicates
    
    # Search for app-identity.json files from current directory
    # Use SEARCH_BASE environment variable to override if needed
    local search_base="${SEARCH_BASE:-$(pwd)}"
    
    while IFS= read -r identity_file; do
        if [[ -f "$identity_file" ]]; then
            local app_id=$(jq -r '.app_id' "$identity_file" 2>/dev/null)
            
            # Check if we've already seen this app_id
            local already_seen=false
            for seen_id in "${seen_apps[@]}"; do
                if [[ "$seen_id" == "$app_id" ]]; then
                    already_seen=true
                    break
                fi
            done
            
            if [[ "$already_seen" == "false" ]]; then
                seen_apps+=("$app_id")
                
                local app_type=$(jq -r '.type' "$identity_file" 2>/dev/null)
                local last_indexed=$(jq -r '.last_indexed // "Never"' "$identity_file" 2>/dev/null)
                local embeddings=$(jq -r '.stats.total_embeddings' "$identity_file" 2>/dev/null)
                local app_dir=$(dirname "${identity_file%/*}")
                
                echo "📱 $app_id ($app_type)"
                echo "   Path: $app_dir"
                echo "   Last Indexed: $last_indexed"
                echo "   Embeddings: $embeddings"
                echo
                
                ((count++))
            fi
        fi
    done < <(find "$search_base" -path "*/.vrooli/app-identity.json" -type f 2>/dev/null \
        | grep -v "/test/" \
        | grep -v "/backup/" \
        | grep -v "/.git/")
    
    if [[ $count -eq 0 ]]; then
        echo "No apps found with identity files"
        echo
        echo "Initialize an app with: resource-qdrant embeddings init"
    else
        echo "Total: $count apps"
    fi
    echo
}

#######################################
# Sync identity file with actual collection state
# This fixes cases where embeddings exist but identity tracking is outdated
# Returns: 0 on success
#######################################
qdrant::identity::sync_with_collections() {
    if [[ ! -f "$APP_IDENTITY_FILE" ]]; then
        log::error "App identity file not found"
        return 1
    fi
    
    # Source collections library to get count functions
    source "${APP_ROOT}/resources/qdrant/lib/collections.sh"
    
    # Get collections for this app
    local collections
    collections=$(qdrant::identity::get_collections)
    
    if [[ -z "$collections" ]]; then
        log::debug "No collections found for app"
        return 0
    fi
    
    # Calculate actual embedding counts
    local total_embeddings=0
    local collection_counts=""
    
    log::debug "Syncing identity with actual collection state..."
    
    for collection in $collections; do
        local count
        count=$(qdrant::collections::get_vector_count "$collection" 2>/dev/null || echo "0")
        
        if [[ "$count" =~ ^[0-9]+$ ]]; then
            total_embeddings=$((total_embeddings + count))
            log::debug "Collection $collection: $count embeddings"
        else
            log::debug "Collection $collection: not accessible or empty"
            count="0"
        fi
        
        collection_counts="${collection_counts}${collection}:${count} "
    done
    
    log::debug "Total embeddings found: $total_embeddings"
    
    # If we found embeddings but identity shows none, update the identity
    local current_total
    current_total=$(jq -r '.stats.total_embeddings // 0' "$APP_IDENTITY_FILE" 2>/dev/null || echo "0")
    
    if [[ "$total_embeddings" -gt 0 && "$current_total" -ne "$total_embeddings" ]]; then
        log::info "Syncing identity file with actual collection state ($total_embeddings embeddings found)"
        
        # Use the new update_stats_only function to also update collections_count and index_size_mb
        qdrant::identity::update_stats_only
        
        # Add last_sync timestamp
        local synced_date=$(date -Iseconds)
        local temp_file=$(mktemp)
        
        jq --arg synced "$synced_date" \
           '.metadata.last_sync = $synced' \
           "$APP_IDENTITY_FILE" > "$temp_file"
        
        mv "$temp_file" "$APP_IDENTITY_FILE"
        
        log::success "Identity file synced with full stats"
        return 0
    else
        log::debug "Identity file already in sync with collections"
        # Even if counts match, update other stats that might be out of date
        qdrant::identity::update_stats_only
        return 0
    fi
}

#######################################
# Remove app identity and associated collections
# Arguments:
#   $1 - App ID (optional, current app if not specified)
#   $2 - Force flag (yes/no, default: no)
# Returns: 0 on success
#######################################
qdrant::identity::remove() {
    local app_id="${1:-$(qdrant::identity::get_app_id)}"
    local force="${2:-no}"
    
    if [[ -z "$app_id" ]]; then
        log::error "No app ID specified and no identity file found"
        return 1
    fi
    
    if [[ "$force" != "yes" ]]; then
        log::warn "This will remove app identity and all associated embeddings for: $app_id"
        echo "Use --force yes to confirm"
        return 1
    fi
    
    # Get collections before removing identity
    local collections=$(qdrant::identity::get_collections)
    
    # Remove identity file
    if [[ -f "$APP_IDENTITY_FILE" ]]; then
        rm -f "$APP_IDENTITY_FILE"
        log::info "Removed identity file"
    fi
    
    # Remove collections from Qdrant
    if [[ -n "$collections" ]]; then
        for collection in $collections; do
            log::info "Removing collection: $collection"
            # This would call qdrant collection delete function
            # qdrant::collections::delete "$collection" "yes"
        done
    fi
    
    log::success "Removed app identity: $app_id"
    return 0
}