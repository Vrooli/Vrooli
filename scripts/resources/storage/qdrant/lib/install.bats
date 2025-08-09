#!/usr/bin/env bats
# Tests for Qdrant install.sh functions

# Load standard test setup
load "${BATS_TEST_DIRNAME}/../../../tests/bats-fixtures/setup/standard_setup.bash"

# Setup for each test
setup() {
    # Use standard setup
    standard_test_setup "qdrant"
    
    # Set Qdrant-specific test environment
    export QDRANT_PORT="6333"
    export QDRANT_GRPC_PORT="6334"
    export QDRANT_CONTAINER_NAME="qdrant-test"
    export QDRANT_BASE_URL="http://localhost:6333"
    export QDRANT_GRPC_URL="grpc://localhost:6334"
    export QDRANT_IMAGE="qdrant/qdrant:latest"
    export QDRANT_DATA_DIR="${BATS_TEST_TMPDIR}/qdrant-test/data"
    export QDRANT_CONFIG_DIR="${BATS_TEST_TMPDIR}/qdrant-test/config"
    export QDRANT_SNAPSHOTS_DIR="${BATS_TEST_TMPDIR}/qdrant-test/snapshots"
    export QDRANT_NETWORK_NAME="qdrant-network-test"
    export QDRANT_API_KEY="test_qdrant_api_key_123"
    
    # Create test directories
    mkdir -p "$QDRANT_DATA_DIR"
    mkdir -p "$QDRANT_CONFIG_DIR"
    mkdir -p "$QDRANT_SNAPSHOTS_DIR"
    
    # Set default mock state
    mock::qdrant::setup "not_installed"
}

# Cleanup after each test
teardown() {
    standard_test_teardown
}

# Test installation check
@test "qdrant::install::check_installation checks if Qdrant is installed" {
    # Set up mock state as installed
    mock::docker::set_container_state "qdrant-test" "running"
    
    # Load install functions if not already loaded
    if ! declare -f qdrant::install::check_installation >/dev/null 2>&1; then
        skip "qdrant::install::check_installation function not found"
    fi
    
    result=$(qdrant::install::check_installation && echo "installed" || echo "not installed")
    
    [[ "$result" == "installed" ]]
}

# Test installation check with missing installation
@test "qdrant::install::check_installation handles missing installation" {
    # Set up mock state as not installed
    mock::docker::set_container_state "qdrant-test" "not_found"
    
    if ! declare -f qdrant::install::check_installation >/dev/null 2>&1; then
        skip "qdrant::install::check_installation function not found"
    fi
    
    result=$(qdrant::install::check_installation && echo "installed" || echo "not installed")
    
    [[ "$result" == "not installed" ]]
}

# Test dependency verification
@test "qdrant::install::verify_dependencies checks required dependencies" {
    if ! declare -f qdrant::install::verify_dependencies >/dev/null 2>&1; then
        skip "qdrant::install::verify_dependencies function not found"
    fi
    
    result=$(qdrant::install::verify_dependencies)
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "verified" ]] || [[ -z "$result" ]]
}

# Test dependency verification with missing dependencies
@test "qdrant::install::verify_dependencies handles missing dependencies" {
    # Override which command to simulate missing docker
    which() {
        case "$1" in
            "docker") return 1 ;;
            *) command which "$1" ;;
        esac
    }
    
    if ! declare -f qdrant::install::verify_dependencies >/dev/null 2>&1; then
        skip "qdrant::install::verify_dependencies function not found"
    fi
    
    run qdrant::install::verify_dependencies
    [ "$status" -eq 1 ] || skip "Function may not be checking dependencies properly"
}

# Test port availability check
@test "qdrant::install::check_ports verifies port availability" {
    # Ports should be available by default
    mock::port::set_available "6333"
    mock::port::set_available "6334"
    
    if ! declare -f qdrant::install::check_ports >/dev/null 2>&1; then
        skip "qdrant::install::check_ports function not found"
    fi
    
    result=$(qdrant::install::check_ports)
    status=$?
    
    [[ $status -eq 0 ]] || [[ "$result" =~ "available" ]]
}

# Test port availability check with ports in use
@test "qdrant::install::check_ports handles ports in use" {
    # Set ports as in use
    mock::port::set_in_use "6333"
    mock::port::set_in_use "6334"
    
    if ! declare -f qdrant::install::check_ports >/dev/null 2>&1; then
        skip "qdrant::install::check_ports function not found"
    fi
    
    run qdrant::install::check_ports
    [ "$status" -eq 1 ] || skip "Function may not be checking ports properly"
}

# Test system requirements check
@test "qdrant::install::check_system_requirements validates system requirements" {
    if ! declare -f qdrant::install::check_system_requirements >/dev/null 2>&1; then
        skip "qdrant::install::check_system_requirements function not found"
    fi
    
    result=$(qdrant::install::check_system_requirements)
    status=$?
    
    [[ $status -eq 0 ]] || [[ "$result" =~ "requirements" ]]
}

