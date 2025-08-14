#!/usr/bin/env bash
set -euo pipefail

# Prevent multiple sourcing to avoid readonly variable conflicts
[[ -n "${VROOLI_COMMON_SOURCED:-}" ]] && return 0
readonly VROOLI_COMMON_SOURCED=1

# Common utilities for local resource setup and management
# This file provides shared functionality for all resource setup scripts
# 
# Updated to use standardized JSON utilities for consistent service.json
# parsing and configuration management.

# Source var.sh first with relative path
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_FLOW_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/ports.sh"
# shellcheck disable=SC1091
source "${var_LIB_DIR}/runtimes/docker.sh"
# shellcheck disable=SC1091
source "${var_PORT_REGISTRY_FILE}"
# shellcheck disable=SC1091
source "${var_REPOSITORY_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"
# shellcheck disable=SC1091
source "${var_SYSTEM_COMMANDS_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/json.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-auto-install.sh" 2>/dev/null || true

# Resource configuration paths
# Use the project's .vrooli directory, not the home directory
readonly VROOLI_CONFIG_DIR="${var_VROOLI_CONFIG_DIR}"
readonly VROOLI_RESOURCES_CONFIG="${var_SERVICE_JSON_FILE}"

# Configuration manager script path
readonly CONFIG_MANAGER_SCRIPT="${var_SCRIPTS_RESOURCES_DIR}/common/config-manager.js"

# Common default ports for local resources (from port registry)
declare -A DEFAULT_PORTS
for resource in "${!RESOURCE_PORTS[@]}"; do
    DEFAULT_PORTS["$resource"]="${RESOURCE_PORTS[$resource]}"
done

# Resource type definitions - distinguish CLI tools from services
declare -A RESOURCE_TYPES=(
    # CLI tools (no port, use command-based health checks)
    ["claude-code"]="cli"
    
    # Web services (port-based)
    ["ollama"]="service"
    ["whisper"]="service"
    ["unstructured-io"]="service"
    ["n8n"]="service"
    ["comfyui"]="service"
    ["node-red"]="service"
    ["windmill"]="service"
    ["huginn"]="service"
    ["minio"]="service"
    ["vault"]="service"
    ["qdrant"]="service"
    ["questdb"]="service"
    ["postgres"]="service"
    ["redis"]="service"
    ["browserless"]="service"
    ["agent-s2"]="service"
    ["searxng"]="service"
)

# Error handling and rollback support
declare -a ROLLBACK_ACTIONS=()
declare -g OPERATION_ID=""

#######################################
# Check if a service is running on the given port
# Arguments:
#   $1 - port number
# Returns:
#   0 if service is running, 1 otherwise
#######################################
resources::is_service_running() {
    local port="$1"
    ports::validate_port "$port"
    
    # Method 1: Try lsof (original method)
    local pids
    pids=$(ports::get_listening_pids "$port" 2>/dev/null)
    if [[ -n "$pids" ]]; then
        return 0
    fi
    
    # Method 2: Use netstat as fallback
    if system::is_command "netstat"; then
        if netstat -tlnp 2>/dev/null | grep -q ":${port} "; then
            return 0
        fi
    fi
    
    # Method 3: Use ss as fallback
    if system::is_command "ss"; then
        if ss -tlnp 2>/dev/null | grep -q ":${port} "; then
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Check if a service responds to HTTP health check
# Arguments:
#   $1 - base URL (e.g., http://localhost:11434)
#   $2 - health endpoint (optional, defaults to empty)
# Returns:
#   0 if healthy, 1 otherwise
#######################################
resources::check_http_health() {
    local base_url="$1"
    local health_endpoint="${2:-}"
    local url="${base_url}${health_endpoint}"
    
    if system::is_command "curl"; then
        curl -f -s --max-time 5 "$url" >/dev/null 2>&1
    else
        log::warn "curl not available, skipping HTTP health check for $url"
        return 1
    fi
}

#######################################
# Enhanced HTTP health check with detailed error reporting
# Arguments:
#   $1 - base URL (e.g., http://localhost:11434)
#   $2 - health endpoint (optional, defaults to empty)
# Outputs:
#   Detailed status message with specific error information
# Returns:
#   0 if healthy, 1 otherwise
#######################################
resources::get_detailed_health_status() {
    local base_url="$1"
    local health_endpoint="${2:-}"
    local url="${base_url}${health_endpoint}"
    
    if ! system::is_command "curl"; then
        echo "curl not available - cannot check health"
        return 1
    fi
    
    # Capture both HTTP status code and curl exit code
    local temp_output
    temp_output=$(mktemp)
    local http_code
    local curl_exit_code
    
    # Use curl with detailed options to capture both status and errors
    http_code=$(curl -s -w "%{http_code}" --max-time 5 --connect-timeout 3 \
                --output "$temp_output" "$url" 2>/dev/null)
    curl_exit_code=$?
    
    # Clean up temp file
    trash::safe_remove "$temp_output" --temp
    
    # Analyze the results and provide specific error messages
    case $curl_exit_code in
        0)
            # Success - check HTTP status code
            case $http_code in
                200|201|202)
                    echo "✅ Healthy"
                    return 0
                    ;;
                404)
                    echo "⚠️  Service running but health endpoint not found (HTTP 404)"
                    return 1
                    ;;
                403)
                    echo "⚠️  Service running but access denied (HTTP 403)"
                    return 1
                    ;;
                401)
                    echo "⚠️  Service running but authentication required (HTTP 401)"
                    return 1
                    ;;
                5*)
                    echo "⚠️  Service running but internal error (HTTP $http_code)"
                    return 1
                    ;;
                *)
                    echo "⚠️  Service responded with HTTP $http_code"
                    return 1
                    ;;
            esac
            ;;
        7)
            echo "⚠️  Service not running (connection refused)"
            return 1
            ;;
        28)
            echo "⚠️  Service running but health check timed out"
            return 1
            ;;
        6)
            echo "⚠️  Network connectivity problem (DNS resolution failed)"
            return 1
            ;;
        *)
            echo "⚠️  Health check failed (curl error $curl_exit_code)"
            return 1
            ;;
    esac
}

