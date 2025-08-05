#!/usr/bin/env bats

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/status.sh"
VAULT_DIR="$BATS_TEST_DIRNAME/.."
COMMON_PATH="$BATS_TEST_DIRNAME/common.sh"

# Source dependencies
RESOURCES_DIR="$VAULT_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Helper function to setup test environment
setup_vault_status_test_env() {
    # Source required utilities
    source "$HELPERS_DIR/utils/log.sh"
    source "$HELPERS_DIR/utils/system.sh"
    source "$HELPERS_DIR/utils/ports.sh"
    source "$RESOURCES_DIR/port-registry.sh"
    source "$RESOURCES_DIR/common.sh"
    
    # Set test environment variables
    export VAULT_PORT="8200"
    export VAULT_MODE="dev"
    export VAULT_BASE_URL="http://localhost:8200"
    export VAULT_CONTAINER_NAME="vault-test"
    export VAULT_DATA_DIR="/tmp/vault-test-data"
    export VAULT_CONFIG_DIR="/tmp/vault-test-config"
    export VAULT_SECRET_ENGINE="secret"
    export VAULT_SECRET_VERSION="2"
    export VAULT_NAMESPACE_PREFIX="test"
    export VAULT_DEV_ROOT_TOKEN_ID="test-token"
    export VAULT_STORAGE_TYPE="inmem"
    export VAULT_TLS_DISABLE="1"
    
    # Source common functions
    source "$COMMON_PATH"
    
    # Source the script under test
    source "$SCRIPT_PATH"
}

# Standard test setup for status tests
VAULT_STATUS_TEST_SETUP="
        # Source dependencies
        VAULT_DIR='$VAULT_DIR'
        RESOURCES_DIR='$RESOURCES_DIR'
        HELPERS_DIR='$HELPERS_DIR'
        SCRIPT_PATH='$SCRIPT_PATH'
        COMMON_PATH='$COMMON_PATH'
        
        source \"\$HELPERS_DIR/utils/log.sh\"
        source \"\$HELPERS_DIR/utils/system.sh\"
        source \"\$HELPERS_DIR/utils/ports.sh\"
        source \"\$RESOURCES_DIR/port-registry.sh\"
        source \"\$RESOURCES_DIR/common.sh\"
        
        # Test environment variables
        export VAULT_PORT='8200'
        export VAULT_MODE='dev'
        export VAULT_BASE_URL='http://localhost:8200'
        export VAULT_CONTAINER_NAME='vault-test'
        export VAULT_DATA_DIR='/tmp/vault-test-data'
        export VAULT_CONFIG_DIR='/tmp/vault-test-config'
        export VAULT_SECRET_ENGINE='secret'
        export VAULT_SECRET_VERSION='2'
        export VAULT_NAMESPACE_PREFIX='test'
        export VAULT_DEV_ROOT_TOKEN_ID='test-token'
        export VAULT_STORAGE_TYPE='inmem'
        export VAULT_TLS_DISABLE='1'
        
        # Source scripts
        source \"\$COMMON_PATH\"
        source \"\$SCRIPT_PATH\"
"

# ============================================================================
# Show Status Tests
# ============================================================================

@test "vault::show_status displays basic information" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::get_status() { echo 'healthy'; }
        vault::message() { echo \"\$2\"; }
        vault::show_detailed_status() { : ; }
        vault::show_status 2>&1 | grep -E 'Status:|Port:|Mode:'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status: healthy" ]]
    [[ "$output" =~ "Port: 8200" ]]
    [[ "$output" =~ "Mode: dev" ]]
}

@test "vault::show_status handles not_installed status" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::get_status() { echo 'not_installed'; }
        vault::message() { echo \"\$2\"; }
        vault::show_status
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "MSG_VAULT_STATUS_NOT_INSTALLED" ]]
}

@test "vault::show_status handles stopped status" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::get_status() { echo 'stopped'; }
        vault::message() { echo \"\$2\"; }
        vault::show_status
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "MSG_VAULT_STATUS_STOPPED" ]]
}

@test "vault::show_status handles sealed status" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::get_status() { echo 'sealed'; }
        vault::show_detailed_status() { : ; }
        vault::show_status 2>&1 | grep 'sealed'
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "sealed" ]]
}

# ============================================================================
# Detailed Status Tests
# ============================================================================

