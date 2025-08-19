#!/usr/bin/env bash
# Cline Installation Functions

# Set script directory for sourcing
CLINE_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source required utilities
# shellcheck disable=SC1091
source "${CLINE_LIB_DIR}/common.sh"

#######################################
# Install Cline VS Code extension
# Returns:
#   0 on success, 1 on failure
#######################################
cline::install() {
    log::header "📦 Installing Cline"
    
    # Check if VS Code is installed
    if ! cline::check_vscode; then
        log::error "${MSG_CLINE_NO_VSCODE}"
        log::info "Please install VS Code from https://code.visualstudio.com/"
        return 1
    fi
    
    # Check if already installed
    if cline::is_installed && [[ "${FORCE:-false}" != "true" ]]; then
        local version
        version=$(cline::get_version)
        log::warn "${MSG_CLINE_ALREADY_INSTALLED} (version: $version)"
        log::info "Use --force to reinstall"
        return 0
    fi
    
    log::info "${MSG_CLINE_INSTALLING}"
    
    # Install the extension using VS Code CLI
    if ! timeout 60 code --install-extension "${CLINE_EXTENSION_ID}" --force 2>&1 | log::file; then
        log::error "${MSG_CLINE_INSTALL_FAILED}"
        return 1
    fi
    
    log::success "${MSG_CLINE_INSTALLED}"
    
    # Configure Cline
    if ! cline::configure; then
        log::warn "Cline installed but configuration incomplete"
        log::info "You may need to configure API keys manually"
    fi
    
    return 0
}

#######################################
# Configure Cline with API keys and settings
# Returns:
#   0 on success, 1 on failure
#######################################
cline::configure() {
    log::info "${MSG_CLINE_CONFIGURING}"
    
    # Create config directory
    if ! cline::create_config_dir; then
        log::error "Failed to create configuration directory"
        return 1
    fi
    
    # Get API key based on provider
    local provider="${CLINE_DEFAULT_PROVIDER}"
    local api_key
    api_key=$(cline::get_api_key "$provider")
    
    if [[ -z "$api_key" ]]; then
        log::warn "${MSG_CLINE_NO_API_KEY} for $provider"
        # Don't fail installation, just warn
    fi
    
    # Create settings file
    local settings_content
    settings_content=$(cat <<EOF
{
    "apiProvider": "${provider}",
    "apiKey": "${api_key}",
    "model": "${CLINE_DEFAULT_MODEL}",
    "temperature": ${CLINE_TEMPERATURE},
    "maxTokens": ${CLINE_MAX_TOKENS},
    "autoApprove": ${CLINE_AUTO_APPROVE},
    "experimentalMode": ${CLINE_EXPERIMENTAL_MODE},
    "ollamaBaseUrl": "${CLINE_OLLAMA_BASE_URL}",
    "useOllama": ${CLINE_USE_OLLAMA},
    "logLevel": "${CLINE_LOG_LEVEL}"
}
EOF
)
    
    # Write settings file
    echo "$settings_content" > "${CLINE_SETTINGS_FILE}"
    
    # Also configure VS Code settings
    local vscode_settings_dir="${HOME}/.config/Code/User"
    if [[ -d "$vscode_settings_dir" ]]; then
        local vscode_settings_file="${vscode_settings_dir}/settings.json"
        
        # Create or update VS Code settings
        if [[ -f "$vscode_settings_file" ]]; then
            # Backup existing settings
            cp "$vscode_settings_file" "${vscode_settings_file}.bak"
        else
            echo "{}" > "$vscode_settings_file"
        fi
        
        # Add Cline-specific settings using jq if available
        if command -v jq &>/dev/null; then
            jq --arg provider "$provider" \
               --arg key "$api_key" \
               --arg model "$CLINE_DEFAULT_MODEL" \
               --argjson temp "$CLINE_TEMPERATURE" \
               --argjson tokens "$CLINE_MAX_TOKENS" \
               '. + {
                   "claude-dev.apiProvider": $provider,
                   "claude-dev.apiKey": $key,
                   "claude-dev.model": $model,
                   "claude-dev.temperature": $temp,
                   "claude-dev.maxTokens": $tokens
               }' "$vscode_settings_file" > "${vscode_settings_file}.tmp" && \
               mv "${vscode_settings_file}.tmp" "$vscode_settings_file"
        fi
    fi
    
    log::success "${MSG_CLINE_CONFIGURED}"
    return 0
}

#######################################
# Uninstall Cline extension
# Returns:
#   0 on success, 1 on failure
#######################################
cline::uninstall() {
    log::header "${MSG_CLINE_UNINSTALLING}"
    
    if ! cline::is_installed; then
        log::warn "Cline is not installed"
        return 0
    fi
    
    # Uninstall the extension
    if ! code --uninstall-extension "${CLINE_EXTENSION_ID}" 2>&1 | log::file; then
        log::error "Failed to uninstall Cline extension"
        return 1
    fi
    
    # Optionally remove configuration
    if [[ "${REMOVE_CONFIG:-false}" == "true" ]]; then
        rm -rf "${CLINE_HOME}"
        log::info "Configuration removed"
    fi
    
    log::success "Cline uninstalled successfully"
    return 0
}

#######################################
# Update Cline extension
# Returns:
#   0 on success, 1 on failure
#######################################
cline::update() {
    log::header "🔄 Updating Cline"
    
    if ! cline::is_installed; then
        log::warn "Cline is not installed. Installing..."
        return cline::install
    fi
    
    # Force reinstall to update
    FORCE=true cline::install
}