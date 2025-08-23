#!/usr/bin/env bats
# Tests for Huginn lib/install.sh

source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_service_test "huginn"
    # Load huginn scripts
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    source "${SCRIPT_DIR}/install.sh"
    
    # Override flow control for automated testing
    flow::is_yes() { return 0; }
    export -f flow::is_yes
}

teardown() {
    teardown_test_environment
}

@test "install.sh: install checks if already installed" {
    mock_docker "success"
    
    # Override to simulate force=no
    export FORCE="no"
    
    run huginn::install
    assert_success
    assert_output_contains "already installed"
}

@test "install.sh: install proceeds with force flag" {
    mock_docker "success"
    export FORCE="yes"
    
    # Mock the actual installation functions
    huginn::stop() { return 0; }
    huginn::uninstall() { return 0; }
    huginn::perform_installation() { 
        echo "Installing Huginn..."
        return 0
    }
    export -f huginn::stop huginn::uninstall huginn::perform_installation
    
    run huginn::install
    assert_success
    assert_output_contains "Installing"
}

@test "install.sh: installation creates required directories" {
    # Use test directories
    export HUGINN_DATA_DIR="$HUGINN_TEST_DIR/data"
    export HUGINN_DB_DIR="$HUGINN_TEST_DIR/db"
    export HUGINN_UPLOADS_DIR="$HUGINN_TEST_DIR/uploads"
    
    # Mock only the Docker operations
    huginn::ensure_network() { return 0; }
    huginn::pull_images() { return 0; }
    huginn::create_containers() { return 0; }
    huginn::start() { return 0; }
    huginn::wait_for_ready() { return 0; }
    export -f huginn::ensure_network huginn::pull_images huginn::create_containers huginn::start huginn::wait_for_ready
    
    run huginn::perform_installation
    assert_success
    
    assert_directory_exists "$HUGINN_DATA_DIR"
    assert_directory_exists "$HUGINN_DB_DIR"
    assert_directory_exists "$HUGINN_UPLOADS_DIR"
}

@test "install.sh: pull_images pulls both images" {
    local images_pulled=()
    
    docker() {
        case "$*" in
            *"pull"*huginn/huginn*)
                images_pulled+=("huginn")
                echo "Pulling huginn image..."
                return 0
                ;;
            *"pull"*postgres*)
                images_pulled+=("postgres")
                echo "Pulling postgres image..."
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::pull_images
    assert_success
    
    [[ " ${images_pulled[@]} " =~ " huginn " ]]
    [[ " ${images_pulled[@]} " =~ " postgres " ]]
}

@test "install.sh: create_containers creates both containers" {
    local containers_created=()
    
    docker() {
        case "$*" in
            *"run"*huginn-postgres*)
                containers_created+=("postgres")
                return 0
                ;;
            *"run"*huginn/huginn*)
                containers_created+=("huginn")
                return 0
                ;;
            *"network ls"*)
                echo "huginn-network"
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::create_containers
    assert_success
    
    [[ " ${containers_created[@]} " =~ " postgres " ]]
    [[ " ${containers_created[@]} " =~ " huginn " ]]
}

@test "install.sh: uninstall removes containers when confirmed" {
    mock_docker "not_running"
    
    # Override confirmation
    flow::is_yes() { return 0; }
    export -f flow::is_yes
    
    # Track what gets removed
    local containers_removed=false
    local network_removed=false
    
    huginn::remove_containers() {
        containers_removed=true
        return 0
    }
    
    huginn::remove_network() {
        network_removed=true
        return 0
    }
    
    export -f huginn::remove_containers huginn::remove_network
    
    run huginn::uninstall
    assert_success
    
    [[ "$containers_removed" == "true" ]]
    [[ "$network_removed" == "true" ]]
}

@test "install.sh: uninstall skips when not confirmed" {
    mock_docker "success"
    
    # Override confirmation to decline
    flow::is_yes() { return 1; }
    export -f flow::is_yes
    
    run huginn::uninstall
    assert_success
    assert_output_contains "cancelled"
}

@test "install.sh: uninstall removes data when requested" {
    export REMOVE_DATA="yes"
    
    # Create test data directories
    export HUGINN_DATA_DIR="$HUGINN_TEST_DIR/data"
    export HUGINN_DB_DIR="$HUGINN_TEST_DIR/db"
    export HUGINN_UPLOADS_DIR="$HUGINN_TEST_DIR/uploads"
    
    mkdir -p "$HUGINN_DATA_DIR" "$HUGINN_DB_DIR" "$HUGINN_UPLOADS_DIR"
    
    # Override other functions
    huginn::stop() { return 0; }
    huginn::remove_containers() { return 0; }
    huginn::remove_network() { return 0; }
    flow::is_yes() { return 0; }
    export -f huginn::stop huginn::remove_containers huginn::remove_network flow::is_yes
    
    run huginn::uninstall
    assert_success
    
    assert_file_not_exists "$HUGINN_DATA_DIR"
    assert_file_not_exists "$HUGINN_DB_DIR"
    assert_file_not_exists "$HUGINN_UPLOADS_DIR"
}

@test "install.sh: uninstall removes volumes when requested" {
    export REMOVE_VOLUMES="yes"
    
    local volumes_removed=false
    
    huginn::remove_volumes() {
        volumes_removed=true
        return 0
    }
    
    # Override other functions
    huginn::stop() { return 0; }
    huginn::remove_containers() { return 0; }
    huginn::remove_network() { return 0; }
    flow::is_yes() { return 0; }
    export -f huginn::stop huginn::remove_containers huginn::remove_network huginn::remove_volumes flow::is_yes
    
    run huginn::uninstall
    assert_success
    
    [[ "$volumes_removed" == "true" ]]
}

@test "install.sh: create_containers sets correct environment" {
    local env_vars=""
    
    docker() {
        case "$*" in
            *"run"*huginn/huginn*)
                env_vars="$*"
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::create_containers
    assert_success
    
    # Check key environment variables are set
    [[ "$env_vars" =~ "DATABASE_ADAPTER=postgresql" ]]
    [[ "$env_vars" =~ "DATABASE_HOST=huginn-postgres" ]]
    [[ "$env_vars" =~ "DATABASE_NAME=huginn" ]]
}
