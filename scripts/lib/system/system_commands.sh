#!/usr/bin/env bash
# system.sh - Cross-platform package manager helpers
set -euo pipefail

# Source guard to prevent re-sourcing
[[ -n "${_SYSTEM_COMMANDS_SH_SOURCED:-}" ]] && return 0
_SYSTEM_COMMANDS_SH_SOURCED=1

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "$var_LIB_UTILS_DIR/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$var_LIB_UTILS_DIR/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Default timeout for system installs (in seconds)
SYSTEM_INSTALL_TIMEOUT=${SYSTEM_INSTALL_TIMEOUT:-420}

system::is_command() {
    # Using 'command -v' is generally preferred and more portable than 'which'
    command -v "$1" >/dev/null 2>&1
}

system::assert_command() {
    local command="$1"
    local error_message="${2:-Command $command not found}"
    if ! system::is_command "$command"; then
        log::error "$error_message"
        exit "${ERROR_COMMAND_NOT_FOUND}"
    fi
}

system::detect_pm() {
    if   system::is_command "apt-get"; then echo "apt-get"
    elif system::is_command "dnf"; then echo "dnf"
    elif system::is_command "yum"; then echo "yum"
    elif system::is_command "pacman"; then echo "pacman"
    elif system::is_command "apk"; then echo "apk"
    elif system::is_command "brew"; then echo "brew"
    else
        log::error "Unsupported pkg manager; please install dependencies manually."
        exit 1
    fi
}

# Detect the operating system platform
# Returns: linux, darwin (mac), windows, or unknown
system::detect_platform() {
    local uname_out
    uname_out="$(uname -s)"
    case "${uname_out}" in
        Linux*)     echo "linux";;
        Darwin*)    echo "darwin";;
        CYGWIN*|MINGW*|MSYS*) echo "windows";;
        *)          echo "unknown";;
    esac
}

# Detect system architecture
# Returns: amd64, arm64, armv7, 386, or unknown
system::detect_arch() {
    local arch
    arch="$(uname -m)"
    case "${arch}" in
        x86_64|amd64) echo "amd64";;
        aarch64|arm64) echo "arm64";;
        armv7l) echo "armv7";;
        i386|i686) echo "386";;
        *) echo "unknown";;
    esac
}

# Given a command name, return the real package to install on this distro.
system::get_package_name() {
    local cmd="$1"
    local pm
    pm=$(system::detect_pm)
    
    case "$cmd" in
        # coreutils commands
        nproc|mkdir|sed|grep|awk)
            case "$pm" in
                apt-get)   echo "coreutils" ;;
                dnf|yum)   echo "coreutils" ;;
                pacman)    echo "coreutils" ;;
                apk)       echo "coreutils" ;;    # Alpine has coreutils
                brew)      echo "coreutils" ;;    # Homebrew coreutils
            esac
            ;;
        free)
            case "$pm" in
              apt-get)   echo "procps" ;;         # Debian/Ubuntu
              dnf|yum)   echo "procps-ng" ;;      # Fedora/RHEL/CentOS
              pacman)    echo "procps-ng" ;;      # Arch
              apk)       echo "procps" ;;
              brew)      echo "procps" ;;         # Homebrew? usually not needed
            esac
            ;;
        *)
            # fallback: assume the package has the same name
            echo "$cmd"
            ;;
    esac
}

