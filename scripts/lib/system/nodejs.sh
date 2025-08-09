#!/usr/bin/env bash
# Install Node.js via platform-appropriate version manager
set -euo pipefail

LIB_SYSTEM_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${LIB_SYSTEM_DIR}/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$var_LIB_SYSTEM_DIR/system_commands.sh"
# shellcheck disable=SC1091
source "$var_LIB_UTILS_DIR/flow.sh"

# Default Node.js version if not specified
DEFAULT_NODE_VERSION="20"

# Retry configuration - only set if not already defined
if [[ -z "${MAX_RETRY_ATTEMPTS:-}" ]]; then
    readonly MAX_RETRY_ATTEMPTS=3
    readonly RETRY_DELAY=2
fi

nodejs::download_with_retry() {
    local url="$1"
    local attempt=1
    
    while [[ $attempt -le $MAX_RETRY_ATTEMPTS ]]; do
        log::info "Download attempt $attempt of $MAX_RETRY_ATTEMPTS..."
        
        if curl -fsSL "$url"; then
            return 0
        fi
        
        if [[ $attempt -lt $MAX_RETRY_ATTEMPTS ]]; then
            log::warning "Download failed, retrying in ${RETRY_DELAY} seconds..."
            sleep "$RETRY_DELAY"
        fi
        
        ((attempt++))
    done
    
    log::error "Download failed after $MAX_RETRY_ATTEMPTS attempts"
    return 1
}

nodejs::detect_platform() {
    local uname_out
    uname_out="$(uname -s)"
    case "${uname_out}" in
        Linux*)     echo "linux";;
        Darwin*)    echo "mac";;
        CYGWIN*|MINGW*|MSYS*) echo "windows";;
        *)          echo "unknown";;
    esac
}

nodejs::get_nvm_dir() {
    echo "${NVM_DIR:-$HOME/.nvm}"
}

nodejs::verify_installation() {
    local expected_version="${1:-}"
    
    if command -v node >/dev/null 2>&1; then
        local installed_version
        installed_version=$(node --version)
        log::success "Node.js installed successfully: $installed_version"
        
        # Also check npm
        local npm_version
        npm_version=$(npm --version 2>/dev/null || echo "not found")
        log::info "npm version: $npm_version"
        
        # Check if version matches expected (if provided)
        if [[ -n "$expected_version" ]] && [[ "$expected_version" != "lts" ]] && [[ "$expected_version" != "latest" ]]; then
            if [[ "$installed_version" == v"$expected_version"* ]]; then
                log::info "Installed version matches requested version"
            else
                log::warning "Installed version ($installed_version) differs from requested (v$expected_version)"
            fi
        fi
        
        return 0
    else
        log::error "Node.js installation verification failed"
        return 1
    fi
}

nodejs::ensure_nvm() {
    local nvm_dir
    nvm_dir=$(nodejs::get_nvm_dir)
    
    # Check if nvm is already installed
    if [[ -s "$nvm_dir/nvm.sh" ]]; then
        log::info "nvm is already installed at $nvm_dir"
        return 0
    fi
    
    log::info "Installing nvm..."
    
    # Download and install nvm with retry
    if ! nodejs::download_with_retry "https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh" | bash; then
        log::error "Failed to download nvm installer"
        return 1
    fi
    
    # Verify installation
    if [[ ! -s "$nvm_dir/nvm.sh" ]]; then
        log::error "Failed to install nvm"
        return 1
    fi
    
    log::success "nvm installed successfully"
    return 0
}

