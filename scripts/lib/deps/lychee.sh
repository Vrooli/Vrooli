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
source "${var_TRASH_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"

# Function to install Lychee for fast markdown link checking
lychee::install() {
    # Check if Lychee is already installed
    if system::is_command "lychee"; then
        log::info "Lychee is already installed"
        return
    fi

    log::header "🔗 Installing Lychee for fast markdown link checking"

    # Determine the installation directory based on sudo availability
    INSTALL_DIR="/usr/local/bin"
    local use_sudo=true

    # Check if we should use local installation
    if [ "${SUDO_MODE:-error}" = "skip" ] || ! sudo::can_use_sudo; then
        use_sudo=false
        log::info "Installing Lychee to user directory."
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
        fi
    fi

    # Determine architecture for static binary
    case "$(uname -m)" in
        x86_64) arch="x86_64" ;;
        aarch64|arm64) arch="aarch64" ;;
        *) log::error "Unsupported architecture: $(uname -m)" ;;
    esac

    # Get the latest release from GitHub API
    log::info "🔍 Fetching latest Lychee release information..."
    
    # Get the latest release info from GitHub API
    local latest_release_info
    if latest_release_info=$(curl -fsSL "https://api.github.com/repos/lycheeverse/lychee/releases/latest" 2>/dev/null); then
        log::info "Retrieved latest release information from GitHub API"
    else
        log::warning "Could not fetch latest release from GitHub API, using fallback versions"
        latest_release_info=""
    fi

    # List of Lychee versions to try (with their respective tag formats)
    local versions_to_try=()
    
    # Add latest version first if we got it from API
    if [[ -n "$latest_release_info" ]]; then
        local latest_tag
        latest_tag=$(echo "$latest_release_info" | jq -r '.tag_name')
        local latest_url
        latest_url=$(echo "$latest_release_info" | jq -r '.assets[] | select(.name | contains("x86_64-unknown-linux-gnu")) | .browser_download_url')
        
        if [[ -n "$latest_url" && "$latest_url" != "null" ]]; then
            versions_to_try+=("$latest_tag:$latest_url")
            log::info "Found latest release: $latest_tag"
        fi
    fi
    
    # Add fallback versions with old tag format (v prefix)
    local fallback_versions=("0.16.1" "0.16.0" "0.15.1" "0.15.0")
    for version in "${fallback_versions[@]}"; do
        local fallback_url="https://github.com/lycheeverse/lychee/releases/download/v${version}/lychee-${arch}-unknown-linux-gnu.tar.gz"
        versions_to_try+=("v${version}:$fallback_url")
    done

    for version_info in "${versions_to_try[@]}"; do
        IFS=':' read -r version download_url <<< "$version_info"
        log::header "📥 Attempting to download Lychee ${version} for ${arch}"
        tmpdir=$(mktemp -d)
        
        # Try downloading the release tarball
        
        if curl -fsSL "$download_url" | tar -xz -C "$tmpdir"; then
            # Move and make executable
            if [ "$use_sudo" = "true" ]; then
                sudo::exec_with_fallback "mv '$tmpdir/lychee' '${INSTALL_DIR}/lychee'"
                sudo::exec_with_fallback "chmod +x '${INSTALL_DIR}/lychee'"
            else
                mv "$tmpdir/lychee" "${INSTALL_DIR}/lychee"
                chmod +x "${INSTALL_DIR}/lychee"
            fi
            trash::safe_remove "$tmpdir" --no-confirm
            log::success "Lychee v${version} installed to ${INSTALL_DIR}"
            
            # Verify installation
            if command -v lychee >/dev/null 2>&1; then
                local installed_version
                installed_version=$(lychee --version | head -n1)
                log::success "Lychee installation verified: ${installed_version}"
            fi
            
            return
        else
            log::warning "Download of Lychee v${version} failed, trying next version"
            trash::safe_remove "$tmpdir" --no-confirm
        fi
    done

    log::error "All fallback Lychee installations failed"
    log::info "You can try installing manually with: curl -sSf https://raw.githubusercontent.com/lycheeverse/lychee/main/install.sh | sh"
    return 1
}

# Function to check if Lychee is working properly
lychee::verify() {
    if ! system::is_command "lychee"; then
        log::error "Lychee is not installed or not in PATH"
        return 1
    fi

    local version_output
    if version_output=$(lychee --version 2>&1); then
        log::success "Lychee is working: ${version_output}"
        return 0
    else
        log::error "Lychee is installed but not working properly"
        return 1
    fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    lychee::install "$@"
fi