#!/usr/bin/env bash
# Helm Mock - Tier 2 (Stateful)
# 
# Provides stateful Helm Kubernetes package manager mocking for testing:
# - Chart repository management (add, remove, update, list)
# - Release lifecycle (install, upgrade, uninstall, rollback)
# - Release status and history tracking
# - Chart search and inspection
# - Error injection for resilience testing
#
# Coverage: ~80% of common Helm operations in 450 lines

# === Configuration ===
declare -gA HELM_RELEASES=()          # Release -> "namespace|chart|status|revision|values"
declare -gA HELM_REPOS=()             # Repo_name -> "url|status"
declare -gA HELM_CHARTS=()            # Chart_name -> "version|app_version|description"
declare -gA HELM_HISTORY=()           # Release -> "rev1:deployed:time|rev2:upgraded:time|..."
declare -gA HELM_CONFIG=(             # Global configuration
    [kubeconfig]="/root/.kube/config"
    [namespace]="default"
    [timeout]="5m0s"
    [error_mode]=""
    [version]="v3.12.0"
)

# Debug mode
declare -g HELM_DEBUG="${HELM_DEBUG:-}"

# === Helper Functions ===
helm_debug() {
    [[ -n "$HELM_DEBUG" ]] && echo "[MOCK:HELM] $*" >&2
}

helm_check_error() {
    case "${HELM_CONFIG[error_mode]}" in
        "connection_failed")
            echo "Error: Kubernetes cluster unreachable" >&2
            return 1
            ;;
        "permission_denied")
            echo "Error: configmaps is forbidden: User \"system:serviceaccount:default:default\" cannot list resource" >&2
            return 1
            ;;
        "chart_not_found")
            echo "Error: failed to download chart" >&2
            return 1
            ;;
        "release_exists")
            echo "Error: release already exists" >&2
            return 1
            ;;
        "timeout")
            echo "Error: timed out waiting for the condition" >&2
            return 1
            ;;
    esac
    return 0
}

helm_timestamp() {
    date '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo '2023-01-01 00:00:00'
}

helm_generate_revision() {
    local release="$1"
    local history="${HELM_HISTORY[$release]:-}"
    local last_rev=0
    
    if [[ -n "$history" ]]; then
        # Get last revision number
        local last_entry="${history##*|}"
        last_rev="${last_entry%%:*}"
    fi
    
    echo $((last_rev + 1))
}

# === Main Helm Command ===
helm() {
    helm_debug "helm called with: $*"
    
    if ! helm_check_error; then
        return $?
    fi
    
    if [[ $# -eq 0 ]]; then
        echo "The Kubernetes package manager"
        echo ""
        echo "Common actions for Helm:"
        echo ""
        echo "- helm repo add:       add a chart repository"
        echo "- helm repo update:    update information of available charts locally"
        echo "- helm search repo:    search for a keyword in charts"
        echo "- helm install:        install a chart"
        echo "- helm upgrade:        upgrade a release"
        echo "- helm list:           list releases"
        echo "- helm uninstall:      uninstall a release"
        return 0
    fi
    
    local command="$1"
    shift
    
    case "$command" in
        version)
            helm_cmd_version "$@"
            ;;
        repo)
            helm_cmd_repo "$@"
            ;;
        search)
            helm_cmd_search "$@"
            ;;
        install)
            helm_cmd_install "$@"
            ;;
        upgrade)
            helm_cmd_upgrade "$@"
            ;;
        uninstall|delete)
            helm_cmd_uninstall "$@"
            ;;
        list|ls)
            helm_cmd_list "$@"
            ;;
        status)
            helm_cmd_status "$@"
            ;;
        rollback)
            helm_cmd_rollback "$@"
            ;;
        get)
            helm_cmd_get "$@"
            ;;
        history)
            helm_cmd_history "$@"
            ;;
        show)
            helm_cmd_show "$@"
            ;;
        pull)
            helm_cmd_pull "$@"
            ;;
        *)
            echo "Error: unknown command \"$command\" for \"helm\"" >&2
            return 1
            ;;
    esac
}

