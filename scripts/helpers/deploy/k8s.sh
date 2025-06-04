#!/usr/bin/env bash
set -euo pipefail

DEPLOY_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${DEPLOY_DIR}/../utils/env.sh"
# shellcheck disable=SC1091
source "${DEPLOY_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${DEPLOY_DIR}/../utils/var.sh"

# Deploy application to Kubernetes using Helm
deploy::deploy_k8s() {
  local target_env="${1:-}"
  if [[ -z "$target_env" ]]; then
    log::error "Kubernetes Helm deployment: target environment (e.g., dev, staging, prod) not specified as the first argument."
    exit 1
  fi
  log::header "🚀 Starting Kubernetes Helm deployment: env=$target_env Version=$VERSION"

  # Check for Helm
  if ! command -v helm >/dev/null 2>&1; then
    log::error "Helm CLI not found; please install Helm."
    exit 1
  fi

  # Dynamically find the packaged chart .tgz file
  # Assumes only one Vrooli application chart .tgz will be in this directory for this version.
  local chart_package_dir="${var_DEST_DIR}/${VERSION}/artifacts/k8s-chart-packages"
  local packaged_chart_path
  packaged_chart_path=$(find "$chart_package_dir" -maxdepth 1 -name "*.tgz" -type f)

  if [[ -z "$packaged_chart_path" ]]; then
    log::error "No packaged Helm chart (.tgz file) found in: $chart_package_dir"
    log::error "Ensure build.sh ran correctly and artifacts.zip.gz was created and unpacked by deploy.sh."
    exit 1
  fi
  # Check if more than one .tgz file was found (could indicate an issue)
  if [[ $(echo "$packaged_chart_path" | wc -l) -ne 1 ]]; then
    log::error "Multiple .tgz files found in $chart_package_dir. Expected only one Vrooli chart package:"
    log::error "$packaged_chart_path"
    exit 1
  fi

  local chart_path="$packaged_chart_path" # Use the found .tgz file for helm commands
  local chart_name # Extract chart name for logging if needed, from the tgz filename
  chart_name=$(basename "$chart_path" "-${VERSION}.tgz")

  local release_name="vrooli-$target_env"
  local namespace="$target_env" # Use target environment as namespace

  log::info "Using Helm chart package: $chart_path"

  log::info "Release name: $release_name"
  log::info "Target namespace: $namespace (will be created if it doesn't exist)"
  # log::info "Using base values file: $base_values_file" # Removed, handled by chart package
  # if [[ -n "$env_values_file" && -f "$env_values_file" ]]; then # Removed
  #   log::info "Using environment values file: $env_values_file" # Removed
  # fi

  log::info "Linting Helm chart package: $chart_path"
  if ! helm lint "$chart_path"; then
    log::error "Helm chart linting failed. Please fix the chart issues before proceeding."
    exit 1
  fi

  # Image tags are now controlled via Helm values files (values-<env>.yaml) and no longer set via script overrides.

  # Construct helm command options
  local helm_cmd_opts=()
  helm_cmd_opts+=("--namespace" "$namespace")
  helm_cmd_opts+=("--create-namespace")

  # Add environment-specific values file if it exists
  # This path assumes deploy.sh unpacks artifacts such that helm-value-files/ is directly under $var_DEST_DIR/$VERSION/artifacts/
  local env_specific_values_path="${var_DEST_DIR}/${VERSION}/artifacts/helm-value-files/values-${target_env}.yaml"

  if [[ -f "$env_specific_values_path" ]]; then
    log::info "Applying environment-specific Helm values from: $env_specific_values_path"
    helm_cmd_opts+=("-f" "$env_specific_values_path")
  else
    log::warning "Environment-specific Helm values file not found at: $env_specific_values_path"
    # Check if the target environment is production or prod
    if [[ "$target_env" == "prod" || "$target_env" == "production" ]]; then
      log::error "CRITICAL: Production deployment for '$target_env' attempted without its specific values file ($env_specific_values_path) in the artifact. Aborting."
      log::error "Ensure 'values-$target_env.yaml' exists in 'k8s/chart/' and was correctly packaged by 'build.sh'."
      exit 1 # Using generic exit code 1 as per existing script patterns
    else
      log::info "Proceeding with default values from chart and --set overrides only for $target_env."
    fi
    # Depending on policy, you might want to exit here if env-specific values are mandatory for this environment:
  fi
  
  # No --set image tag overrides from script. Helm will use tags defined in the values file.
  
  helm_cmd_opts+=("--atomic")
  helm_cmd_opts+=("--timeout" "10m")

  log::info "Preparing to deploy Helm chart '$release_name' with the following value precedence:"
  log::info "  1. Packaged chart defaults (from ${chart_name}-${VERSION}.tgz internal values.yaml)"
  if [[ -f "$env_specific_values_path" ]]; then
    log::info "  2. Environment-specific overrides (from $env_specific_values_path)"
    log::info "  3. Command-line --set arguments (highest precedence)"
  else
    log::info "  2. Command-line --set arguments (highest precedence - no env-specific file found at $env_specific_values_path)"
  fi

  # Execute Helm command
  log::info "Running Helm upgrade --install..."
  if helm upgrade --install "$release_name" "$chart_path" \
    "${helm_cmd_opts[@]}"; then # Wait for resources to be ready, adjust timeout as needed
    log::success "✅ Kubernetes Helm deployment completed successfully for environment: $target_env"
  else
    log::error "❌ Kubernetes Helm deployment failed for environment: $target_env"
    # Attempt to get Helm status or logs for the failed release
    log::info "Attempting to get status for release '$release_name' in namespace '$namespace'..."
    helm status "$release_name" --namespace "$namespace" || true
    log::info "Attempting to get history for release '$release_name' in namespace '$namespace'..."
    helm history "$release_name" --namespace "$namespace" || true
    exit 1
  fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    deploy::deploy_k8s "$@"
fi
