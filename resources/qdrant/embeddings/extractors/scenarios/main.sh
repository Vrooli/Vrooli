#!/usr/bin/env bash
# Modern Scenario Content Extractor for Qdrant Embeddings
# Orchestrates modular extractors to output structured JSON lines
#
# This is the main entry point that sources specialized extractors:
# - documentation.sh: PRD.md, README.md, docs/ folder
# - config.sh: service.json, .vrooli/ metadata, scenario-test.yaml
# - api.sh: API implementation (language-agnostic)
# - cli.sh: CLI implementation (language-agnostic)
# - tests.sh: Test configurations and implementations
# - ui.sh: UI/frontend implementation (framework-agnostic)
# - initialization.sh: Setup workflows (TODO)

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Define paths
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
EXTRACTOR_DIR="${EMBEDDINGS_DIR}/extractors"
SCENARIOS_EXTRACTOR_DIR="${EXTRACTOR_DIR}/scenarios"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified embedding service
source "${EMBEDDINGS_DIR}/lib/embedding-service.sh"

# Source modular extractors
source "${SCENARIOS_EXTRACTOR_DIR}/documentation.sh"
source "${SCENARIOS_EXTRACTOR_DIR}/config.sh"
source "${SCENARIOS_EXTRACTOR_DIR}/api.sh"
source "${SCENARIOS_EXTRACTOR_DIR}/cli.sh"
source "${SCENARIOS_EXTRACTOR_DIR}/tests.sh"
source "${SCENARIOS_EXTRACTOR_DIR}/ui.sh"
# TODO: source "${SCENARIOS_EXTRACTOR_DIR}/initialization.sh"

#######################################
# Extract all scenario information as JSON
# 
# Main orchestrator that calls all modular extractors
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines for all scenario components
#######################################
qdrant::extract::scenario_all_json() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        log::error "Scenario directory not found: $dir" >&2
        return 1
    fi
    
    local scenario_name=$(basename "$dir")
    log::debug "Extracting scenario: $scenario_name" >&2
    
    # Extract documentation (PRD.md, README.md, docs/)
    qdrant::extract::scenario_documentation "$dir" 2>/dev/null || true
    
    # Extract configuration (service.json, .vrooli/, scenario-test.yaml)
    qdrant::extract::scenario_config_all "$dir" 2>/dev/null || true
    
    # Extract API components (language-agnostic)
    qdrant::extract::scenario_api_all "$dir" 2>/dev/null || true
    
    # Extract CLI components (language-agnostic)
    qdrant::extract::scenario_cli_all "$dir" 2>/dev/null || true
    
    # Extract test components (framework-agnostic)
    qdrant::extract::scenario_tests_all "$dir" 2>/dev/null || true
    
    # Extract UI components (framework-agnostic)
    qdrant::extract::scenario_ui_all "$dir" 2>/dev/null || true
    
    # TODO: Extract initialization workflows
    # qdrant::extract::scenario_initialization_all "$dir" 2>/dev/null || true
}

