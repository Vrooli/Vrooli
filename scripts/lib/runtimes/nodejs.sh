#!/usr/bin/env bash
# Install Node.js via platform-appropriate version manager
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
RUNTIME_DIR="${APP_ROOT}/scripts/lib/runtimes"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$var_LIB_SYSTEM_DIR/system_commands.sh"
# shellcheck disable=SC1091
source "$var_LIB_UTILS_DIR/flow.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

# Clean interface for setup.sh
nodejs::ensure_installed() {
    nodejs::check_and_install "$@"
}
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/retry.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/hash.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"

# Default Node.js version if not specified
DEFAULT_NODE_VERSION="20"
# NVM version to install
NVM_VERSION="0.40.3"


# Get platform name with Node.js-specific conventions
nodejs::get_platform() {
    local platform
    platform=$(system::detect_platform)
    # Convert darwin to mac for backward compatibility
    [[ "$platform" == "darwin" ]] && platform="mac"
    echo "$platform"
}

# Simple verification that Node.js is available
nodejs::verify_installation() {
    if system::is_command node; then
        log::success "Node.js installed: $(node --version)"
        system::is_command npm && log::info "npm version: $(npm --version)"
        return 0
    else
        log::error "Node.js installation verification failed"
        return 1
    fi
}

nodejs::ensure_nvm() {
    local nvm_dir
    nvm_dir="${NVM_DIR:-$HOME/.nvm}"
    
    # Check if nvm is already installed
    if [[ -s "$nvm_dir/nvm.sh" ]]; then
        log::info "nvm is already installed at $nvm_dir"
        return 0
    fi
    
    log::info "Installing nvm v${NVM_VERSION}..."
    
    # Download and install nvm with retry
    local nvm_install_url="https://raw.githubusercontent.com/nvm-sh/nvm/v${NVM_VERSION}/install.sh"
    if ! retry::download "$nvm_install_url" | bash; then
        log::error "Failed to install nvm from $nvm_install_url"
        return 1
    fi
    
    # Verify installation
    if [[ ! -s "$nvm_dir/nvm.sh" ]]; then
        log::error "nvm installation verification failed"
        return 1
    fi
    
    log::success "nvm v${NVM_VERSION} installed successfully"
    return 0
}

