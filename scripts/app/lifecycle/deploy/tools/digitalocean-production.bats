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
SCRIPT_PATH="${APP_LIFECYCLE_DEPLOY_TOOLS_DIR}/digitalocean-production.sh"

setup() {
    # Initialize mocks
    mock::logs::reset
    mock::helm::reset
    
    # Mock doctl
    doctl() {
        case "$*" in
            "auth init --access-token-stdin")
                return 0
                ;;
            "kubernetes cluster get"*)
                return ${MOCK_CLUSTER_EXISTS:-1}
                ;;
            "kubernetes cluster kubeconfig save"*)
                return 0
                ;;
            "kubernetes cluster create"*)
                return 0
                ;;
            "databases list")
                [[ ${MOCK_DB_EXISTS:-0} -eq 0 ]] && echo "vrooli-postgres"
                return 0
                ;;
            "databases create"*)
                return 0
                ;;
            "databases connection"*)
                echo "host port user pass db"
                return 0
                ;;
            *)
                echo "doctl $*"
                return 0
                ;;
        esac
    }
    export -f doctl
    
    # Mock kubectl
    kubectl() {
        case "$*" in
            "apply -f"*)
                return 0
                ;;
            "wait"*)
                return 0
                ;;
            "get svc"*"ingress"*"-o jsonpath"*)
                echo "1.2.3.4"
                return 0
                ;;
            "create namespace"*)
                return 0
                ;;
            "create secret"*)
                return 0
                ;;
            "get pods"*)
                echo "pod-1 Running"
                return 0
                ;;
            "get ingress"*)
                echo "ingress-1"
                return 0
                ;;
            "get certificates"*)
                echo "cert-1"
                return 0
                ;;
            *)
                echo "kubectl $*"
                return 0
                ;;
        esac
    }
    export -f kubectl
    
    # Mock command
    command() {
        case "$*" in
            "-v doctl")
                return ${MOCK_DOCTL_INSTALLED:-0}
                ;;
            "-v helm")
                return ${MOCK_HELM_INSTALLED:-0}
                ;;
            "-v kubectl")
                return ${MOCK_KUBECTL_INSTALLED:-0}
                ;;
            *)
                builtin command "$@"
                ;;
        esac
    }
    export -f command
    
    # Mock helm
    helm() {
        echo "helm $*"
        return 0
    }
    export -f helm
    
    # Mock sleep
    sleep() { : ; }
    export -f sleep
    
    # Mock env::load_secrets
    env::load_secrets() { : ; }
    export -f env::load_secrets
    
    # Set required environment variables
    export DIGITALOCEAN_TOKEN="test-token"
    export LETSENCRYPT_EMAIL="test@example.com"
    export DB_USER="testuser"
    export DB_PASSWORD="testpass"
    export DB_NAME="testdb"
    export REDIS_PASSWORD="redispass"
    export JWT_PRIV="jwt-private"
    export JWT_PUB="jwt-public"
    export OPENAI_API_KEY="openai-key"
    export ANTHROPIC_API_KEY="anthropic-key"
    export STRIPE_SECRET_KEY="stripe-key"
    export ADMIN_PASSWORD="admin-pass"
    export DOCKERHUB_USERNAME="docker-user"
    export DOCKERHUB_TOKEN="docker-token"
    export var_ROOT_DIR="/tmp/test-project"
    export NAMESPACE="vrooli"
    
    # Create test directories
    mkdir -p "$var_ROOT_DIR/k8s/chart"
}

teardown() {
    # Clean up mocks
    mock::logs::cleanup
    mock::helm::cleanup
    
    # Clean up test directories
    rm -rf "$var_ROOT_DIR"
    
    unset MOCK_DOCTL_INSTALLED
    unset MOCK_HELM_INSTALLED
    unset MOCK_KUBECTL_INSTALLED
    unset MOCK_CLUSTER_EXISTS
    unset MOCK_DB_EXISTS
}

@test "digitalocean-production functions exist when sourced" {
    run bash -c "source '$SCRIPT_PATH' && declare -f digitalocean_production::check_prerequisites"
    [ "$status" -eq 0 ]
    [[ "$output" =~ digitalocean_production::check_prerequisites ]]
    
    run bash -c "source '$SCRIPT_PATH' && declare -f digitalocean_production::create_cluster"
    [ "$status" -eq 0 ]
    [[ "$output" =~ digitalocean_production::create_cluster ]]
}

@test "digitalocean_production::check_prerequisites fails without doctl" {
    export MOCK_DOCTL_INSTALLED=1
    export MOCK_HELM_INSTALLED=0
    export MOCK_KUBECTL_INSTALLED=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::check_prerequisites 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "doctl CLI not found" ]]
}

@test "digitalocean_production::check_prerequisites fails without helm" {
    export MOCK_DOCTL_INSTALLED=0
    export MOCK_HELM_INSTALLED=1
    export MOCK_KUBECTL_INSTALLED=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::check_prerequisites 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "helm not found" ]]
}

@test "digitalocean_production::check_prerequisites fails without kubectl" {
    export MOCK_DOCTL_INSTALLED=0
    export MOCK_HELM_INSTALLED=0
    export MOCK_KUBECTL_INSTALLED=1
    
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::check_prerequisites 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "kubectl not found" ]]
}

@test "digitalocean_production::check_prerequisites succeeds with all tools" {
    export MOCK_DOCTL_INSTALLED=0
    export MOCK_HELM_INSTALLED=0
    export MOCK_KUBECTL_INSTALLED=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::check_prerequisites 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Prerequisites check completed" ]]
}

@test "digitalocean_production::authenticate fails without token" {
    unset DIGITALOCEAN_TOKEN
    
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::authenticate 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "DIGITALOCEAN_TOKEN environment variable not set" ]]
}

@test "digitalocean_production::authenticate succeeds with token" {
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::authenticate 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Authenticated with DigitalOcean" ]]
}

@test "digitalocean_production::create_cluster handles existing cluster" {
    export MOCK_CLUSTER_EXISTS=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::create_cluster 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "already exists" ]]
}

@test "digitalocean_production::create_cluster creates new cluster" {
    export MOCK_CLUSTER_EXISTS=1
    
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::create_cluster 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating cluster" ]]
}

@test "digitalocean_production::install_operators installs cert-manager and nginx" {
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::install_operators 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "cert-manager" ]]
    [[ "$output" =~ "nginx-ingress" ]]
}

@test "digitalocean_production::create_ssl_issuer creates letsencrypt issuer" {
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::create_ssl_issuer 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SSL certificate issuer created" ]]
}

@test "digitalocean_production::create_secrets creates all required secrets" {
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::create_secrets 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Secrets created successfully" ]]
}

@test "digitalocean_production::create_databases skips existing databases" {
    export MOCK_DB_EXISTS=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::create_databases 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Managed databases" ]]
}

@test "digitalocean_production::deploy_application creates values and deploys" {
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::deploy_application 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Application deployed successfully" ]]
    [ -f "$var_ROOT_DIR/k8s/chart/values-production.yaml" ]
}

@test "digitalocean_production::verify_deployment shows deployment status" {
    run bash -c "
        source '$SCRIPT_PATH'
        digitalocean_production::verify_deployment 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Deployment verification completed" ]]
    [[ "$output" =~ "External IP" ]]
}