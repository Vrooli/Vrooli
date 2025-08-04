#!/usr/bin/env bats
# Tests for Judge0 docker.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export JUDGE0_PORT="2358"
    export JUDGE0_CONTAINER_NAME="judge0-test"
    export JUDGE0_WORKERS_NAME="judge0-workers-test"
    export JUDGE0_NETWORK_NAME="judge0-network-test"
    export JUDGE0_VOLUME_NAME="judge0-data-test"
    export JUDGE0_DATA_DIR="/tmp/judge0-test"
    export JUDGE0_CONFIG_DIR="/tmp/judge0-test/config"
    export JUDGE0_IMAGE="judge0/judge0"
    export JUDGE0_VERSION="1.13.1"
    export JUDGE0_WORKERS_COUNT="2"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$JUDGE0_CONFIG_DIR"
    
    # Mock system functions
    
    # Mock docker commands
}
EOF
                fi
                ;;
            "logs")
                if [[ "$*" =~ "judge0-test" ]]; then
                    echo "Judge0 API server startup complete"
                    echo "Server listening on 0.0.0.0:2358"
                    echo "Database connection established"
                    echo "Worker pool initialized"
                fi
                ;;
            "stats")
                echo "CONTAINER       CPU %     MEM USAGE / LIMIT     MEM %     NET I/O             BLOCK I/O           PIDS"
                echo "judge0-test     5.50%     512MiB / 2GiB         25.00%    1.2MB / 800KB       45MB / 25MB         25"
                ;;
            "network")
                case "$2" in
                    "create")
                        echo "Network created: $JUDGE0_NETWORK_NAME"
                        ;;
                    "ls")
                        echo "NETWORK ID     NAME                    DRIVER    SCOPE"
                        echo "abc123def456   judge0-network-test     bridge    local"
                        ;;
                    "rm")
                        echo "Network removed: $JUDGE0_NETWORK_NAME"
                        ;;
                esac
                ;;
            "volume")
                case "$2" in
                    "create")
                        echo "Volume created: $JUDGE0_VOLUME_NAME"
                        ;;
                    "ls")
                        echo "DRIVER    VOLUME NAME"
                        echo "local     judge0-data-test"
                        ;;
                    "rm")
                        echo "Volume removed: $JUDGE0_VOLUME_NAME"
                        ;;
                esac
                ;;
            "pull")
                echo "Pulling image: ${JUDGE0_IMAGE}:${JUDGE0_VERSION}"
                ;;
            "run")
                echo "Starting container: $JUDGE0_CONTAINER_NAME"
                ;;
            "stop")
                echo "Stopping container: $*"
                ;;
            "rm")
                echo "Removing container: $*"
                ;;
            "exec")
                echo "DOCKER_EXEC: $*"
                ;;
            *) echo "DOCKER: $*" ;;
        esac
        return 0
    }
    
    # Mock docker-compose commands
    docker-compose() {
        case "$1" in
            "up")
                echo "Starting Judge0 services..."
                echo "judge0-server started"
                echo "judge0-workers started"
                echo "judge0-db started"
                echo "judge0-redis started"
                ;;
            "down")
                echo "Stopping Judge0 services..."
                ;;
            "ps")
                echo "    Name                   Command               State           Ports"
                echo "judge0-server    /bin/sh -c ./scripts/server   Up      0.0.0.0:2358->2358/tcp"
                echo "judge0-workers   /bin/sh -c ./scripts/workers  Up"
                echo "judge0-db        postgres                      Up      5432/tcp"
                echo "judge0-redis     redis-server                  Up      6379/tcp"
                ;;
            "logs")
                echo "judge0-server    | API server ready"
                echo "judge0-workers   | Workers initialized"
                echo "judge0-db        | PostgreSQL ready"
                echo "judge0-redis     | Redis ready"
                ;;
            *) echo "DOCKER_COMPOSE: $*" ;;
        esac
        return 0
    }
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".State.Running"*) echo "true" ;;
            *".State.Status"*) echo "running" ;;
            *".Config.Image"*) echo "judge0/judge0:1.13.1" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock log functions
    log::info() { echo "INFO: $1"; }
    log::error() { echo "ERROR: $1"; }
    log::warn() { echo "WARN: $1"; }
    log::success() { echo "SUCCESS: $1"; }
    log::debug() { echo "DEBUG: $1"; }
    log::header() { echo "=== $1 ==="; }
    
    # Load configuration and messages
    source "${JUDGE0_DIR}/config/defaults.sh"
    source "${JUDGE0_DIR}/config/messages.sh"
    judge0::export_config
    judge0::export_messages
    
    # Load the functions to test
    source "${JUDGE0_DIR}/lib/docker.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$JUDGE0_DATA_DIR"
}