# === Command Implementations ===

helm_cmd_version() {
    echo "version.BuildInfo{Version:\"${HELM_CONFIG[version]}\", GitCommit:\"mock\", GitTreeState:\"clean\", GoVersion:\"go1.20.4\"}"
}

helm_cmd_repo() {
    if [[ $# -eq 0 ]]; then
        echo "Error: subcommand required" >&2
        return 1
    fi
    
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        add)
            helm_repo_add "$@"
            ;;
        remove|rm)
            helm_repo_remove "$@"
            ;;
        update)
            helm_repo_update "$@"
            ;;
        list|ls)
            helm_repo_list "$@"
            ;;
        *)
            echo "Error: unknown subcommand \"$subcommand\"" >&2
            return 1
            ;;
    esac
}

helm_repo_add() {
    local name="" url=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force-update|--insecure-skip-tls-verify)
                shift
                ;;
            *)
                if [[ -z "$name" ]]; then
                    name="$1"
                elif [[ -z "$url" ]]; then
                    url="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]] || [[ -z "$url" ]]; then
        echo "Error: NAME and URL required" >&2
        return 1
    fi
    
    HELM_REPOS[$name]="$url|active"
    helm_debug "Added repo: $name -> $url"
    echo "\"$name\" has been added to your repositories"
}

helm_repo_remove() {
    local name="$1"
    
    if [[ -z "$name" ]]; then
        echo "Error: NAME required" >&2
        return 1
    fi
    
    if [[ -z "${HELM_REPOS[$name]}" ]]; then
        echo "Error: no repo named \"$name\" found" >&2
        return 1
    fi
    
    unset HELM_REPOS[$name]
    helm_debug "Removed repo: $name"
    echo "\"$name\" has been removed from your repositories"
}

helm_repo_update() {
    if [[ ${#HELM_REPOS[@]} -eq 0 ]]; then
        echo "Error: no repositories found. You must add one before updating" >&2
        return 1
    fi
    
    echo "Hang tight while we grab the latest from your chart repositories..."
    for repo in "${!HELM_REPOS[@]}"; do
        echo "...Successfully got an update from the \"$repo\" chart repository"
    done
    echo "Update Complete. ⎈Happy Helming!⎈"
}

helm_repo_list() {
    if [[ ${#HELM_REPOS[@]} -eq 0 ]]; then
        echo "Error: no repositories to show" >&2
        return 1
    fi
    
    printf "%-20s %s\n" "NAME" "URL"
    for repo in "${!HELM_REPOS[@]}"; do
        local repo_data="${HELM_REPOS[$repo]}"
        local url="${repo_data%%|*}"
        printf "%-20s %s\n" "$repo" "$url"
    done
}

helm_cmd_search() {
    local subcommand="${1:-repo}"
    shift
    
    if [[ "$subcommand" != "repo" ]]; then
        echo "Error: unknown subcommand \"$subcommand\"" >&2
        return 1
    fi
    
    local keyword="${1:-}"
    
    printf "%-40s %-15s %s\n" "NAME" "CHART VERSION" "APP VERSION"
    
    # Return mock search results
    if [[ -z "$keyword" ]] || [[ "nginx" =~ $keyword ]]; then
        printf "%-40s %-15s %s\n" "bitnami/nginx" "15.1.2" "1.25.1"
    fi
    if [[ -z "$keyword" ]] || [[ "postgresql" =~ $keyword ]]; then
        printf "%-40s %-15s %s\n" "bitnami/postgresql" "12.6.5" "15.3.0"
    fi
    if [[ -z "$keyword" ]] || [[ "redis" =~ $keyword ]]; then
        printf "%-40s %-15s %s\n" "bitnami/redis" "17.11.6" "7.0.12"
    fi
}

helm_cmd_install() {
    local release="" chart="" namespace="${HELM_CONFIG[namespace]}" values=""
    local create_namespace=false wait=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --namespace|-n)
                namespace="$2"
                shift 2
                ;;
            --create-namespace)
                create_namespace=true
                shift
                ;;
            --wait)
                wait=true
                shift
                ;;
            --values|-f)
                values="$2"
                shift 2
                ;;
            --timeout|--set|--set-string)
                shift 2
                ;;
            --debug|--dry-run|--atomic)
                shift
                ;;
            *)
                if [[ -z "$release" ]]; then
                    release="$1"
                elif [[ -z "$chart" ]]; then
                    chart="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$release" ]] || [[ -z "$chart" ]]; then
        echo "Error: RELEASE and CHART required" >&2
        return 1
    fi
    
    if [[ -n "${HELM_RELEASES[$release]}" ]]; then
        echo "Error: release $release already exists" >&2
        return 1
    fi
    
    local revision=1
    local timestamp=$(helm_timestamp)
    
    HELM_RELEASES[$release]="$namespace|$chart|deployed|$revision|$values"
    HELM_HISTORY[$release]="$revision:deployed:$timestamp"
    
    helm_debug "Installed release: $release"
    
    echo "NAME: $release"
    echo "LAST DEPLOYED: $timestamp"
    echo "NAMESPACE: $namespace"
    echo "STATUS: deployed"
    echo "REVISION: $revision"
    echo "TEST SUITE: None"
}

