#!/usr/bin/env bash
set -euo pipefail
DESCRIPTION="Builds specified artifacts for the Vrooli project, as preparation for deployment."

MAIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"

# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/args.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/docker.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/env.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/flow.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/keyless_ssh.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/log.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/system.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/var.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/version.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/zip.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/build/index.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/build/binaries/index.sh"

build::parse_arguments() {
    args::reset

    args::register_help
    args::register_sudo_mode
    args::register_yes
    args::register_location
    args::register_environment

    args::register \
        --name "bundles" \
        --flag "b" \
        --desc "Which bundle types to generate, separated by commas without spaces (default: zip)" \
        --type "value" \
        --options "all|zip|cli" \
        --default "all"
    
    args::register \
        --name "artifacts" \
        --flag "a" \
        --desc "Which container artifacts to include in the bundles, separated by commas without spaces (default: docker)" \
        --type "value" \
        --options "all|docker|k8s" \
        --default "all"

    args::register \
        --name "binaries" \
        --flag "c" \
        --desc "Which platform binaries to build, separated by commas without spaces (default: none)" \
        --type "value" \
        --options "all|windows|mac|linux|android|ios" \
        --default ""

    args::register \
        --name "dest" \
        --flag "d" \
        --desc "Where to save bundles (default: local)" \
        --type "value" \
        --options "local|remote" \
        --default "local"

    # This should ideally be yes by default. We should update this later when the tests are more stable.
    args::register \
        --name "test" \
        --flag "t" \
        --desc "Run tests before building (default: no)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    # This should ideally be yes by default. We should update this later when the files are more stable.
    args::register \
        --name "lint" \
        --flag "q" \
        --desc "Run linting before building (default: no)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "version" \
        --flag "v" \
        --desc "The version of the project (defaults to current version set in package.json)" \
        --type "value" \
        --default ""

    if args::is_asking_for_help "$@"; then
        args::usage "$DESCRIPTION"
        exit_codes::print
        exit 0
    fi

    args::parse "$@" >/dev/null
    
    export SUDO_MODE=$(args::get "sudo-mode")
    export YES=$(args::get "yes")
    export LOCATION=$(args::get "location")
    export ENVIRONMENT=$(args::get "environment")
    # Read comma-separated strings into temp variables
    local bundles_str=$(args::get "bundles")
    local artifacts_str=$(args::get "artifacts")
    local binaries_str=$(args::get "binaries")
    export DEST=$(args::get "dest")
    export TEST=$(args::get "test")
    export LINT=$(args::get "lint")
    
    local user_supplied_version_value=$(args::get "version")
    if [ -n "$user_supplied_version_value" ]; then
        export VERSION="$user_supplied_version_value"
        export VERSION_SPECIFIED_BY_USER="yes"
    else
        export VERSION=$(version::get_project_version) # Sets default if no -v flag
        export VERSION_SPECIFIED_BY_USER="no"
    fi

    # Split the strings into arrays
    IFS=',' read -r -a BUNDLES <<< "$bundles_str"
    IFS=',' read -r -a ARTIFACTS <<< "$artifacts_str"
    IFS=',' read -r -a BINARIES <<< "$binaries_str"

    # Treat explicit 'none' values (or default empty for binaries) as empty lists
    if [ "$bundles_str" = "none" ]; then
        BUNDLES=()
    fi
    if [ "$artifacts_str" = "none" ]; then
        ARTIFACTS=()
    fi
    if [ "$binaries_str" = "none" ]; then
        BINARIES=()
    fi
    # Treat explicit 'all' values as full arrays
    if [ "$bundles_str" = "all" ]; then
        BUNDLES=("zip" "cli")
    fi
    if [ "$artifacts_str" = "all" ]; then
        ARTIFACTS=("docker" "k8s")
    fi
    if [ "$binaries_str" = "all" ]; then
        BINARIES=("windows" "mac" "linux" "android" "ios")
    fi
    # Treat missing values as their default values
    if [ -z "$bundles_str" ]; then
        BUNDLES=("zip")
    fi
    if [ -z "$artifacts_str" ]; then
        ARTIFACTS=("docker")
    fi
    if [ -z "$binaries_str" ]; then
        BINARIES=()
    fi

    # Handle default destination, test, lint
    if [ -z "$DEST" ]; then
        DEST="local"
    fi
    if [ -z "$TEST" ]; then
        TEST="no"
    fi
    if [ -z "$LINT" ]; then
        LINT="no"
    fi
}

