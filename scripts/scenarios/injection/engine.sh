#!/usr/bin/env bash
set -euo pipefail

# Resource Data Injection Engine
# This script orchestrates data injection into resources based on scenario configurations
# It delegates resource-specific injection logic to co-located adapters

DESCRIPTION="Orchestrates data injection into resources for scenario deployment"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PROJECT_ROOT=$(cd "${SCRIPT_DIR}/../../.." && pwd)
RESOURCES_DIR="${PROJECT_ROOT}/scripts/resources"

# Source common utilities if available
if [[ -f "${RESOURCES_DIR}/common.sh" ]]; then
    # shellcheck disable=SC1091
    source "${RESOURCES_DIR}/common.sh"
fi

# Source argument helpers if available
if [[ -f "${PROJECT_ROOT}/scripts/helpers/utils/args.sh" ]]; then
    # shellcheck disable=SC1091
    source "${PROJECT_ROOT}/scripts/helpers/utils/args.sh"
fi

# Default scenarios directory path
readonly DEFAULT_SCENARIOS_DIR="${PROJECT_ROOT}/scripts/scenarios/core"

# Operation tracking for rollback support
declare -a INJECTION_ROLLBACK_ACTIONS=()
declare -g INJECTION_OPERATION_ID=""

#######################################
# Resolve secret using 3-layer fallback strategy
# Arguments:
#   $1 - secret key name
#   $2 - vault path (optional, defaults to "vrooli")
# Returns:
#   0 - success, secret printed to stdout
#   1 - secret not found in any layer
#######################################
resolve_secret() {
    local key="$1"
    local vault_path="${2:-vrooli}"
    local vault_manager="$PROJECT_ROOT/scripts/resources/storage/vault/manage.sh"
    
    # Layer 1: Vault (if available and healthy)
    if [[ -x "$vault_manager" ]]; then
        local vault_secret
        if vault_secret=$("$vault_manager" --action get-secret --path "$vault_path" --key "$key" --format raw 2>/dev/null); then
            if [[ -n "$vault_secret" ]]; then
                echo "$vault_secret"
                return 0
            fi
        fi
    fi
    
    # Layer 2: .vrooli/secrets.json
    if [[ -f ".vrooli/secrets.json" ]]; then
        local json_secret
        if json_secret=$(jq -r ".$key // empty" .vrooli/secrets.json 2>/dev/null); then
            if [[ -n "$json_secret" && "$json_secret" != "null" ]]; then
                echo "$json_secret"
                return 0
            fi
        fi
    fi
    
    # Layer 3: Environment variables
    local env_secret
    if env_secret=$(printenv "$key" 2>/dev/null); then
        if [[ -n "$env_secret" ]]; then
            echo "$env_secret"
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Substitute secrets in JSON configuration
# Arguments:
#   $1 - JSON string with {{SECRET_NAME}} placeholders
# Returns:
#   JSON with secrets resolved or error
#######################################
substitute_secrets_in_json() {
    local json_input="$1"
    local json_output="$json_input"
    
    # Find all {{SECRET_NAME}} patterns
    local secret_patterns
    mapfile -t secret_patterns < <(echo "$json_input" | grep -oP '\{\{[^}]+\}\}' | sort -u)
    
    for pattern in "${secret_patterns[@]}"; do
        local secret_name="${pattern//[{}]/}" # Remove {{ and }}
        local secret_value
        
        if secret_value=$(resolve_secret "$secret_name"); then
            # Escape the secret value for JSON
            local escaped_value
            escaped_value=$(echo "$secret_value" | jq -R .)
            # Remove the quotes added by jq -R
            escaped_value="${escaped_value#\"}"
            escaped_value="${escaped_value%\"}"
            
            # Replace the pattern in JSON
            json_output="${json_output//$pattern/$escaped_value}"
        else
            log::warn "Secret not found: $secret_name (pattern: $pattern)"
            # Keep the placeholder - don't fail the entire initialization
        fi
    done
    
    echo "$json_output"
}

#######################################
# Process service configuration with secret substitution
# Arguments:
#   $1 - scenario directory path
# Returns:
#   Resolved service configuration JSON on stdout
#######################################
injection::process_service_config() {
    local scenario_dir="$1"
    local service_config="$scenario_dir/.vrooli/service.json"
    local secrets_file="$scenario_dir/.vrooli/secrets.json"
    
    if [[ ! -f "$service_config" ]]; then
        log::error "No service.json found at: $service_config"
        return 1
    fi
    
    # Read and substitute secrets in service config
    local raw_config resolved_config
    raw_config=$(cat "$service_config")
    resolved_config=$(substitute_secrets_in_json "$raw_config")
    
    echo "$resolved_config"
}

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
        --name "scenario-dir" \
        --flag "s" \
        --desc "Path to scenario directory containing service.json" \
        --type "value" \
        --default ""
    
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
    export SCENARIO_DIR=$(args::get "scenario-dir")
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
    -a, --action ACTION        Action to perform (default: inject)
                              Options: inject, validate, status, rollback, list-scenarios
    -s, --scenario-dir PATH    Path to scenario directory containing service.json
        --all-active          Process all scenarios in /scripts/scenarios/core/ (yes/no, default: no)
    -f, --force               Force injection even if validation fails (yes/no, default: no)
        --dry-run             Show what would be injected without doing it (yes/no, default: no)
        --yes                 Auto-confirm prompts
    -h, --help                Show this help message

EXAMPLES:
    # Inject specific scenario from directory
    $0 --action inject --scenario-dir /path/to/scenario/campaign-content-studio
    
    # Inject all scenarios in core directory
    $0 --action inject --all-active yes
    
    # Validate scenario service.json configuration
    $0 --action validate --scenario-dir /path/to/scenario/ecommerce-store
    
    # List available scenarios in core directory
    $0 --action list-scenarios
    
    # Dry run to see what would be injected
    $0 --action inject --scenario-dir /path/to/scenario/test-scenario --dry-run yes

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
# Validate scenario directory and service.json
# Arguments:
#   $1 - scenario directory path
# Returns:
#   0 if valid, 1 if invalid
#######################################
injection::validate_scenario_dir() {
    local scenario_dir="$1"
    
    if [[ ! -d "$scenario_dir" ]]; then
        log::error "Scenario directory not found: $scenario_dir"
        return 1
    fi
    
    local service_json="$scenario_dir/service.json"
    if [[ ! -f "$service_json" ]]; then
        log::error "service.json not found in scenario directory: $scenario_dir"
        return 1
    fi
    
    if ! system::is_command "jq"; then
        log::error "jq command not available for service.json validation"
        return 1
    fi
    
    # Basic JSON validation
    if ! jq . "$service_json" >/dev/null 2>&1; then
        log::error "Invalid JSON in service.json: $service_json"
        return 1
    fi
    
    # Check required structure
    local has_service
    has_service=$(jq -e '.service' "$service_json" >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_service" != "true" ]]; then
        log::error "service.json missing 'service' section: $service_json"
        return 1
    fi
    
    log::success "Scenario directory and service.json are valid: $scenario_dir"
    return 0
}

#######################################
# Get scenario service.json configuration
# Arguments:
#   $1 - scenario directory path
# Outputs:
#   Full JSON configuration from service.json
#######################################
injection::get_service_json() {
    local scenario_dir="$1"
    local service_json="$scenario_dir/service.json"
    
    if ! injection::validate_scenario_dir "$scenario_dir"; then
        return 1
    fi
    
    jq . "$service_json"
}

#######################################
# Get scenario metadata from service.json
# Arguments:
#   $1 - scenario directory path
# Outputs:
#   JSON object with name, description, version
#######################################
injection::get_scenario_metadata() {
    local scenario_dir="$1"
    local service_json="$scenario_dir/service.json"
    
    # Validate quietly (without log output)
    if [[ ! -d "$scenario_dir" ]] || [[ ! -f "$service_json" ]] || ! jq . "$service_json" >/dev/null 2>&1; then
        return 1
    fi
    
    jq '{
        name: .service.name,
        displayName: .service.displayName,
        description: .service.description,
        version: .service.version
    }' "$service_json"
}

#######################################
# Get deployment phases from service.json
# Arguments:
#   $1 - scenario directory path
# Outputs:
#   JSON array of deployment phases
#######################################
injection::get_deployment_phases() {
    local scenario_dir="$1"
    local service_json="$scenario_dir/service.json"
    
    if ! injection::validate_scenario_dir "$scenario_dir"; then
        return 1
    fi
    
    # Extract deployment phases, return empty array if not found
    jq '.deployment.initialization.phases // []' "$service_json"
}

#######################################
# Get resource configurations from service.json
# Arguments:
#   $1 - scenario directory path
# Outputs:
#   JSON object with resource configurations
#######################################
injection::get_resource_configs() {
    local scenario_dir="$1"
    local service_json="$scenario_dir/service.json"
    
    if ! injection::validate_scenario_dir "$scenario_dir"; then
        return 1
    fi
    
    # Extract resources section, return empty object if not found
    jq '.resources // {}' "$service_json"
}

#######################################
# Execute a single deployment task
# Arguments:
#   $1 - scenario directory path
#   $2 - task configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::execute_task() {
    local scenario_dir="$1"
    local task_config="$2"
    
    local task_name
    task_name=$(echo "$task_config" | jq -r '.name')
    local task_type
    task_type=$(echo "$task_config" | jq -r '.type')
    
    log::info "🔄 Executing task: $task_name (type: $task_type)"
    
    if [[ "$DRY_RUN" == "yes" ]]; then
        log::info "[DRY RUN] Would execute task: $task_name"
        log::debug "[DRY RUN] Task config: $task_config"
        return 0
    fi
    
    case "$task_type" in
        "script")
            injection::execute_script_task "$scenario_dir" "$task_config"
            ;;
        "sql")
            injection::execute_sql_task "$scenario_dir" "$task_config"  
            ;;
        "resource-inject")
            injection::execute_resource_inject_task "$scenario_dir" "$task_config"
            ;;
        "config")
            injection::execute_config_task "$scenario_dir" "$task_config"
            ;;
        "healthcheck")
            injection::execute_healthcheck_task "$scenario_dir" "$task_config"
            ;;
        *)
            log::error "Unknown task type: $task_type"
            return 1
            ;;
    esac
}

