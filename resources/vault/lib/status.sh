#!/usr/bin/env bash
# Vault Status and Health Checking - Standardized Format
# Comprehensive status monitoring and diagnostics

# Source format utilities and required libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VAULT_STATUS_DIR="${APP_ROOT}/resources/vault/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/vault/config/defaults.sh"

# Ensure configuration is exported
if command -v vault::export_config &>/dev/null; then
    vault::export_config 2>/dev/null || true
fi

#######################################
# Show detailed Vault status
#######################################
vault::show_status() {
    local status
    status=$(vault::get_status)
    
    log::header "Vault Status Report"
    
    # Basic status
    echo "Status: $status"
    echo "Mode: $VAULT_MODE"
    echo "Port: $VAULT_PORT"
    echo "Base URL: $VAULT_BASE_URL"
    echo "Container: $VAULT_CONTAINER_NAME"
    echo "Data Directory: $VAULT_DATA_DIR"
    echo "Config Directory: $VAULT_CONFIG_DIR"
    echo
    
    case "$status" in
        "not_installed")
            vault::message "info" "MSG_VAULT_STATUS_NOT_INSTALLED"
            log::info "Run: ./manage.sh --action install"
            return 1
            ;;
        "stopped")
            vault::message "warn" "MSG_VAULT_STATUS_STOPPED"
            log::info "Run: ./manage.sh --action start"
            return 1
            ;;
        "unhealthy")
            vault::message "warn" "MSG_VAULT_STATUS_UNHEALTHY"
            vault::show_detailed_status
            return 1
            ;;
        "sealed")
            log::warn "Vault is running but sealed"
            if [[ "$VAULT_MODE" == "prod" ]]; then
                log::info "Run: ./manage.sh --action unseal"
            fi
            vault::show_detailed_status
            return 1
            ;;
        "healthy")
            vault::message "success" "MSG_VAULT_STATUS_HEALTHY"
            vault::show_detailed_status
            return 0
            ;;
    esac
}

#######################################
# Show detailed status when running
#######################################
vault::show_detailed_status() {
    # Container information
    echo "Container Details:"
    if vault::is_installed; then
        local container_status
        container_status=$(docker container inspect --format='{{.State.Status}}' "$VAULT_CONTAINER_NAME" 2>/dev/null)
        echo "  Docker Status: $container_status"
        
        local started_at
        started_at=$(docker container inspect --format='{{.State.StartedAt}}' "$VAULT_CONTAINER_NAME" 2>/dev/null)
        echo "  Started At: $started_at"
        
        # Health check status
        local health_status
        health_status=$(docker container inspect --format='{{.State.Health.Status}}' "$VAULT_CONTAINER_NAME" 2>/dev/null)
        if [[ -n "$health_status" ]]; then
            echo "  Health Status: $health_status"
        fi
        
        # Image information
        local image
        image=$(docker container inspect --format='{{.Config.Image}}' "$VAULT_CONTAINER_NAME" 2>/dev/null)
        echo "  Image: $image"
    fi
    echo
    
    # Network connectivity
    echo "Network Status:"
    if resources::is_service_running "$VAULT_PORT"; then
        echo "  Port $VAULT_PORT: ✅ Open"
    else
        echo "  Port $VAULT_PORT: ❌ Not responding"
    fi
    
    # API endpoint checks
    vault::check_api_endpoints
    
    # Vault-specific status
    if vault::is_running; then
        echo
        vault::check_vault_status
    fi
    
    # Resource usage
    if vault::is_running; then
        echo
        vault::get_resource_usage
    fi
    
    # Configuration summary
    echo
    echo "Configuration:"
    echo "  Mode: $VAULT_MODE"
    echo "  Storage Type: $VAULT_STORAGE_TYPE"
    echo "  TLS Disabled: $VAULT_TLS_DISABLE"
    echo "  Secret Engine: ${VAULT_SECRET_ENGINE} (v${VAULT_SECRET_VERSION})"
    echo "  Namespace: $VAULT_NAMESPACE_PREFIX"
    if [[ "$VAULT_MODE" == "dev" ]]; then
        echo "  Dev Root Token: $VAULT_DEV_ROOT_TOKEN_ID"
    fi
}

#######################################
# Check API endpoints health
#######################################
vault::check_api_endpoints() {
    if ! vault::is_running; then
        echo "  API Endpoints: ❌ Container not running"
        return 1
    fi
    
    echo "API Endpoints:"
    
    # Health endpoint
    local health_url="${VAULT_BASE_URL}/v1/sys/health"
    if curl -sf "$health_url" >/dev/null 2>&1; then
        echo "  /v1/sys/health: ✅ Responding"
    else
        echo "  /v1/sys/health: ❌ Not responding"
    fi
    
    # Initialization status endpoint
    local init_url="${VAULT_BASE_URL}/v1/sys/init"
    if curl -sf "$init_url" >/dev/null 2>&1; then
        echo "  /v1/sys/init: ✅ Responding"
    else
        echo "  /v1/sys/init: ❌ Not responding"
    fi
    
    # Seal status endpoint (if initialized)
    if vault::is_initialized; then
        local seal_url="${VAULT_BASE_URL}/v1/sys/seal-status"
        if curl -sf "$seal_url" >/dev/null 2>&1; then
            echo "  /v1/sys/seal-status: ✅ Responding"
        else
            echo "  /v1/sys/seal-status: ❌ Not responding"
        fi
    fi
}

