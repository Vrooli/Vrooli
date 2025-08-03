#!/usr/bin/env bash
set -euo pipefail

# Resource Data Injection Engine
# This script orchestrates data injection into resources based on scenario configurations
# It delegates resource-specific injection logic to co-located adapters

DESCRIPTION="Orchestrates data injection into resources for scenario deployment"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/.."

# Source common utilities
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Default scenarios configuration path
readonly DEFAULT_SCENARIOS_CONFIG="${HOME}/.vrooli/scenarios.json"

# Operation tracking for rollback support
declare -a INJECTION_ROLLBACK_ACTIONS=()
declare -g INJECTION_OPERATION_ID=""

#######################################
# Parse command line arguments
#######################################
injection::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "inject|validate|status|rollback|list-scenarios" \
        --default "inject"
    
    args::register \
        --name "scenario" \
        --flag "s" \
        --desc "Scenario name to inject" \
        --type "value" \
        --default ""
    
    args::register \
        --name "config-file" \
        --flag "c" \
        --desc "Path to scenarios configuration file" \
        --type "value" \
        --default "$DEFAULT_SCENARIOS_CONFIG"
    
    args::register \
        --name "all-active" \
        --desc "Process all active scenarios" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force injection even if validation fails" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "dry-run" \
        --desc "Show what would be injected without actually doing it" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    if args::is_asking_for_help "$@"; then
        injection::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export SCENARIO_NAME=$(args::get "scenario")
    export CONFIG_FILE=$(args::get "config-file")
    export ALL_ACTIVE=$(args::get "all-active")
    export FORCE=$(args::get "force")
    export DRY_RUN=$(args::get "dry-run")
}

#######################################
# Display usage information
#######################################
injection::usage() {
    cat << EOF
Resource Data Injection Engine

USAGE:
    $0 [OPTIONS]

DESCRIPTION:
    Orchestrates data injection into resources based on scenario configurations.
    Delegates resource-specific logic to co-located injection adapters.

OPTIONS:
    -a, --action ACTION     Action to perform (default: inject)
                           Options: inject, validate, status, rollback, list-scenarios
    -s, --scenario NAME     Scenario name to inject
    -c, --config-file PATH  Path to scenarios.json file (default: ~/.vrooli/scenarios.json)
        --all-active        Process all active scenarios (yes/no, default: no)
    -f, --force            Force injection even if validation fails (yes/no, default: no)
        --dry-run          Show what would be injected without doing it (yes/no, default: no)
        --yes              Auto-confirm prompts
    -h, --help             Show this help message

EXAMPLES:
    # Inject specific scenario
    $0 --action inject --scenario vrooli-core
    
    # Inject all active scenarios
    $0 --action inject --all-active yes
    
    # Validate scenario configuration
    $0 --action validate --scenario ecommerce-store
    
    # List available scenarios
    $0 --action list-scenarios
    
    # Dry run to see what would be injected
    $0 --action inject --scenario test-scenario --dry-run yes

EOF
}

#######################################
# Start injection rollback context
# Arguments:
#   $1 - operation description
#######################################
injection::start_rollback_context() {
    local operation="$1"
    INJECTION_OPERATION_ID="${operation}_$(date +%s)"
    INJECTION_ROLLBACK_ACTIONS=()
    log::debug "Started injection rollback context: $INJECTION_OPERATION_ID"
}

#######################################
# Add rollback action for injection
# Arguments:
#   $1 - description
#   $2 - rollback command
#   $3 - priority (optional, default: 0)
#######################################
injection::add_rollback_action() {
    local description="$1"
    local command="$2"
    local priority="${3:-0}"
    
    INJECTION_ROLLBACK_ACTIONS+=("$priority|$description|$command")
    log::debug "Added injection rollback action: $description"
}

