#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Define location for this test file
APP_LIFECYCLE_DEPLOY_DIR="$BATS_TEST_DIRNAME"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/helm.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/logs.sh"

# Path to the script under test
SCRIPT_PATH="${APP_LIFECYCLE_DEPLOY_DIR}/k8s.sh"

setup() {
    # Initialize mocks
    mock::helm::reset
    mock::logs::reset
    
    # Set required environment variables
    export VERSION="1.0.0"
    export var_DEST_DIR="/tmp/test-dest"
    
    # Create test directory structure
    mkdir -p "${var_DEST_DIR}/${VERSION}/artifacts/k8s-chart-packages"
    mkdir -p "${var_DEST_DIR}/${VERSION}/artifacts/helm-value-files"
}

teardown() {
    # Clean up mocks
    mock::helm::cleanup
    mock::logs::cleanup
    
    # Clean up test directories
    trash::safe_remove "${var_DEST_DIR}" --test-cleanup
}

@test "k8s::deploy_k8s function exists when sourced" {
    run bash -c "source '$SCRIPT_PATH' && declare -f k8s::deploy_k8s"
    [ "$status" -eq 0 ]
    [[ "$output" =~ k8s::deploy_k8s ]]
}

@test "k8s::deploy_k8s requires target environment argument" {
    run bash -c "
        source '$SCRIPT_PATH'
        k8s::deploy_k8s 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "target environment" ]] || [[ "$output" =~ "not specified" ]]
}

@test "k8s::deploy_k8s checks for helm command" {
    # Mock helm not found
    command() {
        if [[ "$1" == "-v" ]] && [[ "$2" == "helm" ]]; then
            return 1
        fi
        builtin command "$@"
    }
    export -f command
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s::deploy_k8s 'dev' 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Helm CLI not found" ]] || [[ "$output" =~ "helm" ]]
}

@test "k8s::deploy_k8s checks for prerequisites" {
    # Mock helm available
    command() {
        if [[ "$1" == "-v" ]] && [[ "$2" == "helm" ]]; then
            return 0
        fi
        builtin command "$@"
    }
    export -f command
    
    # Mock prerequisite script exists but fails
    bash() {
        if [[ "$1" == *"k8s-prerequisites.sh" ]]; then
            return 1
        fi
        builtin bash "$@"
    }
    export -f bash
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s::deploy_k8s 'dev' 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Required Kubernetes operators are missing" ]] || [[ "$output" =~ "prerequisites" ]]
}

@test "k8s::deploy_k8s finds packaged chart" {
    # Mock helm available
    command() {
        if [[ "$1" == "-v" ]] && [[ "$2" == "helm" ]]; then
            return 0
        fi
        builtin command "$@"
    }
    export -f command
    
    # Mock prerequisite check passes
    bash() {
        if [[ "$1" == *"k8s-prerequisites.sh" ]]; then
            return 0
        fi
        builtin bash "$@"
    }
    export -f bash
    
    # No chart file exists
    run bash -c "
        source '$SCRIPT_PATH'
        k8s::deploy_k8s 'dev' 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "No packaged Helm chart" ]] || [[ "$output" =~ ".tgz file" ]]
}

@test "k8s::deploy_k8s detects multiple chart files" {
    # Mock helm available
    command() {
        if [[ "$1" == "-v" ]] && [[ "$2" == "helm" ]]; then
            return 0
        fi
        builtin command "$@"
    }
    export -f command
    
    # Mock prerequisite check passes
    bash() {
        if [[ "$1" == *"k8s-prerequisites.sh" ]]; then
            return 0
        fi
        builtin bash "$@"
    }
    export -f bash
    
    # Create multiple chart files
    touch "${var_DEST_DIR}/${VERSION}/artifacts/k8s-chart-packages/chart1.tgz"
    touch "${var_DEST_DIR}/${VERSION}/artifacts/k8s-chart-packages/chart2.tgz"
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s::deploy_k8s 'dev' 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Multiple .tgz files found" ]]
}

@test "k8s::deploy_k8s uses environment-specific values file" {
    # Mock everything needed
    command() {
        if [[ "$1" == "-v" ]] && [[ "$2" == "helm" ]]; then
            return 0
        fi
        builtin command "$@"
    }
    export -f command
    
    bash() {
        if [[ "$1" == *"k8s-prerequisites.sh" ]]; then
            return 0
        fi
        builtin bash "$@"
    }
    export -f bash
    
    helm() {
        echo "helm $*"
        return 0
    }
    export -f helm
    
    repository::get_url() {
        echo "https://github.com/example/repo"
    }
    repository::get_branch() {
        echo "main"
    }
    export -f repository::get_url
    export -f repository::get_branch
    
    git() {
        if [[ "$1" == "rev-parse" ]]; then
            echo "abc123"
        fi
    }
    export -f git
    
    date() {
        echo "2024-01-01T00:00:00Z"
    }
    export -f date
    
    # Create single chart file
    touch "${var_DEST_DIR}/${VERSION}/artifacts/k8s-chart-packages/vrooli-${VERSION}.tgz"
    # Create environment-specific values
    touch "${var_DEST_DIR}/${VERSION}/artifacts/helm-value-files/values-dev.yaml"
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s::deploy_k8s 'dev' 2>&1
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "values-dev.yaml" ]]
}

@test "k8s::deploy_k8s fails for production without values file" {
    # Mock everything needed
    command() {
        if [[ "$1" == "-v" ]] && [[ "$2" == "helm" ]]; then
            return 0
        fi
        builtin command "$@"
    }
    export -f command
    
    bash() {
        if [[ "$1" == *"k8s-prerequisites.sh" ]]; then
            return 0
        fi
        builtin bash "$@"
    }
    export -f bash
    
    # Create single chart file
    touch "${var_DEST_DIR}/${VERSION}/artifacts/k8s-chart-packages/vrooli-${VERSION}.tgz"
    # No production values file
    
    run bash -c "
        source '$SCRIPT_PATH'
        k8s::deploy_k8s 'prod' 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "CRITICAL" ]] || [[ "$output" =~ "Production deployment" ]]
}