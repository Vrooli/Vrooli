#!/usr/bin/env bats

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/docker.sh"
VAULT_DIR="$BATS_TEST_DIRNAME/.."
COMMON_PATH="$BATS_TEST_DIRNAME/common.sh"

# Source dependencies
RESOURCES_DIR="$VAULT_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../lib"

# Standard test setup for docker tests
VAULT_DOCKER_TEST_SETUP="
        # Source dependencies
        VAULT_DIR='$VAULT_DIR'
        RESOURCES_DIR='$RESOURCES_DIR'
        HELPERS_DIR='$HELPERS_DIR'
        SCRIPT_PATH='$SCRIPT_PATH'
        COMMON_PATH='$COMMON_PATH'
        
        source \"\$HELPERS_DIR/utils/log.sh\"
        source \"\$HELPERS_DIR/utils/system_commands.sh\"
        source \"\$HELPERS_DIR/network/ports.sh\"
        source \"\$RESOURCES_DIR/port_registry.sh\"
        source \"\$RESOURCES_DIR/common.sh\"
        
        # Test environment variables
        export VAULT_CONTAINER_NAME='vault-test'
        export VAULT_NETWORK_NAME='vault-test-network'
        export VAULT_IMAGE='hashicorp/vault:test'
        export VAULT_PORT='8200'
        export VAULT_DATA_DIR='/tmp/vault-test-data'
        export VAULT_CONFIG_DIR='/tmp/vault-test-config'
        export VAULT_LOGS_DIR='/tmp/vault-test-logs'
        export VAULT_MODE='dev'
        export VAULT_DEV_ROOT_TOKEN_ID='test-token'
        export VAULT_DEV_LISTEN_ADDRESS='0.0.0.0:8200'
        export VAULT_BASE_URL='http://localhost:8200'
        
        # Source scripts
        source \"\$COMMON_PATH\"
        source \"\$SCRIPT_PATH\"
"

# ============================================================================
# Network Management Tests
# ============================================================================