#######################################
# Execute injection rollback
#######################################
injection::execute_rollback() {
    if [[ ${#INJECTION_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No injection rollback actions to execute"
        return 0
    fi
    
    log::info "Executing injection rollback for operation: $INJECTION_OPERATION_ID"
    
    # Sort by priority (highest first)
    local sorted_actions
    IFS=$'\n' sorted_actions=($(printf '%s\n' "${INJECTION_ROLLBACK_ACTIONS[@]}" | sort -nr))
    
    local success_count=0
    local total_count=${#sorted_actions[@]}
    
    for action in "${sorted_actions[@]}"; do
        IFS='|' read -r priority description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback action completed: $description"
        else
            log::error "Rollback action failed: $description"
        fi
    done
    
    log::info "Injection rollback completed: $success_count/$total_count actions successful"
    
    # Clear rollback context
    INJECTION_ROLLBACK_ACTIONS=()
    INJECTION_OPERATION_ID=""
}

#######################################
# Validate scenarios configuration file
# Arguments:
#   $1 - config file path
# Returns:
#   0 if valid, 1 if invalid
#######################################
injection::validate_config_file() {
    local config_file="$1"
    
    if [[ ! -f "$config_file" ]]; then
        log::error "Scenarios configuration file not found: $config_file"
        return 1
    fi
    
    if ! system::is_command "jq"; then
        log::error "jq command not available for configuration validation"
        return 1
    fi
    
    # Basic JSON validation
    if ! jq . "$config_file" >/dev/null 2>&1; then
        log::error "Invalid JSON in scenarios configuration: $config_file"
        return 1
    fi
    
    # Check required structure
    local has_scenarios
    has_scenarios=$(jq -e '.scenarios' "$config_file" >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_scenarios" != "true" ]]; then
        log::error "Scenarios configuration missing 'scenarios' section"
        return 1
    fi
    
    log::success "Scenarios configuration is valid"
    return 0
}

#######################################
# Get scenario configuration
# Arguments:
#   $1 - scenario name
#   $2 - config file path
# Outputs:
#   JSON configuration for the scenario
#######################################
injection::get_scenario_config() {
    local scenario_name="$1"
    local config_file="$2"
    
    if ! jq -e ".scenarios[\"$scenario_name\"]" "$config_file" 2>/dev/null; then
        log::error "Scenario '$scenario_name' not found in $config_file"
        return 1
    fi
}

#######################################
# Find resource management script
# Arguments:
#   $1 - resource name
# Outputs:
#   Path to resource management script
#######################################
injection::find_resource_script() {
    local resource="$1"
    
    # Search through resource directories
    local script_path
    script_path=$(find "$RESOURCES_DIR" -name "manage.sh" -path "*/$resource/*" | head -1)
    
    if [[ -z "$script_path" ]]; then
        log::debug "Resource management script not found for: $resource"
        return 1
    fi
    
    echo "$script_path"
    return 0
}

#######################################
# Check if resource supports injection
# Arguments:
#   $1 - resource name
# Returns:
#   0 if injection supported, 1 otherwise
#######################################
injection::resource_supports_injection() {
    local resource="$1"
    
    local script_path
    if ! script_path=$(injection::find_resource_script "$resource"); then
        return 1
    fi
    
    local inject_script="$(dirname "$script_path")/inject.sh"
    [[ -f "$inject_script" && -x "$inject_script" ]]
}

#######################################
# Validate resource injection configuration
# Arguments:
#   $1 - resource name
#   $2 - resource configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
injection::validate_resource_config() {
    local resource="$1"
    local resource_config="$2"
    
    if ! injection::resource_supports_injection "$resource"; then
        log::warn "Resource '$resource' does not support injection"
        return 1
    fi
    
    local script_path
    script_path=$(injection::find_resource_script "$resource")
    local inject_script="$(dirname "$script_path")/inject.sh"
    
    # Call resource-specific validation
    if "$inject_script" --validate "$resource_config"; then
        log::debug "Resource '$resource' configuration is valid"
        return 0
    else
        log::error "Resource '$resource' configuration validation failed"
        return 1
    fi
}

#######################################
# Inject data for a single resource
# Arguments:
#   $1 - resource name
#   $2 - resource configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::inject_resource_data() {
    local resource="$1"
    local resource_config="$2"
    
    log::info "🔄 Injecting data for resource: $resource"
    
    if [[ "$DRY_RUN" == "yes" ]]; then
        log::info "[DRY RUN] Would inject data for $resource"
        log::debug "[DRY RUN] Configuration: $resource_config"
        return 0
    fi
    
    # Validate resource supports injection
    if ! injection::resource_supports_injection "$resource"; then
        log::warn "⚠️  Resource '$resource' does not support injection, skipping"
        return 0
    fi
    
    # Validate configuration if not forced
    if [[ "$FORCE" != "yes" ]]; then
        if ! injection::validate_resource_config "$resource" "$resource_config"; then
            log::error "❌ Resource '$resource' configuration validation failed"
            return 1
        fi
    fi
    
    # Perform injection
    local script_path
    script_path=$(injection::find_resource_script "$resource")
    local inject_script="$(dirname "$script_path")/inject.sh"
    
    # Add rollback action before attempting injection
    injection::add_rollback_action \
        "Rollback injection for $resource" \
        "$inject_script --rollback '$resource_config'" \
        10
    
    # Execute injection
    if "$inject_script" --inject "$resource_config"; then
        log::success "✅ Data injection completed for: $resource"
        return 0
    else
        log::error "❌ Data injection failed for: $resource"
        return 1
    fi
}

#######################################
# Process scenario dependencies
# Arguments:
#   $1 - scenario configuration JSON
#   $2 - config file path
#   $3 - visited scenarios (for cycle detection)
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::process_dependencies() {
    local scenario_config="$1"
    local config_file="$2"
    local visited="${3:-}"
    
    local dependencies
    dependencies=$(echo "$scenario_config" | jq -r '.dependencies[]? // empty')
    
    if [[ -z "$dependencies" ]]; then
        log::debug "No dependencies to process"
        return 0
    fi
    
    for dep in $dependencies; do
        # Check for circular dependencies
        if [[ "$visited" == *"$dep"* ]]; then
            log::error "Circular dependency detected: $dep"
            return 1
        fi
        
        log::info "📋 Processing dependency: $dep"
        
        # Recursively inject dependency
        if ! injection::inject_scenario "$dep" "$config_file" "$visited $dep"; then
            log::error "Failed to process dependency: $dep"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Inject data for a scenario
# Arguments:
#   $1 - scenario name
#   $2 - config file path
#   $3 - visited scenarios (for dependency tracking)
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::inject_scenario() {
    local scenario_name="$1"
    local config_file="$2"
    local visited="${3:-}"
    
    log::header "🚀 Injecting Scenario: $scenario_name"
    
    # Start rollback context for this scenario
    injection::start_rollback_context "inject_scenario_$scenario_name"
    
    # Get scenario configuration
    local scenario_config
    if ! scenario_config=$(injection::get_scenario_config "$scenario_name" "$config_file"); then
        return 1
    fi
    
    # Process dependencies first
    if ! injection::process_dependencies "$scenario_config" "$config_file" "$visited"; then
        log::error "Failed to process dependencies for scenario: $scenario_name"
        injection::execute_rollback
        return 1
    fi
    
    # Get resources list
    local resources
    resources=$(echo "$scenario_config" | jq -r '.resources | keys[]? // empty')
    
    if [[ -z "$resources" ]]; then
        log::info "No resources to inject for scenario: $scenario_name"
        return 0
    fi
    
    # Inject data for each resource
    local failed_resources=()
    for resource in $resources; do
        local resource_config
        resource_config=$(echo "$scenario_config" | jq -c ".resources[\"$resource\"]")
        
        if ! injection::inject_resource_data "$resource" "$resource_config"; then
            failed_resources+=("$resource")
        fi
    done
    
    # Check results
    if [[ ${#failed_resources[@]} -eq 0 ]]; then
        log::success "✅ Scenario injection completed successfully: $scenario_name"
        return 0
    else
        log::error "❌ Scenario injection failed for resources: ${failed_resources[*]}"
        injection::execute_rollback
        return 1
    fi
}

#######################################
# Inject all active scenarios
# Arguments:
#   $1 - config file path
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::inject_active_scenarios() {
    local config_file="$1"
    
    log::header "📋 Injecting All Active Scenarios"
    
    if ! injection::validate_config_file "$config_file"; then
        return 1
    fi
    
    # Get active scenarios
    local active_scenarios
    active_scenarios=$(jq -r '.active[]? // empty' "$config_file")
    
    if [[ -z "$active_scenarios" ]]; then
        log::info "No active scenarios found"
        return 0
    fi
    
    log::info "Active scenarios: $active_scenarios"
    
    # Inject each active scenario
    local failed_scenarios=()
    for scenario in $active_scenarios; do
        if ! injection::inject_scenario "$scenario" "$config_file"; then
            failed_scenarios+=("$scenario")
        fi
    done
    
    # Report results
    if [[ ${#failed_scenarios[@]} -eq 0 ]]; then
        log::success "✅ All active scenarios injected successfully"
        return 0
    else
        log::error "❌ Failed to inject scenarios: ${failed_scenarios[*]}"
        return 1
    fi
}

#######################################
# List available scenarios
# Arguments:
#   $1 - config file path
#######################################
injection::list_scenarios() {
    local config_file="$1"
    
    if ! injection::validate_config_file "$config_file"; then
        return 1
    fi
    
    log::header "📋 Available Scenarios"
    
    # Get scenario names and descriptions
    local scenarios
    scenarios=$(jq -r '.scenarios | to_entries[] | "\(.key)|\(.value.description // "No description")"' "$config_file")
    
    if [[ -z "$scenarios" ]]; then
        log::info "No scenarios found in configuration"
        return 0
    fi
    
    echo "$scenarios" | while IFS='|' read -r name description; do
        log::info "• $name: $description"
    done
    
    # Show active scenarios
    local active_scenarios
    active_scenarios=$(jq -r '.active[]? // empty' "$config_file")
    
    if [[ -n "$active_scenarios" ]]; then
        log::info ""
        log::header "🟢 Active Scenarios"
        for scenario in $active_scenarios; do
            log::info "• $scenario"
        done
    fi
}

#######################################
# Main execution function
#######################################
injection::main() {
    injection::parse_arguments "$@"
    
    case "$ACTION" in
        "inject")
            if [[ "$ALL_ACTIVE" == "yes" ]]; then
                injection::inject_active_scenarios "$CONFIG_FILE"
            elif [[ -n "$SCENARIO_NAME" ]]; then
                injection::inject_scenario "$SCENARIO_NAME" "$CONFIG_FILE"
            else
                log::error "Either --scenario or --all-active must be specified for injection"
                injection::usage
                exit 1
            fi
            ;;
        "validate")
            if [[ -n "$SCENARIO_NAME" ]]; then
                local scenario_config
                if scenario_config=$(injection::get_scenario_config "$SCENARIO_NAME" "$CONFIG_FILE"); then
                    log::success "Scenario '$SCENARIO_NAME' configuration is valid"
                else
                    log::error "Scenario '$SCENARIO_NAME' configuration is invalid"
                    exit 1
                fi
            else
                injection::validate_config_file "$CONFIG_FILE"
            fi
            ;;
        "list-scenarios")
            injection::list_scenarios "$CONFIG_FILE"
            ;;
        "rollback")
            injection::execute_rollback
            ;;
        *)
            log::error "Unknown action: $ACTION"
            injection::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    injection::main "$@"
fi