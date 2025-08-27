#!/usr/bin/env bash
set -euo pipefail

# Simple scenario converter with hash-based change detection

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCENARIO_TOOLS_DIR="${APP_ROOT}/scripts/scenarios/tools"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
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

# Arrays to collect scenarios for batch processing
declare -a scenarios_to_convert=()
declare -A scenario_paths=()
declare -A scenario_reasons=()
declare -A scenario_hashes=()

# First pass: Determine which scenarios need conversion
while IFS= read -r scenario_info; do
    [[ -z "$scenario_info" ]] && continue
    
    name="${scenario_info%%:*}"
    location="${scenario_info#*:}"
    [[ "$VERBOSE" == "true" ]] && log::info "Analyzing: $name"
    path="${var_SCENARIOS_DIR}/${location}"
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
        # Create backup if force regenerating a customized app (only if not dry run)
        if [[ "$DRY_RUN" != "true" ]] && [[ "$reason" == "forced" ]] && [[ "$app_customized" == "true" ]] && [[ -d "$app_path" ]]; then
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
        
        # Add to batch conversion list (both dry run and real)
        scenarios_to_convert+=("$name")
        scenario_paths["$name"]="$path"
        scenario_reasons["$name"]="$reason"
        scenario_hashes["$name"]="$current_hash"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            log::info "[DRY RUN] Queued for batch conversion: $name ($reason)"
        else
            log::info "Queued for conversion: $name ($reason)"
        fi
    else
        [[ "$VERBOSE" == "true" ]] && log::info "Skipping $name (unchanged)"
        new_hashes["$name"]="${stored_hashes[$name]:-$current_hash}"
        skipped=$((skipped + 1))
    fi
done <<< "$enabled_scenarios"

