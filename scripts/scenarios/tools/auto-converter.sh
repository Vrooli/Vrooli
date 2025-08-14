#!/usr/bin/env bash
set -euo pipefail

# Simple scenario converter with hash-based change detection

# Source var.sh first with relative path
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Script-specific paths
CATALOG_FILE="${var_SCRIPTS_SCENARIOS_DIR}/catalog.json"
HASH_FILE="${var_DATA_DIR}/scenario-hashes.json"
SCENARIO_TO_APP="${var_SCRIPTS_SCENARIOS_DIR}/tools/scenario-to-app.sh"

# Default app output directory (can be overridden via environment variable)
# This matches the hardcoded path in scenario-to-app.sh
GENERATED_APPS_DIR="${GENERATED_APPS_DIR:-$HOME/generated-apps}"

# Parse arguments
FORCE=false
VERBOSE=false
DRY_RUN=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --force) FORCE=true ;;
        --verbose) VERBOSE=true ;;
        --dry-run) DRY_RUN=true ;;
        --help)
            echo "Usage: $0 [--force] [--verbose] [--dry-run]"
            echo "  --force    Convert all scenarios regardless of changes"
            echo "  --verbose  Show detailed output"
            echo "  --dry-run  Show what would be done without doing it"
            exit 0
            ;;
        # Ignore shell redirections and operators that might be passed accidentally
        '2>&1'|'1>&2'|'&>'|'>'|'<'|'|'|'&&'|'||'|';') ;; # Silently ignore
        *) log::error "Unknown option: $1"; exit 1 ;;
    esac
    shift
done

# Ensure data directory exists
mkdir -p "${var_DATA_DIR}"

# Load stored hashes
declare -A stored_hashes
if [[ -f "$HASH_FILE" ]]; then
    while IFS= read -r line; do
        if [[ "$line" =~ \"([^\"]+)\":[[:space:]]*\"([^\"]+)\" ]]; then
            stored_hashes["${BASH_REMATCH[1]}"]="${BASH_REMATCH[2]}"
        fi
    done < "$HASH_FILE"
    [[ "$VERBOSE" == "true" ]] && log::info "Loaded ${#stored_hashes[@]} stored hashes"
fi

# Calculate scenario hash
calculate_hash() {
    local path="$1"
    local hash_content=""
    
    # Collect content from all relevant files
    if [[ -f "$path/.vrooli/service.json" ]]; then
        hash_content+=$(cat "$path/.vrooli/service.json" 2>/dev/null)
    elif [[ -f "$path/service.json" ]]; then
        hash_content+=$(cat "$path/service.json" 2>/dev/null)
    fi
    
    if [[ -d "$path/initialization" ]]; then
        hash_content+=$(find "$path/initialization" -type f -exec cat {} \; 2>/dev/null)
    fi
    
    if [[ -d "$path/deployment" ]]; then
        hash_content+=$(find "$path/deployment" -type f -name "*.sh" -exec cat {} \; 2>/dev/null)
    fi
    
    if [[ -z "$hash_content" ]]; then
        echo ""
        return 1
    fi
    
    # Calculate hash
    if command -v sha256sum >/dev/null 2>&1; then
        echo "$hash_content" | sha256sum | awk '{print $1}'
    else
        echo "$hash_content" | shasum -a 256 | awk '{print $1}'
    fi
}

# Main processing
log::header "Scenario Auto-Converter"
[[ "$DRY_RUN" == "true" ]] && log::info "DRY RUN MODE"

# Get enabled scenarios from catalog
[[ "$VERBOSE" == "true" ]] && log::info "Reading catalog: $CATALOG_FILE"
enabled_scenarios=$(jq -r '.scenarios[] | select(.enabled == true) | "\(.name):\(.location)"' "$CATALOG_FILE" 2>/dev/null)
if [[ -z "$enabled_scenarios" ]]; then
    log::info "No enabled scenarios found"
    exit 0
fi

# Process scenarios
declare -A new_hashes
converted=0
skipped=0
failed=0
start_time=$(date +%s)

while IFS= read -r scenario_info; do
    [[ -z "$scenario_info" ]] && continue
    
    name="${scenario_info%%:*}"
    location="${scenario_info#*:}"
    [[ "$VERBOSE" == "true" ]] && log::info "Processing: $name"
    path="${var_SCRIPTS_SCENARIOS_DIR}/${location}"
    app_path="${GENERATED_APPS_DIR}/$name"
    
    # Calculate current hash
    current_hash=$(calculate_hash "$path" 2>/dev/null) || {
        log::error "Failed to hash: $name"
        failed=$((failed + 1))
        continue
    }
    
    # Check for app customizations (git-based detection)
    app_customized=false
    customization_reason=""
    
    if [[ -d "$app_path" ]] && [[ -d "$app_path/.git" ]]; then
        # Change to app directory to check git status
        current_dir=$(pwd)
        if cd "$app_path" 2>/dev/null; then
            # Check for uncommitted changes (modified, staged, or untracked files)
            if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null || [[ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]]; then
                app_customized=true
                customization_reason="uncommitted changes"
            # Check for commits after initial generation
            elif [[ $(git rev-list --count HEAD 2>/dev/null || echo "0") -gt 1 ]]; then
                app_customized=true
                customization_reason="additional commits"
            fi
            cd "$current_dir" || true
        fi
    fi
    
    # Check if conversion needed
    needs_conversion=false
    reason=""
    
    if [[ "$FORCE" == "true" ]]; then
        needs_conversion=true
        reason="forced"
    elif [[ "$app_customized" == "true" ]]; then
        needs_conversion=false
        reason="customized"
        log::warning "⚠️  $name has been customized ($customization_reason)"
        
        # Show helpful information about customizations
        if [[ -d "$app_path/.git" ]]; then
            current_dir=$(pwd)
            if cd "$app_path" 2>/dev/null; then
                modified_files=$(git status --porcelain 2>/dev/null | wc -l || echo "0")
                uncommitted_files=$(git status --porcelain 2>/dev/null | head -3 | awk '{print "     " $2}' || true)
                
                if [[ $modified_files -gt 0 ]]; then
                    log::info "   📁 Modified files: $modified_files"
                    if [[ -n "$uncommitted_files" ]] && [[ "$VERBOSE" == "true" ]]; then
                        log::info "   🔍 Recent changes:"
                        echo "$uncommitted_files"
                    fi
                elif [[ "$customization_reason" == "additional commits" ]]; then
                    commit_count=$(git rev-list --count HEAD 2>/dev/null || echo "0")
                    log::info "   📝 Total commits: $commit_count"
                fi
                
                cd "$current_dir" || true
            fi
        fi
        
        log::info "   ⏭️  Skipping regeneration (use --force to override)"
    elif [[ -z "${stored_hashes[$name]:-}" ]]; then
        needs_conversion=true
        reason="new"
    elif [[ "$current_hash" != "${stored_hashes[$name]}" ]]; then
        needs_conversion=true
        reason="changed"
    elif [[ ! -d "$app_path" ]]; then
        needs_conversion=true
        reason="missing"
    fi
    
    if [[ "$needs_conversion" == "true" ]]; then
        log::info "Converting $name ($reason)..."
        
        if [[ "$DRY_RUN" == "true" ]]; then
            log::info "[DRY RUN] Would convert: $name"
        else
            # Create backup if force regenerating a customized app
            if [[ "$reason" == "forced" ]] && [[ "$app_customized" == "true" ]] && [[ -d "$app_path" ]]; then
                log::warning "🔄 Force regenerating customized app: $name"
                
                # Create backup directory structure
                backup_dir="${GENERATED_APPS_DIR}/.backups"
                backup_name="${name}-$(date +%Y%m%d-%H%M%S)"
                backup_path="$backup_dir/$backup_name"
                
                mkdir -p "$backup_dir" || {
                    log::error "Failed to create backup directory: $backup_dir"
                    failed=$((failed + 1))
                    continue
                }
                
                # Copy app to backup location
                if cp -r "$app_path" "$backup_path" 2>/dev/null; then
                    log::info "💾 Backup created: $backup_path"
                    [[ "$VERBOSE" == "true" ]] && log::info "   📂 Backup contains all customizations and version history"
                else
                    log::error "Failed to create backup for $name"
                    failed=$((failed + 1))
                    continue
                fi
            fi
            
            # Run scenario-to-app and capture output for debugging
            if output=$(bash "$SCENARIO_TO_APP" "$name" ${VERBOSE:+--verbose} --force 2>&1); then
                log::success "✓ $name"
                converted=$((converted + 1))
            else
                log::error "✗ $name"
                [[ "$VERBOSE" == "true" ]] && echo "$output" | tail -10
                failed=$((failed + 1))
                continue
            fi
        fi
        new_hashes["$name"]="$current_hash"
    else
        [[ "$VERBOSE" == "true" ]] && log::info "Skipping $name (unchanged)"
        new_hashes["$name"]="${stored_hashes[$name]:-$current_hash}"
        skipped=$((skipped + 1))
    fi
done <<< "$enabled_scenarios"

[[ "$VERBOSE" == "true" ]] && log::info "Processing completed. Converted: $converted, Skipped: $skipped, Failed: $failed"

# Save updated hashes
# Check if array is defined and has elements
if [[ -v new_hashes[@] ]] && [[ ${#new_hashes[@]} -gt 0 ]]; then
    [[ "$VERBOSE" == "true" ]] && log::info "Saving ${#new_hashes[@]} hashes to $HASH_FILE"
    if [[ "$DRY_RUN" != "true" ]]; then
        # Create temp file first to avoid partial writes
        temp_file="${HASH_FILE}.tmp"
        {
            echo "{"
            first=true
            for name in "${!new_hashes[@]}"; do
                [[ "$first" == "true" ]] && first=false || echo ","
                printf '  "%s": "%s"' "$name" "${new_hashes[$name]}"
            done
            echo ""
            echo "}"
        } > "$temp_file" || {
            log::error "Failed to write hash file"
            rm -f "$temp_file"
            exit 1
        }
        
        # Atomic move
        mv "$temp_file" "$HASH_FILE" || {
            log::error "Failed to save hash file"
            rm -f "$temp_file"
            exit 1
        }
    fi
fi

# Summary
end_time=$(date +%s)
duration=$((end_time - start_time))

echo ""
log::header "Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Converted: $converted"
echo "Skipped:   $skipped"
echo "Failed:    $failed"
echo "Time:      ${duration}s"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [[ $converted -gt 0 ]]; then
    log::success "Generated apps in: ${GENERATED_APPS_DIR}/"
elif [[ $failed -eq 0 ]]; then
    log::success "All scenarios up to date"
fi

# Report status but don't block setup completion
if [[ $failed -gt 0 ]]; then
    log::warning "⚠️  System running in degraded mode: $failed scenario(s) failed to convert"
    log::info "Failed scenarios can be converted manually later or may be experimental"
fi

# Always exit successfully to allow setup to complete
exit 0