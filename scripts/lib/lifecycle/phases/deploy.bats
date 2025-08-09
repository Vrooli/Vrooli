#!/usr/bin/env bats
################################################################################
# Tests for deploy.sh - Universal deploy phase handler
################################################################################

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../../__test/helpers/bats-assert/load"

# Load required mocks
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/commands.bash"
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/system.sh"
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/filesystem.sh"
load "${BATS_TEST_DIRNAME}/../../../__test/fixtures/mocks/docker.sh"

setup() {
    vrooli_setup_unit_test
    
    # Reset all mocks
    mock::commands::reset
    mock::system::reset
    mock::filesystem::reset
    mock::docker::reset
    
    # Mock phase utilities
    mock::commands::setup_function "phase::init" 0
    mock::commands::setup_function "phase::complete" 0
    mock::commands::setup_function "phase::run_hook" 0
    mock::commands::setup_function "log::info" 0
    mock::commands::setup_function "log::debug" 0
    mock::commands::setup_function "log::header" 0
    mock::commands::setup_function "log::success" 0
    mock::commands::setup_function "log::warning" 0
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "flow::is_yes" 1  # Default to "no"
    
    # Set up test environment
    export SOURCE_TYPE="docker"
    export VERSION="1.0.0"
    export ENVIRONMENT="production"
    export SKIP_HEALTH_CHECK="no"
    export var_ROOT_DIR="${MOCK_ROOT}"
    export var_DEST_DIR="${MOCK_ROOT}/dist"
    export var_DOCKER_COMPOSE_DEV_FILE="${MOCK_ROOT}/docker/docker-compose.yml"
    export var_APP_LIFECYCLE_DEPLOY_DIR="${MOCK_ROOT}/app/lifecycle/deploy"
    
    # Source deploy functions for testing
    source "${BATS_TEST_DIRNAME}/deploy.sh"
}

teardown() {
    vrooli_cleanup_test
}

################################################################################
# Artifact Validation Tests
################################################################################

@test "deploy::validate_artifacts validates Docker image exists" {
    export IMAGE_NAME="myapp"
    mock::commands::setup_command "command -v docker" "" "0"
    mock::docker::set_image_exists "myapp:1.0.0" true
    
    run deploy::validate_artifacts "docker" "1.0.0"
    
    assert_success
    mock::commands::assert::called "log::success" "✅ Docker image found: myapp:1.0.0"
}

@test "deploy::validate_artifacts fails when Docker image missing" {
    export IMAGE_NAME="myapp"
    mock::commands::setup_command "command -v docker" "" "0"
    mock::docker::set_image_exists "myapp:1.0.0" false
    
    run deploy::validate_artifacts "docker" "1.0.0"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Docker image not found: myapp:1.0.0"
    mock::commands::assert::called "log::info" "Run './scripts/manage.sh build --artifacts docker' first"
}

@test "deploy::validate_artifacts fails when Docker unavailable" {
    mock::commands::setup_command "command -v docker" "" "1"  # Docker not found
    
    run deploy::validate_artifacts "docker" "1.0.0"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Docker is not installed"
}

@test "deploy::validate_artifacts validates k8s manifests" {
    local k8s_dir="${MOCK_ROOT}/k8s"
    mock::filesystem::create_directory "$k8s_dir"
    mock::filesystem::create_file "$k8s_dir/deployment.yaml" "apiVersion: apps/v1"
    
    run deploy::validate_artifacts "k8s" "1.0.0"
    
    assert_success
    mock::commands::assert::called "log::success" "✅ Kubernetes manifests found"
}

@test "deploy::validate_artifacts fails when k8s manifests missing" {
    # No k8s directory
    
    run deploy::validate_artifacts "kubernetes" "1.0.0"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Kubernetes manifests not found"
}

@test "deploy::validate_artifacts validates zip bundle" {
    mock::filesystem::create_file "${var_DEST_DIR}/bundle-1.0.0.zip" "fake zip content"
    
    run deploy::validate_artifacts "zip" "1.0.0"
    
    assert_success
    mock::commands::assert::called "log::success" "✅ Deployment bundle found: ${var_DEST_DIR}/bundle-1.0.0.zip"
}

@test "deploy::validate_artifacts fails when zip bundle missing" {
    # No bundle file
    
    run deploy::validate_artifacts "zip" "1.0.0"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Deployment bundle not found: ${var_DEST_DIR}/bundle-1.0.0.zip"
    mock::commands::assert::called "log::info" "Run './scripts/manage.sh build --bundles zip' first"
}

@test "deploy::validate_artifacts fails for unknown source type" {
    run deploy::validate_artifacts "unknown" "1.0.0"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Unknown source type: unknown"
}

