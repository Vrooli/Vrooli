#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Scenario Auto-Converter
# 
# Intelligently converts enabled scenarios to standalone apps using hash-based
# change detection. Only converts scenarios that are new or have changed.
# 
# Uses the same hash-based optimization pattern as pnpm_tools.sh to avoid
# unnecessary regeneration of unchanged scenarios.
#
# Usage:
#   ./auto-converter.sh [--force] [--verbose] [--dry-run]
#
# Options:
#   --force     Force conversion of all enabled scenarios (ignore hashes)
#   --verbose   Enable detailed output
#   --dry-run   Show what would be converted without actually doing it
#
################################################################################

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SCENARIOS_DIR="${PROJECT_ROOT}/scripts/scenarios/core"
CATALOG_FILE="${PROJECT_ROOT}/scripts/scenarios/catalog.json"
HASH_FILE="${PROJECT_ROOT}/data/scenario-hashes.json"
SCENARIO_TO_APP="${PROJECT_ROOT}/scripts/scenarios/tools/scenario-to-app.sh"

# Source common utilities
RESOURCES_DIR="${PROJECT_ROOT}/scripts/resources"
if [[ -f "${RESOURCES_DIR}/common.sh" ]]; then
    # shellcheck disable=SC1091
    source "${RESOURCES_DIR}/common.sh"
else
    # Fallback logging functions
    log::info() { echo "ℹ️  $*"; }
    log::success() { echo "✅ $*"; }
    log::warning() { echo "⚠️  $*"; }
    log::error() { echo "❌ $*"; }
    log::header() { echo ""; echo "🎯 $*"; echo ""; }
fi

# Source sudo utilities
# shellcheck disable=SC1091
source "${PROJECT_ROOT}/scripts/lib/utils/sudo.sh"

# Configuration
FORCE=false
VERBOSE=false
DRY_RUN=false

# Usage information
show_usage() {
    cat << 'EOF'
Usage: ./auto-converter.sh [options]

Intelligently converts enabled scenarios to standalone apps using hash-based
change detection. Only processes scenarios that are new or have changed.

Options:
  --force       Force conversion of all enabled scenarios (ignore hashes)
  --verbose     Enable detailed logging output
  --dry-run     Show what would be converted without actually doing it
  --help        Show this help message

Hash-Based Intelligence:
  - Tracks scenario changes using SHA-256 hashes
  - Only converts scenarios when content changes
  - Stores hash state in data/scenario-hashes.json
  - Optimizes performance by skipping unchanged scenarios

Examples:
  ./auto-converter.sh                    # Convert only changed/new scenarios
  ./auto-converter.sh --verbose          # With detailed output
  ./auto-converter.sh --force --verbose  # Force convert all enabled scenarios
  ./auto-converter.sh --dry-run          # Preview what would be converted

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                FORCE=true
                ;;
            --verbose)
                VERBOSE=true
                ;;
            --dry-run)
                DRY_RUN=true
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                log::error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
        shift
    done
}

# Validate prerequisites
validate_prerequisites() {
    # Check for required files
    if [[ ! -f "$CATALOG_FILE" ]]; then
        log::error "Catalog file not found: $CATALOG_FILE"
        return 1
    fi
    
    if [[ ! -f "$SCENARIO_TO_APP" ]]; then
        log::error "scenario-to-app.sh not found: $SCENARIO_TO_APP"
        return 1
    fi
    
    if [[ ! -x "$SCENARIO_TO_APP" ]]; then
        log::error "scenario-to-app.sh is not executable: $SCENARIO_TO_APP"
        return 1
    fi
    
    # Check for jq
    if ! command -v jq >/dev/null 2>&1; then
        log::error "jq is required but not installed"
        return 1
    fi
    
    # Check for hash utilities
    if ! command -v shasum >/dev/null 2>&1 && ! command -v sha256sum >/dev/null 2>&1; then
        log::error "Neither shasum nor sha256sum found; cannot compute scenario hashes"
        return 1
    fi
    
    return 0
}

