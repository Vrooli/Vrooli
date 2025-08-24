#!/usr/bin/env bash
# Scenario Content Extractor for Qdrant Embeddings
# Extracts embeddable content from scenario PRDs and configuration files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Define paths from APP_ROOT
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
EXTRACTOR_DIR="${EMBEDDINGS_DIR}/extractors"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified embedding service
source "${EMBEDDINGS_DIR}/lib/embedding-service.sh"

# Temporary directory for extracted content
EXTRACT_TEMP_DIR="/tmp/qdrant-scenario-extract-$$"
trap "rm -rf $EXTRACT_TEMP_DIR" EXIT

#######################################
# Extract content from a PRD.md file
# Arguments:
#   $1 - Path to PRD.md file
# Returns: Extracted content as text
#######################################
qdrant::extract::prd() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        log::error "PRD file not found: $file"
        return 1
    fi
    
    # Extract key sections from PRD
    local title=$(grep "^# " "$file" 2>/dev/null | head -1 | sed 's/^# //' || echo "Unknown Scenario")
    local scenario_dir=$(dirname "$file")
    local scenario_name=$(basename "$scenario_dir")
    
    # Extract executive summary (usually after first heading)
    local summary=$(awk '/^## Executive Summary/,/^##[^#]/' "$file" 2>/dev/null | grep -v "^##" | tr '\n' ' ' | sed 's/  */ /g' || echo "")
    
    # Extract value proposition
    local value_prop=$(awk '/^## Value Proposition/,/^##[^#]/' "$file" 2>/dev/null | grep -v "^##" | tr '\n' ' ' | sed 's/  */ /g' || echo "")
    
    # Extract target users
    local target_users=$(awk '/^## Target Users/,/^##[^#]/' "$file" 2>/dev/null | grep -v "^##" | tr '\n' ' ' | sed 's/  */ /g' || echo "")
    
    # Extract key features
    local features=$(awk '/^## Key Features/,/^##[^#]/' "$file" 2>/dev/null | grep "^- " | sed 's/^- //' | tr '\n' ', ' | sed 's/,$//' || echo "")
    
    # Extract success metrics
    local metrics=$(awk '/^## Success Metrics/,/^##[^#]/' "$file" 2>/dev/null | grep "^- " | sed 's/^- //' | tr '\n' ', ' | sed 's/,$//' || echo "")
    
    # Extract revenue model
    local revenue=$(awk '/^## Revenue Model/,/^##[^#]/' "$file" 2>/dev/null | grep -v "^##" | tr '\n' ' ' | sed 's/  */ /g' || echo "")
    
    # Build embeddable content
    local content="Scenario: $title
Name: $scenario_name
File: $file"
    
    if [[ -n "$summary" ]]; then
        content="$content
Summary: $summary"
    fi
    
    if [[ -n "$value_prop" ]]; then
        content="$content
Value Proposition: $value_prop"
    fi
    
    if [[ -n "$target_users" ]]; then
        content="$content
Target Users: $target_users"
    fi
    
    if [[ -n "$features" ]]; then
        content="$content
Key Features: $features"
    fi
    
    if [[ -n "$metrics" ]]; then
        content="$content
Success Metrics: $metrics"
    fi
    
    if [[ -n "$revenue" ]]; then
        content="$content
Revenue Model: $revenue"
    fi
    
    # Add technical requirements if present
    local tech_stack=$(awk '/^## Technical Requirements/,/^##[^#]/' "$file" 2>/dev/null | grep "^- " | sed 's/^- //' | tr '\n' ', ' | sed 's/,$//' || echo "")
    if [[ -n "$tech_stack" ]]; then
        content="$content
Technical Stack: $tech_stack"
    fi
    
    echo "$content"
}