#######################################
# Get the default port for a resource
# Arguments:
#   $1 - resource name
# Outputs:
#   The default port number
#######################################
resources::get_default_port() {
    local resource="$1"
    echo "${DEFAULT_PORTS[$resource]:-8080}"
}

#######################################
# Get the resource type (cli or service)
# Arguments:
#   $1 - resource name
# Outputs:
#   The resource type (cli|service)
#######################################
resources::get_resource_type() {
    local resource="$1"
    echo "${RESOURCE_TYPES[$resource]:-service}"
}

#######################################
# Check if a CLI resource is available and healthy
# Arguments:
#   $1 - resource name
# Returns:
#   0 if healthy, 1 otherwise
#######################################
resources::is_cli_resource_healthy() {
    local resource="$1"
    
    case "$resource" in
        "claude-code")
            # Use the claude-code resource's health check
            local script_path="${var_SCRIPTS_RESOURCES_DIR}/agents/claude-code/manage.sh"
            if [[ -x "$script_path" ]]; then
                # Run health check and capture both exit code and output
                local health_output
                if health_output=$("$script_path" --action health-check --check-type basic --format json 2>/dev/null); then
                    # Parse JSON to check if status is healthy
                    if echo "$health_output" | grep -q '"status": "healthy"'; then
                        return 0
                    fi
                fi
            fi
            return 1
            ;;
        *)
            # Unknown CLI resource
            return 1
            ;;
    esac
}

#######################################
# Get the health check endpoint for a resource
# Arguments:
#   $1 - resource name
# Outputs:
#   The health check endpoint path
#######################################
resources::get_health_endpoint() {
    local resource="$1"
    
    # Resource-specific health endpoints
    case "$resource" in
        "ollama")
            echo "/api/tags"
            ;;
        "browserless")
            echo "/pressure"
            ;;
        "agent-s2")
            echo "/health"
            ;;
        "n8n")
            echo "/healthz"
            ;;
        "node-red")
            echo "/flows"
            ;;
        "windmill")
            echo "/api/health"
            ;;
        "huginn")
            echo "/"
            ;;
        "comfyui")
            echo "/system_stats"
            ;;
        "whisper")
            echo "/health"
            ;;
        "minio")
            echo "/minio/health/live"
            ;;
        "vault")
            echo "/v1/sys/health"
            ;;
        "searxng")
            echo "/stats"
            ;;
        "claude-code")
            echo "/mcp/health"
            ;;
        "redis")
            # Redis is TCP-based, no HTTP health endpoint
            echo ""
            ;;
        *)
            # Default endpoint for unknown resources
            echo ""
            ;;
    esac
}