build::main() {
    build::parse_arguments "$@"
    log::header "🔨 Starting build for ${ENVIRONMENT} environment..."

    # Mandate --version for production builds and warn/confirm if same as current
    if env::in_production; then # Checks if $ENVIRONMENT is "prod" or "production"
        if [ "${VERSION_SPECIFIED_BY_USER}" != "yes" ]; then
            log::error "ERROR: For production builds (environment: ${ENVIRONMENT}), the --version <version> flag is mandatory."
            log::error "Please specify the exact version to build and deploy."
            exit "${exit_codes_ERROR_USAGE}"
        else
            # Version was specified for a production build, check if it's the same as current
            local current_project_version
            current_project_version=$(version::get_project_version)
            if [ "$VERSION" == "$current_project_version" ]; then
                log::warning "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
                log::warning "!! WARNING: THE SUPPLIED VERSION ($VERSION) IS THE SAME AS THE CURRENT PROJECT VERSION  !!"
                log::warning "!!          IN PACKAGE.JSON. PROCEEDING WILL OVERWRITE ANY EXISTING REMOTE          !!"
                log::warning "!!          ARTIFACTS AND DOCKER IMAGES PUBLISHED FOR THIS VERSION.                 !!"
                log::warning "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
                
                local confirm_overwrite
                log::prompt "Are you sure you want to proceed with building version $VERSION? (y/N): " confirm_overwrite
                if ! flow::is_yes "${confirm_overwrite:-no}"; then # Default to 'no' if user just presses Enter
                    log::info "Build aborted by user."
                    exit "${exit_codes_SUCCESS}"
                fi
                log::info "User confirmed proceeding with version $VERSION."
            fi
        fi
    fi

    source "${MAIN_DIR}/setup.sh" "$@"

    log::info "Cleaning previous build artifacts..."
    clean_build

    # Need to build packages first for tests to run correctly
    build_packages
    verify_build

    # Create Brave Rewards verification file after packages (specifically UI) are built
    if ! brave_rewards::create_verification_file; then
        log::error "Failed to create Brave Rewards verification file."
    fi

    # Create Twilio domain verification file
    if ! twilio_verification::create_verification_file; then
        log::error "Failed to create Twilio domain verification file. Review warnings above."
    fi

    if flow::is_yes "$TEST"; then
        log::header "Running tests..."
        # Run tests without rebuilding packages
        pnpm run test:shell
        pnpm run test:unit
        pnpm run test:run
    fi
    if flow::is_yes "$LINT"; then
        log::header "Running linting..."
        pnpm run lint
    fi

    if [ "${VERSION_SPECIFIED_BY_USER}" = "yes" ]; then
        log::info "User specified version '${VERSION}'. Updating project version..."
        # Note: Updates package.jsons and helm values files with the new version
        version::set_project_version "$VERSION"
    else
        log::info "Using project version '${VERSION}' from package.json. No explicit version update will be performed by this script."
    fi

    # Determine if any binary/desktop builds are requested
    local build_desktop="NO"
    # Check array length correctly
    if [ ${#BINARIES[@]} -gt 0 ]; then
      build_desktop="YES"
    fi

    # Build Electron main/preload scripts if building desktop app
    if flow::is_yes "$build_desktop"; then
      log::header "Building Electron scripts..."
      # Ensure a tsconfig for electron exists
      # Adjust the output dir (-outDir) if needed
      npx tsc --project platforms/desktop/tsconfig.json --outDir dist/desktop || {
        log::error "Failed to build Electron scripts. Ensure platforms/desktop/tsconfig.json is configured."; exit "$ERROR_BUILD_FAILED";
      }
      # Rename output to .cjs to explicitly mark as CommonJS
      mv dist/desktop/main.js dist/desktop/main.cjs || { log::error "Failed to rename main.js to main.cjs"; exit "$ERROR_BUILD_FAILED"; }
      mv dist/desktop/preload.js dist/desktop/preload.cjs || { log::error "Failed to rename preload.js to preload.cjs"; exit "$ERROR_BUILD_FAILED"; }
      log::success "Electron scripts built and renamed to .cjs."
    fi

    log::header "🎁 Preparing build artifacts..."
    local build_dir="${var_DEST_DIR}/${VERSION}"
    # Where to put build artifacts
    local artifacts_dir="${build_dir}/artifacts"
    # Where to put bundles
    local bundles_dir="${build_dir}/bundles"

    # Collect artifacts based on bundle types
    for b in "${BUNDLES[@]}"; do
        case "$b" in
            zip)
                log::info "Building ZIP bundle..."
                zip::copy_project "${artifacts_dir}"
                ;;
            cli)
                log::info "Building CLI executables..."
                package_cli
                ;;
            *)
                log::warning "Unknown bundle type: $b"
                ;;
        esac
    done

    # Collect Docker/Kubernetes artifacts
    for a in "${ARTIFACTS[@]}"; do
        case "$a" in
            docker)
                docker::build_artifacts
                docker::save_images "$artifacts_dir"
                ;;
            k8s)
                log::info "Building Kubernetes artifacts..."
                # Note: Docker commands here are no accident. We need these images stored somewhere so
                # that the k8s deployment can pull them. We're choosing to store them in Docker Hub.
                docker::build_artifacts

                export PROJECT_VERSION="$VERSION" # Ensure $VERSION is used for Docker image tagging
                docker::login_to_dockerhub
                docker::tag_and_push_images

                # --- START NEW K8S BUILD LOGIC ---
                log::info "Packaging Helm chart..."
                local chart_source_path="${var_ROOT_DIR}/k8s/chart"
                local chart_destination_path="${artifacts_dir}/k8s-chart-packages" # Store .tgz in a sub-directory

                if [ ! -d "$chart_source_path" ]; then
                    log::error "Helm chart source directory not found: $chart_source_path"
                    exit "$ERROR_BUILD_FAILED"
                fi
                mkdir -p "$chart_destination_path"

                # Ensure HELM_VERSION is available, ideally it's the same as PROJECT_VERSION
                # The `version` variable in build.sh (set as PROJECT_VERSION) should be used.
                if [ -z "$VERSION" ]; then
                    log::error "Project version (VERSION) is not set. Cannot package Helm chart."
                    exit "$ERROR_BUILD_FAILED"
                fi

                # Get chart name from Chart.yaml to predict the .tgz filename
                # This requires yq or similar, or a simpler grep if Chart.yaml is simple
                local chart_name
                chart_name=$(grep '^name:' "${chart_source_path}/Chart.yaml" | awk '{print $2}')
                if [ -z "$chart_name" ]; then
                    log::error "Could not determine chart name from ${chart_source_path}/Chart.yaml."
                    log::error "Please ensure Chart.yaml contains a valid 'name:' field."
                    exit "$ERROR_BUILD_FAILED"
                fi

                if helm package "$chart_source_path" --version "$VERSION" --app-version "$VERSION" --destination "$chart_destination_path"; then
                    log::success "Helm chart packaged successfully to $chart_destination_path/${chart_name}-${VERSION}.tgz"
                else
                    log::error "Helm chart packaging failed."
                    exit "$ERROR_BUILD_FAILED"
                fi

                log::info "Copying Helm environment-specific values files..."
                local helm_values_source_dir="${var_ROOT_DIR}/k8s/chart"
                local helm_values_dest_dir="${artifacts_dir}/helm-value-files"
                
                mkdir -p "$helm_values_dest_dir"
                
                # Find and copy values-*.yaml files, excluding values.yaml itself
                # Using a loop to handle cases where find might not be ideal or for more control
                local found_values_files=false
                for val_file in "${helm_values_source_dir}/values-"*.yaml; do
                    if [ -f "$val_file" ]; then # Check if the glob matched an actual file
                        if ! cp "$val_file" "$helm_values_dest_dir/"; then
                            log::error "Failed to copy Helm values file: $val_file to $helm_values_dest_dir"
                            # Decide if this is a fatal error
                        else
                            log::info "Copied $val_file to $helm_values_dest_dir"
                            found_values_files=true
                        fi
                    fi
                done

                if [ "$found_values_files" = true ]; then
                    log::success "Successfully copied Helm environment-specific values files."
                else
                    log::warning "No environment-specific Helm values files (values-*.yaml) found in $helm_values_source_dir to copy."
                fi

                # --- END NEW K8S BUILD LOGIC ---
                ;;
            *)
                log::warning "Unknown artifact type: $a";
                ;;
        esac
    done

    # Process platform binaries (e.g. Desktop App)
    for c in "${BINARIES[@]}"; do
        local target_platform=""
        case "$c" in
            windows)
                log::info "Building Windows Desktop App (Electron)..."
                # Ensure Wine is installed using the robust method
                install_wine_robustly
                target_platform="--win --x64"
                ;;
            mac)
                log::info "Building macOS Desktop App (Electron)..."
                target_platform="--mac --x64"
                ;;
            linux)
                log::info "Building Linux Desktop App (Electron)..."
                target_platform="--linux --x64"
                ;;
            # android/ios are likely mobile builds, not desktop - keeping stubs
            android)
                log::info "Building Android package..."
                bash "${MAIN_DIR}/../helpers/build/googlePlayStore.sh"
                continue # Skip electron-builder for android
                ;;
            ios)
                log::info "Building iOS package (stub)"
                # TODO: Add iOS build logic
                continue # Skip electron-builder for ios
                ;;
            *)
                log::warning "Unknown binary/desktop type: $c";
                continue
                ;;
        esac

        if [ -n "$target_platform" ]; then
            log::info "Running electron-builder for $c (this may take several minutes)..."
            # Pass platform and arch flags separately
            npx electron-builder $target_platform || {
              log::error "Electron build failed for $c."; exit "$ERROR_BUILD_FAILED";
            }
            log::success "Electron build completed for $c. Output in dist/desktop/"

            # Copying logic (optional, adjust as needed)
            if env::is_location_local "$DEST"; then
              local dest_dir="${var_DEST_DIR}/desktop/${c}/${VERSION}"
              local source_dir="${var_DEST_DIR}/desktop"
              mkdir -p "${dest_dir}"
              # Copy specific installer/package file(s)
              # This is an example, glob patterns might need adjustment
              find "${source_dir}" -maxdepth 1 -name "Vrooli*.$([ "$c" == "windows" ] && echo "exe" || ([ "$c" == "mac" ] && echo "dmg" || echo "AppImage"))" -exec cp {} "${dest_dir}/" \;
              log::success "Copied $c desktop artifact to ${dest_dir}"
            else
                log::warning "Remote destination not implemented for desktop app $c"
            fi
        fi

    done

    # Zip and compress the entire artifacts directory to the bundles directory
    zip::artifacts "${artifacts_dir}" "${bundles_dir}"

     # --- Remote Destination Handling ---
    if env::is_location_remote "$DEST"; then
        local ssh_key_path=$(keyless_ssh::get_key_path)
        log::info "Setting up SSH connection to remote server ${SITE_IP} using key ${ssh_key_path}..."
        keyless_ssh::connect

        local remote_bundles_dir="${var_REMOTE_DEST_DIR}/${VERSION}/bundles"
        log::info "Ensuring remote bundles directory ${SITE_IP}:${remote_bundles_dir} exists and is empty..."
        ssh -i "$ssh_key_path" "root@${SITE_IP}" "mkdir -p ${remote_bundles_dir} && rm -rf ${remote_bundles_dir}/*" || {
            log::error "Failed to create or clean remote bundles directory ${SITE_IP}:${remote_bundles_dir}"
            exit "$ERROR_REMOTE_OPERATION_FAILED"
        }

        log::info "Copying compressed build artifacts to ${SITE_IP}:${remote_bundles_dir}..."
        rsync -avz --progress -e "ssh -i $ssh_key_path" "${bundles_dir}/artifacts.zip.gz" "root@${SITE_IP}:${remote_bundles_dir}/" || {
            log::error "Failed to copy compressed build artifacts to ${SITE_IP}:${remote_bundles_dir}"
            exit "$ERROR_REMOTE_OPERATION_FAILED"
        }
        log::success "Compressed build artifacts copied to ${SITE_IP}:${remote_bundles_dir}"

        log::success "✅ Remote copy completed. You can now run deploy.sh on the remote server (${SITE_IP})."
    else
        log::success "✅ Local copy completed. You can now run deploy.sh on the local server."
    fi
}

build::main "$@" 