# Calculate hash of scenario directory contents
calculate_scenario_hash() {
    local scenario_path="$1"
    
    if [[ ! -d "$scenario_path" ]]; then
        log::error "Scenario directory not found: $scenario_path"
        return 1
    fi
    
    # Files to include in hash calculation (order matters for reproducible hashes)
    local hash_files=()
    
    # Include service.json (primary config)
    if [[ -f "$scenario_path/.vrooli/service.json" ]]; then
        hash_files+=("$scenario_path/.vrooli/service.json")
    elif [[ -f "$scenario_path/service.json" ]]; then
        hash_files+=("$scenario_path/service.json")
    fi
    
    # Include all initialization files (they define the app's data)
    if [[ -d "$scenario_path/initialization" ]]; then
        while IFS= read -r -d '' file; do
            hash_files+=("$file")
        done < <(find "$scenario_path/initialization" -type f -print0 | sort -z)
    fi
    
    # Include deployment scripts (they affect the app behavior)
    if [[ -d "$scenario_path/deployment" ]]; then
        while IFS= read -r -d '' file; do
            hash_files+=("$file")
        done < <(find "$scenario_path/deployment" -type f -name "*.sh" -print0 | sort -z)
    fi
    
    if [[ ${#hash_files[@]} -eq 0 ]]; then
        log::error "No hashable files found in scenario: $scenario_path"
        return 1
    fi
    
    # Calculate combined hash
    local hash
    if command -v shasum >/dev/null 2>&1; then
        hash=$(printf '%s\n' "${hash_files[@]}" | xargs cat | shasum -a 256 | awk '{print $1}')
    else
        hash=$(printf '%s\n' "${hash_files[@]}" | xargs cat | sha256sum | awk '{print $1}')
    fi
    
    echo "$hash"
}

# Load stored hashes
load_stored_hashes() {
    if [[ -f "$HASH_FILE" ]]; then
        cat "$HASH_FILE"
    else
        echo "{}"
    fi
}

# Save updated hashes
save_hashes() {
    local hashes_json="$1"
    local hash_dir
    hash_dir=$(dirname "$HASH_FILE")
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would save hashes to: $HASH_FILE"
        return 0
    fi
    
    # Create data directory if needed
    mkdir -p "$hash_dir"
    echo "$hashes_json" | jq '.' > "$HASH_FILE"
    
    # Fix permissions if running under sudo
    sudo::restore_owner "$HASH_FILE"
    sudo::restore_owner "$hash_dir"
    
    [[ "$VERBOSE" == "true" ]] && log::info "Updated scenario hashes: $HASH_FILE"
}

# Get enabled scenarios from catalog
get_enabled_scenarios() {
    jq -r '.scenarios[] | select(.enabled == true) | "\(.name):\(.location)"' "$CATALOG_FILE"
}

# Check if generated app exists
app_exists() {
    local scenario_name="$1"
    local app_path="$HOME/generated-apps/$scenario_name"
    
    # Check if the app directory exists and has the required files
    [[ -d "$app_path" ]] && [[ -f "$app_path/.vrooli/service.json" ]]
}

# Convert scenario to app
convert_scenario() {
    local scenario_name="$1"
    local scenario_location="$2"
    
    log::info "Converting scenario: $scenario_name"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would run: $SCENARIO_TO_APP $scenario_name"
        return 0
    fi
    
    local conversion_opts=""
    [[ "$VERBOSE" == "true" ]] && conversion_opts="--verbose"
    
    # Run scenario-to-app.sh with force flag to overwrite existing apps
    if "$SCENARIO_TO_APP" "$scenario_name" $conversion_opts --force; then
        log::success "✅ Successfully converted: $scenario_name"
        return 0
    else
        log::error "❌ Failed to convert: $scenario_name"
        return 1
    fi
}

# Main conversion logic
main() {
    parse_args "$@"
    
    # Track start time
    local start_time=$(date +%s)
    
    # Show header
    log::header "🚀 Vrooli Scenario Auto-Converter"
    if [[ "$DRY_RUN" == "true" ]]; then
        log::warning "DRY RUN MODE - No actual changes will be made"
    fi
    if [[ "$FORCE" == "true" ]]; then
        log::info "FORCE MODE - All enabled scenarios will be converted"
    fi
    if [[ "$VERBOSE" == "true" ]]; then
        log::info "VERBOSE MODE - Detailed output enabled"
    fi
    echo ""
    
    # Validate prerequisites
    log::info "Validating prerequisites..."
    if ! validate_prerequisites; then
        exit 1
    fi
    log::success "Prerequisites validated"
    echo ""
    
    # Load stored hashes
    log::info "Loading scenario hash state..."
    local stored_hashes
    stored_hashes=$(load_stored_hashes)
    [[ "$VERBOSE" == "true" ]] && log::info "Hash file: $HASH_FILE"
    echo ""
    
    # Get enabled scenarios
    log::info "Analyzing enabled scenarios..."
    local enabled_scenarios
    mapfile -t enabled_scenarios < <(get_enabled_scenarios)
    
    if [[ ${#enabled_scenarios[@]} -eq 0 ]]; then
        log::warning "No enabled scenarios found in catalog"
        log::info "To enable scenarios, edit: $CATALOG_FILE"
        exit 0
    fi
    
    log::info "Found ${#enabled_scenarios[@]} enabled scenario(s)"
    if [[ "$VERBOSE" == "true" ]]; then
        for scenario_info in "${enabled_scenarios[@]}"; do
            local name="${scenario_info%:*}"
            log::info "  • $name"
        done
    fi
    echo ""
    
    # Process each enabled scenario
    log::info "Processing scenarios with hash-based intelligence..."
    local updated_hashes="$stored_hashes"
    local conversion_count=0
    local skip_count=0
    local error_count=0
    local failed_scenarios=()
    local converted_scenarios=()
    local skipped_scenarios=()
    
    for scenario_info in "${enabled_scenarios[@]}"; do
        local scenario_name="${scenario_info%:*}"
        local scenario_location="${scenario_info#*:}"
        local scenario_path="$SCENARIOS_DIR/../$scenario_location"
        
        log::info "🔍 Analyzing: $scenario_name"
        
        # Calculate current hash
        local current_hash
        if current_hash=$(calculate_scenario_hash "$scenario_path"); then
            [[ "$VERBOSE" == "true" ]] && log::info "  Current hash: ${current_hash:0:12}..."
            
            # Get stored hash
            local stored_hash
            stored_hash=$(echo "$stored_hashes" | jq -r ".\"$scenario_name\" // \"\"")
            
            # Determine if conversion is needed
            local needs_conversion=false
            local conversion_reason=""
            
            if [[ "$FORCE" == "true" ]]; then
                needs_conversion=true
                conversion_reason="forced conversion"
            elif [[ -z "$stored_hash" ]]; then
                needs_conversion=true
                conversion_reason="new scenario"
            elif [[ "$current_hash" != "$stored_hash" ]]; then
                needs_conversion=true
                conversion_reason="scenario changed"
                [[ "$VERBOSE" == "true" ]] && log::info "  Stored hash:  ${stored_hash:0:12}..."
            elif ! app_exists "$scenario_name"; then
                needs_conversion=true
                conversion_reason="app missing"
                [[ "$VERBOSE" == "true" ]] && log::info "  Generated app not found at: $HOME/generated-apps/$scenario_name"
            else
                needs_conversion=false
                conversion_reason="unchanged"
            fi
            
            if [[ "$needs_conversion" == "true" ]]; then
                log::info "  ⚡ Conversion needed: $conversion_reason"
                
                if convert_scenario "$scenario_name" "$scenario_location"; then
                    # Update hash on successful conversion
                    updated_hashes=$(echo "$updated_hashes" | jq --arg name "$scenario_name" --arg hash "$current_hash" '.[$name] = $hash')
                    conversion_count=$((conversion_count + 1))
                    converted_scenarios+=("$scenario_name")
                else
                    error_count=$((error_count + 1))
                    failed_scenarios+=("$scenario_name")
                fi
            else
                log::info "  ⏭️  Skipping: $conversion_reason"
                skip_count=$((skip_count + 1))
                skipped_scenarios+=("$scenario_name")
            fi
        else
            log::error "  ❌ Failed to calculate hash for: $scenario_name"
            error_count=$((error_count + 1))
            failed_scenarios+=("$scenario_name")
        fi
        
        echo ""
    done
    
    # Save updated hashes
    if [[ "$updated_hashes" != "$stored_hashes" ]]; then
        log::info "Saving updated scenario hashes..."
        save_hashes "$updated_hashes"
    fi
    
    # Calculate total time
    local end_time=$(date +%s)
    local total_time=$((end_time - start_time))
    local minutes=$((total_time / 60))
    local seconds=$((total_time % 60))
    
    # Show detailed summary
    echo ""
    log::header "📊 Conversion Summary"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log::info "Total Scenarios:     ${#enabled_scenarios[@]}"
    log::success "✅ Converted:        $conversion_count"
    log::info "⏭️  Skipped:          $skip_count"
    if [[ $error_count -gt 0 ]]; then
        log::error "❌ Failed:           $error_count"
    else
        log::info "❌ Failed:           0"
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log::info "Total Time:          ${minutes}m ${seconds}s"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Show detailed lists if verbose or if there were failures
    if [[ "$VERBOSE" == "true" ]] || [[ ${#failed_scenarios[@]} -gt 0 ]]; then
        echo ""
        
        if [[ ${#converted_scenarios[@]} -gt 0 ]]; then
            log::success "✅ Successfully Converted:"
            for scenario in "${converted_scenarios[@]}"; do
                echo "     • $scenario"
            done
            echo ""
        fi
        
        if [[ ${#skipped_scenarios[@]} -gt 0 ]] && [[ "$VERBOSE" == "true" ]]; then
            log::info "⏭️  Skipped (unchanged):"
            for scenario in "${skipped_scenarios[@]}"; do
                echo "     • $scenario"
            done
            echo ""
        fi
        
        if [[ ${#failed_scenarios[@]} -gt 0 ]]; then
            log::error "❌ Failed Conversions:"
            for scenario in "${failed_scenarios[@]}"; do
                echo "     • $scenario"
            done
            echo ""
            log::warning "💡 To debug failures, run with --verbose flag:"
            log::info "   ./auto-converter.sh --verbose"
            echo ""
        fi
    fi
    
    if [[ $conversion_count -gt 0 ]]; then
        log::success "🎉 Successfully converted $conversion_count scenario(s)!"
        log::info "Generated apps are available at: ~/generated-apps/"
        log::info "To run a generated app:"
        log::info "  cd ~/generated-apps/<scenario-name>"
        log::info "  ./scripts/manage.sh develop"
    elif [[ $skip_count -gt 0 && $error_count -eq 0 ]]; then
        log::success "🎯 All scenarios are up to date!"
        log::info "No conversions needed - all enabled scenarios already have current apps."
    elif [[ $error_count -gt 0 ]]; then
        log::warning "⚠️  Some scenarios failed to convert (see details above)"
        if [[ "$VERBOSE" != "true" ]]; then
            log::info "Run with --verbose for more details"
        fi
    fi
    
    # Return appropriate exit code
    if [[ $error_count -gt 0 ]]; then
        exit 1
    else
        exit 0
    fi
}

# Execute main function
main "$@"