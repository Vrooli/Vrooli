#!/usr/bin/env bats
# Tests for RESOURCE_NAME lib/install.sh installation functions
#
# Template Usage:
# 1. Copy this file to RESOURCE_NAME/lib/install.bats
# 2. Replace RESOURCE_NAME with your resource name (e.g., "ollama", "n8n")
# 3. Replace INSTALL_FUNCTIONS with your actual installation function names
# 4. Implement resource-specific installation tests
# 5. Remove this header comment block

bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export RESOURCE_NAME_PORT="8080"  # Replace with actual port
    export RESOURCE_NAME_CONTAINER_NAME="RESOURCE_NAME-test"
    export RESOURCE_NAME_IMAGE="RESOURCE_NAME/RESOURCE_NAME:latest"  # Replace with actual image
    export RESOURCE_NAME_DATA_DIR="/tmp/RESOURCE_NAME-test/data"
    export RESOURCE_NAME_CONFIG_DIR="/tmp/RESOURCE_NAME-test/config"
    
    # Installation control flags
    export FORCE="no"
    export YES="no"
    export DRY_RUN="no"
    
    # Get resource directory path
    RESOURCE_NAME_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    
    # Create test directories
    mkdir -p "$RESOURCE_NAME_DATA_DIR"
    mkdir -p "$RESOURCE_NAME_CONFIG_DIR"
    
    # Configure installation mocks
    mock::docker::set_pull_success "RESOURCE_NAME/RESOURCE_NAME:latest"
    mock::docker::set_container_state "RESOURCE_NAME-test" "missing"
    
    # Load configuration and the functions to test
    source "${RESOURCE_NAME_DIR}/config/defaults.sh"
    source "${RESOURCE_NAME_DIR}/config/messages.sh"
    RESOURCE_NAME::export_config
    RESOURCE_NAME::messages::init
    
    # Load the installation functions
    source "${RESOURCE_NAME_DIR}/lib/install.sh"
}

teardown() {
    # Clean up test environment
    cleanup_mocks
    rm -rf "/tmp/RESOURCE_NAME-test"
}

# Test installation prerequisites

@test "RESOURCE_NAME::check_prerequisites should verify Docker availability" {
    run RESOURCE_NAME::check_prerequisites
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::check_prerequisites should detect missing Docker" {
    # Mock missing Docker
    docker() { return 127; }
    export -f docker
    
    run RESOURCE_NAME::check_prerequisites
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Docker" || "$output" =~ "prerequisite" ]]
}

@test "RESOURCE_NAME::check_prerequisites should verify sufficient disk space" {
    # Mock sufficient disk space
    df() { echo "Filesystem     1K-blocks    Used Available Use% Mounted on"; echo "/dev/sda1      100000000 5000000  95000000   5% /"; }
    export -f df
    
    run RESOURCE_NAME::check_prerequisites
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::check_prerequisites should detect insufficient disk space" {
    # Mock insufficient disk space
    df() { echo "Filesystem     1K-blocks    Used Available Use% Mounted on"; echo "/dev/sda1       1000000  900000     100000  90% /"; }
    export -f df
    
    run RESOURCE_NAME::check_prerequisites
    [ "$status" -ne 0 ]
    [[ "$output" =~ "disk space" || "$output" =~ "insufficient" ]]
}

@test "RESOURCE_NAME::check_prerequisites should verify port availability" {
    # Mock port available
    netstat() { echo "No output"; }
    export -f netstat
    
    run RESOURCE_NAME::check_prerequisites
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::check_prerequisites should detect port conflicts" {
    # Mock port in use
    netstat() { echo "tcp        0      0 0.0.0.0:8080            0.0.0.0:*               LISTEN"; }
    export -f netstat
    
    run RESOURCE_NAME::check_prerequisites
    [ "$status" -ne 0 ]
    [[ "$output" =~ "port" || "$output" =~ "conflict" || "$output" =~ "8080" ]]
}

# Test installation process

