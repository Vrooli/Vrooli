#!/usr/bin/env bash
# Enhanced Vault Status with Standardized Format
# Provides both legacy and modern status reporting

# Source the standardized status helper
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
VAULT_LIB_DIR="${VAULT_LIB_DIR:-${APP_ROOT}/resources/vault/lib}"
RESOURCES_DIR="${VAULT_LIB_DIR}/../../.."
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common/status-helper.sh"

#######################################
# Get Vault status in standardized JSON format
#######################################
vault::get_status_json() {
    local service="vault"
    local port="${VAULT_PORT:-8200}"
    local health_endpoint="/v1/sys/health"
    local container_name="${VAULT_CONTAINER_NAME:-vault}"
    
    # Get basic service status from helper
    local basic_status
    basic_status=$(status_helper::check_service "$service" "$port" "$health_endpoint" "$container_name")
    
    # Add Vault-specific details
    local vault_details="{}"
    
    if vault::is_running; then
        # Get Vault-specific status information
        local initialized="false"
        local sealed="true"
        local vault_version=""
        local secret_engines="[]"
        local auth_methods="[]"
        local mode="${VAULT_MODE:-unknown}"
        
        # Check initialization status
        if vault::is_initialized; then
            initialized="true"
            
            # Check seal status
            if ! vault::is_sealed; then
                sealed="false"
                
                # Get additional details when unsealed
                local mounts_response
                mounts_response=$(vault::api_request "GET" "/v1/sys/mounts" 2>/dev/null || echo "{}")
                
                if [[ "$mounts_response" != "{}" ]]; then
                    secret_engines=$(echo "$mounts_response" | jq '[keys[] | select(. != "sys/" and . != "identity/" and . != "cubbyhole/")]' 2>/dev/null || echo "[]")
                fi
                
                local auth_response
                auth_response=$(vault::api_request "GET" "/v1/sys/auth" 2>/dev/null || echo "{}")
                
                if [[ "$auth_response" != "{}" ]]; then
                    auth_methods=$(echo "$auth_response" | jq '[keys[] | select(. != "token/")]' 2>/dev/null || echo "[]")
                fi
            fi
        fi
        
        # Get Vault version from health endpoint
        local health_response
        health_response=$(curl -s "http://localhost:$port/v1/sys/health" 2>/dev/null || echo "{}")
        vault_version=$(echo "$health_response" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown")
        
        # Build Vault-specific details
        vault_details=$(jq -n \
            --arg initialized "$initialized" \
            --arg sealed "$sealed" \
            --arg version "$vault_version" \
            --argjson secret_engines "$secret_engines" \
            --argjson auth_methods "$auth_methods" \
            --arg mode "$mode" \
            '{
                vault_initialized: ($initialized == "true"),
                vault_sealed: ($sealed == "true"),
                vault_version: $version,
                vault_mode: $mode,
                secret_engines: $secret_engines,
                auth_methods: $auth_methods
            }')
    fi
    
    # Merge basic status with Vault details
    echo "$basic_status" | jq --argjson vault_details "$vault_details" '.details += $vault_details'
}

#######################################
# Show status with enhanced format option
#######################################
vault::show_status_enhanced() {
    local format="${1:-human}"
    
    case "$format" in
        "json")
            vault::get_status_json
            ;;
        "human")
            # Use standardized display
            local json_status
            json_status=$(vault::get_status_json)
            status_helper::display_human "$json_status"
            
            # Add Vault-specific human-readable details
            local vault_initialized
            local vault_sealed
            local vault_version
            local vault_mode
            
            vault_initialized=$(echo "$json_status" | jq -r '.details.vault_initialized')
            vault_sealed=$(echo "$json_status" | jq -r '.details.vault_sealed')
            vault_version=$(echo "$json_status" | jq -r '.details.vault_version')
            vault_mode=$(echo "$json_status" | jq -r '.details.vault_mode')
            
            if [[ "$vault_initialized" == "true" ]]; then
                echo "   🔐 Initialized: Yes"
                if [[ "$vault_sealed" == "false" ]]; then
                    echo "   🔓 Sealed: No (ready for use)"
                else
                    echo "   🔒 Sealed: Yes (needs unsealing)"
                fi
            else
                echo "   ❌ Initialized: No (needs initialization)"
            fi
            
            if [[ "$vault_version" != "unknown" ]]; then
                echo "   📦 Version: $vault_version"
            fi
            
            if [[ "$vault_mode" != "unknown" ]]; then
                echo "   ⚙️  Mode: $vault_mode"
            fi
            ;;
        "legacy")
            # Fall back to original status display
            vault::show_status
            ;;
        *)
            echo "Unknown format: $format" >&2
            echo "Supported formats: json, human, legacy" >&2
            return 1
            ;;
    esac
}

