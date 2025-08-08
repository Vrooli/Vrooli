#!/usr/bin/env bash
# Helm System Mocks for Bats tests.
# Provides comprehensive Helm command mocking with a clear mock::helm::* namespace.

# Prevent duplicate loading
if [[ "${HELM_MOCKS_LOADED:-}" == "true" ]]; then
  return 0
fi
export HELM_MOCKS_LOADED="true"

# ----------------------------
# Global mock state & options
# ----------------------------
# Modes: normal, offline, error
export HELM_MOCK_MODE="${HELM_MOCK_MODE:-normal}"

# In-memory state
declare -A MOCK_HELM_RELEASES=()      # release -> "namespace|chart|status|revision"
declare -A MOCK_HELM_REPOS=()         # name -> "url"
declare -A MOCK_HELM_CHARTS=()        # chart -> "version|app_version"
declare -A MOCK_HELM_ERRORS=()        # command -> error_type
declare -A MOCK_HELM_VALUES=()        # release -> "key1=value1,key2=value2"
declare -A MOCK_HELM_HISTORY=()       # release -> "revision1|revision2|..."
declare -A MOCK_HELM_MANIFESTS=()     # release -> "manifest content"

# File-based state persistence for subshell access (BATS compatibility)
export HELM_MOCK_STATE_FILE="${MOCK_LOG_DIR:-/tmp}/helm_mock_state.$$"

# Initialize state file
_helm_mock_init_state_file() {
  if [[ -n "${HELM_MOCK_STATE_FILE}" ]]; then
    {
      echo "declare -A MOCK_HELM_RELEASES=()"
      echo "declare -A MOCK_HELM_REPOS=()"
      echo "declare -A MOCK_HELM_CHARTS=()"
      echo "declare -A MOCK_HELM_VALUES=()"
      echo "declare -A MOCK_HELM_HISTORY=()"
      echo "declare -A MOCK_HELM_MANIFESTS=()"
    } > "${HELM_MOCK_STATE_FILE}"
  fi
}

# Save state to file
_helm_mock_save_state() {
  if [[ -n "${HELM_MOCK_STATE_FILE}" ]]; then
    {
      echo "declare -A MOCK_HELM_RELEASES=("
      for key in "${!MOCK_HELM_RELEASES[@]}"; do
        echo "  [\"$key\"]=\"${MOCK_HELM_RELEASES[$key]}\""
      done
      echo ")"
      
      echo "declare -A MOCK_HELM_REPOS=("
      for key in "${!MOCK_HELM_REPOS[@]}"; do
        echo "  [\"$key\"]=\"${MOCK_HELM_REPOS[$key]}\""
      done
      echo ")"
      
      echo "declare -A MOCK_HELM_CHARTS=("
      for key in "${!MOCK_HELM_CHARTS[@]}"; do
        echo "  [\"$key\"]=\"${MOCK_HELM_CHARTS[$key]}\""
      done
      echo ")"
      
      echo "declare -A MOCK_HELM_VALUES=("
      for key in "${!MOCK_HELM_VALUES[@]}"; do
        echo "  [\"$key\"]=\"${MOCK_HELM_VALUES[$key]}\""
      done
      echo ")"
      
      echo "declare -A MOCK_HELM_HISTORY=("
      for key in "${!MOCK_HELM_HISTORY[@]}"; do
        echo "  [\"$key\"]=\"${MOCK_HELM_HISTORY[$key]}\""
      done
      echo ")"
      
      echo "declare -A MOCK_HELM_MANIFESTS=("
      for key in "${!MOCK_HELM_MANIFESTS[@]}"; do
        echo "  [\"$key\"]=\"${MOCK_HELM_MANIFESTS[$key]}\""
      done
      echo ")"
    } > "${HELM_MOCK_STATE_FILE}"
  fi
}

# Load state from file
_helm_mock_load_state() {
  if [[ -f "${HELM_MOCK_STATE_FILE}" ]]; then
    # shellcheck disable=SC1090
    source "${HELM_MOCK_STATE_FILE}"
  fi
}

# Initialize mock state
_helm_mock_init_state_file

# ----------------------------
# Mock Helm functions
# ----------------------------

# Mock helm command
helm() {
  _helm_mock_load_state
  
  local subcommand="${1:-}"
  shift || true
  
  case "$subcommand" in
    version)
      _helm_mock_version "$@"
      ;;
    repo)
      _helm_mock_repo "$@"
      ;;
    search)
      _helm_mock_search "$@"
      ;;
    pull)
      _helm_mock_pull "$@"
      ;;
    show)
      _helm_mock_show "$@"
      ;;
    install)
      _helm_mock_install "$@"
      ;;
    upgrade)
      _helm_mock_upgrade "$@"
      ;;
    uninstall)
      _helm_mock_uninstall "$@"
      ;;
    rollback)
      _helm_mock_rollback "$@"
      ;;
    list)
      _helm_mock_list "$@"
      ;;
    status)
      _helm_mock_status "$@"
      ;;
    get)
      _helm_mock_get "$@"
      ;;
    history)
      _helm_mock_history "$@"
      ;;
    test)
      _helm_mock_test "$@"
      ;;
    lint)
      _helm_mock_lint "$@"
      ;;
    template)
      _helm_mock_template "$@"
      ;;
    package)
      _helm_mock_package "$@"
      ;;
    create)
      _helm_mock_create "$@"
      ;;
    *)
      echo "Error: Unknown helm subcommand: $subcommand" >&2
      return 1
      ;;
  esac
}

# Mock helm version
_helm_mock_version() {
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: helm not found" >&2
    return 1
  fi
  
  if [[ "$1" == "--short" ]]; then
    echo "v3.12.0+g1234567"
  else
    echo "version.BuildInfo{Version:\"v3.12.0\", GitCommit:\"1234567890abcdef\", GitTreeState:\"clean\", GoVersion:\"go1.20.5\"}"
  fi
  return 0
}

# Mock helm repo operations
_helm_mock_repo() {
  local action="${1:-}"
  shift || true
  
  case "$action" in
    add)
      local name="${1:-}"
      local url="${2:-}"
      if [[ -z "$name" ]] || [[ -z "$url" ]]; then
        echo "Error: repo name and URL required" >&2
        return 1
      fi
      if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
        echo "Error: Failed to add repository $name" >&2
        return 1
      fi
      MOCK_HELM_REPOS["$name"]="$url"
      _helm_mock_save_state
      echo "\"$name\" has been added to your repositories"
      return 0
      ;;
      
    update)
      local repo="${1:-}"
      if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
        echo "Error: Failed to update repositories" >&2
        return 1
      fi
      if [[ -n "$repo" ]]; then
        echo "Hang tight while we grab the latest from your chart repositories..."
        echo "...Successfully got an update from the \"$repo\" chart repository"
      else
        echo "Hang tight while we grab the latest from your chart repositories..."
        for repo_name in "${!MOCK_HELM_REPOS[@]}"; do
          echo "...Successfully got an update from the \"$repo_name\" chart repository"
        done
      fi
      echo "Update Complete. ⎈Happy Helming!⎈"
      return 0
      ;;
      
    list)
      if [[ "${#MOCK_HELM_REPOS[@]}" -eq 0 ]]; then
        return 0
      fi
      echo "NAME	URL"
      for name in "${!MOCK_HELM_REPOS[@]}"; do
        echo "$name	${MOCK_HELM_REPOS[$name]}"
      done
      return 0
      ;;
      
    remove)
      local name="${1:-}"
      if [[ -z "$name" ]]; then
        echo "Error: repo name required" >&2
        return 1
      fi
      if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
        echo "Error: Failed to remove repository $name" >&2
        return 1
      fi
      unset "MOCK_HELM_REPOS[$name]"
      _helm_mock_save_state
      echo "\"$name\" has been removed from your repositories"
      return 0
      ;;
      
    *)
      echo "Error: Unknown repo action: $action" >&2
      return 1
      ;;
  esac
}

