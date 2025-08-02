#!/usr/bin/env bash
set -euo pipefail

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/system.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/helm.sh"

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
    
    # Install Helm CLI for Kubernetes package management
    helm::check_and_install

    # Install NVIDIA container runtime if NVIDIA GPU detected
    common_deps::setup_nvidia_runtime

    log::success "✅ Common dependencies checked/installed."
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