@test "RESOURCE_NAME::install should perform full installation" {
    export YES="yes"
    
    run RESOURCE_NAME::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "installed" || "$output" =~ "complete" ]]
}

@test "RESOURCE_NAME::install should require confirmation when YES=no" {
    export YES="no"
    
    # Mock user declining installation
    read() { return 1; }
    export -f read
    
    run RESOURCE_NAME::install
    [ "$status" -ne 0 ]
    [[ "$output" =~ "cancelled" || "$output" =~ "aborted" ]]
}

@test "RESOURCE_NAME::install should respect dry-run mode" {
    export DRY_RUN="yes"
    export YES="yes"
    
    run RESOURCE_NAME::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "DRY RUN" || "$output" =~ "Would" ]]
    [[ "$output" =~ "docker pull" ]]
    [[ "$output" =~ "docker run" ]]
}

@test "RESOURCE_NAME::install should handle existing installation" {
    # Mock existing container
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    export FORCE="no"
    
    run RESOURCE_NAME::install
    [ "$status" -ne 0 ]
    [[ "$output" =~ "already installed" || "$output" =~ "exists" ]]
}

@test "RESOURCE_NAME::install should handle force reinstallation" {
    # Mock existing container
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    export FORCE="yes"
    export YES="yes"
    
    run RESOURCE_NAME::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "reinstalling" || "$output" =~ "force" ]]
}

# Test Docker image management

@test "RESOURCE_NAME::pull_image should download Docker image" {
    run RESOURCE_NAME::pull_image
    [ "$status" -eq 0 ]
    [[ "$output" =~ "pull" || "$output" =~ "download" ]]
}

@test "RESOURCE_NAME::pull_image should handle image pull failure" {
    # Mock failed image pull
    mock::docker::set_pull_failure "RESOURCE_NAME/RESOURCE_NAME:latest" "network error"
    
    run RESOURCE_NAME::pull_image
    [ "$status" -ne 0 ]
    [[ "$output" =~ "failed" || "$output" =~ "error" ]]
}

@test "RESOURCE_NAME::pull_image should verify image integrity" {
    # Mock successful pull with verification
    mock::docker::set_pull_success "RESOURCE_NAME/RESOURCE_NAME:latest"
    mock::docker::set_image_exists "RESOURCE_NAME/RESOURCE_NAME:latest" "true"
    
    run RESOURCE_NAME::pull_image
    [ "$status" -eq 0 ]
    
    # Verify image is available
    run RESOURCE_NAME::verify_image
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::get_latest_version should fetch latest version info" {
    # Mock version API response (customize based on your resource)
    curl() {
        case "$*" in
            *"api.github.com"*) echo '{"tag_name": "v1.2.3"}' ;;
            *"registry.hub.docker.com"*) echo '{"name": "1.2.3"}' ;;
            *) echo "Unknown API" ;;
        esac
    }
    export -f curl
    
    run RESOURCE_NAME::get_latest_version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "1.2.3" ]]
}

# Test container creation

@test "RESOURCE_NAME::create_container should create Docker container" {
    run RESOURCE_NAME::create_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "created" || "$output" =~ "container" ]]
}

@test "RESOURCE_NAME::create_container should use correct configuration" {
    run RESOURCE_NAME::create_container
    [ "$status" -eq 0 ]
    
    # Verify correct parameters were used (customize based on your resource)
    # This checks that mock Docker was called with expected parameters
    grep -q "\-p 8080:8080" "$MOCK_RESPONSES_DIR/docker_commands.log" || fail "Port mapping not found"
    grep -q "\-v.*data" "$MOCK_RESPONSES_DIR/docker_commands.log" || fail "Data volume not found"
}

@test "RESOURCE_NAME::create_container should handle creation failure" {
    # Mock container creation failure
    mock::docker::set_run_failure "RESOURCE_NAME/RESOURCE_NAME:latest" "insufficient resources"
    
    run RESOURCE_NAME::create_container
    [ "$status" -ne 0 ]
    [[ "$output" =~ "failed" || "$output" =~ "error" ]]
}

