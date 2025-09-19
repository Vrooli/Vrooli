#!/bin/bash
# OpenCode Installation Functions

# Source common functions
source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::install::install_code_server() {
    opencode_detect_vscode
    if [[ -n "${VSCODE_COMMAND}" ]]; then
        return 0
    fi

    if ! command -v curl &>/dev/null; then
        log::error "curl is required to install code-server automatically"
        return 1
    fi

    local install_prefix="${HOME}/.local"
    local install_script
    install_script=$(mktemp)

    log::info "Downloading code-server installation script..."
    if ! curl -fsSL https://code-server.dev/install.sh -o "${install_script}"; then
        log::error "Failed to download code-server installer"
        rm -f "${install_script}"
        return 1
    fi

    log::info "Installing code-server to ${install_prefix} (standalone method)..."
    if ! sh "${install_script}" --method standalone --prefix "${install_prefix}" >/dev/null 2>&1; then
        log::error "code-server installation failed"
        rm -f "${install_script}"
        return 1
    fi

    rm -f "${install_script}"

    if [[ -x "${install_prefix}/bin/code-server" ]]; then
        export PATH="${install_prefix}/bin:${PATH}"
    fi

    opencode_detect_vscode

    if [[ -z "${VSCODE_COMMAND}" ]]; then
        log::warn "code-server installed but not detected on PATH. Add ${install_prefix}/bin to PATH and retry."
        return 1
    fi

    log::success "code-server installed successfully (${VSCODE_COMMAND})"
    log::info "Ensure ${install_prefix}/bin is on your PATH for future shells."
    return 0
}

opencode::install::execute() {
    log::info "Installing OpenCode VS Code extension..."

    opencode_detect_vscode
    if [[ -z "${VSCODE_COMMAND}" ]]; then
        log::warn "VS Code command not found. Attempting to install code-server automatically..."
        if ! opencode::install::install_code_server; then
            log::error "Unable to install code-server automatically. Install VS Code or code-server manually and rerun."
            return 1
        fi
    fi

    opencode_detect_vscode

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
        cat > "${OPENCODE_CONFIG_FILE}" <<EOF_CFG
{
  "provider": "${DEFAULT_MODEL_PROVIDER}",
  "chat_model": "${DEFAULT_CHAT_MODEL}",
  "completion_model": "${DEFAULT_COMPLETION_MODEL}",
  "port": ${OPENCODE_PORT},
  "auto_suggest": true,
  "enable_logging": true
}
EOF_CFG
        log::success "Default configuration created at ${OPENCODE_CONFIG_FILE}"
    fi

    log::success "OpenCode installation completed"
}

opencode::install::uninstall() {
    log::info "Uninstalling OpenCode VS Code extension..."

    opencode_detect_vscode

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
