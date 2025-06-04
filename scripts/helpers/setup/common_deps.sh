#!/usr/bin/env bash
set -euo pipefail

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/system.sh"

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

    log::success "✅ Common dependencies checked/installed."
    return 0
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    common_deps::check_and_install "$@"
fi 