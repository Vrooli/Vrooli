#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Define location for this test file
APP_LIFECYCLE_DEPLOY_DIR="$BATS_TEST_DIRNAME"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/logs.sh"

# Path to the script under test
SCRIPT_PATH="${APP_LIFECYCLE_DEPLOY_DIR}/k8s-prerequisites.sh"

setup() {
    # Initialize mocks
    mock::logs::reset
    
    # Mock kubectl commands
    kubectl() {
        case "$*" in
            "get crd postgresclusters.postgres-operator.crunchydata.com")
                return ${MOCK_PGO_INSTALLED:-1}
                ;;
            "get crd redisfailovers.databases.spotahome.com")
                return ${MOCK_REDIS_INSTALLED:-1}
                ;;
            "get crd vaultsecrets.secrets.hashicorp.com")
                return ${MOCK_VSO_INSTALLED:-1}
                ;;
            "config current-context")
                echo "test-context"
                return 0
                ;;
            "exec -n default deployment/vrooli-server -- wget"*)
                return ${MOCK_VAULT_REACHABLE:-1}
                ;;
            *)
                echo "kubectl $*"
                return 0
                ;;
        esac
    }
    export -f kubectl
    
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
            "yes")
                echo "${MOCK_AUTO_YES:-no}"
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
    
    # Mock k8s_cluster functions
    k8s_cluster::install_pgo_operator() { 
        echo "Installing PGO"
        return 0
    }
    k8s_cluster::install_spotahome_redis_operator() { 
        echo "Installing Redis operator"
        return 0
    }
    k8s_cluster::install_vso_helm_chart() { 
        echo "Installing VSO"
        return 0
    }
    export -f k8s_cluster::install_pgo_operator
    export -f k8s_cluster::install_spotahome_redis_operator
    export -f k8s_cluster::install_vso_helm_chart
    
    # Mock sleep
    sleep() { : ; }
    export -f sleep
}

teardown() {
    # Clean up mocks
    mock::logs::cleanup
    unset MOCK_PGO_INSTALLED
    unset MOCK_REDIS_INSTALLED
    unset MOCK_VSO_INSTALLED
    unset MOCK_VAULT_REACHABLE
    unset MOCK_CHECK_ONLY
    unset MOCK_AUTO_YES
}

@test "k8s-prerequisites functions exist when sourced" {
    run bash -c "source '$SCRIPT_PATH' && declare -f k8s_prerequisites::check_operators"
    [ "$status" -eq 0 ]
    [[ "$output" =~ k8s_prerequisites::check_operators ]]
    
    run bash -c "source '$SCRIPT_PATH' && declare -f k8s_prerequisites::install_operators"
    [ "$status" -eq 0 ]
    [[ "$output" =~ k8s_prerequisites::install_operators ]]
}

@test "k8s_prerequisites::check_operators detects all operators installed" {
    export MOCK_PGO_INSTALLED=0
    export MOCK_REDIS_INSTALLED=0
    export MOCK_VSO_INSTALLED=0
    export MOCK_VAULT_REACHABLE=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_prerequisites::check_operators 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CrunchyData PostgreSQL Operator" ]]
    [[ "$output" =~ "Spotahome Redis Operator" ]]
    [[ "$output" =~ "Vault Secrets Operator" ]]
}

@test "k8s_prerequisites::check_operators detects missing operators" {
    export MOCK_PGO_INSTALLED=1
    export MOCK_REDIS_INSTALLED=1
    export MOCK_VSO_INSTALLED=0
    export MOCK_VAULT_REACHABLE=1
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_prerequisites::check_operators 2>&1
    "
    
    [ "$status" -eq 2 ]  # 2 missing operators
    [[ "$output" =~ "NOT installed" ]]
}

@test "k8s_prerequisites::install_operators installs missing operators" {
    export MOCK_PGO_INSTALLED=1
    export MOCK_REDIS_INSTALLED=0
    export MOCK_VSO_INSTALLED=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_prerequisites::install_operators 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing PGO" ]]
    [[ ! "$output" =~ "Installing Redis operator" ]]  # Already installed
    [[ ! "$output" =~ "Installing VSO" ]]  # Already installed
}

@test "k8s_prerequisites::main with --check-only returns error if operators missing" {
    export MOCK_PGO_INSTALLED=1
    export MOCK_REDIS_INSTALLED=1
    export MOCK_VSO_INSTALLED=1
    export MOCK_CHECK_ONLY="yes"
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_prerequisites::main --check-only 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "missing operators" ]]
}

@test "k8s_prerequisites::main with --yes auto-installs missing operators" {
    export MOCK_PGO_INSTALLED=1
    export MOCK_REDIS_INSTALLED=1
    export MOCK_VSO_INSTALLED=1
    export MOCK_AUTO_YES="yes"
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_prerequisites::main --yes 2>&1
    "
    
    # Should install missing operators
    [[ "$output" =~ "Auto-installing" ]] || [[ "$output" =~ "missing operators" ]]
}

@test "k8s_prerequisites::main succeeds when all operators installed" {
    export MOCK_PGO_INSTALLED=0
    export MOCK_REDIS_INSTALLED=0
    export MOCK_VSO_INSTALLED=0
    export MOCK_VAULT_REACHABLE=0
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s_prerequisites::main 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "All required operators are installed" ]]
}

@test "k8s-prerequisites can be run directly from command line" {
    export MOCK_PGO_INSTALLED=0
    export MOCK_REDIS_INSTALLED=0
    export MOCK_VSO_INSTALLED=0
    export MOCK_VAULT_REACHABLE=0
    export MOCK_CHECK_ONLY="yes"
    
    run bash "$SCRIPT_PATH" --check-only
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "operator" ]]
}