helm_cmd_upgrade() {
    local release="" chart="" namespace="" values=""
    local install=false wait=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --namespace|-n)
                namespace="$2"
                shift 2
                ;;
            --install)
                install=true
                shift
                ;;
            --wait)
                wait=true
                shift
                ;;
            --values|-f)
                values="$2"
                shift 2
                ;;
            --timeout|--set|--set-string)
                shift 2
                ;;
            --debug|--dry-run|--atomic|--force|--recreate-pods)
                shift
                ;;
            *)
                if [[ -z "$release" ]]; then
                    release="$1"
                elif [[ -z "$chart" ]]; then
                    chart="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$release" ]] || [[ -z "$chart" ]]; then
        echo "Error: RELEASE and CHART required" >&2
        return 1
    fi
    
    if [[ -z "${HELM_RELEASES[$release]}" ]]; then
        if [[ "$install" == "true" ]]; then
            helm_cmd_install "$release" "$chart" ${namespace:+--namespace "$namespace"} ${values:+--values "$values"}
            return $?
        else
            echo "Error: release: not found" >&2
            return 1
        fi
    fi
    
    local release_data="${HELM_RELEASES[$release]}"
    IFS='|' read -r old_namespace old_chart old_status old_revision old_values <<< "$release_data"
    
    local new_namespace="${namespace:-$old_namespace}"
    local new_revision=$(helm_generate_revision "$release")
    local timestamp=$(helm_timestamp)
    
    HELM_RELEASES[$release]="$new_namespace|$chart|deployed|$new_revision|$values"
    HELM_HISTORY[$release]="${HELM_HISTORY[$release]}|$new_revision:upgraded:$timestamp"
    
    helm_debug "Upgraded release: $release to revision $new_revision"
    
    echo "Release \"$release\" has been upgraded. Happy Helming!"
    echo "NAME: $release"
    echo "LAST DEPLOYED: $timestamp"
    echo "NAMESPACE: $new_namespace"
    echo "STATUS: deployed"
    echo "REVISION: $new_revision"
}

helm_cmd_uninstall() {
    local release="$1"
    
    if [[ -z "$release" ]]; then
        echo "Error: RELEASE required" >&2
        return 1
    fi
    
    if [[ -z "${HELM_RELEASES[$release]}" ]]; then
        echo "Error: uninstall: Release not loaded: $release" >&2
        return 1
    fi
    
    unset HELM_RELEASES[$release]
    unset HELM_HISTORY[$release]
    
    helm_debug "Uninstalled release: $release"
    echo "release \"$release\" uninstalled"
}

