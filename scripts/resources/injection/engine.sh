#!/usr/bin/env bash
set -euo pipefail

# Resource Data Injection Engine - Simplified but Compatible Version
# Maintains interface compatibility while removing unnecessary complexity

SCRIPTS_SCENARIOS_INJECTION_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPTS_SCENARIOS_INJECTION_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

DEFAULT_SCENARIOS_DIR="${var_SCRIPTS_SCENARIOS_DIR}/core"

# Parse arguments
ACTION="inject"
SCENARIO_NAME=""
CONFIG_FILE=""
ALL_ACTIVE="no"
DRY_RUN="no"
FORCE="no"
INITIALIZATION_DIR=""

#######################################
# Display usage
#######################################
usage() {
    cat << EOF
Resource Data Injection Engine

USAGE:
    $0 [OPTIONS] [INITIALIZATION_DIRECTORY]

OPTIONS:
    -a, --action ACTION        Action to perform (default: inject)
                              Options: inject, validate, list-scenarios
    -s, --scenario NAME        Scenario name to process
        --config-file PATH     Config file (ignored - for compatibility)
        --all-active          Process all scenarios (yes/no, default: no)
        --dry-run             Show what would be done (yes/no, default: no)
        --force               Force injection (yes/no, default: no)
    -h, --help                Show this help

ARGUMENTS:
    INITIALIZATION_DIRECTORY   Directory containing resource initialization files
                              Structure: category/resource/files
                              Example: initialization/automation/comfyui/

EXAMPLES:
    # Inject from initialization directory
    $0 initialization/
    
    # Inject specific scenario
    $0 --action inject --scenario scenario-name
    
    # Inject all scenarios
    $0 --action inject --all-active yes
    
    # Validate scenario
    $0 --action validate --scenario scenario-name
    
    # List available scenarios
    $0 --action list-scenarios

EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -a|--action)
            ACTION="$2"
            shift 2
            ;;
        -s|--scenario)
            SCENARIO_NAME="$2"
            shift 2
            ;;
        --config-file)
            CONFIG_FILE="$2"  # Accept but ignore
            shift 2
            ;;
        --all-active)
            ALL_ACTIVE="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN="$2"
            shift 2
            ;;
        --force|--yes|-y)
            FORCE="yes"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            # If it's a directory path, treat it as initialization directory
            if [[ -d "$1" ]] || [[ "$1" =~ ^[^-] ]]; then
                INITIALIZATION_DIR="$1"
                shift
            else
                log::error "Unknown option: $1"
                usage
                exit 1
            fi
            ;;
    esac
done

#######################################
# Resolve secrets
#######################################
resolve_secret() {
    local key="$1"
    
    # Try environment variable
    local value="${!key:-}"
    if [[ -n "$value" ]]; then
        echo "$value"
        return 0
    fi
    
    # Try .vrooli/secrets.json
    if [[ -f ".vrooli/secrets.json" ]]; then
        value=$(jq -r ".$key // empty" .vrooli/secrets.json 2>/dev/null || true)
        if [[ -n "$value" && "$value" != "null" ]]; then
            echo "$value"
            return 0
        fi
    fi
    
    echo ""
}

#######################################
# Substitute secrets in JSON
#######################################
substitute_secrets() {
    local json="$1"
    local result="$json"
    
    while [[ "$result" =~ \{\{([^}]+)\}\} ]]; do
        local placeholder="${BASH_REMATCH[0]}"
        local secret_name="${BASH_REMATCH[1]}"
        local secret_value
        
        secret_value=$(resolve_secret "$secret_name")
        if [[ -n "$secret_value" ]]; then
            secret_value=$(echo "$secret_value" | sed 's/\\/\\\\/g; s/"/\\"/g')
            result="${result//$placeholder/$secret_value}"
        else
            log::warn "Secret not found: $secret_name"
            result="${result//$placeholder/}"
        fi
    done
    
    echo "$result"
}

#######################################
# Find manage script for resource
#######################################
find_manage_script() {
    local resource="$1"
    find "${var_SCRIPTS_RESOURCES_DIR}" -name "manage.sh" -path "*/${resource}/*" 2>/dev/null | head -1
}