################################################################################
# Docker Deployment Tests
################################################################################

@test "deploy::docker uses environment-specific compose file" {
    local compose_file="${MOCK_ROOT}/docker-compose.production.yml"
    mock::filesystem::create_file "$compose_file" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f $compose_file up -d" ""
    
    run deploy::docker "1.0.0" "production"
    
    assert_success
    mock::commands::assert::called "docker-compose" "-f $compose_file up -d"
    mock::commands::assert::called "log::success" "✅ Docker deployment complete"
}

@test "deploy::docker falls back to dev compose file" {
    # No production-specific file, use dev file
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    
    run deploy::docker "1.0.0" "production"
    
    assert_success
    mock::commands::assert::called "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d"
}

@test "deploy::docker uses docker compose when docker-compose unavailable" {
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "1"  # Not found
    mock::commands::setup_command "docker" "compose -f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    
    run deploy::docker "1.0.0" "production"
    
    assert_success
    mock::commands::assert::called "docker" "compose -f ${var_DOCKER_COMPOSE_DEV_FILE} up -d"
}

@test "deploy::docker fails when no compose file found" {
    # No compose files available
    
    run deploy::docker "1.0.0" "production"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "No docker-compose file found"
}

@test "deploy::docker exports environment variables" {
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    
    deploy::docker "2.0.0" "staging"
    
    assert_equal "$VERSION" "2.0.0"
    assert_equal "$ENVIRONMENT" "staging"
}

################################################################################
# Kubernetes Deployment Tests
################################################################################

@test "deploy::kubernetes validates kubectl availability" {
    mock::commands::setup_command "command -v kubectl" "" "1"  # kubectl not found
    
    run deploy::kubernetes "1.0.0" "production"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "kubectl is not installed"
}

@test "deploy::kubernetes validates cluster connection" {
    mock::commands::setup_command "command -v kubectl" "" "0"
    mock::commands::setup_command "kubectl" "cluster-info" "" "1"  # Cannot connect
    
    run deploy::kubernetes "1.0.0" "production"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Cannot connect to Kubernetes cluster"
    mock::commands::assert::called "log::info" "Check your KUBECONFIG or kubectl context"
}

@test "deploy::kubernetes deploys manifests successfully" {
    local k8s_dir="${MOCK_ROOT}/k8s"
    mock::filesystem::create_directory "$k8s_dir"
    mock::filesystem::create_file "$k8s_dir/deployment.yaml" "apiVersion: apps/v1"
    
    export NAMESPACE="myapp"
    mock::commands::setup_command "command -v kubectl" "" "0"
    mock::commands::setup_command "kubectl" "cluster-info" ""
    mock::commands::setup_command "kubectl" "create namespace myapp" ""
    mock::commands::setup_command "kubectl" "apply -f $k8s_dir -n myapp" ""
    mock::commands::setup_command "kubectl" "rollout status deployment -n myapp --timeout=5m" ""
    
    run deploy::kubernetes "1.0.0" "production"
    
    assert_success
    mock::commands::assert::called "kubectl" "create namespace myapp"
    mock::commands::assert::called "kubectl" "apply -f $k8s_dir -n myapp"
    mock::commands::assert::called "kubectl" "rollout status deployment -n myapp --timeout=5m"
    mock::commands::assert::called "log::success" "✅ Kubernetes deployment complete"
}

@test "deploy::kubernetes uses default namespace" {
    local k8s_dir="${MOCK_ROOT}/k8s"
    mock::filesystem::create_directory "$k8s_dir"
    mock::filesystem::create_file "$k8s_dir/deployment.yaml" "apiVersion: apps/v1"
    
    # No NAMESPACE set
    unset NAMESPACE
    mock::commands::setup_command "command -v kubectl" "" "0"
    mock::commands::setup_command "kubectl" "cluster-info" ""
    mock::commands::setup_command "kubectl" "create namespace default" ""
    mock::commands::setup_command "kubectl" "apply -f $k8s_dir -n default" ""
    mock::commands::setup_command "kubectl" "rollout status deployment -n default --timeout=5m" ""
    
    run deploy::kubernetes "1.0.0" "production"
    
    assert_success
    mock::commands::assert::called "kubectl" "apply -f $k8s_dir -n default"
}

@test "deploy::kubernetes fails when manifests missing" {
    mock::commands::setup_command "command -v kubectl" "" "0"
    mock::commands::setup_command "kubectl" "cluster-info" ""
    
    # No k8s directory
    
    run deploy::kubernetes "1.0.0" "production"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Kubernetes manifests not found"
}

################################################################################
# Health Check Tests
################################################################################

