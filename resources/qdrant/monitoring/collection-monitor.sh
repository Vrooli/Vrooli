#!/usr/bin/env bash
# Collection Growth Monitoring for Qdrant
# Alerts when expected collections are empty or have unexpected data

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
source "${REPO_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
qdrant::export_config 2>/dev/null || {
    # Fallback configuration
    export QDRANT_BASE_URL="${QDRANT_BASE_URL:-http://localhost:6333}"
}

#######################################
# Expected collection configurations
#######################################
declare -A EXPECTED_COLLECTIONS=(
    # Main app collections - should have data
    ["vrooli-main-workflows"]="should_have_data"
    ["vrooli-main-scenarios"]="should_have_data"  
    ["vrooli-main-docs"]="should_have_data"
    
    # Main app collections - may be empty during development
    ["vrooli-main-code"]="warn_if_empty"
    ["vrooli-main-knowledge"]="warn_if_empty"
    ["vrooli-main-resources"]="warn_if_empty"
    ["vrooli-main-filetrees"]="warn_if_empty"
    
    # Vault collections - may be empty if not indexed yet
    ["vault-workflows"]="ok_if_empty"
    ["vault-knowledge"]="ok_if_empty"
    ["vault-filetrees"]="ok_if_empty"
    ["vault-scenarios"]="ok_if_empty"
    ["vault-code"]="ok_if_empty"
    ["vault-resources"]="ok_if_empty"
    
    # Scenario-specific collections
    ["issue_embeddings"]="ok_if_empty"
)

#######################################
# Check single collection status
#######################################
check_collection() {
    local collection="$1"
    local expected_status="$2"
    
    # Get point count
    local count
    count=$(curl -s "${QDRANT_BASE_URL}/collections/${collection}/points/count" \
        -H "Content-Type: application/json" \
        -d '{"exact": true}' | jq -r '.result.count // 0' 2>/dev/null || echo "0")
    
    case "$expected_status" in
        "should_have_data")
            if [[ "$count" -eq 0 ]]; then
                log::error "❌ CRITICAL: $collection is empty but should have data (count: $count)"
                return 1
            else
                log::info "✅ $collection: $count points (healthy)"
            fi
            ;;
        "warn_if_empty") 
            if [[ "$count" -eq 0 ]]; then
                log::warn "⚠️  WARNING: $collection is empty, may need indexing (count: $count)"
            else
                log::info "✅ $collection: $count points (healthy)"
            fi
            ;;
        "ok_if_empty")
            if [[ "$count" -eq 0 ]]; then
                log::debug "📝 $collection: empty (expected, count: $count)"
            else
                log::info "✅ $collection: $count points (populated)"
            fi
            ;;
    esac
    
    return 0
}

#######################################
# Run comprehensive collection monitoring
#######################################
monitor_collections() {
    log::info "🔍 Starting Collection Health Check..."
    echo
    
    local critical_issues=0
    local warnings=0
    
    for collection in "${!EXPECTED_COLLECTIONS[@]}"; do
        local expected="${EXPECTED_COLLECTIONS[$collection]}"
        
        # Check if collection exists
        if ! curl -s "${QDRANT_BASE_URL}/collections/${collection}" | jq -e '.result' >/dev/null 2>&1; then
            log::error "❌ CRITICAL: Collection $collection does not exist"
            ((critical_issues++))
            continue
        fi
        
        # Check collection status
        if ! check_collection "$collection" "$expected"; then
            ((critical_issues++))
        elif [[ "$expected" == "warn_if_empty" ]]; then
            count=$(curl -s "${QDRANT_BASE_URL}/collections/${collection}/points/count" \
                -H "Content-Type: application/json" \
                -d '{"exact": true}' | jq -r '.result.count // 0' 2>/dev/null || echo "0")
            if [[ "$count" -eq 0 ]]; then
                ((warnings++))
            fi
        fi
    done
    
    echo
    log::info "📊 Collection Health Summary:"
    log::info "   Critical Issues: $critical_issues"
    log::info "   Warnings: $warnings"
    
    if [[ $critical_issues -gt 0 ]]; then
        echo
        log::error "🚨 CRITICAL: $critical_issues collections have critical issues"
        log::info "💡 Recommended actions:"
        log::info "   • Check if main app indexing is running"
        log::info "   • Verify Qdrant connectivity"
        log::info "   • Review recent embedding operations"
        return 1
    elif [[ $warnings -gt 0 ]]; then
        echo
        log::warn "⚠️  $warnings collections may need attention"
        log::info "💡 Consider running indexing to populate empty collections"
        return 0
    else
        echo
        log::success "✅ All collections healthy!"
        return 0
    fi
}

#######################################
# Quick status check (for automated monitoring)
#######################################
quick_status() {
    local total_points=0
    local collection_count=0
    
    for collection in "${!EXPECTED_COLLECTIONS[@]}"; do
        if curl -s "${QDRANT_BASE_URL}/collections/${collection}" | jq -e '.result' >/dev/null 2>&1; then
            ((collection_count++))
            local points
            points=$(curl -s "${QDRANT_BASE_URL}/collections/${collection}/points/count" \
                -H "Content-Type: application/json" \
                -d '{"exact": true}' | jq -r '.result.count // 0' 2>/dev/null || echo "0")
            total_points=$((total_points + points))
        fi
    done
    
    echo "Collections: $collection_count, Total Points: $total_points"
}

#######################################
# Main function
#######################################
main() {
    case "${1:-monitor}" in
        "monitor"|"check")
            monitor_collections
            ;;
        "status"|"quick")
            quick_status
            ;;
        *)
            echo "Usage: $0 [monitor|status]"
            echo "  monitor: Full collection health check (default)"
            echo "  status:  Quick status summary"
            exit 1
            ;;
    esac
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi