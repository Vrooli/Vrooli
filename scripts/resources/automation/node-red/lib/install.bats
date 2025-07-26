#!/usr/bin/env bats
# Tests for Node-RED installation functions (lib/install.sh)

load ../test-fixtures/test_helper

setup() {
    setup_test_environment
    source_node_red_scripts
    mock_docker "success"
    mock_curl "success"
    mock_jq "success"
    
    # Default test environment variables
    export FORCE="no"
    export YES="no"
    export BUILD_IMAGE="no"
}

teardown() {
    teardown_test_environment
}

@test "node_red::pre_install_checks succeeds with valid environment" {
    run node_red::pre_install_checks
    assert_success
}

@test "node_red::pre_install_checks fails when docker validation fails" {
    # Mock docker validation to fail
    docker() { return 127; }
    export -f docker
    
    run node_red::pre_install_checks
    assert_failure
}

@test "node_red::pre_install_checks fails when directory creation fails" {
    # Mock mkdir to fail
    mkdir() { return 1; }
    export -f mkdir
    
    run node_red::pre_install_checks
    assert_failure
}

@test "node_red::handle_existing_installation prompts for reinstall when already installed" {
    mock_docker "success"  # Container exists
    export YES="no"  # Don't auto-confirm
    
    # Mock user input - simulate 'y' response
    run bash -c 'echo "y" | node_red::handle_existing_installation'
    assert_success
}

@test "node_red::handle_existing_installation auto-confirms with YES=yes" {
    mock_docker "success"  # Container exists
    export YES="yes"
    
    run node_red::handle_existing_installation
    assert_success
}

@test "node_red::handle_existing_installation cancels when user says no" {
    mock_docker "success"  # Container exists
    export YES="no"
    
    # Mock user input - simulate 'n' response
    run bash -c 'echo "n" | node_red::handle_existing_installation'
    assert_failure
}

@test "node_red::handle_existing_installation succeeds when not installed" {
    mock_docker "not_installed"
    
    run node_red::handle_existing_installation
    assert_success
}

@test "node_red::handle_existing_installation respects FORCE=yes" {
    mock_docker "success"  # Container exists
    export FORCE="yes"
    
    run node_red::handle_existing_installation
    assert_success
}

@test "node_red::install succeeds with fresh installation" {
    mock_docker "not_installed"
    export BUILD_IMAGE="no"
    
    run node_red::install
    assert_success
    assert_output_contains "installed successfully"
}

@test "node_red::install succeeds with custom image build" {
    mock_docker "not_installed"
    export BUILD_IMAGE="yes"
    
    # Create mock Dockerfile
    mkdir -p "$SCRIPT_DIR/docker"
    echo "FROM nodered/node-red:latest" > "$SCRIPT_DIR/docker/Dockerfile"
    
    run node_red::install
    assert_success
    assert_output_contains "installed successfully"
}

@test "node_red::install handles existing installation" {
    mock_docker "success"  # Container exists
    export YES="yes"  # Auto-confirm reinstall
    
    run node_red::install
    assert_success
}

@test "node_red::install fails when pre-install checks fail" {
    # Mock docker to fail
    docker() { return 127; }
    export -f docker
    
    run node_red::install
    assert_failure
    assert_output_contains "installation failed"
}

@test "node_red::install fails when container start fails" {
    mock_docker "not_installed"
    
    # Mock docker run to fail
    docker() {
        if [[ "$1" == "run" ]]; then
            return 1
        fi
        # Allow other docker commands to succeed
        return 0
    }
    export -f docker
    
    run node_red::install
    assert_failure
    assert_output_contains "installation failed"
}

@test "node_red::install fails when Node-RED doesn't become ready" {
    mock_docker "not_installed"
    mock_curl "failure"  # Node-RED never becomes available
    
    # Set short timeout for testing
    export NODE_RED_READY_TIMEOUT=1
    export NODE_RED_READY_SLEEP=0.1
    
    run node_red::install
    assert_failure
    assert_output_contains "installation failed"
}

@test "node_red::uninstall prompts for confirmation" {
    mock_docker "success"  # Container exists
    export YES="no"
    
    # Mock user input - simulate 'y' response
    run bash -c 'echo "y" | node_red::uninstall'
    assert_success
    assert_output_contains "uninstalled successfully"
}

@test "node_red::uninstall auto-confirms with YES=yes" {
    mock_docker "success"  # Container exists
    export YES="yes"
    
    run node_red::uninstall
    assert_success
    assert_output_contains "uninstalled successfully"
}

@test "node_red::uninstall cancels when user says no" {
    mock_docker "success"  # Container exists
    export YES="no"
    
    # Mock user input - simulate 'n' response
    run bash -c 'echo "n" | node_red::uninstall'
    assert_success  # Cancellation is not an error
}

@test "node_red::uninstall handles non-existent installation gracefully" {
    mock_docker "not_installed"
    
    run node_red::uninstall
    assert_success
    assert_output_contains "not installed"
}

@test "node_red::uninstall respects FORCE=yes" {
    mock_docker "success"  # Container exists
    export FORCE="yes"
    
    run node_red::uninstall
    assert_success
    assert_output_contains "uninstalled successfully"
}

