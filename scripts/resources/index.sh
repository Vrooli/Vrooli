#!/usr/bin/env bash
set -euo pipefail

# Main orchestrator for local resource setup and management
# This script provides a unified interface for managing local resources

DESCRIPTION="Manages local development resources (AI, automation, storage, agents)"

# Source var.sh first with relative path
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../lib/utils/var.sh"

# Handle Ctrl+C and other signals gracefully
trap 'echo ""; log::info "Resource installation interrupted by user. Exiting..."; exit 130' INT TERM

# Now source everything else using var_ variables
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"
# shellcheck disable=SC1091
source "${var_REPOSITORY_FILE}"

# Available resources organized by category
declare -A AVAILABLE_RESOURCES=(
    ["ai"]="ollama whisper unstructured-io"
    ["automation"]="n8n comfyui node-red windmill huginn"
    ["storage"]="minio vault qdrant questdb postgres redis"
    ["agents"]="browserless claude-code agent-s2"
    ["search"]="searxng"
)

# All available resources as a flat list
ALL_RESOURCES="ollama whisper unstructured-io n8n comfyui node-red windmill huginn minio vault qdrant questdb postgres redis browserless claude-code agent-s2 searxng"

#######################################
# Parse command line arguments
#######################################
resources::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|list|discover|test|inject|inject-all" \
        --default "install"
    
    args::register \
        --name "resources" \
        --flag "r" \
        --desc "Resources to manage (comma-separated, or 'all', 'ai-only', 'automation-only', 'storage-only', 'agents-only', 'search-only', 'enabled', 'none')" \
        --type "value" \
        --default "none"
    
    args::register \
        --name "category" \
        --flag "c" \
        --desc "Resource category filter" \
        --type "value" \
        --options "ai|automation|storage|agents|search" \
        --default ""
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if resource appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "auto-configure" \
        --desc "Automatically configure discovered resources" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "test-type" \
        --desc "Type of test to run" \
        --type "value" \
        --options "single|multi|scenarios|all" \
        --default "all"
    
    args::register \
        --name "scenario" \
        --desc "Scenario name for injection" \
        --type "value" \
        --default ""
    
    args::register \
        --name "scenarios-config" \
        --desc "Path to scenarios configuration file" \
        --type "value" \
        --default "${HOME}/.vrooli/scenarios.json"
    
    if args::is_asking_for_help "$@"; then
        resources::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export RESOURCES_INPUT=$(args::get "resources")
    export CATEGORY=$(args::get "category")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export AUTO_CONFIGURE=$(args::get "auto-configure")
    export TEST_TYPE=$(args::get "test-type")
    export SCENARIO_NAME=$(args::get "scenario")
    export SCENARIOS_CONFIG=$(args::get "scenarios-config")
}

#######################################
# Display usage information
#######################################
resources::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install --resources enabled               # Install resources marked as enabled in config"
    echo "  $0 --action install --resources ollama                # Install specific resource"
    echo "  $0 --action install --resources ai-only               # Install all AI resources"
    echo "  $0 --action install --resources search-only           # Install all search resources"
    echo "  $0 --action install --resources all                   # Install all resources"
    echo "  $0 --action status --resources ollama,n8n             # Check status of specific resources"
    echo "  $0 --action list                                      # List available resources"
    echo "  $0 --action discover                                  # Discover running resources"
    echo "  $0 --action discover --auto-configure yes             # Discover and configure resources"
    echo "  $0 --action test --resources ollama                   # Test specific resource"
    echo "  $0 --action test --resources ai-only                  # Test AI resources"
    echo "  $0 --action test --resources all                      # Test all healthy resources"
    echo "  $0 --action test --resources all --test-type single   # Run only single-resource tests"
    echo
    echo "Resource Categories:"
    for category in "${!AVAILABLE_RESOURCES[@]}"; do
        echo "  $category: ${AVAILABLE_RESOURCES[$category]}"
    done
    
    # Add repository information if available
    local repo_url
    repo_url=$(repository::get_url 2>/dev/null || echo "")
    if [[ -n "$repo_url" ]]; then
        echo
        echo "Documentation: ${repo_url}/tree/main/scripts/resources"
        echo "Issues: ${repo_url}/issues"
    fi
}