# Mock helm search
_helm_mock_search() {
  local type="${1:-}"
  shift || true
  local keyword="${1:-}"
  
  if [[ "$type" != "repo" ]]; then
    echo "Error: Only 'helm search repo' is supported" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: Failed to search repositories" >&2
    return 1
  fi
  
  echo "NAME                    	CHART VERSION	APP VERSION	DESCRIPTION"
  if [[ -n "$keyword" ]]; then
    echo "bitnami/$keyword       	1.0.0        	2.0.0      	A mock chart for $keyword"
    echo "stable/$keyword        	0.9.0        	1.9.0      	Another mock chart for $keyword"
  else
    echo "bitnami/nginx          	15.0.0       	1.25.0     	NGINX Open Source"
    echo "bitnami/postgresql     	13.0.0       	16.0.0     	PostgreSQL database"
  fi
  return 0
}

# Mock helm pull
_helm_mock_pull() {
  local chart="${1:-}"
  
  if [[ -z "$chart" ]]; then
    echo "Error: chart name required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: Failed to pull chart $chart" >&2
    return 1
  fi
  
  echo "Pulled: $chart"
  return 0
}

# Mock helm show
_helm_mock_show() {
  local section="${1:-}"
  local chart="${2:-}"
  
  if [[ -z "$chart" ]]; then
    echo "Error: chart name required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: Failed to show $section for chart $chart" >&2
    return 1
  fi
  
  case "$section" in
    all|chart)
      echo "apiVersion: v2"
      echo "name: $chart"
      echo "version: 1.0.0"
      echo "description: A mock Helm chart for $chart"
      ;;
    values)
      echo "replicaCount: 1"
      echo "image:"
      echo "  repository: nginx"
      echo "  tag: latest"
      ;;
    readme)
      echo "# $chart"
      echo "This is a mock README for $chart"
      ;;
    *)
      echo "Error: Unknown section: $section" >&2
      return 1
      ;;
  esac
  return 0
}

# Mock helm install
_helm_mock_install() {
  local release=""
  local chart=""
  local namespace="default"
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace)
        namespace="$2"
        shift 2
        ;;
      --create-namespace|--wait|--debug)
        shift
        ;;
      --timeout|--values)
        shift 2
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
    echo "Error: release name and chart required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: Failed to install release $release" >&2
    return 1
  fi
  
  MOCK_HELM_RELEASES["$release"]="$namespace|$chart|deployed|1"
  MOCK_HELM_HISTORY["$release"]="1"
  _helm_mock_save_state
  
  echo "NAME: $release"
  echo "LAST DEPLOYED: $(date)"
  echo "NAMESPACE: $namespace"
  echo "STATUS: deployed"
  echo "REVISION: 1"
  echo "TEST SUITE: None"
  return 0
}

# Mock helm upgrade
_helm_mock_upgrade() {
  local release=""
  local chart=""
  local namespace="default"
  local install_flag=false
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --install)
        install_flag=true
        shift
        ;;
      --namespace)
        namespace="$2"
        shift 2
        ;;
      --create-namespace|--wait|--debug)
        shift
        ;;
      --timeout|--values|--history-max)
        shift 2
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
    echo "Error: release name and chart required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: Failed to upgrade release $release" >&2
    return 1
  fi
  
  # Check if release exists
  if [[ ! -v "MOCK_HELM_RELEASES[$release]" ]]; then
    if [[ "$install_flag" == true ]]; then
      # Install if --install flag is present
      MOCK_HELM_RELEASES["$release"]="$namespace|$chart|deployed|1"
      MOCK_HELM_HISTORY["$release"]="1"
      echo "Release \"$release\" does not exist. Installing it now."
    else
      echo "Error: release $release not found" >&2
      return 1
    fi
  else
    # Upgrade existing release
    local current_info="${MOCK_HELM_RELEASES[$release]}"
    local current_revision
    current_revision=$(echo "$current_info" | cut -d'|' -f4)
    local new_revision=$((current_revision + 1))
    MOCK_HELM_RELEASES["$release"]="$namespace|$chart|deployed|$new_revision"
    
    # Update history
    if [[ -v "MOCK_HELM_HISTORY[$release]" ]]; then
      MOCK_HELM_HISTORY["$release"]="${MOCK_HELM_HISTORY[$release]}|$new_revision"
    else
      MOCK_HELM_HISTORY["$release"]="$new_revision"
    fi
  fi
  
  _helm_mock_save_state
  
  local revision="${MOCK_HELM_RELEASES[$release]##*|}"
  echo "Release \"$release\" has been upgraded. Happy Helming!"
  echo "NAME: $release"
  echo "LAST DEPLOYED: $(date)"
  echo "NAMESPACE: $namespace"
  echo "STATUS: deployed"
  echo "REVISION: $revision"
  return 0
}