system::install_pkg() {
    local pkg="$1"
    local pm prefix
  
    log::header "📦 Installing system package $pkg as $(system::get_package_name "$pkg")"
    pkg=$(system::get_package_name "$pkg")
    pm=$(system::detect_pm)
  
    # Brew never needs sudo; others only if allowed
    if [[ "$pm" == "brew" ]]; then
        prefix=""
    elif flow::can_run_sudo "system package installation ($pkg)"; then
        prefix="sudo"
    else
        prefix=""
        log::warning "No sudo available; running $pm commands as user"
    fi
  
    case "$pm" in
        apt-get)
            # prevent hanging on apt
            timeout --kill-after=10s "${SYSTEM_INSTALL_TIMEOUT}"s ${prefix} apt-get update -qq
            timeout --kill-after=10s "${SYSTEM_INSTALL_TIMEOUT}"s ${prefix} apt-get install -y -qq --no-install-recommends "$pkg"
            ;;
        dnf)
            ${prefix} dnf install -y "$pkg"
            ;;
        yum)
            ${prefix} yum install -y "$pkg"
            ;;
        pacman)
            ${prefix} pacman -Syu --noconfirm "$pkg"
            ;;
        apk)
            ${prefix} apk update
            ${prefix} apk add --no-cache "$pkg"
          ;;
        brew)
            brew install "$pkg"
            ;;
        *)
            log::error "Unsupported pkg manager: $pm"
            exit 1
            ;;
    esac
    
    log::success "Installed $pkg via $pm"
}

system::install_yq_binary() {
    log::header "📦 Installing yq (by Mike Farah) via binary download..."
    local YQ_VERSION="v4.40.5" # Pinning to a known version
    local YQ_OS_TYPE=""
    local YQ_ARCH=""
    local machine_os
    local machine_arch
    local YQ_INSTALL_PATH="/usr/local/bin/yq" # Common location

    machine_os=$(uname -s)
    machine_arch=$(uname -m)

    case "$machine_os" in
        "Linux") YQ_OS_TYPE="linux" ;;
        "Darwin") YQ_OS_TYPE="darwin" ;;
        *)
            log::error "Unsupported OS for yq binary download: $machine_os. Please install yq manually."
            return 1
            ;;
    esac

    case "$machine_arch" in
        "x86_64"|"amd64") YQ_ARCH="amd64" ;;
        "aarch64"|"arm64") YQ_ARCH="arm64" ;;
        *)
            log::error "Unsupported architecture for yq binary download: $machine_arch. Please install yq manually."
            return 1
            ;;
    esac

    local YQ_DOWNLOAD_URL="https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_${YQ_OS_TYPE}_${YQ_ARCH}"
    local TEMP_YQ_DOWNLOAD_PATH="/tmp/yq_${YQ_OS_TYPE}_${YQ_ARCH}"

    log::info "Downloading yq from ${YQ_DOWNLOAD_URL} to ${TEMP_YQ_DOWNLOAD_PATH}"

    if ! curl -sSL "${YQ_DOWNLOAD_URL}" -o "${TEMP_YQ_DOWNLOAD_PATH}"; then
        log::error "Failed to download yq binary from ${YQ_DOWNLOAD_URL}."
        trash::safe_remove "${TEMP_YQ_DOWNLOAD_PATH}" --temp # Clean up
        return 1
    fi

    log::info "Moving yq binary to ${YQ_INSTALL_PATH} and setting permissions."
    # Try to move and set permissions, may require sudo
    if flow::maybe_run_sudo mv "${TEMP_YQ_DOWNLOAD_PATH}" "${YQ_INSTALL_PATH}"; then
        if flow::maybe_run_sudo chmod +x "${YQ_INSTALL_PATH}"; then
            # Verify the downloaded yq works
            if "${YQ_INSTALL_PATH}" --version >/dev/null 2>&1 ; then
                log::success "yq binary (Mike Farah's ${YQ_VERSION}) downloaded and installed to ${YQ_INSTALL_PATH}"
                # No need to rm TEMP_YQ_DOWNLOAD_PATH as mv handled it
                return 0
            else
                log::error "yq downloaded to ${YQ_INSTALL_PATH} but is not working correctly. Cleaning up."
                flow::maybe_run_sudo trash::safe_remove "${YQ_INSTALL_PATH}" --temp
                return 1
            fi
        else
            log::error "Failed to make yq binary executable at ${YQ_INSTALL_PATH}. Please check permissions. Cleaning up."
            flow::maybe_run_sudo trash::safe_remove "${YQ_INSTALL_PATH}" --temp # Cleanup if chmod failed
            return 1
        fi
    else
        log::error "Failed to move yq binary to ${YQ_INSTALL_PATH}. Please check permissions. Cleaning up."
        trash::safe_remove "${TEMP_YQ_DOWNLOAD_PATH}" --temp # mv failed, so temp file still exists
        return 1
    fi
}