#######################################
# Check integration readiness with another service
#######################################
vault::check_integration_with() {
    local other_service="$1"
    
    # Basic integration requirements for Vault
    local vault_status
    vault_status=$(vault::get_status_json)
    
    local vault_ready
    vault_ready=$(echo "$vault_status" | jq -r '.integration_ready')
    
    local vault_unsealed
    vault_unsealed=$(echo "$vault_status" | jq -r '.details.vault_sealed == false')
    
    # Vault is integration-ready if it's healthy and unsealed
    local integration_ready="false"
    local integration_message="Vault not ready for integration"
    
    if [[ "$vault_ready" == "true" && "$vault_unsealed" == "true" ]]; then
        # Test secret access
        local test_path="test/integration-check"
        local test_data='{"test": "integration"}'
        
        # Try to write and read a test secret
        local write_response
        write_response=$(vault::api_request "POST" "/v1/secret/data/$test_path" "$test_data" 2>/dev/null || echo "error")
        
        if [[ "$write_response" != "error" ]]; then
            # Try to read it back
            local read_response
            read_response=$(vault::api_request "GET" "/v1/secret/data/$test_path" 2>/dev/null || echo "error")
            
            if [[ "$read_response" != "error" ]]; then
                integration_ready="true"
                integration_message="Vault is ready for $other_service integration"
                
                # Clean up test secret
                vault::api_request "DELETE" "/v1/secret/data/$test_path" >/dev/null 2>&1 || true
            fi
        fi
    fi
    
    status_helper::check_integration "vault" "$other_service" | \
        jq --arg ready "$integration_ready" --arg msg "$integration_message" \
        '.integration_status = $ready | .message = $msg'
}

#######################################
# Comprehensive service health check
#######################################
vault::health_check_comprehensive() {
    local checks=()
    local overall_status="healthy"
    local issues=()
    
    # Container check
    if vault::is_running; then
        checks+=('{"check": "container", "status": "pass", "details": "Container is running"}')
    else
        checks+=('{"check": "container", "status": "fail", "details": "Container is not running"}')
        overall_status="error"
        issues+=("Container not running - start with: ./manage.sh --action start")
    fi
    
    # Network check
    if resources::is_service_running "${VAULT_PORT:-8200}"; then
        checks+=('{"check": "network", "status": "pass", "details": "Port is accessible"}')
    else
        checks+=('{"check": "network", "status": "fail", "details": "Port is not accessible"}')
        overall_status="error"
        issues+=("Port not accessible - check container and network configuration")
    fi
    
    # API check
    if status_helper::check_http_health "http://localhost:${VAULT_PORT:-8200}" "/v1/sys/health"; then
        checks+=('{"check": "api", "status": "pass", "details": "API is responding"}')
    else
        checks+=('{"check": "api", "status": "fail", "details": "API is not responding"}')
        if [[ "$overall_status" == "healthy" ]]; then
            overall_status="degraded"
        fi
        issues+=("API not responding - check Vault initialization and configuration")
    fi
    
    # Vault-specific checks
    if vault::is_initialized; then
        checks+=('{"check": "initialization", "status": "pass", "details": "Vault is initialized"}')
    else
        checks+=('{"check": "initialization", "status": "fail", "details": "Vault is not initialized"}')
        if [[ "$overall_status" == "healthy" ]]; then
            overall_status="degraded"
        fi
        issues+=("Vault not initialized - run: ./manage.sh --action init-dev")
    fi
    
    if ! vault::is_sealed; then
        checks+=('{"check": "seal_status", "status": "pass", "details": "Vault is unsealed"}')
    else
        checks+=('{"check": "seal_status", "status": "warn", "details": "Vault is sealed"}')
        if [[ "$overall_status" == "healthy" ]]; then
            overall_status="degraded"
        fi
        issues+=("Vault is sealed - unseal with: ./manage.sh --action unseal")
    fi
    
    # Build comprehensive response
    local checks_json
    checks_json=$(printf '%s\n' "${checks[@]}" | jq -s .)
    
    local issues_json
    issues_json=$(printf '%s\n' "${issues[@]}" | jq -R . | jq -s .)
    
    jq -n \
        --arg overall_status "$overall_status" \
        --argjson checks "$checks_json" \
        --argjson issues "$issues_json" \
        '{
            overall_status: $overall_status,
            checks: $checks,
            issues: $issues,
            timestamp: now | strftime("%Y-%m-%dT%H:%M:%S.%3NZ")
        }'
}

#######################################
# Status helper for integration tests
#######################################
vault::status_for_tests() {
    local json_status
    json_status=$(vault::get_status_json)
    
    local status
    local integration_ready
    local vault_unsealed
    
    status=$(echo "$json_status" | jq -r '.status')
    integration_ready=$(echo "$json_status" | jq -r '.integration_ready')
    vault_unsealed=$(echo "$json_status" | jq -r '.details.vault_sealed == false')
    
    # For tests, we need Vault to be healthy AND unsealed
    local test_ready="false"
    if [[ "$status" == "healthy" && "$integration_ready" == "true" && "$vault_unsealed" == "true" ]]; then
        test_ready="true"
    fi
    
    echo "$json_status" | jq --arg test_ready "$test_ready" '.test_ready = ($test_ready == "true")'
}