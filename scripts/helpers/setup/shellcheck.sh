#!/usr/bin/env bash
set -euo pipefail

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/flow.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/system.sh"

# Function to install ShellCheck for shell script linting
shellcheck::install() {
    # Check if ShellCheck is already installed
    if system::is_command "shellcheck"; then
        log::info "ShellCheck is already installed"
        return
    fi

    log::header "🔍 Installing ShellCheck for shell linting"

    # First attempt: install via system package manager
    if system::install_pkg shellcheck; then
        log::success "ShellCheck installed via package manager"
        return
    else
        log::warning "Package manager install failed or timed out, falling back to GitHub releases"
    fi

    # Determine the installation directory and command prefix based on sudo availability
    INSTALL_DIR="/usr/local/bin"
    MV_CMD="sudo mv"
    CHMOD_CMD="sudo chmod"

    if ! flow::can_run_sudo; then
        log::warning "Sudo not available or skipped. Installing ShellCheck to user directory."
        INSTALL_DIR="$HOME/.local/bin"
        MV_CMD="mv"
        CHMOD_CMD="chmod"

        # Ensure the local bin directory exists
        mkdir -p "$INSTALL_DIR"

        # Ensure the local bin directory is in PATH
        if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
            export PATH="$INSTALL_DIR:$PATH"
            log::info "Added $INSTALL_DIR to PATH for current session."
            # Consider adding this to shell profile (e.g., ~/.bashrc or ~/.profile) for persistence
            # echo 'export PATH=\"$HOME/.local/bin:$PATH\"' >> ~/.bashrc # Example for bash
        fi
    fi

    # Determine architecture for static binary
    case "$(uname -m)" in
        x86_64) arch="x86_64" ;;
        aarch64|arm64) arch="aarch64" ;;
        *) log::error "Unsupported architecture: $(uname -m)" ;;
    esac

    # List of ShellCheck versions to try (newest first)
    fallback_versions=("0.10.0" "0.9.0" "0.8.0")
    for version in "${fallback_versions[@]}"; do
        log::header "📥 Attempting to download ShellCheck v${version} for ${arch}"
        tmpdir=$(mktemp -d)
        if curl -fsSL "https://github.com/koalaman/shellcheck/releases/download/v${version}/shellcheck-v${version}.linux.${arch}.tar.xz" \
               | tar -xJ -C "$tmpdir"; then
            # Move and make executable using determined command and path
            ${MV_CMD} "$tmpdir/shellcheck-v${version}/shellcheck" "${INSTALL_DIR}/shellcheck"
            ${CHMOD_CMD} +x "${INSTALL_DIR}/shellcheck"
            rm -rf "$tmpdir"
            success "ShellCheck v${version} installed to ${INSTALL_DIR}"
            return
        else
            log::warning "Download of ShellCheck v${version} failed, trying next version"
            rm -rf "$tmpdir"
        fi
    done

    log::error "All fallback ShellCheck installations failed"
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    shellcheck::install "$@"
fi
