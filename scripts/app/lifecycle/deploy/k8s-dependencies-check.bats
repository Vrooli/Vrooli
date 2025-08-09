#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Define location for this test file
APP_LIFECYCLE_DEPLOY_DIR="$BATS_TEST_DIRNAME"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/logs.sh"

# Path to the script under test
SCRIPT_PATH="${APP_LIFECYCLE_DEPLOY_DIR}/k8s-dependencies-check.sh"

setup() {
    # Initialize mocks
    mock::logs::reset
    
    # Mock kubectl commands
    kubectl() {
        case "$*" in
            "get crd postgresclusters.postgres-operator.crunchydata.com")
                return ${MOCK_PGO_CRD:-1}
                ;;
            "get pods -n postgres-operator"*"Running")
                [[ ${MOCK_PGO_RUNNING:-0} -eq 0 ]] && echo "pod-name Running"
                return 0
                ;;
            "get crd redisfailovers.databases.spotahome.com")
                return ${MOCK_REDIS_CRD:-1}
                ;;
            "get pods -n redis-operator"*"Running")
                [[ ${MOCK_REDIS_RUNNING:-0} -eq 0 ]] && echo "pod-name Running"
                return 0
                ;;
            "get crd vaultsecrets.secrets.hashicorp.com")
                return ${MOCK_VSO_CRD:-1}
                ;;
            "get pods -n vault-secrets-operator-system"*"Running")
                [[ ${MOCK_VSO_RUNNING:-0} -eq 0 ]] && echo "pod-name Running"
                return 0
                ;;
            "get secret"*"-n"*)
                return ${MOCK_PULL_SECRET:-1}
                ;;
            "get storageclass")
                [[ ${MOCK_DEFAULT_SC:-0} -eq 0 ]] && echo "standard (default)"
                return 0
                ;;
            "get pods -A"*"ingress"*)
                [[ ${MOCK_INGRESS:-0} -eq 0 ]] && echo "pod-name Running"
                return 0
                ;;
            "exec -n kube-system deployment/coredns"*)
                return ${MOCK_DNS_WORKING:-1}
                ;;
            "config current-context")
                echo "test-context"
                return 0
                ;;
            *)
                echo "kubectl $*"
                return 0
                ;;
        esac
    }
    export -f kubectl
    
    # Mock other commands
    nslookup() {
        return ${MOCK_REGISTRY_DNS:-0}
    }
    export -f nslookup
    
    curl() {
        return ${MOCK_VAULT_CURL:-1}
    }
    export -f curl
    
    # Mock args functions
    args::reset() { : ; }
    args::register() { : ; }
    args::register_help() { : ; }
    args::is_asking_for_help() { return 1; }
    args::parse() { : ; }
    args::get() {
        case "$1" in
            "check-only")
                echo "${MOCK_CHECK_ONLY:-no}"
                ;;
            "environment")
                echo "${MOCK_ENVIRONMENT:-dev}"
                ;;
            *)
                echo "no"
                ;;
        esac
    }
    export -f args::reset
    export -f args::register
    export -f args::register_help
    export -f args::is_asking_for_help
    export -f args::parse
    export -f args::get
}

teardown() {
    # Clean up mocks
    mock::logs::cleanup
    unset MOCK_PGO_CRD MOCK_PGO_RUNNING
    unset MOCK_REDIS_CRD MOCK_REDIS_RUNNING
    unset MOCK_VSO_CRD MOCK_VSO_RUNNING
    unset MOCK_PULL_SECRET MOCK_DEFAULT_SC
    unset MOCK_INGRESS MOCK_DNS_WORKING
    unset MOCK_REGISTRY_DNS MOCK_VAULT_CURL
    unset MOCK_CHECK_ONLY MOCK_ENVIRONMENT
}

@test "k8s-dependencies-check functions exist when sourced" {
    run bash -c "source '$SCRIPT_PATH' && declare -f k8s_dependencies_check::check_operators"
    [ "$status" -eq 0 ]
    [[ "$output" =~ k8s_dependencies_check::check_operators ]]
    
    run bash -c "source '$SCRIPT_PATH' && declare -f k8s_dependencies_check::check_vault"
    [ "$status" -eq 0 ]
    [[ "$output" =~ k8s_dependencies_check::check_vault ]]
}

@test "k8s_dependencies_check::check_operators detects installed and running operators" {
    export MOCK_PGO_CRD=0
    export MOCK_PGO_RUNNING=0
    export MOCK_REDIS_CRD=0
    export MOCK_REDIS_RUNNING=0
    export MOCK_VSO_CRD=0
    export MOCK_VSO_RUNNING=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_dependencies_check::check_operators 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CrunchyData PostgreSQL Operator (running)" ]]
    [[ "$output" =~ "Spotahome Redis Operator (running)" ]]
    [[ "$output" =~ "Vault Secrets Operator (running)" ]]
}

@test "k8s_dependencies_check::check_operators detects installed but not running operators" {
    export MOCK_PGO_CRD=0
    export MOCK_PGO_RUNNING=1
    export MOCK_REDIS_CRD=0
    export MOCK_REDIS_RUNNING=1
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_dependencies_check::check_operators 2>&1
    "
    
    [ "$status" -eq 2 ]
    [[ "$output" =~ "installed but not running" ]]
}

@test "k8s_dependencies_check::check_vault checks dev environment vault" {
    export MOCK_ENVIRONMENT="dev"
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_dependencies_check::check_vault 2>&1
    "
    
    # Should check in-cluster vault
    [[ "$output" =~ "Vault" ]]
}

@test "k8s_dependencies_check::check_vault checks production environment vault" {
    export MOCK_ENVIRONMENT="prod"
    export VAULT_ADDR="https://vault.example.com"
    export MOCK_VAULT_CURL=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_dependencies_check::check_vault 2>&1
    "
    
    # Should check external vault
    [[ "$output" =~ "External Vault" ]]
}

@test "k8s_dependencies_check::check_registry checks for pull secret" {
    export MOCK_ENVIRONMENT="dev"
    export MOCK_PULL_SECRET=0
    export MOCK_REGISTRY_DNS=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_dependencies_check::check_registry 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Pull Secret" ]]
    [[ "$output" =~ "Registry DNS" ]]
}

@test "k8s_dependencies_check::check_storage detects default storage class" {
    export MOCK_DEFAULT_SC=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_dependencies_check::check_storage 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Default storage class" ]]
}

@test "k8s_dependencies_check::check_networking checks ingress for non-dev" {
    export MOCK_ENVIRONMENT="prod"
    export MOCK_INGRESS=0
    export MOCK_DNS_WORKING=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_dependencies_check::check_networking 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Ingress Controller" ]]
}

@test "k8s_dependencies_check::main succeeds with all dependencies met" {
    export MOCK_PGO_CRD=0
    export MOCK_PGO_RUNNING=0
    export MOCK_REDIS_CRD=0
    export MOCK_REDIS_RUNNING=0
    export MOCK_VSO_CRD=0
    export MOCK_VSO_RUNNING=0
    export MOCK_PULL_SECRET=0
    export MOCK_REGISTRY_DNS=0
    export MOCK_DEFAULT_SC=0
    export MOCK_INGRESS=0
    export MOCK_DNS_WORKING=0
    export MOCK_ENVIRONMENT="dev"
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_dependencies_check::main 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "All dependencies are properly configured" ]]
}

@test "k8s_dependencies_check::main fails with missing dependencies" {
    export MOCK_PGO_CRD=1
    export MOCK_REDIS_CRD=1
    export MOCK_CHECK_ONLY="yes"
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_dependencies_check::main --check-only 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "dependency issues" ]]
}