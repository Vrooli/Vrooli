#!/usr/bin/env bash
# Install Python with development tools and virtual environment support
set -euo pipefail

# Get runtime directory
RUNTIME_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${RUNTIME_DIR}/../utils/var.sh"
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
python::ensure_installed() {
    python::check_and_install "$@"
}

# Default Python version if not specified
DEFAULT_PYTHON_VERSION="3.12"
MIN_PYTHON_VERSION="3.8"

################################################################################
# Python Detection and Version Functions
################################################################################

python::get_python_command() {
    # Return the best available Python command
    if command -v python3 >/dev/null 2>&1; then
        echo "python3"
    elif command -v python >/dev/null 2>&1; then
        # Check if 'python' is Python 3
        local version
        version=$(python --version 2>&1 | grep -oE '[0-9]+\.[0-9]+' | head -1)
        local major_version="${version%%.*}"
        if [[ "$major_version" -ge 3 ]]; then
            echo "python"
        else
            echo ""
        fi
    else
        echo ""
    fi
}

python::get_version() {
    local python_cmd="${1:-$(python::get_python_command)}"
    if [[ -n "$python_cmd" ]] && command -v "$python_cmd" >/dev/null 2>&1; then
        "$python_cmd" --version 2>&1 | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1
    else
        echo ""
    fi
}

python::check_minimum_version() {
    local current_version="$1"
    local min_version="${2:-$MIN_PYTHON_VERSION}"
    
    # Convert versions to comparable format (major * 1000 + minor * 10 + patch)
    local current_comparable
    local min_comparable
    
    current_comparable=$(echo "$current_version" | awk -F. '{print $1 * 1000 + $2 * 10 + ($3 ? $3 : 0)}')
    min_comparable=$(echo "$min_version" | awk -F. '{print $1 * 1000 + $2 * 10 + ($3 ? $3 : 0)}')
    
    [[ $current_comparable -ge $min_comparable ]]
}

python::verify_installation() {
    local python_cmd
    python_cmd=$(python::get_python_command)
    
    if [[ -z "$python_cmd" ]]; then
        log::error "Python 3 not found"
        return 1
    fi
    
    local version
    version=$(python::get_version "$python_cmd")
    
    if python::check_minimum_version "$version" "$MIN_PYTHON_VERSION"; then
        log::success "Python installed: $python_cmd (v$version)"
        
        # Check for pip
        if "$python_cmd" -m pip --version >/dev/null 2>&1; then
            local pip_version
            pip_version=$("$python_cmd" -m pip --version | grep -oE 'pip [0-9]+\.[0-9]+' | cut -d' ' -f2)
            log::info "pip version: $pip_version"
        else
            log::warning "pip not found"
        fi
        
        # Check for venv module
        if "$python_cmd" -m venv --help >/dev/null 2>&1; then
            log::info "venv module: available"
        else
            log::warning "venv module: not available (install python3-venv)"
        fi
        
        return 0
    else
        log::error "Python version $version is below minimum required version $MIN_PYTHON_VERSION"
        return 1
    fi
}

################################################################################
# System Package Installation Functions
################################################################################

