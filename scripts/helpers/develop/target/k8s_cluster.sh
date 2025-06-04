#!/usr/bin/env bash
set -euo pipefail

DEVELOP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/log.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/env.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/docker.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/flow.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/system.sh"


# Global variable to store original Docker env to revert
ORIGINAL_DOCKER_ENV=""

develop::k8s_revert_docker_env() {
    if [[ -n "$ORIGINAL_DOCKER_ENV" ]]; then
        log::info "Reverting Docker environment..."
        eval "$ORIGINAL_DOCKER_ENV"
        # Unset specific Minikube docker-env variables if they were set
        unset DOCKER_TLS_VERIFY DOCKER_HOST DOCKER_CERT_PATH MINIKUBE_ACTIVE_DOCKERD
    fi
}

develop::k8s_ensure_minikube_running() {
    log::info "Checking Minikube status..."
    if ! minikube status &> /dev/null; then
        log::warning "Minikube does not appear to be running."
        if flow::is_yes "${YES:-no}"; then
            log::info "Attempting to start Minikube..."
            if ! minikube start; then
                log::error "Failed to start Minikube. Please start it manually or ensure it's properly configured."
                exit 1
            fi
            log::success "Minikube started successfully."
        else
            log::prompt "Minikube is not running. Attempt to start it? (y/N): " confirm_start_minikube
            if flow::is_yes "$confirm_start_minikube"; then
                log::info "Attempting to start Minikube..."
                if ! minikube start; then
                    log::error "Failed to start Minikube. Please start it manually or ensure it's properly configured."
                    exit 1
                fi
                log::success "Minikube started successfully."
            else
                log::error "Minikube is not running. Please start it to proceed with Kubernetes development."
                exit 1
            fi
        fi
    else
        log::success "Minikube is running."
    fi

    log::info "Setting kubectl context to vrooli-dev-cluster (Minikube)..."
    if ! kubectl config use-context vrooli-dev-cluster &> /dev/null; then
        log::warning "Failed to set kubectl context to 'vrooli-dev-cluster'. This might cause issues if another context is active."
        log::warning "Ensure Minikube was set up correctly and the context 'vrooli-dev-cluster' exists."
        log::info "Attempting to use default 'minikube' context as a fallback..."
        if ! kubectl config use-context minikube &> /dev/null; then
            log::warning "Fallback to 'minikube' context also failed."
        else
            log::success "Successfully set kubectl context to 'minikube' (fallback)."
        fi
    else
        log::success "Kubectl context set to 'vrooli-dev-cluster'."
    fi
}

