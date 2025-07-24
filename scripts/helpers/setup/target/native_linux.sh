#!/usr/bin/env bash
# Posix-compliant script to setup the project for native Linux development/production
set -euo pipefail

SETUP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../utils/pnpm_tools.sh"
# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../utils/flow.sh"

native_linux::setup_native_linux() {
    log::header "Setting up native Linux development/production..."

    # Setup pnpm and generate Prisma client
    pnpm_tools::setup
    
    # Setup local resources if requested
    if [[ -n "${RESOURCES:-}" && "$RESOURCES" != "none" ]]; then
        log::header "🔧 Setting up local resources: $RESOURCES"
        
        local resources_script="${SETUP_TARGET_DIR}/../../../resources/index.sh"
        if [[ -f "$resources_script" ]]; then
            bash "$resources_script" \
                --action install \
                --resources "$RESOURCES" \
                --yes "${YES:-no}"
        else
            log::error "Resource setup script not found: $resources_script"
            log::info "Skipping resource setup"
        fi
    else
        log::info "No local resources requested (use --resources to specify)"
    fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    native_linux::setup_native_linux "$@"
fi 