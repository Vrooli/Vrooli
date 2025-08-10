#!/usr/bin/env bash
set -euo pipefail

# Helm deployment library - comprehensive Helm operations for Kubernetes deployments
# Provides installation, chart management, release operations, and health checks

# Get runtime directory
RUNTIME_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${RUNTIME_DIR}/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

# Global configuration
HELM_TIMEOUT="${HELM_TIMEOUT:-600}"
HELM_NAMESPACE="${HELM_NAMESPACE:-default}"
HELM_HISTORY_MAX="${HELM_HISTORY_MAX:-10}"
HELM_WAIT="${HELM_WAIT:-true}"
HELM_DEBUG="${HELM_DEBUG:-false}"

# Clean interface for setup.sh
helm::ensure_installed() {
    helm::check_and_install "$@"
}

# Check and install Helm CLI for Kubernetes package management
helm::check_and_install() {
    log::info "Checking for Helm CLI..."
    
    # Check if Helm is already installed
    if system::is_command "helm"; then
        local version
        version=$(helm version --short 2>/dev/null | cut -d: -f2 | cut -d+ -f1 | tr -d ' v')
        log::success "Helm is already installed (version: $version)"
        return 0
    fi
    
    log::info "Installing Helm CLI..."
    
    # Try package manager first
    local pm
    pm=$(system::detect_pm)
    
    case "$pm" in
        apt-get|dnf|yum)
            # Install via official Helm script
            if helm::install_via_script; then
                log::success "Helm installed successfully via official script"
                return 0
            fi
            ;;
        brew)
            if system::install_pkg "helm"; then
                log::success "Helm installed successfully via Homebrew"
                return 0
            fi
            ;;
        pacman)
            if system::install_pkg "helm"; then
                log::success "Helm installed successfully via pacman"
                return 0
            fi
            ;;
        apk)
            if system::install_pkg "helm"; then
                log::success "Helm installed successfully via apk"
                return 0
            fi
            ;;
        *)
            log::warning "Unknown package manager $pm, trying official script"
            if helm::install_via_script; then
                log::success "Helm installed successfully via official script"
                return 0
            fi
            ;;
    esac
    
    log::error "Failed to install Helm"
    return 1
}

# Install Helm using the official installation script
helm::install_via_script() {
    local tmpdir
    tmpdir=$(mktemp -d)
    
    # Download and run the official Helm installation script
    if curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 -o "$tmpdir/get_helm.sh"; then
        chmod 700 "$tmpdir/get_helm.sh"
        
        # Check if we need sudo
        local use_sudo="true"
        if [ "${SUDO_MODE:-error}" = "skip" ] || ! command -v sudo >/dev/null 2>&1 || ! sudo -n true >/dev/null 2>&1; then
            use_sudo="false"
            export HELM_INSTALL_DIR="$HOME/.local/bin"
            mkdir -p "$HELM_INSTALL_DIR"
            
            # Ensure local bin is in PATH
            if [[ ":$PATH:" != *":$HELM_INSTALL_DIR:"* ]]; then
                export PATH="$HELM_INSTALL_DIR:$PATH"
            fi
        fi
        
        # Run the installation script
        if [ "$use_sudo" = "true" ]; then
            if sudo "$tmpdir/get_helm.sh"; then
                if command -v trash::safe_remove >/dev/null 2>&1; then
    trash::safe_remove "$tmpdir" --no-confirm
else
    rm -rf "$tmpdir"
fi
                return 0
            fi
        else
            if USE_SUDO=false "$tmpdir/get_helm.sh"; then
                if command -v trash::safe_remove >/dev/null 2>&1; then
    trash::safe_remove "$tmpdir" --no-confirm
else
    rm -rf "$tmpdir"
fi
                return 0
            fi
        fi
    fi
    
    if command -v trash::safe_remove >/dev/null 2>&1; then
    trash::safe_remove "$tmpdir" --no-confirm
else
    rm -rf "$tmpdir"
fi
    return 1
}