develop::k8s_cluster_main() {
    log::header "🚀 Starting Kubernetes Development Environment Setup (Target: k8s-cluster)"

    # The main develop.sh script sources setup.sh, which for k8s-cluster target,
    # should have installed kubectl, helm, and minikube and potentially started minikube.
    # We'll re-verify minikube status here.
    develop::k8s_ensure_minikube_running

    # Trap to revert docker env on exit
    trap develop::k8s_revert_docker_env EXIT SIGINT SIGTERM

    log::info "Configuring shell to use Minikube's Docker daemon..."
    # Save current Docker environment variables to revert later
    ORIGINAL_DOCKER_ENV=$(env | grep '^DOCKER_\|^MINIKUBE_ACTIVE_DOCKERD=' || true)
    
    # Attempt to set Minikube docker-env. Suppress output on success, capture on error.
    local minikube_docker_env_cmd_output
    if minikube_docker_env_cmd_output=$(minikube -p minikube docker-env 2>&1); then
        eval "$minikube_docker_env_cmd_output"
        log::success "Shell configured to use Minikube's Docker daemon."
    else
        log::error "Failed to get Minikube Docker environment. Output:"
        log::error "$minikube_docker_env_cmd_output"
        log::error "Please ensure Minikube is running and configured correctly."
        exit 1
    fi

    log::header "🏗️ Building development Docker images..."
    # docker::build_images sources env vars like ENVIRONMENT, which should be 'dev'
    # It uses docker-compose.yml by default for dev.
    if ! docker::build_images; then
        log::error "Failed to build development Docker images."
        exit 1
    fi
    log::success "Development Docker images built successfully (into Minikube's daemon)."

    log::header "🚢 Deploying application to Minikube via Helm (from source chart)..."
    local release_name="vrooli-dev"
    local namespace="dev" # Development namespace
    local chart_source_path="${var_ROOT_DIR}/k8s/chart/"
    
    # Base values file from the chart source
    local base_values_file="${chart_source_path}values.yaml"
    # Dev-specific values file from the chart source
    local dev_values_file="${chart_source_path}values-dev.yaml"

    local helm_cmd_opts=()
    helm_cmd_opts+=("--namespace" "$namespace")
    helm_cmd_opts+=("--create-namespace")
    helm_cmd_opts+=("-f" "$base_values_file")

    if [[ -f "$dev_values_file" ]]; then
        log::info "Applying development-specific Helm values from: $dev_values_file"
        helm_cmd_opts+=("-f" "$dev_values_file")
    else
        log::warning "Development-specific Helm values file not found at: $dev_values_file. Using base values only."
    fi

    # Override image tags and pull policy for development
    # Images are in Minikube's daemon, so pullPolicy should be IfNotPresent or Never.
    # values-dev.yaml should ideally set pullPolicy: Never or IfNotPresent.
    helm_cmd_opts+=("--set" "services.ui.tag=dev")
    helm_cmd_opts+=("--set" "services.server.tag=dev")
    helm_cmd_opts+=("--set" "services.jobs.tag=dev")
    # Example: helm_cmd_opts+=("--set" "image.pullPolicy=Never") # Or ensure values-dev.yaml sets this

    helm_cmd_opts+=("--atomic") # If the upgrade fails, the upgrade is rolled back to the previous state.
    helm_cmd_opts+=("--timeout" "5m") # How long Helm should wait for K8s resources to be ready.

    log::info "Running Helm upgrade --install for release '$release_name' from chart source '$chart_source_path'..."
    if ! helm upgrade --install "$release_name" "$chart_source_path" "${helm_cmd_opts[@]}"; then
        log::error "Helm deployment to Minikube failed for release '$release_name'."
        # Attempt to get Helm status or logs for the failed release
        log::info "Attempting to get status for release '$release_name' in namespace '$namespace'..."
        helm status "$release_name" --namespace "$namespace" || true
        exit 1
    fi
    log::success "Application deployed/updated successfully to Minikube as release '$release_name' in namespace '$namespace'."

    # Check DETACHED environment variable (set by main develop.sh)
    if flow::is_yes "${DETACHED:-no}"; then
        log::success "Detached mode: Development environment started. View logs and services manually."
        log::info "To stream logs: kubectl logs -f -n $namespace -l app.kubernetes.io/instance=$release_name"
        log::info "To access UI (example): kubectl port-forward svc/$release_name-ui -n $namespace <local_port>:3000"
        log::info "To access Server API (example): kubectl port-forward svc/$release_name-server -n $namespace <local_port>:5329"
    else
        log::success "Development environment started. Streaming application logs..."
        log::info "Access services via 'kubectl port-forward'. Example for UI:"
        log::info "  kubectl port-forward svc/${release_name}-ui -n ${namespace} <your_local_port>:3000"
        log::info "Press Ctrl+C to stop log streaming and exit."
        
        # Stream logs. Adjust label selector as per your Helm chart's conventions.
        # Common Helm labels include app.kubernetes.io/instance=<release_name>
        # or app.kubernetes.io/name=<chart_name> and app.kubernetes.io/component=<service_name>
        if ! kubectl logs -f -n "$namespace" -l "app.kubernetes.io/instance=${release_name}" --tail=50 --max-log-requests=6; then
             log::warning "Log streaming exited. There might have been an issue, or no pods were found matching the selector."
             log::info "You can manually stream logs using: kubectl logs -f -n $namespace -l app.kubernetes.io/instance=$release_name"
        fi
    fi

    # Revert Docker env explicitly before exiting, though trap should also catch it.
    develop::k8s_revert_docker_env
    log::success "Kubernetes Development script finished."
}

# This script is sourced by scripts/main/develop.sh, which calls the main function.
# The main develop.sh script will parse arguments and then call:
# bash "scripts/helpers/develop/target/k8s_cluster.sh" "$@"
# So, we can assume this script itself might be called with arguments,
# but the primary arguments from CLI are already parsed and exported as env vars by the parent.
develop::k8s_cluster_main "$@"