@test "vault::docker::create_network creates network when not exists" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        docker() {
            if [[ \"\$1 \$2 \$3\" == 'network inspect vault-test-network' ]]; then
                return 1  # Network doesn't exist
            elif [[ \"\$1 \$2 \$3\" == 'network create vault-test-network' ]]; then
                echo 'network-id'
                return 0
            fi
        }
        export -f docker
        vault::docker::create_network
    "
    [ "$status" -eq 0 ]
}

@test "vault::docker::create_network skips when network exists" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        docker() {
            if [[ \"\$1 \$2 \$3\" == 'network inspect vault-test-network' ]]; then
                return 0  # Network exists
            elif [[ \"\$1 \$2\" == 'network create' ]]; then
                echo 'Should not be called' >&2
                return 1
            fi
        }
        export -f docker
        vault::docker::create_network
    "
    [ "$status" -eq 0 ]
    [[ ! "$output" =~ "Should not be called" ]]
}

@test "vault::docker::remove_network removes existing network" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        docker() {
            if [[ \"\$1 \$2 \$3\" == 'network inspect vault-test-network' ]]; then
                return 0  # Network exists
            elif [[ \"\$1 \$2 \$3\" == 'network rm vault-test-network' ]]; then
                return 0
            fi
        }
        export -f docker
        vault::docker::remove_network
    "
    [ "$status" -eq 0 ]
}

# ============================================================================
# Image Management Tests
# ============================================================================

@test "vault::docker::pull_image pulls the configured image" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        docker() {
            if [[ \"\$1 \$2\" == 'pull hashicorp/vault:test' ]]; then
                echo 'Pulling image...'
                return 0
            fi
            return 1
        }
        export -f docker
        vault::docker::pull_image
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Pulling image" ]]
}

# ============================================================================
# Container Start Tests
# ============================================================================

@test "vault::docker::start_container returns 0 when already running" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_running() { return 0; }
        vault::message() { echo \"\$2\"; }
        vault::docker::start_container
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "MSG_VAULT_ALREADY_RUNNING" ]]
}

@test "vault::docker::start_container creates network and directories" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_running() { return 1; }
        vault::message() { : ; }
        vault::docker::create_network() { echo 'Network created'; }
        vault::ensure_directories() { echo 'Directories ensured'; }
        vault::create_config() { echo 'Config created'; }
        vault::wait_for_health() { return 0; }
        docker() {
            if [[ \"\$1\" == 'run' ]]; then
                echo 'container-id'
                return 0
            fi
        }
        export -f docker
        vault::docker::start_container 2>&1 | grep -E 'Network created|Directories ensured|Config created'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Network created" ]]
    [[ "$output" =~ "Directories ensured" ]]
    [[ "$output" =~ "Config created" ]]
}

@test "vault::docker::start_container runs container with correct args" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_running() { return 1; }
        vault::message() { : ; }
        vault::docker::create_network() { : ; }
        vault::ensure_directories() { : ; }
        vault::create_config() { : ; }
        vault::wait_for_health() { return 0; }
        docker() {
            if [[ \"\$1\" == 'run' ]]; then
                echo \"Docker args: \$@\"
                echo 'container-id'
                return 0
            fi
        }
        export -f docker
        vault::docker::start_container 2>&1 | grep 'Docker args'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--name vault-test" ]]
    [[ "$output" =~ "--network vault-test-network" ]]
    [[ "$output" =~ "--publish 8200:8200" ]]
}

# ============================================================================
# Container Stop Tests
# ============================================================================

@test "vault::docker::stop_container returns 0 when not running" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_running() { return 1; }
        vault::message() { echo \"\$2\"; }
        vault::docker::stop_container
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "MSG_VAULT_NOT_RUNNING" ]]
}

@test "vault::docker::stop_container stops running container" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_running() { return 0; }
        vault::message() { : ; }
        docker() {
            if [[ \"\$1 \$2\" == 'stop vault-test' ]]; then
                return 0
            fi
            return 1
        }
        export -f docker
        vault::docker::stop_container
    "
    [ "$status" -eq 0 ]
}

# ============================================================================
# Container Restart Tests
# ============================================================================

@test "vault::docker::restart_container calls stop and start" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::message() { : ; }
        vault::docker::stop_container() { echo 'Stopped'; return 0; }
        vault::docker::start_container() { echo 'Started'; return 0; }
        sleep() { : ; }  # Mock sleep
        vault::docker::restart_container
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Stopped" ]]
    [[ "$output" =~ "Started" ]]
}

# ============================================================================
# Container Remove Tests
# ============================================================================

@test "vault::docker::remove_container stops if running then removes" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_running() { return 0; }
        vault::docker::stop_container() { echo 'Container stopped'; }
        vault::is_installed() { return 0; }
        docker() {
            if [[ \"\$1 \$2\" == 'rm vault-test' ]]; then
                return 0
            fi
        }
        export -f docker
        vault::docker::remove_container
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Container stopped" ]]
}

# ============================================================================
# Log Management Tests
# ============================================================================

@test "vault::docker::show_logs shows container logs" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_installed() { return 0; }
        docker() {
            if [[ \"\$1\" == 'logs' ]]; then
                echo 'Container logs here'
                return 0
            fi
        }
        export -f docker
        vault::docker::show_logs 50
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Container logs here" ]]
}

@test "vault::docker::show_logs handles follow flag" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_installed() { return 0; }
        docker() {
            if [[ \"\$1\" == 'logs' ]] && [[ \"\$*\" =~ '--follow' ]]; then
                echo 'Following logs'
                return 0
            fi
        }
        export -f docker
        vault::docker::show_logs 50 follow
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Following logs" ]]
}

# ============================================================================
# Container Exec Tests
# ============================================================================

@test "vault::docker::exec executes command in container" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_running() { return 0; }
        docker() {
            if [[ \"\$1 \$2 \$3\" == 'exec -it vault-test' ]] && [[ \"\$4\" == 'vault' ]]; then
                echo 'Command executed'
                return 0
            fi
        }
        export -f docker
        vault::docker::exec vault status
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Command executed" ]]
}

@test "vault::docker::exec fails when not running" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_running() { return 1; }
        vault::docker::exec vault status
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]]
}

# ============================================================================
# Resource Usage Tests
# ============================================================================

@test "vault::docker::get_resource_usage shows container stats" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_running() { return 0; }
        docker() {
            if [[ \"\$1\" == 'stats' ]]; then
                echo 'CONTAINER  CPU%  MEM USAGE'
                echo 'vault-test  5%   100MB'
                return 0
            fi
        }
        export -f docker
        vault::docker::get_resource_usage
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "vault-test" ]]
    [[ "$output" =~ "5%" ]]
}

# ============================================================================
# Cleanup Tests
# ============================================================================

@test "vault::docker::cleanup removes container and network" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::docker::remove_container() { echo 'Container removed'; }
        docker() {
            case \"\$*\" in
                'network inspect vault-test-network')
                    return 0  # Network exists
                    ;;
                \"network inspect vault-test-network --format {{len .Containers}}\")
                    echo '0'  # No containers
                    return 0
                    ;;
            esac
        }
        export -f docker
        vault::docker::remove_network() { echo 'Network removed'; }
        vault::docker::cleanup
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Container removed" ]]
    [[ "$output" =~ "Network removed" ]]
}

# ============================================================================
# Prerequisites Tests
# ============================================================================

@test "vault::docker::check_prerequisites checks Docker installation" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        command() {
            if [[ \"\$2\" == 'docker' ]]; then
                return 0
            fi
            return 1
        }
        docker() {
            if [[ \"\$1\" == 'info' ]] || [[ \"\$1\" == 'ps' ]]; then
                return 0
            fi
        }
        export -f command docker
        vault::docker::check_prerequisites
    "
    [ "$status" -eq 0 ]
}

@test "vault::docker::check_prerequisites fails without Docker" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        command() { return 1; }
        export -f command
        vault::docker::check_prerequisites
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Docker is not installed" ]]
}

# ============================================================================
# Backup/Restore Tests
# ============================================================================

@test "vault::docker::backup creates backup archive" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::is_installed() { return 0; }
        vault::message() { : ; }
        tar() {
            if [[ \"\$1\" == '-czf' ]]; then
                echo 'Backup created'
                return 0
            fi
        }
        export -f tar
        vault::docker::backup /tmp/test-backup.tar.gz
    "
    [ "$status" -eq 0 ]
}

@test "vault::docker::restore extracts backup archive" {
    run bash -c "
        $VAULT_DOCKER_TEST_SETUP
        vault::message() { : ; }
        vault::is_running() { return 0; }
        vault::docker::stop_container() { return 0; }
        tar() {
            if [[ \"\$1\" == '-xzf' ]]; then
                echo 'Backup restored'
                return 0
            fi
        }
        export -f tar
        touch /tmp/test-backup.tar.gz
        vault::docker::restore /tmp/test-backup.tar.gz
        rm -f /tmp/test-backup.tar.gz
    "
    [ "$status" -eq 0 ]
}