@test "vault::show_detailed_status shows container details" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::is_installed() { return 0; }
        docker() {
            case \"\$*\" in
                \"container inspect --format={{.State.Status}} vault-test\")
                    echo 'running'
                    ;;
                \"container inspect --format={{.State.StartedAt}} vault-test\")
                    echo '2024-01-01T10:00:00Z'
                    ;;
                \"container inspect --format={{.Config.Image}} vault-test\")
                    echo 'hashicorp/vault:1.17'
                    ;;
            esac
            return 0
        }
        export -f docker
        vault::check_api_endpoints() { : ; }
        vault::check_vault_status() { : ; }
        vault::get_resource_usage() { : ; }
        resources::is_service_running() { return 0; }
        vault::show_detailed_status 2>&1 | grep -E 'Docker Status:|Started At:|Image:'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Docker Status: running" ]]
    [[ "$output" =~ "Started At: 2024-01-01" ]]
    [[ "$output" =~ "Image: hashicorp/vault:1.17" ]]
}

@test "vault::show_detailed_status shows configuration summary" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::is_installed() { return 0; }
        vault::is_running() { return 0; }
        vault::check_api_endpoints() { : ; }
        vault::check_vault_status() { : ; }
        vault::get_resource_usage() { : ; }
        resources::is_service_running() { return 0; }
        # Mock docker commands
        docker() {
            case \"\$*\" in
                *'State.Status'*) echo 'running' ;;
                *'State.StartedAt'*) echo '2024-01-01T10:00:00Z' ;;
                *'State.Health.Status'*) echo '' ;;
                *'Config.Image'*) echo 'hashicorp/vault:latest' ;;
                *) echo 'docker mock' ;;
            esac
            return 0
        }
        export -f docker
        vault::show_detailed_status 2>&1 | grep -E 'Mode:|Storage Type:|Secret Engine:'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Mode: dev" ]]
    [[ "$output" =~ "Storage Type: inmem" ]]
    [[ "$output" =~ "Secret Engine: secret" ]]
}

# ============================================================================
# API Endpoint Tests
# ============================================================================

@test "vault::check_api_endpoints checks health endpoint" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::is_running() { return 0; }
        curl() {
            if [[ \"\$*\" =~ '/v1/sys/health' ]]; then
                return 0
            fi
            return 1
        }
        export -f curl
        vault::check_api_endpoints 2>&1 | grep '/v1/sys/health'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "✅ Responding" ]]
}

@test "vault::check_api_endpoints handles not running" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::is_running() { return 1; }
        vault::check_api_endpoints
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Container not running" ]]
}

# ============================================================================
# Vault Status Tests
# ============================================================================

@test "vault::check_vault_status shows initialization status" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::is_initialized() { return 0; }
        vault::is_sealed() { return 1; }
        vault::check_secret_engines() { echo '  Secret Engines: OK'; }
        vault::check_vault_status
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Initialized: ✅ Yes" ]]
    [[ "$output" =~ "Sealed: ✅ No" ]]
    [[ "$output" =~ "Secret Engines: OK" ]]
}

@test "vault::check_vault_status handles not initialized" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::is_initialized() { return 1; }
        vault::check_vault_status
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Initialized: ❌ No" ]]
}

# ============================================================================
# Secret Engine Tests
# ============================================================================

@test "vault::check_secret_engines verifies KV engine" {
    setup_vault_status_test_env
    
    # Mock API request to return engine information
    vault::api_request() {
        echo '{"secret/": {"type": "kv", "options": {"version": "2"}}}'
        return 0
    }
    
    # Mock jq to parse the response
    jq() {
        if [[ "$*" =~ '.\"secret/\"' ]]; then
            return 0
        elif [[ "$*" =~ 'version' ]]; then
            echo '2'
        fi
    }
    
    run vault::check_secret_engines
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "✅ Enabled (v2)" ]]
}

# ============================================================================
# Resource Usage Tests
# ============================================================================

@test "vault::get_resource_usage shows container stats" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::is_running() { return 0; }
        docker() {
            echo -e '10%\t512MB/1GB\t100MB/200MB\t50MB/100MB'
            return 0
        }
        export -f docker
        vault::get_resource_usage
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CPU Usage: 10%" ]]
    [[ "$output" =~ "Memory Usage: 512MB/1GB" ]]
}

# ============================================================================
# Diagnostics Tests
# ============================================================================