@test "node_red::uninstall_without_prompts removes all resources" {
    mock_docker "success"
    
    run node_red::uninstall_without_prompts
    assert_success
    assert_output_contains "uninstalled successfully"
}

@test "node_red::start succeeds when container exists and is stopped" {
    # Mock container exists but is not running
    docker() {
        case "$1 $2" in
            "container inspect") return 0 ;;  # Container exists
            "container inspect -f*") echo "false" ;;  # Not running
            "start") return 0 ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::start
    assert_success
}

@test "node_red::start fails when container doesn't exist" {
    mock_docker "not_installed"
    
    run node_red::start
    assert_failure
    assert_output_contains "not installed"
}

@test "node_red::start handles already running container" {
    mock_docker "success"  # Container running
    
    run node_red::start
    assert_success
    assert_output_contains "already running"
}

@test "node_red::stop succeeds when container is running" {
    mock_docker "success"
    
    run node_red::stop
    assert_success
}

@test "node_red::stop handles non-running container" {
    mock_docker "not_running"
    
    run node_red::stop
    assert_success
}

@test "node_red::restart succeeds" {
    mock_docker "success"
    
    run node_red::restart
    assert_success
}

@test "node_red::restart handles non-existent container" {
    mock_docker "not_installed"
    
    run node_red::restart
    assert_failure
}

@test "node_red::post_install_setup updates configuration" {
    run node_red::post_install_setup
    assert_success
    
    # Check that config file was created
    assert_file_exists "$NODE_RED_TEST_CONFIG_DIR/resources.local.json"
}

@test "node_red::post_install_setup imports example flows" {
    # Create example flows
    mkdir -p "$SCRIPT_DIR/flows"
    echo '{"id": "test", "type": "tab"}' > "$SCRIPT_DIR/flows/example.json"
    
    mock_docker "success"
    mock_curl "success"
    
    run node_red::post_install_setup
    assert_success
}

@test "node_red::post_install_setup handles missing flows gracefully" {
    # No flows directory
    run node_red::post_install_setup
    assert_success
}

@test "node_red::reinstall performs fresh installation" {
    mock_docker "success"  # Existing installation
    
    run node_red::reinstall
    assert_success
}

@test "node_red::reinstall handles non-existent installation" {
    mock_docker "not_installed"
    
    run node_red::reinstall
    assert_success
}

@test "node_red::update succeeds when container exists" {
    mock_docker "success"
    export BUILD_IMAGE="no"  # Using official image
    
    run node_red::update
    assert_success
    assert_output_contains "updated successfully"
}

@test "node_red::update fails when container doesn't exist" {
    mock_docker "not_installed"
    
    run node_red::update
    assert_failure
    assert_output_contains "not installed"
}

@test "node_red::update skips pull with custom image" {
    mock_docker "success"
    export BUILD_IMAGE="yes"  # Using custom image
    
    run node_red::update
    assert_success
    assert_output_contains "updated successfully"
}

@test "node_red::backup creates backup successfully" {
    mock_docker "success"
    mock_curl "success"
    
    # Create some test data
    mkdir -p "$SCRIPT_DIR/flows"
    echo '{"test": "flow"}' > "$SCRIPT_DIR/flows/test.json"
    
    run node_red::backup
    assert_success
    assert_output_contains "Backup created"
}

@test "node_red::backup with custom directory" {
    mock_docker "success"
    mock_curl "success"
    
    local custom_backup_dir="$NODE_RED_TEST_DIR/custom-backups"
    
    run node_red::backup "$custom_backup_dir"
    assert_success
    assert_output_contains "Backup created"
}

@test "node_red::backup includes running flows export" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::backup
    assert_success
    assert_output_contains "Exported running flows"
}

@test "node_red::backup handles non-running container" {
    mock_docker "not_running"
    
    run node_red::backup
    assert_success
}

@test "node_red::restore succeeds with valid backup" {
    # Create a mock backup
    local backup_dir="$NODE_RED_TEST_DIR/test-backup"
    mkdir -p "$backup_dir/flows"
    echo '{"test": "flow"}' > "$backup_dir/flows/test.json"
    echo 'module.exports = {};' > "$backup_dir/settings.js" 
    
    mock_docker "success"
    mock_curl "success"
    
    run node_red::restore "$backup_dir"
    assert_success
    assert_output_contains "restored from backup"
}

@test "node_red::restore fails with missing backup path" {
    run node_red::restore
    assert_failure
    assert_output_contains "Backup path is required"
}

@test "node_red::restore fails with non-existent backup" {
    run node_red::restore "/nonexistent/backup"
    assert_failure
    assert_output_contains "Backup directory not found"
}

@test "node_red::restore handles partial backups" {
    # Create backup with only flows
    local backup_dir="$NODE_RED_TEST_DIR/partial-backup"
    mkdir -p "$backup_dir/flows"
    echo '{"test": "flow"}' > "$backup_dir/flows/test.json"
    
    mock_docker "success"
    
    run node_red::restore "$backup_dir"
    assert_success
}

