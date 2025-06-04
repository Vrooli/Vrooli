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
}

setup::main() {
    setup::parse_arguments "$@"
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
        internet::check_connection
        system::update_and_upgrade
    fi

    # Setup tools
    common_deps::check_and_install
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
        firewall::setup
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

    docker::setup

    # Run the target-specific setup script
    TARGET_SCRIPT="${MAIN_DIR}/../helpers/setup/target/${TARGET//-/_}.sh"
    if [[ ! -f "$TARGET_SCRIPT" ]]; then
        log::error "Target setup script not found: $TARGET_SCRIPT"
        exit "${ERROR_FUNCTION_NOT_FOUND}"
    fi
    bash "$TARGET_SCRIPT" "$@" || exit $?

    log::success "✅ Setup complete. You can now run 'pnpm run develop' or 'bash scripts/main/develop.sh'"

    # Schedule backups if production environment file exists
    if ! flow::is_yes "$IS_CI" && env::prod_file_exists; then
        "${MAIN_DIR}/backup.sh"
    fi
}

setup::main "$@"