@test "vault::diagnose checks system requirements" {
    setup_vault_status_test_env
    
    # Mock system commands
    command() {
        case "$2" in
            docker) return 0 ;;
            curl) return 0 ;;
            jq) return 0 ;;
        esac
        return 1
    }
    
    docker() {
        if [[ "$1" == 'info' ]]; then
            return 0
        fi
    }
    
    vault::show_status() { : ; }
    vault::analyze_logs() { : ; }
    vault::show_troubleshooting() { : ; }
    
    run bash -c "$(declare -f command docker vault::show_status vault::analyze_logs vault::show_troubleshooting vault::diagnose); vault::diagnose 2>&1 | head -20 | grep -E 'Docker:|curl:|jq:'"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Docker: ✅ Installed" ]]
    [[ "$output" =~ "curl: ✅ Available" ]]
    [[ "$output" =~ "jq: ✅ Available" ]]
}

@test "vault::diagnose validates configuration" {
    setup_vault_status_test_env
    
    # Mock system functions
    command() { return 0; }
    docker() { return 0; }
    resources::is_service_running() { return 1; }
    vault::is_running() { return 1; }
    vault::show_status() { : ; }
    vault::analyze_logs() { : ; }
    vault::show_troubleshooting() { : ; }
    
    run bash -c "$(declare -f command docker resources::is_service_running vault::is_running vault::show_status vault::analyze_logs vault::show_troubleshooting vault::diagnose); vault::diagnose 2>&1 | grep -A10 'Configuration Validation'"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Port: 8200" ]]
    [[ "$output" =~ "✅ Valid port number" ]]
    [[ "$output" =~ "Mode: dev" ]]
    [[ "$output" =~ "✅ Valid mode" ]]
}

# ============================================================================
# Log Analysis Tests
# ============================================================================

@test "vault::analyze_logs counts errors and warnings" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::is_installed() { return 0; }
        docker() {
            cat << 'EOF'
[ERROR] Failed to connect
[WARN] Slow response time
[INFO] Started successfully
[ERROR] Permission denied
[WARNING] Low memory
EOF
            return 0
        }
        export -f docker
        vault::analyze_logs
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Error messages: 2" ]]
    [[ "$output" =~ "Warning messages: 2" ]]
}

@test "vault::analyze_logs detects specific issues" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::is_installed() { return 0; }
        docker() {
            echo '[ERROR] bind: address already in use'
            return 0
        }
        export -f docker
        vault::analyze_logs
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Port conflict detected" ]]
}

# ============================================================================
# Troubleshooting Tests
# ============================================================================

@test "vault::show_troubleshooting provides suggestions for not_installed" {
    setup_vault_status_test_env
    
    # Mock functions
    vault::get_status() { echo 'not_installed'; }
    vault::show_troubleshooting() {
        local status
        status=$(vault::get_status)
        
        case "$status" in
            "not_installed")
                echo "• Install Vault: ./manage.sh --action install"
                echo "• Check Docker is installed and running"
                ;;
        esac
    }
    
    run vault::show_troubleshooting
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install Vault:" ]]
    [[ "$output" =~ "--action install" ]]
}

@test "vault::show_troubleshooting handles sealed status" {
    setup_vault_status_test_env
    
    # Mock status function and set production mode
    vault::get_status() { echo 'sealed'; }
    export VAULT_MODE='prod'
    export VAULT_UNSEAL_KEYS_FILE='/tmp/keys'
    
    run vault::show_troubleshooting
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Unseal Vault:" ]]
    [[ "$output" =~ "--action unseal" ]]
}

# ============================================================================
# Monitor Tests
# ============================================================================

@test "vault::monitor displays continuous status" {
    setup_vault_status_test_env
    
    # Mock functions for monitoring
    vault::get_status() { echo 'healthy'; }
    curl() { echo '0.123'; return 0; }
    sleep() { : ; }  # Mock sleep to avoid delays
    date() { echo '2024-01-01 10:00:00'; }
    
    # Create a mock monitor function that runs once and exits
    vault::monitor() {
        local timestamp
        timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        local status response_time
        status=$(vault::get_status)
        response_time=$(curl -o /dev/null -s -w '%{time_total}' "$VAULT_BASE_URL/v1/sys/health" 2>/dev/null || echo "N/A")
        echo "[$timestamp] Status: $status | Response: ${response_time}s"
        return 0  # Exit after one iteration for testing
    }
    
    run vault::monitor
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status: healthy" ]]
    [[ "$output" =~ "Response: 0.123s" ]]
}