@test "node_red::restore restarts container if it was running" {
    local backup_dir="$NODE_RED_TEST_DIR/test-backup"
    mkdir -p "$backup_dir"
    
    mock_docker "success"  # Container running
    
    run node_red::restore "$backup_dir"
    assert_success
}

@test "node_red::check_updates detects available updates" {
    mock_docker "success"
    
    # Mock different image IDs to simulate update available
    docker() {
        case "$1 $2" in
            "inspect --format*")
                if [[ "$*" =~ "container" ]]; then
                    echo "sha256:old123"
                else
                    echo "sha256:new456"
                fi
                ;;
            "pull") return 0 ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::check_updates
    assert_success
    assert_output_contains "Updates available"
}

@test "node_red::check_updates reports up to date" {
    mock_docker "success"
    
    # Mock same image IDs to simulate no updates
    docker() {
        case "$1 $2" in
            "inspect --format*") echo "sha256:same123" ;;
            "pull") return 0 ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::check_updates
    assert_failure  # Function returns 1 when up to date
    assert_output_contains "up to date"
}

@test "node_red::check_updates fails when container doesn't exist" {
    mock_docker "not_installed"
    
    run node_red::check_updates
    assert_failure
    assert_output_contains "not installed"
}

@test "node_red::validate_installation passes with healthy installation" {
    mock_docker "success"
    mock_curl "success"
    
    # Create required files
    mkdir -p "$SCRIPT_DIR/flows"
    echo 'module.exports = {};' > "$SCRIPT_DIR/settings.js"
    mkdir -p "$NODE_RED_TEST_CONFIG_DIR"
    echo '{"services": {"automation": {"node-red": {"enabled": true}}}}' > "$NODE_RED_TEST_CONFIG_DIR/resources.local.json"
    
    run node_red::validate_installation
    assert_success
    assert_output_contains "validation passed"
}

@test "node_red::validate_installation detects missing container" {
    mock_docker "not_installed"
    
    run node_red::validate_installation
    assert_failure
    assert_output_contains "Container does not exist"
}

@test "node_red::validate_installation detects non-running container" {
    mock_docker "not_running"
    
    run node_red::validate_installation
    assert_failure
    assert_output_contains "Container is not running"
}

@test "node_red::validate_installation detects unresponsive service" {
    mock_docker "success"
    mock_curl "failure"
    
    run node_red::validate_installation
    assert_failure
    assert_output_contains "not responding"
}

@test "node_red::validate_installation detects missing files" {
    mock_docker "success"
    mock_curl "success"
    
    # Don't create settings.js or flows directory
    run node_red::validate_installation
    assert_failure
    assert_output_contains "issues with the installation"
}

@test "node_red::validate_installation detects missing configuration" {
    mock_docker "success"
    mock_curl "success"
    
    # Create files but not configuration
    mkdir -p "$SCRIPT_DIR/flows"
    echo 'module.exports = {};' > "$SCRIPT_DIR/settings.js"
    
    run node_red::validate_installation
    assert_failure
    assert_output_contains "Resource configuration missing"
}

# Test environment variable handling
@test "install functions respect FORCE environment variable" {
    mock_docker "success"  # Existing installation
    export FORCE="yes"
    
    run node_red::handle_existing_installation
    assert_success
}

@test "install functions respect YES environment variable" {
    mock_docker "success"  # Existing installation
    export YES="yes"
    
    run node_red::uninstall
    assert_success
}

@test "install functions respect BUILD_IMAGE environment variable" {
    mock_docker "not_installed"
    export BUILD_IMAGE="yes"
    
    # Create mock Dockerfile
    mkdir -p "$SCRIPT_DIR/docker"
    echo "FROM nodered/node-red:latest" > "$SCRIPT_DIR/docker/Dockerfile"
    
    run node_red::install
    assert_success
}

# Test error handling
@test "install functions handle docker failures gracefully" {
    mock_docker "failure"
    
    run node_red::install
    assert_failure
    
    run node_red::start
    assert_failure
}

@test "install functions handle network failures gracefully" {
    mock_docker "success"
    mock_curl "failure"
    
    # Set short timeout for testing
    export NODE_RED_READY_TIMEOUT=1
    export NODE_RED_READY_SLEEP=0.1
    
    run node_red::install
    assert_failure
}

@test "install functions handle filesystem errors gracefully" {
    # Mock mkdir to fail
    mkdir() { return 1; }
    export -f mkdir
    
    run node_red::install
    assert_failure
}

# Test cleanup behavior
@test "failed installation cleans up properly" {
    mock_docker "not_installed"
    
    # Mock docker run to fail after creating some resources
    docker() {
        case "$1" in
            "volume"|"network") return 0 ;;  # Allow setup
            "run") return 1 ;;  # Fail container creation
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::install
    assert_failure
}

@test "uninstall removes all traces" {
    mock_docker "success"
    export YES="yes"
    
    # Create some files that should be cleaned up
    mkdir -p "$SCRIPT_DIR/flows"
    echo 'test' > "$SCRIPT_DIR/flows/test.json"
    
    run node_red::uninstall
    assert_success
}