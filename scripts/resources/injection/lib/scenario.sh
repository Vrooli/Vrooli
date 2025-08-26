#!/usr/bin/env bash
################################################################################
# Injection System v2.0 - Scenario Management
# Functions for loading and managing scenarios
################################################################################
set -euo pipefail

# Default scenario locations
SCENARIO_SEARCH_PATHS=(
    "${var_SCENARIOS_DIR}"
    "${APP_ROOT}/scenarios"
    "${HOME}/.vrooli/scenarios"
    "${PWD}/scenarios"
)

#######################################
# Load scenario configuration
# Arguments:
#   $1 - scenario name or path
# Outputs:
#   JSON configuration
# Returns:
#   0 on success, 1 on failure
#######################################
scenario::load() {
    local scenario="$1"
    local scenario_file=""
    
    # If it's a file path, use directly
    if [[ -f "$scenario" ]]; then
        scenario_file="$scenario"
    else
        # Search for scenario in standard locations
        for path in "${SCENARIO_SEARCH_PATHS[@]}"; do
            # Try different naming patterns
            for pattern in \
                "$path/$scenario/scenario.json" \
                "$path/$scenario/.vrooli/service.json" \
                "$path/$scenario.json"; do
                
                if [[ -f "$pattern" ]]; then
                    scenario_file="$pattern"
                    break 2
                fi
            done
        done
    fi
    
    if [[ -z "$scenario_file" ]]; then
        log::error "Scenario not found: $scenario"
        log::info "Searched locations:"
        for path in "${SCENARIO_SEARCH_PATHS[@]}"; do
            log::info "  - $path"
        done
        return 1
    fi
    
    log::debug "Loading scenario from: $scenario_file"
    
    # Load and process the file
    local config
    config=$(cat "$scenario_file")
    
    # Handle different scenario formats
    # New format: direct scenario object
    if echo "$config" | jq -e '.resources' >/dev/null 2>&1; then
        echo "$config"
    # Legacy format: nested in service.json
    elif echo "$config" | jq -e '.service' >/dev/null 2>&1; then
        # Extract and transform legacy format
        scenario::transform_legacy "$config"
    else
        log::error "Unrecognized scenario format"
        return 1
    fi
}

#######################################
# Transform legacy service.json format to new format
# Arguments:
#   $1 - legacy configuration (JSON)
# Outputs:
#   Transformed configuration (JSON)
#######################################
scenario::transform_legacy() {
    local legacy_config="$1"
    
    # Extract name and description
    local name
    local description
    
    name=$(echo "$legacy_config" | jq -r '.service.name // "unknown"')
    description=$(echo "$legacy_config" | jq -r '.service.description // ""')
    
    # Build new format
    local new_config
    new_config=$(jq -n \
        --arg name "$name" \
        --arg desc "$description" \
        --argjson resources "$(echo "$legacy_config" | scenario::extract_resources)" \
        '{
            name: $name,
            description: $desc,
            resources: $resources
        }')
    
    echo "$new_config"
}

#######################################
# Extract resources with content from legacy format
# Arguments:
#   stdin - legacy configuration
# Outputs:
#   Resources with content (JSON)
#######################################
scenario::extract_resources() {
    jq '
    .resources | 
    to_entries |
    map(
        .value |
        to_entries |
        map({
            key: .key,
            value: {
                content: (
                    if .value.initialization then
                        .value.initialization
                    elif .value.workflows then
                        .value.workflows | map(. + {type: "workflow"})
                    elif .value.data then
                        .value.data
                    else
                        []
                    end
                )
            }
        })
    ) |
    flatten |
    from_entries
    '
}

#######################################
# List available scenarios
# Outputs:
#   List of scenario names with descriptions
#######################################
scenario::list() {
    local found_any=false
    
    for path in "${SCENARIO_SEARCH_PATHS[@]}"; do
        if [[ ! -d "$path" ]]; then
            continue
        fi
        
        # Look for scenario directories
        for scenario_dir in "$path"/*; do
            if [[ ! -d "$scenario_dir" ]]; then
                continue
            fi
            
            local scenario_name
            scenario_name=$(basename "$scenario_dir")
            
            # Look for scenario file
            local scenario_file=""
            for pattern in \
                "$scenario_dir/scenario.json" \
                "$scenario_dir/.vrooli/service.json"; do
                
                if [[ -f "$pattern" ]]; then
                    scenario_file="$pattern"
                    break
                fi
            done
            
            if [[ -n "$scenario_file" ]]; then
                found_any=true
                local description
                description=$(jq -r '.description // .service.description // ""' "$scenario_file" 2>/dev/null || echo "")
                
                echo "• $scenario_name"
                if [[ -n "$description" ]]; then
                    echo "  $description"
                fi
                echo ""
            fi
        done
        
        # Look for standalone JSON files
        for json_file in "$path"/*.json; do
            if [[ -f "$json_file" ]]; then
                found_any=true
                local scenario_name
                scenario_name=$(basename "$json_file" .json)
                
                local description
                description=$(jq -r '.description // ""' "$json_file" 2>/dev/null || echo "")
                
                echo "• $scenario_name"
                if [[ -n "$description" ]]; then
                    echo "  $description"
                fi
                echo ""
            fi
        done
    done
    
    if [[ "$found_any" == "false" ]]; then
        log::info "No scenarios found in:"
        for path in "${SCENARIO_SEARCH_PATHS[@]}"; do
            log::info "  - $path"
        done
        log::info ""
        log::info "Create scenarios in these locations or specify a path directly"
    fi
}