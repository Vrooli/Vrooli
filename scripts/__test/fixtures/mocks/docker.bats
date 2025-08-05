#!/usr/bin/env bats
# Comprehensive tests for Docker mock system
# Tests the docker.sh mock implementation for correctness and integration

# Test setup - load dependencies
setup() {
    # Set up test environment
    export MOCK_UTILS_VERBOSE=false
    export MOCK_VERIFICATION_ENABLED=true
    
    # Load the mock utilities first (required by docker mock)
    source "${BATS_TEST_DIRNAME}/logs.sh"
    
    # Load verification system if available
    if [[ -f "${BATS_TEST_DIRNAME}/verification.bash" ]]; then
        source "${BATS_TEST_DIRNAME}/verification.sh"
    fi
    
    # Load the docker mock
    source "${BATS_TEST_DIRNAME}/docker.sh"
    
    # Initialize clean state for each test
    mock::docker::reset
    
    # Create test log directory
    TEST_LOG_DIR=$(mktemp -d)
    mock::init_logging "$TEST_LOG_DIR"
}

# Wrapper for run command that reloads docker state afterward
run_docker_command() {
    run "$@"
    # Reload state from file after docker commands that might modify state
    # Now the state file uses declare -gA for global scope
    if [[ -n "${DOCKER_MOCK_STATE_FILE}" && -f "$DOCKER_MOCK_STATE_FILE" ]]; then
        eval "$(cat "$DOCKER_MOCK_STATE_FILE")" 2>/dev/null || true
    fi
}

# Test cleanup
teardown() {
    # Clean up test logs
    if [[ -n "${TEST_LOG_DIR:-}" && -d "$TEST_LOG_DIR" ]]; then
        rm -rf "$TEST_LOG_DIR"
    fi
    
    # Note: Removed mock::docker::reset because it was interfering with test execution
    # BATS calls teardown during test execution, wiping state before assertions
    
    # Clean up environment
    unset DOCKER_MOCK_MODE
    unset TEST_LOG_DIR
}

# =============================================================================
# Basic Docker Command Tests
# =============================================================================

@test "docker version should return mock version" {
    run docker --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Docker version 24.0.0" ]]
}

@test "docker help should show mock help" {
    run docker --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage: docker" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "docker invalid command should fail" {
    run docker nonexistent_command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not a docker command" ]]
}

# =============================================================================
# Docker PS Tests
# =============================================================================

@test "docker ps shows table headers" {
    run docker ps
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CONTAINER ID" ]]
    [[ "$output" =~ "IMAGE" ]]
    [[ "$output" =~ "NAMES" ]]
}

@test "docker ps shows running containers only" {
    # Set up test containers
    mock::docker::set_container_state "test_running" "running" "nginx:latest"
    mock::docker::set_container_state "test_stopped" "stopped" "redis:alpine"
    
    run docker ps
    [ "$status" -eq 0 ]
    
    [[ "$output" =~ "test_running" ]]
    [[ "$output" != *"test_stopped"* ]]
}

@test "docker ps -a shows all containers" {
    # Set up test containers
    mock::docker::set_container_state "test_running" "running" "nginx:latest"
    mock::docker::set_container_state "test_stopped" "stopped" "redis:alpine"
    
    run docker ps -a
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test_running" ]]
    [[ "$output" =~ "test_stopped" ]]
}

@test "docker ps with format option" {
    mock::docker::set_container_state "test_container" "running" "test:latest"
    
    run docker ps --format "{{.Names}}"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test_container" ]]
}

# =============================================================================
# Docker Images Tests
# =============================================================================

@test "docker images shows table headers" {
    run docker images
    [ "$status" -eq 0 ]
    [[ "$output" =~ "REPOSITORY" ]]
    [[ "$output" =~ "TAG" ]]
    [[ "$output" =~ "IMAGE ID" ]]
}

@test "docker images shows available images" {
    mock::docker::set_image_available "nginx" "true"
    mock::docker::set_image_available "redis" "true"
    
    run docker images
    [ "$status" -eq 0 ]
    [[ "$output" =~ "nginx" ]]
    [[ "$output" =~ "redis" ]]
}

@test "docker images with format option" {
    mock::docker::set_image_available "test" "true"
    
    run docker images --format "{{.Repository}}"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test" ]]
}

# =============================================================================
# Docker Run Tests
# =============================================================================

@test "docker run creates and starts container" {
    mock::docker::set_image_available "nginx" "true"
    
    run_docker_command docker run --name test_nginx nginx
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Container started" ]]
    
    # Verify container was created and is in stopped state (non-detached)
    [[ -n "${MOCK_DOCKER_CONTAINERS[test_nginx]}" ]]
    local state="${MOCK_DOCKER_CONTAINERS[test_nginx]%%|*}"
    [[ "$state" == "stopped" ]]
}