#######################################
# Replace repository placeholders in scenario templates
# Globals:
#   None
# Arguments:
#   $1 - template file path
# Returns:
#   0 on success, 1 on failure
#######################################
resources::update_scenario_template() {
    local template_file="$1"
    
    if [[ ! -f "$template_file" ]]; then
        log::error "Template file not found: $template_file"
        return 1
    fi
    
    # Get repository URL from service.json
    local repo_url
    repo_url=$(repository::get_url 2>/dev/null || echo "")
    
    if [[ -z "$repo_url" ]]; then
        log::warn "No repository URL found in service.json, skipping template update"
        return 0
    fi
    
    # Replace REPOSITORY_URL_PLACEHOLDER with actual URL
    if grep -q "REPOSITORY_URL_PLACEHOLDER" "$template_file"; then
        log::info "Updating repository URL in template: $template_file"
        sed -i "s|REPOSITORY_URL_PLACEHOLDER|${repo_url}|g" "$template_file"
        
        # Also update related placeholders
        sed -i "s|ISSUES_URL_PLACEHOLDER|${repo_url}/issues|g" "$template_file"
        sed -i "s|HOMEPAGE_URL_PLACEHOLDER|${repo_url}|g" "$template_file"
    fi
    
    return 0
}

#######################################
# Resolve resource list from input
# Outputs: space-separated list of resources
#######################################
resources::resolve_list() {
    local input="$RESOURCES_INPUT"
    local resolved=""
    
    case "$input" in
        "all")
            resolved="$ALL_RESOURCES"
            ;;
        "ai-only")
            resolved="${AVAILABLE_RESOURCES[ai]}"
            ;;
        "automation-only")
            resolved="${AVAILABLE_RESOURCES[automation]}"
            ;;
        "storage-only")
            resolved="${AVAILABLE_RESOURCES[storage]}"
            ;;
        "agents-only")
            resolved="${AVAILABLE_RESOURCES[agents]}"
            ;;
        "search-only")
            resolved="${AVAILABLE_RESOURCES[search]}"
            ;;
        "none"|"")
            resolved=""
            ;;
        "enabled")
            # Get enabled resources from configuration
            # Need to source common.sh if not already sourced
            if ! declare -f resources::get_enabled_from_config >/dev/null 2>&1; then
                # shellcheck disable=SC1091
                source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
            fi
            resolved=$(resources::get_enabled_from_config)
            if [[ -z "$resolved" ]]; then
                log::info "No enabled resources found in configuration" >&2
            fi
            ;;
        *)
            # Parse comma-separated list
            IFS=',' read -ra RESOURCE_ARRAY <<< "$input"
            for resource in "${RESOURCE_ARRAY[@]}"; do
                resource=$(echo "$resource" | tr -d ' ')  # trim whitespace
                if [[ " $ALL_RESOURCES " =~ " $resource " ]]; then
                    resolved="$resolved $resource"
                else
                    log::warn "Unknown resource: $resource (skipping)" >&2
                    log::info "Available resources: $ALL_RESOURCES" >&2
                    # Don't exit, just skip the unknown resource
                fi
            done
            ;;
    esac
    
    # Apply category filter if specified
    if [[ -n "$CATEGORY" && -n "$resolved" ]]; then
        local filtered=""
        local category_resources="${AVAILABLE_RESOURCES[$CATEGORY]}"
        for resource in $resolved; do
            if [[ " $category_resources " =~ " $resource " ]]; then
                filtered="$filtered $resource"
            fi
        done
        resolved="$filtered"
    fi
    
    echo "$resolved"
}

#######################################
# Get the category for a resource
# Arguments:
#   $1 - resource name
# Outputs: category name
#######################################
resources::get_category() {
    local resource="$1"
    for category in "${!AVAILABLE_RESOURCES[@]}"; do
        if [[ " ${AVAILABLE_RESOURCES[$category]} " =~ " $resource " ]]; then
            echo "$category"
            return
        fi
    done
    echo "unknown"
}

#######################################
# Get the script path for a resource
# Arguments:
#   $1 - resource name
# Outputs: script path
#######################################
resources::get_script_path() {
    local resource="$1"
    local category
    category=$(resources::get_category "$resource")
    echo "${RESOURCES_DIR}/${category}/${resource}/manage.sh"
}

#######################################
# Check if a resource script exists
# Arguments:
#   $1 - resource name
# Returns: 0 if exists, 1 otherwise
#######################################
resources::script_exists() {
    local resource="$1"
    local script_path
    script_path=$(resources::get_script_path "$resource")
    [[ -f "$script_path" ]]
}