system::check_and_install() {
    local cmd="$1"
    log::info "Checking for $cmd..."
    if system::is_command "$cmd"; then
        log::success "$cmd is already installed."
        return 0
    fi
  
    log::warning "$cmd not found. Installing…"
    if [[ "$cmd" == "yq" ]]; then
        if ! system::install_yq_binary; then
             log::error "Could not install $cmd using binary download method—please install it manually."
             exit "${ERROR_DEPENDENCY_MISSING}"
        fi
    else
        if ! system::install_pkg "$cmd"; then
            log::error "Could not install $cmd using package manager—please install it manually."
            exit "${ERROR_DEPENDENCY_MISSING}"
        fi
    fi
  
    if system::is_command "$cmd"; then
        log::success "$cmd installed successfully."
    else
        # This case should ideally be caught by the specific install functions' error handling
        log::error "Installation of $cmd was reported as successful, but the command is still not found."
        exit "${ERROR_DEPENDENCY_MISSING}"
    fi
}

# Update package lists
system::update() {
    log::header "🔄 Updating system package lists"
    if system::is_command "apt-get"; then
        # Check if we have permission to update package lists
        local update_cmd="apt-get"
        local can_update=true
        
        # Check if we need elevated permissions by testing write access to apt directories
        if [[ ! -w /var/lib/apt/lists ]]; then
            # We need elevated permissions
            # Save current SUDO_MODE to restore later
            local saved_sudo_mode="${SUDO_MODE:-}"
            # Temporarily set SUDO_MODE to skip to avoid error exit
            export SUDO_MODE="skip"
            
            if flow::can_run_sudo "system package list update (apt-get update)"; then
                update_cmd="sudo apt-get"
                can_update=true
            else
                # Can't run sudo and can't update without it
                can_update=false
            fi
            
            # Restore original SUDO_MODE
            export SUDO_MODE="$saved_sudo_mode"
        fi
        
        if [[ "$can_update" == "true" ]]; then
            if $update_cmd update; then
                log::success "apt-get update complete"
            else
                log::warning "apt-get update failed, but continuing anyway"
            fi
        else
            log::warning "⚠️  Skipping package list update - insufficient permissions"
            log::warning "   To update package lists, run: sudo apt-get update"
            log::warning "   Or run setup with: sudo ./scripts/manage.sh setup"
            log::info "Continuing with potentially outdated package information..."
        fi
    elif system::is_command "brew"; then
        if brew update; then
            log::success "Homebrew update complete"
        else
            log::warning "Homebrew update failed, but continuing anyway"
        fi
    else
        log::warning "No supported package manager found for update - skipping"
    fi
}

# Upgrade installed packages
system::upgrade() {
    log::header "⬆️ Upgrading system packages"
    if system::is_command "apt-get"; then
        # Check if we have permission to upgrade packages
        local upgrade_cmd="apt-get"
        local can_upgrade=true
        
        # Check if we need elevated permissions
        if [[ ! -w /var/lib/dpkg ]]; then
            # We need elevated permissions
            # Save current SUDO_MODE to restore later
            local saved_sudo_mode="${SUDO_MODE:-}"
            # Temporarily set SUDO_MODE to skip to avoid error exit
            export SUDO_MODE="skip"
            
            if flow::can_run_sudo "system package upgrade (apt-get upgrade)"; then
                upgrade_cmd="sudo apt-get"
                can_upgrade=true
            else
                # Can't run sudo and can't upgrade without it
                can_upgrade=false
            fi
            
            # Restore original SUDO_MODE
            export SUDO_MODE="$saved_sudo_mode"
        fi
        
        if [[ "$can_upgrade" == "true" ]]; then
            if $upgrade_cmd -y upgrade; then
                log::success "apt-get upgrade complete"
            else
                log::warning "apt-get upgrade failed, but continuing anyway"
            fi
        else
            log::warning "⚠️  Skipping package upgrade - insufficient permissions"
            log::warning "   To upgrade packages, run: sudo apt-get upgrade"
            log::warning "   Or run setup with: sudo ./scripts/manage.sh setup"
            log::info "Continuing with current package versions..."
        fi
    elif system::is_command "brew"; then
        if brew upgrade; then
            log::success "Homebrew upgrade complete"
        else
            log::warning "Homebrew upgrade failed, but continuing anyway"
        fi
    else
        log::warning "No supported package manager found for upgrade - skipping"
    fi
} 

