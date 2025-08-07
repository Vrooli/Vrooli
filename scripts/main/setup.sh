#!/usr/bin/env bash
set -euo pipefail
DESCRIPTION="Prepares the project for development or production."

MAIN_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source core utilities that both monorepo and standalone need
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/log.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/var.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/args.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/flow.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/targetMatcher.sh"

# Conditionally source monorepo-specific utilities
if [[ "${VROOLI_CONTEXT:-monorepo}" == "monorepo" ]]; then
    # shellcheck disable=SC1091
    source "${MAIN_DIR}/../helpers/utils/ci.sh"
    # shellcheck disable=SC1091
    source "${MAIN_DIR}/../helpers/utils/docker.sh"
    # shellcheck disable=SC1091
    source "${MAIN_DIR}/../helpers/utils/domainCheck.sh"
    # shellcheck disable=SC1091
    source "${MAIN_DIR}/../helpers/utils/env.sh"
    # shellcheck disable=SC1091
    source "${MAIN_DIR}/../helpers/utils/jwt.sh"
    # shellcheck disable=SC1091
    source "${MAIN_DIR}/../helpers/utils/ports.sh"
    # shellcheck disable=SC1091
    source "${MAIN_DIR}/../helpers/utils/proxy.sh"
    # shellcheck disable=SC1091
    source "${MAIN_DIR}/../helpers/utils/repository.sh"
fi

# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/setup/index.sh"

setup::parse_arguments() {
    args::reset

    args::register_help
    args::register_sudo_mode
    args::register_yes
    args::register_location
    args::register_environment
    args::register_target

    args::register \
        --name "clean" \
        --flag "c" \
        --desc "Remove previous artefacts (volumes, ~/.pnpm-store, etc.)" \
        --type "value" \
        --options "yes|no" \
        --default "no"

    args::register \
        --name "ci-cd" \
        --flag "d" \
        --desc "True if running in CI/CD (via GitHub Actions)" \
        --type "value" \
        --options "yes|no" \
        --default "no"

    args::register \
        --name "resources" \
        --flag "r" \
        --desc "Local resources to install (comma-separated, or 'all', 'ai-only', 'enabled', 'none')" \
        --type "value" \
        --default "enabled"

    args::register \
        --name "fix-remote" \
        --flag "f" \
        --desc "Quick fix for remote desktop issues (restarts GDM3, ensures memory protection)" \
        --type "value" \
        --options "yes|no" \
        --default "no"

    if args::is_asking_for_help "$@"; then
        args::usage "$DESCRIPTION"
        exit_codes::print
        exit 0
    fi

    # Capture positional arguments to prevent stray output
    local -a POSITIONAL_ARGS
    POSITIONAL_ARGS=($(args::parse "$@"))

    export TARGET=$(args::get "target")
    export CLEAN=$(args::get "clean")
    export IS_CI=$(args::get "ci-cd")
    export SUDO_MODE=$(args::get "sudo-mode")
    export YES=$(args::get "yes")
    export LOCATION=$(args::get "location")
    export ENVIRONMENT=$(args::get "environment")
    export RESOURCES=$(args::get "resources")
    export FIX_REMOTE=$(args::get "fix-remote")
    
    # CI environments should not install resources by default
    if flow::is_yes "$IS_CI" && [[ "$RESOURCES" == "enabled" ]]; then
        log::info "CI environment detected, changing resources default from 'enabled' to 'none'"
        export RESOURCES="none"
    fi
}