nodejs::source_nvm() {
    local nvm_dir
    nvm_dir=$(nodejs::get_nvm_dir)
    
    # Source nvm
    export NVM_DIR="$nvm_dir"
    # shellcheck disable=SC1091
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
    # shellcheck disable=SC1091
    [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"
}

nodejs::install_node() {
    local version="${1:-$DEFAULT_NODE_VERSION}"
    
    log::header "Installing Node.js..."
    
    # Ensure nvm is installed
    nodejs::ensure_nvm || return 1
    
    # Source nvm in current shell
    nodejs::source_nvm
    
    # Check if node is already installed
    if command -v node >/dev/null 2>&1; then
        local current_version
        current_version=$(node --version)
        log::info "Node.js is already installed: $current_version"
        
        # Check if it's the version we want
        if [[ "$current_version" == v"$version"* ]]; then
            log::success "Node.js $version is already installed"
            return 0
        fi
    fi
    
    # Install the specified Node version
    log::info "Installing Node.js $version..."
    nvm install "$version" || {
        log::error "Failed to install Node.js $version"
        return 1
    }
    
    # Use the installed version
    nvm use "$version" || {
        log::error "Failed to switch to Node.js $version"
        return 1
    }
    
    # Set as default
    nvm alias default "$version" || {
        log::error "Failed to set Node.js $version as default"
        return 1
    }
    
    # Verify installation
    nodejs::verify_installation "$version"
}

nodejs::install_node_volta() {
    local version="${1:-$DEFAULT_NODE_VERSION}"
    
    log::header "Installing Node.js via Volta..."
    
    # Check if Volta is already installed
    if ! command -v volta >/dev/null 2>&1; then
        log::info "Volta not found. Installing Volta..."
        
        # Check if Homebrew is available (Mac)
        if command -v brew >/dev/null 2>&1; then
            brew install volta || {
                log::error "Failed to install Volta via Homebrew"
                return 1
            }
        else
            # Fallback to curl installation with retry
            if ! nodejs::download_with_retry "https://get.volta.sh" | bash; then
                log::error "Failed to install Volta"
                return 1
            fi
        fi
        
        # Source Volta
        export VOLTA_HOME="$HOME/.volta"
        export PATH="$VOLTA_HOME/bin:$PATH"
    fi
    
    # Check if Node is already installed via Volta
    if volta list node 2>/dev/null | grep -q "node@"; then
        local current_version
        current_version=$(node --version 2>/dev/null || echo "unknown")
        log::info "Node.js is already installed via Volta: $current_version"
    fi
    
    # Install Node.js
    log::info "Installing Node.js $version via Volta..."
    volta install "node@$version" || {
        log::error "Failed to install Node.js $version"
        return 1
    }
    
    # Verify installation
    nodejs::verify_installation "$version"
}

nodejs::install_node_windows() {
    local version="${1:-lts}"  # Windows nvm uses "lts" instead of version numbers
    
    log::header "Installing Node.js via nvm-windows..."
    
    # Check if nvm-windows is installed
    if ! command -v nvm >/dev/null 2>&1; then
        log::error "nvm-windows is not installed. Please install it from https://github.com/coreybutler/nvm-windows"
        log::info "Run the Windows setup script or install manually"
        return 1
    fi
    
    # Install Node.js
    log::info "Installing Node.js $version..."
    nvm install "$version" || {
        log::error "Failed to install Node.js $version"
        return 1
    }
    
    # Use the installed version
    nvm use "$version" || {
        log::error "Failed to switch to Node.js $version"
        return 1
    }
    
    # Verify installation
    nodejs::verify_installation "$version"
}

nodejs::source_environment() {
    local platform
    platform=$(nodejs::detect_platform)
    
    case "$platform" in
        linux)
            nodejs::source_nvm
            ;;
        mac)
            # Source Volta
            export VOLTA_HOME="${VOLTA_HOME:-$HOME/.volta}"
            export PATH="$VOLTA_HOME/bin:$PATH"
            ;;
        windows)
            # nvm-windows doesn't need sourcing in the same way
            # It modifies the system PATH
            :
            ;;
    esac
}