#######################################
# Validate that a service on a port is the expected service
# Arguments:
#   $1 - resource name
#   $2 - port number
# Returns:
#   0 if service is validated, 1 otherwise
# Outputs:
#   Validation status message
#######################################
resources::validate_service_identity() {
    local resource="$1"
    local port="$2"
    local base_url="http://localhost:$port"
    
    if ! resources::is_service_running "$port"; then
        echo "❌ No service running on port $port"
        return 1
    fi
    
    # Service-specific validation based on unique identifiers
    case "$resource" in
        "ollama")
            # Check for Ollama-specific API
            if curl -s "$base_url/api/version" 2>/dev/null | grep -q "version"; then
                echo "✅ Validated as Ollama"
                return 0
            fi
            ;;
        "searxng")
            # Check for SearXNG specific headers or content
            if curl -s -I "$base_url" 2>/dev/null | grep -qi "searxng\|searx"; then
                echo "✅ Validated as SearXNG"
                return 0
            fi
            # Also check HTML content
            if curl -s "$base_url" 2>/dev/null | grep -qi 'meta.*searxng\|searx'; then
                echo "✅ Validated as SearXNG"
                return 0
            fi
            ;;
        "browserless")
            # Check for Browserless-specific endpoints
            if curl -s "$base_url/pressure" 2>/dev/null | grep -q "running\|queued"; then
                echo "✅ Validated as Browserless"
                return 0
            fi
            ;;
        "n8n")
            # Check for n8n specific endpoints
            if curl -s "$base_url/healthz" 2>/dev/null | grep -qi "ok\|healthy"; then
                echo "✅ Validated as n8n"
                return 0
            fi
            ;;
        "minio")
            # Check MinIO health endpoint
            if curl -s "$base_url/minio/health/live" 2>/dev/null; then
                echo "✅ Validated as MinIO"
                return 0
            fi
            ;;
        "agent-s2")
            # Check Agent-S2 specific endpoints
            if curl -s "$base_url/health" 2>/dev/null | grep -qi "agent.*s2\|healthy"; then
                echo "✅ Validated as Agent-S2"
                return 0
            fi
            ;;
        "node-red")
            # Check for Node-RED flows endpoint
            if curl -s "$base_url/flows" 2>/dev/null | head -1 | grep -q "\[\|flows"; then
                echo "✅ Validated as Node-RED"
                return 0
            fi
            ;;
        "vault")
            # Check Vault sys endpoint
            if curl -s "$base_url/v1/sys/health" 2>/dev/null | grep -q "initialized\|sealed"; then
                echo "✅ Validated as Vault"
                return 0
            fi
            ;;
        "whisper")
            # Check for Whisper service - check docs or openapi.json endpoint
            if curl -s "$base_url/docs" 2>/dev/null | grep -qi "whisper\|swagger"; then
                echo "✅ Validated as Whisper"
                return 0
            elif curl -s "$base_url/openapi.json" 2>/dev/null | grep -q "Whisper Asr Webservice"; then
                echo "✅ Validated as Whisper"
                return 0
            fi
            ;;
        "unstructured-io")
            # Check for Unstructured-IO service via healthcheck endpoint
            if curl -s "$base_url/healthcheck" 2>/dev/null | grep -qi "ok\|healthy"; then
                echo "✅ Validated as Unstructured-IO"
                return 0
            elif curl -s "$base_url/general/v0/general" 2>/dev/null | head -1 | grep -q "422\|405"; then
                # The API endpoint exists but needs proper request
                echo "✅ Validated as Unstructured-IO"
                return 0
            fi
            ;;
        "postgres")
            # PostgreSQL validation - check if it's a postgres container
            local postgres_container
            postgres_container=$(docker ps --format "{{.Names}}" | grep "^vrooli-postgres-" | head -1)
            if [[ -n "$postgres_container" ]]; then
                echo "✅ Validated as PostgreSQL"
                return 0
            fi
            ;;
        "redis")
            # Redis validation - check if it's the redis container
            if docker ps --format "{{.Names}}" | grep -q "vrooli-redis-resource"; then
                echo "✅ Validated as Redis"
                return 0
            fi
            ;;
        "comfyui")
            # Check ComfyUI system stats
            if curl -s "$base_url/system_stats" 2>/dev/null | grep -q "system\|python_version"; then
                echo "✅ Validated as ComfyUI"
                return 0
            fi
            ;;
        "qdrant")
            # Check Qdrant API
            if curl -s "$base_url/collections" 2>/dev/null | grep -q "collections\|result"; then
                echo "✅ Validated as Qdrant"
                return 0  
            fi
            ;;
        *)
            # For unknown services, just check if something is responding
            if curl -s -I "$base_url" 2>/dev/null | grep -q "200\|301\|302"; then
                echo "⚠️  Service responding but cannot validate identity"
                return 0
            fi
            ;;
    esac
    
    echo "❌ Service on port $port is not $resource"
    return 1
}

