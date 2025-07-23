#!/usr/bin/env bash
# Enable Corepack, activate pnpm, install dependencies, and generate the Prisma client.
set -euo pipefail

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/system.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/var.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../setup/nodejs.sh"

# Ensure pnpm is available, using corepack if possible, otherwise fallback to npm install -g pnpm
pnpm_tools::ensure_pnpm() {
    # Check if Node.js is available to the actual user (important when running with sudo)
    local need_node_install=false
    
    if [[ -n "${SUDO_USER:-}" ]]; then
        # Running with sudo - check if Node.js is available to the actual user
        if ! sudo -u "${SUDO_USER}" bash -c 'command -v node >/dev/null 2>&1'; then
            need_node_install=true
            log::info "Node.js not found for user ${SUDO_USER}. Installing Node.js..."
        else
            log::info "Node.js is available to ${SUDO_USER}: $(sudo -u "${SUDO_USER}" bash -c 'node --version')"
        fi
    else
        # Not running with sudo - check normally
        if ! command -v node >/dev/null 2>&1; then
            need_node_install=true
            log::info "Node.js not found. Installing Node.js first..."
        else
            log::info "Node.js is already installed: $(node --version)"
        fi
    fi
    
    # Install Node.js if needed
    if [[ "$need_node_install" == "true" ]]; then
        nodejs::check_and_install || {
            log::error "Failed to install Node.js"
            return 1
        }
        
        # Source Node.js environment to make node available in current shell
        nodejs::source_environment
    fi
    # Try to use corepack if available and pnpm is available after activation
    if system::is_command "corepack"; then
        corepack enable || true
        corepack prepare pnpm@latest --activate || true
    fi

    # If pnpm is still not available, install it locally for the user
    if ! system::is_command "pnpm"; then
        echo "pnpm not found, installing standalone binary..."
        export PNPM_HOME="$HOME/.local/share/pnpm"
        mkdir -p "$PNPM_HOME"
        # Download pnpm installer with retry
        nodejs::download_with_retry "https://get.pnpm.io/install.sh" | bash -
        export PATH="$PNPM_HOME:$PATH"
    fi

    if ! system::is_command "pnpm"; then
        log::error "pnpm is still not available after attempted install. Exiting."
        exit 1
    fi
}

pnpm_tools::generate_prisma_client() {
    HASH_FILE="${var_DATA_DIR}/schema-hash"

    # Compute current schema hash
    if system::is_command "shasum"; then
        CURRENT_HASH=$(shasum -a 256 "$var_DB_SCHEMA_FILE" | awk '{print $1}')
    elif system::is_command "sha256sum"; then
        CURRENT_HASH=$(sha256sum "$var_DB_SCHEMA_FILE" | awk '{print $1}')
    else
        log::error "Neither shasum nor sha256sum found; cannot compute schema hash"
        exit 1
    fi

    # Read previous hash (if any)
    PREV_HASH=""
    if [ -f "$HASH_FILE" ]; then
        PREV_HASH=$(cat "$HASH_FILE")
    fi

    # Compare and decide whether to regenerate
    if [ "$CURRENT_HASH" = "$PREV_HASH" ]; then
        log::info "Schema unchanged; skipping Prisma client generation"
    else
        log::info "Schema changed; generating Prisma client..."
        pnpm --filter @vrooli/prisma run generate
        mkdir -p "$var_DATA_DIR"
        echo "$CURRENT_HASH" > "$HASH_FILE"
    fi
}

# Function to enable Corepack, install pnpm dependencies, and generate Prisma client
pnpm_tools::setup() {
    log::header "🔧 Enabling Corepack and installing dependencies..."
    pnpm_tools::ensure_pnpm

    log::info "Installing dependencies via pnpm..."
    pnpm install

    # Generate Prisma client if (and only if) the schema changed
    pnpm_tools::generate_prisma_client

    # Setup Vrooli CLI after dependencies are installed
    if [[ -f "${SETUP_DIR}/../setup/vrooli_cli.sh" ]]; then
        # shellcheck disable=SC1091
        source "${SETUP_DIR}/../setup/vrooli_cli.sh"
        vrooli_cli::setup
    fi
}