python::install_system_packages() {
    local version="${1:-$DEFAULT_PYTHON_VERSION}"
    local pm
    pm=$(system::detect_pm)
    
    log::header "Installing Python system packages..."
    
    # Check if we can run privileged operations
    if ! flow::can_run_sudo "Python system packages installation"; then
        log::warning "Cannot install system packages without sudo privileges"
        log::info "Skipping system package installation"
        return 1
    fi
    
    case "$pm" in
        apt-get)
            # Update package list first
            system::update
            
            # Install Python and essential development packages
            local packages=(
                "python3"
                "python3-pip"
                "python3-venv"        # Virtual environment support
                "python3-dev"         # Development headers
                "python3-setuptools"  # Setup tools
                "python3-wheel"       # Wheel support
                "build-essential"     # Compiler tools for building extensions
                "libssl-dev"          # SSL support
                "libffi-dev"          # Foreign function interface
                "python3-distutils"   # Distutils (may be separate in some distros)
            )
            
            # Try to install specific version if available
            local version_packages=(
                "python${version}"
                "python${version}-venv"
                "python${version}-dev"
                "python${version}-distutils"
            )
            
            # Check if specific version packages exist
            for pkg in "${version_packages[@]}"; do
                if apt-cache show "$pkg" >/dev/null 2>&1; then
                    packages+=("$pkg")
                    log::info "Adding version-specific package: $pkg"
                fi
            done
            
            # Install all packages
            for pkg in "${packages[@]}"; do
                if ! dpkg -l | grep -q "^ii  $pkg "; then
                    log::info "Installing $pkg..."
                    system::install_pkg "$pkg"
                else
                    log::info "$pkg is already installed"
                fi
            done
            
            # Create python3 symlink if specific version was installed
            if command -v "python${version}" >/dev/null 2>&1; then
                log::info "Setting python${version} as default python3"
                flow::maybe_run_sudo update-alternatives --install /usr/bin/python3 python3 "/usr/bin/python${version}" 10
            fi
            ;;
            
        yum|dnf)
            # For RHEL/CentOS/Fedora
            system::update
            
            local packages=(
                "python3"
                "python3-pip"
                "python3-devel"
                "python3-setuptools"
                "python3-wheel"
                "gcc"
                "gcc-c++"
                "make"
                "openssl-devel"
                "libffi-devel"
            )
            
            # Try version-specific packages
            if [[ "$pm" == "dnf" ]]; then
                # Fedora might have version-specific packages
                local ver_major="${version%%.*}"
                local ver_minor="${version#*.}"
                ver_minor="${ver_minor%%.*}"
                
                if dnf list "python${ver_major}${ver_minor}" >/dev/null 2>&1; then
                    packages+=("python${ver_major}${ver_minor}")
                    packages+=("python${ver_major}${ver_minor}-devel")
                fi
            fi
            
            for pkg in "${packages[@]}"; do
                log::info "Installing $pkg..."
                system::install_pkg "$pkg"
            done
            ;;
            
        brew)
            # macOS with Homebrew
            log::info "Installing Python via Homebrew..."
            brew install python@${version} || brew install python3
            
            # Install additional tools
            brew install pipx
            ;;
            
        apk)
            # Alpine Linux
            system::update
            
            local packages=(
                "python3"
                "py3-pip"
                "python3-dev"
                "gcc"
                "musl-dev"
                "libffi-dev"
                "openssl-dev"
                "make"
            )
            
            for pkg in "${packages[@]}"; do
                log::info "Installing $pkg..."
                system::install_pkg "$pkg"
            done
            ;;
            
        pacman)
            # Arch Linux
            system::update
            
            local packages=(
                "python"
                "python-pip"
                "python-setuptools"
                "python-wheel"
                "base-devel"
            )
            
            for pkg in "${packages[@]}"; do
                log::info "Installing $pkg..."
                system::install_pkg "$pkg"
            done
            ;;
            
        *)
            log::warning "Package manager $pm not fully supported for Python installation"
            log::info "Attempting generic Python 3 installation..."
            system::install_pkg python3
            system::install_pkg python3-pip
            ;;
    esac
    
    log::success "Python system packages installed"
    return 0
}

################################################################################
# Pip Configuration and Management
################################################################################