#######################################
# Check if a resource is API-based (external service)
# Arguments:
#   $1 - resource name
# Returns: 0 if API-based, 1 otherwise
#######################################
resources::is_api_based() {
    local resource="$1"
    
    # Common API-based resources that don't need local installation
    case "$resource" in
        cloudflare|openrouter|openai|anthropic|cohere|huggingface)
            return 0
            ;;
        *)
            # Check if resource configuration indicates it's API-based
            # This would require parsing the config file, but for now we'll use the known list
            return 1
            ;;
    esac
}

#######################################
# Execute action on a single resource
# Arguments:
#   $1 - resource name
#   $2 - action
# Returns: exit code from resource script
#######################################
resources::execute_action() {
    local resource="$1"
    local action="$2"
    local script_path
    script_path=$(resources::get_script_path "$resource")
    
    # Check if this is an API-based resource that doesn't need local installation
    if resources::is_api_based "$resource"; then
        if [[ "$action" == "install" ]]; then
            log::info "🌐 $resource: API-based service (no local installation needed)"
            log::info "   Configure API keys in environment variables and enable in service.json"
            return 0  # Success - no installation needed
        elif [[ "$action" == "status" ]]; then
            log::info "🌐 $resource: API-based service (check configuration and API keys)"
            return 0
        else
            log::info "🌐 $resource: API-based service (action '$action' not applicable)"
            return 0
        fi
    fi
    
    if ! resources::script_exists "$resource"; then
        log::warn "⚠️  Resource script not found: $script_path"
        log::info "   Skipping $resource (resource not implemented or moved)"
        return 2  # Special return code for missing scripts
    fi
    
    log::header "🔧 $resource: $action"
    
    # Execute the resource script with the action
    # Don't let script failures propagate with 'set -e'
    local exit_code=0
    bash "$script_path" --action "$action" --force "$FORCE" --yes "$YES" || exit_code=$?
    
    if [[ $exit_code -ne 0 ]]; then
        log::warn "⚠️  Resource $resource failed with exit code $exit_code"
        # Check for common failure patterns and provide helpful messages
        case "$exit_code" in
            1)
                log::info "   This is often due to port conflicts, permission issues, or missing dependencies"
                ;;
            2)
                log::info "   This typically indicates a configuration or validation error"
                ;;
            *)
                log::info "   Check the output above for specific error details"
                ;;
        esac
    fi
    
    return $exit_code
}

#######################################
# List available resources
#######################################
resources::list_available() {
    log::header "📋 Available Resources"
    
    for category in "${!AVAILABLE_RESOURCES[@]}"; do
        log::info "Category: $category"
        for resource in ${AVAILABLE_RESOURCES[$category]}; do
            local script_path
            script_path=$(resources::get_script_path "$resource")
            local status="❌ Not implemented"
            
            if [[ -f "$script_path" ]]; then
                status="✅ Available"
                # Check if actually installed/running
                local port
                port=$(resources::get_default_port "$resource")
                if resources::is_service_running "$port"; then
                    status="🟢 Running"
                fi
            fi
            
            echo "  - $resource: $status"
        done
        echo
    done
}

#######################################
# Check Claude Code MCP integration status
#######################################
resources::check_mcp_integration() {
    echo
    log::header "🔗 Claude Code MCP Integration Status"
    
    local claude_code_script="${RESOURCES_DIR}/agents/claude-code/manage.sh"
    
    if [[ ! -f "$claude_code_script" ]]; then
        log::info "Claude Code resource not available"
        log::info "Install with: $0 --action install --resources claude-code"
        return 0
    fi
    
    # Check Claude Code installation status
    if bash "$claude_code_script" --action status >/dev/null 2>&1; then
        log::success "✅ Claude Code is installed"
        
        # Check MCP registration status
        local mcp_status
        # Use || true to prevent script exit on error due to set -e
        mcp_status=$(bash "$claude_code_script" --action mcp-status --format json 2>/dev/null || true)
        if [[ -n "$mcp_status" ]]; then
            # Parse JSON status (simple approach)
            if echo "$mcp_status" | grep -q '"registered":true'; then
                log::success "✅ Vrooli MCP server is registered with Claude Code"
                local scopes
                scopes=$(echo "$mcp_status" | grep -o '"scopes":\[[^]]*\]' | sed 's/"scopes":\[//' | sed 's/\]//' | tr ',' ' ' | tr -d '"')
                log::info "   Registered scopes: $scopes"
                log::info "🎯 You can use @vrooli in Claude Code to access Vrooli tools"
            else
                log::warn "⚠️  Vrooli MCP server is not registered"
                log::info "   Register with: $claude_code_script --action register-mcp"
            fi
            
            # Check if Vrooli server is detected
            if echo "$mcp_status" | grep -q '"detected":true'; then
                log::success "✅ Vrooli server is accessible for MCP"
            else
                log::warn "⚠️  Vrooli server not detected or not running"
                log::info "   Ensure Vrooli is running with MCP enabled"
            fi
        else
            log::warn "⚠️  Could not check MCP status"
            log::info "   Check manually with: $claude_code_script --action mcp-status"
        fi
    else
        log::info "Claude Code is available but not installed"
        log::info "Install with: $claude_code_script --action install"
    fi
}