#######################################
# Process batch of scenarios
# 
# Discovers and processes all scenario directories
#
# Arguments:
#   $1 - Parent directory (optional, defaults to APP_ROOT/scenarios)
#   $2 - Output file (optional, defaults to stdout)
# Returns: Number of scenarios processed
#######################################
qdrant::extract::scenarios_batch_json() {
    local parent_dir="${1:-$APP_ROOT/scenarios}"
    local output_file="${2:-/dev/stdout}"
    
    if [[ ! -d "$parent_dir" ]]; then
        log::error "Parent directory not found: $parent_dir" >&2
        return 1
    fi
    
    log::info "Processing scenarios in: $parent_dir" >&2
    
    # Clear output file if specified
    [[ "$output_file" != "/dev/stdout" ]] && > "$output_file"
    
    local scenario_dirs=()
    local scenario_count=0
    local total_sections=0
    
    # Find all scenario directories (must have PRD.md or service.json)
    while IFS= read -r scenario_dir; do
        # A valid scenario has at least PRD.md or service.json/.vrooli/service.json
        if [[ -f "$scenario_dir/PRD.md" ]] || [[ -f "$scenario_dir/service.json" ]] || [[ -f "$scenario_dir/.vrooli/service.json" ]]; then
            scenario_dirs+=("$scenario_dir")
        fi
    done < <(find "$parent_dir" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort)
    
    if [[ ${#scenario_dirs[@]} -eq 0 ]]; then
        log::warn "No scenarios found in $parent_dir" >&2
        return 0
    fi
    
    log::info "Found ${#scenario_dirs[@]} scenarios to process" >&2
    
    # Process each scenario
    for scenario_dir in "${scenario_dirs[@]}"; do
        local scenario_name=$(basename "$scenario_dir")
        
        # Count lines before processing
        local before_count=0
        [[ "$output_file" != "/dev/stdout" ]] && before_count=$(wc -l < "$output_file" 2>/dev/null || echo 0)
        
        # Extract all components
        qdrant::extract::scenario_all_json "$scenario_dir" >> "$output_file"
        
        # Count lines after processing
        local after_count=0
        [[ "$output_file" != "/dev/stdout" ]] && after_count=$(wc -l < "$output_file" 2>/dev/null || echo 0)
        
        local sections_added=$((after_count - before_count))
        
        if [[ $sections_added -gt 0 ]]; then
            ((scenario_count++))
            ((total_sections += sections_added))
            log::debug "  Extracted $sections_added sections from $scenario_name" >&2
        fi
    done
    
    log::success "Processed $scenario_count scenarios with $total_sections total sections" >&2
    echo "$scenario_count"
}

#######################################
# Analyze scenarios and provide summary
# 
# Provides detailed analysis of available scenarios
#
# Arguments:
#   $1 - Parent directory (optional)
# Returns: Analysis report to stderr
#######################################
qdrant::extract::analyze_scenarios() {
    local parent_dir="${1:-$APP_ROOT/scenarios}"
    
    echo "=== Scenario Analysis ===" >&2
    echo >&2
    
    if [[ ! -d "$parent_dir" ]]; then
        echo "❌ No scenarios directory found" >&2
        return 1
    fi
    
    # Count scenarios by component type
    local total=0
    local with_prd=0
    local with_service_config=0
    local with_api=0
    local with_cli=0
    local with_ui=0
    local with_tests=0
    local with_docs=0
    local with_initialization=0
    
    while IFS= read -r scenario_dir; do
        ((total++))
        local name=$(basename "$scenario_dir")
        
        # Check for various components
        [[ -f "$scenario_dir/PRD.md" ]] && ((with_prd++))
        [[ -f "$scenario_dir/service.json" ]] || [[ -f "$scenario_dir/.vrooli/service.json" ]] && ((with_service_config++))
        [[ -d "$scenario_dir/api" ]] && ((with_api++))
        [[ -d "$scenario_dir/cli" ]] && ((with_cli++))
        [[ -d "$scenario_dir/ui" ]] && ((with_ui++))
        [[ -d "$scenario_dir/test" ]] || [[ -f "$scenario_dir/scenario-test.yaml" ]] && ((with_tests++))
        [[ -f "$scenario_dir/README.md" ]] || [[ -d "$scenario_dir/docs" ]] && ((with_docs++))
        [[ -d "$scenario_dir/initialization" ]] && ((with_initialization++))
        
        echo "  🎯 $name" >&2
    done < <(find "$parent_dir" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort)
    
    echo >&2
    echo "Summary:" >&2
    echo "  Total scenarios: $total" >&2
    echo >&2
    echo "Component Coverage:" >&2
    echo "  PRD documents: $with_prd/$total" >&2
    echo "  Service config: $with_service_config/$total" >&2
    echo "  API implementation: $with_api/$total" >&2
    echo "  CLI tools: $with_cli/$total" >&2
    echo "  UI frontend: $with_ui/$total" >&2
    echo "  Test suites: $with_tests/$total" >&2
    echo "  Documentation: $with_docs/$total" >&2
    echo "  Initialization: $with_initialization/$total" >&2
    echo >&2
    
    # Analyze completeness levels
    local full_stack_count=0  # Has API + UI + tests
    local backend_only_count=0  # Has API but no UI
    local frontend_only_count=0  # Has UI but no API
    local documented_count=0  # Has PRD + README/docs
    
    while IFS= read -r scenario_dir; do
        local has_api="false"
        local has_ui="false"
        local has_tests="false"
        local has_prd="false"
        local has_docs="false"
        
        [[ -d "$scenario_dir/api" ]] && has_api="true"
        [[ -d "$scenario_dir/ui" ]] && has_ui="true"
        [[ -d "$scenario_dir/test" ]] || [[ -f "$scenario_dir/scenario-test.yaml" ]] && has_tests="true"
        [[ -f "$scenario_dir/PRD.md" ]] && has_prd="true"
        [[ -f "$scenario_dir/README.md" ]] || [[ -d "$scenario_dir/docs" ]] && has_docs="true"
        
        if [[ "$has_api" == "true" && "$has_ui" == "true" && "$has_tests" == "true" ]]; then
            ((full_stack_count++))
        elif [[ "$has_api" == "true" && "$has_ui" == "false" ]]; then
            ((backend_only_count++))
        elif [[ "$has_api" == "false" && "$has_ui" == "true" ]]; then
            ((frontend_only_count++))
        fi
        
        if [[ "$has_prd" == "true" && "$has_docs" == "true" ]]; then
            ((documented_count++))
        fi
    done < <(find "$parent_dir" -mindepth 1 -maxdepth 1 -type d 2>/dev/null)
    
    echo "Architecture Patterns:" >&2
    echo "  Full-stack (API+UI+Tests): $full_stack_count" >&2
    echo "  Backend-only (API, no UI): $backend_only_count" >&2
    echo "  Frontend-only (UI, no API): $frontend_only_count" >&2
    echo "  Well-documented (PRD+README): $documented_count" >&2
    echo >&2
    
    # Test JSON extraction on first scenario
    local first_scenario=$(find "$parent_dir" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | head -1)
    if [[ -n "$first_scenario" ]]; then
        echo "Extraction Test on $(basename "$first_scenario"):" >&2
        
        local sample_output=$(qdrant::extract::scenario_all_json "$first_scenario" 2>/dev/null | head -5)
        local valid_count=0
        local line_count=0
        
        while IFS= read -r line; do
            ((line_count++))
            if echo "$line" | jq -e '.metadata' >/dev/null 2>&1; then
                ((valid_count++))
            fi
        done <<< "$sample_output"
        
        if [[ $valid_count -eq $line_count && $line_count -gt 0 ]]; then
            echo "  ✅ All $line_count lines are valid JSON" >&2
            
            # Show component types found
            echo "  Component types extracted:" >&2
            echo "$sample_output" | jq -r '.metadata.component_type' | sort -u | while read -r type; do
                echo "    • $type" >&2
            done
        else
            echo "  ⚠️  JSON validation: $valid_count/$line_count valid" >&2
        fi
    fi
}

#######################################
# Detect scenario category from content
# 
# Categorizes scenarios based on PRD content and structure
#
# Arguments:
#   $1 - Scenario directory path
# Returns: Category string
#######################################
qdrant::extract::detect_scenario_category() {
    local scenario_dir="$1"
    local scenario_name=$(basename "$scenario_dir")
    
    # Check service.json first if available
    if [[ -f "$scenario_dir/service.json" ]]; then
        local category=$(jq -r '.category // empty' "$scenario_dir/service.json" 2>/dev/null)
        if [[ -n "$category" ]]; then
            echo "$category"
            return
        fi
    fi
    
    # Check .vrooli/service.json
    if [[ -f "$scenario_dir/.vrooli/service.json" ]]; then
        local category=$(jq -r '.category // empty' "$scenario_dir/.vrooli/service.json" 2>/dev/null)
        if [[ -n "$category" ]]; then
            echo "$category"
            return
        fi
    fi
    
    # Categorize based on name patterns and structure
    case "$scenario_name" in
        *dashboard*|*monitor*|*tracker*|*analytics*)
            echo "analytics"
            ;;
        *api*|*server*|*backend*|*service*)
            echo "backend"
            ;;
        *ui*|*frontend*|*web*|*interface*)
            echo "frontend"
            ;;
        *test*|*testing*)
            echo "testing"
            ;;
        *ai*|*agent*|*intelligence*|*assistant*)
            echo "ai"
            ;;
        *data*|*database*|*storage*)
            echo "data"
            ;;
        *workflow*|*automation*)
            echo "automation"
            ;;
        *generator*|*creator*|*builder*)
            echo "generator"
            ;;
        *manager*|*organizer*|*planner*)
            echo "productivity"
            ;;
        *)
            # Categorize based on components
            if [[ -d "$scenario_dir/api" && -d "$scenario_dir/ui" ]]; then
                echo "fullstack"
            elif [[ -d "$scenario_dir/api" ]]; then
                echo "backend"
            elif [[ -d "$scenario_dir/ui" ]]; then
                echo "frontend"
            else
                echo "utility"
            fi
            ;;
    esac
}

