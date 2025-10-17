#!/usr/bin/env bats
# Tests for Judge0 install.sh functions

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"
    
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
    export JUDGE0_LOGS_DIR="/tmp/judge0-test/logs"
    export JUDGE0_SUBMISSIONS_DIR="/tmp/judge0-test/submissions"
    export JUDGE0_API_KEY="test_api_key_12345"
    export JUDGE0_WORKERS_COUNT="2"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$JUDGE0_CONFIG_DIR"
    mkdir -p "$JUDGE0_LOGS_DIR"
    mkdir -p "$JUDGE0_SUBMISSIONS_DIR"
    
    # Mock system functions
    
    system::check_port() {
        local port="$1"
        if [[ "$port" == "2358" ]]; then
            return 1  # Port is free
        fi
        return 0
    }
    
    # Mock docker commands
    
    # Mock docker-compose commands
    docker-compose() {
        case "$1" in
            "up")
                echo "Creating network \"${JUDGE0_NETWORK_NAME}\""
                echo "Creating volume \"${JUDGE0_VOLUME_NAME}\""
                echo "Creating judge0-db ..."
                echo "Creating judge0-redis ..."
                echo "Creating judge0-server ..."
                echo "Creating judge0-workers ..."
                echo "judge0-db started"
                echo "judge0-redis started"
                echo "judge0-server started"
                echo "judge0-workers started"
                ;;
            "ps")
                echo "      Name                     Command               State           Ports"
                echo "judge0-server      /scripts/server.sh           Up      0.0.0.0:2358->2358/tcp"
                echo "judge0-workers     /scripts/workers.sh          Up"
                echo "judge0-db          postgres                     Up      5432/tcp"
                echo "judge0-redis       redis-server                 Up      6379/tcp"
                ;;
            *) echo "DOCKER_COMPOSE: $*" ;;
        esac
        return 0
    }
    
    # Mock curl for health checks
    
    # Mock openssl for API key generation
    openssl() {
        echo "abc123def456ghi789jkl012mno345pq"
    }
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".version"*) echo "1.13.1" ;;
            *".hostname"*) echo "judge0-server" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock log functions
    
    # Mock Judge0 functions from other modules
    judge0::create_directories() { echo "Directories created"; }
    judge0::generate_api_key() { echo "abc123def456ghi789jkl012mno345pq"; }
    judge0::save_api_key() { echo "API key saved"; }
    judge0::docker::check_docker() { return 0; }
    judge0::docker::create_network() { echo "Network created"; }
    judge0::docker::create_volume() { echo "Volume created"; }
    judge0::docker::pull_images() { echo "Images pulled"; }
    judge0::api::health_check() { return 0; }
    
    # Load configuration and messages
    source "${JUDGE0_DIR}/config/defaults.sh"
    source "${JUDGE0_DIR}/config/messages.sh"
    judge0::export_config
    judge0::export_messages
    
    # Load the functions to test
    source "${JUDGE0_DIR}/lib/install.sh"
}

# Cleanup after each test
teardown() {
    trash::safe_remove "$JUDGE0_DATA_DIR" --test-cleanup
}

# Test prerequisites check
@test "judge0::install::check_prerequisites verifies system requirements" {
    result=$(judge0::install::check_prerequisites)
    
    [[ "$result" =~ "prerequisites" ]] || [[ "$result" =~ "requirements" ]]
    [[ "$result" =~ "Docker" ]]
}

# Test prerequisites check with missing Docker
@test "judge0::install::check_prerequisites detects missing Docker" {
    # Override Docker check to fail
    judge0::docker::check_docker() { return 1; }
    
    run judge0::install::check_prerequisites
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "Docker" ]]
}

# Test port availability check
@test "judge0::install::check_port_availability verifies port is free" {
    result=$(judge0::install::check_port_availability)
    
    [[ "$result" =~ "port" ]] || [[ "$result" =~ "available" ]]
    [[ "$result" =~ "2358" ]]
}

# Test port availability check with port in use
@test "judge0::install::check_port_availability detects port conflicts" {
    # Override port check to indicate port in use
    system::check_port() {
        local port="$1"
        if [[ "$port" == "2358" ]]; then
            return 0  # Port is in use
        fi
        return 1
    }
    
    run judge0::install::check_port_availability
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "in use" ]]
}