python::ensure_pip() {
    local python_cmd
    python_cmd=$(python::get_python_command)
    
    if [[ -z "$python_cmd" ]]; then
        log::error "Python not found, cannot install pip"
        return 1
    fi
    
    # Check if pip is already available
    if "$python_cmd" -m pip --version >/dev/null 2>&1; then
        log::info "pip is already installed"
        
        # Upgrade pip if it's old
        log::info "Upgrading pip to latest version..."
        "$python_cmd" -m pip install --upgrade pip 2>/dev/null || {
            # If upgrading fails due to externally-managed-environment, try with --user
            "$python_cmd" -m pip install --user --upgrade pip 2>/dev/null || {
                log::warning "Could not upgrade pip (may be externally managed)"
            }
        }
        return 0
    fi
    
    log::info "pip not found, installing..."
    
    # Try ensurepip first (built into Python)
    if "$python_cmd" -m ensurepip --version >/dev/null 2>&1; then
        log::info "Installing pip using ensurepip..."
        "$python_cmd" -m ensurepip --upgrade || {
            log::warning "ensurepip failed, trying alternative method"
        }
    fi
    
    # If pip still not available, download get-pip.py
    if ! "$python_cmd" -m pip --version >/dev/null 2>&1; then
        log::info "Downloading get-pip.py..."
        local get_pip_url="https://bootstrap.pypa.io/get-pip.py"
        local get_pip_file="/tmp/get-pip.py"
        
        if retry::download "$get_pip_url" > "$get_pip_file"; then
            "$python_cmd" "$get_pip_file" --user || {
                log::error "Failed to install pip"
                rm -f "$get_pip_file"
                return 1
            }
            rm -f "$get_pip_file"
        else
            log::error "Failed to download get-pip.py"
            return 1
        fi
    fi
    
    # Verify pip installation
    if "$python_cmd" -m pip --version >/dev/null 2>&1; then
        log::success "pip installed successfully"
        return 0
    else
        log::error "pip installation failed"
        return 1
    fi
}

python::configure_pip() {
    local python_cmd
    python_cmd=$(python::get_python_command)
    
    if [[ -z "$python_cmd" ]]; then
        return 1
    fi
    
    log::info "Configuring pip..."
    
    # Create pip configuration directory
    local pip_config_dir="$HOME/.config/pip"
    mkdir -p "$pip_config_dir"
    
    # Create or update pip.conf with sensible defaults
    local pip_conf="$pip_config_dir/pip.conf"
    if [[ ! -f "$pip_conf" ]]; then
        cat > "$pip_conf" <<EOF
[global]
# Disable pip version check to avoid warnings
disable-pip-version-check = true

# Use binary packages when available (faster installation)
prefer-binary = true

# Increase timeout for slow connections
timeout = 120

# Use system certificates
cert = /etc/ssl/certs/ca-certificates.crt

[install]
# Compile Python bytecode for better performance
compile = true

# Use user directory for installations when not in venv
user = false
EOF
        log::info "Created pip configuration at $pip_conf"
    fi
    
    return 0
}

################################################################################
# Virtual Environment Support
################################################################################

python::ensure_venv_support() {
    local python_cmd
    python_cmd=$(python::get_python_command)
    
    if [[ -z "$python_cmd" ]]; then
        log::error "Python not found"
        return 1
    fi
    
    # Check if venv module is available
    if "$python_cmd" -m venv --help >/dev/null 2>&1; then
        log::info "venv module is already available"
        return 0
    fi
    
    log::warning "venv module not available, attempting to install..."
    
    # Detect package manager and install venv package
    local pm
    pm=$(system::detect_pm)
    
    if ! flow::can_run_sudo "python3-venv installation"; then
        log::error "Cannot install python3-venv without sudo privileges"
        log::info "Please ask your system administrator to install python3-venv"
        return 1
    fi
    
    case "$pm" in
        apt-get)
            # Determine Python version for package name
            local python_version
            python_version=$(python::get_version "$python_cmd")
            local major_minor="${python_version%.*}"
            
            # Try version-specific package first
            local venv_pkg="python${major_minor}-venv"
            if ! apt-cache show "$venv_pkg" >/dev/null 2>&1; then
                venv_pkg="python3-venv"
            fi
            
            log::info "Installing $venv_pkg..."
            system::install_pkg "$venv_pkg"
            ;;
            
        yum|dnf)
            system::install_pkg "python3-venv" || system::install_pkg "python3-virtualenv"
            ;;
            
        *)
            log::warning "Cannot automatically install venv for package manager: $pm"
            log::info "Please install python3-venv manually"
            return 1
            ;;
    esac
    
    # Verify venv is now available
    if "$python_cmd" -m venv --help >/dev/null 2>&1; then
        log::success "venv module installed successfully"
        return 0
    else
        log::error "Failed to install venv module"
        return 1
    fi
}

