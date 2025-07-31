#!/usr/bin/env bats
# Tests for Huginn lib/docker.sh

load ../test_fixtures/test_helper

setup() {
    setup_test_environment
    mock_docker "success"
    source_huginn_scripts
}

teardown() {
    teardown_test_environment
}

@test "docker.sh: start succeeds when containers exist but stopped" {
    # First mock as not running, then as success after start
    local call_count=0
    docker() {
        case "$1" in
            "container")
                if [[ "$2" == "inspect" ]]; then
                    if [[ $call_count -eq 0 ]]; then
                        call_count=$((call_count + 1))
                        echo "false"  # Not running initially
                    else
                        echo "true"   # Running after start
                    fi
                fi
                ;;
            "start")
                return 0  # Start succeeds
                ;;
            "ps")
                echo -e "huginn\nhuginn-postgres"
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::start
    assert_success
    assert_output_contains "Starting Huginn"
}

@test "docker.sh: start fails when containers don't exist" {
    mock_docker "not_installed"
    
    run huginn::start
    assert_failure
    assert_output_contains "not installed"
}

@test "docker.sh: stop succeeds when containers are running" {
    mock_docker "success"
    
    run huginn::stop
    assert_success
    assert_output_contains "Stopping Huginn"
}

@test "docker.sh: stop handles already stopped containers" {
    mock_docker "not_running"
    
    # Override container exists check
    huginn::container_exists() { return 0; }
    huginn::db_container_exists() { return 0; }
    export -f huginn::container_exists huginn::db_container_exists
    
    run huginn::stop
    assert_success
    assert_output_contains "already stopped"
}

@test "docker.sh: restart performs stop and start" {
    mock_docker "success"
    
    # Track function calls
    local stop_called=false
    local start_called=false
    
    huginn::stop() {
        stop_called=true
        return 0
    }
    
    huginn::start() {
        start_called=true
        return 0
    }
    
    export -f huginn::stop huginn::start
    
    run huginn::restart
    assert_success
    
    [[ "$stop_called" == "true" ]]
    [[ "$start_called" == "true" ]]
}

@test "docker.sh: remove_containers removes both containers" {
    local containers_removed=()
    
    docker() {
        case "$*" in
            *"rm -f huginn"*) 
                containers_removed+=("huginn")
                return 0
                ;;
            *"rm -f huginn-postgres"*) 
                containers_removed+=("huginn-postgres")
                return 0
                ;;
            *"ps -a --format"*)
                echo -e "huginn\nhuginn-postgres"
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::remove_containers
    assert_success
    
    # Verify both containers were removed
    [[ " ${containers_removed[@]} " =~ " huginn " ]]
    [[ " ${containers_removed[@]} " =~ " huginn-postgres " ]]
}

@test "docker.sh: remove_network removes network" {
    local network_removed=false
    
    docker() {
        case "$*" in
            *"network rm"*huginn-network*) 
                network_removed=true
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
    
    run huginn::remove_network
    assert_success
    [[ "$network_removed" == "true" ]]
}

@test "docker.sh: remove_volumes removes all volumes" {
    local volumes_removed=()
    
    docker() {
        case "$*" in
            *"volume rm"*)
                if [[ "$*" =~ "huginn_postgres_data" ]]; then
                    volumes_removed+=("postgres")
                elif [[ "$*" =~ "huginn_uploads" ]]; then
                    volumes_removed+=("uploads")
                fi
                return 0
                ;;
            *"volume ls"*)
                echo -e "huginn_postgres_data\nhuginn_uploads"
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::remove_volumes
    assert_success
    
    # Verify both volumes were removed
    [[ " ${volumes_removed[@]} " =~ " postgres " ]]
    [[ " ${volumes_removed[@]} " =~ " uploads " ]]
}

@test "docker.sh: view_logs shows container logs" {
    mock_docker "success"
    
    run huginn::view_logs "app" "50" "no"
    assert_success
    assert_output_contains "started successfully"
}

@test "docker.sh: view_logs handles database container" {
    docker() {
        case "$*" in
            *"logs"*huginn-postgres*)
                echo "PostgreSQL started successfully"
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::view_logs "db" "50" "no"
    assert_success
    assert_output_contains "PostgreSQL"
}

@test "docker.sh: get_container_stats returns statistics" {
    docker() {
        case "$*" in
            *"stats"*"--no-stream"*)
                echo "CONTAINER     CPU %     MEM USAGE / LIMIT     MEM %"
                echo "huginn        0.5%      512MiB / 1GiB         50%"
                echo "huginn-postgres  0.1%   64MiB / 512MiB       12.5%"
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::get_container_stats
    assert_success
    assert_output_contains "huginn"
    assert_output_contains "0.5%"
}

@test "docker.sh: cleanup_old_resources removes stopped containers" {
    # Mock showing containers exist but are stopped
    local huginn_removed=false
    local db_removed=false
    
    docker() {
        case "$*" in
            *"ps -a --format"*)
                echo -e "huginn\nhuginn-postgres"
                ;;
            *"ps --format"*)
                echo ""  # No running containers
                ;;
            *"rm huginn"*)
                huginn_removed=true
                return 0
                ;;
            *"rm huginn-postgres"*)
                db_removed=true
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    # Override is_running to return false
    huginn::is_running() { return 1; }
    export -f huginn::is_running
    
    run huginn::cleanup_old_resources
    assert_success
    [[ "$huginn_removed" == "true" ]]
    [[ "$db_removed" == "true" ]]
}
