#!/usr/bin/env bash
# system.sh - Cross-platform package manager helpers
set -euo pipefail

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/flow.sh"

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
    elif flow::can_run_sudo; then
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
        rm -f "${TEMP_YQ_DOWNLOAD_PATH}" # Clean up
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
                flow::maybe_run_sudo rm -f "${YQ_INSTALL_PATH}"
                return 1
            fi
        else
            log::error "Failed to make yq binary executable at ${YQ_INSTALL_PATH}. Please check permissions. Cleaning up."
            flow::maybe_run_sudo rm -f "${YQ_INSTALL_PATH}" # Cleanup if chmod failed
            return 1
        fi
    else
        log::error "Failed to move yq binary to ${YQ_INSTALL_PATH}. Please check permissions. Cleaning up."
        rm -f "${TEMP_YQ_DOWNLOAD_PATH}" # mv failed, so temp file still exists
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
        # If we can sudo, prefix apt commands; otherwise run as current user
        local update_cmd="apt-get"
        if flow::can_run_sudo; then
            update_cmd="sudo apt-get"
        else
            log::info "No sudo available, running apt-get update as current user"
        fi
        $update_cmd update
        log::success "apt-get update complete"
    elif system::is_command "brew"; then
        brew update
        log::success "Homebrew update complete"
    else
        log::error "No supported package manager found for update"
    fi
}

# Upgrade installed packages
system::upgrade() {
    log::header "⬆️ Upgrading system packages"
    if system::is_command "apt-get"; then
        # If we can sudo, prefix apt commands; otherwise run as current user
        local upgrade_cmd="apt-get"
        if flow::can_run_sudo; then
            upgrade_cmd="sudo apt-get"
        else
            log::info "No sudo available, running apt-get upgrade as current user"
        fi
        $upgrade_cmd -y upgrade
        log::success "apt-get upgrade complete"
    elif system::is_command "brew"; then
        brew upgrade
        log::success "Homebrew upgrade complete"
    else
        log::error "No supported package manager found for upgrade"
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