# Test configuration setup
@test "judge0::install::setup_configuration creates configuration files" {
    result=$(judge0::install::setup_configuration)
    
    [[ "$result" =~ "configuration" ]] || [[ "$result" =~ "setup" ]]
    [ -f "${JUDGE0_CONFIG_DIR}/api_key" ]
    [ -f "${JUDGE0_CONFIG_DIR}/docker-compose.yml" ]
}

# Test API key generation during setup
@test "judge0::install::setup_configuration generates API key" {
    result=$(judge0::install::setup_configuration)
    
    [[ "$result" =~ "API key" ]] || [[ "$result" =~ "generated" ]]
    [ -f "${JUDGE0_CONFIG_DIR}/api_key" ]
    
    # Verify API key file permissions
    local perms=$(stat -c %a "${JUDGE0_CONFIG_DIR}/api_key" 2>/dev/null || stat -f %A "${JUDGE0_CONFIG_DIR}/api_key")
    [[ "$perms" =~ ^6[0-9][0-9]$ ]]  # Should be 600 or similar
}

# Test Docker Compose file generation
@test "judge0::install::create_compose_file generates docker-compose.yml" {
    result=$(judge0::install::create_compose_file)
    
    [[ "$result" =~ "compose" ]] || [[ "$result" =~ "generated" ]]
    [ -f "${JUDGE0_CONFIG_DIR}/docker-compose.yml" ]
    
    # Verify compose file structure
    grep -q "version:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    grep -q "services:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    grep -q "judge0-server:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    grep -q "judge0-workers:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    grep -q "judge0-db:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    grep -q "judge0-redis:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
}

# Test Docker infrastructure setup
@test "judge0::install::setup_docker_infrastructure creates Docker resources" {
    result=$(judge0::install::setup_docker_infrastructure)
    
    [[ "$result" =~ "Docker" ]] || [[ "$result" =~ "infrastructure" ]]
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "volume" ]]
}

# Test service installation
@test "judge0::install::install_services deploys Judge0 services" {
    result=$(judge0::install::install_services)
    
    [[ "$result" =~ "service" ]] || [[ "$result" =~ "install" ]]
    [[ "$result" =~ "started" ]] || [[ "$result" =~ "deploy" ]]
}

# Test service startup wait
@test "judge0::install::wait_for_startup waits for services to be ready" {
    result=$(judge0::install::wait_for_startup 10)
    
    [[ "$result" =~ "ready" ]] || [[ "$result" =~ "startup" ]]
}

# Test service startup timeout
@test "judge0::install::wait_for_startup handles startup timeout" {
    # Override health check to always fail
    judge0::api::health_check() { return 1; }
    
    run judge0::install::wait_for_startup 1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "timeout" ]] || [[ "$output" =~ "failed" ]]
}

# Test post-installation validation
@test "judge0::install::validate_installation verifies successful installation" {
    result=$(judge0::install::validate_installation)
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "installation" ]]
    [[ "$result" =~ "success" ]] || [[ "$result" =~ "complete" ]]
}

# Test complete installation workflow
@test "judge0::install::install performs complete installation" {
    result=$(judge0::install::install)
    
    [[ "$result" =~ "install" ]]
    [[ "$result" =~ "Judge0" ]]
    [[ "$result" =~ "success" ]] || [[ "$result" =~ "complete" ]]
}

# Test installation with custom worker count
@test "judge0::install::install supports custom worker count" {
    export JUDGE0_WORKERS_COUNT="4"
    
    result=$(judge0::install::install)
    
    [[ "$result" =~ "install" ]]
    [[ "$result" =~ "4" ]] || [[ "$result" =~ "worker" ]]
}

# Test installation rollback on failure
@test "judge0::install::rollback_installation cleans up failed installation" {
    result=$(judge0::install::rollback_installation)
    
    [[ "$result" =~ "rollback" ]] || [[ "$result" =~ "cleanup" ]]
    [[ "$result" =~ "installation" ]]
}