#######################################
# Validate and get safe port for resource
# Arguments:
#   $1 - resource name
#   $2 - requested port (optional)
#   $3 - force flag (optional, defaults to "no")
# Returns:
#   0 if port is safe, 1 if conflicts exist
# Outputs:
#   The validated port number
#######################################
resources::validate_port() {
    local resource="$1"
    local requested_port="${2:-}"
    local force_flag="${3:-no}"
    
    # Use requested port or fall back to default
    local port="${requested_port:-$(resources::get_default_port "$resource")}"
    
    # Validate the port assignment
    if ! ports::validate_assignment "$port" "$resource"; then
        log::error "Port $port conflicts with Vrooli services or is invalid"
        return 1
    fi
    
    # Check if port is already in use
    if ports::is_port_in_use "$port"; then
        log::warn "Port $port is already in use"
        
        # Try to find what's using it
        local pids
        pids=$(ports::get_listening_pids "$port" 2>/dev/null || echo "")
        if [[ -n "$pids" ]]; then
            log::info "Process(es) using port $port: $pids"
        fi
        
        # Check if it's the same service already running (for --force scenarios)
        local process_name=""
        if [[ -n "$pids" ]]; then
            # Get the process name for the first PID
            process_name=$(ps -p "$pids" -o comm= 2>/dev/null | head -n1 | tr -d ' ')
        fi
        
        # Allow if it's the same service and force is enabled
        if [[ "$resource" == "$process_name" ]] && [[ "$force_flag" == "yes" ]]; then
            log::info "Port is used by same service ($resource) and --force is enabled, proceeding"
            echo "$port"
            return 0
        fi
        
        # Special case for ollama service
        if [[ "$resource" == "ollama" ]] && [[ "$process_name" == "ollama" ]] && [[ "$force_flag" == "yes" ]]; then
            log::info "Port is used by existing Ollama service and --force is enabled, proceeding"
            echo "$port"
            return 0
        fi
        
        # Check if it's the same service already running (allow existing services)
        if [[ "$resource" == "$process_name" ]]; then
            log::info "Port is already used by the same service ($resource), assuming it's correctly configured"
            echo "$port"
            return 0
        fi
        
        # Be more tolerant of port conflicts - suggest alternatives but don't fail hard
        local category=""
        case "$resource" in
            ollama|whisper) category="AI" ;;
            n8n|node-red|comfyui) category="automation" ;;
            minio) category="storage" ;;
            browserless|claude-code|huginn) category="agents" ;;
        esac
        
        log::info "💡 Port conflict detected but continuing setup"
        log::info "   The $resource service may need manual configuration or a different port"
        log::info "   Consider setting a custom port: export ${resource^^}_CUSTOM_PORT=<alternative-port>"
        log::info "   Or stop the conflicting process and retry: sudo lsof -ti:$port | xargs sudo kill"
        
        # Return the port anyway but with warning status
        echo "$port"
        return 2  # Warning status - port has conflict but we'll continue
    fi
    
    echo "$port"
    return 0
}

#######################################
# Create vrooli config directory if it doesn't exist
#######################################
resources::ensure_config_dir() {
    if [[ ! -d "$VROOLI_CONFIG_DIR" ]]; then
        log::info "Creating Vrooli config directory: $VROOLI_CONFIG_DIR"
        mkdir -p "$VROOLI_CONFIG_DIR"
    fi
}

#######################################
# Start a rollback context for error recovery
# Arguments:
#   $1 - operation name (e.g., "install_ollama")
#######################################
resources::start_rollback_context() {
    local operation="$1"
    OPERATION_ID="${operation}_$(date +%s)"
    ROLLBACK_ACTIONS=()
    log::debug "Started rollback context: $OPERATION_ID"
}

#######################################
# Add a rollback action to the current context
# Arguments:
#   $1 - description of the action
#   $2 - command to execute for rollback
#   $3 - priority (optional, default: 0, higher = execute first)
#######################################
resources::add_rollback_action() {
    local description="$1"
    local command="$2"
    local priority="${3:-0}"
    
    ROLLBACK_ACTIONS+=("$priority|$description|$command")
    log::debug "Added rollback action: $description"
}

