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

    # Link the CLI globally using pnpm
    log::info "Linking Vrooli CLI globally..."
    if ! pnpm link --global; then
        log::warning "Failed to link CLI globally with pnpm, trying alternative method..."
        
        # Alternative: Create symlink in user's local bin
        local user_bin_dir="${HOME}/.local/bin"
        mkdir -p "${user_bin_dir}"
        
        if ln -sf "${cli_wrapper}" "${user_bin_dir}/vrooli"; then
            log::info "Created symlink in ${user_bin_dir}/vrooli"
            
            # Check if user's local bin is in PATH
            if [[ ":${PATH}:" != *":${user_bin_dir}:"* ]]; then
                log::warning "⚠️  ${user_bin_dir} is not in your PATH."
                log::warning "Add the following line to your ~/.bashrc or ~/.zshrc:"
                log::warning "export PATH=\"${user_bin_dir}:\$PATH\""
                log::warning "Then run: source ~/.bashrc (or source ~/.zshrc)"
            fi
        else
            log::error "Failed to create symlink"
            return "${ERROR_OPERATION_FAILED}"
        fi
    fi

    # Return to original directory
    cd - > /dev/null || true

    # Verify installation
    log::info "Verifying Vrooli CLI installation..."
    if command -v vrooli &> /dev/null; then
        vrooli --version || log::info "Version check failed, but CLI is available"
        log::success "✅ Vrooli CLI setup complete!"
        log::info "You can now use 'vrooli' command from anywhere."
    else
        log::warning "⚠️  Vrooli CLI installed but not found in PATH."
        log::warning "You may need to refresh your shell or add the install location to PATH."
        log::info "Try running: hash -r"
    fi

    return 0
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vrooli_cli::setup "$@"
fi