# Mock helm uninstall
_helm_mock_uninstall() {
  local release=""
  local namespace="default"
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace)
        namespace="$2"
        shift 2
        ;;
      --wait|--debug)
        shift
        ;;
      --timeout)
        shift 2
        ;;
      *)
        release="$1"
        shift
        ;;
    esac
  done
  
  if [[ -z "$release" ]]; then
    echo "Error: release name required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: Failed to uninstall release $release" >&2
    return 1
  fi
  
  if [[ ! -v "MOCK_HELM_RELEASES[$release]" ]]; then
    echo "Error: release: not found" >&2
    return 1
  fi
  
  unset "MOCK_HELM_RELEASES[$release]"
  unset "MOCK_HELM_HISTORY[$release]"
  unset "MOCK_HELM_VALUES[$release]"
  unset "MOCK_HELM_MANIFESTS[$release]"
  _helm_mock_save_state
  
  echo "release \"$release\" uninstalled"
  return 0
}

# Mock helm rollback
_helm_mock_rollback() {
  local release=""
  local revision="0"
  local namespace="default"
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace)
        namespace="$2"
        shift 2
        ;;
      --wait|--debug)
        shift
        ;;
      --timeout)
        shift 2
        ;;
      *)
        if [[ -z "$release" ]]; then
          release="$1"
        else
          revision="$1"
        fi
        shift
        ;;
    esac
  done
  
  if [[ -z "$release" ]]; then
    echo "Error: release name required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: Failed to rollback release $release" >&2
    return 1
  fi
  
  if [[ ! -v "MOCK_HELM_RELEASES[$release]" ]]; then
    echo "Error: release: not found" >&2
    return 1
  fi
  
  echo "Rollback was a success! Happy Helming!"
  return 0
}

# Mock helm list
_helm_mock_list() {
  local namespace=""
  local all_namespaces=false
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace)
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
  
  echo "NAME	NAMESPACE	REVISION	UPDATED	STATUS	CHART	APP VERSION"
  
  for release in "${!MOCK_HELM_RELEASES[@]}"; do
    local info="${MOCK_HELM_RELEASES[$release]}"
    local rel_namespace
    rel_namespace=$(echo "$info" | cut -d'|' -f1)
    local chart
    chart=$(echo "$info" | cut -d'|' -f2)
    local status
    status=$(echo "$info" | cut -d'|' -f3)
    local revision
    revision=$(echo "$info" | cut -d'|' -f4)
    
    if [[ "$all_namespaces" == true ]] || [[ -z "$namespace" ]] || [[ "$rel_namespace" == "$namespace" ]]; then
      echo "$release	$rel_namespace	$revision	$(date)	$status	$chart-1.0.0	1.0.0"
    fi
  done
  
  return 0
}