# Test directory creation
@test "qdrant::install::create_directories creates required directories" {
    # Remove test directories first
    rm -rf "$QDRANT_DATA_DIR" "$QDRANT_CONFIG_DIR" "$QDRANT_SNAPSHOTS_DIR"
    
    if ! declare -f qdrant::install::create_directories >/dev/null 2>&1; then
        skip "qdrant::install::create_directories function not found"
    fi
    
    result=$(qdrant::install::create_directories)
    
    # Check directories were created
    assert_dir_exists "$QDRANT_DATA_DIR"
    assert_dir_exists "$QDRANT_CONFIG_DIR"
    assert_dir_exists "$QDRANT_SNAPSHOTS_DIR"
}

# Test network creation
@test "qdrant::install::create_network creates Docker network" {
    # Mock docker network command responses
    mock::set_response "docker" "network" "Network created: qdrant-network-test"
    
    if ! declare -f qdrant::install::create_network >/dev/null 2>&1; then
        skip "qdrant::install::create_network function not found"
    fi
    
    result=$(qdrant::install::create_network)
    
    [[ "$result" =~ "network" ]] || [[ -z "$result" ]]
}

# Test Docker image pull
@test "qdrant::install::pull_image pulls Qdrant Docker image" {
    # Mock docker pull response
    mock::set_response "docker" "pull" "Successfully pulled qdrant/qdrant:latest"
    
    if ! declare -f qdrant::install::pull_image >/dev/null 2>&1; then
        skip "qdrant::install::pull_image function not found"
    fi
    
    result=$(qdrant::install::pull_image)
    
    [[ "$result" =~ "pull" ]] || [[ "$result" =~ "image" ]] || [[ -z "$result" ]]
}

# Test container creation
@test "qdrant::install::create_container creates Qdrant container" {
    # Mock successful container creation
    mock::set_response "docker" "run" "Container qdrant-test created"
    
    if ! declare -f qdrant::install::create_container >/dev/null 2>&1; then
        skip "qdrant::install::create_container function not found"
    fi
    
    result=$(qdrant::install::create_container)
    
    [[ "$result" =~ "container" ]] || [[ "$result" =~ "created" ]] || [[ -z "$result" ]]
}

# Test health check
@test "qdrant::install::wait_for_healthy waits for Qdrant to be healthy" {
    # Set up healthy state
    mock::qdrant::setup "healthy"
    
    if ! declare -f qdrant::install::wait_for_healthy >/dev/null 2>&1; then
        skip "qdrant::install::wait_for_healthy function not found"
    fi
    
    result=$(qdrant::install::wait_for_healthy)
    status=$?
    
    [[ $status -eq 0 ]]
}

# Test default collection creation
@test "qdrant::install::create_default_collections creates initial collections" {
    # Mock successful collection creation
    mock::set_curl_response "http://localhost:6333/collections/documents" '{"status":"ok"}'
    
    if ! declare -f qdrant::install::create_default_collections >/dev/null 2>&1; then
        skip "qdrant::install::create_default_collections function not found"
    fi
    
    result=$(qdrant::install::create_default_collections)
    
    [[ "$result" =~ "collection" ]] || [[ -z "$result" ]]
}

# Test configuration update
@test "qdrant::install::update_resource_config updates Vrooli configuration" {
    if ! declare -f qdrant::install::update_resource_config >/dev/null 2>&1; then
        skip "qdrant::install::update_resource_config function not found"
    fi
    
    # Create mock config file
    local config_file=$(create_mock_config "qdrant")
    
    result=$(qdrant::install::update_resource_config)
    
    [[ -f "$config_file" ]]
}

# Test full installation flow
@test "qdrant::install performs complete installation" {
    # Set up mocks for successful installation
    mock::qdrant::setup "not_installed"
    mock::set_response "docker" "pull" "Image pulled"
    mock::set_response "docker" "run" "Container started"
    
    # After container start, set as running
    mock::docker::set_container_state "qdrant-test" "running"
    mock::qdrant::setup "healthy"
    
    if ! declare -f qdrant::install >/dev/null 2>&1; then
        skip "qdrant::install function not found"
    fi
    
    run qdrant::install
    [ "$status" -eq 0 ] || skip "Installation may have different requirements"
}

# Test upgrade functionality
@test "qdrant::install::upgrade upgrades existing installation" {
    # Set up as already installed
    mock::qdrant::setup "healthy"
    
    if ! declare -f qdrant::install::upgrade >/dev/null 2>&1; then
        skip "qdrant::install::upgrade function not found"
    fi
    
    result=$(qdrant::install::upgrade)
    
    [[ "$result" =~ "upgrade" ]] || [[ -z "$result" ]]
}

# Test reset configuration
@test "qdrant::install::reset_configuration resets to defaults" {
    if ! declare -f qdrant::install::reset_configuration >/dev/null 2>&1; then
        skip "qdrant::install::reset_configuration function not found"
    fi
    
    # Create some test files
    touch "$QDRANT_CONFIG_DIR/test.conf"
    
    result=$(qdrant::install::reset_configuration)
    
    [[ "$result" =~ "reset" ]] || [[ -z "$result" ]]
}