################################################################################
# Development Tools Installation
################################################################################

python::install_dev_tools() {
    local python_cmd
    python_cmd=$(python::get_python_command)
    
    if [[ -z "$python_cmd" ]]; then
        log::error "Python not found"
        return 1
    fi
    
    log::header "Installing Python development tools..."
    
    # Install pipx for managing Python applications
    python::install_pipx
    
    # Install common development tools via pipx if available
    if command -v pipx >/dev/null 2>&1; then
        local tools=(
            "black"           # Code formatter
            "flake8"          # Linter
            "mypy"            # Type checker
            "pytest"          # Testing framework
            "poetry"          # Dependency management
            "virtualenvwrapper" # Virtual environment management
            "ipython"         # Enhanced Python shell
        )
        
        for tool in "${tools[@]}"; do
            if pipx list | grep -q "$tool"; then
                log::info "$tool is already installed"
            else
                log::info "Installing $tool..."
                pipx install "$tool" 2>/dev/null || {
                    log::warning "Failed to install $tool (may not be available)"
                }
            fi
        done
    fi
    
    log::success "Python development tools installation complete"
    return 0
}

python::install_pipx() {
    local python_cmd
    python_cmd=$(python::get_python_command)
    
    if command -v pipx >/dev/null 2>&1; then
        log::info "pipx is already installed"
        return 0
    fi
    
    log::info "Installing pipx..."
    
    # Check if we can install system-wide
    local pm
    pm=$(system::detect_pm)
    
    case "$pm" in
        apt-get)
            if flow::can_run_sudo "pipx installation"; then
                system::install_pkg "pipx"
                if command -v pipx >/dev/null 2>&1; then
                    pipx ensurepath
                    log::success "pipx installed via apt"
                    return 0
                fi
            fi
            ;;
        brew)
            brew install pipx
            pipx ensurepath
            log::success "pipx installed via brew"
            return 0
            ;;
    esac
    
    # Fallback to pip installation
    log::info "Installing pipx via pip..."
    
    # Try to install in user directory
    "$python_cmd" -m pip install --user pipx 2>/dev/null || {
        log::warning "Could not install pipx"
        return 1
    }
    
    # Add to PATH
    local pipx_bin="$HOME/.local/bin"
    if [[ -d "$pipx_bin" ]] && [[ ":$PATH:" != *":$pipx_bin:"* ]]; then
        export PATH="$pipx_bin:$PATH"
        log::info "Added $pipx_bin to PATH"
        log::info "Add the following to your shell profile:"
        log::info "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
    
    # Ensure pipx paths
    if command -v pipx >/dev/null 2>&1; then
        pipx ensurepath
        log::success "pipx installed successfully"
        return 0
    else
        log::warning "pipx installed but not in PATH"
        return 1
    fi
}

################################################################################
# Handle Externally Managed Environment (PEP 668)
################################################################################

python::handle_externally_managed() {
    local python_cmd
    python_cmd=$(python::get_python_command)
    
    # Check if environment is externally managed
    local marker_file="/usr/lib/python*/EXTERNALLY-MANAGED"
    if ! ls $marker_file >/dev/null 2>&1; then
        # No externally managed marker
        return 0
    fi
    
    log::info "Detected externally managed Python environment (PEP 668)"
    log::info "This is common on Ubuntu 23.04+ and Debian 12+"
    
    # Provide solutions
    log::info "Solutions for package installation:"
    log::info "  1. Use virtual environments (recommended):"
    log::info "     python3 -m venv myenv"
    log::info "     source myenv/bin/activate"
    log::info "     pip install <package>"
    log::info ""
    log::info "  2. Use pipx for applications:"
    log::info "     pipx install <application>"
    log::info ""
    log::info "  3. Use system packages when available:"
    log::info "     sudo apt install python3-<package>"
    log::info ""
    log::info "  4. Force user installation (use with caution):"
    log::info "     pip install --user <package>"
    log::info ""
    log::info "  5. Override protection (NOT recommended):"
    log::info "     pip install --break-system-packages <package>"
    
    return 0
}