# Mock helm status
_helm_mock_status() {
  local release=""
  local namespace="default"
  local output=""
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace)
        namespace="$2"
        shift 2
        ;;
      --output|-o)
        output="$2"
        shift 2
        ;;
      *)
        release="$1"
        shift
        ;;
    esac
  done
  
  if [[ -z "$release" ]]; then
    echo "Error: release name required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: Failed to get status for release $release" >&2
    return 1
  fi
  
  if [[ ! -v "MOCK_HELM_RELEASES[$release]" ]]; then
    echo "Error: release: not found" >&2
    return 1
  fi
  
  local info="${MOCK_HELM_RELEASES[$release]}"
  local rel_namespace
  rel_namespace=$(echo "$info" | cut -d'|' -f1)
  local chart
  chart=$(echo "$info" | cut -d'|' -f2)
  local status
  status=$(echo "$info" | cut -d'|' -f3)
  local revision
  revision=$(echo "$info" | cut -d'|' -f4)
  
  if [[ "$output" == "json" ]]; then
    echo "{\"name\":\"$release\",\"namespace\":\"$rel_namespace\",\"revision\":$revision,\"status\":\"$status\"}"
  elif [[ "$output" == "yaml" ]]; then
    echo "name: $release"
    echo "namespace: $rel_namespace"
    echo "revision: $revision"
    echo "status: $status"
  else
    echo "NAME: $release"
    echo "LAST DEPLOYED: $(date)"
    echo "NAMESPACE: $rel_namespace"
    echo "STATUS: $status"
    echo "REVISION: $revision"
    echo "TEST SUITE: None"
  fi
  
  return 0
}

# Mock helm get
_helm_mock_get() {
  local action="${1:-}"
  shift || true
  
  case "$action" in
    values)
      _helm_mock_get_values "$@"
      ;;
    manifest)
      _helm_mock_get_manifest "$@"
      ;;
    *)
      echo "Error: Unknown get action: $action" >&2
      return 1
      ;;
  esac
}

# Mock helm get values
_helm_mock_get_values() {
  local release=""
  local namespace="default"
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace)
        namespace="$2"
        shift 2
        ;;
      --all)
        shift
        ;;
      *)
        release="$1"
        shift
        ;;
    esac
  done
  
  if [[ -z "$release" ]]; then
    echo "Error: release name required" >&2
    return 1
  fi
  
  if [[ ! -v "MOCK_HELM_RELEASES[$release]" ]]; then
    echo "Error: release: not found" >&2
    return 1
  fi
  
  if [[ -v "MOCK_HELM_VALUES[$release]" ]]; then
    echo "${MOCK_HELM_VALUES[$release]//,/$'\n'}"
  else
    echo "USER-SUPPLIED VALUES:"
    echo "replicaCount: 2"
  fi
  
  return 0
}

# Mock helm get manifest
_helm_mock_get_manifest() {
  local release=""
  local namespace="default"
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace)
        namespace="$2"
        shift 2
        ;;
      --revision)
        shift 2
        ;;
      *)
        release="$1"
        shift
        ;;
    esac
  done
  
  if [[ -z "$release" ]]; then
    echo "Error: release name required" >&2
    return 1
  fi
  
  if [[ ! -v "MOCK_HELM_RELEASES[$release]" ]]; then
    echo "Error: release: not found" >&2
    return 1
  fi
  
  if [[ -v "MOCK_HELM_MANIFESTS[$release]" ]]; then
    echo "${MOCK_HELM_MANIFESTS[$release]}"
  else
    echo "---"
    echo "# Source: $release/templates/deployment.yaml"
    echo "apiVersion: apps/v1"
    echo "kind: Deployment"
    echo "metadata:"
    echo "  name: $release"
    echo "  namespace: $namespace"
  fi
  
  return 0
}