setup::monorepo() {
    # Validate target and canonicalize
    if ! canonical=$(match_target "$TARGET"); then
        args::usage "$DESCRIPTION"
        exit "$ERROR_USAGE"
    fi
    export TARGET="$canonical"
    
    # In standalone mode, default to skip sudo unless explicitly set
    if [[ "${VROOLI_CONTEXT:-}" == "standalone" ]] && [[ -z "${SUDO_MODE_EXPLICIT:-}" ]]; then
        log::info "Standalone app detected, defaulting to SUDO_MODE=skip"
        export SUDO_MODE="skip"
    fi
    
    # Handle quick remote desktop fix if requested
    if flow::is_yes "$FIX_REMOTE"; then
        log::header "🔧 Quick fix for remote desktop issues..."
        
        # Only works for native Linux
        if [[ "$TARGET" != "native-linux" ]]; then
            log::error "Remote desktop fix is only available for native-linux target"
            exit "$ERROR_USAGE"
        fi
        
        local remote_session_script="${MAIN_DIR}/../helpers/setup/remote_session_protect.sh"
        if [[ -f "$remote_session_script" ]]; then
            bash "$remote_session_script" configure
            log::success "✅ Remote desktop fix complete"
            log::info "If you're still having issues, try: sudo systemctl restart gdm3"
            exit 0
        else
            log::error "Remote session protection script not found"
            exit "$ERROR_FUNCTION_NOT_FOUND"
        fi
    fi
    
    log::header "🔨 Starting project setup for $(match_target "$TARGET")..."

    # Prepare the system
    if ! flow::is_yes "$IS_CI"; then
        permissions::make_scripts_executable
        clock::fix
        # Run network diagnostics but don't fail if some tests fail (e.g., local services not yet running)
        network_diagnostics::run || {
            log::info "Some network tests failed, but continuing with setup..."
        }
        
        # Setup firewall EARLY to ensure outbound HTTPS works for subsequent operations
        # This prevents issues with GitHub clones, package downloads, etc.
        # But only if sudo is available - otherwise skip it
        if flow::can_run_sudo "firewall setup" 2>/dev/null; then
            log::info "Setting up firewall early to ensure network connectivity..."
            firewall::setup "${ENVIRONMENT:-development}"
        else
            log::warning "Firewall setup requires sudo privileges - skipping"
            log::info "Note: You may need to manually configure firewall rules for full functionality"
        fi
        
        system::update_and_upgrade
    fi

    # Setup tools
    common_deps::check_and_install
    
    # Initialize .vrooli configuration files
    config::init
    
    # Configure git to ignore file permission changes
    # This prevents shell scripts from showing as modified when only permissions change
    if command -v git &> /dev/null && [ -d ".git" ]; then
        log::info "Configuring git to ignore file permission changes..."
        git config core.filemode false
        log::success "✅ Git configured to ignore permission changes"
    fi
    
    # Clean up volumes & caches
    if flow::is_yes "$CLEAN"; then
        clean::main
    fi

    if ! flow::is_yes "$IS_CI"; then
        jwt::generate_key_pair
    fi

    env::load_secrets
    check_location_if_not_set
    env::construct_derived_secrets

    if ! flow::is_yes "$IS_CI" && env::is_location_remote; then
        system::purge_apt_update_notifier

        ports::check_and_free "${PORT_DB:-5432}"
        ports::check_and_free "${PORT_JOBS:-4001}"
        ports::check_and_free "${PORT_REDIS:-6379}"
        ports::check_and_free "${PORT_SERVER:-5329}"
        ports::check_and_free "${PORT_UI:-3000}"

        proxy::setup
        # firewall::setup is now called earlier in the setup process
    fi

    if ! flow::is_yes "$IS_CI" && env::is_location_local; then
        ci::generate_key_pair
    fi

    # Setup Stripe CLI testing for local development environments
    if env::in_development && ! flow::is_yes "$IS_CI"; then
        # Check if any common Stripe keys are set
        if [[ -n "${STRIPE_API_KEY-}" ]] || \
           [[ -n "${STRIPE_SECRET_KEY-}" ]] || \
           [[ -n "${STRIPE_PUBLISHABLE_KEY-}" ]] || \
           [[ -n "${STRIPE_WEBHOOK_SECRET-}" ]]; then
            log::info "Stripe environment variables detected. Attempting Stripe CLI setup..."
            # Call the setup function from the sourced script
            if stripe_cli::setup; then
                log::info "Stripe CLI setup function completed successfully."
            else
                log::error "Stripe CLI setup function reported an issue. Check logs above."
            fi
        else
            log::info "No Stripe environment variables (STRIPE_API_KEY, STRIPE_SECRET_KEY, STRIPE_PUBLISHABLE_KEY, STRIPE_WEBHOOK_SECRET) found."
            log::info "Skipping Stripe CLI setup. To enable, define at least one of these in your .env file."
        fi
    fi

    # Both CI and development environments can run tests
    if flow::is_yes "$IS_CI" || env::in_development; then
        bats::install
        shellcheck::install
    fi

    # Setup Docker with better error handling
    if ! docker::setup; then
        log::error "Docker setup failed"
        log::info "Running Docker diagnostics..."
        docker::diagnose || true
        
        # Provide helpful error message based on environment
        if grep -qi microsoft /proc/version 2>/dev/null || [[ -f /proc/sys/fs/binfmt_misc/WSLInterop ]]; then
            log::error ""
            log::error "Docker setup failed in WSL environment!"
            log::error "Please ensure Docker Desktop is running on Windows and WSL integration is enabled."
            log::error ""
            log::error "You can run './diagnose-docker.sh' for detailed diagnostics."
        else
            log::error ""
            log::error "Docker setup failed!"
            log::error "You can run './diagnose-docker.sh' for detailed diagnostics."
        fi
        exit "${ERROR_DOCKER_SETUP_FAILED}"
    fi

    # Run the target-specific setup script
    TARGET_SCRIPT="${MAIN_DIR}/../helpers/setup/target/${TARGET//-/_}.sh"
    if [[ ! -f "$TARGET_SCRIPT" ]]; then
        log::error "Target setup script not found: $TARGET_SCRIPT"
        exit "${ERROR_FUNCTION_NOT_FOUND}"
    fi
    bash "$TARGET_SCRIPT" "$@" || exit $?

    # Initialize Vrooli platform using injection engine
    setup::initialize_platform

    log::success "✅ Setup complete. You can now run 'pnpm run develop' or 'bash scripts/main/develop.sh'"

    # Execute repository postInstall hook if configured
    if repository::run_hook "postInstall" 2>/dev/null; then
        log::success "✅ Repository postInstall hook completed"
    else
        log::debug "No postInstall hook configured or execution failed"
    fi

    # Schedule backups only if in production environment (not just if prod file exists)
    if ! flow::is_yes "$IS_CI" && env::in_production; then
        "${MAIN_DIR}/backup.sh"
        log::success "🎉 Setup and backup completed successfully!"
    else
        log::success "🎉 Setup completed successfully!"
    fi
}