#######################################
# Extract content from .scenario.yaml file
# Arguments:
#   $1 - Path to .scenario.yaml file
# Returns: Extracted content as text
#######################################
qdrant::extract::scenario_config() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        log::error "Scenario config not found: $file"
        return 1
    fi
    
    # Parse YAML (basic extraction without yq dependency)
    local name=$(grep "^name:" "$file" | cut -d: -f2- | tr -d ' "' || echo "")
    local description=$(grep "^description:" "$file" | cut -d: -f2- | tr -d '"' | sed 's/^ *//' || echo "")
    local version=$(grep "^version:" "$file" | cut -d: -f2- | tr -d ' "' || echo "")
    local category=$(grep "^category:" "$file" | cut -d: -f2- | tr -d ' "' || echo "")
    local status=$(grep "^status:" "$file" | cut -d: -f2- | tr -d ' "' || echo "")
    local complexity=$(grep "^complexity:" "$file" | cut -d: -f2- | tr -d ' "' || echo "")
    
    # Extract resources
    local resources=$(awk '/^resources:/,/^[a-z]/' "$file" 2>/dev/null | grep "  - " | sed 's/  - //' | tr '\n' ', ' | sed 's/,$//' || echo "")
    
    # Extract tags
    local tags=$(awk '/^tags:/,/^[a-z]/' "$file" 2>/dev/null | grep "  - " | sed 's/  - //' | tr '\n' ', ' | sed 's/,$//' || echo "")
    
    # Build content
    local content="Scenario Configuration: $name
Version: $version
File: $file"
    
    [[ -n "$description" ]] && content="$content
Description: $description"
    
    [[ -n "$category" ]] && content="$content
Category: $category"
    
    [[ -n "$status" ]] && content="$content
Status: $status"
    
    [[ -n "$complexity" ]] && content="$content
Complexity: $complexity"
    
    [[ -n "$resources" ]] && content="$content
Required Resources: $resources"
    
    [[ -n "$tags" ]] && content="$content
Tags: $tags"
    
    echo "$content"
}

