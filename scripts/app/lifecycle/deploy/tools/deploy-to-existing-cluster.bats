#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Define location for this test file
APP_LIFECYCLE_DEPLOY_TOOLS_DIR="$BATS_TEST_DIRNAME"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_TOOLS_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/logs.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/helm.sh"

# Path to the script under test
SCRIPT_PATH="${APP_LIFECYCLE_DEPLOY_TOOLS_DIR}/deploy-to-existing-cluster.sh"

setup() {
    # Initialize mocks
    mock::logs::reset
    mock::helm::reset
    
    # Mock kubectl
    kubectl() {
        case "$*" in
            "cluster-info")
                [[ ${MOCK_CLUSTER_CONNECTED:-1} -eq 0 ]] && echo "Kubernetes control plane is running"
                return ${MOCK_CLUSTER_CONNECTED:-1}
                ;;
            "config current-context")
                echo "test-context"
                return 0
                ;;
            "get nodes -o wide")
                echo "node1 Ready"
                return 0
                ;;
            "get namespaces")
                echo "default kube-system"
                return 0
                ;;
            "create namespace"*)
                return 0
                ;;
            "apply -f -")
                return 0
                ;;
            "create secret"*)
                return 0
                ;;
            "get svc"*"ingress"*)
                echo '{"status":{"loadBalancer":{"ingress":[{"ip":"1.2.3.4"}]}}}'
                return 0
                ;;
            *)
                echo "kubectl $*"
                return 0
                ;;
        esac
    }
    export -f kubectl
    
    # Mock helm
    command() {
        if [[ "$1" == "-v" ]] && [[ "$2" == "helm" ]]; then
            return ${MOCK_HELM_INSTALLED:-0}
        fi
        builtin command "$@"
    }
    export -f command
    
    helm() {
        echo "helm $*"
        return 0
    }
    export -f helm
    
    # Mock read for user input
    read() {
        case "$2" in
            "response")
                eval "$2='${MOCK_USER_RESPONSE:-n}'"
                ;;
            "env_choice")
                eval "$2='${MOCK_ENV_CHOICE:-n}'"
                ;;
            "ENVIRONMENT")
                eval "$2='development'"
                ;;
            "db_option")
                eval "$2='${MOCK_DB_OPTION:-2}'"
                ;;
            *)
                eval "$2='test-value'"
                ;;
        esac
    }
    export -f read
    
    # Mock env functions
    env::load_secrets() { : ; }
    export -f env::load_secrets
    
    # Set required vars
    export NAMESPACE="vrooli"
    export HELM_RELEASE="vrooli"
    export var_ROOT_DIR="/tmp/test-project"
    export var_APP_UTILS_DIR="$var_ROOT_DIR/scripts/app/utils"
    
    # Create test directories
    mkdir -p "$var_ROOT_DIR/k8s/chart"
    mkdir -p "$var_APP_UTILS_DIR"
    touch "$var_APP_UTILS_DIR/env.sh"
}

teardown() {
    # Clean up mocks
    mock::logs::cleanup
    mock::helm::cleanup
    
    # Clean up test directories
    rm -rf "$var_ROOT_DIR"
    
    unset MOCK_CLUSTER_CONNECTED
    unset MOCK_HELM_INSTALLED
    unset MOCK_USER_RESPONSE
    unset MOCK_ENV_CHOICE
    unset MOCK_DB_OPTION
}

@test "deploy-to-existing-cluster functions exist when sourced" {
    run bash -c "source '$SCRIPT_PATH' && declare -f deploy_to_existing_cluster::check_prerequisites"
    [ "$status" -eq 0 ]
    [[ "$output" =~ deploy_to_existing_cluster::check_prerequisites ]]
    
    run bash -c "source '$SCRIPT_PATH' && declare -f deploy_to_existing_cluster::deploy_application"
    [ "$status" -eq 0 ]
    [[ "$output" =~ deploy_to_existing_cluster::deploy_application ]]
}

@test "deploy_to_existing_cluster::check_prerequisites fails without kubectl connection" {
    export MOCK_CLUSTER_CONNECTED=1
    
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::check_prerequisites 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "kubectl is not connected to a cluster" ]]
}

@test "deploy_to_existing_cluster::check_prerequisites fails without helm" {
    export MOCK_CLUSTER_CONNECTED=0
    export MOCK_HELM_INSTALLED=1
    
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::check_prerequisites 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "helm not found" ]]
}

@test "deploy_to_existing_cluster::check_prerequisites succeeds with all tools" {
    export MOCK_CLUSTER_CONNECTED=0
    export MOCK_HELM_INSTALLED=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::check_prerequisites 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Prerequisites check completed" ]]
}

@test "deploy_to_existing_cluster::confirm_cluster exits on no confirmation" {
    export MOCK_USER_RESPONSE="n"
    
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::confirm_cluster 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "cancelled by user" ]]
}

@test "deploy_to_existing_cluster::create_namespace creates namespace" {
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::create_namespace 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Namespace" ]]
    [[ "$output" =~ "ready" ]]
}

@test "deploy_to_existing_cluster::setup_internal_databases creates simple credentials" {
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::setup_internal_databases 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Internal database secrets created" ]]
}

@test "deploy_to_existing_cluster::setup_database_secrets handles user choice" {
    export MOCK_DB_OPTION="2"
    
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::setup_database_secrets 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "database" ]]
}

@test "deploy_to_existing_cluster::create_deployment_values generates values file" {
    export EXTERNAL_DB_ENABLED="false"
    
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::create_deployment_values 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Deployment configuration created" ]]
    [ -f "$var_ROOT_DIR/k8s/chart/values-current-deployment.yaml" ]
}

@test "deploy_to_existing_cluster::deploy_application runs helm upgrade" {
    touch "$var_ROOT_DIR/k8s/chart/values-current-deployment.yaml"
    
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::deploy_application 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "helm upgrade" ]]
}

@test "deploy_to_existing_cluster::verify_deployment shows deployment status" {
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::verify_deployment 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Pod status" ]]
    [[ "$output" =~ "Service status" ]]
}

@test "deploy_to_existing_cluster::show_next_steps displays post-deployment tasks" {
    run bash -c "
        source '$SCRIPT_PATH'
        deploy_to_existing_cluster::show_next_steps 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Post-Deployment Tasks" ]]
    [[ "$output" =~ "Configure DNS" ]]
}