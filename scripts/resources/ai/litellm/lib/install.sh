#!/bin/bash
# LiteLLM installation functionality

# Get script directory
LITELLM_INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${LITELLM_INSTALL_DIR}/core.sh"
source "${LITELLM_INSTALL_DIR}/docker.sh"
source "${LITELLM_INSTALL_DIR}/../../../lib/resources/install-resource-cli.sh"

# Install LiteLLM resource
litellm::install() {
    local verbose="${1:-false}"
    local force="${2:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Installing LiteLLM resource"
    
    # Check Docker availability
    if ! command -v docker >/dev/null 2>&1; then
        log::error "Docker is not installed or not in PATH"
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        return 1
    fi
    
    # Initialize LiteLLM
    if ! litellm::init "$verbose"; then
        log::error "Failed to initialize LiteLLM"
        return 1
    fi
    
    # Add port to registry
    litellm::register_port "$verbose"
    
    # Pull Docker image
    [[ "$verbose" == "true" ]] && log::info "Pulling LiteLLM Docker image"
    if ! timeout "$LITELLM_INSTALL_TIMEOUT" docker pull "$LITELLM_IMAGE"; then
        log::error "Failed to pull LiteLLM Docker image"
        return 1
    fi
    
    # Install CLI
    [[ "$verbose" == "true" ]] && log::info "Installing LiteLLM CLI"
    install_resource_cli "litellm" "${LITELLM_RESOURCE_DIR}/cli.sh" || {
        log::error "Failed to install LiteLLM CLI"
        return 1
    }
    
    # Start the service
    [[ "$verbose" == "true" ]] && log::info "Starting LiteLLM service"
    if ! litellm::start "$verbose"; then
        log::warn "LiteLLM installed but failed to start"
        return 1
    fi
    
    # Test the installation
    [[ "$verbose" == "true" ]] && log::info "Testing LiteLLM installation"
    if litellm::test_connection 10 "$verbose"; then
        [[ "$verbose" == "true" ]] && log::info "LiteLLM installation successful"
        litellm::show_installation_summary
        return 0
    else
        log::warn "LiteLLM installed but health check failed"
        return 1
    fi
}

# Uninstall LiteLLM resource
litellm::uninstall() {
    local verbose="${1:-false}"
    local keep_data="${2:-false}"
    local force="${3:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Uninstalling LiteLLM resource"
    
    # Stop the service
    litellm::stop "$verbose" "$force"
    
    # Remove Docker image
    if [[ "$force" == "true" ]]; then
        [[ "$verbose" == "true" ]] && log::info "Removing Docker image"
        docker rmi "$LITELLM_IMAGE" >/dev/null 2>&1 || true
    fi
    
    # Remove data if not keeping
    if [[ "$keep_data" == "false" ]]; then
        [[ "$verbose" == "true" ]] && log::info "Removing data directories"
        rm -rf "$LITELLM_CONFIG_DIR" "$LITELLM_LOG_DIR" "$LITELLM_DATA_DIR"
    fi
    
    # Remove CLI
    [[ "$verbose" == "true" ]] && log::info "Removing LiteLLM CLI"
    uninstall_resource_cli "litellm" || true
    
    [[ "$verbose" == "true" ]] && log::info "LiteLLM uninstalled successfully"
    return 0
}

# Register port in port registry
litellm::register_port() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Registering LiteLLM port in registry"
    
    # Check if port registry exists
    local port_registry="${LITELLM_RESOURCE_DIR}/../../../resources/port_registry.sh"
    
    if [[ ! -f "$port_registry" ]]; then
        log::warn "Port registry not found"
        return 1
    fi
    
    # Check if port is already registered
    if grep -q "\\[\"litellm\"\\]" "$port_registry" 2>/dev/null; then
        [[ "$verbose" == "true" ]] && log::info "LiteLLM port already registered"
        return 0
    fi
    
    # Add port to registry (add to AI services section)
    local temp_file=$(mktemp)
    
    # Find the line with unstructured-io and add after it
    awk '
    /\["unstructured-io"\]="11450"/ {
        print
        print "    [\"litellm\"]=\"11435\"         # LiteLLM unified LLM proxy server"
        next
    }
    { print }
    ' "$port_registry" > "$temp_file"
    
    if [[ -s "$temp_file" ]] && mv "$temp_file" "$port_registry"; then
        [[ "$verbose" == "true" ]] && log::info "LiteLLM port registered successfully"
        return 0
    else
        rm -f "$temp_file"
        log::warn "Failed to register LiteLLM port"
        return 1
    fi
}

