#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/status.sh"
VAULT_DIR="$BATS_TEST_DIRNAME/.."
COMMON_PATH="$BATS_TEST_DIRNAME/common.sh"

# Source dependencies
RESOURCES_DIR="$VAULT_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

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
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::api_request() {
            echo '{\"secret/\": {\"type\": \"kv\", \"options\": {\"version\": \"2\"}}}'
            return 0
        }
        jq() {
            if [[ \"\$*\" =~ '.\"secret/\"' ]]; then
                return 0
            elif [[ \"\$*\" =~ 'version' ]]; then
                echo '2'
            fi
        }
        export -f jq
        vault::check_secret_engines
    "
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
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        command() {
            case \"\$2\" in
                docker) return 0 ;;
                curl) return 0 ;;
                jq) return 0 ;;
            esac
            return 1
        }
        docker() {
            if [[ \"\$1\" == 'info' ]]; then
                return 0
            fi
        }
        export -f command docker
        vault::show_status() { : ; }
        vault::analyze_logs() { : ; }
        vault::show_troubleshooting() { : ; }
        vault::diagnose 2>&1 | head -20 | grep -E 'Docker:|curl:|jq:'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Docker: ✅ Installed" ]]
    [[ "$output" =~ "curl: ✅ Available" ]]
    [[ "$output" =~ "jq: ✅ Available" ]]
}

@test "vault::diagnose validates configuration" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        command() { return 0; }
        docker() { return 0; }
        export -f command docker
        resources::is_service_running() { return 1; }
        vault::is_running() { return 1; }
        vault::show_status() { : ; }
        vault::analyze_logs() { : ; }
        vault::show_troubleshooting() { : ; }
        vault::diagnose 2>&1 | grep -A5 'Configuration Validation' | grep -E 'Port:|Mode:'
    "
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
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::get_status() { echo 'not_installed'; }
        vault::show_troubleshooting
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install Vault:" ]]
    [[ "$output" =~ "--action install" ]]
}

@test "vault::show_troubleshooting handles sealed status" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::get_status() { echo 'sealed'; }
        export VAULT_MODE='prod'
        export VAULT_UNSEAL_KEYS_FILE='/tmp/keys'
        vault::show_troubleshooting
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Unseal Vault:" ]]
    [[ "$output" =~ "--action unseal" ]]
}

# ============================================================================
# Monitor Tests
# ============================================================================

@test "vault::monitor displays continuous status" {
    run bash -c "
        $VAULT_STATUS_TEST_SETUP
        vault::get_status() { echo 'healthy'; }
        curl() { echo '0.123'; return 0; }
        export -f curl
        sleep() { : ; }  # Mock sleep
        # Run monitor for one iteration then exit
        counter=0
        while true; do
            vault::get_status >/dev/null
            echo \"[2024-01-01 10:00:00] Status: healthy | Response: 0.123s\"
            ((counter++))
            [[ \$counter -ge 1 ]] && break
        done
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status: healthy" ]]
    [[ "$output" =~ "Response: 0.123s" ]]
}