#######################################
# Execute all rollback actions in priority order
#######################################
resources::execute_rollback() {
    if [[ ${#ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No rollback actions to execute"
        return 0
    fi
    
    log::info "Executing rollback for operation: $OPERATION_ID"
    
    # Sort rollback actions by priority (highest first)
    local sorted_actions
    IFS=$'\n' sorted_actions=($(printf '%s\n' "${ROLLBACK_ACTIONS[@]}" | sort -nr))
    
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
    
    log::info "Rollback completed: $success_count/$total_count actions successful"
    
    # Clear rollback context
    ROLLBACK_ACTIONS=()
    OPERATION_ID=""
}

#######################################
# Improved error handling with user-friendly messages
# Arguments:
#   $1 - error message
#   $2 - error category (optional: network, permission, config, system)
#   $3 - suggested remediation (optional)
#######################################
resources::handle_error() {
    local error_message="$1"
    local error_category="${2:-unknown}"
    local remediation="${3:-}"
    
    log::error "Operation failed: $error_message"
    
    # Provide category-specific guidance
    case "$error_category" in
        "network")
            log::info "💡 Network Issue Remediation:"
            log::info "  • Check your internet connection"
            log::info "  • Verify firewall settings allow outbound connections"
            log::info "  • Test connectivity: curl -I https://google.com"
            ;;
        "permission")
            log::info "💡 Permission Issue Remediation:"
            log::info "  • Ensure you have sudo privileges: sudo -v"
            log::info "  • Check file/directory permissions"
            log::info "  • Verify user is in necessary groups (docker, etc.)"
            ;;
        "config")
            log::info "💡 Configuration Issue Remediation:"
            log::info "  • Validate configuration syntax"
            log::info "  • Check for conflicting settings"
            log::info "  • Consider restoring from backup"
            ;;
        "system")
            log::info "💡 System Issue Remediation:"
            log::info "  • Check system requirements are met"
            log::info "  • Verify required dependencies are installed"
            log::info "  • Check available disk space and memory"
            ;;
    esac
    
    if [[ -n "$remediation" ]]; then
        log::info "💡 Specific Remediation: $remediation"
    fi
    
    # Add repository context if available
    if command -v repository::get_url >/dev/null 2>&1; then
        local repo_url
        repo_url=$(repository::get_url 2>/dev/null || echo "")
        if [[ -n "$repo_url" ]]; then
            log::info "📚 For more help, see:"
            log::info "  • Documentation: ${repo_url}/tree/main/scripts/resources"
            log::info "  • Report issue: ${repo_url}/issues/new"
        fi
    fi
    
    # Execute rollback if context exists
    if [[ -n "$OPERATION_ID" ]]; then
        resources::execute_rollback
    fi
}

#######################################
# Validate configuration using TypeScript schema
#######################################
resources::validate_config() {
    log::info "Validating resource configuration..."
    
    if [[ ! -f "$CONFIG_MANAGER_SCRIPT" ]]; then
        log::warn "Configuration manager script not found, skipping validation"
        return 0
    fi
    
    if ! system::is_command "node"; then
        log::warn "Node.js not available, skipping configuration validation"
        return 0
    fi
    
    if node "$CONFIG_MANAGER_SCRIPT" validate; then
        log::success "Configuration validation passed"
        return 0
    else
        log::error "Configuration validation failed"
        return 1
    fi
}

#######################################
# Add or update a resource configuration using TypeScript manager
# Arguments:
#   $1 - category (ai, automation, storage, agents)
#   $2 - resource name
#   $3 - base URL
#   $4 - additional config JSON (optional)
#######################################
resources::update_config() {
    local category="$1"
    local resource_name="$2"
    local base_url="$3"
    local additional_config="${4:-{}}"
    
    log::info "Updating resource configuration for $category/$resource_name"
    
    # Create base configuration
    local resource_config
    if system::is_command "jq"; then
        # Create base config first, then merge with additional config
        local base_config="{\"enabled\": true, \"baseUrl\": \"$base_url\", \"healthCheck\": {\"intervalMs\": 60000, \"timeoutMs\": 5000}}"
        
        # Try to merge configurations safely
        if [[ "$additional_config" != "{}" ]] && [[ -n "$additional_config" ]]; then
            # Use echo and pipe to avoid shell expansion issues
            resource_config=$(echo "$base_config" | jq --arg additional "$additional_config" '. + ($additional | fromjson)' 2>/dev/null)
            
            # If merge fails, just use base config
            if [[ $? -ne 0 ]] || [[ -z "$resource_config" ]]; then
                log::warn "Failed to merge additional configuration, using base config only"
                resource_config="$base_config"
            fi
        else
            resource_config="$base_config"
        fi
    else
        # Fallback to basic JSON
        resource_config="{\"enabled\": true, \"baseUrl\": \"$base_url\", \"healthCheck\": {\"intervalMs\": 60000, \"timeoutMs\": 5000}}"
    fi
    
    # Use TypeScript configuration manager if available
    if [[ -f "$CONFIG_MANAGER_SCRIPT" ]] && system::is_command "node"; then
        if node "$CONFIG_MANAGER_SCRIPT" update \
            --category "$category" \
            --resource "$resource_name" \
            --config "$resource_config"; then
            log::success "Configuration updated using TypeScript manager"
            
            # Add rollback action
            resources::add_rollback_action \
                "Remove $category/$resource_name configuration" \
                "node \"$CONFIG_MANAGER_SCRIPT\" remove --category \"$category\" --resource \"$resource_name\"" \
                10
            
            return 0
        else
            log::warn "TypeScript configuration manager failed, falling back to manual method"
        fi
    fi
    
    # Fallback to legacy method with improved error handling
    resources::ensure_config_dir
    resources::init_config_legacy
    
    if ! system::is_command "jq"; then
        resources::handle_error \
            "jq command not available for configuration management" \
            "system" \
            "Install jq: sudo apt-get install jq (Ubuntu/Debian) or brew install jq (macOS)"
        return 1
    fi
    
    # Atomic update using temporary file
    local temp_config
    temp_config=$(mktemp) || {
        resources::handle_error "Failed to create temporary file" "system"
        return 1
    }
    
    # Add rollback action for temp file cleanup
    resources::add_rollback_action \
        "Clean up temporary configuration file" \
        "trash::safe_remove \"$temp_config\" --temp" \
        1
    
    # Update configuration atomically
    if jq \
        --arg category "$category" \
        --arg resource "$resource_name" \
        --argjson config "$resource_config" \
        '.resources[$category][$resource] = $config' \
        "$VROOLI_RESOURCES_CONFIG" > "$temp_config"; then
        
        # Validate the result
        if jq . "$temp_config" > /dev/null 2>&1; then
            mv "$temp_config" "$VROOLI_RESOURCES_CONFIG"
            log::success "Updated resource configuration for $category/$resource_name"
            
            # Add rollback action
            resources::add_rollback_action \
                "Remove $category/$resource_name configuration (legacy)" \
                "jq 'del(.resources.$category.$resource_name)' \"$VROOLI_RESOURCES_CONFIG\" > \"$temp_config\" && mv \"$temp_config\" \"$VROOLI_RESOURCES_CONFIG\"" \
                10
            
            return 0
        else
            trash::safe_remove "$temp_config" --temp
            resources::handle_error "Generated configuration is invalid JSON" "config"
            return 1
        fi
    else
        trash::safe_remove "$temp_config" --temp
        resources::handle_error "Failed to update configuration with jq" "config"
        return 1
    fi
}

