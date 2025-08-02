#!/usr/bin/env bash
# Posix-compliant script to setup the project for native Linux development/production
set -euo pipefail

SETUP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../utils/pnpm_tools.sh"
# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../utils/flow.sh"

native_linux::setup_native_linux() {
    log::header "Setting up native Linux development/production..."

    # Setup pnpm and generate Prisma client
    pnpm_tools::setup
    
    # Configure kernel parameters for resources
    # Let the kernel config script itself determine if changes are needed and handle sudo availability
    local kernel_config_script="${SETUP_TARGET_DIR}/../kernel_config.sh"
    if [[ -f "$kernel_config_script" ]]; then
        bash "$kernel_config_script" configure || {
            log::warning "Kernel configuration failed - some resources may not work properly"
        }
    fi
    
    # Setup local resources if requested
    if [[ -n "${RESOURCES:-}" && "$RESOURCES" != "none" ]]; then
        # Handle special case for "enabled" option
        if [[ "$RESOURCES" == "enabled" ]]; then
            # Check if this is a first run (no config exists)
            if [[ ! -f "$HOME/.vrooli/resources.local.json" ]]; then
                log::info "First run detected - will install recommended resources"
                # For first run, install essential AI resource
                RESOURCES="ollama"
            else
                # Config exists, so resources::get_enabled_from_config will handle it
                log::info "Installing resources marked as enabled in configuration"
            fi
        fi
        
        log::header "🔧 Setting up local resources: $RESOURCES"
        
        local resources_script="${SETUP_TARGET_DIR}/../../../resources/index.sh"
        if [[ -f "$resources_script" ]]; then
            bash "$resources_script" \
                --action install \
                --resources "$RESOURCES" \
                --yes "${YES:-no}"
                
            # Auto-register Vrooli MCP server with Claude Code if both are available
            native_linux::setup_mcp_integration
        else
            log::error "Resource setup script not found: $resources_script"
            log::info "Skipping resource setup"
        fi
    else
        log::info "No local resources requested (use --resources to specify)"
    fi
}

#######################################
# Setup MCP integration between Vrooli and Claude Code
#######################################
native_linux::setup_mcp_integration() {
    # Check if Claude Code was installed during resource setup
    local claude_code_script="${SETUP_TARGET_DIR}/../../../resources/agents/claude-code/manage.sh"
    
    if [[ ! -f "$claude_code_script" ]]; then
        # No Claude Code management script found, skip MCP integration
        return 0
    fi
    
    log::info "🔗 Checking for MCP integration opportunities..."
    
    # Check if Claude Code is installed
    if ! bash "$claude_code_script" --action status >/dev/null 2>&1; then
        log::info "Claude Code not installed, skipping MCP auto-registration"
        return 0
    fi
    
    # Wait a moment for any services to start up after resource installation
    sleep 2
    
    # Check if Vrooli server is running (for local development)
    local vrooli_health_check="http://localhost:3000/mcp/health"
    if command -v curl >/dev/null 2>&1; then
        if curl -f -s --max-time 5 "$vrooli_health_check" >/dev/null 2>&1; then
            log::info "Vrooli server detected, attempting MCP auto-registration..."
            
            # Attempt auto-registration with user scope (good for development)
            if bash "$claude_code_script" --action register-mcp --scope user --yes "${YES:-no}" 2>/dev/null; then
                log::success "✅ Vrooli MCP server registered with Claude Code!"
                log::info "🎯 You can now use @vrooli in Claude Code to access Vrooli tools"
            else
                log::info "MCP auto-registration was attempted but may have failed"
                log::info "You can manually register with: $claude_code_script --action register-mcp"
            fi
        else
            log::info "Vrooli server not running yet, skipping MCP auto-registration"
            log::info "After starting Vrooli, you can register MCP with:"
            log::info "  $claude_code_script --action register-mcp"
        fi
    else
        log::info "curl not available, skipping MCP health check"
        log::info "You can manually register MCP with:"
        log::info "  $claude_code_script --action register-mcp"
    fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    native_linux::setup_native_linux "$@"
fi 