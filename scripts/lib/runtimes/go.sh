#!/usr/bin/env bash
# Install Go via platform-appropriate method
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
source "${var_LIB_SYSTEM_DIR}/trash.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/retry.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/permissions.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"

# Clean interface for setup.sh
go::ensure_installed() {
    go::check_and_install "$@"
}

# Default Go version if not specified
DEFAULT_GO_VERSION="1.21"

go::download_with_retry() {
    local url="$1"
    local output_file="$2"
    local max_attempts="${3:-3}"
    local delay="${4:-2}"
    
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        log::info "Download attempt $attempt of $max_attempts..."
        log::info "Downloading from: $url"
        log::info "Saving to: $output_file"
        
        # Use curl with extended timeouts for large file downloads
        if curl -fsSL \
            --connect-timeout 30 \
            --max-time 600 \
            --retry 2 \
            --retry-delay 1 \
            "$url" -o "$output_file"; then
            
            # Verify file was downloaded and has content
            if [[ -f "$output_file" ]] && [[ -s "$output_file" ]]; then
                log::success "Download completed successfully"
                return 0
            else
                log::error "Downloaded file is empty or doesn't exist"
            fi
        fi
        
        if [[ $attempt -lt $max_attempts ]]; then
            log::warning "Download failed, retrying in ${delay} seconds..."
            # Clean up failed download
            [[ -f "$output_file" ]] && rm -f "$output_file"
            sleep "$delay"
        fi
        
        ((attempt++))
    done
    
    log::error "Download failed after $max_attempts attempts"
    return 1
}

go::detect_arch() {
    # Map Go-specific architecture names
    local arch
    arch=$(system::detect_arch)
    
    # Go uses slightly different naming
    case "$arch" in
        armv7) echo "armv6l" ;;  # Go uses armv6l for ARMv7
        *) echo "$arch" ;;       # Others match
    esac
}

go::get_go_root() {
    echo "${GOROOT:-/usr/local/go}"
}

go::verify_installation() {
    local expected_version="${1:-}"
    
    if command -v go >/dev/null 2>&1; then
        local installed_version
        installed_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | sed 's/go//')
        log::success "Go installed successfully: go$installed_version"
        
        # Check GOPATH and GOROOT
        local goroot
        goroot=$(go env GOROOT 2>/dev/null || echo "not set")
        log::info "GOROOT: $goroot"
        
        local gopath
        gopath=$(go env GOPATH 2>/dev/null || echo "not set")
        log::info "GOPATH: $gopath"
        
        # Check if version matches expected (if provided)
        if [[ -n "$expected_version" ]]; then
            if [[ "$installed_version" == "$expected_version"* ]]; then
                log::info "Installed version matches requested version"
            else
                log::warning "Installed version ($installed_version) differs from requested ($expected_version)"
            fi
        fi
        
        return 0
    else
        log::error "Go installation verification failed"
        return 1
    fi
}

go::get_latest_version() {
    local major_version="${1:-$DEFAULT_GO_VERSION}"
    
    # Get latest patch version for the major version
    local latest_version
    latest_version=$(curl -s "https://go.dev/dl/" | grep -oE "go${major_version}\.[0-9]+" | head -1 | sed 's/^go//')
    
    if [[ -n "$latest_version" ]]; then
        echo "$latest_version"
    else
        # Fallback to provided version with .0 patch
        echo "${major_version}.0"
    fi
}

go::remove_old_installation() {
    local go_root="$1"
    
    if [[ -d "$go_root" ]]; then
        log::info "Removing previous Go installation at $go_root..."
        
        # Check if we need sudo to remove the directory
        if [[ ! -w "$go_root" ]] || [[ ! -w "${go_root%/*}" ]]; then
            if permissions::check_sudo; then
                sudo bash -c "source ${var_LIB_SYSTEM_DIR}/trash.sh && trash::safe_remove '$go_root' --no-confirm"
            else
                log::warning "Cannot remove $go_root without sudo permissions"
                return 1
            fi
        else
            trash::safe_remove "$go_root" --no-confirm
        fi
    fi
}