#######################################
# Execute a script task
# Arguments:
#   $1 - scenario directory path
#   $2 - task configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::execute_script_task() {
    local scenario_dir="$1"
    local task_config="$2"
    
    local script_path
    script_path=$(echo "$task_config" | jq -r '.script')
    local timeout
    timeout=$(echo "$task_config" | jq -r '.timeout // "30s"')
    
    # Make script path relative to scenario directory
    local full_script_path="$scenario_dir/$script_path"
    
    if [[ ! -f "$full_script_path" ]]; then
        log::error "Script not found: $full_script_path"
        return 1
    fi
    
    if [[ ! -x "$full_script_path" ]]; then
        log::error "Script not executable: $full_script_path"
        return 1
    fi
    
    log::debug "Executing script: $full_script_path (timeout: $timeout)"
    
    # Execute script with timeout
    if timeout "$timeout" "$full_script_path"; then
        log::success "Script executed successfully: $script_path"
        return 0
    else
        log::error "Script execution failed: $script_path"
        return 1
    fi
}

#######################################
# Execute a SQL task
# Arguments:
#   $1 - scenario directory path
#   $2 - task configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::execute_sql_task() {
    local scenario_dir="$1"
    local task_config="$2"
    
    local target
    target=$(echo "$task_config" | jq -r '.target')
    local file_path
    file_path=$(echo "$task_config" | jq -r '.file // empty')
    local query
    query=$(echo "$task_config" | jq -r '.query // empty')
    
    # Make file path relative to scenario directory if provided
    if [[ -n "$file_path" && "$file_path" != "null" ]]; then
        local full_file_path="$scenario_dir/$file_path"
        if [[ ! -f "$full_file_path" ]]; then
            log::error "SQL file not found: $full_file_path"
            return 1
        fi
        query=$(cat "$full_file_path")
    fi
    
    if [[ -z "$query" || "$query" == "null" ]]; then
        log::error "No SQL query or file provided for task"
        return 1
    fi
    
    log::debug "Executing SQL task for target: $target"
    
    # TODO: Implement SQL execution based on target
    # For now, just log what would be executed
    log::info "Would execute SQL against $target: ${query:0:100}..."
    
    return 0
}