#######################################
# Extract content from all scenarios
# Arguments:
#   $1 - Directory path (optional, defaults to current)
# Returns: 0 on success
#######################################
qdrant::extract::scenarios_batch() {
    local dir="${1:-.}"
    local output_file="${2:-${EXTRACT_TEMP_DIR}/scenarios.txt}"
    
    mkdir -p "$(dirname "$output_file")"
    
    # Find all scenario files
    local prd_files=()
    local config_files=()
    
    while IFS= read -r file; do
        prd_files+=("$file")
    done < <(find "$dir" -type f -name "PRD.md" -path "*/scenarios/*" 2>/dev/null)
    
    while IFS= read -r file; do
        config_files+=("$file")
    done < <(find "$dir" -type f -name ".scenario.yaml" 2>/dev/null)
    
    local total_files=$((${#prd_files[@]} + ${#config_files[@]}))
    
    if [[ $total_files -eq 0 ]]; then
        log::warn "No scenario files found in $dir"
        return 0
    fi
    
    log::info "Extracting content from $total_files scenario files"
    
    # Extract PRDs
    local count=0
    for file in "${prd_files[@]}"; do
        local content=$(qdrant::extract::prd "$file")
        if [[ -n "$content" ]]; then
            echo "$content" >> "$output_file"
            echo "---SEPARATOR---" >> "$output_file"
            ((count++))
        fi
    done
    
    # Extract configs
    for file in "${config_files[@]}"; do
        local content=$(qdrant::extract::scenario_config "$file")
        if [[ -n "$content" ]]; then
            echo "$content" >> "$output_file"
            echo "---SEPARATOR---" >> "$output_file"
            ((count++))
        fi
    done
    
    log::success "Extracted content from $count scenario files"
    echo "$output_file"
}

#######################################
# Create metadata for scenario embedding
# Arguments:
#   $1 - Scenario file path
# Returns: JSON metadata
#######################################
qdrant::extract::scenario_metadata() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "{}"
        return
    fi
    
    local file_type="unknown"
    local scenario_name=""
    local scenario_path=$(dirname "$file")
    
    if [[ "$file" == *"PRD.md" ]]; then
        file_type="prd"
        scenario_name=$(basename "$scenario_path")
    elif [[ "$file" == *".scenario.yaml" ]]; then
        file_type="config"
        scenario_name=$(grep "^name:" "$file" | cut -d: -f2- | tr -d ' "' || basename "$scenario_path")
    fi
    
    # Get file stats
    local file_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
    local modified=$(stat -f%Sm -t '%Y-%m-%dT%H:%M:%S' "$file" 2>/dev/null || stat -c %y "$file" 2>/dev/null | cut -d' ' -f1-2 | tr ' ' 'T')
    
    # Check for associated files
    local has_prd="false"
    local has_config="false"
    local has_workflows="false"
    
    [[ -f "$scenario_path/PRD.md" ]] && has_prd="true"
    [[ -f "$scenario_path/.scenario.yaml" ]] && has_config="true"
    [[ -d "$scenario_path/initialization" ]] && has_workflows="true"
    
    # Build metadata JSON
    jq -n \
        --arg name "$scenario_name" \
        --arg file "$file" \
        --arg file_type "$file_type" \
        --arg scenario_path "$scenario_path" \
        --arg file_size "$file_size" \
        --arg modified "$modified" \
        --arg has_prd "$has_prd" \
        --arg has_config "$has_config" \
        --arg has_workflows "$has_workflows" \
        --arg type "scenario" \
        --arg extractor "scenarios" \
        '{
            scenario_name: $name,
            source_file: $file,
            file_type: $file_type,
            scenario_path: $scenario_path,
            file_size: ($file_size | tonumber),
            file_modified: $modified,
            has_prd: ($has_prd == "true"),
            has_config: ($has_config == "true"),
            has_workflows: ($has_workflows == "true"),
            content_type: $type,
            extractor: $extractor
        }'
}

#######################################
# Find all scenario directories
# Arguments:
#   $1 - Root directory to search
# Returns: List of scenario directories
#######################################
qdrant::extract::find_scenarios() {
    local root="${1:-.}"
    
    # Find directories containing PRD.md or .scenario.yaml
    local dirs=()
    
    while IFS= read -r prd_file; do
        local dir=$(dirname "$prd_file")
        dirs+=("$dir")
    done < <(find "$root" -type f -name "PRD.md" -path "*/scenarios/*" 2>/dev/null)
    
    while IFS= read -r config_file; do
        local dir=$(dirname "$config_file")
        dirs+=("$dir")
    done < <(find "$root" -type f -name ".scenario.yaml" 2>/dev/null)
    
    # Remove duplicates
    printf '%s\n' "${dirs[@]}" | sort -u
}

#######################################
# Generate summary of all scenarios
# Arguments:
#   $1 - Directory to analyze
# Returns: Summary report
#######################################
qdrant::extract::scenarios_summary() {
    local dir="${1:-.}"
    
    echo "=== Scenario Analysis Summary ==="
    echo
    
    local scenario_dirs=()
    while IFS= read -r scenario_dir; do
        scenario_dirs+=("$scenario_dir")
    done < <(qdrant::extract::find_scenarios "$dir")
    
    if [[ ${#scenario_dirs[@]} -eq 0 ]]; then
        echo "No scenarios found"
        return 0
    fi
    
    echo "Total Scenarios: ${#scenario_dirs[@]}"
    echo
    
    # Analyze scenario completeness
    local complete_count=0
    local with_prd=0
    local with_config=0
    local with_workflows=0
    local categories=()
    local statuses=()
    
    for scenario_dir in "${scenario_dirs[@]}"; do
        local scenario_name=$(basename "$scenario_dir")
        
        # Check components
        local has_all=true
        
        if [[ -f "$scenario_dir/PRD.md" ]]; then
            ((with_prd++))
        else
            has_all=false
        fi
        
        if [[ -f "$scenario_dir/.scenario.yaml" ]]; then
            ((with_config++))
            
            # Extract category and status
            local category=$(grep "^category:" "$scenario_dir/.scenario.yaml" | cut -d: -f2- | tr -d ' "' || echo "")
            local status=$(grep "^status:" "$scenario_dir/.scenario.yaml" | cut -d: -f2- | tr -d ' "' || echo "")
            
            [[ -n "$category" ]] && categories+=("$category")
            [[ -n "$status" ]] && statuses+=("$status")
        else
            has_all=false
        fi
        
        if [[ -d "$scenario_dir/initialization" ]]; then
            ((with_workflows++))
        else
            has_all=false
        fi
        
        [[ "$has_all" == "true" ]] && ((complete_count++))
    done
    
    echo "Complete Scenarios: $complete_count / ${#scenario_dirs[@]}"
    echo "With PRD: $with_prd"
    echo "With Config: $with_config"
    echo "With Workflows: $with_workflows"
    echo
    
    # Category breakdown
    if [[ ${#categories[@]} -gt 0 ]]; then
        echo "Categories:"
        printf '%s\n' "${categories[@]}" | sort | uniq -c | sort -rn | while read count cat; do
            echo "  $count: $cat"
        done
        echo
    fi
    
    # Status breakdown
    if [[ ${#statuses[@]} -gt 0 ]]; then
        echo "Status:"
        printf '%s\n' "${statuses[@]}" | sort | uniq -c | sort -rn | while read count status; do
            echo "  $count: $status"
        done
        echo
    fi
    
    # List scenarios
    echo "Scenarios:"
    for scenario_dir in "${scenario_dirs[@]}"; do
        local scenario_name=$(basename "$scenario_dir")
        echo "  • $scenario_name"
    done
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
    local output_file="$TEMP_DIR/scenarios.txt"
    qdrant::extract::scenarios_batch "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No scenarios found for processing"
        echo "0"
        return 0
    fi
    
    # Process each scenario through unified embedding service
    local content=""
    local processing_scenario=false
    
    while IFS= read -r line; do
        if [[ "$line" == "Scenario:"* ]] || [[ "$line" == "Scenario Configuration:"* ]]; then
            # Start of new scenario content
            processing_scenario=true
            content="$line"
        elif [[ "$line" == "---SEPARATOR---" ]] && [[ "$processing_scenario" == true ]]; then
            # End of scenario, process it
            if [[ -n "$content" ]]; then
                # Extract scenario metadata from content
                local metadata
                metadata=$(qdrant::extract::scenario_metadata_from_content "$content")
                
                # Process through unified embedding service
                if qdrant::embedding::process_item "$content" "scenario" "$collection" "$app_id" "$metadata"; then
                    ((count++))
                fi
            fi
            processing_scenario=false
            content=""
        elif [[ "$processing_scenario" == true ]]; then
            # Continue accumulating scenario content
            content="${content}"$'\n'"${line}"
        fi
    done < "$output_file"
    
    log::debug "Created $count scenario embeddings"
    echo "$count"
}

#######################################
# Extract metadata from scenario content text
# Arguments:
#   $1 - Scenario content text
# Returns: JSON metadata object
#######################################
qdrant::extract::scenario_metadata_from_content() {
    local content="$1"
    
    # Extract key information from the content text
    local name
    if [[ "$content" == "Scenario:"* ]]; then
        # PRD format
        name=$(echo "$content" | grep "^Scenario:" | cut -d: -f2- | sed 's/^ *//')
    elif [[ "$content" == "Scenario Configuration:"* ]]; then
        # Config format
        name=$(echo "$content" | grep "^Scenario Configuration:" | cut -d: -f2- | sed 's/^ *//')
    fi
    
    local file_path
    file_path=$(echo "$content" | grep -o "File: .*" | cut -d: -f2- | sed 's/^ *//')
    
    # Extract other metadata from content
    local category
    category=$(echo "$content" | grep -o "Category: [^$]*" | cut -d: -f2- | sed 's/^ *//' | head -1)
    
    local status
    status=$(echo "$content" | grep -o "Status: [^$]*" | cut -d: -f2- | sed 's/^ *//' | head -1)
    
    local complexity
    complexity=$(echo "$content" | grep -o "Complexity: [^$]*" | cut -d: -f2- | sed 's/^ *//' | head -1)
    
    local version
    version=$(echo "$content" | grep -o "Version: [^$]*" | cut -d: -f2- | sed 's/^ *//' | head -1)
    
    local resources
    resources=$(echo "$content" | grep -o "Required Resources: [^$]*" | cut -d: -f2- | sed 's/^ *//' | head -1)
    
    local features
    features=$(echo "$content" | grep -o "Key Features: [^$]*" | cut -d: -f2- | sed 's/^ *//' | head -1)
    
    local target_users
    target_users=$(echo "$content" | grep -o "Target Users: [^$]*" | cut -d: -f2- | sed 's/^ *//' | head -1)
    
    # Determine file type
    local file_type="prd"
    if [[ "$file_path" == *".scenario.yaml" ]]; then
        file_type="config"
    fi
    
    # Build metadata JSON
    jq -n \
        --arg name "${name:-Unknown}" \
        --arg file_path "${file_path:-}" \
        --arg category "${category:-}" \
        --arg status "${status:-}" \
        --arg complexity "${complexity:-}" \
        --arg version "${version:-}" \
        --arg resources "${resources:-}" \
        --arg features "${features:-}" \
        --arg target_users "${target_users:-}" \
        --arg file_type "$file_type" \
        '{
            scenario_name: $name,
            source_file: $file_path,
            category: $category,
            status: $status,
            complexity: $complexity,
            version: $version,
            required_resources: $resources,
            key_features: $features,
            target_users: $target_users,
            file_type: $file_type,
            content_type: "scenario",
            extractor: "scenarios"
        }'
}

# Export processing function for manage.sh
export -f qdrant::embeddings::process_scenarios