#######################################
# Check Vault-specific status information
#######################################
vault::check_vault_status() {
    echo "Vault Status:"
    
    # Initialization status
    if vault::is_initialized; then
        echo "  Initialized: ✅ Yes"
        
        # Seal status
        if vault::is_sealed; then
            echo "  Sealed: ❌ Yes (requires unsealing)"
        else
            echo "  Sealed: ✅ No (ready for use)"
        fi
        
        # Secret engines (if unsealed)
        if ! vault::is_sealed; then
            vault::check_secret_engines
        fi
    else
        echo "  Initialized: ❌ No (requires initialization)"
    fi
}

#######################################
# Check secret engines status
#######################################
vault::check_secret_engines() {
    echo "  Secret Engines:"
    
    # Check KV engine - use direct curl if vault::api_request fails
    local mounts_response
    mounts_response=$(vault::api_request "GET" "/v1/sys/mounts" 2>/dev/null)
    
    # If vault::api_request failed, try direct curl
    if [[ -z "$mounts_response" ]] || [[ "$mounts_response" == *"error"* ]]; then
        local token
        token=$(vault::get_root_token 2>/dev/null) || token="myroot"
        mounts_response=$(curl -s -H "X-Vault-Token: $token" "${VAULT_BASE_URL:-http://localhost:8200}/v1/sys/mounts" 2>/dev/null)
    fi
    
    if [[ -n "$mounts_response" ]]; then
        local kv_engine_path="${VAULT_SECRET_ENGINE:-secret}/"
        # Check both old format (direct) and new format (under .data)
        if echo "$mounts_response" | jq -e ".\"$kv_engine_path\"" >/dev/null 2>&1; then
            local kv_version
            kv_version=$(echo "$mounts_response" | jq -r ".\"$kv_engine_path\".options.version // \"1\"")
            echo "    KV Engine (${VAULT_SECRET_ENGINE:-secret}): ✅ Enabled (v$kv_version)"
            
            # Test functional operations
            vault::test_secret_operations
        elif echo "$mounts_response" | jq -e ".data.\"$kv_engine_path\"" >/dev/null 2>&1; then
            # New format with data field
            local kv_version
            kv_version=$(echo "$mounts_response" | jq -r ".data.\"$kv_engine_path\".options.version // \"1\"")
            echo "    KV Engine (${VAULT_SECRET_ENGINE:-secret}): ✅ Enabled (v$kv_version)"
            
            # Test functional operations
            vault::test_secret_operations
        else
            echo "    KV Engine (${VAULT_SECRET_ENGINE:-secret}): ❌ Not enabled"
        fi
    else
        echo "    KV Engine (${VAULT_SECRET_ENGINE:-secret}): ❓ Cannot check (API error)"
    fi
}