#######################################
# Execute a resource injection task
# Arguments:
#   $1 - scenario directory path
#   $2 - task configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::execute_resource_inject_task() {
    local scenario_dir="$1"
    local task_config="$2"
    
    local target
    target=$(echo "$task_config" | jq -r '.target')
    local config_path
    config_path=$(echo "$task_config" | jq -r '.config')
    
    # Parse target (e.g., "automation.n8n" -> resource="n8n", category="automation")
    local resource_category
    resource_category=$(echo "$target" | cut -d'.' -f1)
    local resource_name
    resource_name=$(echo "$target" | cut -d'.' -f2)
    
    log::debug "Resource inject task: category=$resource_category, resource=$resource_name, config=$config_path"
    
    # Make config path relative to scenario directory
    local full_config_path="$scenario_dir/$config_path"
    
    if [[ ! -e "$full_config_path" ]]; then
        log::error "Resource config path not found: $full_config_path"
        return 1
    fi
    
    # Get resource configuration from service.json
    local resource_config
    resource_config=$(jq -c ".resources.$resource_category.$resource_name // {}" "$scenario_dir/service.json")
    
    if [[ "$resource_config" == "{}" ]]; then
        log::warn "No resource configuration found for $target in service.json"
    fi
    
    # Call the resource injection mapper
    injection::inject_resource_from_service_json "$resource_name" "$resource_config" "$scenario_dir"
}

