#!/usr/bin/env bash
################################################################################
# Population System v2.0 - Scenario Management
# Functions for loading and managing scenarios
################################################################################
set -euo pipefail

# Default scenario locations - delayed evaluation to avoid unbound variable errors
get_scenario_search_paths() {
    echo "${var_SCENARIOS_DIR:-$(dirname "$0")/../../scenarios}"
    echo "${APP_ROOT}/scenarios"
    echo "${HOME}/.vrooli/scenarios" 
    echo "${PWD}/scenarios"
}

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
        while IFS= read -r path; do
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
        done < <(get_scenario_search_paths)
    fi
    
    if [[ -z "$scenario_file" ]]; then
        log::error "Scenario not found: $scenario"
        log::info "Searched locations:"
        while IFS= read -r path; do
            log::info "  - $path"
        done < <(get_scenario_search_paths)
        return 1
    fi
    
    log::debug "Loading scenario from: $scenario_file"
    
    # Load and process the file
    local config
    config=$(cat "$scenario_file")
    
    # Require dependencies.resources map to be present
    if echo "$config" | jq -e '.dependencies.resources' >/dev/null 2>&1; then
        echo "$config"
    else
        log::error "Scenario configuration missing dependencies.resources"
        return 1
    fi
}

#######################################
# List available scenarios
# Outputs:
#   List of scenario names with descriptions
#######################################
scenario::list() {
    local found_any=false
    
    while IFS= read -r path; do
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
    done < <(get_scenario_search_paths)
    
    if [[ "$found_any" == "false" ]]; then
        log::info "No scenarios found in:"
        while IFS= read -r path; do
            log::info "  - $path"
        done < <(get_scenario_search_paths)
        log::info ""
        log::info "Create scenarios in these locations or specify a path directly"
    fi
}
