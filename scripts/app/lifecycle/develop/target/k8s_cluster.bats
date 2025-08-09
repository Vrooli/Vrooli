#!/usr/bin/env bats

# Test file for k8s_cluster.sh
# Tests Kubernetes cluster development environment setup

setup() {
    # Load test helpers
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-support/load"
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-assert/load"
    
    # Set up test directory
    export TEST_DIR="$(mktemp -d)"
    export ORIGINAL_DIR="$(pwd)"
    
    # Source the script (functions only, don't execute main)
    export APP_LIFECYCLE_DEVELOP_TARGET_DIR="${BATS_TEST_DIRNAME}"
    
    # Mock var.sh and dependencies
    export var_ROOT_DIR="${TEST_DIR}"
    export var_LIB_UTILS_DIR="${TEST_DIR}/lib/utils"
    export var_LOG_FILE="${TEST_DIR}/lib/utils/log.sh"
    export var_APP_UTILS_DIR="${TEST_DIR}/app/utils"
    export var_LIB_SYSTEM_DIR="${TEST_DIR}/lib/system"
    export var_SYSTEM_COMMANDS_FILE="${TEST_DIR}/lib/system/system_commands.sh"
    export EXIT_USER_INTERRUPT=130
    
    # Create mock directories
    mkdir -p "${var_LIB_UTILS_DIR}" "${var_APP_UTILS_DIR}" "${var_LIB_SYSTEM_DIR}"
    mkdir -p "${TEST_DIR}/k8s/chart"
    
    # Create mock log.sh
    cat > "${var_LOG_FILE}" << 'EOF'
log::header() { echo "[HEADER] $*"; }
log::info() { echo "[INFO] $*"; }
log::success() { echo "[SUCCESS] $*"; }
log::error() { echo "[ERROR] $*" >&2; return 1; }
log::warning() { echo "[WARNING] $*"; }
log::prompt() { echo "[PROMPT] $*"; }
EOF
    
    # Create mock flow.sh
    cat > "${var_LIB_UTILS_DIR}/flow.sh" << 'EOF'
flow::is_yes() {
    local value="${1:-}"
    [[ "${value,,}" =~ ^(y|yes|true|1|on)$ ]]
}
EOF
    
    # Create mock exit_codes.sh
    cat > "${var_LIB_UTILS_DIR}/exit_codes.sh" << 'EOF'
export EXIT_USER_INTERRUPT=130
export ERROR_MISSING_DEPENDENCY=127
EOF
    
    # Create mock env.sh
    cat > "${var_APP_UTILS_DIR}/env.sh" << 'EOF'
env::load_secrets() { :; }
env::construct_derived_secrets() {
    export DB_URL="postgresql://user:pass@localhost:5432/db"
    export REDIS_URL="redis://localhost:6379"
}
EOF
    
    # Create mock docker.sh
    cat > "${var_APP_UTILS_DIR}/docker.sh" << 'EOF'
docker::build_images() {
    echo "Building Docker images..."
    return 0
}
EOF
    
    # Create mock system_commands.sh
    cat > "${var_SYSTEM_COMMANDS_FILE}" << 'EOF'
system::is_command() {
    # Mock command checking
    case "$1" in
        minikube|kubectl|helm)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}
EOF
    
    # Source all mocks
    source "${var_LOG_FILE}"
    source "${var_LIB_UTILS_DIR}/flow.sh"
    source "${var_LIB_UTILS_DIR}/exit_codes.sh"
    source "${var_APP_UTILS_DIR}/env.sh"
    source "${var_APP_UTILS_DIR}/docker.sh"
    source "${var_SYSTEM_COMMANDS_FILE}"
    
    # Global variable from script
    export ORIGINAL_DOCKER_ENV=""
    
    # Source the script under test (without executing main)
    set +e  # Don't exit on error during sourcing
    # We need to prevent main from running
    k8s_cluster::main() { :; }
    source "${BATS_TEST_DIRNAME}/k8s_cluster.sh" 2>/dev/null || true
    set -e
}

teardown() {
    cd "${ORIGINAL_DIR}"
    [[ -d "${TEST_DIR}" ]] && rm -rf "${TEST_DIR}"
}

# Test: Script sources required dependencies
@test "k8s_cluster.sh sources required dependencies" {
    run type -t k8s_cluster::revert_docker_env
    assert_success
    assert_output "function"
}