# Install a specific version of Helm
helm::install_version() {
    local version="${1:-latest}"
    local tmpdir
    tmpdir=$(mktemp -d)
    
    log::info "Installing Helm version: $version"
    
    # Determine system architecture
    local arch
    case "$(uname -m)" in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv7l) arch="arm" ;;
        *) 
            log::error "Unsupported architecture: $(uname -m)"
            return 1
            ;;
    esac
    
    # Determine OS
    local os
    case "$(uname -s)" in
        Linux) os="linux" ;;
        Darwin) os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *)
            log::error "Unsupported OS: $(uname -s)"
            return 1
            ;;
    esac
    
    # Download URL
    local url="https://get.helm.sh/helm-${version}-${os}-${arch}.tar.gz"
    
    # Download and extract
    if curl -fsSL "$url" | tar -xz -C "$tmpdir"; then
        # Determine installation directory
        local install_dir="/usr/local/bin"
        local use_sudo="true"
        
        if [ "${SUDO_MODE:-error}" = "skip" ] || ! command -v sudo >/dev/null 2>&1 || ! sudo -n true >/dev/null 2>&1; then
            use_sudo="false"
            install_dir="$HOME/.local/bin"
            mkdir -p "$install_dir"
            
            # Ensure local bin is in PATH
            if [[ ":$PATH:" != *":$install_dir:"* ]]; then
                export PATH="$install_dir:$PATH"
            fi
        fi
        
        # Install the binary
        if [ "$use_sudo" = "true" ]; then
            sudo mv "$tmpdir/${os}-${arch}/helm" "$install_dir/helm"
            sudo chmod +x "$install_dir/helm"
        else
            mv "$tmpdir/${os}-${arch}/helm" "$install_dir/helm"
            chmod +x "$install_dir/helm"
        fi
        
        trash::safe_remove "$tmpdir" --no-confirm
        log::success "Helm $version installed to $install_dir"
        return 0
    fi
    
    trash::safe_remove "$tmpdir" --no-confirm
    log::error "Failed to install Helm $version"
    return 1
}

# Verify Helm installation
helm::verify() {
    if system::is_command "helm"; then
        local version
        version=$(helm version --short 2>/dev/null | cut -d: -f2 | cut -d+ -f1 | tr -d ' v')
        log::success "Helm is installed (version: $version)"
        return 0
    else
        log::error "Helm is not installed"
        return 1
    fi
}

# ========================================
# Repository Management
# ========================================

# Add a Helm repository
helm::repo_add() {
    local name="${1:?Repository name required}"
    local url="${2:?Repository URL required}"
    local username="${3:-}"
    local password="${4:-}"
    
    log::info "Adding Helm repository: $name"
    
    local cmd="helm repo add"
    [[ -n "$username" ]] && cmd="$cmd --username '$username'"
    [[ -n "$password" ]] && cmd="$cmd --password '$password'"
    cmd="$cmd '$name' '$url'"
    
    if eval "$cmd" 2>/dev/null; then
        helm repo update "$name" 2>/dev/null
        log::success "Repository $name added successfully"
        return 0
    else
        log::error "Failed to add repository $name"
        return 1
    fi
}

# Update Helm repositories
helm::repo_update() {
    local repo="${1:-}"
    
    if [[ -n "$repo" ]]; then
        log::info "Updating repository: $repo"
        if helm repo update "$repo" 2>/dev/null; then
            log::success "Repository $repo updated"
            return 0
        else
            log::error "Failed to update repository $repo"
            return 1
        fi
    else
        log::info "Updating all repositories"
        if helm repo update 2>/dev/null; then
            log::success "All repositories updated"
            return 0
        else
            log::error "Failed to update repositories"
            return 1
        fi
    fi
}

# List Helm repositories
helm::repo_list() {
    helm repo list 2>/dev/null || true
}

# Remove a Helm repository
helm::repo_remove() {
    local name="${1:?Repository name required}"
    
    log::info "Removing repository: $name"
    if helm repo remove "$name" 2>/dev/null; then
        log::success "Repository $name removed"
        return 0
    else
        log::error "Failed to remove repository $name"
        return 1
    fi
}

# ========================================
# Chart Management
# ========================================

