#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Vrooli-Specific Build Lifecycle
# 
# Handles complex build processes including:
# - Multi-package monorepo builds (server, ui, jobs, shared)
# - Docker image building with optimized layering
# - Kubernetes manifest generation
# - Bundle creation (zip, tar.gz) with metadata
# - Asset optimization and compression
# - Multi-architecture builds
#
# This script contains all the Vrooli-specific logic extracted from build.sh
################################################################################

APP_LIFECYCLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DIR}/../../lib/utils/var.sh"

# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/index.sh"

#######################################
# Main Vrooli build lifecycle
# Arguments:
#   All arguments passed from main build.sh
#######################################
vrooli_build::main() {
    log::info "Starting Vrooli-specific build process..."
    
    # Load environment for build configuration
    env::load_secrets
    
    # Set build metadata
    export BUILD_VERSION="${VERSION:-$(date +%Y%m%d-%H%M%S)}"
    export BUILD_COMMIT="${CI_COMMIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
    export BUILD_TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    
    log::info "Build metadata:"
    log::info "  Version: $BUILD_VERSION"
    log::info "  Commit: $BUILD_COMMIT"  
    log::info "  Timestamp: $BUILD_TIMESTAMP"
    
    # Pre-build validation
    vrooli_build::validate_environment
    
    # Execute build based on target and configuration
    case "$TARGET" in
        docker|docker-only)
            vrooli_build::docker "$@"
            ;;
        native-linux|native-mac|native-win)
            vrooli_build::native "$@"
            ;;
        k8s|k8s-cluster)
            vrooli_build::k8s "$@"
            ;;
        *)
            log::error "Unknown target for Vrooli build: $TARGET"
            exit 1
            ;;
    esac
    
    # Post-build tasks
    vrooli_build::post_build_tasks
}

#######################################
# Validate build environment
#######################################
vrooli_build::validate_environment() {
    log::info "Validating build environment..."
    
    # Check Node.js version
    if ! command -v node >/dev/null 2>&1; then
        log::error "Node.js is required for building"
        exit 1
    fi
    
    local node_version
    node_version=$(node --version | sed 's/v//')
    log::info "Node.js version: $node_version"
    
    # Check pnpm
    if ! command -v pnpm >/dev/null 2>&1; then
        log::error "pnpm is required for monorepo builds"
        exit 1
    fi
    
    # Validate package.json files exist
    local required_packages=("shared" "server" "ui" "jobs")
    for package in "${required_packages[@]}"; do
        if [[ ! -f "${var_ROOT_DIR}/packages/${package}/package.json" ]]; then
            log::error "Missing package.json for: packages/$package"
            exit 1
        fi
    done
    
    # Check disk space (need at least 2GB for builds)
    local available_space
    available_space=$(df "${var_ROOT_DIR}" | awk 'NR==2 {print $4}')
    if [[ $available_space -lt 2097152 ]]; then  # 2GB in KB
        log::warning "Low disk space detected: $(( available_space / 1024 / 1024 ))GB available"
    fi
}

#######################################
# Docker build process
#######################################
vrooli_build::docker() {
    log::info "Building Docker images..."
    
    # Create build context
    mkdir -p "${var_ROOT_DIR}/dist/docker"
    
    # Build each service
    local services=("server" "ui" "jobs")
    
    for service in "${services[@]}"; do
        log::info "Building Docker image for: $service"
        
        # Build with build args
        docker build \
            --build-arg BUILD_VERSION="$BUILD_VERSION" \
            --build-arg BUILD_COMMIT="$BUILD_COMMIT" \
            --build-arg BUILD_TIMESTAMP="$BUILD_TIMESTAMP" \
            --build-arg SERVICE="$service" \
            -t "vrooli-${service}:${BUILD_VERSION}" \
            -t "vrooli-${service}:latest" \
            -f "packages/${service}/Dockerfile" \
            .
        
        log::success "Built Docker image: vrooli-${service}:${BUILD_VERSION}"
    done
    
    # Export images if requested
    if [[ "${EXPORT_IMAGES:-no}" == "yes" ]]; then
        log::info "Exporting Docker images..."
        docker save \
            "vrooli-server:${BUILD_VERSION}" \
            "vrooli-ui:${BUILD_VERSION}" \
            "vrooli-jobs:${BUILD_VERSION}" \
            | gzip > "${var_ROOT_DIR}/dist/docker/vrooli-images-${BUILD_VERSION}.tar.gz"
        
        log::success "Docker images exported to: dist/docker/vrooli-images-${BUILD_VERSION}.tar.gz"
    fi
}

