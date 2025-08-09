#!/usr/bin/env bash
set -euo pipefail

# Simple Port Migration Tool
# Quickly migrates the most common hardcoded port patterns

SCENARIO_TOOLS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SCENARIO_TOOLS_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Port mappings (most common ones)
declare -A PORT_REPLACEMENTS=(
    ["localhost:11434"]="\${service.ollama.url}"
    ["localhost:5678"]="\${service.n8n.url}"
    ["localhost:5681"]="\${service.windmill.url}"
    ["localhost:9200"]="\${service.searxng.url}"
    ["localhost:6333"]="\${service.qdrant.url}"
    ["localhost:9000"]="\${service.minio.url}"
    ["localhost:5433"]="\${service.postgres.url}"
    ["localhost:4110"]="\${service.browserless.url}"
    ["localhost:8090"]="\${service.whisper.url}"
    ["localhost:11450"]="\${service.unstructured-io.url}"
    ["http://localhost:11434"]="\${service.ollama.url}"
    ["http://localhost:5678"]="\${service.n8n.url}"
    ["http://localhost:5681"]="\${service.windmill.url}"
    ["http://localhost:9200"]="\${service.searxng.url}"
    ["http://localhost:6333"]="\${service.qdrant.url}"
    ["http://localhost:9000"]="\${service.minio.url}"
    ["http://localhost:4110"]="\${service.browserless.url}"
    ["http://localhost:8090"]="\${service.whisper.url}"
    ["http://localhost:11450"]="\${service.unstructured-io.url}"
)

# Show help
show_help() {
    echo "Simple Port Migration Tool"
    echo ""
    echo "Usage: migrate-ports-simple.sh <scenario-path> [--dry-run]"
    echo ""
    echo "Migrates common hardcoded port patterns to service references."
    echo ""
    echo "Options:"
    echo "  --dry-run   Show what would be changed without making changes"
    echo "  --help      Show this help"
    echo ""
}

# Migrate a single file
migrate_file() {
    local file="$1"
    local dry_run="${2:-false}"
    local changed=false
    
    # Read file content
    local content
    content=$(cat "$file")
    local original_content="$content"
    
    # Apply all replacements
    for old_pattern in "${!PORT_REPLACEMENTS[@]}"; do
        local new_pattern="${PORT_REPLACEMENTS[$old_pattern]}"
        
        if echo "$content" | grep -q -F "$old_pattern"; then
            content="${content//$old_pattern/$new_pattern}"
            changed=true
            echo "    $old_pattern → $new_pattern"
        fi
    done
    
    # Write changes
    if [[ "$changed" == "true" ]]; then
        if [[ "$dry_run" == "false" ]]; then
            # Create backup
            cp "$file" "${file}.backup"
            echo "$content" > "$file"
            log::success "  ✓ Updated: $(basename "$file")"
        else
            log::info "  [DRY RUN] Would update: $(basename "$file")"
        fi
    else
        log::info "  No changes needed: $(basename "$file")"
    fi
    
    echo "$changed"
}

# Main migration function
migrate_scenario() {
    local scenario_path="$1"
    local dry_run="${2:-false}"
    
    log::info "Migrating scenario: $(basename "$scenario_path")"
    [[ "$dry_run" == "true" ]] && log::info "[DRY RUN MODE]"
    
    local total_files=0
    local changed_files=0
    
    # Key file patterns to migrate
    local file_patterns=(
        "$scenario_path/.vrooli/service.json"
        "$scenario_path/initialization/automation/n8n/*.json"
        "$scenario_path/initialization/automation/windmill/*.json"
        "$scenario_path/initialization/configuration/*.json"
        "$scenario_path/deployment/*.sh"
    )
    
    for pattern in "${file_patterns[@]}"; do
        for file in $pattern; do
            if [[ -f "$file" ]]; then
                ((total_files++))
                log::info "Processing: $(realpath --relative-to="$scenario_path" "$file")"
                
                local file_changed
                file_changed=$(migrate_file "$file" "$dry_run")
                
                if [[ "$file_changed" == "true" ]]; then
                    ((changed_files++))
                fi
            fi
        done
    done
    
    # Summary
    echo ""
    log::info "Migration Summary:"
    log::info "  Files processed: $total_files"
    log::info "  Files changed: $changed_files"
    
    if [[ "$changed_files" -gt 0 ]]; then
        if [[ "$dry_run" == "false" ]]; then
            log::success "✅ Migration completed successfully!"
            log::info "💡 Backup files created with .backup extension"
        else
            log::info "🔍 Run without --dry-run to apply these changes"
        fi
    else
        log::success "✅ No migration needed - scenario already uses service references!"
    fi
}

# Main script
main() {
    local scenario_path=""
    local dry_run=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run)
                dry_run=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            -*)
                log::error "Unknown option: $1"
                exit 1
                ;;
            *)
                scenario_path="$1"
                shift
                ;;
        esac
    done
    
    # Validate arguments
    if [[ -z "$scenario_path" ]]; then
        log::error "Scenario path is required"
        show_help
        exit 1
    fi
    
    if [[ ! -d "$scenario_path" ]]; then
        log::error "Directory not found: $scenario_path"
        exit 1
    fi
    
    # Run migration
    migrate_scenario "$scenario_path" "$dry_run"
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi