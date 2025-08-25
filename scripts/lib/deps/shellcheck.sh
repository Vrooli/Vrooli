#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
LIB_DEPS_DIR="${APP_ROOT}/scripts/lib/deps"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"

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

    # Determine the installation directory based on sudo availability
    INSTALL_DIR="/usr/local/bin"
    local use_sudo=true

    # Check if we should use local installation
    if [ "${SUDO_MODE:-error}" = "skip" ] || ! sudo::can_use_sudo; then
        use_sudo=false
        log::info "Installing ShellCheck to user directory."
        INSTALL_DIR="$HOME/.local/bin"

        # Ensure the local bin directory exists
        if sudo::is_running_as_sudo; then
            sudo::mkdir_as_user "$INSTALL_DIR"
        else
            mkdir -p "$INSTALL_DIR"
        fi

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
            # Move and make executable
            if [ "$use_sudo" = "true" ]; then
                sudo::exec_with_fallback "mv '$tmpdir/shellcheck-v${version}/shellcheck' '${INSTALL_DIR}/shellcheck'"
                sudo::exec_with_fallback "chmod +x '${INSTALL_DIR}/shellcheck'"
            else
                mv "$tmpdir/shellcheck-v${version}/shellcheck" "${INSTALL_DIR}/shellcheck"
                chmod +x "${INSTALL_DIR}/shellcheck"
            fi
            trash::safe_remove "$tmpdir" --no-confirm
            log::success "ShellCheck v${version} installed to ${INSTALL_DIR}"
            return
        else
            log::warning "Download of ShellCheck v${version} failed, trying next version"
            trash::safe_remove "$tmpdir" --no-confirm
        fi
    done

    log::error "All fallback ShellCheck installations failed"
    return 1
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    shellcheck::install "$@"
fi
