#!/usr/bin/env bash
################################################################################
# Generic Setup Phase Handler
# 
# Handles ONLY generic setup tasks applicable to ANY application:
# - System preparation (permissions, clock sync)
# - Core dependency installation (git, jq, bash essentials)
# - Docker/container setup and verification
# - Network diagnostics and firewall
# - Development tools (bats, shellcheck)
#
# App-specific setup (resources, packages, etc.) should be defined
# in each app's .vrooli/service.json lifecycle.setup.steps
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Minimal phase functions (previously in common.sh)
phase::init() {
    local phase_name="${1:-unknown}"
    export PHASE_NAME="$phase_name"
    PHASE_START_TIME=$(date +%s)
    export PHASE_START_TIME
    
    log::header "🚀 Starting $PHASE_NAME phase"
    log::debug "Phase handler: ${BASH_SOURCE[1]:-unknown}"
    log::debug "Project root: ${var_ROOT_DIR:-not set}"
}

phase::complete() {
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - ${PHASE_START_TIME:-0}))
    
    log::success "✅ $PHASE_NAME phase completed in ${duration}s"
}
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/target_matcher.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/permissions.sh"  # Consolidated permissions
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"  # Sudo utilities
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/clock.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/kernel_config.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/diagnostics/network_diagnostics.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/common_deps.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/config.sh"  # Consolidated config
# shellcheck disable=SC1091
source "${var_LIB_DIR}/runtimes/docker.sh"
# shellcheck disable=SC1091
source "${var_LIB_DIR}/runtimes/nodejs.sh"
# shellcheck disable=SC1091
source "${var_LIB_DIR}/runtimes/go.sh"
# shellcheck disable=SC1091
source "${var_LIB_DIR}/runtimes/python.sh"
# shellcheck disable=SC1091
source "${var_LIB_DIR}/runtimes/helm.sh"
# shellcheck disable=SC1091
source "${var_LIB_DIR}/runtimes/sqlite.sh"
# shellcheck disable=SC1091
source "${var_LIB_TOOLS_DIR}/buf.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/firewall.sh"
# shellcheck disable=SC1091
source "${var_LIB_DEPS_DIR}/bats.sh"
# shellcheck disable=SC1091
source "${var_LIB_DEPS_DIR}/shellcheck.sh"
# shellcheck disable=SC1091
source "${var_LIB_DEPS_DIR}/lychee.sh"
# shellcheck disable=SC1091
source "${var_LIB_DEPS_DIR}/ast-grep.sh"
# shellcheck disable=SC1091
source "${var_LIB_DEPS_DIR}/js-yaml.sh"
# shellcheck disable=SC1091
source "${var_LIB_DEPS_DIR}/ajv.sh"

################################################################################
# Main Setup Logic
################################################################################