#######################################
# Inject resource data
#######################################
inject_resource() {
    local resource_name="$1"
    local resource_config="$2"
    local scenario_dir="$3"
    
    # Check if enabled
    local enabled
    enabled=$(echo "$resource_config" | jq -r '.enabled // true')
    if [[ "$enabled" != "true" ]]; then
        log::debug "Resource $resource_name is disabled"
        return 0
    fi
    
    # Find manage script
    local manage_script
    manage_script=$(find_manage_script "$resource_name")
    
    if [[ -z "$manage_script" ]] || [[ ! -x "$manage_script" ]]; then
        log::debug "No manage script for: $resource_name"
        return 0
    fi
    
    # Get initialization data (handle both array and object formats)
    local init_data
    init_data=$(echo "$resource_config" | jq -c '.initialization // []')
    
    if [[ "$init_data" == "[]" ]] || [[ "$init_data" == "{}" ]]; then
        return 0
    fi
    
    # Convert array format to object format that resources expect
    if echo "$init_data" | jq -e 'type == "array"' >/dev/null 2>&1; then
        # Group array items by type and transform for resource consumption
        local grouped_data
        grouped_data=$(echo "$init_data" | jq '
            group_by(.type) | 
            map({
                key: (.[0].type + "s"), 
                value: map(
                    # Add name field from filename and set defaults
                    . + {
                        name: (.file | split("/")[-1] | split(".")[0]),
                        enabled: (.enabled // true)
                    } | 
                    # Remove type field as resources dont need it
                    del(.type)
                )
            }) | 
            from_entries'
        )
        init_data="$grouped_data"
    fi
    
    # Let resources handle their own path resolution - just pass the scenario dir context
    init_data=$(echo "$init_data" | jq --arg dir "$scenario_dir" '. + {"_scenario_dir": $dir}')
    
    log::info "Injecting $resource_name..."
    
    if [[ "$DRY_RUN" == "yes" ]]; then
        log::info "[DRY RUN] Would inject $resource_name"
        return 0
    fi
    
    # Validate injection config first
    if ! "$manage_script" --action validate-injection --injection-config "$init_data" >/dev/null 2>&1; then
        log::warn "Configuration validation failed for $resource_name (resource may not support new interface yet)"
        # Continue anyway during migration period
    fi
    
    # Call manage script with inject action - pass JSON directly
    if "$manage_script" --action inject --injection-config "$init_data" 2>&1; then
        log::success "✓ $resource_name"
        return 0
    else
        log::error "✗ $resource_name"
        return 1
    fi
}

#######################################
# Process scenario directory
#######################################
process_scenario() {
    local scenario_dir="$1"
    local service_json="$scenario_dir/.vrooli/service.json"
    
    if [[ ! -f "$service_json" ]]; then
        log::error "No service.json: $service_json"
        return 1
    fi
    
    # Read and process service.json
    local service_config
    service_config=$(cat "$service_json")
    service_config=$(substitute_secrets "$service_config")
    
    local scenario_name
    scenario_name=$(echo "$service_config" | jq -r '.service.name // "unknown"')
    
    log::header "📦 $scenario_name"
    
    # Process all resources
    local failed=0
    for category in automation data storage ai; do
        local resources
        resources=$(echo "$service_config" | jq -r --arg cat "$category" '.resources[$cat] // {} | keys[]' 2>/dev/null || true)
        
        for resource in $resources; do
            local resource_config
            resource_config=$(echo "$service_config" | jq -c --arg cat "$category" --arg res "$resource" '.resources[$cat][$res]')
            
            if ! inject_resource "$resource" "$resource_config" "$scenario_dir"; then
                ((failed++))
            fi
        done
    done
    
    if [[ $failed -eq 0 ]]; then
        log::success "✅ Completed: $scenario_name"
        return 0
    else
        log::error "❌ Failed: $scenario_name ($failed errors)"
        return 1
    fi
}

#######################################
# Process initialization directory
#######################################
process_initialization_directory() {
    local init_dir="$1"
    
    if [[ ! -d "$init_dir" ]]; then
        log::error "Initialization directory not found: $init_dir"
        return 1
    fi
    
    log::header "📁 Processing initialization directory: $init_dir"
    
    # Look for service.json in the parent directory
    local service_json
    if [[ -f "${init_dir}/../.vrooli/service.json" ]]; then
        service_json="${init_dir}/../.vrooli/service.json"
    elif [[ -f "${init_dir}/.vrooli/service.json" ]]; then
        service_json="${init_dir}/.vrooli/service.json"
    elif [[ -f "${PWD}/.vrooli/service.json" ]]; then
        service_json="${PWD}/.vrooli/service.json"
    else
        log::error "No service.json found in expected locations"
        log::info "Expected locations:"
        log::info "  ${init_dir}/../.vrooli/service.json"
        log::info "  ${init_dir}/.vrooli/service.json"
        log::info "  ${PWD}/.vrooli/service.json"
        log::info "Injection requires proper service.json configuration"
        return 1
    fi
    
    # If we have a service.json, process it normally using the directory containing it
    local parent_dir="$(dirname "$(dirname "$service_json")")"
    process_scenario "$parent_dir"
}


#######################################
# Validate scenario
#######################################
validate_scenario() {
    local scenario_dir="$1"
    local service_json="$scenario_dir/.vrooli/service.json"
    
    if [[ ! -f "$service_json" ]]; then
        log::error "Missing service.json"
        return 1
    fi
    
    if ! jq . "$service_json" >/dev/null 2>&1; then
        log::error "Invalid JSON"
        return 1
    fi
    
    log::success "Valid scenario"
    return 0
}

#######################################
# List scenarios
#######################################
list_scenarios() {
    log::header "📋 Available Scenarios"
    
    for dir in "${DEFAULT_SCENARIOS_DIR}"/*/; do
        if [[ -d "$dir" ]] && [[ -f "$dir/.vrooli/service.json" ]]; then
            local name
            name=$(jq -r '.service.name // "unknown"' "$dir/.vrooli/service.json" 2>/dev/null || echo "unknown")
            local desc
            desc=$(jq -r '.service.description // ""' "$dir/.vrooli/service.json" 2>/dev/null || echo "")
            log::info "• $name - $desc"
        fi
    done
}

#######################################
# Main
#######################################
main() {
    case "$ACTION" in
        "inject")
            if [[ -n "$INITIALIZATION_DIR" ]]; then
                # Process initialization directory
                process_initialization_directory "$INITIALIZATION_DIR"
            elif [[ "$ALL_ACTIVE" == "yes" ]]; then
                # Process all scenarios
                for dir in "${DEFAULT_SCENARIOS_DIR}"/*/; do
                    if [[ -d "$dir" ]]; then
                        process_scenario "$dir"
                    fi
                done
            elif [[ -n "$SCENARIO_NAME" ]]; then
                # Find scenario by name
                local scenario_dir="${DEFAULT_SCENARIOS_DIR}/${SCENARIO_NAME}"
                if [[ ! -d "$scenario_dir" ]]; then
                    log::error "Scenario not found: $SCENARIO_NAME"
                    exit 1
                fi
                process_scenario "$scenario_dir"
            else
                log::error "Specify --scenario, --all-active, or provide initialization directory"
                exit 1
            fi
            ;;
        "validate")
            if [[ -n "$SCENARIO_NAME" ]]; then
                local scenario_dir="${DEFAULT_SCENARIOS_DIR}/${SCENARIO_NAME}"
                if [[ ! -d "$scenario_dir" ]]; then
                    log::error "Scenario not found: $SCENARIO_NAME"
                    exit 1
                fi
                validate_scenario "$scenario_dir"
            else
                log::error "Specify --scenario"
                exit 1
            fi
            ;;
        "list-scenarios")
            list_scenarios
            ;;
        *)
            log::error "Unknown action: $ACTION"
            exit 1
            ;;
    esac
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi