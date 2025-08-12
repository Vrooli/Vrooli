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
GENERATED_APPS_DIR="${GENERATED_APPS_DIR:-${var_ROOT_DIR}/generated-apps}"

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
        ((failed++))
        continue
    }
    
    # Check if conversion needed
    needs_conversion=false
    reason=""
    
    if [[ "$FORCE" == "true" ]]; then
        needs_conversion=true
        reason="forced"
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
            if "$SCENARIO_TO_APP" "$name" ${VERBOSE:+--verbose} --force >/dev/null 2>&1; then
                log::success "✓ $name"
                ((converted++))
            else
                log::error "✗ $name"
                ((failed++))
                continue
            fi
        fi
        new_hashes["$name"]="$current_hash"
    else
        [[ "$VERBOSE" == "true" ]] && log::info "Skipping $name (unchanged)"
        new_hashes["$name"]="${stored_hashes[$name]}"
        ((skipped++))
    fi
done <<< "$enabled_scenarios"

# Save updated hashes
if [[ "$DRY_RUN" != "true" ]] && [[ ${#new_hashes[@]} -gt 0 ]]; then
    {
        echo "{"
        first=true
        for name in "${!new_hashes[@]}"; do
            [[ "$first" == "true" ]] && first=false || echo ","
            printf '  "%s": "%s"' "$name" "${new_hashes[$name]}"
        done
        echo ""
        echo "}"
    } > "$HASH_FILE"
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

[[ $failed -gt 0 ]] && exit 1 || exit 0