# Limits the number of system update calls
system::should_run_update() {
    if system::is_command "apt-get"; then
        # Use apt list timestamp to throttle updates
        local last_update
        last_update=$(stat -c %Y /var/lib/apt/lists/)
        local current_time
        current_time=$(date +%s)
        local update_interval=$((24 * 60 * 60))
        if ((current_time - last_update > update_interval)); then
            return 0
        else
            return 1
        fi
    elif system::is_command "brew"; then
        # Always run brew update
        return 0
    else
        # Unknown package manager: skip
        return 1
    fi
}

# Limit the number of system upgrade calls
system::should_run_upgrade() {
    if system::is_command "apt-get"; then
        # Use dpkg status timestamp to throttle upgrades
        local last_upgrade
        last_upgrade=$(stat -c %Y /var/lib/dpkg/status)
        local current_time
        current_time=$(date +%s)
        local upgrade_interval=$((7 * 24 * 60 * 60))
        if ((current_time - last_upgrade > upgrade_interval)); then
            return 0
        else
            return 1
        fi
    elif system::is_command "brew"; then
        # Always run brew upgrade
        return 0
    else
        # Unknown package manager: skip
        return 1
    fi
}

system::update_and_upgrade() {
    if system::should_run_update; then
        system::update
    else
        log::info "Skipping system update - last update was less than 24 hours ago"
    fi
    if system::should_run_upgrade; then
        system::upgrade
    else
        log::info "Skipping system upgrade - last upgrade was less than 1 week ago"
    fi
}

# Purges apt update notifier, which can cause hangs on some systems
system::purge_apt_update_notifier() {
    if system::is_command "apt-get"; then
        log::info "Purging apt update-notifier packages (if present)..."
        flow::maybe_run_sudo apt-get purge -y update-notifier update-notifier-common || log::info "Update notifier not present or already purged."
        log::success "Finished attempting to purge update-notifier."
    else
        log::info "Not an apt-based system, skipping update-notifier purge."
    fi
}

