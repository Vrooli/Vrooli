#!/usr/bin/env bash
set -euo pipefail

DESCRIPTION="Inject agents and scenarios into Huginn automation platform"

# Source guard to prevent multiple sourcing
[[ -n "${_HUGINN_INJECT_SOURCED:-}" ]] && return 0
export _HUGINN_INJECT_SOURCED=1

# Get script directory and source framework
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HUGINN_LIB_DIR="${APP_ROOT}/resources/huginn/lib"

# shellcheck disable=SC1091
source "${HUGINN_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/inject_framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/validation_utils.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

# Load Huginn configuration and infrastructure
if command -v inject_framework::load_adapter_config &>/dev/null; then
    inject_framework::load_adapter_config "huginn" "${HUGINN_LIB_DIR%/*"
fi

# Source Huginn lib functions (load common, status, and api)
for lib_file in "${HUGINN_LIB_DIR}/"{common,status,api}.sh; do
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" || log::warn "Could not load $lib_file"
    fi
done

# Huginn-specific configuration with readonly protection
if [[ -z "${HUGINN_BASE_URL:-}" ]]; then
    HUGINN_BASE_URL="http://localhost:4111"
fi
if ! readonly -p | grep -q "HUGINN_BASE_URL"; then
    readonly HUGINN_BASE_URL
fi

#######################################
# Huginn-specific health check
# Returns: 0 if healthy, 1 otherwise
#######################################
huginn::check_health() {
    # Use existing Huginn health check from status.sh
    if huginn::health_check 2>/dev/null; then
        log::debug "Huginn is healthy and ready for data injection"
        return 0
    else
        log::error "Huginn is not accessible for data injection"
        log::info "Ensure Huginn is running: vrooli resource huginn start"
        return 1
    fi
}

#######################################
# Validate individual agent
# Arguments:
#   $1 - agent configuration object  
#   $2 - index
#   $3 - agent name
# Returns:
#   0 if valid, 1 if invalid
#######################################
huginn::validate_agent() {
    local agent="$1"
    local index="$2" 
    local name="$3"
    
    # Validate required fields with context
    local file
    file=$(echo "$agent" | jq -r '.file // empty')
    if [[ -z "$file" ]]; then
        log::error "Agent '$name' at index $index missing required 'file' field"
        return 1
    fi
    
    # Validate agent file exists and is valid JSON - use same path resolution as import
    local agent_file
    if [[ "$file" == /* ]]; then
        # Absolute path
        agent_file="$file"
    elif [[ -n "${HUGINN_SCENARIO_DIR:-}" ]]; then
        agent_file="$HUGINN_SCENARIO_DIR/$file"
    else
        agent_file=$(inject_framework::resolve_file_path "$file")
    fi
    
    if [[ ! -f "$agent_file" ]]; then
        log::error "Agent file not found for '$name': $agent_file"
        log::debug "Checked paths: HUGINN_SCENARIO_DIR=${HUGINN_SCENARIO_DIR:-unset}, resolved_path=$agent_file"
        return 1
    fi
    
    # Validate JSON structure
    if ! jq empty "$agent_file" 2>/dev/null; then
        log::error "Invalid JSON in agent file for '$name': $agent_file"
        return 1
    fi
    
    log::debug "Agent '$name' validated successfully"
    return 0
}

#######################################
# Process and import a single agent
# Arguments:
#   $1 - agent configuration object
#   $2 - index (for logging)
#   $3 - agent name (for logging)
# Returns:
#   0 on success, 1 on failure
#######################################
huginn::process_agent() {
    local agent="$1"
    local index="$2"
    local name="$3"
    
    # Extract file path
    local file
    file=$(echo "$agent" | jq -r '.file // empty')
    
    # Resolve full path
    local agent_file
    if [[ "$file" == /* ]]; then
        # Absolute path
        agent_file="$file"
    elif [[ -n "${HUGINN_SCENARIO_DIR:-}" ]]; then
        agent_file="$HUGINN_SCENARIO_DIR/$file"
    else
        agent_file=$(inject_framework::resolve_file_path "$file")
    fi
    
    log::info "Processing agent '$name' from $agent_file"
    
    # Read agent content
    local agent_content
    agent_content=$(cat "$agent_file")
    
    # Import agent via Huginn API
    if huginn::api_import_agent "$agent_content"; then
        log::success "Successfully imported agent '$name'"
        return 0
    else
        log::error "Failed to import agent '$name'"
        return 1
    fi
}

#######################################
# Process and import a single scenario
# Arguments:
#   $1 - scenario configuration object
#   $2 - index (for logging)
#   $3 - scenario name (for logging)
# Returns:
#   0 on success, 1 on failure
#######################################
huginn::process_scenario() {
    local scenario="$1"
    local index="$2"
    local name="$3"
    
    # Extract file path
    local file
    file=$(echo "$scenario" | jq -r '.file // empty')
    
    # Resolve full path
    local scenario_file
    if [[ "$file" == /* ]]; then
        # Absolute path
        scenario_file="$file"
    elif [[ -n "${HUGINN_SCENARIO_DIR:-}" ]]; then
        scenario_file="$HUGINN_SCENARIO_DIR/$file"
    else
        scenario_file=$(inject_framework::resolve_file_path "$file")
    fi
    
    log::info "Processing scenario '$name' from $scenario_file"
    
    # Read scenario content
    local scenario_content
    scenario_content=$(cat "$scenario_file")
    
    # Import scenario via Huginn API
    if huginn::api_import_scenario "$scenario_content"; then
        log::success "Successfully imported scenario '$name'"
        return 0
    else
        log::error "Failed to import scenario '$name'"
        return 1
    fi
}

#######################################
# Main injection implementation for Huginn
# Globals:
#   HUGINN_BASE_URL
#   HUGINN_API_KEY (if required)
# Arguments:
#   None (uses inject_framework context)
# Returns:
#   0 on success, 1 on failure
#######################################
huginn::inject_data() {
    log::header "Huginn Data Injection"
    
    # Check health first
    if ! huginn::check_health; then
        return 1
    fi
    
    local total_success=0
    local total_failed=0
    
    # Process agents if present
    local agents_json="${INJECT_DATA_JSON:-}"
    if [[ -n "$agents_json" ]]; then
        local agents
        agents=$(echo "$agents_json" | jq -c '.agents // []')
        
        if [[ "$agents" != "[]" && "$agents" != "null" ]]; then
            log::info "Processing Huginn agents..."
            
            local agent_count
            agent_count=$(echo "$agents" | jq 'length')
            log::debug "Found $agent_count agents to process"
            
            for i in $(seq 0 $((agent_count - 1))); do
                local agent
                agent=$(echo "$agents" | jq -c ".[$i]")
                local agent_name
                agent_name=$(echo "$agent" | jq -r '.name // "unnamed"')
                
                # Validate agent
                if ! huginn::validate_agent "$agent" "$i" "$agent_name"; then
                    ((total_failed++))
                    continue
                fi
                
                # Process agent
                if huginn::process_agent "$agent" "$i" "$agent_name"; then
                    ((total_success++))
                else
                    ((total_failed++))
                fi
            done
        fi
    fi
    
    # Process scenarios if present
    local scenarios_json="${INJECT_DATA_JSON:-}"
    if [[ -n "$scenarios_json" ]]; then
        local scenarios
        scenarios=$(echo "$scenarios_json" | jq -c '.scenarios // []')
        
        if [[ "$scenarios" != "[]" && "$scenarios" != "null" ]]; then
            log::info "Processing Huginn scenarios..."
            
            local scenario_count
            scenario_count=$(echo "$scenarios" | jq 'length')
            log::debug "Found $scenario_count scenarios to process"
            
            for i in $(seq 0 $((scenario_count - 1))); do
                local scenario
                scenario=$(echo "$scenarios" | jq -c ".[$i]")
                local scenario_name
                scenario_name=$(echo "$scenario" | jq -r '.name // "unnamed"')
                
                # Process scenario (using agent validator for now)
                if ! huginn::validate_agent "$scenario" "$i" "$scenario_name"; then
                    ((total_failed++))
                    continue
                fi
                
                # Process scenario
                if huginn::process_scenario "$scenario" "$i" "$scenario_name"; then
                    ((total_success++))
                else
                    ((total_failed++))
                fi
            done
        fi
    fi
    
    # Report results
    log::info "Injection complete: $total_success successful, $total_failed failed"
    
    if [[ $total_failed -gt 0 ]]; then
        return 1
    fi
    
    return 0
}

# Register adapter with injection framework
if command -v inject_framework::register_adapter &>/dev/null; then
    inject_framework::register_adapter "huginn" "huginn::inject_data" "huginn::check_health" "huginn::validate_agent"
fi

# Export main function for direct use
export -f huginn::inject_data
export -f huginn::check_health
export -f huginn::validate_agent