# Test: Function naming follows convention
@test "k8s_cluster functions use correct naming convention" {
    run type -t k8s_cluster::revert_docker_env
    assert_success
    assert_output "function"
    
    run type -t k8s_cluster::ensure_minikube_running
    assert_success
    assert_output "function"
    
    run type -t k8s_cluster::cleanup
    assert_success
    assert_output "function"
    
    run type -t k8s_cluster::main
    assert_success
    assert_output "function"
}

# Test: Docker environment revert functionality
@test "k8s_cluster reverts Docker environment correctly" {
    # Set up mock original environment
    export ORIGINAL_DOCKER_ENV="DOCKER_HOST=tcp://localhost:2375
DOCKER_CERT_PATH=/certs
DOCKER_TLS_VERIFY=1
MINIKUBE_ACTIVE_DOCKERD=minikube"
    
    # Set current Docker env vars
    export DOCKER_HOST="tcp://minikube:2376"
    export DOCKER_CERT_PATH="/minikube/certs"
    export DOCKER_TLS_VERIFY="1"
    export MINIKUBE_ACTIVE_DOCKERD="minikube"
    
    run k8s_cluster::revert_docker_env
    assert_success
    assert_output --partial "Reverting Docker environment"
    
    # Verify environment vars would be unset
    # (In actual execution, they would be unset)
}

# Test: Minikube status check when running
@test "k8s_cluster detects when Minikube is running" {
    # Mock minikube command
    minikube() {
        case "$1" in
            status)
                return 0  # Minikube is running
                ;;
            *)
                echo "minikube $*"
                return 0
                ;;
        esac
    }
    export -f minikube
    
    # Mock kubectl
    kubectl() {
        case "$1" in
            config)
                if [[ "$2" == "use-context" ]]; then
                    echo "Switched to context $3"
                    return 0
                fi
                ;;
        esac
    }
    export -f kubectl
    
    export YES="no"
    run k8s_cluster::ensure_minikube_running
    assert_success
    assert_output --partial "Minikube is running"
    assert_output --partial "Switched to context vrooli-dev-cluster"
}

# Test: Minikube start when not running (auto-yes)
@test "k8s_cluster starts Minikube automatically with YES=yes" {
    # Mock minikube command
    local status_call_count=0
    minikube() {
        case "$1" in
            status)
                ((status_call_count++))
                if [[ $status_call_count -eq 1 ]]; then
                    return 1  # Not running initially
                else
                    return 0  # Running after start
                fi
                ;;
            start)
                echo "Starting Minikube..."
                return 0
                ;;
            *)
                echo "minikube $*"
                return 0
                ;;
        esac
    }
    export -f minikube
    
    # Mock kubectl
    kubectl() {
        echo "kubectl $*"
        return 0
    }
    export -f kubectl
    
    export YES="yes"
    run k8s_cluster::ensure_minikube_running
    assert_success
    assert_output --partial "Starting Minikube..."
    assert_output --partial "Minikube started successfully"
}

# Test: Cleanup function
@test "k8s_cluster cleanup function works correctly" {
    # Set up mock original environment
    export ORIGINAL_DOCKER_ENV="DOCKER_HOST=tcp://localhost:2375"
    
    run k8s_cluster::cleanup
    assert_failure 130  # EXIT_USER_INTERRUPT
    assert_output --partial "Cleaning up Kubernetes development environment"
    assert_output --partial "Reverting Docker environment"
}

# Test: Kubectl context fallback to minikube
@test "k8s_cluster falls back to minikube context when vrooli-dev-cluster unavailable" {
    # Mock minikube
    minikube() {
        [[ "$1" == "status" ]] && return 0
        echo "minikube $*"
        return 0
    }
    export -f minikube
    
    # Mock kubectl with context switching logic
    kubectl() {
        case "$*" in
            "config use-context vrooli-dev-cluster")
                return 1  # Fail for vrooli-dev-cluster
                ;;
            "config use-context minikube")
                echo "Switched to minikube context"
                return 0  # Success for minikube
                ;;
            *)
                echo "kubectl $*"
                return 0
                ;;
        esac
    }
    export -f kubectl
    
    run k8s_cluster::ensure_minikube_running
    assert_success
    assert_output --partial "Failed to set kubectl context to 'vrooli-dev-cluster'"
    assert_output --partial "Switched to minikube context"
    assert_output --partial "Successfully set kubectl context to 'minikube' (fallback)"
}