#######################################
# Get the health check endpoint for a resource
# Arguments:
#   $1 - resource name
# Outputs: health endpoint path
#######################################
resources::get_health_endpoint() {
    local resource="$1"
    case "$resource" in
        "agent-s2") echo "/health" ;;
        "browserless") echo "/pressure" ;;
        "ollama") echo "/api/tags" ;;
        "comfyui") echo "/" ;;  # ComfyUI root endpoint works better than system_stats
        "n8n") echo "/healthz" ;;
        "huginn") echo "/" ;;
        "whisper") echo "/docs" ;;  # Whisper has docs endpoint, not health
        "node-red") echo "/flows" ;;
        "windmill") echo "/api/version" ;;
        "minio") echo "/minio/health/live" ;;  # MinIO health endpoint
        "searxng") echo "/stats" ;;
        "claude-code") echo "/mcp/health" ;;
        "unstructured-io") echo "/healthcheck" ;;
        "qdrant") echo "/" ;;  # Qdrant root endpoint returns version info
        "vault") echo "/v1/sys/health" ;;  # HashiCorp Vault health endpoint
        "questdb") echo "/status" ;;  # QuestDB status endpoint
        "postgres") echo "/health" ;;  # PostgreSQL health endpoint (custom endpoint)
        *) echo "" ;;  # Return empty for unknown resources
    esac
}