#######################################
# Execute a config task
# Arguments:
#   $1 - scenario directory path
#   $2 - task configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::execute_config_task() {
    local scenario_dir="$1"
    local task_config="$2"
    
    local file_path
    file_path=$(echo "$task_config" | jq -r '.file')
    
    # Make file path relative to scenario directory
    local full_file_path="$scenario_dir/$file_path"
    
    if [[ ! -f "$full_file_path" ]]; then
        log::error "Config file not found: $full_file_path"
        return 1
    fi
    
    log::debug "Loading config from: $full_file_path"
    
    # TODO: Implement config loading based on requirements
    # For now, just validate the JSON
    if jq . "$full_file_path" >/dev/null 2>&1; then
        log::success "Config file is valid JSON: $file_path"
        return 0
    else
        log::error "Invalid JSON in config file: $file_path"
        return 1
    fi
}

#######################################
# Execute a healthcheck task
# Arguments:
#   $1 - scenario directory path
#   $2 - task configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::execute_healthcheck_task() {
    local scenario_dir="$1"
    local task_config="$2"
    
    local target
    target=$(echo "$task_config" | jq -r '.target')
    local timeout
    timeout=$(echo "$task_config" | jq -r '.timeout // "30s"')
    local retries
    retries=$(echo "$task_config" | jq -r '.retries // 3')
    
    log::debug "Healthcheck: target=$target, timeout=$timeout, retries=$retries"
    
    # Convert timeout to seconds for retry loop
    local timeout_seconds
    timeout_seconds=$(echo "$timeout" | sed 's/s$//')
    
    for ((i=1; i<=retries; i++)); do
        log::info "Healthcheck attempt $i/$retries: $target"
        
        if curl --silent --fail --max-time "$timeout_seconds" "$target" >/dev/null 2>&1; then
            log::success "Healthcheck passed: $target"
            return 0
        fi
        
        if [[ $i -lt $retries ]]; then
            log::warn "Healthcheck failed, retrying in 5 seconds..."
            sleep 5
        fi
    done
    
    log::error "Healthcheck failed after $retries attempts: $target"
    return 1
}