#######################################
# Update CLI resource configuration
# Arguments:
#   $1 - category (ai, automation, storage, agents)
#   $2 - resource name
#   $3 - command name
#   $4 - additional config JSON (optional)
#######################################
resources::update_cli_config() {
    local category="$1"
    local resource_name="$2"
    local command="$3"
    local additional_config="${4:-{}}"
    
    log::info "Updating CLI resource configuration for $category/$resource_name"
    
    # Create CLI-specific configuration
    local resource_config
    if system::is_command "jq"; then
        # Create base config for CLI tools
        local base_config="{\"enabled\": true, \"type\": \"cli\", \"command\": \"$command\"}"
        
        # Try to merge configurations safely
        if [[ "$additional_config" != "{}" ]] && [[ -n "$additional_config" ]]; then
            resource_config=$(echo "$base_config" | jq --arg additional "$additional_config" '. + ($additional | fromjson)' 2>/dev/null)
            
            # If merge fails, just use base config
            if [[ $? -ne 0 ]] || [[ -z "$resource_config" ]]; then
                log::warn "Failed to merge additional configuration, using base config only"
                resource_config="$base_config"
            fi
        else
            resource_config="$base_config"
        fi
    else
        log::warn "jq not available, using simplified configuration"
        resource_config="{\"enabled\": true, \"type\": \"cli\", \"command\": \"$command\"}"
    fi
    
    # Use the same configuration update logic as regular resources
    local config_path="${VROOLI_CONFIG_DIR}/service.json"
    local temp_config
    temp_config=$(mktemp)
    
    # Try TypeScript configuration manager first
    if [[ -f "$CONFIG_MANAGER_SCRIPT" ]] && system::is_command "node"; then
        if node "$CONFIG_MANAGER_SCRIPT" update \
            --category "$category" \
            --resource "$resource_name" \
            --config "$resource_config"; then
            log::success "CLI resource configuration updated using TypeScript manager"
            return 0
        else
            log::warn "TypeScript configuration manager failed, falling back to manual method"
        fi
    fi
    
    # Fallback to manual jq-based update
    if system::is_command "jq"; then
        if jq --arg category "$category" \
              --arg resource "$resource_name" \
              --argjson config "$resource_config" \
              '.resources[$category][$resource] = $config' \
              "$config_path" > "$temp_config" 2>/dev/null; then
            
            if jq empty "$temp_config" 2>/dev/null; then
                mv "$temp_config" "$config_path"
                log::success "CLI resource configuration updated successfully"
                return 0
            fi
        fi
    fi
    
    trash::safe_remove "$temp_config" --temp
    log::error "Failed to update CLI resource configuration"
    return 1
}