#######################################
# Run generic setup tasks
# Handles ONLY generic setup operations
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
setup::generic_main() {
    # Initialize phase
    phase::init "Setup"
    
    # Get parameters from environment or defaults
    local target="${TARGET:-native-linux}"
    local clean="${CLEAN:-no}"
    local is_ci="${IS_CI:-no}"
    local sudo_mode="${SUDO_MODE:-ask}"
    local environment="${ENVIRONMENT:-development}"
    local resources="${RESOURCES:-enabled}"
    
    log::info "Generic setup starting..."
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
    
    # Set SUDO_MODE based on context if not explicitly set
    if [[ -z "${SUDO_MODE_EXPLICIT:-}" ]]; then
        # If running as root or with sudo, default to error mode
        if sudo::is_root || sudo::is_running_as_sudo; then
            log::debug "Running with sudo privileges - setting SUDO_MODE=error"
            export SUDO_MODE="error"
        else
            log::debug "Not running with sudo - setting SUDO_MODE=skip"
            export SUDO_MODE="skip"
        fi
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
        
        # Run network diagnostics with fast mode support
        if [[ "${FAST_MODE:-false}" == "true" ]]; then
            log::info "Fast mode: Running basic connectivity check..."
            # Just do basic connectivity check for fast mode
            if ping -c 1 -W 2 8.8.8.8 >/dev/null 2>&1; then
                log::success "✅ Basic connectivity verified"
            else
                log::warning "No internet connectivity detected, proceeding anyway"
            fi
        else
            log::info "Running comprehensive network diagnostics..."
            if ! network_diagnostics::run; then
                log::warning "Network diagnostic tests failed - attempting automatic fixes..."
                
                # Try to fix common network issues
                network_diagnostics_fixes::fix_ipv6_issues || true
                network_diagnostics_fixes::fix_ipv4_only_issues || true
                network_diagnostics_fixes::fix_dns_issues || true
                network_diagnostics_fixes::fix_ufw_blocking || true
                
                # Re-run diagnostics to verify fixes
                log::info "Re-running diagnostics to verify fixes..."
                if network_diagnostics::run; then
                    log::success "✅ Network issues resolved by automatic fixes"
                else
                    log::warning "Some network issues remain - continuing with setup..."
                fi
            else
                log::success "✅ All network diagnostics passed"
            fi
        fi
        
        # Setup firewall early for network connectivity  
        if [[ "${FAST_MODE:-false}" == "true" ]]; then
            log::info "Fast mode: Skipping firewall setup"
        elif flow::can_run_sudo "firewall setup" 2>/dev/null; then
            log::info "Setting up firewall..."
            firewall::setup "$environment"
        else
            log::warning "Firewall setup requires sudo privileges - skipping"
        fi
        
        # Skip system package updates - let apps decide if needed
        log::debug "System package updates skipped (handle in app-specific setup if needed)"
    fi

    # Kernel parameter tuning (inotify limits, resource requirements)
    kernel_config::configure_for_resources || log::warning "Kernel parameter configuration skipped or failed"
    
    # Step 2: Install Common Dependencies
    log::header "🔧 Installing Dependencies"
    common_deps::check_and_install
    
    # Step 2a: Orchestrator removed - using process manager instead
    log::info "✅ Process manager ready"
    
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
        lychee::install
        ast_grep::install
    fi
    
    # Step 6: Install Common Runtimes
    log::header "🚀 Installing Common Runtimes"
    
    # Docker (commonly needed)
    log::info "Installing Docker..."
    if ! docker::ensure_installed; then
        log::error "Docker installation failed"
        log::info "Running Docker diagnostics..."
        docker::diagnose || true
        return "${ERROR_DOCKER_SETUP_FAILED:-1}"
    fi
    
    # Node.js (commonly needed)
    log::info "Installing Node.js..."
    nodejs::ensure_installed || log::warning "Node.js installation failed (not critical)"

    if system::is_command node; then
        js_yaml::install || log::warning "js-yaml CLI installation skipped"
        ajv_cli::install || log::warning "ajv CLI installation skipped"
    else
        log::warning "Skipping js-yaml/ajv CLI install because Node.js is unavailable"
    fi
    
    # Python (commonly needed - with venv support)
    log::info "Installing Python..."
    python::ensure_installed || log::warning "Python installation failed (not critical)"
    
    # Go (commonly needed)
    log::info "Installing Go..."
    # Install Go dev tools in development environment
    if [[ "$environment" == "development" ]]; then
        export GO_INSTALL_DEV_TOOLS=true
    fi
    go::ensure_installed || log::warning "Go installation failed (not critical)"

    # Buf (protobuf CLI for contracts/codegen)
    log::info "Installing Buf (proto CLI)..."
    buf::ensure_installed || log::warning "Buf installation failed (not critical)"
    
    # Helm (for Kubernetes deployments)
    log::info "Installing Helm..."
    helm::ensure_installed || log::warning "Helm installation failed (not critical)"
    
    # SQLite (for database operations)
    log::info "Installing SQLite..."
    sqlite::ensure_installed || log::warning "SQLite installation failed (not critical)"
    
    # Cloudflare tunnel support
    log::info "Setting up Cloudflare tunnel support..."
    if [[ -f "${var_LIB_SERVICE_DIR}/cloudflare-tunnel.sh" ]]; then
        # shellcheck disable=SC1091
        source "${var_LIB_SERVICE_DIR}/cloudflare-tunnel.sh"
        main || log::warning "Cloudflare tunnel setup failed (not critical)"
    else
        log::debug "Cloudflare tunnel setup script not found - skipping"
    fi
    
    # Step 7: Install Enabled Resources
    log::header "🔨 Installing Enabled Resources"
    # shellcheck disable=SC1091
    source "${var_LIB_DIR}/resources/resource-orchestrator.sh"
    if [[ -f "${var_SERVICE_JSON_FILE}" ]]; then
        resource_auto::install_enabled "${var_SERVICE_JSON_FILE}" || {
            log::warning "Some resources failed to install - continuing with setup"
        }
    else
        log::debug "No service.json found - skipping resource installation"
    fi
    
    
    # Step 8: Complete generic setup
    log::info "Generic setup tasks completed"
    log::info "App-specific setup will be handled by service.json configuration"
    
    # Step 9: Complete phase
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
        setup::generic_main "$@"
    else
        log::error "This script should be called through the lifecycle engine"
        log::info "Use: ./scripts/manage.sh setup [options]"
        exit 1
    fi
fi