#######################################
# Initialize Vrooli platform using injection engine
# Executes platform initialization if service.json exists
#######################################
setup::initialize_platform() {
    local service_config="${PROJECT_ROOT:-$(pwd)}/.vrooli/service.json"
    local init_dir="${PROJECT_ROOT:-$(pwd)}/initialization"
    local injection_engine="${PROJECT_ROOT:-$(pwd)}/scripts/scenarios/injection/engine.sh"
    
    log::header "🚀 Initializing Vrooli Platform"
    
    # Check if we have the unified initialization structure
    if [[ ! -f "$service_config" || ! -d "$init_dir" ]]; then
        log::info "Unified initialization not configured yet, skipping platform init"
        log::info "Missing: service.json=$(test -f "$service_config" && echo "✓" || echo "✗") initialization/=$(test -d "$init_dir" && echo "✓" || echo "✗")"
        return 0
    fi
    
    # Check if injection engine exists
    if [[ ! -x "$injection_engine" ]]; then
        log::warn "Injection engine not found or not executable: $injection_engine"
        return 0
    fi
    
    # Check if already initialized (version-aware)
    if setup::check_initialization_state; then
        log::info "Platform already initialized for this version"
        return 0
    fi
    
    # Initialize Vrooli core
    log::info "Running Vrooli core initialization..."
    if "$injection_engine" --action inject --scenario-dir "${PROJECT_ROOT:-$(pwd)}"; then
        setup::mark_initialized
        log::success "✅ Vrooli platform initialized"
    else
        log::error "❌ Platform initialization failed"
        return 1
    fi
}

