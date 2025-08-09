#!/usr/bin/env bash
set -euo pipefail

APP_LIFECYCLE_SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_SETUP_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"

# Install Helm CLI for Kubernetes package management
helm::check_and_install() {
    if ! system::is_command "helm"; then
        log::info "📦 Installing Helm..."
        # Use official Helm installation script to fetch and install the latest version
        # Adding retries for network reliability
        local attempt_num=1
        local max_attempts=3
        until curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash; do
            if (( attempt_num == max_attempts )); then
                log::error "Failed to install Helm after $max_attempts attempts."
                return 1
            fi
            log::info "Helm installation attempt $attempt_num failed. Retrying in 5 seconds..."
            sleep 5
            attempt_num=$((attempt_num+1))
        done
        log::success "Helm installed successfully"
    else
        log::info "helm is already installed"
    fi
    
    # Ensure HashiCorp repo is added and updated (useful for Vault charts)
    if system::is_command "helm"; then
        log::info "Adding/Updating HashiCorp Helm repository..."
        helm repo add hashicorp https://helm.releases.hashicorp.com > /dev/null 2>&1 || log::warning "Failed to add HashiCorp repo (maybe already added)."
        helm repo update hashicorp > /dev/null 2>&1 || log::warning "Failed to update HashiCorp Helm repository."
    fi
    
    return 0
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    helm::check_and_install "$@"
fi