go::install_binary() {
    local version="${1:-$DEFAULT_GO_VERSION}"
    local install_dir="${2:-/usr/local}"
    
    log::header "Installing Go via binary download..."
    
    # Get exact version (including patch version)
    local exact_version
    exact_version=$(go::get_latest_version "$version")
    log::info "Installing Go version: $exact_version"
    
    # Detect platform and architecture
    local platform arch
    platform=$(system::detect_platform)
    arch=$(go::detect_arch)
    
    if [[ "$platform" == "unknown" ]] || [[ "$arch" == "unknown" ]]; then
        log::error "Unsupported platform: $platform/$arch"
        return 1
    fi
    
    # Build download URL
    local filename="go${exact_version}.${platform}-${arch}.tar.gz"
    local url="https://go.dev/dl/${filename}"
    local go_root="${install_dir}/go"
    
    log::info "Downloading Go from: $url"
    
    # Create temporary directory
    local temp_dir
    temp_dir=$(mktemp -d)
    local temp_file="${temp_dir}/${filename}"
    
    # Download with retry
    if ! go::download_with_retry "$url" "$temp_file"; then
        log::error "Failed to download Go from $url"
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
    
    # Verify the downloaded file is a valid tar.gz
    if ! file "$temp_file" | grep -q "gzip compressed"; then
        log::error "Downloaded file is not a valid gzip archive"
        log::info "Downloaded file type: $(file "$temp_file")"
        if [[ -f "$temp_file" ]]; then
            log::info "File size: $(wc -c < "$temp_file") bytes"
            log::info "First 100 characters:"
            head -c 100 "$temp_file" | cat -v
        fi
        trash::safe_remove "$temp_dir" --no-confirm
        return 1
    fi
    
    # Remove any existing installation
    go::remove_old_installation "$go_root"
    
    # Extract to installation directory
    log::info "Extracting Go to $go_root..."
    
    # Check if we need sudo for the install directory
    if [[ "$install_dir" == "/usr/local" ]] && [[ ! -w "$install_dir" ]]; then
        if permissions::check_sudo; then
            sudo tar -C "$install_dir" -xzf "$temp_file" || {
                log::error "Failed to extract Go archive"
                trash::safe_remove "$temp_dir" --no-confirm
                return 1
            }
        else
            log::warning "Cannot write to $install_dir without sudo permissions"
            log::info "Consider installing to your home directory instead"
            trash::safe_remove "$temp_dir" --no-confirm
            return 1
        fi
    else
        tar -C "$install_dir" -xzf "$temp_file" || {
            log::error "Failed to extract Go archive"
            trash::safe_remove "$temp_dir" --no-confirm
            return 1
        }
    fi
    
    # Clean up
    trash::safe_remove "$temp_dir" --no-confirm
    
    # Set up environment for current session
    export GOROOT="$go_root"
    export PATH="$go_root/bin:$PATH"
    
    # Verify installation
    go::verify_installation "$exact_version"
}

go::install_package_manager() {
    local version="${1:-$DEFAULT_GO_VERSION}"
    local platform
    platform=$(system::detect_platform)
    
    log::header "Installing Go via package manager..."
    
    # Check for sudo permissions if needed
    if [[ "$platform" == "linux" ]] && [[ $EUID -ne 0 ]]; then
        if ! permissions::check_sudo; then
            log::warning "Package manager installation requires sudo permissions"
            log::info "Falling back to user-local installation"
            go::install_binary "$version"
            return $?
        fi
    fi
    
    case "$platform" in
        linux)
            if command -v apt-get >/dev/null 2>&1; then
                # Ubuntu/Debian
                log::info "Installing Go via apt..."
                if [[ $EUID -eq 0 ]]; then
                    apt-get update
                    apt-get install -y golang-go
                else
                    sudo apt-get update
                    sudo apt-get install -y golang-go
                fi
                
            elif command -v yum >/dev/null 2>&1; then
                # RHEL/CentOS/Fedora
                log::info "Installing Go via yum..."
                if [[ $EUID -eq 0 ]]; then
                    yum install -y golang
                else
                    sudo yum install -y golang
                fi
                
            elif command -v pacman >/dev/null 2>&1; then
                # Arch Linux
                log::info "Installing Go via pacman..."
                if [[ $EUID -eq 0 ]]; then
                    pacman -S --noconfirm go
                else
                    sudo pacman -S --noconfirm go
                fi
                
            elif command -v apk >/dev/null 2>&1; then
                # Alpine Linux
                log::info "Installing Go via apk..."
                if [[ $EUID -eq 0 ]]; then
                    apk add --no-cache go
                else
                    sudo apk add --no-cache go
                fi
                
            else
                log::warning "No supported package manager found, falling back to binary installation"
                go::install_binary "$version"
                return $?
            fi
            ;;
        darwin)
            if command -v brew >/dev/null 2>&1; then
                log::info "Installing Go via Homebrew..."
                brew install go
            else
                log::warning "Homebrew not found, falling back to binary installation"
                go::install_binary "$version"
                return $?
            fi
            ;;
        *)
            log::error "Package manager installation not supported on $platform"
            return 1
            ;;
    esac
    
    # Verify installation
    go::verify_installation
}

