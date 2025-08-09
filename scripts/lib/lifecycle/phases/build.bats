#!/usr/bin/env bats
################################################################################
# Tests for build.sh - Universal build phase handler
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
    export TEST="no"
    export LINT="no"
    export VERSION=""
    export BUNDLES="zip"
    export ARTIFACTS="docker"
    export ENVIRONMENT="development"
    export var_ROOT_DIR="${MOCK_ROOT}"
    export var_SERVICE_JSON_FILE="${MOCK_ROOT}/.vrooli/service.json"
    export var_SCRIPTS_DIR="${MOCK_ROOT}/scripts"
    export var_APP_PACKAGE_DIR="${MOCK_ROOT}/app/package"
    
    # Create mock package.json
    mock::filesystem::create_file "${MOCK_ROOT}/package.json" '{
        "name": "test-app",
        "version": "1.0.0",
        "scripts": {
            "build": "echo building",
            "test": "echo testing",
            "test:unit": "echo unit testing",
            "test:integration": "echo integration testing",
            "lint": "echo linting"
        }
    }'
    
    # Source build functions for testing
    source "${BATS_TEST_DIRNAME}/build.sh"
}

teardown() {
    vrooli_cleanup_test
}

################################################################################
# Test Execution Tests
################################################################################

@test "build::run_tests skips tests when disabled" {
    export TEST="no"
    mock::commands::setup_function "flow::is_yes" 1  # Return false
    
    run build::run_tests
    
    assert_success
    mock::commands::assert::called "log::info" "Tests skipped (use --test yes to enable)"
}

@test "build::run_tests runs pnpm test:unit when available" {
    export TEST="yes"
    mock::commands::setup_function "flow::is_yes" 0  # Return true
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:unit" ""
    
    run build::run_tests
    
    assert_success
    mock::commands::assert::called "pnpm" "test:unit"
    mock::commands::assert::called "log::success" "✅ Tests passed"
}

@test "build::run_tests falls back to pnpm test" {
    export TEST="yes"
    mock::commands::setup_function "flow::is_yes" 0
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:unit" "" "127"  # Command not found
    mock::commands::setup_command "pnpm" "test" ""
    
    run build::run_tests
    
    assert_success
    mock::commands::assert::called "pnpm" "test"
}

@test "build::run_tests uses npm when pnpm unavailable" {
    export TEST="yes"
    mock::commands::setup_function "flow::is_yes" 0
    mock::commands::setup_command "command -v pnpm" "" "1"  # pnpm not found
    mock::commands::setup_command "command -v npm" "" "0"
    mock::commands::setup_command "npm" "test" ""
    
    run build::run_tests
    
    assert_success
    mock::commands::assert::called "npm" "test"
}

@test "build::run_tests tries manage.sh script" {
    export TEST="yes"
    mock::commands::setup_function "flow::is_yes" 0
    mock::commands::setup_command "command -v pnpm" "" "1"
    mock::commands::setup_command "command -v npm" "" "1"
    mock::filesystem::create_file "${var_SCRIPTS_DIR}/manage.sh" "#!/bin/bash\necho 'manage.sh test'"
    mock::commands::setup_command "./scripts/manage.sh" "test" ""
    
    run build::run_tests
    
    assert_success
    mock::commands::assert::called "./scripts/manage.sh" "test"
}

@test "build::run_tests handles test failure" {
    export TEST="yes"
    mock::commands::setup_function "flow::is_yes" 0
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "test:unit" "" "1"  # Test failure
    
    run build::run_tests
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Tests failed"
}

@test "build::run_tests warns when no test command found" {
    export TEST="yes"
    mock::commands::setup_function "flow::is_yes" 0
    mock::commands::setup_command "command -v pnpm" "" "1"
    mock::commands::setup_command "command -v npm" "" "1"
    # No manage.sh script
    
    run build::run_tests
    
    assert_success
    mock::commands::assert::called "log::warning" "No test command found, skipping tests"
}