#######################################
# Execute a deployment phase
# Arguments:
#   $1 - scenario directory path
#   $2 - phase configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::execute_phase() {
    local scenario_dir="$1"
    local phase_config="$2"
    
    local phase_name
    phase_name=$(echo "$phase_config" | jq -r '.name')
    local parallel
    parallel=$(echo "$phase_config" | jq -r '.parallel // false')
    
    log::header "🚀 Executing Phase: $phase_name"
    
    # Get tasks array
    local tasks
    tasks=$(echo "$phase_config" | jq -c '.tasks[]?')
    
    if [[ -z "$tasks" ]]; then
        log::info "No tasks in phase: $phase_name"
        return 0
    fi
    
    local failed_tasks=()
    
    if [[ "$parallel" == "true" ]]; then
        log::info "Executing tasks in parallel"
        
        # Execute tasks in parallel
        local pids=()
        while IFS= read -r task; do
            injection::execute_task "$scenario_dir" "$task" &
            pids+=($!)
        done <<< "$tasks"
        
        # Wait for all tasks to complete
        for pid in "${pids[@]}"; do
            if ! wait "$pid"; then
                failed_tasks+=("task_$pid")
            fi
        done
    else
        log::info "Executing tasks sequentially"
        
        # Execute tasks sequentially
        while IFS= read -r task; do
            local task_name
            task_name=$(echo "$task" | jq -r '.name')
            
            if ! injection::execute_task "$scenario_dir" "$task"; then
                failed_tasks+=("$task_name")
                # Stop on first failure for sequential execution
                break
            fi
        done <<< "$tasks"
    fi
    
    # Check results
    if [[ ${#failed_tasks[@]} -eq 0 ]]; then
        log::success "✅ Phase completed successfully: $phase_name"
        return 0
    else
        log::error "❌ Phase failed with failed tasks: ${failed_tasks[*]}"
        return 1
    fi
}

#######################################
# Execute all deployment phases for a scenario
# Arguments:
#   $1 - scenario directory path
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::execute_deployment_phases() {
    local scenario_dir="$1"
    
    log::header "🚀 Executing Deployment Phases"
    
    local phases
    if ! phases=$(injection::get_deployment_phases "$scenario_dir"); then
        log::error "Failed to get deployment phases"
        return 1
    fi
    
    if [[ "$phases" == "[]" ]]; then
        log::info "No deployment phases defined for scenario"
        return 0
    fi
    
    local failed_phases=()
    
    # Check if there are any phases to execute
    local phase_count
    if [[ "$phases" == "[]" ]]; then
        phase_count=0
    else
        phase_count=$(echo "$phases" | jq 'length')
    fi
    
    if [[ "$phase_count" -gt 0 ]]; then
        # Execute each phase sequentially
        while IFS= read -r phase; do
            if [[ -n "$phase" ]]; then
                local phase_name
                phase_name=$(echo "$phase" | jq -r '.name')
                
                if ! injection::execute_phase "$scenario_dir" "$phase"; then
                    failed_phases+=("$phase_name")
                    # Stop on first phase failure
                    break
                fi
            fi
        done <<< "$(echo "$phases" | jq -c '.[]')"
    fi
    
    # Check results
    if [[ ${#failed_phases[@]} -eq 0 ]]; then
        log::success "✅ All deployment phases completed successfully"
        return 0
    else
        log::error "❌ Deployment failed at phases: ${failed_phases[*]}"
        return 1
    fi
}

#######################################
# Map service.json resource config to adapter format
# Arguments:
#   $1 - resource name (e.g., "n8n", "postgres")
#   $2 - resource configuration JSON from service.json
#   $3 - scenario directory path
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::inject_resource_from_service_json() {
    local resource_name="$1"
    local resource_config="$2"
    local scenario_dir="$3"
    
    log::debug "Mapping service.json resource config for: $resource_name"
    
    # Check if resource is enabled
    local enabled
    enabled=$(echo "$resource_config" | jq -r '.enabled // false')
    
    if [[ "$enabled" != "true" ]]; then
        log::info "Resource '$resource_name' is disabled, skipping injection"
        return 0
    fi
    
    # Convert service.json format to adapter format
    local adapter_config
    case "$resource_name" in
        "n8n")
            adapter_config=$(injection::map_n8n_config "$resource_config" "$scenario_dir")
            ;;
        "windmill")
            adapter_config=$(injection::map_windmill_config "$resource_config" "$scenario_dir")
            ;;
        "postgres")
            adapter_config=$(injection::map_postgres_config "$resource_config" "$scenario_dir")
            ;;
        "node-red")
            adapter_config=$(injection::map_node_red_config "$resource_config" "$scenario_dir")
            ;;
        "minio")
            adapter_config=$(injection::map_minio_config "$resource_config" "$scenario_dir")
            ;;
        "qdrant")
            adapter_config=$(injection::map_qdrant_config "$resource_config" "$scenario_dir")
            ;;
        *)
            log::warn "No mapper implemented for resource: $resource_name"
            # Use generic mapping as fallback
            adapter_config=$(injection::map_generic_config "$resource_config" "$scenario_dir")
            ;;
    esac
    
    if [[ -z "$adapter_config" || "$adapter_config" == "null" ]]; then
        log::warn "No adapter configuration generated for resource: $resource_name"
        return 0
    fi
    
    # Call the existing resource injection function with mapped config
    injection::inject_resource_data "$resource_name" "$adapter_config"
}

