#!/usr/bin/env bash
# Claude Code Installation Functions
# Handles installation and uninstallation of Claude Code CLI

#######################################
# Install Claude Code
#######################################
claude_code::install() {
    log::header "📦 Installing Claude Code"
    
    # Check if already installed
    if claude_code::is_installed && [[ "$FORCE" != "yes" ]]; then
        local version
        version=$(claude_code::get_version)
        log::warn "Claude Code is already installed (version: $version)"
        log::info "Use --force yes to reinstall"
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
    if npm install -g "$CLAUDE_PACKAGE"; then
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
    if npm uninstall -g "$CLAUDE_PACKAGE"; then
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