helm_cmd_list() {
    local namespace="" all_namespaces=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --namespace|-n)
                namespace="$2"
                shift 2
                ;;
            --all-namespaces|-A)
                all_namespaces=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    printf "%-20s %-15s %-10s %-10s %s\n" "NAME" "NAMESPACE" "REVISION" "STATUS" "CHART"
    
    for release in "${!HELM_RELEASES[@]}"; do
        local release_data="${HELM_RELEASES[$release]}"
        IFS='|' read -r rel_namespace chart status revision values <<< "$release_data"
        
        if [[ "$all_namespaces" == "true" ]] || [[ -z "$namespace" ]] || [[ "$rel_namespace" == "$namespace" ]]; then
            printf "%-20s %-15s %-10s %-10s %s\n" "$release" "$rel_namespace" "$revision" "$status" "$chart"
        fi
    done
}

helm_cmd_status() {
    local release="$1"
    
    if [[ -z "$release" ]]; then
        echo "Error: RELEASE required" >&2
        return 1
    fi
    
    local release_data="${HELM_RELEASES[$release]}"
    if [[ -z "$release_data" ]]; then
        echo "Error: release: not found" >&2
        return 1
    fi
    
    IFS='|' read -r namespace chart status revision values <<< "$release_data"
    
    echo "NAME: $release"
    echo "LAST DEPLOYED: $(helm_timestamp)"
    echo "NAMESPACE: $namespace"
    echo "STATUS: $status"
    echo "REVISION: $revision"
    echo "TEST SUITE: None"
}

helm_cmd_rollback() {
    local release="$1"
    local target_revision="${2:-0}"
    
    if [[ -z "$release" ]]; then
        echo "Error: RELEASE required" >&2
        return 1
    fi
    
    if [[ -z "${HELM_RELEASES[$release]}" ]]; then
        echo "Error: release: not found" >&2
        return 1
    fi
    
    local release_data="${HELM_RELEASES[$release]}"
    IFS='|' read -r namespace chart status revision values <<< "$release_data"
    
    local new_revision=$(helm_generate_revision "$release")
    local timestamp=$(helm_timestamp)
    
    HELM_RELEASES[$release]="$namespace|$chart|deployed|$new_revision|$values"
    HELM_HISTORY[$release]="${HELM_HISTORY[$release]}|$new_revision:rollback:$timestamp"
    
    helm_debug "Rolled back release: $release to revision $new_revision"
    echo "Rollback was a success! Happy Helming!"
}

helm_cmd_history() {
    local release="$1"
    
    if [[ -z "$release" ]]; then
        echo "Error: RELEASE required" >&2
        return 1
    fi
    
    local history="${HELM_HISTORY[$release]}"
    if [[ -z "$history" ]]; then
        echo "Error: release: not found" >&2
        return 1
    fi
    
    printf "%-10s %-25s %-15s %s\n" "REVISION" "UPDATED" "STATUS" "CHART"
    
    IFS='|' read -ra entries <<< "$history"
    for entry in "${entries[@]}"; do
        IFS=':' read -r rev status timestamp <<< "$entry"
        printf "%-10s %-25s %-15s %s\n" "$rev" "$timestamp" "$status" "mock-chart-1.0.0"
    done
}

helm_cmd_show() {
    local subcommand="${1:-all}"
    local chart="${2:-}"
    
    if [[ -z "$chart" ]]; then
        echo "Error: CHART required" >&2
        return 1
    fi
    
    case "$subcommand" in
        all|chart)
            echo "apiVersion: v2"
            echo "name: ${chart##*/}"
            echo "version: 1.0.0"
            echo "description: A Helm chart for ${chart##*/}"
            echo "type: application"
            ;;
        values)
            echo "replicaCount: 1"
            echo "image:"
            echo "  repository: nginx"
            echo "  tag: \"1.16.0\""
            echo "  pullPolicy: IfNotPresent"
            ;;
        readme)
            echo "# ${chart##*/}"
            echo "This is a Helm chart for ${chart##*/}"
            ;;
        *)
            echo "Error: unknown show subcommand \"$subcommand\"" >&2
            return 1
            ;;
    esac
}

helm_cmd_pull() {
    local chart="$1"
    
    if [[ -z "$chart" ]]; then
        echo "Error: CHART required" >&2
        return 1
    fi
    
    helm_debug "Pulled chart: $chart"
    echo "Pulled: $chart"
}