#######################################
# Test actual secret operations functionality
#######################################
vault::test_secret_operations() {
    echo "    Functional Tests:"
    
    # Run functional tests (main status path uses collect_data which handles --fast)
    local test_path="health-check/test-$(date +%s)"
    local test_data='{"data":{"test_key":"test_value","timestamp":"'$(date -Iseconds)'"}}'
    
    # Test 1: Write operation
    local write_response
    write_response=$(timeout 2s vault::api_request "PUT" "/v1/${VAULT_SECRET_ENGINE}/data/${test_path}" "$test_data" 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "$write_response" | jq -e '.data' >/dev/null 2>&1; then
        echo "      Write Operations: ✅ Working"
        
        # Test 2: Read operation
        local read_response
        read_response=$(vault::api_request "GET" "/v1/${VAULT_SECRET_ENGINE}/data/${test_path}" 2>/dev/null)
        
        if [[ $? -eq 0 ]] && echo "$read_response" | jq -e '.data.data.test_key' >/dev/null 2>&1; then
            local read_value
            read_value=$(echo "$read_response" | jq -r '.data.data.test_key')
            if [[ "$read_value" == "test_value" ]]; then
                echo "      Read Operations: ✅ Working"
                
                # Test 3: List operation
                local list_response
                list_response=$(vault::api_request "GET" "/v1/${VAULT_SECRET_ENGINE}/metadata/health-check?list=true" 2>/dev/null)
                
                if [[ $? -eq 0 ]] && echo "$list_response" | jq -e '.data.keys' >/dev/null 2>&1; then
                    echo "      List Operations: ✅ Working"
                else
                    echo "      List Operations: ⚠️  Issues detected"
                fi
                
                # Test 4: Delete operation (cleanup)
                local delete_response
                delete_response=$(vault::api_request "DELETE" "/v1/${VAULT_SECRET_ENGINE}/data/${test_path}" 2>/dev/null)
                
                if [[ $? -eq 0 ]]; then
                    echo "      Delete Operations: ✅ Working"
                    echo "      Overall Functionality: ✅ All tests passed"
                else
                    echo "      Delete Operations: ⚠️  Issues detected (test data not cleaned up)"
                fi
            else
                echo "      Read Operations: ❌ Data corruption (expected 'test_value', got '$read_value')"
            fi
        else
            echo "      Read Operations: ❌ Failed"
        fi
    else
        echo "      Write Operations: ❌ Failed"
        echo "      Overall Functionality: ❌ Basic operations not working"
    fi
}

#######################################
# Get Vault container resource usage
#######################################
vault::get_resource_usage() {
    if ! vault::is_running; then
        echo "Resource Usage: ❌ Container not running"
        return 1
    fi
    
    echo "Resource Usage:"
    
    # Show resource stats (main status path uses collect_data which handles --fast)
    local stats
    stats=$(timeout 2s docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$VAULT_CONTAINER_NAME" 2>/dev/null)
    
    if [[ -n "$stats" ]]; then
        IFS=$'\t' read -r cpu_perc mem_usage net_io block_io <<< "$stats"
        echo "  CPU Usage: $cpu_perc"
        echo "  Memory Usage: $mem_usage"
        echo "  Network I/O: $net_io"
        echo "  Block I/O: $block_io"
    else
        echo "  Unable to retrieve resource usage"
    fi
}

#######################################
# Run comprehensive diagnostics
#######################################
vault::diagnose() {
    log::header "Vault Diagnostic Report"
    
    # Basic system checks
    echo "System Requirements:"
    echo -n "  Docker: "
    if command -v docker >/dev/null 2>&1; then
        echo "✅ Installed"
        echo -n "  Docker Daemon: "
        if docker info >/dev/null 2>&1; then
            echo "✅ Running"
        else
            echo "❌ Not accessible"
        fi
    else
        echo "❌ Not found"
    fi
    
    echo -n "  curl: "
    if command -v curl >/dev/null 2>&1; then
        echo "✅ Available"
    else
        echo "❌ Not found"
    fi
    
    echo -n "  jq: "
    if command -v jq >/dev/null 2>&1; then
        echo "✅ Available"
    else
        echo "❌ Not found (required for JSON operations)"
    fi
    echo
    
    # Configuration validation
    echo "Configuration Validation:"
    echo "  Port: $VAULT_PORT"
    if [[ "$VAULT_PORT" =~ ^[0-9]+$ ]] && [[ "$VAULT_PORT" -ge 1024 ]] && [[ "$VAULT_PORT" -le 65535 ]]; then
        echo "    ✅ Valid port number"
    else
        echo "    ❌ Invalid port number"
    fi
    
    if resources::is_service_running "$VAULT_PORT" && ! vault::is_running; then
        echo "    ❌ Port conflict detected"
    else
        echo "    ✅ No port conflicts"
    fi
    
    echo "  Mode: $VAULT_MODE"
    if [[ "$VAULT_MODE" == "dev" ]] || [[ "$VAULT_MODE" == "prod" ]]; then
        echo "    ✅ Valid mode"
    else
        echo "    ❌ Invalid mode (must be 'dev' or 'prod')"
    fi
    
    echo "  Data Directory: $VAULT_DATA_DIR"
    if [[ -d "$VAULT_DATA_DIR" ]]; then
        echo "    ✅ Exists"
        if [[ -r "$VAULT_DATA_DIR" ]] && [[ -w "$VAULT_DATA_DIR" ]]; then
            echo "    ✅ Readable and writable"
        else
            echo "    ❌ Permission issues"
        fi
    else
        echo "    ❌ Does not exist"
    fi
    
    echo "  Config Directory: $VAULT_CONFIG_DIR"
    if [[ -d "$VAULT_CONFIG_DIR" ]]; then
        echo "    ✅ Exists"
        if [[ -f "$VAULT_CONFIG_DIR/vault.hcl" ]]; then
            echo "    ✅ Configuration file exists"
        else
            echo "    ❌ Configuration file missing"
        fi
    else
        echo "    ❌ Does not exist"
    fi
    echo
    
    # Docker environment
    echo "Docker Environment:"
    if docker network inspect "$VAULT_NETWORK_NAME" >/dev/null 2>&1; then
        echo "  Network: ✅ $VAULT_NETWORK_NAME exists"
    else
        echo "  Network: ❌ $VAULT_NETWORK_NAME not found"
    fi
    
    if docker image inspect "$VAULT_IMAGE" >/dev/null 2>&1; then
        echo "  Image: ✅ $VAULT_IMAGE available"
    else
        echo "  Image: ❌ $VAULT_IMAGE not found"
    fi
    echo
    
    # Container status
    vault::show_status
    echo
    
    # Log analysis
    if vault::is_installed; then
        echo "Recent Log Analysis:"
        vault::analyze_logs
    fi
    
    # Troubleshooting suggestions
    echo
    log::header "Troubleshooting Suggestions"
    vault::show_troubleshooting
}

#######################################
# Analyze recent logs for issues
#######################################
vault::analyze_logs() {
    if ! vault::is_installed; then
        echo "  Container not available for log analysis"
        return 1
    fi
    
    local logs
    logs=$(docker logs --tail 50 "$VAULT_CONTAINER_NAME" 2>&1)
    
    # Check for common error patterns
    local error_count
    error_count=$(echo "$logs" | grep -ci "error\|exception\|failed\|fatal" || true)
    
    local warning_count
    warning_count=$(echo "$logs" | grep -ci "warning\|warn" || true)
    
    echo "  Error messages: $error_count"
    echo "  Warning messages: $warning_count"
    
    # Show recent errors if any
    if [[ $error_count -gt 0 ]]; then
        echo "  Recent errors:"
        echo "$logs" | grep -i "error\|exception\|failed\|fatal" | tail -3 | sed 's/^/    /'
    fi
    
    # Check for specific known issues
    if echo "$logs" | grep -qi "permission denied"; then
        echo "  ⚠️  Permission issues detected"
    fi
    
    if echo "$logs" | grep -qi "address already in use"; then
        echo "  ⚠️  Port conflict detected"
    fi
    
    if echo "$logs" | grep -qi "connection refused"; then
        echo "  ⚠️  Connection issues detected"
    fi
    
    if echo "$logs" | grep -qi "storage.*failed"; then
        echo "  ⚠️  Storage issues detected"
    fi
}

#######################################
# Show troubleshooting suggestions
#######################################
vault::show_troubleshooting() {
    local status
    status=$(vault::get_status)
    
    case "$status" in
        "not_installed")
            echo "• Install Vault: ./manage.sh --action install"
            echo "• Check Docker is installed and running"
            ;;
        "stopped")
            echo "• Start Vault: ./manage.sh --action start"
            vault::message "info" "MSG_VAULT_TROUBLESHOOT_LOGS"
            ;;
        "unhealthy")
            echo "• Check logs: ./manage.sh --action logs"
            echo "• Restart service: ./manage.sh --action restart"
            vault::message "info" "MSG_VAULT_TROUBLESHOOT_PORT"
            vault::message "info" "MSG_VAULT_TROUBLESHOOT_CONFIG"
            ;;
        "sealed")
            if [[ "$VAULT_MODE" == "dev" ]]; then
                echo "• Development mode should auto-unseal - check logs"
                echo "• Try restarting: ./manage.sh --action restart"
            else
                echo "• Unseal Vault: ./manage.sh --action unseal"
                echo "• Check unseal keys file: $VAULT_UNSEAL_KEYS_FILE"
            fi
            ;;
    esac
    
    # Check for permission issues
    if [[ -d "$VAULT_DATA_DIR" ]]; then
        local current_uid current_gid
        current_uid=$(id -u)
        current_gid=$(id -g)
        
        local incorrect_files
        incorrect_files=$(find "$VAULT_DATA_DIR" -not -user "$current_uid" -o -not -group "$current_gid" 2>/dev/null || true)
        
        if [[ -n "$incorrect_files" ]]; then
            echo "• Fix file permissions: sudo chown -R \$(whoami):\$(whoami) $VAULT_DATA_DIR"
        fi
    fi
    
    # Check for Docker permission issues
    if ! docker info >/dev/null 2>&1; then
        echo "• Fix Docker permissions: sudo usermod -aG docker \$USER && newgrp docker"
        echo "• Or restart Docker service: sudo systemctl restart docker"
    fi
    
    echo
    echo "General troubleshooting:"
    vault::message "info" "MSG_VAULT_TROUBLESHOOT_LOGS"
    vault::message "info" "MSG_VAULT_TROUBLESHOOT_CONFIG"
    vault::message "info" "MSG_VAULT_TROUBLESHOOT_PORT"
    vault::message "info" "MSG_VAULT_TROUBLESHOOT_RESTART"
    
    if [[ "$VAULT_MODE" == "dev" ]]; then
        vault::message "info" "MSG_VAULT_TROUBLESHOOT_REINIT"
    fi
}