# Test configuration backup during installation
@test "judge0::install::backup_existing_config backs up existing configuration" {
    # Create existing config
    echo "existing config" > "${JUDGE0_CONFIG_DIR}/existing.conf"
    
    result=$(judge0::install::backup_existing_config)
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "existing" ]]
}

# Test configuration restoration
@test "judge0::install::restore_config_backup restores configuration backup" {
    result=$(judge0::install::restore_config_backup "/tmp/config_backup.tar.gz")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "backup" ]]
}

# Test uninstallation
@test "judge0::install::uninstall removes Judge0 installation" {
    export YES="yes"
    
    result=$(judge0::install::uninstall)
    
    [[ "$result" =~ "uninstall" ]] || [[ "$result" =~ "remove" ]]
    [[ "$result" =~ "Judge0" ]]
}

# Test uninstallation confirmation
@test "judge0::install::uninstall respects user confirmation" {
    export YES="no"
    
    result=$(judge0::install::uninstall)
    
    [[ "$result" =~ "cancelled" ]] || [[ "$result" =~ "aborted" ]]
}

# Test selective uninstallation
@test "judge0::install::uninstall_components removes specific components" {
    export YES="yes"
    
    result=$(judge0::install::uninstall_components "containers")
    
    [[ "$result" =~ "containers" ]]
    [[ "$result" =~ "removed" ]] || [[ "$result" =~ "uninstall" ]]
}

# Test upgrade preparation
@test "judge0::install::prepare_upgrade prepares for version upgrade" {
    result=$(judge0::install::prepare_upgrade "1.14.0")
    
    [[ "$result" =~ "upgrade" ]] || [[ "$result" =~ "prepare" ]]
    [[ "$result" =~ "1.14.0" ]]
}

# Test upgrade execution
@test "judge0::install::perform_upgrade executes version upgrade" {
    result=$(judge0::install::perform_upgrade "1.14.0")
    
    [[ "$result" =~ "upgrade" ]] || [[ "$result" =~ "perform" ]]
    [[ "$result" =~ "1.14.0" ]]
}

# Test installation verification
@test "judge0::install::verify_installation checks installation integrity" {
    result=$(judge0::install::verify_installation)
    
    [[ "$result" =~ "verify" ]] || [[ "$result" =~ "installation" ]]
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "integrity" ]]
}

# Test resource requirements check
@test "judge0::install::check_resources verifies system resources" {
    result=$(judge0::install::check_resources)
    
    [[ "$result" =~ "resource" ]] || [[ "$result" =~ "system" ]]
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "disk" ]]
}

# Test dependency installation
@test "judge0::install::install_dependencies installs required dependencies" {
    result=$(judge0::install::install_dependencies)
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "install" ]]
}

# Test security setup
@test "judge0::install::setup_security configures security settings" {
    result=$(judge0::install::setup_security)
    
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "configure" ]]
}

# Test monitoring setup
@test "judge0::install::setup_monitoring configures monitoring" {
    result=$(judge0::install::setup_monitoring)
    
    [[ "$result" =~ "monitoring" ]] || [[ "$result" =~ "configure" ]]
}

# Test logging setup
@test "judge0::install::setup_logging configures logging" {
    result=$(judge0::install::setup_logging)
    
    [[ "$result" =~ "logging" ]] || [[ "$result" =~ "configure" ]]
}

# Test network configuration
@test "judge0::install::configure_network sets up network configuration" {
    result=$(judge0::install::configure_network)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "configure" ]]
}

# Test firewall configuration
@test "judge0::install::configure_firewall sets up firewall rules" {
    result=$(judge0::install::configure_firewall)
    
    [[ "$result" =~ "firewall" ]] || [[ "$result" =~ "configure" ]]
}

# Test installation documentation
@test "judge0::install::generate_documentation creates installation docs" {
    result=$(judge0::install::generate_documentation)
    
    [[ "$result" =~ "documentation" ]] || [[ "$result" =~ "generate" ]]
}

# Test installation summary
@test "judge0::install::show_installation_summary displays installation details" {
    result=$(judge0::install::show_installation_summary)
    
    [[ "$result" =~ "summary" ]] || [[ "$result" =~ "installation" ]]
    [[ "$result" =~ "Judge0" ]]
}