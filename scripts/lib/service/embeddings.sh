#!/usr/bin/env bash
#######################################
# Embeddings Integration Library
# Provides integration with the qdrant semantic knowledge system
#######################################

set -euo pipefail

#######################################
# Auto-refresh embeddings when git changes are detected
# This hooks into the existing git change detection system
# Arguments:
#   $1 - Scripts directory (optional, defaults to var_SCRIPTS_DIR)
# Returns:
#   0 on success (always succeeds, logs errors but continues)
#######################################
embeddings::refresh_on_changes() {
    local scripts_dir="${1:-${var_SCRIPTS_DIR}}"
    
    # Check if embeddings system is available
    local embeddings_manage="${scripts_dir}/resources/storage/qdrant/embeddings/manage.sh"
    
    if [[ ! -f "$embeddings_manage" ]]; then
        log::debug "Embeddings system not available, skipping auto-refresh"
        return 0
    fi
    
    # Check if app has embeddings initialized
    if [[ ! -f ".vrooli/app-identity.json" ]]; then
        log::debug "No app identity found, skipping embeddings auto-refresh"
        return 0
    fi
    
    # Source embeddings system (safely)
    if ! source "$embeddings_manage" 2>/dev/null; then
        log::debug "Failed to source embeddings system, skipping auto-refresh"
        return 0
    fi
    
    # Check if embeddings need refresh (git commit changed)
    local app_id
    app_id=$(qdrant::identity::get_app_id 2>/dev/null) || {
        log::debug "Could not get app ID, skipping embeddings auto-refresh"
        return 0
    }
    
    if [[ -z "$app_id" ]]; then
        log::debug "App ID empty, skipping embeddings auto-refresh"
        return 0
    fi
    
    # Check if refresh is needed
    if qdrant::identity::needs_reindex 2>/dev/null; then
        log::info "🔄 Git changes detected, auto-refreshing semantic knowledge..."
        
        # Run refresh in background to not slow down development
        {
            if qdrant::embeddings::refresh "$app_id" "yes" >/dev/null 2>&1; then
                log::success "✅ Semantic knowledge updated for: $app_id"
            else
                log::warn "⚠️  Semantic knowledge refresh failed for: $app_id"
            fi
        } &
        
        # Don't wait for completion - let it run in background
        log::debug "Embeddings refresh started in background"
    else
        log::debug "Embeddings are current, no refresh needed"
    fi
    
    return 0
}