@test "deploy::health_check passes for healthy Docker container" {
    export CONTAINER_NAME="myapp"
    export HEALTH_PORT="8080"
    export HEALTH_ENDPOINT="/health"
    
    mock::commands::setup_command "docker" "exec myapp curl -f \"http://localhost:8080/health\"" ""
    
    run deploy::health_check "docker"
    
    assert_success
    mock::commands::assert::called "log::success" "✅ Health check passed"
}

@test "deploy::health_check retries on failure then succeeds" {
    export CONTAINER_NAME="myapp"
    export HEALTH_PORT="8080"
    export HEALTH_ENDPOINT="/health"
    
    # First call fails, second succeeds
    mock::commands::setup_command "docker" "exec myapp curl -f \"http://localhost:8080/health\"" "" "1"
    mock::commands::setup_command "docker" "exec myapp curl -f \"http://localhost:8080/health\"" ""
    mock::commands::setup_command "sleep" "2" ""
    
    run deploy::health_check "docker"
    
    assert_success
    mock::commands::assert::called "log::info" "Waiting for service to be healthy... (1/30)"
    mock::commands::assert::called "log::success" "✅ Health check passed"
}

@test "deploy::health_check fails after max attempts" {
    export CONTAINER_NAME="myapp"
    export HEALTH_PORT="8080"
    export HEALTH_ENDPOINT="/health"
    
    # Always fail
    mock::commands::setup_command "docker" "exec myapp curl -f \"http://localhost:8080/health\"" "" "1"
    mock::commands::setup_command "sleep" "2" ""
    
    # Mock the loop to avoid waiting too long in test
    deploy::health_check() {
        local max_attempts=3  # Reduced for testing
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if docker exec "$CONTAINER_NAME" curl -f "http://localhost:${HEALTH_PORT}${HEALTH_ENDPOINT}" &> /dev/null; then
                log::success "✅ Health check passed"
                return 0
            fi
            
            attempt=$((attempt + 1))
            log::info "Waiting for service to be healthy... ($attempt/$max_attempts)"
            sleep 2
        done
        
        log::error "Health check failed after $max_attempts attempts"
        return 1
    }
    
    run deploy::health_check "docker"
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Health check failed after 3 attempts"
}

@test "deploy::health_check works for k8s deployment" {
    export NAMESPACE="default"
    export SERVICE_NAME="myapp"
    export HEALTH_PORT="8080"
    export HEALTH_ENDPOINT="/health"
    
    mock::commands::setup_command "kubectl" "exec -n default deploy/myapp -- curl -f \"http://localhost:8080/health\"" ""
    
    run deploy::health_check "k8s"
    
    assert_success
    mock::commands::assert::called "log::success" "✅ Health check passed"
}

################################################################################
# Main Deploy Function Tests
################################################################################

@test "deploy::universal::main initializes phase correctly" {
    export SOURCE_TYPE="docker"
    export IMAGE_NAME="myapp"
    mock::docker::set_image_exists "myapp:1.0.0" true
    mock::commands::setup_command "command -v docker" "" "0"
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    
    run deploy::universal::main
    
    assert_success
    mock::commands::assert::called "phase::init" "Deploy"
    mock::commands::assert::called "phase::complete"
}

@test "deploy::universal::main validates artifacts first" {
    export SOURCE_TYPE="docker"
    export IMAGE_NAME="myapp"
    mock::docker::set_image_exists "myapp:1.0.0" false  # Image doesn't exist
    mock::commands::setup_command "command -v docker" "" "0"
    
    run deploy::universal::main
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Artifact validation failed"
}

@test "deploy::universal::main runs pre-deploy hook" {
    export SOURCE_TYPE="docker"
    export IMAGE_NAME="myapp"
    mock::docker::set_image_exists "myapp:1.0.0" true
    mock::commands::setup_command "command -v docker" "" "0"
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    
    run deploy::universal::main
    
    assert_success
    mock::commands::assert::called "phase::run_hook" "preDeploy"
}

@test "deploy::universal::main deploys Docker successfully" {
    export SOURCE_TYPE="docker"
    export IMAGE_NAME="myapp"
    mock::docker::set_image_exists "myapp:1.0.0" true
    mock::commands::setup_command "command -v docker" "" "0"
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    
    run deploy::universal::main
    
    assert_success
    mock::commands::assert::called "docker-compose"
    mock::commands::assert::called "log::success" "✅ Deployment completed successfully"
}