go::setup_environment() {
    local go_root="${1:-/usr/local/go}"
    local user_profile="${2:-$HOME/.profile}"
    
    log::info "Setting up Go environment..."
    
    # Create environment setup
    local env_setup="# Go environment
export GOROOT=\"$go_root\"
export PATH=\"\$GOROOT/bin:\$PATH\"

# Go workspace (optional - Go modules don't require GOPATH)
export GOPATH=\"\$HOME/go\"
export PATH=\"\$GOPATH/bin:\$PATH\""

    # Check if Go environment is already configured
    if [[ -f "$user_profile" ]] && grep -q "GOROOT" "$user_profile"; then
        log::info "Go environment already configured in $user_profile"
    else
        log::info "Adding Go environment to $user_profile"
        echo "$env_setup" >> "$user_profile"
    fi
    
    # Also add to bashrc if it exists
    local bashrc="$HOME/.bashrc"
    if [[ -f "$bashrc" ]] && ! grep -q "GOROOT" "$bashrc"; then
        log::info "Adding Go environment to $bashrc"
        echo "$env_setup" >> "$bashrc"
    fi
    
    # Source for current session
    export GOROOT="$go_root"
    export PATH="$GOROOT/bin:$PATH"
    export GOPATH="${GOPATH:-$HOME/go}"
    export PATH="$GOPATH/bin:$PATH"
    
    log::success "Go environment configured"
}

go::install_system_wide() {
    local version="${1:-$DEFAULT_GO_VERSION}"
    
    log::header "Installing Go system-wide..."
    
    # Check if running with proper permissions
    if ! sudo::is_root; then
        log::error "System-wide installation requires root privileges"
        return 1
    fi
    
    # Install via binary to /usr/local
    go::install_binary "$version" "/usr/local" || return 1
    
    # Set up system-wide environment
    log::info "Configuring system-wide Go environment..."
    
    # Add to /etc/profile.d for all users
    local profile_script="/etc/profile.d/go.sh"
    cat > "$profile_script" << 'EOF'
# Go environment - system-wide
export GOROOT="/usr/local/go"
export PATH="$GOROOT/bin:$PATH"
EOF
    chmod +x "$profile_script"
    
    # Create symlinks in /usr/bin for convenience
    ln -sf /usr/local/go/bin/go /usr/bin/go 2>/dev/null || true
    ln -sf /usr/local/go/bin/gofmt /usr/bin/gofmt 2>/dev/null || true
    
    log::success "Go installed system-wide at /usr/local/go"
    
    # If we have SUDO_USER, also set up their personal environment
    if sudo::is_running_as_sudo; then
        local user_home
        user_home=$(sudo::get_actual_home)
        local actual_user
        actual_user=$(sudo::get_actual_user)
        sudo::exec_as_actual_user "
            export GOPATH=\"${user_home}/go\"
            mkdir -p \"\$GOPATH\"/{bin,src,pkg}
        "
        log::info "Created Go workspace for user: ${actual_user} at ${user_home}/go"
    fi
    
    return 0
}