################################################################################
# Linting Tests
################################################################################

@test "build::run_linting skips linting when disabled" {
    export LINT="no"
    mock::commands::setup_function "flow::is_yes" 1  # Return false
    
    run build::run_linting
    
    assert_success
    mock::commands::assert::called "log::info" "Linting skipped (use --lint yes to enable)"
}

@test "build::run_linting runs pnpm lint when available" {
    export LINT="yes"
    mock::commands::setup_function "flow::is_yes" 0  # Return true
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "lint" ""
    
    run build::run_linting
    
    assert_success
    mock::commands::assert::called "pnpm" "lint"
    mock::commands::assert::called "log::success" "✅ Linting passed"
}

@test "build::run_linting uses npm when pnpm unavailable" {
    export LINT="yes"
    mock::commands::setup_function "flow::is_yes" 0
    mock::commands::setup_command "command -v pnpm" "" "1"
    mock::commands::setup_command "command -v npm" "" "0"
    mock::commands::setup_command "npm" "run lint" ""
    
    run build::run_linting
    
    assert_success
    mock::commands::assert::called "npm" "run lint"
}

@test "build::run_linting handles linting failure" {
    export LINT="yes"
    mock::commands::setup_function "flow::is_yes" 0
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "lint" "" "1"  # Lint failure
    
    run build::run_linting
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Linting failed"
}

################################################################################
# Version Management Tests
################################################################################

@test "build::get_version returns explicit VERSION variable" {
    export VERSION="2.1.0"
    
    run build::get_version
    
    assert_success
    assert_output "2.1.0"
}

@test "build::get_version reads from package.json" {
    unset VERSION
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-r '.version // empty' '${MOCK_ROOT}/package.json'" "1.0.0"
    
    run build::get_version
    
    assert_success
    assert_output "1.0.0"
}

@test "build::get_version reads from service.json" {
    unset VERSION
    mock::filesystem::create_file "${var_SERVICE_JSON_FILE}" '{
        "service": {
            "version": "1.5.0"
        }
    }'
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-r '.version // empty' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "jq" "-r '.service.version // .version // empty' '${var_SERVICE_JSON_FILE}'" "1.5.0"
    
    run build::get_version
    
    assert_success
    assert_output "1.5.0"
}

@test "build::get_version uses git describe" {
    unset VERSION
    mock::filesystem::create_directory "${MOCK_ROOT}/.git"
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-r '.version // empty' '${MOCK_ROOT}/package.json'" ""
    mock::commands::setup_command "jq" "-r '.service.version // .version // empty' '${var_SERVICE_JSON_FILE}'" ""
    mock::commands::setup_command "command -v git" "" "0"
    mock::commands::setup_command "git" "describe --tags --always" "v1.2.3-4-g5678abc"
    
    run build::get_version
    
    assert_success
    assert_output "v1.2.3-4-g5678abc"
}

@test "build::get_version defaults to 'dev'" {
    unset VERSION
    mock::commands::setup_command "command -v jq" "" "1"  # No jq
    mock::commands::setup_command "command -v git" "" "1"  # No git
    
    run build::get_version
    
    assert_success
    assert_output "dev"
}

################################################################################
# Docker Artifact Tests
################################################################################

@test "build::docker_artifacts builds from docker-compose" {
    local compose_file="${MOCK_ROOT}/docker-compose.yaml"
    mock::filesystem::create_file "$compose_file" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "0"
    mock::commands::setup_command "docker-compose" "build" ""
    
    run build::docker_artifacts "1.0.0"
    
    assert_success
    mock::commands::assert::called "docker-compose" "build"
    mock::commands::assert::called "log::success" "✅ Docker artifacts built"
}