# Search for charts in repositories
helm::search() {
    local keyword="${1:?Search keyword required}"
    local repo="${2:-}"
    
    if [[ -n "$repo" ]]; then
        helm search repo "$repo/$keyword" 2>/dev/null || true
    else
        helm search repo "$keyword" 2>/dev/null || true
    fi
}

# Pull a chart to local filesystem
helm::pull() {
    local chart="${1:?Chart name required}"
    local version="${2:-}"
    local destination="${3:-.}"
    
    local cmd="helm pull '$chart'"
    [[ -n "$version" ]] && cmd="$cmd --version '$version'"
    cmd="$cmd --destination '$destination' --untar"
    
    log::info "Pulling chart: $chart"
    if eval "$cmd" 2>/dev/null; then
        log::success "Chart $chart pulled successfully"
        return 0
    else
        log::error "Failed to pull chart $chart"
        return 1
    fi
}

# Show chart information
helm::show() {
    local chart="${1:?Chart name required}"
    local section="${2:-all}"  # all, values, chart, readme, crds
    
    helm show "$section" "$chart" 2>/dev/null || true
}

# ========================================
# Release Management
# ========================================

# Install a Helm release
helm::install() {
    local release="${1:?Release name required}"
    local chart="${2:?Chart name/path required}"
    local namespace="${3:-$HELM_NAMESPACE}"
    local values_file="${4:-}"
    local wait="${5:-$HELM_WAIT}"
    local timeout="${6:-$HELM_TIMEOUT}"
    shift 6 2>/dev/null || true
    local extra_args=("$@")
    
    local cmd="helm install '$release' '$chart'"
    cmd="$cmd --namespace '$namespace' --create-namespace"
    cmd="$cmd --timeout '${timeout}s'"
    
    [[ "$wait" == "true" ]] && cmd="$cmd --wait"
    [[ -n "$values_file" ]] && [[ -f "$values_file" ]] && cmd="$cmd --values '$values_file'"
    [[ "$HELM_DEBUG" == "true" ]] && cmd="$cmd --debug"
    
    # Add extra arguments
    for arg in "${extra_args[@]:-}"; do
        [[ -n "$arg" ]] && cmd="$cmd $arg"
    done
    
    log::info "Installing release: $release in namespace: $namespace"
    if eval "$cmd" 2>&1 | tee /tmp/helm_install_$$.log; then
        log::success "Release $release installed successfully"
        rm -f /tmp/helm_install_$$.log
        return 0
    else
        log::error "Failed to install release $release"
        cat /tmp/helm_install_$$.log >&2
        rm -f /tmp/helm_install_$$.log
        return 1
    fi
}

# Upgrade a Helm release
helm::upgrade() {
    local release="${1:?Release name required}"
    local chart="${2:?Chart name/path required}"
    local namespace="${3:-$HELM_NAMESPACE}"
    local values_file="${4:-}"
    local wait="${5:-$HELM_WAIT}"
    local timeout="${6:-$HELM_TIMEOUT}"
    shift 6 2>/dev/null || true
    local extra_args=("$@")
    
    local cmd="helm upgrade '$release' '$chart'"
    cmd="$cmd --namespace '$namespace'"
    cmd="$cmd --timeout '${timeout}s'"
    cmd="$cmd --history-max '$HELM_HISTORY_MAX'"
    
    [[ "$wait" == "true" ]] && cmd="$cmd --wait"
    [[ -n "$values_file" ]] && [[ -f "$values_file" ]] && cmd="$cmd --values '$values_file'"
    [[ "$HELM_DEBUG" == "true" ]] && cmd="$cmd --debug"
    
    # Add extra arguments
    for arg in "${extra_args[@]:-}"; do
        [[ -n "$arg" ]] && cmd="$cmd $arg"
    done
    
    log::info "Upgrading release: $release in namespace: $namespace"
    if eval "$cmd" 2>&1 | tee /tmp/helm_upgrade_$$.log; then
        log::success "Release $release upgraded successfully"
        rm -f /tmp/helm_upgrade_$$.log
        return 0
    else
        log::error "Failed to upgrade release $release"
        cat /tmp/helm_upgrade_$$.log >&2
        rm -f /tmp/helm_upgrade_$$.log
        return 1
    fi
}