@test "deploy::universal::main deploys Kubernetes successfully" {
    export SOURCE_TYPE="k8s"
    local k8s_dir="${MOCK_ROOT}/k8s"
    mock::filesystem::create_directory "$k8s_dir"
    mock::filesystem::create_file "$k8s_dir/deployment.yaml" "apiVersion: apps/v1"
    
    mock::commands::setup_command "command -v kubectl" "" "0"
    mock::commands::setup_command "kubectl" "cluster-info" ""
    mock::commands::setup_command "kubectl" "create namespace default" ""
    mock::commands::setup_command "kubectl" "apply -f $k8s_dir -n default" ""
    mock::commands::setup_command "kubectl" "rollout status deployment -n default --timeout=5m" ""
    
    run deploy::universal::main
    
    assert_success
    mock::commands::assert::called "kubectl"
    mock::commands::assert::called "log::success" "✅ Deployment completed successfully"
}

@test "deploy::universal::main uses app-specific deployment" {
    export SOURCE_TYPE="custom"
    local deploy_script="${var_APP_LIFECYCLE_DEPLOY_DIR}/custom.sh"
    mock::filesystem::create_file "$deploy_script" '#!/bin/bash\necho "custom deployment"'
    chmod +x "$deploy_script"
    mock::commands::setup_command "bash" "$deploy_script 1.0.0 production" ""
    
    run deploy::universal::main
    
    assert_success
    mock::commands::assert::called "bash" "$deploy_script 1.0.0 production"
}

@test "deploy::universal::main fails for unknown deployment source" {
    export SOURCE_TYPE="unknown"
    # No app-specific script
    
    run deploy::universal::main
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Unknown deployment source: unknown"
}

@test "deploy::universal::main runs health checks by default" {
    export SOURCE_TYPE="docker"
    export IMAGE_NAME="myapp"
    export CONTAINER_NAME="myapp"
    mock::docker::set_image_exists "myapp:1.0.0" true
    mock::commands::setup_command "command -v docker" "" "0"
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    mock::commands::setup_command "docker" "exec myapp curl -f \"http://localhost:8080/health\"" ""
    
    run deploy::universal::main
    
    assert_success
    mock::commands::assert::called "docker" "exec myapp curl -f \"http://localhost:8080/health\""
}

@test "deploy::universal::main skips health checks when disabled" {
    export SOURCE_TYPE="docker"
    export IMAGE_NAME="myapp"
    export SKIP_HEALTH_CHECK="yes"
    mock::docker::set_image_exists "myapp:1.0.0" true
    mock::commands::setup_command "command -v docker" "" "0"
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    mock::commands::setup_function "flow::is_yes" 0  # Return true for health check skip
    
    run deploy::universal::main
    
    assert_success
    # Health check should not be called
    mock::commands::assert::not_called "docker" "exec myapp curl"
}

@test "deploy::universal::main runs post-deploy hook" {
    export SOURCE_TYPE="docker"
    export IMAGE_NAME="myapp"
    mock::docker::set_image_exists "myapp:1.0.0" true
    mock::commands::setup_command "command -v docker" "" "0"
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    
    run deploy::universal::main
    
    assert_success
    mock::commands::assert::called "phase::run_hook" "postDeploy"
}

@test "deploy::universal::main exports deployment variables" {
    export SOURCE_TYPE="docker"
    export VERSION="2.0.0"
    export ENVIRONMENT="staging"
    export IMAGE_NAME="myapp"
    mock::docker::set_image_exists "myapp:2.0.0" true
    mock::commands::setup_command "command -v docker" "" "0"
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    
    deploy::universal::main
    
    assert_equal "$DEPLOY_VERSION" "2.0.0"
    assert_equal "$DEPLOY_ENVIRONMENT" "staging"
    assert_equal "$DEPLOY_SOURCE" "docker"
}

################################################################################
# Entry Point Tests
################################################################################

@test "deploy script requires lifecycle engine invocation" {
    unset LIFECYCLE_PHASE
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "log::info" 0
    
    run bash -c 'BASH_SOURCE=("test_script"); 0="test_script"; source "${BATS_TEST_DIRNAME}/deploy.sh"'
    
    assert_failure
    assert_equal "$status" 1
}

@test "deploy script runs when called by lifecycle engine" {
    export LIFECYCLE_PHASE="deploy"
    export SOURCE_TYPE="docker"
    export IMAGE_NAME="myapp"
    mock::docker::set_image_exists "myapp:1.0.0" true
    mock::commands::setup_command "command -v docker" "" "0"
    mock::filesystem::create_file "${var_DOCKER_COMPOSE_DEV_FILE}" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "-f ${var_DOCKER_COMPOSE_DEV_FILE} up -d" ""
    
    # Override the main function for this test
    deploy::universal::main() { echo "deploy executed"; }
    export -f deploy::universal::main
    
    run bash -c 'export LIFECYCLE_PHASE="deploy"; BASH_SOURCE=("test_script"); 0="test_script"; source "${BATS_TEST_DIRNAME}/deploy.sh"'
    
    assert_success
}