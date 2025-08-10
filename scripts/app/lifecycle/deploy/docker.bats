#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Define location for this test file
APP_LIFECYCLE_DEPLOY_DIR="$BATS_TEST_DIRNAME"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/docker.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/logs.sh"

# Path to the script under test
SCRIPT_PATH="${APP_LIFECYCLE_DEPLOY_DIR}/docker.sh"

setup() {
    # Initialize mocks
    mock::docker::reset
    mock::logs::reset
    
    # Set required environment variables
    export ENVIRONMENT="test"
    export var_ROOT_DIR="/tmp/test-project"
}

teardown() {
    # Clean up mocks
    mock::docker::cleanup
    mock::logs::cleanup
}

@test "docker::deploy_docker function exists when sourced" {
    run bash -c "source '$SCRIPT_PATH' && declare -f docker::deploy_docker"
    [ "$status" -eq 0 ]
    [[ "$output" =~ docker::deploy_docker ]]
}

@test "docker::deploy_docker loads images from tar if present" {
    # Setup mock for docker functions
    docker::load_images_from_tar() {
        echo "Loading images from $1"
        return 0
    }
    docker::get_compose_file() {
        echo "/path/to/docker-compose.yml"
        return 0
    }
    docker::compose() {
        echo "Docker compose called with: $*"
        return 0
    }
    docker::kill_all() {
        echo "Killing all containers"
        return 0
    }
    export -f docker::load_images_from_tar
    export -f docker::get_compose_file
    export -f docker::compose
    export -f docker::kill_all
    
    # Create test artifact directory with tar file
    local test_dir="/tmp/test-artifacts-$$"
    mkdir -p "$test_dir"
    touch "$test_dir/docker-images.tar"
    
    run bash -c "
        source '$SCRIPT_PATH'
        docker::deploy_docker '$test_dir' 2>&1
    "
    
    # Clean up
    trash::safe_remove "$test_dir" --test-cleanup
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Loading images from" ]]
    [[ "$output" =~ "docker-images.tar" ]]
}

@test "docker::deploy_docker handles missing tar file gracefully" {
    # Setup mock for docker functions
    docker::get_compose_file() {
        echo "/path/to/docker-compose.yml"
        return 0
    }
    docker::compose() {
        echo "Docker compose called with: $*"
        return 0
    }
    docker::kill_all() {
        echo "Killing all containers"
        return 0
    }
    export -f docker::get_compose_file
    export -f docker::compose
    export -f docker::kill_all
    
    # Create test artifact directory without tar file
    local test_dir="/tmp/test-artifacts-$$"
    mkdir -p "$test_dir"
    
    run bash -c "
        source '$SCRIPT_PATH'
        docker::deploy_docker '$test_dir' 2>&1
    "
    
    # Clean up
    trash::safe_remove "$test_dir" --test-cleanup
    
    [ "$status" -eq 0 ]
    [[ ! "$output" =~ "Loading images from" ]]
}

@test "docker::deploy_docker runs docker compose commands in correct order" {
    # Setup mock for docker functions
    local command_order=""
    docker::get_compose_file() {
        echo "/path/to/docker-compose.yml"
        return 0
    }
    docker::compose() {
        command_order="${command_order}compose:$1;"
        echo "Docker compose called with: $*"
        return 0
    }
    docker::kill_all() {
        command_order="${command_order}kill_all;"
        echo "Killing all containers"
        return 0
    }
    export -f docker::get_compose_file
    export -f docker::compose
    export -f docker::kill_all
    export command_order
    
    # Create test artifact directory
    local test_dir="/tmp/test-artifacts-$$"
    mkdir -p "$test_dir"
    
    run bash -c "
        source '$SCRIPT_PATH'
        docker::deploy_docker '$test_dir' 2>&1
        echo \"Order: \$command_order\"
    "
    
    # Clean up
    trash::safe_remove "$test_dir" --test-cleanup
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Order: compose:down;kill_all;compose:up;" ]]
}

@test "docker::deploy_docker handles pushd/popd errors" {
    # Setup mock for docker functions
    docker::get_compose_file() {
        echo "/path/to/docker-compose.yml"
        return 0
    }
    export -f docker::get_compose_file
    
    # Use non-existent root directory
    export var_ROOT_DIR="/non/existent/directory"
    
    run bash -c "
        source '$SCRIPT_PATH'
        docker::deploy_docker '/tmp' 2>&1
    "
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Failed to change directory" ]]
}

@test "docker::deploy_docker can be run directly from command line" {
    # Setup mock for docker functions
    docker::load_images_from_tar() {
        return 0
    }
    docker::get_compose_file() {
        echo "/path/to/docker-compose.yml"
        return 0
    }
    docker::compose() {
        return 0
    }
    docker::kill_all() {
        return 0
    }
    export -f docker::load_images_from_tar
    export -f docker::get_compose_file
    export -f docker::compose
    export -f docker::kill_all
    
    # Create test artifact directory
    local test_dir="/tmp/test-artifacts-$$"
    mkdir -p "$test_dir"
    
    run bash -c "$SCRIPT_PATH '$test_dir' 2>&1"
    
    # Clean up
    trash::safe_remove "$test_dir" --test-cleanup
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Docker deployment completed" ]]
}