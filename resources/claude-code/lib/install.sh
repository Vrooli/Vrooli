#!/usr/bin/env bash
# Claude Code Installation Functions
# Handles installation and uninstallation of Claude Code CLI

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLAUDE_CODE_LIB_DIR="${APP_ROOT}/resources/claude-code/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CLAUDE_CODE_LIB_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/resource-cli-helper.sh" 2>/dev/null || true

#######################################
# Install Claude Code
#######################################
claude_code::install() {
    log::header "📦 Installing Claude Code"
    
    # Initialize FORCE with default value if not set
    FORCE="${FORCE:-false}"
    
    # Check if already installed
    if claude_code::is_installed && [[ "$FORCE" != "true" ]]; then
        local version
        version=$(claude_code::get_version)
        log::warn "Claude Code is already installed (version: $version)"
        log::info "Use --force to reinstall"
        return 0
    fi
    
    # Check Node.js requirements
    log::info "Checking prerequisites..."
    if ! claude_code::check_node_version; then
        log::error "Node.js $MIN_NODE_VERSION or newer is required"
        log::info "Please install Node.js from https://nodejs.org/"
        return 1
    fi
    
    local node_version
    node_version=$(node --version)
    log::success "✓ Node.js $node_version detected"
    
    # Check npm
    if ! system::is_command npm; then
        log::error "npm is required but not found"
        log::info "npm should be installed with Node.js"
        return 1
    fi
    
    local npm_version
    npm_version=$(npm --version)
    log::success "✓ npm $npm_version detected"
    
    # Install Claude Code globally
    log::info "Installing Claude Code globally..."
    if sudo::exec_as_actual_user "npm install -g $CLAUDE_PACKAGE"; then
        log::success "✓ Claude Code installed successfully"
    else
        log::error "Failed to install Claude Code"
        return 1
    fi
    
    # Verify installation
    if claude_code::is_installed; then
        local version
        version=$(claude_code::get_version)
        log::success "✓ Claude Code $version is ready to use"
        
        # Update resource configuration for CLI tool
        if resources::update_cli_config "agents" "$RESOURCE_NAME" "claude" \
            '{"requiresAuth":true,"version":"'$version'","healthCommand":"health-check"}'; then
            log::success "✓ Resource configuration updated"
        else
            log::warn "⚠️  Failed to update resource configuration"
        fi
        
        # Install CLI command
        if "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "${CLAUDE_CODE_SCRIPT_DIR:-${APP_ROOT}/resources/claude-code}" 2>/dev/null; then
            log::success "✓ CLI command 'resource-claude-code' installed"
        else
            log::warn "⚠️  CLI installation failed (non-critical)"
        fi
        
        # Show next steps
        claude_code::install_next_steps
        
        return 0
    else
        log::error "Installation verification failed"
        return 1
    fi
}

#######################################
# Uninstall Claude Code
#######################################
claude_code::uninstall() {
    log::header "🗑️  Uninstalling Claude Code"
    
    if ! claude_code::is_installed; then
        log::warn "Claude Code is not installed"
        return 0
    fi
    
    # Confirm uninstallation
    if ! confirm "Remove Claude Code CLI?"; then
        log::info "Uninstallation cancelled"
        return 0
    fi
    
    # Uninstall globally
    log::info "Removing Claude Code..."
    if sudo::exec_as_actual_user "npm uninstall -g $CLAUDE_PACKAGE"; then
        log::success "✓ Claude Code removed successfully"
    else
        log::error "Failed to uninstall Claude Code"
        return 1
    fi
    
    # Update configuration
    if resources::remove_from_config "agents" "$RESOURCE_NAME"; then
        log::success "✓ Resource configuration updated"
    fi
    
    return 0
}