# Test configuration setup

@test "RESOURCE_NAME::setup_configuration should create config files" {
    run RESOURCE_NAME::setup_configuration
    [ "$status" -eq 0 ]
    [ -d "$RESOURCE_NAME_CONFIG_DIR" ]
}

@test "RESOURCE_NAME::setup_configuration should use provided configuration" {
    # Create custom config
    cat > "/tmp/custom-config.yml" << EOF
port: 9090
debug: true
EOF
    export RESOURCE_NAME_CONFIG_FILE="/tmp/custom-config.yml"
    
    run RESOURCE_NAME::setup_configuration
    [ "$status" -eq 0 ]
    [ -f "$RESOURCE_NAME_CONFIG_DIR/config.yml" ]
    
    # Verify config was copied
    grep -q "port: 9090" "$RESOURCE_NAME_CONFIG_DIR/config.yml"
    
    # Cleanup
    rm -f "/tmp/custom-config.yml"
}

@test "RESOURCE_NAME::setup_configuration should create default config" {
    run RESOURCE_NAME::setup_configuration
    [ "$status" -eq 0 ]
    [ -f "$RESOURCE_NAME_CONFIG_DIR/config.yml" ]
    
    # Verify default values (customize based on your resource)
    grep -q "port: 8080" "$RESOURCE_NAME_CONFIG_DIR/config.yml"
}

# Test data directory setup

@test "RESOURCE_NAME::setup_data_directory should create data directories" {
    run RESOURCE_NAME::setup_data_directory
    [ "$status" -eq 0 ]
    [ -d "$RESOURCE_NAME_DATA_DIR" ]
}

@test "RESOURCE_NAME::setup_data_directory should set correct permissions" {
    run RESOURCE_NAME::setup_data_directory
    [ "$status" -eq 0 ]
    
    # Check directory permissions (customize as needed)
    [ -r "$RESOURCE_NAME_DATA_DIR" ]
    [ -w "$RESOURCE_NAME_DATA_DIR" ]
    [ -x "$RESOURCE_NAME_DATA_DIR" ]
}

@test "RESOURCE_NAME::setup_data_directory should handle permission errors" {
    # Mock permission denied
    mkdir() { return 1; }
    export -f mkdir
    
    run RESOURCE_NAME::setup_data_directory
    [ "$status" -ne 0 ]
    [[ "$output" =~ "permission" || "$output" =~ "failed" ]]
}

# Test post-installation steps

@test "RESOURCE_NAME::post_install should complete setup" {
    # Mock successful container creation
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    mock::http::set_endpoint_response "http://localhost:8080/health" "200" "healthy"
    
    run RESOURCE_NAME::post_install
    [ "$status" -eq 0 ]
}

@test "RESOURCE_NAME::post_install should wait for service ready" {
    # Mock service becoming ready
    mock::http::set_endpoint_sequence "http://localhost:8080/health" "503,503,200" "starting,starting,healthy"
    
    run timeout 10 RESOURCE_NAME::post_install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ready" || "$output" =~ "started" ]]
}

@test "RESOURCE_NAME::post_install should handle startup timeout" {
    # Mock service never becoming ready
    mock::http::set_endpoint_response "http://localhost:8080/health" "503" "starting"
    
    run timeout 5 RESOURCE_NAME::post_install
    [ "$status" -ne 0 ]
    [[ "$output" =~ "timeout" || "$output" =~ "failed to start" ]]
}

# Test resource-specific installation

@test "RESOURCE_NAME specific installation tasks" {
    # Example for AI resources:
    # run RESOURCE_NAME::download_models
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "models downloaded" ]]
    
    # Example for automation resources:
    # run RESOURCE_NAME::import_workflows
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "workflows imported" ]]
    
    # Example for storage resources:
    # run RESOURCE_NAME::initialize_storage
    # [ "$status" -eq 0 ]
    # [[ "$output" =~ "storage initialized" ]]
    
    # Replace this with actual resource-specific tests
    skip "Implement resource-specific installation tests"
}