# Test Docker availability check
@test "judge0::docker::check_docker checks Docker availability" {
    result=$(judge0::docker::check_docker)
    
    [[ "$result" =~ "Docker" ]] || [[ "$result" =~ "available" ]]
}

# Test Docker availability check with missing Docker
@test "judge0::docker::check_docker handles missing Docker" {
    # Override system check to fail
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    run judge0::docker::check_docker
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "not found" ]]
}

# Test image pulling
@test "judge0::docker::pull_images pulls required Docker images" {
    result=$(judge0::docker::pull_images)
    
    [[ "$result" =~ "pull" ]] || [[ "$result" =~ "image" ]]
    [[ "$result" =~ "$JUDGE0_IMAGE" ]]
}

# Test container existence check
@test "judge0::docker::container_exists checks if container exists" {
    result=$(judge0::docker::container_exists && echo "exists" || echo "not found")
    
    [[ "$result" == "exists" ]]
}

# Test container existence check with missing container
@test "judge0::docker::container_exists handles missing container" {
    # Override docker ps to return empty
    docker() {
        case "$1" in
            "ps") echo "" ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(judge0::docker::container_exists && echo "exists" || echo "not found")
    
    [[ "$result" == "not found" ]]
}

# Test container running check
@test "judge0::docker::is_running checks if container is running" {
    result=$(judge0::docker::is_running && echo "running" || echo "stopped")
    
    [[ "$result" == "running" ]]
}

# Test container running check with stopped container
@test "judge0::docker::is_running handles stopped container" {
    # Override docker inspect to show stopped state
    docker() {
        case "$1" in
            "inspect")
                echo '{"State":{"Running":false,"Status":"exited"}}'
                ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(judge0::docker::is_running && echo "running" || echo "stopped")
    
    [[ "$result" == "stopped" ]]
}

# Test server container start
@test "judge0::docker::start_server starts Judge0 server container" {
    result=$(judge0::docker::start_server)
    
    [[ "$result" =~ "start" ]] || [[ "$result" =~ "server" ]]
    [[ "$result" =~ "$JUDGE0_CONTAINER_NAME" ]]
}

# Test worker containers start
@test "judge0::docker::start_workers starts Judge0 worker containers" {
    result=$(judge0::docker::start_workers)
    
    [[ "$result" =~ "worker" ]] || [[ "$result" =~ "start" ]]
    [[ "$result" =~ "$JUDGE0_WORKERS_COUNT" ]]
}

# Test container stop
@test "judge0::docker::stop_containers stops Judge0 containers" {
    result=$(judge0::docker::stop_containers)
    
    [[ "$result" =~ "stop" ]] || [[ "$result" =~ "container" ]]
}

# Test container removal
@test "judge0::docker::remove_containers removes Judge0 containers" {
    export YES="yes"
    
    result=$(judge0::docker::remove_containers)
    
    [[ "$result" =~ "remov" ]] || [[ "$result" =~ "container" ]]
}

# Test container restart
@test "judge0::docker::restart_containers restarts Judge0 containers" {
    result=$(judge0::docker::restart_containers)
    
    [[ "$result" =~ "restart" ]] || [[ "$result" =~ "container" ]]
}

# Test network creation
@test "judge0::docker::create_network creates Docker network" {
    result=$(judge0::docker::create_network)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "creat" ]]
    [[ "$result" =~ "$JUDGE0_NETWORK_NAME" ]]
}

# Test network removal
@test "judge0::docker::remove_network removes Docker network" {
    result=$(judge0::docker::remove_network)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "remov" ]]
    [[ "$result" =~ "$JUDGE0_NETWORK_NAME" ]]
}

# Test volume creation
@test "judge0::docker::create_volume creates Docker volume" {
    result=$(judge0::docker::create_volume)
    
    [[ "$result" =~ "volume" ]] || [[ "$result" =~ "creat" ]]
    [[ "$result" =~ "$JUDGE0_VOLUME_NAME" ]]
}

# Test volume removal
@test "judge0::docker::remove_volume removes Docker volume" {
    result=$(judge0::docker::remove_volume)
    
    [[ "$result" =~ "volume" ]] || [[ "$result" =~ "remov" ]]
    [[ "$result" =~ "$JUDGE0_VOLUME_NAME" ]]
}

# Test Docker Compose file generation
@test "judge0::docker::generate_compose_file creates docker-compose.yml" {
    result=$(judge0::docker::generate_compose_file)
    
    [[ "$result" =~ "compose" ]] || [[ "$result" =~ "generated" ]]
    [ -f "${JUDGE0_CONFIG_DIR}/docker-compose.yml" ]
    
    # Verify compose file contains required services
    grep -q "version:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    grep -q "judge0-server:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    grep -q "judge0-workers:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
}