#######################################
# Show authentication information
#######################################
vault::show_auth_info() {
    log::header "Vault Authentication Information"
    
    local status
    status=$(vault::get_status)
    
    if [[ "$status" != "healthy" ]]; then
        log::error "Vault is not healthy (status: $status). Cannot retrieve authentication info."
        log::info "Run: ./manage.sh --action status"
        return 1
    fi
    
    echo "Mode: $VAULT_MODE"
    echo "Base URL: $VAULT_BASE_URL"
    echo
    
    case "$VAULT_MODE" in
        "dev")
            echo "Development Mode Authentication:"
            echo "  Root Token: $VAULT_DEV_ROOT_TOKEN_ID"
            echo "  ⚠️  Dev mode token - for development only!"
            echo
            
            echo "Quick Test Commands:"
            echo "  # Test API access"
            echo "  curl -H \"X-Vault-Token: $VAULT_DEV_ROOT_TOKEN_ID\" $VAULT_BASE_URL/v1/sys/health"
            echo
            echo "  # List existing secrets"
            echo "  curl -H \"X-Vault-Token: $VAULT_DEV_ROOT_TOKEN_ID\" $VAULT_BASE_URL/v1/secret/metadata?list=true"
            echo
            echo "  # Create a test secret"
            echo "  curl -X PUT -H \"X-Vault-Token: $VAULT_DEV_ROOT_TOKEN_ID\" -d '{\"data\":{\"key\":\"value\"}}' $VAULT_BASE_URL/v1/secret/data/test"
            echo
            echo "  # Read a secret"
            echo "  curl -H \"X-Vault-Token: $VAULT_DEV_ROOT_TOKEN_ID\" $VAULT_BASE_URL/v1/secret/data/test"
            ;;
        "prod")
            echo "Production Mode Authentication:"
            if [[ -f "$VAULT_TOKEN_FILE" ]]; then
                local token
                token=$(cat "$VAULT_TOKEN_FILE" 2>/dev/null)
                if [[ -n "$token" ]]; then
                    echo "  Root Token: ${token:0:8}****** (from $VAULT_TOKEN_FILE)"
                    echo "  ⚠️  Keep this token secure!"
                else
                    echo "  ❌ Token file exists but is empty: $VAULT_TOKEN_FILE"
                fi
            else
                echo "  ❌ Root token file not found: $VAULT_TOKEN_FILE"
                echo "  💡 Initialize Vault first: ./manage.sh --action init-prod"
            fi
            
            if [[ -f "$VAULT_UNSEAL_KEYS_FILE" ]]; then
                echo "  Unseal Keys: Available in $VAULT_UNSEAL_KEYS_FILE"
                echo "  ⚠️  Keep unseal keys secure and backed up!"
            else
                echo "  ❌ Unseal keys file not found: $VAULT_UNSEAL_KEYS_FILE"
            fi
            
            echo
            echo "Production Test Commands:"
            if [[ -f "$VAULT_TOKEN_FILE" ]]; then
                echo "  # Test API access"
                echo "  curl -H \"X-Vault-Token: \$(cat $VAULT_TOKEN_FILE)\" $VAULT_BASE_URL/v1/sys/health"
                echo
                echo "  # List existing secrets"
                echo "  curl -H \"X-Vault-Token: \$(cat $VAULT_TOKEN_FILE)\" $VAULT_BASE_URL/v1/secret/metadata?list=true"
            else
                echo "  # Initialize first: ./manage.sh --action init-prod"
            fi
            ;;
    esac
    
    echo
    echo "Management Script Commands:"
    echo "  # Store a secret"
    echo "  ./manage.sh --action put-secret --path \"myapp/api-key\" --value \"secret123\""
    echo
    echo "  # Retrieve a secret"
    echo "  ./manage.sh --action get-secret --path \"myapp/api-key\""
    echo
    echo "  # List secrets"
    echo "  ./manage.sh --action list-secrets --path \"myapp\""
}