@test "build::docker_artifacts uses docker compose when docker-compose unavailable" {
    local compose_file="${MOCK_ROOT}/docker-compose.yaml"
    mock::filesystem::create_file "$compose_file" "version: '3.8'"
    mock::commands::setup_command "command -v docker-compose" "" "1"  # Not found
    mock::commands::setup_command "docker" "compose build" ""
    
    run build::docker_artifacts "1.0.0"
    
    assert_success
    mock::commands::assert::called "docker" "compose build"
}

@test "build::docker_artifacts builds from Dockerfile" {
    mock::filesystem::create_file "${MOCK_ROOT}/Dockerfile" "FROM node:16"
    mock::commands::setup_command "docker" "build -t app:1.0.0 ." ""
    
    run build::docker_artifacts "1.0.0"
    
    assert_success
    mock::commands::assert::called "docker" "build -t app:1.0.0 ."
    mock::commands::assert::called "log::success" "✅ Docker image built: app:1.0.0"
}

@test "build::docker_artifacts warns when no Docker config found" {
    # No compose files or Dockerfile
    
    run build::docker_artifacts "1.0.0"
    
    assert_success
    mock::commands::assert::called "log::warning" "No Docker configuration found"
}

################################################################################
# Kubernetes Artifact Tests
################################################################################

@test "build::k8s_artifacts processes k8s manifests" {
    local k8s_dir="${MOCK_ROOT}/k8s"
    mock::filesystem::create_directory "$k8s_dir"
    mock::filesystem::create_file "$k8s_dir/deployment.yaml" "apiVersion: apps/v1"
    
    run build::k8s_artifacts "1.0.0"
    
    assert_success
    mock::commands::assert::called "log::info" "Found Kubernetes manifests: $k8s_dir"
    mock::commands::assert::called "log::success" "✅ Kubernetes artifacts prepared"
}

@test "build::k8s_artifacts processes templates" {
    local k8s_dir="${MOCK_ROOT}/k8s"
    mock::filesystem::create_directory "$k8s_dir"
    mock::filesystem::create_file "$k8s_dir/deployment.template.yaml" 'image: myapp:$VERSION'
    mock::commands::setup_command "command -v envsubst" "" "0"
    mock::commands::setup_command "envsubst" "" "image: myapp:1.0.0"
    
    run build::k8s_artifacts "1.0.0"
    
    assert_success
    mock::commands::assert::called "envsubst"
}

@test "build::k8s_artifacts warns when no k8s config found" {
    # No k8s directories
    
    run build::k8s_artifacts "1.0.0"
    
    assert_success
    mock::commands::assert::called "log::warning" "No Kubernetes configuration found"
}

################################################################################
# Main Build Function Tests
################################################################################

@test "build::universal::main initializes phase correctly" {
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    
    run build::universal::main
    
    assert_success
    mock::commands::assert::called "phase::init" "Build"
    mock::commands::assert::called "phase::complete"
    assert_equal "$VERSION" "1.0.0"
}

@test "build::universal::main runs pre-build hook" {
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    
    run build::universal::main
    
    assert_success
    mock::commands::assert::called "phase::run_hook" "preBuild"
}

@test "build::universal::main aborts on test failure when enabled" {
    export TEST="yes"
    mock::commands::setup_function "flow::is_yes" 0  # Return true for test check
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    mock::commands::setup_function "build::run_tests" 1  # Test failure
    
    run build::universal::main
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Build aborted due to test failures"
}

@test "build::universal::main aborts on linting failure when enabled" {
    export LINT="yes"
    mock::commands::setup_function "flow::is_yes" 0 0 1  # Test pass, lint enabled
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    mock::commands::setup_function "build::run_tests" 0
    mock::commands::setup_function "build::run_linting" 1  # Lint failure
    
    run build::universal::main
    
    assert_failure
    assert_equal "$status" 1
    mock::commands::assert::called "log::error" "Build aborted due to linting failures"
}