# Export main functions
export -f qdrant::extract::scenario_all_json
export -f qdrant::extract::scenarios_batch_json
export -f qdrant::extract::analyze_scenarios
export -f qdrant::extract::detect_scenario_category

# Legacy function names for backward compatibility
qdrant::extract::scenarios_batch() {
    local dir="${1:-.}"
    local output_file="${2:-/tmp/scenarios.jsonl}"
    qdrant::extract::scenarios_batch_json "$dir" "$output_file" >&2
    echo "$output_file"
}

qdrant::extract::find_scenarios() {
    local root="${1:-.}"
    find "$root" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | \
        while read -r dir; do
            if [[ -f "$dir/PRD.md" ]] || [[ -f "$dir/service.json" ]] || [[ -f "$dir/.vrooli/service.json" ]]; then
                echo "$dir"
            fi
        done | sort -u
}

#######################################
# Process scenarios using unified embedding service
# Arguments:
#   $1 - App ID
# Returns: Number of scenarios processed
#######################################
qdrant::embeddings::process_scenarios() {
    local app_id="$1"
    local collection="${app_id}-scenarios"
    local count=0
    
    # Extract scenarios to temp file
    local output_file="$TEMP_DIR/scenarios.jsonl"
    qdrant::extract::scenarios_batch_json "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No scenarios found for processing"
        echo "0"
        return 0
    fi
    
    # Process each JSON line through unified embedding service
    while IFS= read -r json_line; do
        if [[ -n "$json_line" ]]; then
            # Parse JSON to extract content and metadata
            local content
            content=$(echo "$json_line" | jq -r '.content // empty' 2>/dev/null)
            
            local metadata
            metadata=$(echo "$json_line" | jq -c '.metadata // {}' 2>/dev/null)
            
            if [[ -n "$content" ]]; then
                # Process through unified embedding service with structured metadata
                if qdrant::embedding::process_item "$content" "scenario" "$collection" "$app_id" "$metadata"; then
                    ((count++))
                fi
            fi
        fi
    done < "$output_file"
    
    log::debug "Created $count scenario embeddings"
    echo "$count"
}

export -f qdrant::extract::scenarios_batch
export -f qdrant::extract::find_scenarios
export -f qdrant::embeddings::process_scenarios