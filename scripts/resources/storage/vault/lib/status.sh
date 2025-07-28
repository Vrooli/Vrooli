#!/usr/bin/env bash
# Vault Status and Health Checking
# Comprehensive status monitoring and diagnostics

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
    echo "  Secret Engine: $VAULT_SECRET_ENGINE (v$VAULT_SECRET_VERSION)"
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
    
    # Check KV engine
    local mounts_response
    mounts_response=$(vault::api_request "GET" "/v1/sys/mounts" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        local kv_engine_path="${VAULT_SECRET_ENGINE}/"
        if echo "$mounts_response" | jq -e ".\"$kv_engine_path\"" >/dev/null 2>&1; then
            local kv_version
            kv_version=$(echo "$mounts_response" | jq -r ".\"$kv_engine_path\".options.version // \"1\"")
            echo "    KV Engine ($VAULT_SECRET_ENGINE): ✅ Enabled (v$kv_version)"
        else
            echo "    KV Engine ($VAULT_SECRET_ENGINE): ❌ Not enabled"
        fi
    else
        echo "    KV Engine ($VAULT_SECRET_ENGINE): ❓ Cannot check (API error)"
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
    local stats
    stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$VAULT_CONTAINER_NAME" 2>/dev/null)
    
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