# Mock helm history
_helm_mock_history() {
  local release=""
  local namespace="default"
  local max="256"
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace)
        namespace="$2"
        shift 2
        ;;
      --max)
        max="$2"
        shift 2
        ;;
      *)
        release="$1"
        shift
        ;;
    esac
  done
  
  if [[ -z "$release" ]]; then
    echo "Error: release name required" >&2
    return 1
  fi
  
  if [[ ! -v "MOCK_HELM_RELEASES[$release]" ]]; then
    echo "Error: release: not found" >&2
    return 1
  fi
  
  echo "REVISION	UPDATED                 	STATUS    	CHART           	APP VERSION	DESCRIPTION"
  
  if [[ -v "MOCK_HELM_HISTORY[$release]" ]]; then
    local revisions="${MOCK_HELM_HISTORY[$release]}"
    IFS='|' read -ra REV_ARRAY <<< "$revisions"
    local count=0
    for rev in "${REV_ARRAY[@]}"; do
      if [[ $count -ge $max ]]; then
        break
      fi
      echo "$rev	$(date)	deployed  	mock-chart-1.0.0	1.0.0      	Upgrade complete"
      count=$((count + 1))
    done
  else
    echo "1	$(date)	deployed  	mock-chart-1.0.0	1.0.0      	Install complete"
  fi
  
  return 0
}

# Mock helm test
_helm_mock_test() {
  local release=""
  local namespace="default"
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace)
        namespace="$2"
        shift 2
        ;;
      --timeout)
        shift 2
        ;;
      *)
        release="$1"
        shift
        ;;
    esac
  done
  
  if [[ -z "$release" ]]; then
    echo "Error: release name required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: test failed for release $release" >&2
    return 1
  fi
  
  if [[ ! -v "MOCK_HELM_RELEASES[$release]" ]]; then
    echo "Error: release: not found" >&2
    return 1
  fi
  
  echo "NAME: $release"
  echo "LAST DEPLOYED: $(date)"
  echo "NAMESPACE: $namespace"
  echo "STATUS: deployed"
  echo ""
  echo "TEST SUITE:     $release-test"
  echo "Last Started:   $(date)"
  echo "Last Completed: $(date)"
  echo "Phase:          Succeeded"
  
  return 0
}

# Mock helm lint
_helm_mock_lint() {
  local chart="${1:-}"
  
  if [[ -z "$chart" ]]; then
    echo "Error: chart path required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "[ERROR] Chart.yaml: chart name must not be empty" >&2
    echo ""
    echo "Error: 1 chart(s) linted, 1 chart(s) failed" >&2
    return 1
  fi
  
  echo "==> Linting $chart"
  echo "[INFO] Chart.yaml: icon is recommended"
  echo ""
  echo "1 chart(s) linted, 0 chart(s) failed"
  
  return 0
}

# Mock helm template
_helm_mock_template() {
  local release=""
  local chart=""
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace|--values|--output-dir)
        shift 2
        ;;
      --debug)
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
    echo "Error: release name and chart required" >&2
    return 1
  fi
  
  echo "---"
  echo "# Source: $chart/templates/deployment.yaml"
  echo "apiVersion: apps/v1"
  echo "kind: Deployment"
  echo "metadata:"
  echo "  name: $release"
  
  return 0
}

# Mock helm package
_helm_mock_package() {
  local chart="${1:-}"
  
  if [[ -z "$chart" ]]; then
    echo "Error: chart path required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: failed to package chart" >&2
    return 1
  fi
  
  echo "Successfully packaged chart and saved it to: $chart-1.0.0.tgz"
  
  return 0
}

# Mock helm create
_helm_mock_create() {
  local name="${1:-}"
  
  if [[ -z "$name" ]]; then
    echo "Error: chart name required" >&2
    return 1
  fi
  
  if [[ "${HELM_MOCK_MODE}" == "error" ]]; then
    echo "Error: failed to create chart" >&2
    return 1
  fi
  
  echo "Creating $name"
  
  return 0
}

# ----------------------------
# Mock control functions
# ----------------------------

# Set mock mode
mock::helm::set_mode() {
  export HELM_MOCK_MODE="${1:-normal}"
}

# Reset all mocks
mock::helm::reset() {
  # Reset state arrays
  unset MOCK_HELM_RELEASES
  unset MOCK_HELM_REPOS
  unset MOCK_HELM_CHARTS
  unset MOCK_HELM_VALUES
  unset MOCK_HELM_HISTORY
  unset MOCK_HELM_MANIFESTS
  unset MOCK_HELM_ERRORS
  
  # Reinitialize arrays
  declare -gA MOCK_HELM_RELEASES=()
  declare -gA MOCK_HELM_REPOS=()
  declare -gA MOCK_HELM_CHARTS=()
  declare -gA MOCK_HELM_VALUES=()
  declare -gA MOCK_HELM_HISTORY=()
  declare -gA MOCK_HELM_MANIFESTS=()
  declare -gA MOCK_HELM_ERRORS=()
  
  export HELM_MOCK_MODE="normal"
  
  # Only save state if function is available
  if declare -f _helm_mock_save_state >/dev/null 2>&1; then
    _helm_mock_save_state
  elif [[ -n "${HELM_MOCK_STATE_FILE}" ]]; then
    # Fallback: directly write empty state
    {
      echo "declare -A MOCK_HELM_RELEASES=()"
      echo "declare -A MOCK_HELM_REPOS=()"
      echo "declare -A MOCK_HELM_CHARTS=()"
      echo "declare -A MOCK_HELM_VALUES=()"
      echo "declare -A MOCK_HELM_HISTORY=()"
      echo "declare -A MOCK_HELM_MANIFESTS=()"
    } > "${HELM_MOCK_STATE_FILE}"
  fi
}

# Add a mock release
mock::helm::add_release() {
  local release="${1:?Release name required}"
  local namespace="${2:-default}"
  local chart="${3:-mock-chart}"
  local status="${4:-deployed}"
  local revision="${5:-1}"
  
  MOCK_HELM_RELEASES["$release"]="$namespace|$chart|$status|$revision"
  MOCK_HELM_HISTORY["$release"]="$revision"
  
  # Only save state if function is available
  if declare -f _helm_mock_save_state >/dev/null 2>&1; then
    _helm_mock_save_state
  fi
}

# Add a mock repository
mock::helm::add_repo() {
  local name="${1:?Repository name required}"
  local url="${2:?Repository URL required}"
  
  MOCK_HELM_REPOS["$name"]="$url"
  
  # Only save state if function is available
  if declare -f _helm_mock_save_state >/dev/null 2>&1; then
    _helm_mock_save_state
  fi
}

# Set release values
mock::helm::set_values() {
  local release="${1:?Release name required}"
  local values="${2:?Values required}"
  
  MOCK_HELM_VALUES["$release"]="$values"
  
  # Only save state if function is available
  if declare -f _helm_mock_save_state >/dev/null 2>&1; then
    _helm_mock_save_state
  fi
}

# Set release manifest
mock::helm::set_manifest() {
  local release="${1:?Release name required}"
  local manifest="${2:?Manifest required}"
  
  MOCK_HELM_MANIFESTS["$release"]="$manifest"
  
  # Only save state if function is available
  if declare -f _helm_mock_save_state >/dev/null 2>&1; then
    _helm_mock_save_state
  fi
}

# Check if release exists
mock::helm::release_exists() {
  local release="${1:?Release name required}"
  
  # Only load state if function is available
  if declare -f _helm_mock_load_state >/dev/null 2>&1; then
    _helm_mock_load_state
  fi
  
  [[ -v "MOCK_HELM_RELEASES[$release]" ]]
}

# Get release info
mock::helm::get_release_info() {
  local release="${1:?Release name required}"
  
  # Only load state if function is available
  if declare -f _helm_mock_load_state >/dev/null 2>&1; then
    _helm_mock_load_state
  fi
  
  echo "${MOCK_HELM_RELEASES[$release]:-}"
}

# Export mock functions
export -f helm
export -f mock::helm::set_mode
export -f mock::helm::reset
export -f mock::helm::add_release
export -f mock::helm::add_repo
export -f mock::helm::set_values
export -f mock::helm::set_manifest
export -f mock::helm::release_exists
export -f mock::helm::get_release_info

# Clean up on exit
trap 'rm -f "${HELM_MOCK_STATE_FILE}"' EXIT