go::install_dev_tools() {
    log::header "Installing Go development tools..."
    
    # Check if Go is installed first
    if ! command -v go >/dev/null 2>&1; then
        log::error "Go not found - cannot install development tools"
        return 1
    fi
    
    # Install golangci-lint (comprehensive Go linter)
    if command -v golangci-lint >/dev/null 2>&1; then
        log::info "golangci-lint is already installed"
        local version
        version=$(golangci-lint --version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
        log::info "Current version: $version"
    else
        log::info "Installing golangci-lint..."
        
        # Install via Go modules (recommended method)
        if go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null; then
            log::success "golangci-lint installed successfully"
        else
            log::warning "Failed to install golangci-lint via go install, trying curl method..."
            
            # Fallback to curl installation
            local install_script
            install_script=$(mktemp)
            if curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh -o "$install_script" && \
               sh "$install_script" -b "$(go env GOPATH)/bin" latest; then
                log::success "golangci-lint installed via install script"
            else
                log::error "Failed to install golangci-lint"
                rm -f "$install_script"
                return 1
            fi
            rm -f "$install_script"
        fi
    fi
    
    # Install other useful Go tools
    local tools=(
        "golang.org/x/tools/cmd/goimports@latest"    # Better import formatting
        "github.com/fatih/gomodifytags@latest"        # Struct tag manipulation  
        "github.com/josharian/impl@latest"            # Interface implementation generator
        "github.com/go-delve/delve/cmd/dlv@latest"    # Go debugger
        "honnef.co/go/tools/cmd/staticcheck@latest"   # Advanced static analysis
    )
    
    for tool in "${tools[@]}"; do
        local tool_name
        tool_name=$(basename "${tool%%@*}")
        
        if command -v "$tool_name" >/dev/null 2>&1; then
            log::info "$tool_name is already installed"
        else
            log::info "Installing $tool_name..."
            if go install "$tool" 2>/dev/null; then
                log::success "$tool_name installed successfully"
            else
                log::warning "Failed to install $tool_name (may not be compatible with current Go version)"
            fi
        fi
    done
    
    # Verify GOPATH/bin is in PATH
    local gopath_bin
    if command -v go >/dev/null 2>&1; then
        gopath_bin="$(go env GOPATH)/bin"
        if [[ -d "$gopath_bin" ]] && [[ ":$PATH:" != *":$gopath_bin:"* ]]; then
            log::warning "GOPATH/bin ($gopath_bin) is not in PATH"
            log::info "Add the following to your shell profile:"
            log::info "  export PATH=\"\$(go env GOPATH)/bin:\$PATH\""
        fi
    fi
    
    log::success "Go development tools installation complete"
    return 0
}

go::check_and_install() {
    local version="${1:-$DEFAULT_GO_VERSION}"
    local platform
    platform=$(system::detect_platform)
    
    # Check if Go is already installed
    if command -v go >/dev/null 2>&1; then
        local current_version
        current_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | sed 's/go//')
        log::info "Go is already installed: $current_version"
        
        # Check if it matches the requested version (major.minor)
        local requested_major_minor
        requested_major_minor=$(echo "$version" | grep -oE '[0-9]+\.[0-9]+')
        local current_major_minor  
        current_major_minor=$(echo "$current_version" | grep -oE '[0-9]+\.[0-9]+')
        
        if [[ "$current_major_minor" == "$requested_major_minor" ]]; then
            log::success "Go $current_version matches requested version $version"
            return 0
        else
            log::info "Installed Go $current_version differs from requested $version"
            log::info "Proceeding with installation of Go $version"
        fi
    fi
    
    # Check if we're running with sudo privileges
    if sudo::is_running_as_sudo && [[ "$platform" == "linux" ]]; then
        log::info "Running with sudo - installing Go system-wide"
        go::install_system_wide "$version"
        return $?
    fi
    
    # Determine installation method based on platform and preferences
    case "$platform" in
        linux)
            # On Linux, prefer binary installation for consistency
            if [[ $EUID -eq 0 ]]; then
                go::install_system_wide "$version"
            else
                go::install_binary "$version" "$HOME/.local" || {
                    log::warning "Binary installation failed, trying package manager..."
                    go::install_package_manager "$version"
                }
                # Set up user environment
                go::setup_environment "$HOME/.local/go" "$HOME/.profile"
            fi
            ;;
        darwin)
            # On macOS, prefer Homebrew if available, otherwise binary
            if command -v brew >/dev/null 2>&1; then
                go::install_package_manager "$version" || {
                    log::warning "Homebrew installation failed, trying binary..."
                    go::install_binary "$version"
                    go::setup_environment
                }
            else
                go::install_binary "$version"
                go::setup_environment
            fi
            ;;
        windows)
            log::error "Windows installation not yet supported via this script"
            log::info "Please download Go from https://go.dev/dl/ and install manually"
            return 1
            ;;
        *)
            log::error "Unsupported platform: $platform"
            return 1
            ;;
    esac
    
    # Install development tools if requested
    if [[ "${GO_INSTALL_DEV_TOOLS:-false}" == "true" ]]; then
        go::install_dev_tools
    fi
}

go::show_post_install_info() {
    log::info ""
    log::info "🎉 Go installation complete!"
    log::info ""
    log::info "To use Go in new shell sessions, either:"
    log::info "  1. Start a new terminal, or"
    log::info "  2. Run: source ~/.profile"
    log::info ""
    log::info "Verify installation:"
    log::info "  go version"
    log::info "  go env GOROOT"
    log::info "  go env GOPATH"
    log::info ""
    log::info "Create your first Go program:"
    log::info "  mkdir -p ~/hello && cd ~/hello"
    log::info "  echo 'package main' > main.go"
    log::info "  echo 'import \"fmt\"' >> main.go"
    log::info "  echo 'func main() { fmt.Println(\"Hello, World!\") }' >> main.go"
    log::info "  go run main.go"
    log::info ""
}

# If this script is run directly, invoke its main function
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    go::check_and_install "$@"
    go::show_post_install_info
fi