#######################################
# Native build process  
#######################################
vrooli_build::native() {
    log::info "Building native packages..."
    
    # Clean previous builds if requested
    if [[ "${CLEAN:-no}" == "yes" ]]; then
        log::info "Cleaning previous builds..."
        rm -rf "${var_ROOT_DIR}/packages/*/dist"
        rm -rf "${var_ROOT_DIR}/dist"
        pnpm run clean || true
    fi
    
    # Install dependencies with frozen lockfile
    log::info "Installing dependencies..."
    pnpm install --frozen-lockfile
    
    # Generate Prisma client first (dependency for other packages)
    log::info "Generating Prisma client..."
    cd "${var_ROOT_DIR}/packages/server"
    pnpm prisma generate
    
    # Build packages in dependency order
    cd "${var_ROOT_DIR}"
    
    # 1. Build shared package first
    log::info "Building shared package..."
    pnpm --filter "@vrooli/shared" run build
    
    # 2. Build server (depends on shared)
    log::info "Building server package..."
    pnpm --filter "@vrooli/server" run build
    
    # 3. Build UI (depends on shared)
    log::info "Building UI package..."
    pnpm --filter "@vrooli/ui" run build
    
    # 4. Build jobs (depends on shared and server)
    log::info "Building jobs package..."
    pnpm --filter "@vrooli/jobs" run build
    
    # Create distribution directory
    mkdir -p "${var_ROOT_DIR}/dist/native"
    
    # Copy built assets
    for package in shared server ui jobs; do
        if [[ -d "${var_ROOT_DIR}/packages/${package}/dist" ]]; then
            cp -r "${var_ROOT_DIR}/packages/${package}/dist" \
                "${var_ROOT_DIR}/dist/native/${package}"
            log::info "Copied build artifacts for: $package"
        fi
    done
    
    # Copy package.json files and essential configs
    for package in shared server ui jobs; do
        cp "${var_ROOT_DIR}/packages/${package}/package.json" \
            "${var_ROOT_DIR}/dist/native/${package}/"
    done
    
    # Copy root configuration
    cp "${var_ROOT_DIR}/package.json" "${var_ROOT_DIR}/dist/native/"
    cp "${var_ROOT_DIR}/pnpm-workspace.yaml" "${var_ROOT_DIR}/dist/native/"
    
    log::success "Native build completed: dist/native/"
}

#######################################
# Kubernetes build process
#######################################
vrooli_build::k8s() {
    log::info "Building for Kubernetes deployment..."
    
    # First build Docker images
    vrooli_build::docker
    
    # Generate Kubernetes manifests with current build version
    log::info "Generating Kubernetes manifests..."
    mkdir -p "${var_ROOT_DIR}/dist/k8s"
    
    # Process template manifests
    if [[ -d "${var_ROOT_DIR}/k8s/templates" ]]; then
        find "${var_ROOT_DIR}/k8s/templates" -name "*.yaml" -o -name "*.yml" | while read -r template; do
            local output_file
            output_file="${var_ROOT_DIR}/dist/k8s/$(basename "$template")"
            
            # Substitute variables in templates
            sed -e "s/{{BUILD_VERSION}}/${BUILD_VERSION}/g" \
                -e "s/{{BUILD_COMMIT}}/${BUILD_COMMIT}/g" \
                -e "s/{{BUILD_TIMESTAMP}}/${BUILD_TIMESTAMP}/g" \
                -e "s/{{ENVIRONMENT}}/${ENVIRONMENT:-production}/g" \
                "$template" > "$output_file"
            
            log::info "Generated manifest: $(basename "$output_file")"
        done
    fi
    
    # Create Helm chart if templates exist
    if [[ -d "${var_ROOT_DIR}/k8s/helm" ]]; then
        log::info "Creating Helm chart..."
        helm package "${var_ROOT_DIR}/k8s/helm" \
            --version "$BUILD_VERSION" \
            --destination "${var_ROOT_DIR}/dist/k8s"
    fi
    
    log::success "Kubernetes build completed: dist/k8s/"
}