@test "build::universal::main builds with pnpm when available" {
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    mock::commands::setup_command "command -v jq" "" "1"  # No service.json steps
    mock::commands::setup_command "command -v pnpm" "" "0"
    mock::commands::setup_command "pnpm" "build" ""
    
    run build::universal::main
    
    assert_success
    mock::commands::assert::called "pnpm" "build"
}

@test "build::universal::main builds with npm when pnpm unavailable" {
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    mock::commands::setup_command "command -v jq" "" "1"  # No service.json steps
    mock::commands::setup_command "command -v pnpm" "" "1"  # pnpm not found
    mock::commands::setup_command "command -v npm" "" "0"
    mock::commands::setup_command "npm" "run build" ""
    
    run build::universal::main
    
    assert_success
    mock::commands::assert::called "npm" "run build"
}

@test "build::universal::main skips generic build when service.json has steps" {
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    mock::filesystem::create_file "${var_SERVICE_JSON_FILE}" '{
        "lifecycle": {
            "build": {
                "steps": [
                    {"name": "custom", "command": "echo custom build"}
                ]
            }
        }
    }'
    mock::commands::setup_command "command -v jq" "" "0"
    mock::commands::setup_command "jq" "-r '.lifecycle.build.steps // [] | length' '${var_SERVICE_JSON_FILE}'" "1"
    
    run build::universal::main
    
    assert_success
    mock::commands::assert::called "log::info" "Build steps will be executed from service.json"
    mock::commands::assert::not_called "pnpm"
    mock::commands::assert::not_called "npm"
}

@test "build::universal::main builds artifacts" {
    export ARTIFACTS="docker,k8s"
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    mock::commands::setup_function "build::docker_artifacts" 0
    mock::commands::setup_function "build::k8s_artifacts" 0
    
    run build::universal::main
    
    assert_success
    mock::commands::assert::called "build::docker_artifacts" "1.0.0"
    mock::commands::assert::called "build::k8s_artifacts" "1.0.0"
}

@test "build::universal::main creates bundles" {
    export BUNDLES="zip"
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    mock::filesystem::create_file "${var_APP_PACKAGE_DIR}/bundle.sh" "#!/bin/bash\necho bundling"
    mock::commands::setup_command "bash" "${var_APP_PACKAGE_DIR}/bundle.sh" ""
    
    run build::universal::main
    
    assert_success
    mock::commands::assert::called "bash" "${var_APP_PACKAGE_DIR}/bundle.sh"
}

@test "build::universal::main runs post-build hook" {
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    
    run build::universal::main
    
    assert_success
    mock::commands::assert::called "phase::run_hook" "postBuild"
}

@test "build::universal::main exports build variables" {
    export ARTIFACTS="docker"
    export BUNDLES="zip"
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    
    build::universal::main
    
    assert_equal "$BUILD_VERSION" "1.0.0"
    assert_equal "$BUILD_ARTIFACTS" "docker"
    assert_equal "$BUILD_BUNDLES" "zip"
}

################################################################################
# Entry Point Tests
################################################################################

@test "build script requires lifecycle engine invocation" {
    unset LIFECYCLE_PHASE
    mock::commands::setup_function "log::error" 0
    mock::commands::setup_function "log::info" 0
    
    run bash -c 'BASH_SOURCE=("test_script"); 0="test_script"; source "${BATS_TEST_DIRNAME}/build.sh"'
    
    assert_failure
    assert_equal "$status" 1
}

@test "build script runs when called by lifecycle engine" {
    export LIFECYCLE_PHASE="build"
    mock::commands::setup_function "build::get_version" 0 "1.0.0"
    
    # Override the main function for this test
    build::universal::main() { echo "build executed"; }
    export -f build::universal::main
    
    run bash -c 'export LIFECYCLE_PHASE="build"; BASH_SOURCE=("test_script"); 0="test_script"; source "${BATS_TEST_DIRNAME}/build.sh"'
    
    assert_success
}