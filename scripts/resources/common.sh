#!/usr/bin/env bash
set -euo pipefail

# Common utilities for local resource setup and management
# This file provides shared functionality for all resource setup scripts

RESOURCES_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
MAIN_DIR="${RESOURCES_DIR}/../main"

# Source required utilities
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/log.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/flow.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/ports.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/system.sh"

# Resource configuration paths
readonly VROOLI_CONFIG_DIR="${HOME}/.vrooli"
readonly VROOLI_RESOURCES_CONFIG="${VROOLI_CONFIG_DIR}/resources.local.json"

# Configuration manager script path
readonly CONFIG_MANAGER_SCRIPT="${RESOURCES_DIR}/config-manager.js"

# Source the port registry for centralized port management
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/port-registry.sh"

# Common default ports for local resources (from port registry)
declare -A DEFAULT_PORTS
for resource in "${!RESOURCE_PORTS[@]}"; do
    DEFAULT_PORTS["$resource"]="${RESOURCE_PORTS[$resource]}"
done

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
    
    local pids
    pids=$(ports::get_listening_pids "$port" 2>/dev/null) || return 1
    [[ -n "$pids" ]]
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
# Validate and get safe port for resource
# Arguments:
#   $1 - resource name
#   $2 - requested port (optional)
# Returns:
#   0 if port is safe, 1 if conflicts exist
# Outputs:
#   The validated port number
#######################################
resources::validate_port() {
    local resource="$1"
    local requested_port="${2:-}"
    
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
        
        # Suggest alternative
        local category=""
        case "$resource" in
            ollama|localai) category="AI" ;;
            n8n|node-red) category="automation" ;;
            minio|ipfs) category="storage" ;;
            puppeteer|playwright) category="agents" ;;
        esac
        
        log::info "Consider using a different port for $resource ($category service)"
        return 1
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
        # Use jq to merge configurations if available
        resource_config=$(jq -n \
            --arg baseUrl "$base_url" \
            --argjson additional "$additional_config" \
            '{
                enabled: true,
                baseUrl: $baseUrl,
                healthCheck: {
                    intervalMs: 60000,
                    timeoutMs: 5000
                }
            } + $additional')
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
        "rm -f \"$temp_config\"" \
        1
    
    # Update configuration atomically
    if jq \
        --arg category "$category" \
        --arg resource "$resource_name" \
        --argjson config "$resource_config" \
        '.services[$category][$resource] = $config' \
        "$VROOLI_RESOURCES_CONFIG" > "$temp_config"; then
        
        # Validate the result
        if jq . "$temp_config" > /dev/null 2>&1; then
            mv "$temp_config" "$VROOLI_RESOURCES_CONFIG"
            log::success "Updated resource configuration for $category/$resource_name"
            
            # Add rollback action
            resources::add_rollback_action \
                "Remove $category/$resource_name configuration (legacy)" \
                "jq 'del(.services.$category.$resource_name)' \"$VROOLI_RESOURCES_CONFIG\" > \"$temp_config\" && mv \"$temp_config\" \"$VROOLI_RESOURCES_CONFIG\"" \
                10
            
            return 0
        else
            rm -f "$temp_config"
            resources::handle_error "Generated configuration is invalid JSON" "config"
            return 1
        fi
    else
        rm -f "$temp_config"
        resources::handle_error "Failed to update configuration with jq" "config"
        return 1
    fi
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
        'del(.services[$category][$resource])' \
        "$VROOLI_RESOURCES_CONFIG" > "$temp_config"; then
        
        mv "$temp_config" "$VROOLI_RESOURCES_CONFIG"
        log::success "Removed resource configuration for $category/$resource_name"
    else
        rm -f "$temp_config"
        resources::handle_error "Failed to remove configuration with jq" "config"
        return 1
    fi
}

#######################################
# Initialize an empty resources configuration (legacy fallback)
#######################################
resources::init_config_legacy() {
    if [[ ! -f "$VROOLI_RESOURCES_CONFIG" ]]; then
        log::info "Creating initial resources configuration: $VROOLI_RESOURCES_CONFIG"
        cat > "$VROOLI_RESOURCES_CONFIG" << 'EOF'
{
  "version": "1.0.0",
  "enabled": true,
  "services": {
    "ai": {},
    "automation": {},
    "storage": {},
    "agents": {}
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
# Check if user has sudo privileges
# Returns:
#   0 if user can sudo, 1 otherwise
#######################################
resources::can_sudo() {
    sudo -n true 2>/dev/null
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
    
    if ! resources::can_sudo; then
        log::error "Sudo privileges required to install systemd service"
        return 1
    fi
    
    log::info "Installing systemd service: $service_name"
    
    # Write service file
    echo "$service_content" | sudo tee "$service_file" >/dev/null
    
    # Reload systemd and enable service
    sudo systemctl daemon-reload
    sudo systemctl enable "$service_name"
    
    log::success "Systemd service $service_name installed and enabled"
}

#######################################
# Start a systemd service
# Arguments:
#   $1 - service name
#######################################
resources::start_service() {
    local service_name="$1"
    
    if ! resources::can_sudo; then
        log::error "Sudo privileges required to start systemd service"
        return 1
    fi
    
    log::info "Starting service: $service_name"
    sudo systemctl start "$service_name"
    log::success "Service $service_name started"
}

#######################################
# Stop a systemd service
# Arguments:
#   $1 - service name
#######################################
resources::stop_service() {
    local service_name="$1"
    
    if ! resources::can_sudo; then
        log::error "Sudo privileges required to stop systemd service"
        return 1
    fi
    
    log::info "Stopping service: $service_name"
    sudo systemctl stop "$service_name" 2>/dev/null || true
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
    
    # Check configuration
    if [[ -f "$VROOLI_RESOURCES_CONFIG" ]] && system::is_command "jq"; then
        local config_exists
        config_exists=$(jq -r --arg name "$resource_name" \
            '.services.ai[$name] // .services.automation[$name] // .services.storage[$name] // .services.agents[$name] // empty' \
            "$VROOLI_RESOURCES_CONFIG" 2>/dev/null)
        
        if [[ -n "$config_exists" && "$config_exists" != "null" ]]; then
            log::success "✅ Resource configuration found"
        else
            log::warn "⚠️  Resource not configured in $VROOLI_RESOURCES_CONFIG"
        fi
    fi
}