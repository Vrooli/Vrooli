#!/bin/bash
# OpenCode Installation Functions

# Source common functions
source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::install::execute() {
    log::info "Installing OpenCode VS Code extension..."
    
    # Check if VS Code is available
    if [[ -z "${VSCODE_COMMAND}" ]]; then
        log::error "VS Code or code-server not found. Please install VS Code first."
        return 1
    fi
    
    # Ensure directories exist
    opencode_ensure_dirs
    
    # Install the Twinny extension
    log::info "Installing Twinny AI extension (${OPENCODE_EXTENSION_ID})..."
    if ${VSCODE_COMMAND} --install-extension "${OPENCODE_EXTENSION_ID}" --force; then
        log::success "Twinny extension installed successfully"
    else
        log::error "Failed to install Twinny extension"
        return 1
    fi
    
    # Create default configuration if it doesn't exist
    if [[ ! -f "${OPENCODE_CONFIG_FILE}" ]]; then
        log::info "Creating default configuration..."
        cat > "${OPENCODE_CONFIG_FILE}" <<EOF
{
  "provider": "${DEFAULT_MODEL_PROVIDER}",
  "chat_model": "${DEFAULT_CHAT_MODEL}",
  "completion_model": "${DEFAULT_COMPLETION_MODEL}",
  "port": ${OPENCODE_PORT},
  "auto_suggest": true,
  "enable_logging": true
}
EOF
        log::success "Default configuration created at ${OPENCODE_CONFIG_FILE}"
    fi
    
    log::success "OpenCode installation completed"
}

opencode::install::uninstall() {
    log::info "Uninstalling OpenCode VS Code extension..."
    
    # Uninstall the extension
    if [[ -n "${VSCODE_COMMAND}" ]]; then
        log::info "Uninstalling Twinny extension..."
        if ${VSCODE_COMMAND} --uninstall-extension "${OPENCODE_EXTENSION_ID}"; then
            log::success "Twinny extension uninstalled successfully"
        else
            log::warning "Failed to uninstall extension or extension was not installed"
        fi
    fi
    
    # Remove configuration and data
    if [[ -d "${OPENCODE_DATA_DIR}" ]]; then
        log::info "Removing OpenCode data directory..."
        rm -rf "${OPENCODE_DATA_DIR}"
        log::success "OpenCode data directory removed"
    fi
    
    log::success "OpenCode uninstallation completed"
}