# Install or upgrade a release
helm::upgrade_install() {
    local release="${1:?Release name required}"
    local chart="${2:?Chart name/path required}"
    local namespace="${3:-$HELM_NAMESPACE}"
    local values_file="${4:-}"
    local wait="${5:-$HELM_WAIT}"
    local timeout="${6:-$HELM_TIMEOUT}"
    shift 6 2>/dev/null || true
    local extra_args=("$@")
    
    local cmd="helm upgrade --install '$release' '$chart'"
    cmd="$cmd --namespace '$namespace' --create-namespace"
    cmd="$cmd --timeout '${timeout}s'"
    cmd="$cmd --history-max '$HELM_HISTORY_MAX'"
    
    [[ "$wait" == "true" ]] && cmd="$cmd --wait"
    [[ -n "$values_file" ]] && [[ -f "$values_file" ]] && cmd="$cmd --values '$values_file'"
    [[ "$HELM_DEBUG" == "true" ]] && cmd="$cmd --debug"
    
    # Add extra arguments
    for arg in "${extra_args[@]:-}"; do
        [[ -n "$arg" ]] && cmd="$cmd $arg"
    done
    
    log::info "Installing/upgrading release: $release in namespace: $namespace"
    if eval "$cmd" 2>&1 | tee /tmp/helm_upgrade_install_$$.log; then
        log::success "Release $release installed/upgraded successfully"
        rm -f /tmp/helm_upgrade_install_$$.log
        return 0
    else
        log::error "Failed to install/upgrade release $release"
        cat /tmp/helm_upgrade_install_$$.log >&2
        rm -f /tmp/helm_upgrade_install_$$.log
        return 1
    fi
}

# Uninstall a Helm release
helm::uninstall() {
    local release="${1:?Release name required}"
    local namespace="${2:-$HELM_NAMESPACE}"
    local wait="${3:-$HELM_WAIT}"
    local timeout="${4:-$HELM_TIMEOUT}"
    
    local cmd="helm uninstall '$release'"
    cmd="$cmd --namespace '$namespace'"
    cmd="$cmd --timeout '${timeout}s'"
    
    [[ "$wait" == "true" ]] && cmd="$cmd --wait"
    [[ "$HELM_DEBUG" == "true" ]] && cmd="$cmd --debug"
    
    log::info "Uninstalling release: $release from namespace: $namespace"
    if eval "$cmd" 2>/dev/null; then
        log::success "Release $release uninstalled successfully"
        return 0
    else
        log::error "Failed to uninstall release $release"
        return 1
    fi
}

# Rollback a release to a previous revision
helm::rollback() {
    local release="${1:?Release name required}"
    local revision="${2:-0}"  # 0 means previous revision
    local namespace="${3:-$HELM_NAMESPACE}"
    local wait="${4:-$HELM_WAIT}"
    local timeout="${5:-$HELM_TIMEOUT}"
    
    local cmd="helm rollback '$release' '$revision'"
    cmd="$cmd --namespace '$namespace'"
    cmd="$cmd --timeout '${timeout}s'"
    
    [[ "$wait" == "true" ]] && cmd="$cmd --wait"
    [[ "$HELM_DEBUG" == "true" ]] && cmd="$cmd --debug"
    
    log::info "Rolling back release: $release to revision: $revision"
    if eval "$cmd" 2>/dev/null; then
        log::success "Release $release rolled back successfully"
        return 0
    else
        log::error "Failed to rollback release $release"
        return 1
    fi
}

# ========================================
# Release Information
# ========================================

# List releases
helm::list() {
    local namespace="${1:-}"
    local all_namespaces="${2:-false}"
    
    local cmd="helm list"
    [[ "$all_namespaces" == "true" ]] && cmd="$cmd --all-namespaces"
    [[ -n "$namespace" ]] && [[ "$all_namespaces" != "true" ]] && cmd="$cmd --namespace '$namespace'"
    
    eval "$cmd" 2>/dev/null || true
}

