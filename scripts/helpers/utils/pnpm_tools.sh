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

# Helper function to run commands as the actual user when running under sudo
# This ensures that files created by pnpm operations are owned by the actual user, not root
pnpm_tools::run_as_user() {
    local cmd="$*"
    if [[ -n "${SUDO_USER:-}" ]]; then
        # Running with sudo - execute as the actual user
        sudo -u "${SUDO_USER}" bash -c "$cmd"
    else
        # Not running with sudo - execute normally
        bash -c "$cmd"
    fi
}

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
        pnpm_tools::run_as_user "corepack enable" || true
        pnpm_tools::run_as_user "corepack prepare pnpm@latest --activate" || true
    fi

    # If pnpm is still not available, install it locally for the user
    if ! system::is_command "pnpm"; then
        echo "pnpm not found, installing standalone binary..."
        
        # Set PNPM_HOME based on actual user's home directory
        local actual_home="$HOME"
        if [[ -n "${SUDO_USER:-}" ]]; then
            actual_home=$(eval echo "~${SUDO_USER}")
        fi
        export PNPM_HOME="$actual_home/.local/share/pnpm"
        
        # Create directory with proper ownership
        if [[ -n "${SUDO_USER:-}" ]]; then
            sudo -u "${SUDO_USER}" mkdir -p "$PNPM_HOME"
        else
            mkdir -p "$PNPM_HOME"
        fi
        
        # Download pnpm installer as actual user
        if [[ -n "${SUDO_USER:-}" ]]; then
            sudo -u "${SUDO_USER}" bash -c "export PNPM_HOME='$PNPM_HOME' && curl -fsSL https://get.pnpm.io/install.sh | bash -"
        else
            nodejs::download_with_retry "https://get.pnpm.io/install.sh" | bash -
        fi
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
        # Run pnpm generate as actual user to ensure proper file ownership
        pnpm_tools::run_as_user "cd '$var_ROOT_DIR' && pnpm --filter @vrooli/prisma run generate"
        
        # Create data directory with proper ownership
        if [[ -n "${SUDO_USER:-}" ]]; then
            sudo -u "${SUDO_USER}" mkdir -p "$var_DATA_DIR"
            echo "$CURRENT_HASH" | sudo -u "${SUDO_USER}" tee "$HASH_FILE" > /dev/null
        else
            mkdir -p "$var_DATA_DIR"
            echo "$CURRENT_HASH" > "$HASH_FILE"
        fi
    fi
}

# Function to enable Corepack, install pnpm dependencies, and generate Prisma client
pnpm_tools::setup() {
    log::header "🔧 Enabling Corepack and installing dependencies..."
    pnpm_tools::ensure_pnpm

    # Fix ownership of existing directories if running with sudo
    if [[ -n "${SUDO_USER:-}" ]]; then
        log::info "Fixing ownership of existing project directories..."
        
        # Fix node_modules directories
        log::info "  - Fixing node_modules directories..."
        find "$var_ROOT_DIR" -type d -name "node_modules" -exec chown -R "${SUDO_USER}:${SUDO_USER}" {} + 2>/dev/null || true
        
        # Fix dist directories (build outputs)
        log::info "  - Fixing dist directories..."
        find "$var_ROOT_DIR/packages" -type d -name "dist" ! -path "*/node_modules/*" -exec chown -R "${SUDO_USER}:${SUDO_USER}" {} + 2>/dev/null || true
        
        # Fix test-dist directories (test build outputs)
        log::info "  - Fixing test-dist directories..."
        find "$var_ROOT_DIR/packages" -type d -name "test-dist" ! -path "*/node_modules/*" -exec chown -R "${SUDO_USER}:${SUDO_USER}" {} + 2>/dev/null || true
        
        # Fix .tsbuildinfo files (TypeScript build cache)
        log::info "  - Fixing TypeScript build info files..."
        find "$var_ROOT_DIR/packages" -type f -name ".tsbuildinfo" -exec chown "${SUDO_USER}:${SUDO_USER}" {} + 2>/dev/null || true
        
        # Fix the data directory if it exists
        if [[ -d "$var_DATA_DIR" ]]; then
            log::info "  - Fixing data directory..."
            chown -R "${SUDO_USER}:${SUDO_USER}" "$var_DATA_DIR" 2>/dev/null || true
        fi
        
        log::success "Ownership fixes complete"
    fi

    log::info "Installing dependencies via pnpm..."
    # Run pnpm install as actual user to ensure node_modules are owned by the user
    pnpm_tools::run_as_user "cd '$var_ROOT_DIR' && pnpm install"

    # Generate Prisma client if (and only if) the schema changed
    pnpm_tools::generate_prisma_client

    # Setup Vrooli CLI after dependencies are installed
    if [[ -f "${SETUP_DIR}/../setup/vrooli_cli.sh" ]]; then
        # shellcheck disable=SC1091
        source "${SETUP_DIR}/../setup/vrooli_cli.sh"
        vrooli_cli::setup
    fi

    # Fix hardcoded configuration paths
    pnpm_tools::fix_config_paths
}

