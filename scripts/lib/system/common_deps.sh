#!/usr/bin/env bash
set -euo pipefail

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$var_LIB_SYSTEM_DIR/system_commands.sh"
# shellcheck disable=SC1091
source "$var_LIB_DIR/runtimes/helm.sh"

common_deps::check_and_install() {
    log::header "⚙️ Checking common dependencies (curl, jq)..."

    system::check_and_install "curl"
    system::check_and_install "jq"

    # Install Linux-only utilities only on supported package managers
    local pm
    pm=$(system::detect_pm)
    case "$pm" in
        apt-get|dnf|yum|pacman|apk)
            system::check_and_install "nproc"
            system::check_and_install "free"
            system::check_and_install "systemctl"
            ;;
        *)
            log::info "Skipping Linux-only dependencies (nproc, free, systemctl) on package manager $pm"
            ;;
    esac

    system::check_and_install "bc"
    system::check_and_install "awk"
    system::check_and_install "sed"
    system::check_and_install "grep"
    system::check_and_install "mkdir"
    system::check_and_install "script"
    system::check_and_install "yq"
    system::check_and_install "ffmpeg"
    
    # Install native module build dependencies for isolated-vm and other packages
    common_deps::setup_native_build_env
    
    # Install Helm CLI for Kubernetes package management
    helm::check_and_install

    # Install NVIDIA container runtime if NVIDIA GPU detected
    common_deps::setup_nvidia_runtime

    log::success "✅ Common dependencies checked/installed."
    return 0
}

# Helper to install packages that don't create commands (like -dev packages)
common_deps::install_dev_package() {
    local pkg="$1"
    local check_for="${2:-$1}"  # Optional: what to check for instead
    
    # For dev packages, check if already installed via package manager
    if [[ "$(system::detect_pm)" == "apt-get" ]]; then
        if dpkg -l "$pkg" 2>/dev/null | grep -q "^ii"; then
            log::success "$pkg is already installed."
            return 0
        fi
    fi
    
    log::info "Installing $pkg..."
    if system::install_pkg "$pkg"; then
        log::success "$pkg installed successfully."
        return 0
    else
        log::warning "Failed to install $pkg"
        return 1
    fi
}

# Setup native module build environment for Node.js packages like isolated-vm
common_deps::setup_native_build_env() {
    log::info "🔧 Setting up native module build environment..."
    
    local pm
    pm=$(system::detect_pm)
    
    case "$pm" in
        apt-get)
            # Install build-essential meta-package if gcc not present
            if ! system::is_command "gcc"; then
                common_deps::install_dev_package "build-essential" "gcc"
            fi
            system::check_and_install "gcc"
            system::check_and_install "python3"
            # python3-dev is a dev package that doesn't create a command
            common_deps::install_dev_package "python3-dev"
            system::check_and_install "make"
            system::check_and_install "g++"
            ;;
        dnf|yum)
            system::check_and_install "gcc"
            system::check_and_install "gcc-c++"
            system::check_and_install "make"
            system::check_and_install "python3"
            # python3-devel is a dev package
            common_deps::install_dev_package "python3-devel"
            ;;
        pacman)
            # base-devel is a meta-package group
            if ! system::is_command "gcc"; then
                common_deps::install_dev_package "base-devel" "gcc"
            fi
            system::check_and_install "python"
            ;;
        apk)
            # build-base is a meta-package
            if ! system::is_command "gcc"; then
                common_deps::install_dev_package "build-base" "gcc"
            fi
            system::check_and_install "python3"
            common_deps::install_dev_package "python3-dev"
            system::check_and_install "make"
            system::check_and_install "g++"
            ;;
        *)
            log::warning "Unknown package manager $pm, skipping native build environment setup"
            log::info "Please ensure you have: build-essential, python3, make, g++"
            return 0
            ;;
    esac
    
    log::success "✅ Native module build environment ready"
    return 0
}

# Setup NVIDIA container runtime if GPU detected
common_deps::setup_nvidia_runtime() {
    # Check if NVIDIA GPU is present
    if system::has_nvidia_gpu; then
        log::info "NVIDIA GPU detected, installing container runtime..."
        
        if system::install_nvidia_container_runtime; then
            log::success "✅ NVIDIA Container Runtime installed"
        else
            log::warn "Failed to install NVIDIA Container Runtime (GPU support may be limited)"
        fi
    else
        log::info "No NVIDIA GPU detected, skipping container runtime installation"
    fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    common_deps::check_and_install "$@"
fi 