# Cross-platform path canonicalization: resolves symlinks if possible, else lexical fallback
system::canonicalize() {
  local input="$1"
  # Use realpath if available to fully resolve symlinks
  if system::is_command realpath; then
    realpath "$input"
  elif system::is_command readlink; then
    # readlink -f resolves symlinks and canonicalizes
    readlink -f "$input"
  else
    # Pure Bash fallback: expand relative paths and normalize . and .. components
    if [[ "$input" != /* ]]; then
      input="$PWD/$input"
    fi
    # Remove '/./' segments
    input="${input//\/\.\//\/}"
    # Split into components
    local IFS='/'
    read -r -a _segments <<< "$input"
    local _result=()
    for _seg in "${_segments[@]}"; do
      case "$_seg" in
        ''|'.') continue ;;
        '..')
          if (( ${#_result[@]} > 0 )); then
            unset '_result[${#_result[@]}-1]'
          fi
          ;;
        *) _result+=("$_seg") ;;
      esac
    done
    # Reconstruct the path
    local canonical="/"
    for _seg in "${_result[@]}"; do
      canonical="${canonical%/}/$_seg"
    done
    echo "$canonical"
  fi
}

# Check if NVIDIA GPU is present
system::has_nvidia_gpu() {
    # Check if nvidia-smi exists and can detect GPU
    if system::is_command "nvidia-smi"; then
        nvidia-smi >/dev/null 2>&1 && return 0
    fi
    
    # Fallback: check lspci for NVIDIA devices
    if system::is_command "lspci"; then
        lspci | grep -i nvidia >/dev/null 2>&1 && return 0
    fi
    
    return 1
}

# Install NVIDIA Container Runtime
system::install_nvidia_container_runtime() {
    # Check if already installed
    if system::is_command "nvidia-container-runtime"; then
        log::info "NVIDIA Container Runtime already installed"
        return 0
    fi
    
    # Check sudo availability once before attempting installation
    if ! flow::can_run_sudo "NVIDIA Container Runtime installation"; then
        log::warning "NVIDIA Container Runtime needs to be installed but sudo is not available"
        log::info "The following changes would be made:"
        log::info "  - Add NVIDIA container toolkit repository"
        log::info "  - Install nvidia-container-toolkit package"
        log::info "  - Configure Docker daemon for GPU support"
        log::info "To install NVIDIA Container Runtime, run with sudo or use --sudo-mode error"
        return 1
    fi
    
    local distro
    distro=$(system::detect_pm)
    
    case "$distro" in
        apt-get)
            system::install_nvidia_runtime_apt "$distro"
            ;;
        dnf|yum)
            system::install_nvidia_runtime_yum "$distro"
            ;;
        pacman)
            system::install_nvidia_runtime_pacman
            ;;
        *)
            log::warn "Unsupported package manager for NVIDIA Container Runtime: $distro"
            return 1
            ;;
    esac
}

# Install NVIDIA runtime on Ubuntu/Debian
system::install_nvidia_runtime_apt() {
    local distro="$1"
    log::info "Installing NVIDIA Container Runtime for Ubuntu/Debian..."
    
    # Set up the repository (sudo availability already verified by caller)
    log::info "Adding NVIDIA container toolkit repository..."
    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | \
        sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
    
    curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
        sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
        sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
    
    # Update and install
    log::info "Updating package lists and installing nvidia-container-toolkit..."
    sudo apt-get update -qq
    sudo apt-get install -y nvidia-container-toolkit
    
    log::success "NVIDIA Container Runtime installed successfully"
}

# Install NVIDIA runtime on CentOS/RHEL/Fedora  
system::install_nvidia_runtime_yum() {
    local distro="$1"
    log::info "Installing NVIDIA Container Runtime for CentOS/RHEL/Fedora..."
    
    # Set up the repository (sudo availability already verified by caller)
    log::info "Adding NVIDIA container toolkit repository..."
    curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | \
        sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo
    
    # Install
    log::info "Installing nvidia-container-toolkit package..."
    sudo "$distro" install -y nvidia-container-toolkit
    
    log::success "NVIDIA Container Runtime installed successfully"
}

# Install NVIDIA runtime on Arch Linux
system::install_nvidia_runtime_pacman() {
    log::info "Installing NVIDIA Container Runtime for Arch Linux..."
    
    # Install from AUR (requires manual intervention)
    log::warn "Arch Linux requires manual AUR installation of nvidia-container-toolkit"
    log::info "Please install manually: yay -S nvidia-container-toolkit"
    return 1
}

# Additional system utility functions for Judge0 and other resources

#######################################
# Get total memory in MB
# Returns: Total memory in MB as integer
#######################################
system::get_total_memory_mb() {
    if system::is_command "free"; then
        free -m | awk '/^Mem:/{print $2}'
    elif [[ -f /proc/meminfo ]]; then
        # Fallback: parse /proc/meminfo
        awk '/^MemTotal:/{printf "%.0f", $2/1024}' /proc/meminfo
    else
        # Default fallback if neither command works
        echo "2048"
    fi
}

#######################################
# Get free disk space in GB for a given path
# Arguments:
#   $1 - path to check (defaults to $HOME)
# Returns: Free space in GB as integer
#######################################
system::get_free_space_gb() {
    local path="${1:-$HOME}"
    
    if system::is_command "df"; then
        # Use df to get available space in GB
        df -BG "$path" 2>/dev/null | awk 'NR==2{gsub(/G/,"",$4); print int($4)}'
    elif system::is_command "du"; then
        # Fallback using du (less reliable)
        local available=$(du -sb "$path" 2>/dev/null | awk '{print int((1024*1024*1024*10 - $1)/(1024*1024*1024))}')
        echo "${available:-10}"
    else
        # Default fallback
        echo "10"
    fi
}