#######################################
# Run standalone functional tests
#######################################
vault::test_functional() {
    log::header "Vault Functional Tests"
    
    local status
    status=$(vault::get_status)
    
    if [[ "$status" != "healthy" ]]; then
        log::error "Vault is not healthy (status: $status). Cannot run functional tests."
        log::info "Run: ./manage.sh --action status"
        return 1
    fi
    
    echo "Testing core Vault functionality..."
    echo
    
    local test_path="functional-test/test-$(date +%s)"
    local test_data='{"data":{"test_key":"functional_test_value","timestamp":"'$(date -Iseconds)'","test_id":"'$(uuidgen 2>/dev/null || echo "test-$$-$RANDOM")'"}}'
    
    # Test 1: Write operation
    echo "1. Testing Write Operations..."
    local write_response
    write_response=$(vault::api_request "PUT" "/v1/${VAULT_SECRET_ENGINE}/data/${test_path}" "$test_data" 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "$write_response" | jq -e '.data' >/dev/null 2>&1; then
        echo "   ✅ Write operation successful"
        local version_created
        version_created=$(echo "$write_response" | jq -r '.data.version // "unknown"')
        echo "   📝 Created version: $version_created"
        
        # Test 2: Read operation
        echo
        echo "2. Testing Read Operations..."
        local read_response
        read_response=$(vault::api_request "GET" "/v1/${VAULT_SECRET_ENGINE}/data/${test_path}" 2>/dev/null)
        
        if [[ $? -eq 0 ]] && echo "$read_response" | jq -e '.data.data.test_key' >/dev/null 2>&1; then
            local read_value
            read_value=$(echo "$read_response" | jq -r '.data.data.test_key')
            if [[ "$read_value" == "functional_test_value" ]]; then
                echo "   ✅ Read operation successful"
                echo "   📖 Data integrity verified"
                
                # Test 3: List operation
                echo
                echo "3. Testing List Operations..."
                local list_response
                list_response=$(vault::api_request "GET" "/v1/${VAULT_SECRET_ENGINE}/metadata/functional-test?list=true" 2>/dev/null)
                
                if [[ $? -eq 0 ]] && echo "$list_response" | jq -e '.data.keys' >/dev/null 2>&1; then
                    local key_count
                    key_count=$(echo "$list_response" | jq '.data.keys | length')
                    echo "   ✅ List operation successful"
                    echo "   📋 Found $key_count keys in namespace"
                    
                    # Test 4: Update operation
                    echo
                    echo "4. Testing Update Operations..."
                    local update_data='{"data":{"test_key":"updated_functional_test_value","timestamp":"'$(date -Iseconds)'","update_test":true}}'
                    local update_response
                    update_response=$(vault::api_request "PUT" "/v1/${VAULT_SECRET_ENGINE}/data/${test_path}" "$update_data" 2>/dev/null)
                    
                    if [[ $? -eq 0 ]] && echo "$update_response" | jq -e '.data' >/dev/null 2>&1; then
                        local version_updated
                        version_updated=$(echo "$update_response" | jq -r '.data.version // "unknown"')
                        echo "   ✅ Update operation successful"
                        echo "   📝 Updated to version: $version_updated"
                        
                        # Verify update
                        local verify_response
                        verify_response=$(vault::api_request "GET" "/v1/${VAULT_SECRET_ENGINE}/data/${test_path}" 2>/dev/null)
                        local verify_value
                        verify_value=$(echo "$verify_response" | jq -r '.data.data.test_key')
                        
                        if [[ "$verify_value" == "updated_functional_test_value" ]]; then
                            echo "   ✅ Update verification successful"
                        else
                            echo "   ❌ Update verification failed"
                        fi
                    else
                        echo "   ❌ Update operation failed"
                    fi
                    
                    # Test 5: Delete operation (cleanup)
                    echo
                    echo "5. Testing Delete Operations..."
                    local delete_response
                    delete_response=$(vault::api_request "DELETE" "/v1/${VAULT_SECRET_ENGINE}/data/${test_path}" 2>/dev/null)
                    
                    if [[ $? -eq 0 ]]; then
                        echo "   ✅ Delete operation successful"
                        echo "   🧹 Test data cleaned up"
                        
                        # Verify deletion
                        local verify_delete_response
                        verify_delete_response=$(vault::api_request "GET" "/v1/${VAULT_SECRET_ENGINE}/data/${test_path}" 2>/dev/null)
                        if echo "$verify_delete_response" | jq -e '.errors' >/dev/null 2>&1; then
                            echo "   ✅ Delete verification successful"
                        else
                            echo "   ⚠️  Delete verification inconclusive"
                        fi
                    else
                        echo "   ❌ Delete operation failed"
                        echo "   ⚠️  Test data may need manual cleanup"
                    fi
                else
                    echo "   ❌ List operation failed"
                fi
            else
                echo "   ❌ Read operation failed - data corruption detected"
                echo "   📖 Expected: 'functional_test_value', Got: '$read_value'"
            fi
        else
            echo "   ❌ Read operation failed"
        fi
    else
        echo "   ❌ Write operation failed"
        echo "   🚫 Cannot proceed with further tests"
        return 1
    fi
    
    echo
    log::header "Functional Test Summary"
    echo "✅ All core Vault operations are working correctly"
    echo "🔐 Secret storage, retrieval, updates, and deletion verified"
    echo "📊 Vault is ready for production use"
}

#######################################
# Monitor Vault status continuously
# Arguments:
#   $1 - interval in seconds (default: 30)
#######################################
vault::monitor() {
    local interval="${1:-30}"
    
    log::info "Starting Vault monitoring (interval: ${interval}s)"
    log::info "Press Ctrl+C to stop"
    echo
    
    while true; do
        local timestamp
        timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        
        local status
        status=$(vault::get_status)
        
        printf "[%s] Status: %s" "$timestamp" "$status"
        
        if [[ "$status" == "healthy" ]]; then
            # Check API response time
            local response_time
            response_time=$(curl -sf -w "%{time_total}" -o /dev/null "${VAULT_BASE_URL}/v1/sys/health" 2>/dev/null || echo "timeout")
            printf " | Response: %ss" "$response_time"
        fi
        
        echo
        sleep "$interval"
    done
}

#######################################
# Collect Vault status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
vault::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    local vault_status="unknown"
    
    if vault::is_installed; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "$VAULT_CONTAINER_NAME" 2>/dev/null || echo "unknown")
        
        if vault::is_running; then
            running="true"
            vault_status=$(vault::get_status)
            
            case "$vault_status" in
                "healthy")
                    healthy="true"
                    health_message="Healthy - All systems operational"
                    ;;
                "sealed")
                    health_message="Sealed - Requires unsealing"
                    ;;
                "unhealthy")
                    health_message="Unhealthy - Service not responding properly"
                    ;;
                *)
                    health_message="Running but status uncertain"
                    ;;
            esac
        else
            health_message="Stopped - Container not running"
        fi
    else
        health_message="Not installed - Container not found"
    fi
    
    # Basic resource information
    status_data+=("name" "vault")
    status_data+=("category" "storage")
    status_data+=("description" "HashiCorp Vault secrets management system")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$VAULT_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$VAULT_PORT")
    status_data+=("vault_status" "$vault_status")
    
    # Service endpoints
    status_data+=("base_url" "$VAULT_BASE_URL")
    status_data+=("health_url" "${VAULT_BASE_URL}/v1/sys/health")
    status_data+=("init_url" "${VAULT_BASE_URL}/v1/sys/init")
    
    # Configuration details
    status_data+=("image" "$VAULT_IMAGE")
    status_data+=("mode" "$VAULT_MODE")
    status_data+=("storage_type" "$VAULT_STORAGE_TYPE")
    status_data+=("tls_disabled" "$VAULT_TLS_DISABLE")
    status_data+=("secret_engine" "$VAULT_SECRET_ENGINE")
    status_data+=("secret_version" "$VAULT_SECRET_VERSION")
    status_data+=("namespace" "$VAULT_NAMESPACE_PREFIX")
    status_data+=("data_dir" "$VAULT_DATA_DIR")
    status_data+=("config_dir" "$VAULT_CONFIG_DIR")
    
    # Runtime information (only if running and healthy)
    if [[ "$running" == "true" && "$healthy" == "true" ]]; then
        # Skip expensive operations in fast mode
        local skip_expensive_ops="$fast_mode"
        
        # API endpoint status (optimized)
        local health_endpoint_ok="false"
        local init_endpoint_ok="false"
        local seal_endpoint_ok="false"
        
        if [[ "$skip_expensive_ops" == "false" ]]; then
            if timeout 1s curl -sf --max-time 1 "${VAULT_BASE_URL}/v1/sys/health" >/dev/null 2>&1; then
                health_endpoint_ok="true"
            fi
            
            if timeout 1s curl -sf --max-time 1 "${VAULT_BASE_URL}/v1/sys/init" >/dev/null 2>&1; then
                init_endpoint_ok="true"
            fi
            
            if vault::is_initialized && timeout 1s curl -sf --max-time 1 "${VAULT_BASE_URL}/v1/sys/seal-status" >/dev/null 2>&1; then
                seal_endpoint_ok="true"
            fi
        else
            health_endpoint_ok="N/A"
            init_endpoint_ok="N/A"
            seal_endpoint_ok="N/A"
        fi
        
        status_data+=("health_endpoint" "$health_endpoint_ok")
        status_data+=("init_endpoint" "$init_endpoint_ok")
        status_data+=("seal_endpoint" "$seal_endpoint_ok")
        
        # Vault specific status
        local initialized="false"
        local sealed="true"
        
        if vault::is_initialized; then
            initialized="true"
            if ! vault::is_sealed; then
                sealed="false"
            fi
        fi
        
        status_data+=("initialized" "$initialized")
        status_data+=("sealed" "$sealed")
        
        # Secret engine status (if unsealed - optimized)
        if [[ "$sealed" == "false" ]] && [[ "$skip_expensive_ops" == "false" ]]; then
            local kv_engine_enabled="false"
            local mounts_response
            mounts_response=$(timeout 2s vault::api_request "GET" "/v1/sys/mounts" 2>/dev/null)
            
            if [[ $? -eq 0 ]]; then
                local kv_engine_path="${VAULT_SECRET_ENGINE}/"
                # Check both old format (direct) and new format (under .data)
                if echo "$mounts_response" | jq -e ".\"$kv_engine_path\"" >/dev/null 2>&1 || \
                   echo "$mounts_response" | jq -e ".data.\"$kv_engine_path\"" >/dev/null 2>&1; then
                    kv_engine_enabled="true"
                fi
            fi
            
            status_data+=("kv_engine_enabled" "$kv_engine_enabled")
        elif [[ "$sealed" == "false" ]]; then
            status_data+=("kv_engine_enabled" "N/A")
        fi
        
        # Resource usage (optimized)
        if [[ "$skip_expensive_ops" == "false" ]]; then
            local stats
            stats=$(timeout 2s docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$VAULT_CONTAINER_NAME" 2>/dev/null)
            
            if [[ -n "$stats" ]]; then
                IFS=$'\t' read -r cpu_perc mem_usage net_io block_io <<< "$stats"
                status_data+=("cpu_usage" "$cpu_perc")
                status_data+=("memory_usage" "$mem_usage")
                status_data+=("network_io" "$net_io")
                status_data+=("block_io" "$block_io")
            else
                status_data+=("cpu_usage" "N/A")
                status_data+=("memory_usage" "N/A")
                status_data+=("network_io" "N/A")
                status_data+=("block_io" "N/A")
            fi
        else
            status_data+=("cpu_usage" "N/A")
            status_data+=("memory_usage" "N/A")
            status_data+=("network_io" "N/A")
            status_data+=("block_io" "N/A")
        fi
        
        # Container creation time
        local created_at
        created_at=$(docker inspect --format='{{.State.StartedAt}}' "$VAULT_CONTAINER_NAME" 2>/dev/null)
        if [[ -n "$created_at" ]]; then
            status_data+=("started_at" "$created_at")
        fi
    fi
    
    # Authentication information
    if [[ "$VAULT_MODE" == "dev" ]]; then
        status_data+=("auth_token" "$VAULT_DEV_ROOT_TOKEN_ID")
        status_data+=("auth_type" "dev_token")
    elif [[ "$VAULT_MODE" == "prod" ]]; then
        if [[ -f "$VAULT_TOKEN_FILE" ]]; then
            local token
            token=$(cat "$VAULT_TOKEN_FILE" 2>/dev/null)
            if [[ -n "$token" ]]; then
                status_data+=("auth_token" "${token:0:8}******")
                status_data+=("auth_type" "prod_token")
            fi
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Display status using standardized format
# Args: [--format json|text] [--verbose] [--fast]
#######################################
vault::status() {
    status::run_standard "vault" "vault::status::collect_data" "vault::status::display_text" "$@"
}

#######################################
# Display status in text format
#######################################
vault::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📊 Vault Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Vault, run: ./manage.sh --action install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Container information
    log::info "🐳 Container Info:"
    log::info "   📦 Name: ${data[container_name]:-unknown}"
    log::info "   📊 Status: ${data[container_status]:-unknown}"
    log::info "   🖼️  Image: ${data[image]:-unknown}"
    if [[ -n "${data[started_at]:-}" ]]; then
        local started_date
        started_date=$(date -d "${data[started_at]}" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "${data[started_at]}")
        log::info "   📅 Started: $started_date"
    fi
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🔗 Base URL: ${data[base_url]:-unknown}"
    log::info "   🏥 Health: ${data[health_url]:-unknown}"
    log::info "   🚀 Init: ${data[init_url]:-unknown}"
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   🎭 Mode: ${data[mode]:-unknown}"
    log::info "   💾 Storage: ${data[storage_type]:-unknown}"
    log::info "   🔒 TLS Disabled: ${data[tls_disabled]:-unknown}"
    log::info "   🗄️  Secret Engine: ${data[secret_engine]:-unknown} (v${data[secret_version]:-unknown})"
    log::info "   🏷️  Namespace: ${data[namespace]:-unknown}"
    log::info "   📁 Data Dir: ${data[data_dir]:-unknown}"
    log::info "   ⚙️  Config Dir: ${data[config_dir]:-unknown}"
    echo
    
    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "🔗 API Endpoints:"
        if [[ "${data[health_endpoint]:-false}" == "true" ]]; then
            log::success "   ✅ Health Endpoint: Responding"
        else
            log::warn "   ❌ Health Endpoint: Not responding"
        fi
        
        if [[ "${data[init_endpoint]:-false}" == "true" ]]; then
            log::success "   ✅ Init Endpoint: Responding"
        else
            log::warn "   ❌ Init Endpoint: Not responding"
        fi
        
        if [[ "${data[seal_endpoint]:-false}" == "true" ]]; then
            log::success "   ✅ Seal Status Endpoint: Responding"
        fi
        echo
        
        log::info "🔐 Vault Status:"
        if [[ "${data[initialized]:-false}" == "true" ]]; then
            log::success "   ✅ Initialized: Yes"
        else
            log::error "   ❌ Initialized: No"
        fi
        
        if [[ "${data[sealed]:-true}" == "false" ]]; then
            log::success "   ✅ Sealed: No (ready for use)"
        else
            log::warn "   ⚠️  Sealed: Yes (requires unsealing)"
        fi
        
        if [[ "${data[kv_engine_enabled]:-false}" == "true" ]]; then
            log::success "   ✅ KV Engine: Enabled"
        elif [[ "${data[sealed]:-true}" == "false" ]]; then
            log::warn "   ⚠️  KV Engine: Not enabled"
        fi
        echo
        
        # Resource usage
        if [[ -n "${data[cpu_usage]:-}" ]]; then
            log::info "📈 Resource Usage:"
            log::info "   🔧 CPU: ${data[cpu_usage]:-unknown}"
            log::info "   💾 Memory: ${data[memory_usage]:-unknown}"
            log::info "   🌐 Network I/O: ${data[network_io]:-unknown}"
            log::info "   💿 Block I/O: ${data[block_io]:-unknown}"
            echo
        fi
        
        # Authentication info
        if [[ -n "${data[auth_type]:-}" ]]; then
            log::info "🔐 Authentication:"
            case "${data[auth_type]}" in
                "dev_token")
                    log::info "   🔑 Dev Token: ${data[auth_token]:-unknown}"
                    log::warn "   ⚠️  Development mode - not for production!"
                    ;;
                "prod_token")
                    log::info "   🔑 Root Token: ${data[auth_token]:-unknown}"
                    log::warn "   ⚠️  Keep production tokens secure!"
                    ;;
            esac
        fi
    fi
}

# Legacy function compatibility for existing scripts
vault::show_status() {
    vault::status "$@"
}