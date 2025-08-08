#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Vrooli-Specific Setup Script
#
# This script contains all the Vrooli-specific setup logic that was previously
# in setup::monorepo. It's now a standalone script that the lifecycle engine
# calls via the service.json configuration.
#
# This separation allows the main setup.sh to become a thin wrapper around
# the lifecycle engine, while this script handles Vrooli-specific concerns.
################################################################################

APP_LIFECYCLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DIR}/../../lib/utils/var.sh"

# Now use the variables for cleaner paths
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/env.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/jwt.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/domainCheck.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/ports.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/proxy.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/ci.sh"
# shellcheck disable=SC1091
source "${var_APP_LIFECYCLE_SETUP_DIR}/index.sh"

# Export outputs for subsequent steps
export var_ROOT_DIR
export VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")

main() {
    log::header "🔧 Vrooli-Specific Setup"
    
    # Get environment variables from lifecycle
    local IS_CI="${IS_CI:-no}"
    local CLEAN="${CLEAN:-no}"
    local LOCATION="${LOCATION:-}"
    local ENVIRONMENT="${ENVIRONMENT:-development}"
    
    # JWT key generation
    if ! flow::is_yes "$IS_CI"; then
        log::info "Generating JWT key pair..."
        jwt::generate_key_pair
    fi
    
    # Load secrets and environment
    log::info "Loading secrets..."
    env::load_secrets
    check_location_if_not_set
    env::construct_derived_secrets
    
    # Clean volumes if requested
    if flow::is_yes "$CLEAN"; then
        log::info "Cleaning volumes and caches..."
        clean::main
    fi
    
    # Remote-specific setup
    if ! flow::is_yes "$IS_CI" && env::is_location_remote; then
        log::info "Configuring for remote environment..."
        
        # Purge apt update notifier (Ubuntu-specific)
        if command -v apt-get &> /dev/null; then
            system::purge_apt_update_notifier
        fi
        
        # Check and free ports
        log::info "Checking required ports..."
        ports::check_and_free "${PORT_DB:-5432}"
        ports::check_and_free "${PORT_JOBS:-4001}"
        ports::check_and_free "${PORT_REDIS:-6379}"
        ports::check_and_free "${PORT_SERVER:-5329}"
        ports::check_and_free "${PORT_UI:-3000}"
        
        # Setup proxy if needed
        proxy::setup
        
        # Setup firewall
        firewall::setup "${ENVIRONMENT}"
    fi
    
    # Local development specific
    if ! flow::is_yes "$IS_CI" && env::is_location_local; then
        log::info "Configuring for local development..."
        
        # Generate CI keys if needed
        ci::generate_key_pair
    fi
    
    # Stripe CLI setup for development
    if env::in_development && ! flow::is_yes "$IS_CI"; then
        if [[ -n "${STRIPE_API_KEY-}" ]] || \
           [[ -n "${STRIPE_SECRET_KEY-}" ]] || \
           [[ -n "${STRIPE_PUBLISHABLE_KEY-}" ]] || \
           [[ -n "${STRIPE_WEBHOOK_SECRET-}" ]]; then
            log::info "Setting up Stripe CLI..."
            if stripe_cli::setup; then
                log::success "✅ Stripe CLI configured"
            else
                log::warning "⚠️  Stripe CLI setup incomplete"
            fi
        fi
    fi
    
    # Install test tools for development/CI
    if flow::is_yes "$IS_CI" || env::in_development; then
        log::info "Installing test tools..."
        bats::install
        shellcheck::install
    fi
    
    # Repository hooks
    if command -v repository::run_hook &> /dev/null; then
        if repository::run_hook "postInstall" 2>/dev/null; then
            log::success "✅ Repository postInstall hook completed"
        fi
    fi
    
    # Schedule backups for production
    if ! flow::is_yes "$IS_CI" && env::in_production; then
        log::info "Scheduling production backups..."
        "${var_ROOT_DIR}/scripts/manage.sh backup"
    fi
    
    log::success "✅ Vrooli-specific setup complete"
    
    # Output variables for lifecycle engine
    echo "var_ROOT_DIR=$var_ROOT_DIR"
    echo "VERSION=$VERSION"
}

# Only run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi