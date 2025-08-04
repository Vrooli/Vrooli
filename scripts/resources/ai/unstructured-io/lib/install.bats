#!/usr/bin/env bats
# Tests for Unstructured.io install.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export UNSTRUCTURED_IO_CUSTOM_PORT="9999"
    export UNSTRUCTURED_IO_CONTAINER_NAME="unstructured-io-test"
    export UNSTRUCTURED_IO_BASE_URL="http://localhost:9999"
    export UNSTRUCTURED_IO_IMAGE="downloads.unstructured.io/unstructured-io/unstructured-api:0.0.78"
    export UNSTRUCTURED_IO_API_PORT="8000"
    export UNSTRUCTURED_IO_MEMORY_LIMIT="4g"
    export UNSTRUCTURED_IO_CPU_LIMIT="2.0"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    UNSTRUCTURED_IO_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock rollback functions
    resources::add_rollback_action() {
        echo "ROLLBACK: $1"
        return 0
    }
    
    resources::start_rollback_context() {
        echo "START_ROLLBACK: $1"
        return 0
    }
    
    resources::handle_error() {
        echo "HANDLE_ERROR: $1 - $2 - $3"
        return 1
    }
    
    resources::update_config() {
        echo "UPDATE_CONFIG: $1 $2 $3"
        return 0
    }
    
    resources::remove_config() {
        echo "REMOVE_CONFIG: $1 $2"
        return 0
    }
    
    # Mock system functions
    
    system::is_port_in_use() {
        return 1  # Port available
    }
    
    # Mock Docker functions
    
    # Mock log functions
    
    
    
    
    # Mock other common functions
    unstructured_io::check_docker() { return 0; }
    unstructured_io::container_exists() { return 1; }  # No existing container
    unstructured_io::container_running() { return 0; }
    unstructured_io::check_port_available() { return 0; }
    unstructured_io::status() { return 0; }
    
    # Load configuration and messages
    source "${UNSTRUCTURED_IO_DIR}/config/defaults.sh"
    source "${UNSTRUCTURED_IO_DIR}/config/messages.sh"
    unstructured_io::export_config
    unstructured_io::export_messages
    
    # Load the functions to test
    source "${UNSTRUCTURED_IO_DIR}/lib/install.sh"
}

# Test successful installation
@test "unstructured_io::install succeeds with clean environment" {
    result=$(unstructured_io::install "no")
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "DOCKER_RUN:" ]]
    [[ "$result" =~ "UPDATE_CONFIG:" ]]
    [[ "$result" =~ "SUCCESS:" ]]
}

# Test installation with Docker unavailable
@test "unstructured_io::install fails with Docker unavailable" {
    # Override Docker check to fail
    unstructured_io::check_docker() {
        log::error "Docker not available"
        return 1
    }
    
    run unstructured_io::install "no"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Docker not available" ]]
}

# Test installation with existing container (no force)
@test "unstructured_io::install skips when container exists without force" {
    # Override container check to return existing container
    unstructured_io::container_exists() { return 0; }
    
    result=$(unstructured_io::install "no")
    
    [[ "$result" =~ "already installed" ]]
    [[ ! "$result" =~ "DOCKER_RUN:" ]]
}

# Test installation with existing container (force)
@test "unstructured_io::install reinstalls when container exists with force" {
    # Override container check to return existing container
    unstructured_io::container_exists() { return 0; }
    
    # Mock uninstall function
    unstructured_io::uninstall() {
        echo "UNINSTALL_CALLED"
        return 0
    }
    
    result=$(unstructured_io::install "yes")
    
    [[ "$result" =~ "UNINSTALL_CALLED" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "DOCKER_RUN:" ]]
}

# Test installation with port conflict
@test "unstructured_io::install fails with port conflict" {
    # Override port check to return conflict
    unstructured_io::check_port_available() {
        log::error "Port $UNSTRUCTURED_IO_PORT is in use"
        return 1
    }
    
    run unstructured_io::install "no"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Port $UNSTRUCTURED_IO_PORT is in use" ]]
}

# Test Docker pull failure
@test "unstructured_io::install handles Docker pull failure" {
    # Override docker to fail on pull
    docker() {
        case "$1" in
            "pull")
                echo "ERROR: Pull failed"
                return 1
                ;;
            "ps")
                echo ""
                ;;
            *) return 0 ;;
        esac
    }
    
    run unstructured_io::install "no"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Pull failed" ]]
}

# Test Docker run failure
@test "unstructured_io::install handles Docker run failure" {
    # Override docker to fail on run
    docker() {
        case "$1" in
            "run")
                echo "ERROR: Run failed"
                return 1
                ;;
            "pull")
                echo "DOCKER_PULL: $*"
                return 0
                ;;
            "ps")
                echo ""
                ;;
            *) return 0 ;;
        esac
    }
    
    run unstructured_io::install "no"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Run failed" ]]
}

# Test container startup verification
@test "unstructured_io::install verifies container startup" {
    # Mock container startup check
    unstructured_io::wait_for_startup() {
        echo "STARTUP_CHECK_CALLED"
        return 0
    }
    
    result=$(unstructured_io::install "no")
    
    [[ "$result" =~ "STARTUP_CHECK_CALLED" ]]
}

# Test container startup verification failure
@test "unstructured_io::install handles startup verification failure" {
    # Mock container startup check to fail
    unstructured_io::wait_for_startup() {
        echo "STARTUP_FAILED"
        return 1
    }
    
    run unstructured_io::install "no"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "STARTUP_FAILED" ]]
}

# Test uninstallation
@test "unstructured_io::uninstall removes container successfully" {
    # Override container check to return existing container
    unstructured_io::container_exists() { return 0; }
    
    result=$(unstructured_io::uninstall)
    
    [[ "$result" =~ "DOCKER_STOP:" ]]
    [[ "$result" =~ "DOCKER_RM:" ]]
    [[ "$result" =~ "REMOVE_CONFIG:" ]]
}

# Test uninstallation with no existing container
@test "unstructured_io::uninstall handles missing container gracefully" {
    # Override container check to return no container
    unstructured_io::container_exists() { return 1; }
    
    result=$(unstructured_io::uninstall)
    
    [[ "$result" =~ "not found" ]]
    [[ ! "$result" =~ "DOCKER_STOP:" ]]
}

# Test service start
@test "unstructured_io::start succeeds with existing container" {
    # Override container check to return existing but stopped container
    unstructured_io::container_exists() { return 0; }
    unstructured_io::container_running() { return 1; }  # Not running
    
    # Mock docker start
    docker() {
        case "$1" in
            "start")
                echo "DOCKER_START: $*"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    
    result=$(unstructured_io::start)
    
    [[ "$result" =~ "DOCKER_START:" ]]
}

# Test service start with missing container
@test "unstructured_io::start fails with missing container" {
    # Override container check to return no container
    unstructured_io::container_exists() { return 1; }
    
    run unstructured_io::start
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]]
}

# Test service start with already running container
@test "unstructured_io::start handles already running container" {
    # Override checks to return running container
    unstructured_io::container_exists() { return 0; }
    unstructured_io::container_running() { return 0; }  # Already running
    
    result=$(unstructured_io::start)
    
    [[ "$result" =~ "already running" ]]
}

# Test service stop
@test "unstructured_io::stop succeeds with running container" {
    # Override checks to return running container
    unstructured_io::container_exists() { return 0; }
    unstructured_io::container_running() { return 0; }  # Running
    
    result=$(unstructured_io::stop)
    
    [[ "$result" =~ "DOCKER_STOP:" ]]
}

# Test service stop with stopped container
@test "unstructured_io::stop handles already stopped container" {
    # Override checks to return stopped container
    unstructured_io::container_exists() { return 0; }
    unstructured_io::container_running() { return 1; }  # Not running
    
    result=$(unstructured_io::stop)
    
    [[ "$result" =~ "not running" ]]
}

# Test service restart
@test "unstructured_io::restart performs stop and start sequence" {
    # Mock stop and start functions
    unstructured_io::stop() {
        echo "STOP_CALLED"
        return 0
    }
    
    unstructured_io::start() {
        echo "START_CALLED"
        return 0
    }
    
    result=$(unstructured_io::restart)
    
    [[ "$result" =~ "STOP_CALLED" ]]
    [[ "$result" =~ "START_CALLED" ]]
}

# Test startup wait function
@test "unstructured_io::wait_for_startup succeeds when service becomes available" {
    # Mock curl to succeed after a delay simulation
    curl() {
        case "$*" in
            *"/health"*) 
                echo "OK"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    
    # Mock sleep to avoid actual waiting
    sleep() { return 0; }
    
    unstructured_io::wait_for_startup
    [ "$?" -eq 0 ]
}

# Test startup wait function timeout
@test "unstructured_io::wait_for_startup fails on timeout" {
    # Mock curl to always fail
    curl() {
        return 1
    }
    
    # Mock sleep to avoid actual waiting
    sleep() { return 0; }
    
    # Set short timeout for testing
    export UNSTRUCTURED_IO_STARTUP_MAX_WAIT="1"
    
    run unstructured_io::wait_for_startup
    [ "$status" -eq 1 ]
}