#!/usr/bin/env bash
# Deploys specified build artifacts to the specified destinations.
# This script is meant to be run on the production server
set -euo pipefail
DESCRIPTION="Deploys a specific Vrooli service artifact to the target environment."

MAIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ——— Default values ——— #
export ENV_FILE=""

# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/args.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/flow.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/log.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/var.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/version.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/zip.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/deploy/docker.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/deploy/k8s.sh"

# Default values set in parse_arguments
TARGET=""
SOURCE_TYPE=""
LOCATION=""
DETACHED=""
VERSION=""

# --- Argument Parsing ---

usage() {
    args::usage "$DESCRIPTION"
    exit_codes::print
}

deploy::parse_arguments() {
    args::reset

    args::register_help
    args::register_sudo_mode
    args::register_yes
    args::register_environment

    args::register \
        --name "source" \
        --flag "s" \
        --desc "The type of artifact/service to deploy." \
        --required "true" \
        --type "value" \
        --options "docker|k8s|windows|android" # Add other valid types as needed

    args::register \
        --name "detached" \
        --flag "x" \
        --desc "Skip teardown of reverse proxy on script exit (default: no)." \
        --type "value" \
        --options "yes|no" \
        --default "yes"

    args::register \
        --name "version" \
        --flag "v" \
        --desc "The version of the project artifacts to deploy (defaults to version in ../../package.json)." \
        --type "value" \
        --default ""

    if args::is_asking_for_help "$@"; then
        usage
        exit "$EXIT_SUCCESS"
    fi

    args::parse "$@"

    export SUDO_MODE=$(args::get "sudo-mode")
    export YES=$(args::get "yes")
    export SOURCE_TYPE=$(args::get "source")
    export LOCATION="remote"
    export DETACHED=$(args::get "detached")
    export VERSION=$(args::get "version")
    export ENVIRONMENT=$(args::get "environment")

    # Set default version if not provided
    if [ -z "$VERSION" ]; then
        VERSION=$(version::get_project_version)
        if [ -z "$VERSION" ]; then
          log::error "Could not determine project version from ../../package.json. Please specify with -v."
          exit "$ERROR_CONFIGURATION"
        fi
        log::info "Using project version from package.json: $VERSION"
    fi
}

# --- Main Deployment Logic ---

deploy::main() {
    deploy::parse_arguments "$@"

    log::header "🚀 Starting deployment of '$SOURCE_TYPE' in environment '$ENVIRONMENT' to version $VERSION locally..."

    source "${MAIN_DIR}/setup.sh" "$@"

    log::header "🎁 Loading build artifacts..."
    local build_dir="${var_DEST_DIR}/${VERSION}"
    # Where to put build artifacts
    local artifacts_dir="${build_dir}/artifacts"
    # Where to put bundles
    local bundles_dir="${build_dir}/bundles"

    mkdir -p "$artifacts_dir"
    zip::unzip_artifacts "${bundles_dir}/artifacts.zip.gz" "$artifacts_dir"
    zip::load_artifacts "$artifacts_dir" "$var_DEST_DIR"

    if env::is_location_remote; then
        proxy::setup
        if ! flow::is_yes "$DETACHED"; then
            trap 'info "Tearing down Caddy reverse proxy..."; stop_reverse_proxy' EXIT INT TERM
        fi
    fi

    # Execute deployment based on the single source type
    log::info "Deploying $SOURCE_TYPE (Version: $VERSION)..."
    case "$SOURCE_TYPE" in
        docker)
            deploy::deploy_docker "$artifacts_dir"
            ;;
        k8s)
            deploy::deploy_k8s "$ENVIRONMENT"
            ;;
        windows)
            log::info "Deploying Windows binary (stub) from $artifacts_dir"
            # Add actual deployment logic here if needed
            ;;
        android)
            log::info "Deploying Android package (stub) from $artifacts_dir"
            # Add actual deployment logic here if needed
            ;;
        *)
            # This case should not be reached due to args::register options
            log::error "Unknown source type: $SOURCE_TYPE";
            exit "$ERROR_USAGE"
            ;;
    esac

    log::success "✅ Deployment completed for $SOURCE_TYPE"
}

deploy::main "$@" 