@test "docker run with detach returns container ID" {
    mock::docker::set_image_available "nginx" "true"
    
    run_docker_command docker run -d --name test_nginx nginx
    [ "$status" -eq 0 ]
    # Should return a 12-character container ID
    [[ ${#output} -eq 12 ]]
    
    # Verify container is running
    local state="${MOCK_DOCKER_CONTAINERS[test_nginx]%%|*}"
    [[ "$state" == "running" ]]
}

@test "docker run with environment variables" {
    mock::docker::set_image_available "app" "true"
    
    run_docker_command docker run -d --name test_app -e NODE_ENV=production -e PORT=3000 app
    [ "$status" -eq 0 ]
    
    # Verify environment variables were stored
    [[ "${MOCK_DOCKER_CONTAINER_ENV[test_app]}" =~ "NODE_ENV=production" ]]
    [[ "${MOCK_DOCKER_CONTAINER_ENV[test_app]}" =~ "PORT=3000" ]]
}

@test "docker run with port mapping" {
    mock::docker::set_image_available "web" "true"
    
    run_docker_command docker run -d --name test_web -p 8080:80 web
    [ "$status" -eq 0 ]
    
    # Verify port mapping was stored
    [[ "${MOCK_DOCKER_CONTAINER_PORTS[test_web]}" =~ "8080:80" ]]
}

@test "docker run with --rm removes container after execution" {
    mock::docker::set_image_available "temp" "true"
    
    run_docker_command docker run --rm --name temp_container temp
    [ "$status" -eq 0 ]
    
    # Container should not exist after --rm execution
    [[ -z "${MOCK_DOCKER_CONTAINERS[temp_container]}" ]]
}

@test "docker run fails with duplicate container name" {
    mock::docker::set_image_available "nginx" "true"
    mock::docker::set_container_state "existing" "running" "nginx"
    
    run docker run --name existing nginx
    [ "$status" -eq 125 ]
    [[ "$output" =~ "Conflict" ]]
    [[ "$output" =~ "already in use" ]]
}

@test "docker run fails with missing image" {
    run docker run --name test_missing missing_image
    [ "$status" -eq 125 ]
    [[ "$output" =~ "Unable to find image" ]]
}

# =============================================================================
# Docker Exec Tests
# =============================================================================

@test "docker exec runs command in container" {
    mock::docker::set_container_state "test_container" "running" "nginx"
    
    run docker exec test_container echo "hello"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Mock exec output" ]]
}

@test "docker exec fails with non-existent container" {
    run docker exec nonexistent echo "hello"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No such container" ]]
}

@test "docker exec fails with stopped container" {
    mock::docker::set_container_state "stopped_container" "stopped" "nginx"
    
    run docker exec stopped_container echo "hello"
    [ "$status" -eq 126 ]
    [[ "$output" =~ "not running" ]]
}

@test "docker exec health check returns healthy" {
    mock::docker::set_container_state "healthy_container" "running" "nginx"
    
    run docker exec healthy_container health
    [ "$status" -eq 0 ]
    [[ "$output" == "healthy" ]]
}

# =============================================================================
# Docker Logs Tests
# =============================================================================

@test "docker logs shows container logs" {
    mock::docker::set_container_state "log_container" "running" "nginx"
    
    run docker logs log_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Container log_container started" ]]
    [[ "$output" =~ "Ready to accept connections" ]]
}

@test "docker logs fails with non-existent container" {
    run docker logs nonexistent
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No such container" ]]
}

@test "docker logs with follow option" {
    mock::docker::set_container_state "streaming_container" "running" "nginx"
    
    run docker logs -f streaming_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Streaming logs" ]]
}

# =============================================================================
# Docker Inspect Tests
# =============================================================================

@test "docker inspect returns JSON by default" {
    mock::docker::set_container_state "inspect_container" "running" "nginx:latest"
    
    run docker inspect inspect_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ '{"' ]]
    [[ "$output" =~ '"Status": "running"' ]]
    [[ "$output" =~ '"Image": "nginx:latest"' ]]
}

@test "docker inspect with format returns specific field" {
    mock::docker::set_container_state "format_container" "stopped" "redis:alpine"
    
    run docker inspect --format "{{.State.Status}}" format_container
    [ "$status" -eq 0 ]
    [[ "$output" == "stopped" ]]
}

@test "docker inspect fails with non-existent container" {
    run docker inspect nonexistent
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No such container" ]]
}

# =============================================================================
# Docker Pull Tests
# =============================================================================

@test "docker pull downloads image" {
    run_docker_command docker pull nginx:latest
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Pulling nginx:latest" ]]
    [[ "$output" =~ "Status: Downloaded" ]]
    
    # Verify image is now available
    [[ "${MOCK_DOCKER_IMAGES[nginx:latest]}" == "true" ]]
}

# =============================================================================
# Docker Build Tests
# =============================================================================

@test "docker build creates image" {
    run_docker_command docker build -t myapp:v1 .
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Building image" ]]
    [[ "$output" =~ "Successfully built" ]]
    [[ "$output" =~ "Successfully tagged myapp:v1" ]]
    
    # Verify image is available
    [[ "${MOCK_DOCKER_IMAGES[myapp:v1]}" == "true" ]]
}

# =============================================================================
# Container Lifecycle Tests
# =============================================================================

@test "docker start activates stopped container" {
    mock::docker::set_container_state "lifecycle_container" "stopped" "nginx"
    
    run_docker_command docker start lifecycle_container
    [ "$status" -eq 0 ]
    [[ "$output" == "lifecycle_container" ]]
    
    # Verify state changed to running
    local state="${MOCK_DOCKER_CONTAINERS[lifecycle_container]%%|*}"
    [[ "$state" == "running" ]]
}

@test "docker stop deactivates running container" {
    mock::docker::set_container_state "lifecycle_container" "running" "nginx"
    
    run_docker_command docker stop lifecycle_container
    [ "$status" -eq 0 ]
    [[ "$output" == "lifecycle_container" ]]
    
    # Verify state changed to stopped
    local state="${MOCK_DOCKER_CONTAINERS[lifecycle_container]%%|*}"
    [[ "$state" == "stopped" ]]
}

@test "docker restart cycles container" {
    mock::docker::set_container_state "lifecycle_container" "stopped" "nginx"
    
    run_docker_command docker restart lifecycle_container
    [ "$status" -eq 0 ]
    [[ "$output" == "lifecycle_container" ]]
    
    # Verify state is running after restart
    local state="${MOCK_DOCKER_CONTAINERS[lifecycle_container]%%|*}"
    [[ "$state" == "running" ]]
}

# =============================================================================
# Container Removal Tests
# =============================================================================

@test "docker rm removes container" {
    mock::docker::set_container_state "removable_container" "stopped" "nginx"
    
    run_docker_command docker rm removable_container
    [ "$status" -eq 0 ]
    [[ "$output" == "removable_container" ]]
    
    # Verify container was removed
    [[ -z "${MOCK_DOCKER_CONTAINERS[removable_container]}" ]]
}

@test "docker rm fails with non-existent container" {
    run docker rm nonexistent
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No such container" ]]
}

@test "docker rmi removes image" {
    mock::docker::set_image_available "removable:latest" "true"
    
    run_docker_command docker rmi removable:latest
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Untagged: removable:latest" ]]
    
    # Verify image was removed
    [[ "${MOCK_DOCKER_IMAGES[removable:latest]}" != "true" ]]
}

# =============================================================================
# Network Management Tests
# =============================================================================

@test "docker network ls shows networks" {
    run docker network ls
    [ "$status" -eq 0 ]
    [[ "$output" =~ "NETWORK ID" ]]
    [[ "$output" =~ "bridge" ]]
    [[ "$output" =~ "host" ]]
}

@test "docker network create creates network" {
    run_docker_command docker network create test_network
    [ "$status" -eq 0 ]
    [[ ${#output} -eq 12 ]]  # Should return network ID
    
    # Verify network was created
    [[ -n "${MOCK_DOCKER_NETWORKS[test_network]}" ]]
}

@test "docker network rm removes network" {
    # Create network first
    MOCK_DOCKER_NETWORKS["removable_network"]="abc123def456"
    
    run_docker_command docker network rm removable_network
    [ "$status" -eq 0 ]
    [[ "$output" == "removable_network" ]]
    
    # Verify network was removed
    [[ -z "${MOCK_DOCKER_NETWORKS[removable_network]}" ]]
}

@test "docker network inspect shows network details" {
    mock::docker::set_network "inspect_network" "xyz789abc123"
    
    run docker network inspect inspect_network
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"Name":"inspect_network"' ]]
    [[ "$output" =~ '"Id":"xyz789abc123"' ]]
}

# =============================================================================
# Volume Management Tests
# =============================================================================

@test "docker volume ls shows volumes" {
    run docker volume ls
    [ "$status" -eq 0 ]
    [[ "$output" =~ "DRIVER" ]]
    [[ "$output" =~ "VOLUME NAME" ]]
}

@test "docker volume create creates volume" {
    run_docker_command docker volume create test_volume
    [ "$status" -eq 0 ]
    [[ "$output" == "test_volume" ]]
    
    # Verify volume was created
    [[ "${MOCK_DOCKER_VOLUMES[test_volume]}" == "local" ]]
}

@test "docker volume rm removes volume" {
    MOCK_DOCKER_VOLUMES["removable_volume"]="local"
    
    run_docker_command docker volume rm removable_volume
    [ "$status" -eq 0 ]
    [[ "$output" == "removable_volume" ]]
    
    # Verify volume was removed
    [[ -z "${MOCK_DOCKER_VOLUMES[removable_volume]}" ]]
}

# =============================================================================
# Docker Compose Tests
# =============================================================================

@test "docker compose up creates services" {
    run docker compose up
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating network" ]]
    [[ "$output" =~ "Creating test_service_1" ]]
}

@test "docker compose down stops services" {
    run docker compose down
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Stopping test_service_1" ]]
    [[ "$output" =~ "Removing test_service_1" ]]
}

@test "docker compose ps shows service status" {
    run docker compose ps
    [ "$status" -eq 0 ]
    [[ "$output" =~ "NAME" ]]
    [[ "$output" =~ "SERVICE" ]]
    [[ "$output" =~ "STATUS" ]]
}

# =============================================================================
# Error Mode Tests
# =============================================================================

@test "docker commands fail in offline mode" {
    export DOCKER_MOCK_MODE="offline"
    
    run docker ps
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Cannot connect to the Docker daemon" ]]
}

@test "docker commands fail in error mode" {
    export DOCKER_MOCK_MODE="error"
    
    run docker images
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Docker command failed" ]]
}

# =============================================================================
# Error Injection Tests
# =============================================================================

@test "injected errors override normal behavior" {
    mock::docker::inject_error "ps" "network_timeout"
    
    run docker ps
    [ "$status" -eq 1 ]
    [[ "$output" =~ "request canceled" ]]
}

@test "container not found error injection" {
    mock::docker::inject_error "start" "container_not_found"
    
    run docker start nonexistent
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No such container: nonexistent" ]]
}

@test "port already in use error injection" {
    mock::docker::inject_error "run" "port_already_in_use"
    
    run docker run -p 8080:80 nginx
    [ "$status" -eq 125 ]
    [[ "$output" =~ "address already in use" ]]
}

# =============================================================================
# Scenario Builder Tests
# =============================================================================

@test "create_running_stack builds complete environment" {
    mock::docker::scenario::create_running_stack "myapp"
    
    # Verify all components are running
    [[ "${MOCK_DOCKER_CONTAINERS[myapp_db]%%|*}" == "running" ]]
    [[ "${MOCK_DOCKER_CONTAINERS[myapp_app]%%|*}" == "running" ]]
    [[ "${MOCK_DOCKER_CONTAINERS[myapp_cache]%%|*}" == "running" ]]
    
    # Verify network was created
    [[ -n "${MOCK_DOCKER_NETWORKS[myapp_network]}" ]]
    
    # Verify metadata was set
    [[ -n "${MOCK_DOCKER_CONTAINER_ENV[myapp_app]}" ]]
    [[ -n "${MOCK_DOCKER_CONTAINER_PORTS[myapp_app]}" ]]
}

@test "create_stopped_stack builds stopped environment" {
    mock::docker::scenario::create_stopped_stack "stopped_app"
    
    # Verify all components are stopped
    [[ "${MOCK_DOCKER_CONTAINERS[stopped_app_db]%%|*}" == "stopped" ]]
    [[ "${MOCK_DOCKER_CONTAINERS[stopped_app_app]%%|*}" == "stopped" ]]
    [[ "${MOCK_DOCKER_CONTAINERS[stopped_app_cache]%%|*}" == "stopped" ]]
}

# =============================================================================
# Assertion Helper Tests
# =============================================================================

@test "assert_container_running succeeds for running container" {
    mock::docker::set_container_state "running_test" "running" "nginx"
    
    run mock::docker::assert::container_running "running_test"
    [ "$status" -eq 0 ]
}

@test "assert_container_running fails for stopped container" {
    mock::docker::set_container_state "stopped_test" "stopped" "nginx"
    
    run mock::docker::assert::container_running "stopped_test"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not running" ]]
}

@test "assert_container_exists succeeds for existing container" {
    mock::docker::set_container_state "existing_test" "running" "nginx"
    
    run mock::docker::assert::container_exists "existing_test"
    [ "$status" -eq 0 ]
}

@test "assert_container_not_exists succeeds for non-existent container" {
    run mock::docker::assert::container_not_exists "nonexistent_test"
    [ "$status" -eq 0 ]
}

@test "assert_port_mapped succeeds when port is mapped" {
    mock::docker::set_container_state "port_test" "running" "nginx"
    MOCK_DOCKER_CONTAINER_PORTS["port_test"]="8080:80 3000:3000"
    
    run mock::docker::assert::port_mapped "port_test" "8080"
    [ "$status" -eq 0 ]
}

@test "assert_env_set succeeds when environment variable is set" {
    mock::docker::set_container_state "env_test" "running" "app"
    MOCK_DOCKER_CONTAINER_ENV["env_test"]="NODE_ENV=production PORT=3000"
    
    run mock::docker::assert::env_set "env_test" "NODE_ENV=production"
    [ "$status" -eq 0 ]
}

# =============================================================================
# Logging Integration Tests
# =============================================================================

@test "docker commands are logged to docker_calls.log" {
    # Ensure we have a test log directory
    [[ -n "$TEST_LOG_DIR" ]]
    
    # Run some docker commands
    docker ps >/dev/null
    docker images >/dev/null
    
    # Check that commands were logged
    [[ -f "$TEST_LOG_DIR/docker_calls.log" ]]
    
    local log_content=$(cat "$TEST_LOG_DIR/docker_calls.log")
    [[ "$log_content" =~ "docker: ps" ]]
    [[ "$log_content" =~ "docker: images" ]]
}

@test "docker state changes are logged" {
    mock::docker::set_container_state "log_test" "running" "nginx"
    
    # Check that state change was logged
    [[ -f "$TEST_LOG_DIR/used_mocks.log" ]]
    
    local log_content=$(cat "$TEST_LOG_DIR/used_mocks.log")
    [[ "$log_content" =~ "docker_container_state:log_test:running" ]]
}

# =============================================================================
# Verification Integration Tests
# =============================================================================

@test "docker commands trigger verification recording" {
    # Skip if verification system not available
    if ! command -v mock::verify::record_call &>/dev/null; then
        skip "Verification system not available"
    fi
    
    # Reset verification state
    mock::verify::reset
    
    # Run a docker command
    docker version >/dev/null
    
    # Verify the call was recorded (basic check)
    run mock::verify::was_called "docker" "version"
    [ "$status" -eq 0 ]
}

# =============================================================================
# Debug and Utility Tests
# =============================================================================

@test "debug dump shows current state" {
    mock::docker::set_container_state "debug_container" "running" "nginx"
    mock::docker::set_image_available "debug_image" "true"
    MOCK_DOCKER_NETWORKS["debug_network"]="abc123"
    
    run mock::docker::debug::dump_state
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Docker Mock State Dump" ]]
    [[ "$output" =~ "debug_container" ]]
    [[ "$output" =~ "debug_image" ]]
    [[ "$output" =~ "debug_network" ]]
}

@test "get container info helpers work correctly" {
    mock::docker::set_container_state "info_test" "running" "nginx"
    MOCK_DOCKER_CONTAINER_ENV["info_test"]="NODE_ENV=test"
    MOCK_DOCKER_CONTAINER_PORTS["info_test"]="8080:80"
    MOCK_DOCKER_CONTAINER_VOLUMES["info_test"]="/data:/app/data"
    
    run mock::docker::get::container_env "info_test"
    [[ "$output" == "NODE_ENV=test" ]]
    
    run mock::docker::get::container_ports "info_test"
    [[ "$output" == "8080:80" ]]
    
    run mock::docker::get::container_volumes "info_test"
    [[ "$output" == "/data:/app/data" ]]
}

@test "reset function clears all state" {
    # Set up some state
    mock::docker::set_container_state "reset_test" "running" "nginx"
    mock::docker::set_image_available "reset_image" "true"
    MOCK_DOCKER_NETWORKS["reset_network"]="abc123"
    
    # Verify state exists
    [[ -n "${MOCK_DOCKER_CONTAINERS[reset_test]}" ]]
    [[ -n "${MOCK_DOCKER_IMAGES[reset_image]}" ]]
    [[ -n "${MOCK_DOCKER_NETWORKS[reset_network]}" ]]
    
    # Reset and verify state is cleared
    mock::docker::reset
    
    [[ -z "${MOCK_DOCKER_CONTAINERS[reset_test]}" ]]
    [[ -z "${MOCK_DOCKER_IMAGES[reset_image]}" ]]
    [[ -z "${MOCK_DOCKER_NETWORKS[reset_network]}" ]]
}