# Get release status
helm::status() {
    local release="${1:?Release name required}"
    local namespace="${2:-$HELM_NAMESPACE}"
    local format="${3:-}"  # json, yaml
    
    local cmd="helm status '$release' --namespace '$namespace'"
    [[ -n "$format" ]] && cmd="$cmd --output '$format'"
    
    eval "$cmd" 2>/dev/null || true
}

# Get release values
helm::get_values() {
    local release="${1:?Release name required}"
    local namespace="${2:-$HELM_NAMESPACE}"
    local all="${3:-false}"
    
    local cmd="helm get values '$release' --namespace '$namespace'"
    [[ "$all" == "true" ]] && cmd="$cmd --all"
    
    eval "$cmd" 2>/dev/null || true
}

# Get release history
helm::history() {
    local release="${1:?Release name required}"
    local namespace="${2:-$HELM_NAMESPACE}"
    local max="${3:-$HELM_HISTORY_MAX}"
    
    helm history "$release" --namespace "$namespace" --max "$max" 2>/dev/null || true
}

# Get release manifest
helm::get_manifest() {
    local release="${1:?Release name required}"
    local namespace="${2:-$HELM_NAMESPACE}"
    local revision="${3:-}"
    
    local cmd="helm get manifest '$release' --namespace '$namespace'"
    [[ -n "$revision" ]] && cmd="$cmd --revision '$revision'"
    
    eval "$cmd" 2>/dev/null || true
}

# ========================================
# Health and Testing
# ========================================

# Test a release
helm::test() {
    local release="${1:?Release name required}"
    local namespace="${2:-$HELM_NAMESPACE}"
    local timeout="${3:-$HELM_TIMEOUT}"
    
    log::info "Testing release: $release"
    if helm test "$release" --namespace "$namespace" --timeout "${timeout}s" 2>/dev/null; then
        log::success "Release $release tests passed"
        return 0
    else
        log::error "Release $release tests failed"
        return 1
    fi
}

# Check if a release exists
helm::release_exists() {
    local release="${1:?Release name required}"
    local namespace="${2:-$HELM_NAMESPACE}"
    
    helm status "$release" --namespace "$namespace" >/dev/null 2>&1
}

# Wait for release to be ready
helm::wait_for_release() {
    local release="${1:?Release name required}"
    local namespace="${2:-$HELM_NAMESPACE}"
    local timeout="${3:-$HELM_TIMEOUT}"
    
    log::info "Waiting for release $release to be ready..."
    
    local elapsed=0
    while [ $elapsed -lt "$timeout" ]; do
        if helm status "$release" --namespace "$namespace" 2>/dev/null | grep -q "STATUS: deployed"; then
            log::success "Release $release is ready"
            return 0
        fi
        sleep 5
        elapsed=$((elapsed + 5))
    done
    
    log::error "Timeout waiting for release $release"
    return 1
}

# ========================================
# Utility Functions
# ========================================

# Validate chart
helm::lint() {
    local chart="${1:?Chart path required}"
    local values_file="${2:-}"
    
    local cmd="helm lint '$chart'"
    [[ -n "$values_file" ]] && [[ -f "$values_file" ]] && cmd="$cmd --values '$values_file'"
    
    log::info "Linting chart: $chart"
    if eval "$cmd" 2>/dev/null; then
        log::success "Chart validation passed"
        return 0
    else
        log::error "Chart validation failed"
        return 1
    fi
}

# Render templates locally
helm::template() {
    local release="${1:?Release name required}"
    local chart="${2:?Chart name/path required}"
    local namespace="${3:-$HELM_NAMESPACE}"
    local values_file="${4:-}"
    local output_dir="${5:-}"
    
    local cmd="helm template '$release' '$chart'"
    cmd="$cmd --namespace '$namespace'"
    
    [[ -n "$values_file" ]] && [[ -f "$values_file" ]] && cmd="$cmd --values '$values_file'"
    [[ -n "$output_dir" ]] && cmd="$cmd --output-dir '$output_dir'"
    [[ "$HELM_DEBUG" == "true" ]] && cmd="$cmd --debug"
    
    eval "$cmd" 2>/dev/null || true
}