#######################################
# Check if platform is already initialized for current version
#######################################
setup::check_initialization_state() {
    local state_file="$HOME/.vrooli/init-state.json"
    local service_config="${PROJECT_ROOT:-$(pwd)}/.vrooli/service.json"
    local current_version
    
    [[ -f "$service_config" ]] || return 1
    current_version=$(jq -r '.service.version' "$service_config" 2>/dev/null)
    
    if [[ -f "$state_file" ]]; then
        local init_version
        init_version=$(jq -r '.version' "$state_file" 2>/dev/null)
        [[ "$init_version" == "$current_version" ]] && return 0
    fi
    return 1
}

#######################################
# Mark platform as initialized for current version
#######################################
setup::mark_initialized() {
    local state_file="$HOME/.vrooli/init-state.json"
    local service_config="${PROJECT_ROOT:-$(pwd)}/.vrooli/service.json"
    local current_version
    current_version=$(jq -r '.service.version' "$service_config" 2>/dev/null)
    
    mkdir -p "$(dirname "$state_file")"
    jq -n --arg v "$current_version" --arg t "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        '{version: $v, timestamp: $t}' > "$state_file"
}

#######################################
# Minimal setup for standalone apps
# Handles basic initialization without monorepo dependencies
#######################################
setup::standalone() {
    log::header "🚀 Starting standalone app setup..."
    
    # Validate target and canonicalize
    if ! canonical=$(match_target "$TARGET"); then
        args::usage "$DESCRIPTION"
        exit "$ERROR_USAGE"
    fi
    export TARGET="$canonical"
    
    # Standalone apps default to skip sudo
    export SUDO_MODE="skip"
    log::info "Standalone app detected, SUDO_MODE set to skip"
    
    # Basic system preparation
    permissions::make_scripts_executable
    clock::fix
    
    # Check and install common dependencies
    common_deps::check_and_install
    
    # Initialize .vrooli configuration files
    config::init
    
    # Configure git to ignore file permission changes if in a git repo
    if command -v git &> /dev/null && [ -d ".git" ]; then
        log::info "Configuring git to ignore file permission changes..."
        git config core.filemode false
        log::success "✅ Git configured to ignore permission changes"
    fi
    
    # Run the target-specific setup script
    TARGET_SCRIPT="${MAIN_DIR}/../helpers/setup/target/${TARGET//-/_}.sh"
    if [[ ! -f "$TARGET_SCRIPT" ]]; then
        log::error "Target setup script not found: $TARGET_SCRIPT"
        exit "${ERROR_FUNCTION_NOT_FOUND}"
    fi
    bash "$TARGET_SCRIPT" "$@" || exit $?
    
    # Check if standalone app has its own initialization
    local service_json="${var_ROOT_DIR}/.vrooli/service.json"
    if [[ -f "$service_json" ]]; then
        log::info "Found service.json for standalone app"
        
        # Check for deployment.initialization phases
        local has_init
        has_init=$(jq -r '.deployment.initialization.phases // empty' "$service_json" 2>/dev/null)
        
        if [[ -n "$has_init" ]]; then
            log::info "Standalone app has initialization phases defined"
            # Future: Process initialization phases here
            # For now, this is a placeholder - the app's startup.sh will handle it
        fi
    fi
    
    log::success "✅ Standalone app setup complete"
    log::info "The app can now be started using its deployment configuration"
}

#######################################
# Main dispatcher - routes to appropriate setup function
#######################################
setup::main() {
    if [[ "${VROOLI_CONTEXT:-monorepo}" == "monorepo" ]]; then
        setup::monorepo "$@"
    else
        setup::standalone "$@"
    fi
}

# Only parse arguments if this script is being run directly, not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    setup::parse_arguments "$@"
    setup::main "$@"
else
    # If sourced, set default values for variables that would normally be parsed
    export CLEAN="${CLEAN:-no}"
    export IS_CI="${IS_CI:-no}"
    # In standalone mode, default to skip to avoid blocking on sudo
    if [[ "${VROOLI_CONTEXT:-}" == "standalone" ]]; then
        export SUDO_MODE="${SUDO_MODE:-skip}"
    else
        export SUDO_MODE="${SUDO_MODE:-error}"
    fi
    export YES="${YES:-no}"
    export LOCATION="${LOCATION:-}"
    export ENVIRONMENT="${ENVIRONMENT:-development}"
fi