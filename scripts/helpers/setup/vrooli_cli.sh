#!/usr/bin/env bash
# This script sets up the Vrooli CLI for local development and production environments.
# It builds the CLI package and makes it available system-wide.

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/exit_codes.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/system.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/var.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/nodejs.sh"

vrooli_cli::setup() {
    log::header "Setting up the Vrooli CLI..."

    # Check if vrooli CLI is already installed and working
    if command -v vrooli &> /dev/null; then
        log::info "Vrooli CLI is already installed. Verifying version:"
        vrooli --version || true
        log::success "Vrooli CLI setup check complete."
        return 0
    fi

    log::info "Vrooli CLI not found. Proceeding with installation."

    # Ensure we're in the project root
    local project_root="${var_ROOT_DIR}"
    if [[ ! -d "${project_root}/packages/cli" ]]; then
        log::error "CLI package not found at ${project_root}/packages/cli"
        return "${ERROR_FILE_NOT_FOUND}"
    fi

    # Navigate to CLI package directory
    cd "${project_root}/packages/cli" || {
        log::error "Failed to navigate to CLI package directory"
        return "${ERROR_DIRECTORY_NOT_FOUND}"
    }

    # Ensure Node.js is available before building
    if ! command -v node >/dev/null 2>&1; then
        log::info "Node.js not found. Sourcing Node.js environment..."
        nodejs::source_environment || {
            log::error "Failed to source Node.js environment"
            return "${ERROR_DEPENDENCY_MISSING}"
        }
    fi

    # Build the CLI package
    log::info "Building Vrooli CLI package..."
    if ! pnpm run build; then
        log::error "Failed to build CLI package"
        return "${ERROR_BUILD_FAILED}"
    fi

    # Ensure the dist/index.js exists and is executable
    if [[ ! -f "dist/index.js" ]]; then
        log::error "CLI build output not found at dist/index.js"
        return "${ERROR_FILE_NOT_FOUND}"
    fi

    chmod +x dist/index.js

    # Create a wrapper script for the CLI
    local cli_wrapper="${project_root}/packages/cli/bin/vrooli"
    mkdir -p "$(dirname "${cli_wrapper}")"
    
    cat > "${cli_wrapper}" << EOF
#!/usr/bin/env bash
# Vrooli CLI wrapper script
# Use readlink to resolve the actual path when called through a symlink
REAL_PATH="\$(readlink -f "\${BASH_SOURCE[0]}")"
SCRIPT_DIR="\$(cd "\$(dirname "\${REAL_PATH}")" && pwd)"
CLI_DIR="\$(cd "\${SCRIPT_DIR}/.." && pwd)"

# Execute the Node.js CLI with proper environment
exec node "\${CLI_DIR}/dist/index.js" "\$@"
EOF

    chmod +x "${cli_wrapper}"

    # Detect if running with sudo and get the actual user
    local actual_user=""
    local actual_home=""
    local user_bin_dir=""
    
    if [[ -n "${SUDO_USER:-}" ]]; then
        # Running with sudo
        actual_user="${SUDO_USER}"
        actual_home=$(eval echo "~${actual_user}")
        log::info "Running with sudo. Installing CLI for user: ${actual_user}"
    else
        # Running as regular user
        actual_user="${USER}"
        actual_home="${HOME}"
    fi
    
    # Try multiple installation methods in order of preference
    local install_success=false
    
    # Method 1: Try pnpm global install (might fail due to permissions or missing setup)
    log::info "Attempting global installation with pnpm..."
    if pnpm link --global 2>/dev/null; then
        install_success=true
        log::success "Successfully linked CLI globally with pnpm"
    else
        log::info "pnpm global link failed, trying alternative methods..."
    fi
    
    # Method 2: Install to actual user's local bin directory
    if [[ "${install_success}" != "true" ]]; then
        user_bin_dir="${actual_home}/.local/bin"
        log::info "Installing CLI to user's local bin: ${user_bin_dir}"
        
        # Create directory with proper ownership
        if [[ -n "${SUDO_USER:-}" ]]; then
            # Running with sudo - create as the actual user
            sudo -u "${actual_user}" mkdir -p "${user_bin_dir}"
        else
            mkdir -p "${user_bin_dir}"
        fi
        
        # Create symlink
        if ln -sf "${cli_wrapper}" "${user_bin_dir}/vrooli"; then
            log::info "Created symlink in ${user_bin_dir}/vrooli"
            
            # Fix ownership if running with sudo
            if [[ -n "${SUDO_USER:-}" ]]; then
                chown -h "${actual_user}:${actual_user}" "${user_bin_dir}/vrooli"
            fi
            
            install_success=true
            
            # Check if user's local bin is in PATH
            if [[ ":${PATH}:" != *":${user_bin_dir}:"* ]]; then
                log::warning "⚠️  ${user_bin_dir} is not in PATH."
                log::info "To add it to PATH, run as ${actual_user}:"
                log::info "echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
                log::info "source ~/.bashrc"
                
                # Try to add to user's bashrc if we have permission
                local bashrc_file="${actual_home}/.bashrc"
                if [[ -f "${bashrc_file}" ]]; then
                    if ! grep -q "/.local/bin" "${bashrc_file}"; then
                        if [[ -n "${SUDO_USER:-}" ]]; then
                            # Add as the actual user
                            sudo -u "${actual_user}" bash -c "echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> '${bashrc_file}'"
                            log::success "Added ${user_bin_dir} to ${actual_user}'s PATH in .bashrc"
                        elif [[ -w "${bashrc_file}" ]]; then
                            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "${bashrc_file}"
                            log::success "Added ${user_bin_dir} to PATH in .bashrc"
                        fi
                    fi
                fi
            fi
        else
            log::error "Failed to create symlink in ${user_bin_dir}"
        fi
    fi
    
    # Method 3: If still not successful, try /usr/local/bin (requires sudo)
    if [[ "${install_success}" != "true" ]] && [[ -w "/usr/local/bin" || -n "${SUDO_USER:-}" ]]; then
        log::info "Attempting installation to /usr/local/bin..."
        if ln -sf "${cli_wrapper}" "/usr/local/bin/vrooli"; then
            install_success=true
            log::success "Created symlink in /usr/local/bin/vrooli"
        fi
    fi
    
    if [[ "${install_success}" != "true" ]]; then
        log::error "Failed to install CLI to any standard location"
        return "${ERROR_OPERATION_FAILED}"
    fi

    # Return to original directory
    cd - > /dev/null || true

    # Verify installation
    log::info "Verifying Vrooli CLI installation..."
    
    # Check if CLI is available in current shell
    local cli_available=false
    if command -v vrooli &> /dev/null; then
        cli_available=true
    elif [[ -x "${user_bin_dir}/vrooli" ]]; then
        # CLI exists but not in current PATH
        log::info "CLI installed at ${user_bin_dir}/vrooli but not in current PATH"
    elif [[ -x "/usr/local/bin/vrooli" ]]; then
        # CLI exists in /usr/local/bin
        cli_available=true
    fi
    
    if [[ "${cli_available}" == "true" ]]; then
        vrooli --version 2>/dev/null || log::info "Version check failed, but CLI is available"
        log::success "✅ Vrooli CLI setup complete!"
        if [[ -n "${SUDO_USER:-}" ]]; then
            log::info "CLI installed for user: ${actual_user}"
            log::info "The user may need to reload their shell or run: source ~/.bashrc"
        else
            log::info "You can now use 'vrooli' command from anywhere."
        fi
    else
        log::warning "⚠️  Vrooli CLI installed but not found in current PATH."
        if [[ -n "${SUDO_USER:-}" ]]; then
            log::info "To use the CLI, ${actual_user} should:"
            log::info "1. Exit this sudo session"
            log::info "2. Run: source ~/.bashrc"
            log::info "3. Run: vrooli --version"
        else
            log::warning "You may need to refresh your shell or add the install location to PATH."
            log::info "Try running: source ~/.bashrc"
        fi
    fi

    return 0
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vrooli_cli::setup "$@"
fi