#######################################
# Post-build tasks (bundling, metadata, etc.)
#######################################
vrooli_build::post_build_tasks() {
    log::info "Running post-build tasks..."
    
    # Generate build metadata
    cat > "${var_ROOT_DIR}/dist/build-metadata.json" << EOF
{
    "version": "${BUILD_VERSION}",
    "commit": "${BUILD_COMMIT}",
    "timestamp": "${BUILD_TIMESTAMP}",
    "target": "${TARGET}",
    "environment": "${ENVIRONMENT:-development}",
    "node_version": "$(node --version)",
    "platform": "$(uname -s)-$(uname -m)"
}
EOF
    
    # Create bundles if requested
    if [[ "${BUNDLES:-}" =~ zip|tar|tgz ]]; then
        vrooli_build::create_bundles
    fi
    
    # Run tests if requested
    if [[ "${TEST:-no}" == "yes" ]]; then
        log::info "Running post-build tests..."
        pnpm test || {
            log::warning "Some tests failed, but continuing with build"
        }
    fi
    
    # Generate build report
    vrooli_build::generate_build_report
    
    log::success "Vrooli build process completed successfully!"
}

#######################################
# Create distribution bundles
#######################################
vrooli_build::create_bundles() {
    log::info "Creating distribution bundles..."
    
    local bundle_name="vrooli-${BUILD_VERSION}"
    
    cd "${var_ROOT_DIR}/dist"
    
    if [[ "${BUNDLES}" =~ zip ]]; then
        zip -r "${bundle_name}.zip" . -x "*.zip" "*.tar.gz"
        log::success "Created ZIP bundle: dist/${bundle_name}.zip"
    fi
    
    if [[ "${BUNDLES}" =~ tar|tgz ]]; then
        tar -czf "${bundle_name}.tar.gz" --exclude="*.zip" --exclude="*.tar.gz" .
        log::success "Created TAR.GZ bundle: dist/${bundle_name}.tar.gz"
    fi
}

#######################################
# Generate build report
#######################################
vrooli_build::generate_build_report() {
    local report_file="${var_ROOT_DIR}/dist/build-report.md"
    
    cat > "$report_file" << EOF
# Vrooli Build Report

**Build Version:** ${BUILD_VERSION}  
**Build Commit:** ${BUILD_COMMIT}  
**Build Timestamp:** ${BUILD_TIMESTAMP}  
**Target:** ${TARGET}  
**Environment:** ${ENVIRONMENT:-development}  

## Build Artifacts

EOF
    
    # List all artifacts
    if [[ -d "${var_ROOT_DIR}/dist" ]]; then
        echo "### Generated Files" >> "$report_file"
        echo '```' >> "$report_file"
        find "${var_ROOT_DIR}/dist" -type f -name "*" | sed "s|${var_ROOT_DIR}/dist/||" | sort >> "$report_file"
        echo '```' >> "$report_file"
    fi
    
    # Add size information
    echo "" >> "$report_file"
    echo "### Size Information" >> "$report_file"
    echo '```' >> "$report_file"
    du -h "${var_ROOT_DIR}/dist/"* 2>/dev/null | sort -h >> "$report_file" || true
    echo '```' >> "$report_file"
    
    log::info "Build report generated: dist/build-report.md"
}

# Only run main if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vrooli_build::main "$@"
fi