helm_cmd_get() {
    local subcommand="${1:-values}"
    local release="${2:-}"
    
    if [[ -z "$release" ]]; then
        echo "Error: RELEASE required" >&2
        return 1
    fi
    
    local release_data="${HELM_RELEASES[$release]}"
    if [[ -z "$release_data" ]]; then
        echo "Error: release: not found" >&2
        return 1
    fi
    
    case "$subcommand" in
        values)
            echo "# Default values for $release"
            echo "replicaCount: 1"
            ;;
        manifest)
            echo "---"
            echo "# Source: $release/templates/deployment.yaml"
            echo "apiVersion: apps/v1"
            echo "kind: Deployment"
            echo "metadata:"
            echo "  name: $release"
            ;;
        notes)
            echo "NOTES:"
            echo "Thank you for installing $release."
            ;;
        *)
            echo "Error: unknown get subcommand \"$subcommand\"" >&2
            return 1
            ;;
    esac
}

# === State Management ===
helm_mock_reset() {
    helm_debug "Resetting mock state"
    
    HELM_RELEASES=()
    HELM_REPOS=()
    HELM_CHARTS=()
    HELM_HISTORY=()
    HELM_CONFIG[error_mode]=""
    
    # Initialize defaults
    helm_mock_init_defaults
}

helm_mock_init_defaults() {
    # Default repositories
    HELM_REPOS["bitnami"]="https://charts.bitnami.com/bitnami|active"
    HELM_REPOS["stable"]="https://charts.helm.sh/stable|active"
    
    # Default charts
    HELM_CHARTS["nginx"]="15.1.2|1.25.1|NGINX Open Source"
    HELM_CHARTS["postgresql"]="12.6.5|15.3.0|PostgreSQL database"
    HELM_CHARTS["redis"]="17.11.6|7.0.12|Redis in-memory data store"
}

helm_mock_set_error() {
    HELM_CONFIG[error_mode]="$1"
    helm_debug "Set error mode: $1"
}

helm_mock_dump_state() {
    echo "=== Helm Mock State ==="
    echo "Releases: ${#HELM_RELEASES[@]}"
    for release in "${!HELM_RELEASES[@]}"; do
        echo "  $release: ${HELM_RELEASES[$release]}"
    done
    echo "Repositories: ${#HELM_REPOS[@]}"
    for repo in "${!HELM_REPOS[@]}"; do
        echo "  $repo: ${HELM_REPOS[$repo]}"
    done
    echo "Error Mode: ${HELM_CONFIG[error_mode]:-none}"
    echo "=================="
}

# === Convention-based Test Functions ===
test_helm_connection() {
    helm_debug "Testing connection..."
    
    local result
    result=$(helm version 2>&1)
    
    if [[ "$result" =~ "Version" ]]; then
        helm_debug "Connection test passed"
        return 0
    else
        helm_debug "Connection test failed"
        return 1
    fi
}

test_helm_health() {
    helm_debug "Testing health..."
    
    test_helm_connection || return 1
    
    helm repo add test-repo https://example.com/charts >/dev/null 2>&1 || return 1
    helm repo list >/dev/null 2>&1 || return 1
    helm repo remove test-repo >/dev/null 2>&1 || return 1
    
    helm_debug "Health test passed"
    return 0
}

test_helm_basic() {
    helm_debug "Testing basic operations..."
    
    helm install test-release bitnami/nginx >/dev/null 2>&1 || return 1
    helm list | grep -q "test-release" || return 1
    helm upgrade test-release bitnami/nginx >/dev/null 2>&1 || return 1
    helm uninstall test-release >/dev/null 2>&1 || return 1
    
    helm_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f helm
export -f test_helm_connection test_helm_health test_helm_basic
export -f helm_mock_reset helm_mock_set_error helm_mock_dump_state
export -f helm_debug helm_check_error

# Initialize with defaults
helm_mock_reset
helm_debug "Helm Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