#######################################
# Remove a resource from configuration
# Arguments:
#   $1 - category (ai, automation, storage, agents)
#   $2 - resource name
#######################################
resources::remove_config() {
    local category="$1"
    local resource_name="$2"
    
    log::info "Removing resource configuration for $category/$resource_name"
    
    # Use TypeScript configuration manager if available
    if [[ -f "$CONFIG_MANAGER_SCRIPT" ]] && system::is_command "node"; then
        if node "$CONFIG_MANAGER_SCRIPT" remove \
            --category "$category" \
            --resource "$resource_name"; then
            log::success "Configuration removed using TypeScript manager"
            return 0
        else
            log::warn "TypeScript configuration manager failed, falling back to manual method"
        fi
    fi
    
    # Fallback to legacy method
    if [[ ! -f "$VROOLI_RESOURCES_CONFIG" ]]; then
        log::info "Configuration file does not exist, nothing to remove"
        return 0
    fi
    
    if ! system::is_command "jq"; then
        resources::handle_error \
            "jq command not available for configuration management" \
            "system" \
            "Install jq: sudo apt-get install jq (Ubuntu/Debian) or brew install jq (macOS)"
        return 1
    fi
    
    local temp_config
    temp_config=$(mktemp) || {
        resources::handle_error "Failed to create temporary file" "system"
        return 1
    }
    
    if jq \
        --arg category "$category" \
        --arg resource "$resource_name" \
        'del(.resources[$category][$resource])' \
        "$VROOLI_RESOURCES_CONFIG" > "$temp_config"; then
        
        mv "$temp_config" "$VROOLI_RESOURCES_CONFIG"
        log::success "Removed resource configuration for $category/$resource_name"
    else
        trash::safe_remove "$temp_config" --temp
        resources::handle_error "Failed to remove configuration with jq" "config"
        return 1
    fi
}

