#!/usr/bin/env bash
################################################################################
# Universal Build Phase Handler
# 
# Handles generic build tasks:
# - Test execution (if enabled)
# - Linting (if enabled)
# - Artifact generation
# - Package bundling
# - Version management
#
# App-specific logic should be in app/lifecycle/build.sh
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

################################################################################
# Build Functions
################################################################################

#######################################
# Run tests before build
# Returns:
#   0 if tests pass or skipped, 1 on failure
#######################################
build::run_tests() {
    local test_enabled="${TEST:-no}"
    
    if ! flow::is_yes "$test_enabled"; then
        log::info "Tests skipped (use --test yes to enable)"
        return 0
    fi
    
    log::header "🧪 Running Tests"
    
    # Look for test scripts
    local test_commands=(
        "pnpm test:unit"
        "pnpm test"
        "npm test"
        "./scripts/manage.sh test"
    )
    
    for cmd in "${test_commands[@]}"; do
        if [[ "$cmd" == "pnpm"* ]] && command -v pnpm &> /dev/null; then
            log::info "Running: $cmd"
            if (cd "${var_ROOT_DIR}" && eval "$cmd"); then
                log::success "✅ Tests passed"
                return 0
            else
                log::error "Tests failed"
                return 1
            fi
        elif [[ "$cmd" == "npm"* ]] && command -v npm &> /dev/null; then
            log::info "Running: $cmd"
            if (cd "${var_ROOT_DIR}" && eval "$cmd"); then
                log::success "✅ Tests passed"
                return 0
            else
                log::error "Tests failed"
                return 1
            fi
        elif [[ "$cmd" == "./scripts/manage.sh"* ]] && [[ -f "${var_SCRIPTS_DIR}/manage.sh" ]]; then
            log::info "Running: $cmd"
            if (cd "${var_ROOT_DIR}" && eval "$cmd"); then
                log::success "✅ Tests passed"
                return 0
            else
                log::error "Tests failed"
                return 1
            fi
        fi
    done
    
    log::warning "No test command found, skipping tests"
    return 0
}

#######################################
# Run linting before build
# Returns:
#   0 if linting passes or skipped, 1 on failure
#######################################
build::run_linting() {
    local lint_enabled="${LINT:-no}"
    
    if ! flow::is_yes "$lint_enabled"; then
        log::info "Linting skipped (use --lint yes to enable)"
        return 0
    fi
    
    log::header "🔍 Running Linting"
    
    # Look for lint scripts
    local lint_commands=(
        "pnpm lint"
        "npm run lint"
        "./scripts/manage.sh lint"
    )
    
    for cmd in "${lint_commands[@]}"; do
        if [[ "$cmd" == "pnpm"* ]] && command -v pnpm &> /dev/null; then
            log::info "Running: $cmd"
            if (cd "${var_ROOT_DIR}" && eval "$cmd"); then
                log::success "✅ Linting passed"
                return 0
            else
                log::error "Linting failed"
                return 1
            fi
        elif [[ "$cmd" == "npm"* ]] && command -v npm &> /dev/null; then
            log::info "Running: $cmd"
            if (cd "${var_ROOT_DIR}" && eval "$cmd"); then
                log::success "✅ Linting passed"
                return 0
            else
                log::error "Linting failed"
                return 1
            fi
        elif [[ "$cmd" == "./scripts/manage.sh"* ]] && [[ -f "${var_SCRIPTS_DIR}/manage.sh" ]]; then
            log::info "Running: $cmd"
            if (cd "${var_ROOT_DIR}" && eval "$cmd"); then
                log::success "✅ Linting passed"
                return 0
            else
                log::error "Linting failed"
                return 1
            fi
        fi
    done
    
    log::warning "No lint command found, skipping linting"
    return 0
}