################################################################################
# Main Installation Function
################################################################################

python::check_and_install() {
    local requested_version="${1:-$DEFAULT_PYTHON_VERSION}"
    
    log::header "Python Setup"
    
    # Check current Python installation
    local python_cmd
    python_cmd=$(python::get_python_command)
    
    if [[ -n "$python_cmd" ]]; then
        local current_version
        current_version=$(python::get_version "$python_cmd")
        log::info "Found Python: $python_cmd (v$current_version)"
        
        # Check if version meets requirements
        if python::check_minimum_version "$current_version" "$MIN_PYTHON_VERSION"; then
            log::info "Python version meets minimum requirements"
        else
            log::warning "Python version is below minimum, will attempt to install newer version"
            python_cmd=""
        fi
    else
        log::info "Python 3 not found, will install"
    fi
    
    # Install or upgrade Python if needed
    if [[ -z "$python_cmd" ]] || [[ "${PYTHON_FORCE_INSTALL:-false}" == "true" ]]; then
        python::install_system_packages "$requested_version"
        
        # Recheck after installation
        python_cmd=$(python::get_python_command)
        if [[ -z "$python_cmd" ]]; then
            log::error "Failed to install Python"
            return 1
        fi
    fi
    
    # Ensure pip is installed and configured
    python::ensure_pip
    python::configure_pip
    
    # Ensure venv support
    python::ensure_venv_support
    
    # Handle externally managed environment
    python::handle_externally_managed
    
    # Install development tools if requested
    if [[ "${PYTHON_INSTALL_DEV_TOOLS:-false}" == "true" ]]; then
        python::install_dev_tools
    fi
    
    # Final verification
    if python::verify_installation; then
        log::success "Python setup completed successfully"
        
        # Show quick start info
        log::info ""
        log::info "Quick start:"
        log::info "  Create virtual environment: $python_cmd -m venv myenv"
        log::info "  Activate it: source myenv/bin/activate"
        log::info "  Install packages: pip install <package>"
        
        return 0
    else
        log::error "Python setup verification failed"
        return 1
    fi
}

################################################################################
# Utility Functions for Scripts
################################################################################

# Create a virtual environment for a project
python::create_venv() {
    local venv_path="${1:-.venv}"
    local python_cmd="${2:-$(python::get_python_command)}"
    
    if [[ -z "$python_cmd" ]]; then
        log::error "Python not found"
        return 1
    fi
    
    if [[ -d "$venv_path" ]]; then
        log::info "Virtual environment already exists at $venv_path"
        return 0
    fi
    
    log::info "Creating virtual environment at $venv_path..."
    
    if ! "$python_cmd" -m venv "$venv_path"; then
        log::error "Failed to create virtual environment"
        log::info "Ensure python3-venv is installed"
        return 1
    fi
    
    log::success "Virtual environment created at $venv_path"
    log::info "Activate it with: source $venv_path/bin/activate"
    return 0
}

# Install requirements from a requirements file
python::install_requirements() {
    local requirements_file="${1:-requirements.txt}"
    local venv_path="${2:-.venv}"
    
    if [[ ! -f "$requirements_file" ]]; then
        log::error "Requirements file not found: $requirements_file"
        return 1
    fi
    
    # Create venv if it doesn't exist
    if [[ ! -d "$venv_path" ]]; then
        python::create_venv "$venv_path" || return 1
    fi
    
    # Activate venv and install requirements
    log::info "Installing requirements from $requirements_file..."
    
    # Use venv's pip directly without activating
    if [[ -f "$venv_path/bin/pip" ]]; then
        "$venv_path/bin/pip" install -r "$requirements_file"
    else
        log::error "pip not found in virtual environment"
        return 1
    fi
    
    log::success "Requirements installed successfully"
    return 0
}

# If this script is run directly, invoke its main function
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    python::check_and_install "$@"
fi