# Test installation validation

@test "RESOURCE_NAME::validate_installation should verify complete installation" {
    # Mock successful installation
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    mock::http::set_endpoint_response "http://localhost:8080/health" "200" "healthy"
    
    run RESOURCE_NAME::validate_installation
    [ "$status" -eq 0 ]
    [[ "$output" =~ "valid" || "$output" =~ "successful" ]]
}

@test "RESOURCE_NAME::validate_installation should detect incomplete installation" {
    # Mock incomplete installation (container exists but not running)
    mock::docker::set_container_state "RESOURCE_NAME-test" "stopped"
    
    run RESOURCE_NAME::validate_installation
    [ "$status" -ne 0 ]
    [[ "$output" =~ "incomplete" || "$output" =~ "failed" ]]
}

@test "RESOURCE_NAME::validate_installation should check all components" {
    # Mock complete installation
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    mock::http::set_endpoint_response "http://localhost:8080/health" "200" "healthy"
    
    run RESOURCE_NAME::validate_installation
    [ "$status" -eq 0 ]
    
    # Should check multiple aspects (customize based on your resource)
    [[ "$output" =~ "container" ]]
    [[ "$output" =~ "health" ]]
    [[ "$output" =~ "configuration" ]]
}

# Test uninstallation

@test "RESOURCE_NAME::uninstall should remove installation" {
    # Mock existing installation
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    export FORCE="yes"
    export YES="yes"
    
    run RESOURCE_NAME::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "removed" || "$output" =~ "uninstalled" ]]
}

@test "RESOURCE_NAME::uninstall should require confirmation" {
    export FORCE="no"
    export YES="no"
    
    # Mock user declining uninstallation
    read() { return 1; }
    export -f read
    
    run RESOURCE_NAME::uninstall
    [ "$status" -ne 0 ]
    [[ "$output" =~ "cancelled" || "$output" =~ "aborted" ]]
}

@test "RESOURCE_NAME::uninstall should preserve data when requested" {
    export PRESERVE_DATA="yes"
    export FORCE="yes"
    export YES="yes"
    
    run RESOURCE_NAME::uninstall
    [ "$status" -eq 0 ]
    [ -d "$RESOURCE_NAME_DATA_DIR" ]
    [[ "$output" =~ "preserved" || "$output" =~ "keeping data" ]]
}

# Test upgrade process

@test "RESOURCE_NAME::upgrade should update to latest version" {
    # Mock existing installation with older version
    mock::docker::set_container_state "RESOURCE_NAME-test" "running"
    mock::docker::set_image_version "RESOURCE_NAME/RESOURCE_NAME:latest" "1.0.0"
    
    export YES="yes"
    run RESOURCE_NAME::upgrade
    [ "$status" -eq 0 ]
    [[ "$output" =~ "upgraded" || "$output" =~ "updated" ]]
}

@test "RESOURCE_NAME::upgrade should handle backup before upgrade" {
    export BACKUP_BEFORE_UPGRADE="yes"
    export YES="yes"
    
    run RESOURCE_NAME::upgrade
    [ "$status" -eq 0 ]
    [[ "$output" =~ "backup" ]]
    [ -d "$RESOURCE_NAME_DATA_DIR/backup" ]
}

# Add more resource-specific installation tests here:

# Example templates for different resource types:

# For AI resources:
# @test "RESOURCE_NAME::install should download required models" { ... }
# @test "RESOURCE_NAME::install should configure GPU support" { ... }

# For automation resources:
# @test "RESOURCE_NAME::install should setup database" { ... }
# @test "RESOURCE_NAME::install should configure authentication" { ... }

# For storage resources:
# @test "RESOURCE_NAME::install should initialize storage volumes" { ... }
# @test "RESOURCE_NAME::install should setup backup configuration" { ... }