#######################################
# Discover running resources
#######################################
resources::discover_running() {
    log::header "🔍 Discovering Running Resources"
    
    local found_any=false
    local discovered_resources=()
    
    for resource in $ALL_RESOURCES; do
        local resource_type
        resource_type=$(resources::get_resource_type "$resource")
        
        if [[ "$resource_type" == "cli" ]]; then
            # Handle CLI tools
            if resources::is_cli_resource_healthy "$resource"; then
                log::success "✅ $resource is available (CLI tool)"
                log::info "   Service validation: ✅ Validated as CLI tool"
                
                found_any=true
                discovered_resources+=("$resource")
                
                # Get detailed health information for CLI tools
                local script_path="${RESOURCES_DIR}/agents/$resource/manage.sh"
                if [[ -x "$script_path" ]]; then
                    local health_output
                    health_output=$("$script_path" --action health-check --check-type basic --format text 2>/dev/null | head -5)
                    if [[ -n "$health_output" ]]; then
                        echo "$health_output" | while IFS= read -r line; do
                            if [[ "$line" =~ ^(Overall Status|Version|Authentication): ]]; then
                                log::info "   $line"
                            fi
                        done
                    fi
                fi
            else
                continue
            fi
        else
            # Handle service-based resources
            local port
            port=$(resources::get_default_port "$resource")
            
            if resources::is_service_running "$port"; then
                # Validate it's actually the expected service
                local validation_status
                validation_status=$(resources::validate_service_identity "$resource" "$port")
                
                if [[ $? -eq 0 ]]; then
                    log::success "✅ $resource is running on port $port"
                    log::info "   Service validation: $validation_status"
                else
                    log::warn "⚠️  Port $port is in use but not by $resource"
                    log::info "   Service validation: $validation_status"
                    continue
                fi
                
                found_any=true
                discovered_resources+=("$resource")
                
                # Check health based on resource type
                if [[ "$resource" == "redis" ]]; then
                # Redis health check via TCP ping
                local redis_health
                if docker exec vrooli-redis-resource redis-cli ping 2>/dev/null | grep -q "PONG"; then
                    redis_health="✅ Healthy (Redis PING successful)"
                else
                    redis_health="❌ Unhealthy (Redis PING failed)"
                fi
                log::info "   Redis health check: $redis_health"
            elif [[ "$resource" == "postgres" ]]; then
                # PostgreSQL health check via pg_isready - find any healthy vrooli-postgres container
                local postgres_health
                local postgres_container
                postgres_container=$(docker ps --format "{{.Names}}" | grep "^vrooli-postgres-" | head -1)
                
                if [[ -n "$postgres_container" ]] && docker exec "$postgres_container" pg_isready -h localhost -p 5432 2>/dev/null; then
                    postgres_health="✅ Healthy (PostgreSQL ready)"
                else
                    postgres_health="❌ Unhealthy (PostgreSQL not ready)"
                fi
                log::info "   PostgreSQL health check: $postgres_health"
            else
                # HTTP health check for other resources
                local base_url="http://localhost:$port"
                local health_endpoint
                health_endpoint=$(resources::get_health_endpoint "$resource")
                local health_status
                health_status=$(resources::get_detailed_health_status "$base_url" "$health_endpoint")
                    log::info "   HTTP health check: $health_status"
                fi
                
                # Check model integrity for AI resources
            case "$resource" in
                "comfyui")
                    # Check if ComfyUI models are valid
                    if resources::script_exists "$resource"; then
                        local script_path
                        script_path=$(resources::get_script_path "$resource")
                        local model_status=$("$script_path" --action status 2>&1 | grep -A5 "Model Integrity" | grep -E "✅|❌|⚠️" | wc -l)
                        local valid_models=$("$script_path" --action status 2>&1 | grep -A5 "Model Integrity" | grep "✅" | wc -l)
                        local invalid_models=$("$script_path" --action status 2>&1 | grep -A5 "Model Integrity" | grep "❌" | wc -l)
                        
                        if [[ $model_status -gt 0 ]]; then
                            if [[ $invalid_models -gt 0 ]]; then
                                log::warn "   Model check: ⚠️  $invalid_models corrupted models detected"
                                log::info "   Run: $script_path --action download-models"
                            elif [[ $valid_models -eq 0 ]]; then
                                log::warn "   Model check: ⚠️  No models installed"
                                log::info "   Run: $script_path --action download-models"
                            else
                                log::info "   Model check: ✅ $valid_models models valid"
                            fi
                        fi
                    fi
                    ;;
                "ollama")
                    # Check if Ollama has models
                    local model_count=$(curl -s http://localhost:$port/api/tags 2>/dev/null | jq -r '.models | length' 2>/dev/null || echo "0")
                    if [[ "$model_count" -gt 0 ]]; then
                        log::info "   Model check: ✅ $model_count models available"
                    else
                        log::warn "   Model check: ⚠️  No models installed"
                        log::info "   Run: ollama pull llama3.1:8b"
                    fi
                    ;;
                esac
            else
                continue
            fi
        fi
    done
    
    if ! $found_any; then
        log::info "No known resources detected running"
        return 0
    fi
    
    # Auto-configure if requested
    if [[ "$AUTO_CONFIGURE" == "yes" ]]; then
        echo
        log::header "🔧 Auto-configuring discovered resources"
        
        for resource in "${discovered_resources[@]}"; do
            log::info "Configuring $resource..."
            
            # Check if already configured
            local category
            category=$(resources::get_category "$resource")
            local port
            port=$(resources::get_default_port "$resource")
            local base_url="http://localhost:$port"
            
            # Use the resource's manage script to update configuration only
            local script_path
            script_path=$(resources::get_script_path "$resource")
            
            if [[ -f "$script_path" ]]; then
                # Call the script with a special action to just update config
                # Most scripts have an update_config function
                log::info "Updating configuration for $resource..."
                
                # For now, directly update config since not all scripts may support config-only action
                if resources::script_exists "$resource"; then
                    # Update the configuration using the common function
                    local additional_config="{}"
                    
                    # Resource-specific configurations
                    case "$resource" in
                        "ollama")
                            additional_config='{"models":{"defaultModel":"llama3.1:8b","supportsFunctionCalling":true},"api":{"version":"v1","modelsEndpoint":"/api/tags","chatEndpoint":"/api/chat","generateEndpoint":"/api/generate"}}'
                            ;;
                        "n8n")
                            additional_config='{"api":{"version":"v1","workflowsEndpoint":"/api/v1/workflows","executionsEndpoint":"/api/v1/executions","credentialsEndpoint":"/api/v1/credentials"}}'
                            ;;
                    esac
                    
                    if resources::update_config "$category" "$resource" "$base_url" "$additional_config"; then
                        log::success "✅ $resource configured successfully"
                    else
                        log::warn "⚠️  Failed to configure $resource"
                    fi
                else
                    log::warn "⚠️  No script found for $resource, skipping configuration"
                fi
            else
                log::warn "⚠️  Script not found for $resource: $script_path"
            fi
        done
        
        echo
        log::success "✅ Auto-configuration complete"
        log::info "Configured resources are now available in ~/.vrooli/service.json"
    else
        echo
        log::info "To configure discovered resources for Vrooli, run:"
        log::info "  $0 --action install --resources <resource-name>"
        log::info "Or to auto-configure all discovered resources:"
        log::info "  $0 --action discover --auto-configure yes"
    fi
    
    # Check Claude Code MCP integration status
    # Temporarily disabled to debug hanging issue
    # resources::check_mcp_integration
}