nodejs::install_system_wide() {
    local version="${1:-$DEFAULT_NODE_VERSION}"
    
    log::header "Installing Node.js system-wide..."
    
    # Check if running with proper permissions
    if [[ $EUID -ne 0 ]] && [[ -z "${SUDO_USER:-}" ]]; then
        log::error "System-wide installation requires root privileges"
        return 1
    fi
    
    local platform
    platform=$(nodejs::detect_platform)
    
    case "$platform" in
        linux)
            # Install Node.js using NodeSource repository for consistent system-wide installation
            log::info "Setting up NodeSource repository for Node.js ${version}.x..."
            
            # Install prerequisites
            if command -v apt-get >/dev/null 2>&1; then
                apt-get update
                apt-get install -y ca-certificates curl gnupg
                
                # Add NodeSource GPG key
                mkdir -p /etc/apt/keyrings
                curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
                
                # Add NodeSource repository
                NODE_MAJOR="${version}"
                echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_${NODE_MAJOR}.x nodistro main" > /etc/apt/sources.list.d/nodesource.list
                
                # Update and install
                apt-get update
                apt-get install -y nodejs
                
            elif command -v yum >/dev/null 2>&1; then
                # For RHEL/CentOS/Fedora
                curl -fsSL https://rpm.nodesource.com/setup_${version}.x | bash -
                yum install -y nodejs
                
            else
                log::warning "Package manager not supported for automatic installation"
                log::info "Falling back to manual binary installation..."
                
                # Manual installation from nodejs.org binaries
                local arch
                arch=$(uname -m)
                case "$arch" in
                    x86_64) arch="x64" ;;
                    aarch64) arch="arm64" ;;
                    armv7l) arch="armv7l" ;;
                    *) log::error "Unsupported architecture: $arch"; return 1 ;;
                esac
                
                # Get the latest version number for the major version
                local latest_version
                latest_version=$(curl -sL "https://nodejs.org/dist/latest-v${version}.x/" | grep -oP 'node-v\K[0-9]+\.[0-9]+\.[0-9]+' | head -1)
                if [[ -z "$latest_version" ]]; then
                    log::error "Failed to determine latest Node.js version"
                    return 1
                fi
                
                local node_url="https://nodejs.org/dist/v${latest_version}/node-v${latest_version}-linux-${arch}.tar.xz"
                log::info "Downloading Node.js v${latest_version} from $node_url..."
                
                cd /tmp || return 1
                if ! nodejs::download_with_retry "$node_url" > "node.tar.xz"; then
                    log::error "Failed to download Node.js"
                    return 1
                fi
                
                tar -xf node.tar.xz
                rm node.tar.xz
                
                # Find the extracted directory
                local node_dir
                node_dir=$(find . -maxdepth 1 -type d -name "node-v*" | head -1)
                
                # Copy to /usr/local
                cp -r "${node_dir}"/* /usr/local/
                rm -rf "${node_dir}"
                
                # Create symlinks in /usr/bin for compatibility
                ln -sf /usr/local/bin/node /usr/bin/node
                ln -sf /usr/local/bin/npm /usr/bin/npm
                ln -sf /usr/local/bin/npx /usr/bin/npx
            fi
            ;;
        mac)
            # On Mac, use Homebrew for system-wide installation
            if command -v brew >/dev/null 2>&1; then
                log::info "Installing Node.js via Homebrew..."
                brew install node@${version} || brew install node
            else
                log::error "Homebrew not found. Please install Homebrew first."
                return 1
            fi
            ;;
        *)
            log::error "System-wide installation not supported on $platform"
            return 1
            ;;
    esac
    
    # Verify installation
    if command -v node >/dev/null 2>&1; then
        log::success "Node.js installed system-wide: $(node --version)"
        log::info "npm version: $(npm --version)"
        
        # Enable corepack globally
        if command -v corepack >/dev/null 2>&1; then
            log::info "Enabling corepack globally..."
            corepack enable
        fi
        
        return 0
    else
        log::error "System-wide Node.js installation failed"
        return 1
    fi
}

nodejs::check_and_install() {
    local platform
    platform=$(nodejs::detect_platform)
    
    # If running with sudo on Linux, check if Node.js is available to the actual user
    if [[ -n "${SUDO_USER:-}" ]] && [[ "$platform" == "linux" ]]; then
        log::info "Running with sudo - checking if Node.js is available to user: ${SUDO_USER}"
        
        # Check if Node.js is available to the actual user (not root)
        local node_available_to_user=false
        if sudo -u "${SUDO_USER}" bash -c 'command -v node >/dev/null 2>&1'; then
            node_available_to_user=true
            local user_node_version
            user_node_version=$(sudo -u "${SUDO_USER}" bash -c 'node --version 2>/dev/null' || echo "unknown")
            log::info "Node.js is available to ${SUDO_USER}: ${user_node_version}"
        else
            log::info "Node.js is NOT available to ${SUDO_USER}"
        fi
        
        # If Node.js is not available to the actual user, install it system-wide
        if [[ "$node_available_to_user" != "true" ]]; then
            log::info "Installing Node.js system-wide so all users can access it"
            nodejs::install_system_wide "$@"
            return $?
        else
            # Node.js is already available to the user
            log::info "Node.js is already accessible to ${SUDO_USER}, skipping installation"
            return 0
        fi
    fi
    
    # Otherwise use user-specific installation
    case "$platform" in
        linux)
            # Check if we're in a shell that can source nvm
            if [[ -n "${BASH_VERSION:-}" ]] || [[ -n "${ZSH_VERSION:-}" ]]; then
                nodejs::install_node "$@"
            else
                log::error "Node.js installation on Linux requires bash or zsh shell"
                return 1
            fi
            ;;
        mac)
            nodejs::install_node_volta "$@"
            ;;
        windows)
            nodejs::install_node_windows "$@"
            ;;
        *)
            log::error "Unsupported platform: $platform"
            return 1
            ;;
    esac
}

# If this script is run directly, invoke its main function
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    nodejs::check_and_install "$@"
fi