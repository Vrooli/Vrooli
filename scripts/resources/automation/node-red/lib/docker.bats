#!/usr/bin/env bats
# Tests for Node-RED Docker functions (lib/docker.sh)

source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

setup() {
    setup_test_environment
    source_node_red_scripts
    mock_docker "success"
    mock_curl "success"
    mock_jq "success"
}

teardown() {
    teardown_test_environment
}

@test "node_red::build_custom_image succeeds with valid Dockerfile" {
    # Create a mock Dockerfile
    mkdir -p "$SCRIPT_DIR/docker"
    cat > "$SCRIPT_DIR/docker/Dockerfile" << 'EOF'
FROM nodered/node-red:latest
USER root
RUN mkdir -p /test
USER node-red
EOF
    
    run node_red::build_custom_image
    assert_success
    assert_output_contains "built successfully"
}

@test "node_red::build_custom_image fails with missing Dockerfile" {
    run node_red::build_custom_image
    assert_failure
    assert_output_contains "Dockerfile not found"
}

@test "node_red::build_custom_image fails when docker build fails" {
    # Create Dockerfile but mock docker build to fail
    mkdir -p "$SCRIPT_DIR/docker"
    echo "FROM nodered/node-red:latest" > "$SCRIPT_DIR/docker/Dockerfile"
    
    # Mock docker build to fail
    docker() {
        if [[ "$1" == "build" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::build_custom_image
    assert_failure
    assert_output_contains "Failed to build"
}

@test "node_red::create_network creates network successfully" {
    run node_red::create_network
    assert_success
}

@test "node_red::create_network handles existing network gracefully" {
    # Mock network inspect to succeed (network exists)
    docker() {
        if [[ "$1" == "network" && "$2" == "inspect" ]]; then
            return 0
        fi
        return 0
    }
    export -f docker
    
    run node_red::create_network
    assert_success
}

@test "node_red::create_network fails when docker network create fails" {
    # Mock network inspect to fail (network doesn't exist) and create to fail
    docker() {
        if [[ "$1" == "network" && "$2" == "inspect" ]]; then
            return 1
        elif [[ "$1" == "network" && "$2" == "create" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::create_network
    assert_failure
}

@test "node_red::create_volume creates volume successfully" {
    run node_red::create_volume
    assert_success
}

@test "node_red::create_volume handles existing volume gracefully" {
    # Mock volume inspect to succeed (volume exists)
    docker() {
        if [[ "$1" == "volume" && "$2" == "inspect" ]]; then
            return 0
        fi
        return 0
    }
    export -f docker
    
    run node_red::create_volume
    assert_success
}

@test "node_red::build_docker_command creates proper run command with custom image" {
    export BUILD_IMAGE="yes"
    
    run node_red::build_docker_command
    assert_success
    assert_output_contains "docker run"
    assert_output_contains "$IMAGE_NAME"
    assert_output_contains "$CONTAINER_NAME"
    assert_output_contains "$RESOURCE_PORT:1880"
}

@test "node_red::build_docker_command creates proper run command with official image" {
    export BUILD_IMAGE="no"
    
    run node_red::build_docker_command
    assert_success
    assert_output_contains "docker run"
    assert_output_contains "nodered/node-red:latest"
}

@test "node_red::start_container succeeds with valid configuration" {
    export BUILD_IMAGE="no"
    
    run node_red::start_container
    assert_success
}

@test "node_red::start_container builds custom image when requested" {
    export BUILD_IMAGE="yes"
    
    # Create mock Dockerfile
    mkdir -p "$SCRIPT_DIR/docker"
    echo "FROM nodered/node-red:latest" > "$SCRIPT_DIR/docker/Dockerfile"
    
    run node_red::start_container
    assert_success
}

@test "node_red::start_container fails when docker run fails" {
    # Mock docker run to fail
    docker() {
        if [[ "$1" == "run" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::start_container
    assert_failure
}

@test "node_red::stop_container succeeds when container is running" {
    mock_docker "success"
    
    run node_red::stop_container
    assert_success
}

@test "node_red::stop_container handles non-running container gracefully" {
    mock_docker "not_running"
    
    run node_red::stop_container
    assert_success
}

@test "node_red::stop_container handles non-existent container gracefully" {
    mock_docker "not_installed"
    
    run node_red::stop_container
    assert_success
}

@test "node_red::start_existing_container succeeds" {
    mock_docker "success"
    
    run node_red::start_existing_container
    assert_success
}

@test "node_red::start_existing_container fails when container doesn't exist" {
    mock_docker "not_installed"
    
    run node_red::start_existing_container
    assert_failure
}

@test "node_red::restart_container succeeds" {
    mock_docker "success"
    
    run node_red::restart_container
    assert_success
}

@test "node_red::restart_container handles non-existent container" {
    mock_docker "not_installed"
    
    run node_red::restart_container
    assert_failure
}

@test "node_red::remove_container removes container successfully" {
    mock_docker "success"
    
    run node_red::remove_container
    assert_success
}

@test "node_red::remove_container handles non-existent container gracefully" {
    mock_docker "not_installed"
    
    run node_red::remove_container
    assert_success
}

@test "node_red::remove_volume removes volume successfully" {
    # Mock volume inspect to succeed (volume exists)
    docker() {
        if [[ "$1" == "volume" && "$2" == "inspect" ]]; then
            return 0
        fi
        return 0
    }
    export -f docker
    
    run node_red::remove_volume
    assert_success
}

@test "node_red::remove_volume handles non-existent volume gracefully" {
    # Mock volume inspect to fail (volume doesn't exist)
    docker() {
        if [[ "$1" == "volume" && "$2" == "inspect" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::remove_volume
    assert_success
}

@test "node_red::remove_custom_image removes image successfully" {
    # Mock image inspect to succeed (image exists)
    docker() {
        if [[ "$1" == "image" && "$2" == "inspect" ]]; then
            return 0
        fi
        return 0
    }
    export -f docker
    
    run node_red::remove_custom_image
    assert_success
}

@test "node_red::remove_custom_image handles non-existent image gracefully" {
    # Mock image inspect to fail (image doesn't exist)
    docker() {
        if [[ "$1" == "image" && "$2" == "inspect" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::remove_custom_image
    assert_success
}

@test "node_red::get_container_info returns container information" {
    mock_docker "success"
    
    run node_red::get_container_info
    assert_success
}

@test "node_red::get_container_info fails when container doesn't exist" {
    mock_docker "not_installed"
    
    run node_red::get_container_info
    assert_failure
}

@test "node_red::custom_image_exists returns true when image exists" {
    # Mock image inspect to succeed
    docker() {
        if [[ "$1" == "image" && "$2" == "inspect" ]]; then
            return 0
        fi
        return 0
    }
    export -f docker
    
    run node_red::custom_image_exists
    assert_success
}

@test "node_red::custom_image_exists returns false when image doesn't exist" {
    # Mock image inspect to fail
    docker() {
        if [[ "$1" == "image" && "$2" == "inspect" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::custom_image_exists
    assert_failure
}

@test "node_red::pull_official_image pulls image successfully" {
    run node_red::pull_official_image
    assert_success
}

@test "node_red::pull_official_image fails when docker pull fails" {
    # Mock docker pull to fail
    docker() {
        if [[ "$1" == "pull" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::pull_official_image
    assert_failure
}

@test "node_red::validate_docker_setup succeeds when docker is available" {
    run node_red::validate_docker_setup
    assert_success
}

@test "node_red::validate_docker_setup fails when docker is not available" {
    # Mock docker to not exist
    docker() { return 127; }
    export -f docker
    
    run node_red::validate_docker_setup
    assert_failure
    assert_output_contains "Docker is not installed"
}

@test "node_red::validate_docker_setup fails when docker daemon is not running" {
    # Mock docker ps to fail
    docker() {
        if [[ "$1" == "ps" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::validate_docker_setup
    assert_failure
    assert_output_contains "Docker daemon is not running"
}

@test "node_red::cleanup_docker_resources removes all resources" {
    mock_docker "success"
    
    run node_red::cleanup_docker_resources
    assert_success
}

@test "node_red::cleanup_docker_resources handles missing resources gracefully" {
    mock_docker "not_installed"
    
    run node_red::cleanup_docker_resources
    assert_success
}

@test "node_red::exec_in_container executes command successfully" {
    mock_docker "success"
    
    run node_red::exec_in_container "echo" "test"
    assert_success
}

@test "node_red::exec_in_container fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::exec_in_container "echo" "test"
    assert_failure
}

@test "node_red::copy_to_container copies file successfully" {
    mock_docker "success"
    
    # Create a test file
    echo "test content" > "$NODE_RED_TEST_DIR/test.txt"
    
    run node_red::copy_to_container "$NODE_RED_TEST_DIR/test.txt" "/tmp/test.txt"
    assert_success
}

@test "node_red::copy_to_container fails with missing source file" {
    run node_red::copy_to_container "/nonexistent/file.txt" "/tmp/test.txt"
    assert_failure
    assert_output_contains "Source file does not exist"
}

@test "node_red::copy_from_container copies file successfully" {
    mock_docker "success"
    
    run node_red::copy_from_container "/data/flows.json" "$NODE_RED_TEST_DIR/flows.json"
    assert_success
}

@test "node_red::copy_from_container fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::copy_from_container "/data/flows.json" "$NODE_RED_TEST_DIR/flows.json"
    assert_failure
}

@test "node_red::get_docker_stats returns statistics" {
    mock_docker "success"
    
    run node_red::get_docker_stats
    assert_success
}

@test "node_red::get_docker_stats fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::get_docker_stats
    assert_failure
}

@test "node_red::inspect_network returns network information" {
    # Mock network inspect to succeed
    docker() {
        if [[ "$1" == "network" && "$2" == "inspect" ]]; then
            echo '{"Name": "vrooli-test-network"}'
            return 0
        fi
        return 0
    }
    export -f docker
    
    run node_red::inspect_network
    assert_success
}

@test "node_red::inspect_network fails when network doesn't exist" {
    # Mock network inspect to fail
    docker() {
        if [[ "$1" == "network" && "$2" == "inspect" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::inspect_network
    assert_failure
}

# Test environment variable handling
@test "functions use correct container name from environment" {
    export CONTAINER_NAME="custom-node-red"
    
    mock_docker "success"
    run node_red::is_installed
    assert_success
}

@test "functions use correct image name from environment" {
    export IMAGE_NAME="custom-node-red:test"
    export BUILD_IMAGE="yes"
    
    # Create mock Dockerfile
    mkdir -p "$SCRIPT_DIR/docker"
    echo "FROM nodered/node-red:latest" > "$SCRIPT_DIR/docker/Dockerfile"
    
    run node_red::build_custom_image
    assert_success
}

# Test error handling
@test "functions handle docker permission errors" {
    # Mock docker to return permission denied error
    docker() {
        echo "permission denied" >&2
        return 126
    }
    export -f docker
    
    run node_red::is_installed
    assert_failure
}

@test "functions handle docker network errors" {
    # Mock docker network commands to fail
    docker() {
        if [[ "$1" == "network" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::create_network
    assert_failure
}

# Test concurrent operations
@test "docker operations work when called concurrently" {
    mock_docker "success"
    
    # Run multiple operations in background
    node_red::is_installed &
    node_red::get_container_info &
    node_red::custom_image_exists &
    
    wait  # Wait for all background processes
    
    # All should have completed successfully
    [[ $? -eq 0 ]]
}