#######################################
# Get project version
# Returns:
#   Version string
#######################################
build::get_version() {
    local version="${VERSION:-}"
    
    if [[ -n "$version" ]]; then
        echo "$version"
        return
    fi
    
    # Try to get version from package.json
    if [[ -f "${var_ROOT_DIR}/package.json" ]] && command -v jq &> /dev/null; then
        version=$(jq -r '.version // empty' "${var_ROOT_DIR}/package.json")
    fi
    
    # Try to get version from service.json
    if [[ -z "$version" ]] && [[ -f "${var_SERVICE_JSON_FILE}" ]] && command -v jq &> /dev/null; then
        version=$(jq -r '.service.version // .version // empty' "${var_SERVICE_JSON_FILE}")
    fi
    
    # Try git describe
    if [[ -z "$version" ]] && command -v git &> /dev/null && [[ -d "${var_ROOT_DIR}/.git" ]]; then
        version=$(cd "${var_ROOT_DIR}" && git describe --tags --always 2>/dev/null || echo "")
    fi
    
    # Default to dev
    if [[ -z "$version" ]]; then
        version="dev"
    fi
    
    echo "$version"
}

#######################################
# Build Docker artifacts
# Arguments:
#   $1 - Version
# Returns:
#   0 on success
#######################################
build::docker_artifacts() {
    local version="${1:-dev}"
    
    log::info "Building Docker artifacts (version: $version)..."
    
    # Look for docker-compose files
    local compose_files=(
        "${var_DOCKER_COMPOSE_DEV_FILE}"
        "${var_ROOT_DIR}/docker-compose.yaml"
        "${var_ROOT_DIR}/compose.yml"
        "${var_ROOT_DIR}/compose.yaml"
    )
    
    for compose_file in "${compose_files[@]}"; do
        if [[ -f "$compose_file" ]]; then
            log::info "Found compose file: $compose_file"
            
            if command -v docker-compose &> /dev/null; then
                (cd "$PROJECT_ROOT" && docker-compose build)
            else
                (cd "$PROJECT_ROOT" && docker compose build)
            fi
            
            log::success "✅ Docker artifacts built"
            return 0
        fi
    done
    
    # Look for Dockerfile
    if [[ -f "${var_ROOT_DIR}/Dockerfile" ]]; then
        log::info "Building from Dockerfile..."
        local image_name="${IMAGE_NAME:-app}"
        (cd "${var_ROOT_DIR}" && docker build -t "${image_name}:${version}" .)
        log::success "✅ Docker image built: ${image_name}:${version}"
        return 0
    fi
    
    log::warning "No Docker configuration found"
    return 0
}

#######################################
# Build Kubernetes artifacts
# Arguments:
#   $1 - Version
# Returns:
#   0 on success
#######################################
build::k8s_artifacts() {
    local version="${1:-dev}"
    
    log::info "Building Kubernetes artifacts (version: $version)..."
    
    # Look for k8s directories
    local k8s_dirs=(
        "${var_ROOT_DIR}/k8s"
        "${var_ROOT_DIR}/kubernetes"
        "${var_ROOT_DIR}/manifests"
        "${var_ROOT_DIR}/deploy/k8s"
    )
    
    for k8s_dir in "${k8s_dirs[@]}"; do
        if [[ -d "$k8s_dir" ]]; then
            log::info "Found Kubernetes manifests: $k8s_dir"
            
            # Process templates if they exist
            if command -v envsubst &> /dev/null; then
                find "$k8s_dir" -name "*.template.yaml" -o -name "*.template.yml" | while read -r template; do
                    output="${template%.template.*}.yaml"
                    log::info "Processing template: $template"
                    VERSION="$version" envsubst < "$template" > "$output"
                done
            fi
            
            log::success "✅ Kubernetes artifacts prepared"
            return 0
        fi
    done
    
    log::warning "No Kubernetes configuration found"
    return 0
}

################################################################################
# Main Build Logic
################################################################################