# Package a chart
helm::package() {
    local chart="${1:?Chart path required}"
    local destination="${2:-.}"
    local version="${3:-}"
    
    local cmd="helm package '$chart' --destination '$destination'"
    [[ -n "$version" ]] && cmd="$cmd --version '$version'"
    
    log::info "Packaging chart: $chart"
    if eval "$cmd" 2>/dev/null; then
        log::success "Chart packaged successfully"
        return 0
    else
        log::error "Failed to package chart"
        return 1
    fi
}

# Create a new chart
helm::create() {
    local name="${1:?Chart name required}"
    local starter="${2:-}"
    
    local cmd="helm create '$name'"
    [[ -n "$starter" ]] && cmd="$cmd --starter '$starter'"
    
    log::info "Creating chart: $name"
    if eval "$cmd" 2>/dev/null; then
        log::success "Chart $name created successfully"
        return 0
    else
        log::error "Failed to create chart $name"
        return 1
    fi
}

# Export functions
export -f helm::check_and_install
export -f helm::install_via_script
export -f helm::install_version
export -f helm::verify
export -f helm::repo_add
export -f helm::repo_update
export -f helm::repo_list
export -f helm::repo_remove
export -f helm::search
export -f helm::pull
export -f helm::show
export -f helm::install
export -f helm::upgrade
export -f helm::upgrade_install
export -f helm::uninstall
export -f helm::rollback
export -f helm::list
export -f helm::status
export -f helm::get_values
export -f helm::history
export -f helm::get_manifest
export -f helm::test
export -f helm::release_exists
export -f helm::wait_for_release
export -f helm::lint
export -f helm::template
export -f helm::package
export -f helm::create

# If this script is run directly, show usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
        echo "Helm deployment library - comprehensive Helm operations"
        echo ""
        echo "Usage: source helm.sh"
        echo ""
        echo "Available functions:"
        echo "  Installation:"
        echo "    helm::check_and_install    - Install Helm if not present"
        echo "    helm::install_version      - Install specific Helm version"
        echo "    helm::verify              - Verify Helm installation"
        echo ""
        echo "  Repository Management:"
        echo "    helm::repo_add            - Add a repository"
        echo "    helm::repo_update         - Update repositories"
        echo "    helm::repo_list          - List repositories"
        echo "    helm::repo_remove        - Remove a repository"
        echo ""
        echo "  Chart Management:"
        echo "    helm::search             - Search for charts"
        echo "    helm::pull               - Download a chart"
        echo "    helm::show               - Show chart information"
        echo "    helm::lint               - Validate a chart"
        echo "    helm::package            - Package a chart"
        echo "    helm::create             - Create a new chart"
        echo ""
        echo "  Release Management:"
        echo "    helm::install            - Install a release"
        echo "    helm::upgrade            - Upgrade a release"
        echo "    helm::upgrade_install    - Install or upgrade a release"
        echo "    helm::uninstall          - Uninstall a release"
        echo "    helm::rollback           - Rollback a release"
        echo ""
        echo "  Release Information:"
        echo "    helm::list               - List releases"
        echo "    helm::status             - Get release status"
        echo "    helm::get_values         - Get release values"
        echo "    helm::history            - Get release history"
        echo "    helm::get_manifest       - Get release manifest"
        echo ""
        echo "  Testing & Health:"
        echo "    helm::test               - Test a release"
        echo "    helm::release_exists     - Check if release exists"
        echo "    helm::wait_for_release   - Wait for release to be ready"
        echo ""
        echo "  Utilities:"
        echo "    helm::template           - Render templates locally"
        echo ""
        echo "Environment Variables:"
        echo "  HELM_TIMEOUT       - Operation timeout (default: 600s)"
        echo "  HELM_NAMESPACE     - Default namespace (default: default)"
        echo "  HELM_HISTORY_MAX   - Max history revisions (default: 10)"
        echo "  HELM_WAIT          - Wait for operations (default: true)"
        echo "  HELM_DEBUG         - Enable debug output (default: false)"
    else
        helm::check_and_install "$@"
    fi
fi