#!/usr/bin/env bash
set -euo pipefail
DESCRIPTION="Prepares the project for development or production."

MAIN_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/args.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/ci.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/docker.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/domainCheck.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/env.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/flow.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/jwt.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/log.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/ports.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/proxy.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/targetMatcher.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/var.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/repository.sh"
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
    
    # CI environments should not install resources by default
    if flow::is_yes "$IS_CI" && [[ "$RESOURCES" == "enabled" ]]; then
        log::info "CI environment detected, changing resources default from 'enabled' to 'none'"
        export RESOURCES="none"
    fi
}

setup::main() {
    # Validate target and canonicalize
    if ! canonical=$(match_target "$TARGET"); then
        args::usage "$DESCRIPTION"
        exit "$ERROR_USAGE"
    fi
    export TARGET="$canonical"
    
    log::header "🔨 Starting project setup for $(match_target "$TARGET")..."

    # Prepare the system
    if ! flow::is_yes "$IS_CI"; then
        permissions::make_scripts_executable
        clock::fix
        network_diagnostics::run
        
        # Setup firewall EARLY to ensure outbound HTTPS works for subsequent operations
        # This prevents issues with GitHub clones, package downloads, etc.
        log::info "Setting up firewall early to ensure network connectivity..."
        firewall::setup "${ENVIRONMENT:-development}"
        
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

# Only parse arguments if this script is being run directly, not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    setup::parse_arguments "$@"
    setup::main "$@"
else
    # If sourced, set default values for variables that would normally be parsed
    export CLEAN="${CLEAN:-no}"
    export IS_CI="${IS_CI:-no}"
    export SUDO_MODE="${SUDO_MODE:-error}"
    export YES="${YES:-no}"
    export LOCATION="${LOCATION:-}"
    export ENVIRONMENT="${ENVIRONMENT:-development}"
fi