#######################################
# Run universal build tasks
# Handles generic build operations
# Globals:
#   TEST
#   LINT
#   VERSION
#   BUNDLES
#   ARTIFACTS
#   ENVIRONMENT
# Returns:
#   0 on success, 1 on failure
#######################################
build::universal::main() {
    # Initialize phase
    phase::init "Build"
    
    # Get parameters from environment or defaults
    local test="${TEST:-no}"
    local lint="${LINT:-no}"
    local version="${VERSION:-}"
    local bundles="${BUNDLES:-zip}"
    local artifacts="${ARTIFACTS:-docker}"
    local environment="${ENVIRONMENT:-development}"
    
    # Get version
    version=$(build::get_version)
    export VERSION="$version"
    
    log::info "Universal build starting..."
    log::debug "Parameters:"
    log::debug "  Version: $version"
    log::debug "  Test: $test"
    log::debug "  Lint: $lint"
    log::debug "  Bundles: $bundles"
    log::debug "  Artifacts: $artifacts"
    log::debug "  Environment: $environment"
    
    # Step 1: Run pre-build hook
    phase::run_hook "preBuild"
    
    # Step 2: Run tests if enabled
    if ! build::run_tests; then
        log::error "Build aborted due to test failures"
        return 1
    fi
    
    # Step 3: Run linting if enabled
    if ! build::run_linting; then
        log::error "Build aborted due to linting failures"
        return 1
    fi
    
    # Step 4: Build application
    log::header "🔨 Building Application"
    
    # Check if service.json has build steps defined
    local has_steps=false
    local service_json="${var_SERVICE_JSON_FILE:-${var_ROOT_DIR}/.vrooli/service.json}"
    
    if [[ -f "$service_json" ]] && command -v jq &> /dev/null; then
        local build_steps
        build_steps=$(jq -r '.lifecycle.build.steps // [] | length' "$service_json" 2>/dev/null || echo "0")
        if [[ "$build_steps" -gt 0 ]]; then
            has_steps=true
            log::info "Build steps will be executed from service.json"
        fi
    fi
    
    # If no steps defined, try generic build commands
    if [[ "$has_steps" == "false" ]]; then
        if [[ -f "${var_ROOT_DIR}/package.json" ]]; then
            if command -v pnpm &> /dev/null; then
                log::info "Running: pnpm build"
                (cd "$PROJECT_ROOT" && pnpm build) || {
                    log::error "Build failed"
                    return 1
                }
            elif command -v npm &> /dev/null; then
                log::info "Running: npm run build"
                (cd "$PROJECT_ROOT" && npm run build) || {
                    log::error "Build failed"
                    return 1
                }
            fi
        fi
    fi
    
    # Step 5: Build artifacts
    log::header "📦 Building Artifacts"
    
    # Parse artifacts list
    IFS=',' read -ra ARTIFACT_LIST <<< "$artifacts"
    
    for artifact in "${ARTIFACT_LIST[@]}"; do
        case "$artifact" in
            docker)
                build::docker_artifacts "$version"
                ;;
            k8s|kubernetes)
                build::k8s_artifacts "$version"
                ;;
            all)
                build::docker_artifacts "$version"
                build::k8s_artifacts "$version"
                ;;
            *)
                log::warning "Unknown artifact type: $artifact"
                ;;
        esac
    done
    
    # Step 6: Create bundles
    if [[ "$bundles" != "none" ]]; then
        log::header "📦 Creating Bundles"
        
        local bundle_script="${var_APP_PACKAGE_DIR:-}/bundle.sh"
        if [[ -f "$bundle_script" ]]; then
            log::info "Creating bundles: $bundles"
            BUNDLES="$bundles" bash "$bundle_script"
        else
            log::warning "Bundle creation not available"
        fi
    fi
    
    # Step 7: Run post-build hook
    phase::run_hook "postBuild"
    
    # Complete phase
    phase::complete
    
    # Export build information
    export BUILD_VERSION="$version"
    export BUILD_ARTIFACTS="$artifacts"
    export BUILD_BUNDLES="$bundles"
    
    log::success "✅ Build completed successfully"
    log::info "Version: $version"
    
    return 0
}

#######################################
# Entry point for direct execution
#######################################
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Check if being called by lifecycle engine
    if [[ "${LIFECYCLE_PHASE:-}" == "build" ]]; then
        build::universal::main "$@"
    else
        log::error "This script should be called through the lifecycle engine"
        log::info "Use: ./scripts/manage.sh build [options]"
        exit 1
    fi
fi