# Test Docker Compose services start
@test "judge0::docker::compose_up starts services via Docker Compose" {
    # Create minimal compose file
    cat > "${JUDGE0_CONFIG_DIR}/docker-compose.yml" <<EOF
version: '3.8'
services:
  judge0-server:
    image: judge0/judge0:1.13.1
    ports:
      - "2358:2358"
EOF
    
    result=$(judge0::docker::compose_up)
    
    [[ "$result" =~ "start" ]] || [[ "$result" =~ "service" ]]
}

# Test Docker Compose services stop
@test "judge0::docker::compose_down stops services via Docker Compose" {
    result=$(judge0::docker::compose_down)
    
    [[ "$result" =~ "stop" ]] || [[ "$result" =~ "service" ]]
}

# Test container logs
@test "judge0::docker::get_logs retrieves container logs" {
    result=$(judge0::docker::get_logs)
    
    [[ "$result" =~ "startup complete" ]] || [[ "$result" =~ "Server listening" ]]
}

# Test container logs with tail option
@test "judge0::docker::get_logs supports tail option" {
    result=$(judge0::docker::get_logs 10)
    
    [[ "$result" =~ "startup complete" ]] || [[ "$result" =~ "logs" ]]
}

# Test container statistics
@test "judge0::docker::get_stats shows container resource usage" {
    result=$(judge0::docker::get_stats)
    
    [[ "$result" =~ "CPU" ]] || [[ "$result" =~ "MEM" ]]
    [[ "$result" =~ "judge0-test" ]]
}

# Test container health check
@test "judge0::docker::health_check verifies container health" {
    result=$(judge0::docker::health_check)
    
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "running" ]]
}

# Test container execution
@test "judge0::docker::exec_command executes commands in container" {
    result=$(judge0::docker::exec_command "echo 'test'")
    
    [[ "$result" =~ "DOCKER_EXEC" ]]
    [[ "$result" =~ "echo 'test'" ]]
}

# Test image cleanup
@test "judge0::docker::cleanup_images removes unused Judge0 images" {
    result=$(judge0::docker::cleanup_images)
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "image" ]]
}

# Test complete cleanup
@test "judge0::docker::cleanup_all removes all Judge0 Docker resources" {
    export YES="yes"
    
    result=$(judge0::docker::cleanup_all)
    
    [[ "$result" =~ "cleanup" ]]
    [[ "$result" =~ "container" ]] || [[ "$result" =~ "network" ]] || [[ "$result" =~ "volume" ]]
}

# Test worker scaling
@test "judge0::docker::scale_workers adjusts worker container count" {
    result=$(judge0::docker::scale_workers 4)
    
    [[ "$result" =~ "scale" ]] || [[ "$result" =~ "worker" ]]
    [[ "$result" =~ "4" ]]
}

# Test container backup
@test "judge0::docker::backup_containers creates container backup" {
    result=$(judge0::docker::backup_containers "/tmp/judge0_backup.tar")
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "export" ]]
}

# Test container restoration
@test "judge0::docker::restore_containers restores containers from backup" {
    result=$(judge0::docker::restore_containers "/tmp/judge0_backup.tar")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "import" ]]
}

# Test resource monitoring
@test "judge0::docker::monitor_resources monitors container resources" {
    result=$(judge0::docker::monitor_resources)
    
    [[ "$result" =~ "monitor" ]] || [[ "$result" =~ "resource" ]]
}

# Test container update
@test "judge0::docker::update_containers updates Judge0 containers" {
    result=$(judge0::docker::update_containers)
    
    [[ "$result" =~ "update" ]] || [[ "$result" =~ "container" ]]
}

# Test service status
@test "judge0::docker::service_status shows comprehensive service status" {
    result=$(judge0::docker::service_status)
    
    [[ "$result" =~ "service" ]] || [[ "$result" =~ "status" ]]
    [[ "$result" =~ "judge0" ]]
}

# Test environment validation
@test "judge0::docker::validate_environment validates Docker environment" {
    result=$(judge0::docker::validate_environment)
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "environment" ]]
}

# Test port conflict detection
@test "judge0::docker::check_port_conflicts detects port conflicts" {
    result=$(judge0::docker::check_port_conflicts)
    
    [[ "$result" =~ "port" ]] || [[ "$result" =~ "conflict" ]]
}

# Test security configuration
@test "judge0::docker::configure_security sets up container security" {
    result=$(judge0::docker::configure_security)
    
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "configured" ]]
}

# Test performance tuning
@test "judge0::docker::tune_performance optimizes container performance" {
    result=$(judge0::docker::tune_performance)
    
    [[ "$result" =~ "performance" ]] || [[ "$result" =~ "tuned" ]]
}

# Test diagnostic collection
@test "judge0::docker::collect_diagnostics gathers diagnostic information" {
    result=$(judge0::docker::collect_diagnostics)
    
    [[ "$result" =~ "diagnostic" ]] || [[ "$result" =~ "collected" ]]
}