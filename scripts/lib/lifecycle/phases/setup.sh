#!/usr/bin/env bash
################################################################################
# Universal Setup Phase Handler
# 
# Handles generic setup tasks applicable to any application:
# - System preparation
# - Dependency installation
# - Docker/container setup
# - Network diagnostics
# - Generic resource installation
#
# App-specific logic should be in app/lifecycle/setup.sh
################################################################################

set -euo pipefail

# Get script directory
LIB_LIFECYCLE_PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/target_matcher.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/permissions.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/permissions.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/clock.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/network_diagnostics.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/common_deps.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/config.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/config.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/docker.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/firewall.sh"
# shellcheck disable=SC1091
source "${var_LIB_DEPS_DIR}/bats.sh"
# shellcheck disable=SC1091
source "${var_LIB_DEPS_DIR}/shellcheck.sh"

################################################################################
# Main Setup Logic
################################################################################

#######################################
# Run universal setup tasks
# Handles all generic setup operations
# Globals:
#   TARGET
#   CLEAN
#   IS_CI
#   SUDO_MODE
#   ENVIRONMENT
#   RESOURCES
# Returns:
#   0 on success, 1 on failure
#######################################
setup::universal_main() {
    # Initialize phase
    phase::init "Setup"
    
    # Get parameters from environment or defaults
    local target="${TARGET:-native-linux}"
    local clean="${CLEAN:-no}"
    local is_ci="${IS_CI:-no}"
    local sudo_mode="${SUDO_MODE:-ask}"
    local environment="${ENVIRONMENT:-development}"
    local resources="${RESOURCES:-enabled}"
    
    log::info "Universal setup starting..."
    log::debug "Parameters:"
    log::debug "  Target: $target"
    log::debug "  Clean: $clean"
    log::debug "  CI: $is_ci"
    log::debug "  Sudo: $sudo_mode"
    log::debug "  Environment: $environment"
    log::debug "  Resources: $resources"
    
    # Validate and canonicalize target
    if ! canonical=$(target_matcher::match_target "$target"); then
        log::error "Invalid target: $target"
        return "$ERROR_USAGE"
    fi
    export TARGET="$canonical"
    
    # In standalone mode, default to skip sudo unless explicitly set
    if phase::is_standalone && [[ -z "${SUDO_MODE_EXPLICIT:-}" ]]; then
        log::info "Standalone app detected, defaulting to SUDO_MODE=skip"
        export SUDO_MODE="skip"
    fi
    
    # Step 1: System Preparation
    log::header "📦 System Preparation"
    
    if ! flow::is_yes "$is_ci"; then
        # Make scripts executable
        log::info "Setting script permissions..."
        permissions::make_scripts_executable
        
        # Fix system clock if needed
        log::info "Checking system clock..."
        clock::fix
        
        # Run network diagnostics
        log::info "Running network diagnostics..."
        network_diagnostics::run || {
            log::info "Some network tests failed, but continuing with setup..."
        }
        
        # Setup firewall early for network connectivity
        if flow::can_run_sudo "firewall setup" 2>/dev/null; then
            log::info "Setting up firewall..."
            firewall::setup "$environment"
        else
            log::warning "Firewall setup requires sudo privileges - skipping"
        fi
        
        # Update system packages
        if phase::is_monorepo; then
            log::info "Updating system packages..."
            system::update_and_upgrade
        fi
    fi
    
    # Step 2: Install Common Dependencies
    log::header "🔧 Installing Dependencies"
    common_deps::check_and_install
    
    # Step 3: Initialize Configuration
    log::info "Initializing configuration..."
    config::init
    
    # Step 4: Configure Git (if in repo)
    if command -v git &> /dev/null && [[ -d "${var_ROOT_DIR}/.git" ]]; then
        log::info "Configuring git..."
        (cd "${var_ROOT_DIR}" && git config core.filemode false)
        log::success "✅ Git configured to ignore permission changes"
    fi
    
    # Step 5: Install Test Tools (for development/CI)
    if flow::is_yes "$is_ci" || [[ "$environment" == "development" ]]; then
        log::info "Installing test tools..."
        bats::install
        shellcheck::install
    fi
    
    # Step 6: Docker Setup
    log::header "🐳 Docker Setup"
    if ! docker::setup; then
        log::error "Docker setup failed"
        log::info "Running Docker diagnostics..."
        docker::diagnose || true
        
        # Provide helpful error message
        if grep -qi microsoft /proc/version 2>/dev/null || [[ -f /proc/sys/fs/binfmt_misc/WSLInterop ]]; then
            log::error ""
            log::error "Docker setup failed in WSL environment!"
            log::error "Please ensure Docker Desktop is running on Windows and WSL integration is enabled."
        else
            log::error ""
            log::error "Docker setup failed!"
            log::error "Run './diagnose-docker.sh' for detailed diagnostics."
        fi
        return "${ERROR_DOCKER_SETUP_FAILED:-1}"
    fi
    
    # Step 7: Target-Specific Setup (handled by service.json)
    # The actual target-specific setup is defined in service.json
    # This step is kept for logging purposes only
    log::info "Target-specific setup will be handled by service.json configuration"
    
    # Step 8: Install Resources (if configured)
    if [[ "$resources" != "none" ]]; then
        local resource_script="${var_SCRIPTS_RESOURCES_DIR}/index.sh"
        if [[ -f "$resource_script" ]]; then
            log::header "📦 Installing Resources"
            log::info "Resource mode: $resources"
            
            # The resource script handles its own logic based on the mode
            export RESOURCES="$resources"
            if bash "$resource_script" install; then
                log::success "✅ Resources installed"
            else
                log::warning "⚠️  Some resources failed to install"
            fi
        else
            log::debug "Resource management not available"
        fi
    fi
    
    # Step 9: Run post-setup hook
    phase::run_hook "postSetup"
    
    # Complete phase
    phase::complete
    
    # Export key variables for next phases
    export SETUP_COMPLETE="true"
    export SETUP_TARGET="$TARGET"
    export SETUP_ENVIRONMENT="$environment"
    
    return 0
}

#######################################
# Entry point for direct execution
#######################################
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Check if being called by lifecycle engine
    if [[ "${LIFECYCLE_PHASE:-}" == "setup" ]]; then
        setup::universal_main "$@"
    else
        log::error "This script should be called through the lifecycle engine"
        log::info "Use: ./scripts/manage.sh setup [options]"
        exit 1
    fi
fi