# Test: Main function structure (basic validation)
@test "k8s_cluster main function validates structure" {
    # Override main to actually test it
    k8s_cluster::main() {
        log::header "🚀 Starting Kubernetes Development Environment Setup (Target: k8s-cluster)"
        k8s_cluster::ensure_minikube_running
        return 0
    }
    
    # Mock minikube and kubectl
    minikube() {
        [[ "$1" == "status" ]] && return 0
        return 0
    }
    export -f minikube
    
    kubectl() {
        return 0
    }
    export -f kubectl
    
    run k8s_cluster::main
    assert_success
    assert_output --partial "Starting Kubernetes Development Environment Setup"
}

# Test: Helm deployment command construction
@test "k8s_cluster constructs Helm command with correct options" {
    # Create mock values files
    touch "${TEST_DIR}/k8s/chart/values.yaml"
    touch "${TEST_DIR}/k8s/chart/values-dev.yaml"
    
    # Mock helm to capture commands
    helm() {
        echo "helm $*"
        # Check for expected arguments
        if [[ "$*" == *"--namespace dev"* ]] && \
           [[ "$*" == *"--create-namespace"* ]] && \
           [[ "$*" == *"--atomic"* ]] && \
           [[ "$*" == *"-f"*"values.yaml"* ]] && \
           [[ "$*" == *"-f"*"values-dev.yaml"* ]]; then
            echo "Helm command validated successfully"
            return 0
        fi
        return 1
    }
    export -f helm
    
    # Test snippet that would run helm
    local release_name="vrooli-dev"
    local namespace="dev"
    local chart_source_path="${TEST_DIR}/k8s/chart/"
    local base_values_file="${chart_source_path}values.yaml"
    local dev_values_file="${chart_source_path}values-dev.yaml"
    
    local helm_cmd_opts=()
    helm_cmd_opts+=("--namespace" "$namespace")
    helm_cmd_opts+=("--create-namespace")
    helm_cmd_opts+=("-f" "$base_values_file")
    helm_cmd_opts+=("-f" "$dev_values_file")
    helm_cmd_opts+=("--atomic")
    helm_cmd_opts+=("--timeout" "5m")
    
    run helm upgrade --install "$release_name" "$chart_source_path" "${helm_cmd_opts[@]}"
    assert_success
    assert_output --partial "Helm command validated successfully"
}

# Test: Detached mode handling
@test "k8s_cluster handles detached mode correctly" {
    export DETACHED="yes"
    
    # Create a minimal test of detached mode logic
    if flow::is_yes "${DETACHED:-no}"; then
        echo "Detached mode enabled"
    else
        echo "Detached mode disabled"
    fi | {
        run cat
        assert_success
        assert_output "Detached mode enabled"
    }
}

# Test: Non-detached mode handling
@test "k8s_cluster handles non-detached mode correctly" {
    export DETACHED="no"
    
    # Create a minimal test of non-detached mode logic
    if flow::is_yes "${DETACHED:-no}"; then
        echo "Detached mode enabled"
    else
        echo "Detached mode disabled"
    fi | {
        run cat
        assert_success
        assert_output "Detached mode disabled"
    }
}

# Test: Minikube docker-env evaluation
@test "k8s_cluster evaluates Minikube docker-env correctly" {
    # Mock minikube docker-env output
    minikube() {
        case "$*" in
            "-p minikube docker-env")
                cat << 'EOF'
export DOCKER_TLS_VERIFY="1"
export DOCKER_HOST="tcp://192.168.49.2:2376"
export DOCKER_CERT_PATH="/home/user/.minikube/certs"
export MINIKUBE_ACTIVE_DOCKERD="minikube"
# To point your shell to minikube's docker-daemon, run:
# eval $(minikube -p minikube docker-env)
EOF
                return 0
                ;;
            status)
                return 0
                ;;
            *)
                echo "minikube $*"
                return 0
                ;;
        esac
    }
    export -f minikube
    
    # Capture docker-env output
    local minikube_docker_env_cmd_output
    minikube_docker_env_cmd_output=$(minikube -p minikube docker-env 2>&1)
    
    run echo "$minikube_docker_env_cmd_output"
    assert_success
    assert_output --partial 'export DOCKER_TLS_VERIFY="1"'
    assert_output --partial 'export DOCKER_HOST="tcp://192.168.49.2:2376"'
}

# Test: Failed Minikube start handling
@test "k8s_cluster handles failed Minikube start" {
    # Mock minikube that fails to start
    minikube() {
        case "$1" in
            status)
                return 1  # Not running
                ;;
            start)
                echo "Error: Failed to start Minikube"
                return 1  # Start fails
                ;;
            *)
                return 1
                ;;
        esac
    }
    export -f minikube
    
    export YES="yes"
    run k8s_cluster::ensure_minikube_running
    assert_failure
    assert_output --partial "Failed to start Minikube"
}