nodejs::source_nvm() {
    local nvm_dir
    nvm_dir="${NVM_DIR:-$HOME/.nvm}"
    
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
    if system::is_command node; then
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
    if ! system::is_command volta; then
        log::info "Volta not found. Installing Volta..."
        
        # Check if Homebrew is available (Mac)
        if system::is_command brew; then
            brew install volta || {
                log::error "Failed to install Volta via Homebrew"
                return 1
            }
        else
                # Fallback to curl installation with retry
            local volta_install_url="https://get.volta.sh"
            log::info "Installing Volta from $volta_install_url..."
            if ! retry::download "$volta_install_url" | bash; then
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
    if ! system::is_command nvm; then
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
    platform=$(nodejs::get_platform)
    
    case "$platform" in
        linux)
            nodejs::source_nvm
            ;;
        darwin|mac)
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
    
    # Check if we can run privileged operations
    if ! flow::can_run_sudo "system-wide Node.js installation"; then
        log::error "System-wide installation requires root privileges"
        log::info "Run with sudo or use --sudo-mode ask to enable privileged operations"
        return 1
    fi
    
    local platform
    platform=$(nodejs::get_platform)
    
    case "$platform" in
        linux)
            # Install Node.js using NodeSource repository for consistent system-wide installation
            log::info "Setting up NodeSource repository for Node.js ${version}.x..."
            
            # Install prerequisites using system package utilities
            local pm
            pm=$(system::detect_pm)
            
            case "$pm" in
                apt-get)
                    # Install prerequisites
                    system::install_pkg ca-certificates
                    system::install_pkg curl
                    system::install_pkg gnupg
                    
                    # Add NodeSource GPG key with retry
                    flow::maybe_run_sudo mkdir -p /etc/apt/keyrings
                    local gpg_key_url="https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key"
                    if ! retry::download "$gpg_key_url" | flow::maybe_run_sudo gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg; then
                        log::error "Failed to download NodeSource GPG key"
                        return 1
                    fi
                    
                    # Add NodeSource repository
                    NODE_MAJOR="${version}"
                    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_${NODE_MAJOR}.x nodistro main" | flow::maybe_run_sudo tee /etc/apt/sources.list.d/nodesource.list > /dev/null
                    
                    # Update and install using system utilities
                    system::update
                    system::install_pkg nodejs
                    ;;
                yum|dnf)
                    # For RHEL/CentOS/Fedora with retry
                    local rpm_setup_url="https://rpm.nodesource.com/setup_${version}.x"
                    if ! retry::download "$rpm_setup_url" | flow::maybe_run_sudo bash -; then
                        log::error "Failed to setup NodeSource repository"
                        return 1
                    fi
                    system::install_pkg nodejs
                    ;;
                *)
                    log::warning "Package manager $pm not supported for automatic installation"
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
                
                # Get the latest version number for the major version with retry
                local version_list_url="https://nodejs.org/dist/latest-v${version}.x/"
                local latest_version
                latest_version=$(retry::download "$version_list_url" 3 2 | grep -oP 'node-v\K[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)
                if [[ -z "$latest_version" ]]; then
                    log::error "Failed to determine latest Node.js version"
                    return 1
                fi
                    
                    local node_url="https://nodejs.org/dist/v${latest_version}/node-v${latest_version}-linux-${arch}.tar.xz"
                    local node_tarball="/tmp/node-v${latest_version}.tar.xz"
                    log::info "Downloading Node.js v${latest_version}..."
                    
                    if ! retry::download "$node_url" > "$node_tarball"; then
                        log::error "Failed to download Node.js from $node_url"
                        return 1
                    fi
                    
                    cd /tmp || return 1
                
                tar -xf "$node_tarball"
                rm -f "$node_tarball"
                
                # Find the extracted directory
                local node_dir
                node_dir=$(find . -maxdepth 1 -type d -name "node-v*" | head -1)
                
                # Copy to /usr/local
                flow::maybe_run_sudo cp -r "${node_dir}"/* /usr/local/
                trash::safe_remove "${node_dir}" --no-confirm
                
                # Create symlinks in /usr/bin for compatibility
                flow::maybe_run_sudo ln -sf /usr/local/bin/node /usr/bin/node
                flow::maybe_run_sudo ln -sf /usr/local/bin/npm /usr/bin/npm
                flow::maybe_run_sudo ln -sf /usr/local/bin/npx /usr/bin/npx
                    ;;
            esac
            ;;
        darwin|mac)
            # On Mac, use Homebrew for system-wide installation
            if system::is_command brew; then
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
    nodejs::verify_installation
    
    # Enable corepack globally if available
    if system::is_command corepack; then
        log::info "Enabling corepack globally..."
        flow::maybe_run_sudo corepack enable
    fi
}

nodejs::check_and_install() {
    local platform
    platform=$(nodejs::get_platform)
    
    # If running with sudo on Linux, check if Node.js is available to the actual user
    if sudo::is_running_as_sudo && [[ "$platform" == "linux" ]] && flow::can_run_sudo "user Node.js check"; then
        local actual_user
        actual_user=$(sudo::get_actual_user)
        log::info "Running with sudo - checking if Node.js is available to user: ${actual_user}"
        
        # Check if Node.js is available to the actual user (not root)
        local node_available_to_user=false
        if sudo::exec_as_actual_user 'command -v node >/dev/null 2>&1'; then
            node_available_to_user=true
            local user_node_version
            user_node_version=$(sudo::exec_as_actual_user 'node --version 2>/dev/null' || echo "unknown")
            log::info "Node.js is available to ${actual_user}: ${user_node_version}"
        else
            log::info "Node.js is NOT available to ${actual_user}"
        fi
        
        # If Node.js is not available to the actual user, install it system-wide
        if [[ "$node_available_to_user" != "true" ]]; then
            log::info "Installing Node.js system-wide so all users can access it"
            nodejs::install_system_wide "$@"
            return $?
        else
            # Node.js is already available to the user
            log::info "Node.js is already accessible to ${actual_user}, skipping installation"
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
        darwin|mac)
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

################################################################################
# Package Manager Functions (pnpm, npm, yarn)
################################################################################


# Ensure pnpm is available, using corepack if possible, otherwise fallback to standalone installation
nodejs::ensure_pnpm() {
    local version="${1:-latest}"
    
    # Check if pnpm is already available
    if system::is_command pnpm; then
        local current_version
        current_version=$(pnpm --version 2>/dev/null || echo "unknown")
        log::info "pnpm is already installed: v${current_version}"
        return 0
    fi
    
    log::info "Installing pnpm..."
    
    # Try to use corepack first (comes with Node.js 16.10+)
    if system::is_command corepack; then
        log::info "Using corepack to enable pnpm..."
        
        # Enable corepack and prepare pnpm
        sudo::exec_as_actual_user "corepack enable" || true
        
        if [[ "$version" == "latest" ]]; then
            sudo::exec_as_actual_user "corepack prepare pnpm@latest --activate" || true
        else
            sudo::exec_as_actual_user "corepack prepare pnpm@${version} --activate" || true
        fi
        
        # Check if pnpm is now available
        if system::is_command pnpm; then
            log::success "pnpm installed via corepack: $(pnpm --version)"
            return 0
        fi
    fi
    
    # Fallback to standalone installation
    log::info "Installing pnpm standalone..."
    
    # Set PNPM_HOME based on actual user's home directory
    local actual_home
    actual_home=$(sudo::get_actual_home)
    export PNPM_HOME="$actual_home/.local/share/pnpm"
    
    # Create directory with proper ownership
    sudo::mkdir_as_user "$PNPM_HOME"
    
    # Download and run pnpm installer with retry
    local pnpm_install_url="https://get.pnpm.io/install.sh"
    local install_script
    install_script=$(retry::download "$pnpm_install_url")
    if [[ -z "$install_script" ]]; then
        log::error "Failed to download pnpm installer"
        return 1
    fi
    echo "$install_script" | sudo::exec_as_actual_user "export PNPM_HOME='$PNPM_HOME' && bash -"
    
    # Add to PATH for current session
    export PATH="$PNPM_HOME:$PATH"
    
    # Verify installation
    if system::is_command pnpm; then
        log::success "pnpm installed successfully: $(pnpm --version)"
        
        # Inform user about PATH update needed
        log::info "Add the following to your shell profile to use pnpm:"
        log::info "  export PNPM_HOME=\"$PNPM_HOME\""
        log::info "  export PATH=\"\$PNPM_HOME:\$PATH\""
        
        return 0
    else
        log::error "Failed to install pnpm"
        return 1
    fi
}

# Ensure npm is available and up to date
nodejs::ensure_npm() {
    if ! system::is_command npm; then
        log::error "npm is not available. Please install Node.js first."
        return 1
    fi
    
    local current_version
    current_version=$(npm --version)
    log::info "npm version: $current_version"
    
    # Update npm to latest if requested
    if [[ "${1:-}" == "--update" ]]; then
        log::info "Updating npm to latest version..."
        sudo::exec_as_actual_user "npm install -g npm@latest"
        log::success "npm updated to: $(npm --version)"
    fi
    
    return 0
}

# Ensure yarn is available
nodejs::ensure_yarn() {
    local version="${1:-latest}"
    
    # Check if yarn is already available
    if system::is_command yarn; then
        local current_version
        current_version=$(yarn --version 2>/dev/null || echo "unknown")
        log::info "yarn is already installed: v${current_version}"
        return 0
    fi
    
    log::info "Installing yarn..."
    
    # Try corepack first
    if system::is_command corepack; then
        log::info "Using corepack to enable yarn..."
        sudo::exec_as_actual_user "corepack enable"
        
        if [[ "$version" == "latest" ]]; then
            sudo::exec_as_actual_user "corepack prepare yarn@stable --activate"
        else
            sudo::exec_as_actual_user "corepack prepare yarn@${version} --activate"
        fi
        
        if system::is_command yarn; then
            log::success "yarn installed via corepack: $(yarn --version)"
            return 0
        fi
    fi
    
    # Fallback to npm installation
    if system::is_command npm; then
        log::info "Installing yarn via npm..."
        sudo::exec_as_actual_user "npm install -g yarn"
        
        if system::is_command yarn; then
            log::success "yarn installed via npm: $(yarn --version)"
            return 0
        fi
    fi
    
    log::error "Failed to install yarn"
    return 1
}

# Detect which package manager is being used in the current project
nodejs::detect_package_manager() {
    local project_dir="${1:-$(pwd)}"
    
    # Check for lock files
    if [[ -f "$project_dir/pnpm-lock.yaml" ]]; then
        echo "pnpm"
    elif [[ -f "$project_dir/yarn.lock" ]]; then
        echo "yarn"
    elif [[ -f "$project_dir/package-lock.json" ]]; then
        echo "npm"
    elif [[ -f "$project_dir/package.json" ]]; then
        # No lock file, check packageManager field in package.json
        if system::is_command jq; then
            local pkg_manager
            pkg_manager=$(jq -r '.packageManager // ""' "$project_dir/package.json")
            if [[ -n "$pkg_manager" ]]; then
                # Extract manager name from format like "pnpm@8.0.0"
                echo "${pkg_manager%%@*}"
            else
                echo "npm"  # Default to npm
            fi
        else
            echo "npm"  # Default to npm
        fi
    else
        echo "none"  # No package.json found
    fi
}

# Install dependencies using the appropriate package manager
nodejs::install_dependencies() {
    local project_dir="${1:-$(pwd)}"
    local flags="${2:-}"
    
    local pkg_manager
    pkg_manager=$(nodejs::detect_package_manager "$project_dir")
    
    if [[ "$pkg_manager" == "none" ]]; then
        log::warning "No package.json found in $project_dir"
        return 1
    fi
    
    log::info "Installing dependencies with $pkg_manager..."
    
    # Change to project directory
    pushd "$project_dir" > /dev/null || return 1
    
    case "$pkg_manager" in
        pnpm)
            nodejs::ensure_pnpm || return 1
            sudo::exec_as_actual_user "pnpm install $flags"
            ;;
        yarn)
            nodejs::ensure_yarn || return 1
            sudo::exec_as_actual_user "yarn install $flags"
            ;;
        npm)
            nodejs::ensure_npm || return 1
            sudo::exec_as_actual_user "npm install $flags"
            ;;
    esac
    
    local result=$?
    popd > /dev/null || return 1
    
    if [[ $result -eq 0 ]]; then
        log::success "Dependencies installed successfully"
    else
        log::error "Failed to install dependencies"
    fi
    
    return $result
}


# If this script is run directly, invoke its main function
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    nodejs::check_and_install "$@"
fi