# Show installation summary
litellm::show_installation_summary() {
    cat <<EOF

🎉 LiteLLM Installation Complete!

Service Information:
  Status: Running
  URL: ${LITELLM_API_BASE}
  Port: ${LITELLM_PORT}
  Container: ${LITELLM_CONTAINER_NAME}

Configuration:
  Config: ${LITELLM_CONFIG_FILE}
  Environment: ${LITELLM_ENV_FILE}
  Data: ${LITELLM_DATA_DIR}
  Logs: ${LITELLM_LOG_DIR}

Next Steps:
  1. Configure API keys for your providers:
     • Edit ${LITELLM_ENV_FILE}
     • Or store in Vault: vault kv put secret/vrooli/openai api_key="sk-..."
     
  2. Test the installation:
     • resource-litellm status
     • resource-litellm list-models
     • resource-litellm test-model gpt-3.5-turbo
     
  3. Use with Claude Code:
     • export ANTHROPIC_BASE_URL="${LITELLM_API_BASE}"
     • export ANTHROPIC_AUTH_TOKEN="$(cat ${LITELLM_ENV_FILE} | grep LITELLM_MASTER_KEY | cut -d'=' -f2)"

For help: resource-litellm help

EOF
}

# Validate installation
litellm::validate() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Validating LiteLLM installation"
    
    # Check if CLI is installed
    if ! command -v resource-litellm >/dev/null 2>&1; then
        log::error "LiteLLM CLI not found"
        return 1
    fi
    
    # Check if container exists
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${LITELLM_CONTAINER_NAME}$"; then
        log::error "LiteLLM container not found"
        return 1
    fi
    
    # Check if service is running
    if ! litellm::is_running; then
        log::error "LiteLLM service is not running"
        return 1
    fi
    
    # Check connectivity
    if ! litellm::test_connection 10 "$verbose"; then
        log::error "LiteLLM health check failed"
        return 1
    fi
    
    # Check configuration files
    if [[ ! -f "$LITELLM_CONFIG_FILE" ]]; then
        log::error "LiteLLM configuration file not found"
        return 1
    fi
    
    if [[ ! -f "$LITELLM_ENV_FILE" ]]; then
        log::error "LiteLLM environment file not found"
        return 1
    fi
    
    [[ "$verbose" == "true" ]] && log::info "LiteLLM validation successful"
    return 0
}

# Upgrade LiteLLM
litellm::upgrade() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Upgrading LiteLLM"
    
    # Create backup
    local backup_dir
    backup_dir=$(litellm::backup "" "$verbose")
    
    [[ "$verbose" == "true" ]] && log::info "Backup created at: $backup_dir"
    
    # Update image and restart
    if litellm::update "$verbose"; then
        [[ "$verbose" == "true" ]] && log::info "LiteLLM upgraded successfully"
        return 0
    else
        log::error "LiteLLM upgrade failed"
        [[ "$verbose" == "true" ]] && log::info "Backup available at: $backup_dir"
        return 1
    fi
}

# Reset LiteLLM to defaults
litellm::reset() {
    local verbose="${1:-false}"
    local keep_data="${2:-true}"
    
    [[ "$verbose" == "true" ]] && log::info "Resetting LiteLLM to defaults"
    
    # Stop service
    litellm::stop "$verbose"
    
    # Remove configuration (but keep data by default)
    rm -f "$LITELLM_CONFIG_FILE" "$LITELLM_ENV_FILE"
    
    if [[ "$keep_data" == "false" ]]; then
        rm -rf "$LITELLM_DATA_DIR"/*
    fi
    
    # Recreate default configuration
    litellm::create_default_config "$verbose"
    
    # Restart service
    litellm::start "$verbose"
    
    [[ "$verbose" == "true" ]] && log::info "LiteLLM reset to defaults"
    return 0
}