# Second pass: Batch convert scenarios if any need conversion
if [[ ${#scenarios_to_convert[@]} -gt 0 ]]; then
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would batch convert ${#scenarios_to_convert[@]} scenarios: ${scenarios_to_convert[*]}"
        # In dry run mode, simulate success for hash tracking
        for scenario in "${scenarios_to_convert[@]}"; do
            new_hashes["$scenario"]="${scenario_hashes[$scenario]}"
            converted=$((converted + 1))
        done
    else
        log::info "Batch converting ${#scenarios_to_convert[@]} scenario(s) in chunks of 20..."
        
        # Process scenarios in chunks to avoid resource exhaustion
        batch_start=$(date +%s)
        batch_timeout=600  # 10 minutes timeout per chunk (doubled for larger chunks)
        chunk_size=20
        total_scenarios=${#scenarios_to_convert[@]}
        chunks_processed=0
        chunks_succeeded=0
        
        # Process scenarios in chunks of 20
        for ((i = 0; i < total_scenarios; i += chunk_size)); do
            chunk_start_idx=$i
            chunk_end_idx=$((i + chunk_size))
            if [[ chunk_end_idx -gt total_scenarios ]]; then
                chunk_end_idx=$total_scenarios
            fi
            
            # Build chunk scenarios array
            chunk_scenarios=("${scenarios_to_convert[@]:$chunk_start_idx:$((chunk_end_idx - chunk_start_idx))}")
            chunks_processed=$((chunks_processed + 1))
            chunk_num=$chunks_processed
            total_chunks=$(((total_scenarios + chunk_size - 1) / chunk_size))
            
            log::info "Processing chunk $chunk_num/$total_chunks: ${chunk_scenarios[*]}"
            
            # Build the command arguments for this chunk
            batch_args=()
            for scenario in "${chunk_scenarios[@]}"; do
                batch_args+=("$scenario")
            done
            
            # Add common flags
            [[ "$VERBOSE" == "true" ]] && batch_args+=("--verbose")
            batch_args+=("--force")
            [[ "$DRY_RUN" == "true" ]] && batch_args+=("--dry-run")
            
            # Process this chunk
            chunk_succeeded=false
        
            if command -v timeout >/dev/null 2>&1; then
                # Use timeout command to prevent hangs
                if [[ "$VERBOSE" == "true" ]]; then
                    # Run directly without capturing output
                    if timeout "$batch_timeout" bash "$SCENARIO_TO_APP" "${batch_args[@]}"; then
                        chunk_succeeded=true
                    else
                        exit_code=$?
                        if [[ $exit_code -eq 124 ]]; then
                            log::warning "Chunk $chunk_num timed out after ${batch_timeout}s"
                        else
                            log::warning "Chunk $chunk_num failed with exit code $exit_code"
                        fi
                    fi
                else
                    # Non-verbose: capture output for error reporting using temp file
                    chunk_output_file=$(mktemp)
                    if timeout "$batch_timeout" bash "$SCENARIO_TO_APP" "${batch_args[@]}" &> "$chunk_output_file"; then
                        chunk_succeeded=true
                    else
                        exit_code=$?
                        if [[ $exit_code -eq 124 ]]; then
                            log::warning "Chunk $chunk_num timed out after ${batch_timeout}s"
                        else
                            log::warning "Chunk $chunk_num failed with exit code $exit_code"
                        fi
                        # Store output content for later use
                        chunk_output=$(cat "$chunk_output_file" 2>/dev/null || echo "No output captured")
                    fi
                    rm -f "$chunk_output_file"
                fi
            else
                # Fallback without timeout (less safe)
                if [[ "$VERBOSE" == "true" ]]; then
                    if bash "$SCENARIO_TO_APP" "${batch_args[@]}"; then
                        chunk_succeeded=true
                    fi
                else
                    # Non-verbose fallback: capture output using temp file
                    chunk_output_file=$(mktemp)
                    if bash "$SCENARIO_TO_APP" "${batch_args[@]}" &> "$chunk_output_file"; then
                        chunk_succeeded=true
                    else
                        # Store output content for later use
                        chunk_output=$(cat "$chunk_output_file" 2>/dev/null || echo "No output captured")
                    fi
                    rm -f "$chunk_output_file"
                fi
            fi
            
            if [[ "$chunk_succeeded" == "true" ]]; then
                chunks_succeeded=$((chunks_succeeded + 1))
                # Mark chunk scenarios as successfully converted
                for scenario in "${chunk_scenarios[@]}"; do
                    log::success "✓ $scenario"
                    new_hashes["$scenario"]="${scenario_hashes[$scenario]}"
                    converted=$((converted + 1))
                done
                log::info "Chunk $chunk_num/$total_chunks completed successfully"
            else
                log::warning "Chunk $chunk_num/$total_chunks failed, processing scenarios individually..."
                [[ "$VERBOSE" == "false" ]] && [[ -n "${chunk_output:-}" ]] && echo "$chunk_output" | tail -3
                
                # Process failed chunk scenarios individually
                for scenario in "${chunk_scenarios[@]}"; do
                    log::info "Converting $scenario individually..."
                    individual_args=("$scenario")
                    [[ "$VERBOSE" == "true" ]] && individual_args+=("--verbose")
                    individual_args+=("--force")
                    [[ "$DRY_RUN" == "true" ]] && individual_args+=("--dry-run")
                    
                    if command -v timeout >/dev/null 2>&1; then
                        individual_output_file=$(mktemp)
                        if timeout 60 bash "$SCENARIO_TO_APP" "${individual_args[@]}" &> "$individual_output_file"; then
                            log::success "✓ $scenario"
                            new_hashes["$scenario"]="${scenario_hashes[$scenario]}"
                            converted=$((converted + 1))
                        else
                            individual_exit_code=$?
                            if [[ $individual_exit_code -eq 124 ]]; then
                                log::error "✗ $scenario (timed out after 60s)"
                            else
                                log::error "✗ $scenario (exit code: $individual_exit_code)"
                            fi
                            if [[ "$VERBOSE" == "true" ]]; then
                                tail -5 "$individual_output_file" 2>/dev/null || echo "No output to display"
                            fi
                            failed=$((failed + 1))
                        fi
                        rm -f "$individual_output_file"
                    else
                        individual_output_file=$(mktemp)
                        if bash "$SCENARIO_TO_APP" "${individual_args[@]}" &> "$individual_output_file"; then
                            log::success "✓ $scenario"
                            new_hashes["$scenario"]="${scenario_hashes[$scenario]}"
                            converted=$((converted + 1))
                        else
                            log::error "✗ $scenario"
                            if [[ "$VERBOSE" == "true" ]]; then
                                tail -5 "$individual_output_file" 2>/dev/null || echo "No output to display"
                            fi
                            failed=$((failed + 1))
                        fi
                        rm -f "$individual_output_file"
                    fi
                done
            fi
        done
        
        # Report chunk processing results
        batch_end=$(date +%s)
        batch_duration=$((batch_end - batch_start))
        
        if [[ $chunks_succeeded -eq $chunks_processed ]]; then
            log::success "All $chunks_processed chunk(s) completed successfully in ${batch_duration}s"
        else
            failed_chunks=$((chunks_processed - chunks_succeeded))
            log::warning "$failed_chunks of $chunks_processed chunk(s) had failures - some scenarios processed individually"
            log::info "Chunk processing completed in ${batch_duration}s"
        fi
    fi
fi

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