#######################################
# Map n8n resource configuration
# Arguments:
#   $1 - n8n resource config JSON
#   $2 - scenario directory path
# Outputs:
#   Mapped configuration JSON for n8n adapter
#######################################
injection::map_n8n_config() {
    local resource_config="$1"
    local scenario_dir="$2"
    
    local workflows
    workflows=$(echo "$resource_config" | jq '.initialization.workflows // []')
    
    local credentials
    credentials=$(echo "$resource_config" | jq '.initialization.credentials // []')
    
    # Convert file paths to absolute paths
    local mapped_workflows
    mapped_workflows=$(echo "$workflows" | jq --arg scenario_dir "$scenario_dir" '
        map(if .file then .file = ($scenario_dir + "/" + .file) else . end)
    ')
    
    # Build adapter configuration
    jq -n \
        --argjson workflows "$mapped_workflows" \
        --argjson credentials "$credentials" \
        '{
            workflows: $workflows,
            credentials: $credentials
        }'
}

#######################################
# Map windmill resource configuration
# Arguments:
#   $1 - windmill resource config JSON
#   $2 - scenario directory path
# Outputs:
#   Mapped configuration JSON for windmill adapter
#######################################
injection::map_windmill_config() {
    local resource_config="$1"
    local scenario_dir="$2"
    
    local apps
    apps=$(echo "$resource_config" | jq '.initialization.apps // []')
    
    local scripts
    scripts=$(echo "$resource_config" | jq '.initialization.scripts // []')
    
    # Convert file paths to absolute paths
    local mapped_apps
    mapped_apps=$(echo "$apps" | jq --arg scenario_dir "$scenario_dir" '
        map(if .file then .file = ($scenario_dir + "/" + .file) else . end)
    ')
    
    local mapped_scripts
    mapped_scripts=$(echo "$scripts" | jq --arg scenario_dir "$scenario_dir" '
        map(if .file then .file = ($scenario_dir + "/" + .file) else . end)
    ')
    
    # Build adapter configuration
    jq -n \
        --argjson apps "$mapped_apps" \
        --argjson scripts "$mapped_scripts" \
        '{
            apps: $apps,
            scripts: $scripts
        }'
}

#######################################
# Map postgres resource configuration
# Arguments:
#   $1 - postgres resource config JSON
#   $2 - scenario directory path
# Outputs:
#   Mapped configuration JSON for postgres adapter
#######################################
injection::map_postgres_config() {
    local resource_config="$1"
    local scenario_dir="$2"
    
    local data
    data=$(echo "$resource_config" | jq '.initialization.data // []')
    
    # Convert file paths to absolute paths and categorize by type
    local mapped_data
    mapped_data=$(echo "$data" | jq --arg scenario_dir "$scenario_dir" '
        map(if .file then .file = ($scenario_dir + "/" + .file) else . end)
    ')
    
    # Build adapter configuration
    jq -n \
        --argjson data "$mapped_data" \
        '{
            data: $data
        }'
}

#######################################
# Map node-red resource configuration
# Arguments:
#   $1 - node-red resource config JSON
#   $2 - scenario directory path
# Outputs:
#   Mapped configuration JSON for node-red adapter
#######################################
injection::map_node_red_config() {
    local resource_config="$1"
    local scenario_dir="$2"
    
    local flows
    flows=$(echo "$resource_config" | jq '.initialization.flows // []')
    
    # Convert file paths to absolute paths
    local mapped_flows
    mapped_flows=$(echo "$flows" | jq --arg scenario_dir "$scenario_dir" '
        map(if .file then .file = ($scenario_dir + "/" + .file) else . end)
    ')
    
    # Build adapter configuration
    jq -n \
        --argjson flows "$mapped_flows" \
        '{
            flows: $flows
        }'
}

#######################################
# Map minio resource configuration
# Arguments:
#   $1 - minio resource config JSON
#   $2 - scenario directory path
# Outputs:
#   Mapped configuration JSON for minio adapter
#######################################
injection::map_minio_config() {
    local resource_config="$1"
    local scenario_dir="$2"
    
    local buckets
    buckets=$(echo "$resource_config" | jq '.initialization.buckets // []')
    
    local files
    files=$(echo "$resource_config" | jq '.initialization.files // []')
    
    # Convert file paths to absolute paths for file uploads
    local mapped_files
    mapped_files=$(echo "$files" | jq --arg scenario_dir "$scenario_dir" '
        map(if .source then .source = ($scenario_dir + "/" + .source) else . end)
    ')
    
    # Build adapter configuration
    jq -n \
        --argjson buckets "$buckets" \
        --argjson files "$mapped_files" \
        '{
            buckets: $buckets,
            files: $files
        }'
}

#######################################
# Map qdrant resource configuration
# Arguments:
#   $1 - qdrant resource config JSON
#   $2 - scenario directory path
# Outputs:
#   Mapped configuration JSON for qdrant adapter
#######################################
injection::map_qdrant_config() {
    local resource_config="$1"
    local scenario_dir="$2"
    
    local collections
    collections=$(echo "$resource_config" | jq '.initialization.collections // []')
    
    local vectors
    vectors=$(echo "$resource_config" | jq '.initialization.vectors // []')
    
    # Convert file paths to absolute paths for vector data
    local mapped_vectors
    mapped_vectors=$(echo "$vectors" | jq --arg scenario_dir "$scenario_dir" '
        map(if .file then .file = ($scenario_dir + "/" + .file) else . end)
    ')
    
    # Build adapter configuration
    jq -n \
        --argjson collections "$collections" \
        --argjson vectors "$mapped_vectors" \
        '{
            collections: $collections,
            vectors: $vectors
        }'
}

#######################################
# Map generic resource configuration (fallback)
# Arguments:
#   $1 - resource config JSON
#   $2 - scenario directory path
# Outputs:
#   Basic mapped configuration JSON
#######################################
injection::map_generic_config() {
    local resource_config="$1"
    local scenario_dir="$2"
    
    # Extract initialization section and convert relative paths
    local initialization
    initialization=$(echo "$resource_config" | jq '.initialization // {}')
    
    # Simple mapping that attempts to convert file paths
    echo "$initialization" | jq --arg scenario_dir "$scenario_dir" '
        def convert_paths:
            if type == "object" then
                with_entries(
                    if .key == "file" and (.value | type) == "string" then
                        .value = ($scenario_dir + "/" + .value)
                    else
                        .value |= convert_paths
                    end
                )
            elif type == "array" then
                map(convert_paths)
            else
                .
            end;
        convert_paths
    '
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
# Inject scenario from directory
# Arguments:
#   $1 - scenario directory path
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::inject_scenario_from_dir() {
    local scenario_dir="$1"
    
    # Validate scenario directory
    if ! injection::validate_scenario_dir "$scenario_dir"; then
        return 1
    fi
    
    # Get scenario name directly from service.json
    local service_json="$scenario_dir/service.json"
    local scenario_name
    scenario_name=$(jq -r '.service.name' "$service_json")
    
    log::header "🚀 Injecting Scenario: $scenario_name"
    log::info "Directory: $scenario_dir"
    
    # Start rollback context for this scenario
    injection::start_rollback_context "inject_scenario_$scenario_name"
    
    # Process service config with secret substitution
    local service_config_resolved
    if ! service_config_resolved=$(injection::process_service_config "$scenario_dir"); then
        log::error "Failed to process service configuration for scenario: $scenario_name"
        injection::execute_rollback
        return 1
    fi
    
    # Execute deployment phases with resolved config
    if ! injection::execute_deployment_phases "$scenario_dir" "$service_config_resolved"; then
        log::error "Failed to execute deployment phases for scenario: $scenario_name"
        injection::execute_rollback
        return 1
    fi
    
    log::success "✅ Scenario injection completed successfully: $scenario_name"
    return 0
}

#######################################
# Inject all scenarios in core directory
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::inject_all_core_scenarios() {
    log::header "📋 Injecting All Core Scenarios"
    
    if [[ ! -d "$DEFAULT_SCENARIOS_DIR" ]]; then
        log::error "Core scenarios directory not found: $DEFAULT_SCENARIOS_DIR"
        return 1
    fi
    
    # Find all scenario directories (those containing service.json)
    local scenario_dirs=()
    while IFS= read -r -d '' dir; do
        scenario_dirs+=("$dir")
    done < <(find "$DEFAULT_SCENARIOS_DIR" -name "service.json" -type f -exec dirname {} \; -print0 | sort -z)
    
    if [[ ${#scenario_dirs[@]} -eq 0 ]]; then
        log::info "No scenarios found in core directory"
        return 0
    fi
    
    log::info "Found ${#scenario_dirs[@]} scenarios in core directory"
    
    # Inject each scenario
    local failed_scenarios=()
    for scenario_dir in "${scenario_dirs[@]}"; do
        local scenario_name
        scenario_name=$(basename "$scenario_dir")
        
        log::info "Processing scenario: $scenario_name"
        
        if ! injection::inject_scenario_from_dir "$scenario_dir"; then
            failed_scenarios+=("$scenario_name")
        fi
    done
    
    # Report results
    if [[ ${#failed_scenarios[@]} -eq 0 ]]; then
        log::success "✅ All core scenarios injected successfully"
        return 0
    else
        log::error "❌ Failed to inject scenarios: ${failed_scenarios[*]}"
        return 1
    fi
}

#######################################
# List available scenarios in core directory
# Returns:
#   0 if successful, 1 if failed
#######################################
injection::list_core_scenarios() {
    log::header "📋 Available Core Scenarios"
    
    if [[ ! -d "$DEFAULT_SCENARIOS_DIR" ]]; then
        log::error "Core scenarios directory not found: $DEFAULT_SCENARIOS_DIR"
        return 1
    fi
    
    # Find all scenario directories and get their metadata
    local found_scenarios=false
    while IFS= read -r -d '' service_json; do
        found_scenarios=true
        local scenario_dir
        scenario_dir=$(dirname "$service_json")
        local scenario_name
        scenario_name=$(basename "$scenario_dir")
        
        # Get metadata directly from service.json
        local service_json="$scenario_dir/service.json"
        if [[ -f "$service_json" ]] && jq . "$service_json" >/dev/null 2>&1; then
            local display_name
            display_name=$(jq -r '.service.displayName // .service.name' "$service_json")
            local description
            description=$(jq -r '.service.description // "No description"' "$service_json")
            local version
            version=$(jq -r '.service.version // "unknown"' "$service_json")
            
            log::info "• $scenario_name ($display_name v$version)"
            log::info "  $description"
            log::info "  Path: $scenario_dir"
            log::info ""
        else
            log::warn "• $scenario_name (invalid service.json)"
            log::info "  Path: $scenario_dir"
            log::info ""
        fi
    done < <(find "$DEFAULT_SCENARIOS_DIR" -name "service.json" -type f -print0 | sort -z)
    
    if [[ "$found_scenarios" != "true" ]]; then
        log::info "No scenarios found in core directory"
        return 0
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
injection::main() {
    injection::parse_arguments "$@"
    
    case "$ACTION" in
        "inject")
            if [[ "$ALL_ACTIVE" == "yes" ]]; then
                injection::inject_all_core_scenarios
            elif [[ -n "$SCENARIO_DIR" ]]; then
                injection::inject_scenario_from_dir "$SCENARIO_DIR"
            else
                log::error "Either --scenario-dir or --all-active must be specified for injection"
                injection::usage
                exit 1
            fi
            ;;
        "validate")
            if [[ -n "$SCENARIO_DIR" ]]; then
                if injection::validate_scenario_dir "$SCENARIO_DIR"; then
                    log::success "Scenario directory is valid: $SCENARIO_DIR"
                else
                    log::error "Scenario directory validation failed: $SCENARIO_DIR"
                    exit 1
                fi
            else
                log::error "--scenario-dir must be specified for validation"
                injection::usage
                exit 1
            fi
            ;;
        "list-scenarios")
            injection::list_core_scenarios
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