#######################################
# Run interface validation for resources
# Arguments: List of resource names to validate
# Returns: 0 if all pass, 1 if any fail, 2 if validation error
#######################################
resources::run_interface_validation() {
    local resources_to_validate=("$@")
    
    if [[ ${#resources_to_validate[@]} -eq 0 ]]; then
        log::info "🔍 No resources to validate"
        return 0
    fi
    
    log::header "🔍 Interface Validation"
    log::info "Validating interface compliance for ${#resources_to_validate[@]} resources..."
    
    local validation_passed=0
    local validation_failed=0
    local validation_errors=0
    local failed_resources=()
    
    # Load the interface validation library if available
    local interface_lib="$RESOURCES_DIR/tests/lib/interface-validation.sh"
    if [[ -f "$interface_lib" ]]; then
        # shellcheck disable=SC1090
        source "$interface_lib" 2>/dev/null || log::warn "Could not load interface validation library"
    fi
    
    for resource in "${resources_to_validate[@]}"; do
        log::info "🔍 Validating: $resource"
        
        # Get resource category and script path
        local category
        category=$(resources::get_category "$resource")
        local script_path
        script_path=$(resources::get_script_path "$resource")
        
        if [[ ! -f "$script_path" ]]; then
            log::error "❌ $resource: manage.sh not found at $script_path"
            validation_errors=$((validation_errors + 1))
            failed_resources+=("$resource (missing script)")
            continue
        fi
        
        # Basic interface validation checks
        local validation_result=0
        local issues=()
        
        # Check 1: Script is executable
        if [[ ! -x "$script_path" ]]; then
            issues+=("Script not executable")
            validation_result=1
        fi
        
        # Check 2: Required actions exist
        local required_actions=("install" "start" "stop" "status" "logs")
        for action in "${required_actions[@]}"; do
            if ! grep -q "^[[:space:]]*$action)" "$script_path"; then
                issues+=("Missing action: $action")
                validation_result=1
            fi
        done
        
        # Check 3: Help patterns exist
        if ! grep -q "\-\-help\|help)\|usage" "$script_path"; then
            issues+=("Missing --help pattern or usage function")
            validation_result=1
        fi
        
        # Check 4: Error handling present
        if ! grep -q "set -euo pipefail\|set -e" "$script_path"; then
            issues+=("Missing error handling (set -e)")
            validation_result=1
        fi
        
        # Check 5: Required directory structure
        local resource_dir
        resource_dir=$(dirname "$script_path")
        if [[ ! -d "$resource_dir/config" ]]; then
            issues+=("Missing config/ directory")
            validation_result=1
        fi
        if [[ ! -d "$resource_dir/lib" ]]; then
            issues+=("Missing lib/ directory")
            validation_result=1
        fi
        
        # Report results for this resource
        if [[ $validation_result -eq 0 ]]; then
            log::success "✅ $resource: Interface validation passed"
            validation_passed=$((validation_passed + 1))
        else
            log::error "❌ $resource: Interface validation failed"
            for issue in "${issues[@]}"; do
                log::error "   • $issue"
            done
            validation_failed=$((validation_failed + 1))
            failed_resources+=("$resource")
        fi
    done
    
    # Summary
    echo
    log::info "Interface Validation Summary:"
    log::info "  ✅ Passed: $validation_passed"
    if [[ $validation_failed -gt 0 ]]; then
        log::warn "  ❌ Failed: $validation_failed (${failed_resources[*]})"
    fi
    if [[ $validation_errors -gt 0 ]]; then
        log::error "  💥 Errors: $validation_errors"
    fi
    
    # Return appropriate code
    if [[ $validation_errors -gt 0 ]]; then
        return 2  # Validation errors (missing scripts, etc.)
    elif [[ $validation_failed -gt 0 ]]; then
        return 1  # Interface compliance failures
    else
        return 0  # All passed
    fi
}

#######################################
# Run integration tests with auto-discovery
#######################################
resources::run_tests() {
    log::header "🧪 Resource Integration Tests"
    
    # Step 1: Auto-discover healthy resources with explicit filtering
    log::info "Auto-discovering healthy resources..."
    local healthy_resources=()
    local api_based_count=0
    local not_running_count=0
    local unhealthy_count=0
    
    for resource in $ALL_RESOURCES; do
        # Check if this is an API-based resource first
        if resources::is_api_based "$resource"; then
            log::info "🌐 $resource: Skipping (API-based service - no local testing needed)"
            api_based_count=$((api_based_count + 1))
            continue
        fi
        
        local port
        port=$(resources::get_default_port "$resource")
        
        # Check if port is defined for this resource
        if [[ -z "$port" ]]; then
            log::info "⏭️  $resource: Skipping (no port defined - not a local service)"
            not_running_count=$((not_running_count + 1))
            continue
        fi
        
        if resources::is_service_running "$port"; then
            local base_url="http://localhost:$port"
            local health_endpoint
            health_endpoint=$(resources::get_health_endpoint "$resource")
            
            if resources::check_http_health "$base_url" "$health_endpoint"; then
                healthy_resources+=("$resource")
                log::success "✅ $resource is healthy and ready for testing (port $port)"
            else
                log::warn "❌ $resource: Running on port $port but failing health checks"
                unhealthy_count=$((unhealthy_count + 1))
            fi
        else
            log::info "💤 $resource: Not running on expected port $port (skipping)"
            not_running_count=$((not_running_count + 1))
        fi
    done
    
    # Summary of filtering results
    echo
    log::info "Discovery Summary:"
    log::info "  ✅ Healthy local resources: ${#healthy_resources[@]}"
    log::info "  🌐 API-based resources (skipped): $api_based_count"
    log::info "  💤 Not running/no port: $not_running_count" 
    log::info "  ❌ Unhealthy: $unhealthy_count"
    echo
    
    if [[ ${#healthy_resources[@]} -eq 0 ]]; then
        log::error "No healthy resources found for testing"
        log::info "Start resources first, then run tests"
        return 1
    fi
    
    # Step 2: Filter resources if specific resource requested
    local test_resources=("${healthy_resources[@]}")
    if [[ -n "$RESOURCES_INPUT" && "$RESOURCES_INPUT" != "all" && "$RESOURCES_INPUT" != "enabled" ]]; then
        local resource_list
        resource_list=$(resources::resolve_list)
        
        local filtered_resources=()
        for resource in $resource_list; do
            if [[ " ${healthy_resources[*]} " =~ " $resource " ]]; then
                filtered_resources+=("$resource")
            else
                log::warn "⚠️  Requested resource $resource is not healthy, skipping"
            fi
        done
        test_resources=("${filtered_resources[@]}")
    fi
    
    # Step 3: Run interface validation before integration tests
    resources::run_interface_validation "${test_resources[@]}"
    local interface_validation_result=$?
    if [[ $interface_validation_result -ne 0 ]]; then
        log::warn "⚠️  Interface validation issues found, but continuing with integration tests"
        log::info "💡 Fix interface issues for better test reliability"
    fi
    
    # Step 4: Execute integration tests with proper environment
    local tests_dir="${RESOURCES_DIR}/tests"
    export HEALTHY_RESOURCES_STR="${test_resources[*]}"
    export SCRIPT_DIR="$tests_dir"
    export RESOURCES_DIR="$RESOURCES_DIR"
    
    log::info "Testing resources: ${test_resources[*]}"
    log::info "Executing test runner..."
    
    # Check if test runner exists
    if [[ ! -f "$tests_dir/run.sh" ]]; then
        log::error "Test runner not found at: $tests_dir/run.sh"
        log::info "Make sure you're running from the correct directory"
        return 1
    fi
    
    # Execute the test runner with populated environment
    bash "$tests_dir/run.sh" "$@"
}

#######################################
# Main execution function
#######################################
resources::main() {
    resources::parse_arguments "$@"
    
    case "$ACTION" in
        "list")
            resources::list_available
            return 0
            ;;
        "discover")
            resources::discover_running
            return 0
            ;;
        "test")
            resources::run_tests
            return 0
            ;;
        "inject")
            if [[ -n "$SCENARIO_NAME" ]]; then
                "${var_ROOT_DIR}/scripts/scenarios/injection/engine.sh" --action inject --scenario "$SCENARIO_NAME" --config-file "$SCENARIOS_CONFIG"
            else
                log::error "Scenario name required for injection action"
                log::info "Use: --scenario SCENARIO_NAME"
                exit 1
            fi
            return 0
            ;;
        "inject-all")
            "${var_ROOT_DIR}/scripts/scenarios/injection/engine.sh" --action inject --all-active yes --config-file "$SCENARIOS_CONFIG"
            return 0
            ;;
    esac
    
    local resource_list
    resource_list=$(resources::resolve_list)
    
    if [[ -z "$resource_list" ]]; then
        # Special handling for "enabled" when no resources are enabled
        if [[ "$RESOURCES_INPUT" == "enabled" ]]; then
            log::info "No resources to process (none are enabled in configuration)"
            exit 0
        fi
        # Special handling for "none" - just exit successfully
        if [[ "$RESOURCES_INPUT" == "none" ]]; then
            log::info "No resources to process (--resources none specified)"
            exit 0
        fi
        log::warn "No valid resources found to process"
        log::info "Use --resources to specify resources, or --action list to see available options"
        # Don't exit with error code - this allows main setup to continue
        exit 0
    fi
    
    log::header "🚀 Resource Management"
    log::info "Action: $ACTION"
    log::info "Resources: $resource_list"
    echo
    
    local processed_count=0
    local success_count=0
    local warning_count=0
    local skipped_count=0
    local failed_resources=()
    local skipped_resources=()
    
    for resource in $resource_list; do
        processed_count=$((processed_count + 1))
        
        local result_code
        resources::execute_action "$resource" "$ACTION"
        result_code=$?
        
        case "$result_code" in
            0)
                success_count=$((success_count + 1))
                log::success "✅ $resource: $ACTION completed successfully"
                ;;
            2)
                # Missing script - treat as skipped
                skipped_count=$((skipped_count + 1))
                skipped_resources+=("$resource")
                log::info "⏭️  $resource: Skipped (not implemented)"
                ;;
            *)
                # Other failures - treat as warnings but continue
                warning_count=$((warning_count + 1))
                failed_resources+=("$resource")
                log::warn "⚠️  $resource: $ACTION failed but continuing with other resources"
                ;;
        esac
        
        # Add spacing between resources
        if [[ $processed_count -lt $(echo "$resource_list" | wc -w) ]]; then
            echo
        fi
    done
    
    echo
    log::header "📊 Summary"
    log::info "Processed: $processed_count resources"
    log::info "Successful: $success_count resources"
    
    if [[ $skipped_count -gt 0 ]]; then
        log::info "Skipped: $skipped_count resources (${skipped_resources[*]})"
    fi
    
    if [[ $warning_count -gt 0 ]]; then
        log::warn "Failed: $warning_count resources (${failed_resources[*]})"
    fi
    
    # Determine overall status
    if [[ $success_count -eq $processed_count ]]; then
        log::success "✅ All operations completed successfully"
    elif [[ $success_count -gt 0 ]]; then
        log::success "✅ Partial success: $success_count/$processed_count resources completed successfully"
        if [[ $warning_count -gt 0 ]]; then
            log::info "💡 Some resources failed but the main setup can continue"
            log::info "   Failed resources can be installed individually later if needed"
        fi
        # Don't exit with error code for partial success
    else
        log::warn "⚠️  No resources were successfully $ACTION"
        if [[ $skipped_count -eq $processed_count ]]; then
            log::info "💡 All resources were skipped (not implemented), this may be expected"
        else
            log::info "💡 You may need to address the issues above and retry"
        fi
        # Don't exit with error code even for no successes, let the main setup continue
    fi
    
    # Show next steps for install action
    if [[ "$ACTION" == "install" && $success_count -gt 0 ]]; then
        echo
        log::header "🎯 Next Steps"
        log::info "1. Resources have been configured in ~/.vrooli/service.json"
        log::info "2. Start your Vrooli development environment to begin using these resources"
        log::info "3. Check resource status with: $0 --action status --resources $RESOURCES_INPUT"
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resources::main "$@"
fi