# Function to fix hardcoded paths in configuration files
pnpm_tools::fix_config_paths() {
    log::header "🔧 Fixing hardcoded configuration paths..."
    
    # Get the actual project root (handle symlinks)
    local project_root
    project_root=$(cd "$var_ROOT_DIR" && pwd -P)
    
    # Fix VS Code settings if they exist
    local vscode_settings="$project_root/.vscode/settings.json"
    if [[ -f "$vscode_settings" ]]; then
        log::info "Updating VS Code settings paths..."
        
        # Ensure jq is installed for JSON manipulation
        system::check_and_install "jq"
        
        if command -v jq >/dev/null 2>&1; then
            # Create backup
            cp "$vscode_settings" "${vscode_settings}.backup" 2>/dev/null || true
            
            # Update any hardcoded paths to use the current project root
            # This handles paths like /root/Programming/Vrooli or /root/Vrooli
            jq --arg root "$project_root" '
                if ."yaml.schemas" then
                    ."yaml.schemas" |= with_entries(
                        .value |= gsub("/root/[^/]+/Vrooli"; $root)
                    )
                else . end
            ' "$vscode_settings" > "${vscode_settings}.tmp" && \
            mv "${vscode_settings}.tmp" "$vscode_settings"
            
            # Fix ownership if running with sudo
            if [[ -n "${SUDO_USER:-}" ]]; then
                chown "${SUDO_USER}:${SUDO_USER}" "$vscode_settings" 2>/dev/null || true
            fi
            
            log::success "VS Code settings updated"
        fi
    fi
    
    # Fix integration test paths
    local integration_setup="$project_root/packages/integration/src/setup/global-setup.ts"
    if [[ -f "$integration_setup" ]]; then
        log::info "Updating integration test paths..."
        
        # Use sed to update PROJECT_DIR
        # This is safer than parsing TypeScript with jq
        sed -i.backup "s|process\.env\.PROJECT_DIR = \"/root/Vrooli\"|process.env.PROJECT_DIR = \"$project_root\"|g" "$integration_setup"
        
        # Fix ownership if running with sudo
        if [[ -n "${SUDO_USER:-}" ]]; then
            chown "${SUDO_USER}:${SUDO_USER}" "$integration_setup" 2>/dev/null || true
        fi
        
        log::success "Integration test paths updated"
    fi
    
    # Clear TypeScript caches after path updates
    log::info "Clearing TypeScript caches..."
    find "$project_root" -name ".tsbuildinfo" -type f -delete 2>/dev/null || true
    find "$project_root" -path "*/node_modules/.cache" -type d -exec rm -rf {} + 2>/dev/null || true
    
    log::success "Configuration paths fixed"
}