#######################################
# Get list of enabled resources from configuration
# Uses standardized JSON utilities for consistent parsing
# Returns:
#   Space-separated list of enabled resource names
#######################################
resources::get_enabled_from_config() {
    if [[ ! -f "$VROOLI_RESOURCES_CONFIG" ]]; then
        # Config doesn't exist, return empty
        echo ""
        return 0
    fi
    
    # Use standardized JSON utility for robust resource extraction
    local enabled_resources
    enabled_resources=$(json::get_enabled_resources "" "$VROOLI_RESOURCES_CONFIG" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log::warn "Failed to parse resource configuration using JSON utilities" >&2
        echo ""
        return 0
    fi
    
    echo "$enabled_resources"
}

#######################################
# Initialize an empty resources configuration (legacy fallback)
#######################################
resources::init_config_legacy() {
    if [[ ! -f "$VROOLI_RESOURCES_CONFIG" ]]; then
        log::info "Creating initial resources configuration: $VROOLI_RESOURCES_CONFIG"
        cat > "$VROOLI_RESOURCES_CONFIG" << 'EOF'
{
  "version": "2.0.0",
  "service": {
    "name": "vrooli-local",
    "displayName": "Vrooli Local Instance",
    "description": "Local Vrooli development environment",
    "version": "2.0.0",
    "type": "platform"
  },
  "resources": {
    "ai": {},
    "automation": {},
    "storage": {},
    "agents": {},
    "execution": {}
  }
}
EOF
        
        # Validate the created configuration
        if ! jq . "$VROOLI_RESOURCES_CONFIG" > /dev/null 2>&1; then
            resources::handle_error "Failed to create valid initial configuration" "config"
            return 1
        fi
    fi
}

#######################################
# Wait for a service to become available
# Arguments:
#   $1 - service name (for logging)
#   $2 - port number
#   $3 - timeout in seconds (optional, defaults to 30)
# Returns:
#   0 if service becomes available, 1 if timeout
#######################################
resources::wait_for_service() {
    local service_name="$1"
    local port="$2"
    local timeout="${3:-30}"
    
    log::info "Waiting for $service_name to start on port $port (timeout: ${timeout}s)..."
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if resources::is_service_running "$port"; then
            log::success "$service_name is running on port $port"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log::error "$service_name failed to start within ${timeout}s"
    return 1
}

#######################################
# Download a file with progress indication
# Arguments:
#   $1 - URL
#   $2 - output file path
# Returns:
#   0 if successful, 1 otherwise
#######################################
resources::download_file() {
    local url="$1"
    local output_file="$2"
    
    log::info "Downloading: $url"
    
    if system::is_command "curl"; then
        curl -fL --progress-bar -o "$output_file" "$url"
    elif system::is_command "wget"; then
        wget -q --show-progress -O "$output_file" "$url"
    else
        log::error "Neither curl nor wget available for downloading"
        return 1
    fi
}


#######################################
# Install a systemd service
# Arguments:
#   $1 - service name
#   $2 - service file content
#######################################
resources::install_systemd_service() {
    local service_name="$1"
    local service_content="$2"
    local service_file="/etc/systemd/system/${service_name}.service"
    
    if ! sudo::can_use_sudo; then
        log::error "Sudo privileges required to install systemd service"
        return 1
    fi
    
    log::info "Installing systemd service: $service_name"
    
    # Write service file
    echo "$service_content" | sudo::exec_with_fallback "tee '$service_file'" >/dev/null
    
    # Reload systemd and enable service
    sudo::exec_with_fallback "systemctl daemon-reload"
    sudo::exec_with_fallback "systemctl enable '$service_name'"
    
    log::success "Systemd service $service_name installed and enabled"
}

#######################################
# Start a systemd service
# Arguments:
#   $1 - service name
#######################################
resources::start_service() {
    local service_name="$1"
    
    if ! sudo::can_use_sudo; then
        log::error "Sudo privileges required to start systemd service"
        return 1
    fi
    
    log::info "Starting service: $service_name"
    sudo::exec_with_fallback "systemctl start '$service_name'"
    log::success "Service $service_name started"
}

#######################################
# Stop a systemd service
# Arguments:
#   $1 - service name
#######################################
resources::stop_service() {
    local service_name="$1"
    
    if ! sudo::can_use_sudo; then
        log::error "Sudo privileges required to stop systemd service"
        return 1
    fi
    
    log::info "Stopping service: $service_name"
    sudo::exec_with_fallback "systemctl stop '$service_name'" 2>/dev/null || true
    log::success "Service $service_name stopped"
}

#######################################
# Get status of a systemd service
# Arguments:
#   $1 - service name
# Returns:
#   0 if service is active, 1 otherwise
#######################################
resources::is_service_active() {
    local service_name="$1"
    systemctl is-active --quiet "$service_name" 2>/dev/null
}

#######################################
# Check if a binary exists in PATH
# Arguments:
#   $1 - binary name
# Returns:
#   0 if binary exists, 1 otherwise
#######################################
resources::binary_exists() {
    local binary="$1"
    system::is_command "$binary"
}

#######################################
# Print resource status summary
# Arguments:
#   $1 - resource name
#   $2 - port
#   $3 - service name (optional)
#######################################
#######################################
# Ensure Docker is available and running
# Returns: 0 if Docker is ready, 1 otherwise
#######################################
resources::ensure_docker() {
    # Check if Docker is installed
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        log::info "Please install Docker first: https://docs.docker.com/get-docker/"
        return 1
    fi
    
    # Try to start Docker if needed
    if ! docker::start; then
        log::error "Failed to start Docker"
        return 1
    fi
    
    # Verify Docker is accessible
    if ! docker::run version >/dev/null 2>&1; then
        log::error "Docker is not accessible"
        docker::_diagnose_permission_issue
        return 1
    fi
    
    log::success "Docker is ready"
    return 0
}

resources::print_status() {
    local resource_name="$1"
    local port="$2"
    local service_name="${3:-}"
    
    log::header "📊 $resource_name Status"
    
    # Check if binary exists
    if resources::binary_exists "$resource_name"; then
        log::success "✅ $resource_name binary installed"
    else
        log::error "❌ $resource_name binary not found"
    fi
    
    # Check if service is running
    if resources::is_service_running "$port"; then
        log::success "✅ $resource_name service running on port $port"
    else
        log::error "❌ $resource_name service not running on port $port"
    fi
    
    # Check systemd service if provided
    if [[ -n "$service_name" ]]; then
        if resources::is_service_active "$service_name"; then
            log::success "✅ $service_name systemd service active"
        else
            log::warn "⚠️  $service_name systemd service not active"
        fi
    fi
    
    # Check configuration using standardized JSON utilities
    if [[ -f "$VROOLI_RESOURCES_CONFIG" ]]; then
        local config_exists=false
        
        # Check common resource categories
        for category in ai automation storage agents; do
            if json::get_resource_config "$category" "$resource_name" "$VROOLI_RESOURCES_CONFIG" >/dev/null 2>&1; then
                local resource_config
                resource_config=$(json::get_resource_config "$category" "$resource_name" "$VROOLI_RESOURCES_CONFIG")
                if [[ -n "$resource_config" && "$resource_config" != "{}" ]]; then
                    config_exists=true
                    break
                fi
            fi
        done
        
        if [[ "$config_exists" == "true" ]]; then
            log::success "✅ Resource configuration found"
        else
            log::warn "⚠️  Resource not configured in $VROOLI_RESOURCES_CONFIG"
        fi
    fi
}

#######################################
# Show repository information in resource status
# This adds project context to resource status displays
#######################################
resources::show_repository_info() {
    local repo_url repo_branch
    repo_url=$(repository::get_url 2>/dev/null || echo "")
    repo_branch=$(repository::get_branch 2>/dev/null || echo "")
    
    if [[ -n "$repo_url" ]]; then
        echo
        log::info "Project Repository:"
        log::info "  URL: $repo_url"
        log::info "  Branch: $repo_branch"
        log::info "  Issues: $repo_url/issues"
        log::info "  Documentation: $repo_url#readme"
    fi
}