#!/usr/bin/env bash
# Vault Security Health Monitoring - Enhanced security operations monitoring
# Provides comprehensive security health checks and audit capabilities

# Source required libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VAULT_SECURITY_DIR="${APP_ROOT}/resources/vault/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/vault/config/defaults.sh"
# shellcheck disable=SC1091
source "${VAULT_SECURITY_DIR}/common.sh"
# shellcheck disable=SC1091
source "${VAULT_SECURITY_DIR}/api.sh"

#######################################
# Perform comprehensive security health check
# Returns: 0 if all security checks pass, 1 otherwise
#######################################
vault::security::health_check() {
    local verbose="${1:-false}"
    local overall_status=0
    local check_results=()
    local security_score=100
    
    log::header "Vault Security Health Check"
    
    # 1. Check encryption status
    log::info "Checking encryption status..."
    if vault::security::check_encryption; then
        check_results+=("✅ Encryption: Active")
    else
        check_results+=("❌ Encryption: Failed")
        overall_status=1
        security_score=$((security_score - 20))
    fi
    
    # 2. Check seal status
    log::info "Checking seal status..."
    local seal_status
    seal_status=$(vault::security::check_seal_status)
    if [[ "$seal_status" == "unsealed" ]]; then
        check_results+=("✅ Seal Status: Unsealed (operational)")
    elif [[ "$seal_status" == "sealed" ]]; then
        check_results+=("⚠️  Seal Status: Sealed (requires unseal)")
        overall_status=1
    else
        check_results+=("❌ Seal Status: Unknown")
        overall_status=1
    fi
    
    # 3. Check audit logging
    log::info "Checking audit logging..."
    if vault::security::check_audit_enabled; then
        check_results+=("✅ Audit Logging: Enabled")
    else
        check_results+=("⚠️  Audit Logging: Disabled")
        if [[ "$VAULT_MODE" == "prod" ]]; then
            overall_status=1
        fi
    fi
    
    # 4. Check access control policies
    log::info "Checking access control policies..."
    local policy_count
    policy_count=$(vault::security::count_policies)
    if [[ $policy_count -gt 0 ]]; then
        check_results+=("✅ Access Policies: $policy_count configured")
    else
        check_results+=("⚠️  Access Policies: None configured")
    fi
    
    # 5. Check token expiration
    log::info "Checking token expiration..."
    if vault::security::check_token_expiry; then
        check_results+=("✅ Token Expiry: Valid")
    else
        check_results+=("⚠️  Token Expiry: Expired or invalid")
        overall_status=1
    fi
    
    # 6. Check TLS configuration
    log::info "Checking TLS configuration..."
    if [[ "$VAULT_TLS_DISABLE" == "1" ]]; then
        if [[ "$VAULT_MODE" == "dev" ]]; then
            check_results+=("⚠️  TLS: Disabled (dev mode)")
        else
            check_results+=("❌ TLS: Disabled (security risk)")
            overall_status=1
        fi
    else
        check_results+=("✅ TLS: Enabled")
    fi
    
    # 7. Check secret rotation metrics
    log::info "Checking secret rotation metrics..."
    local rotation_status
    rotation_status=$(vault::security::check_rotation_status)
    check_results+=("ℹ️  Secret Rotation: $rotation_status")
    
    # 8. Check for recent security events
    log::info "Checking recent security events..."
    local recent_errors
    recent_errors=$(docker logs --tail 100 "$VAULT_CONTAINER_NAME" 2>&1 | grep -c "error\|unauthorized\|forbidden" || echo "0")
    if [[ $recent_errors -eq 0 ]]; then
        check_results+=("✅ Security Events: No recent errors")
    else
        check_results+=("⚠️  Security Events: $recent_errors errors in recent logs")
        security_score=$((security_score - 5))
    fi
    
    # Calculate security grade
    local security_grade="A+"
    if [[ $security_score -lt 95 ]]; then security_grade="A"; fi
    if [[ $security_score -lt 90 ]]; then security_grade="B+"; fi
    if [[ $security_score -lt 85 ]]; then security_grade="B"; fi
    if [[ $security_score -lt 80 ]]; then security_grade="C"; fi
    if [[ $security_score -lt 70 ]]; then security_grade="D"; fi
    if [[ $security_score -lt 60 ]]; then security_grade="F"; fi
    
    # Display results
    echo ""
    log::header "Security Health Summary"
    for result in "${check_results[@]}"; do
        echo "  $result"
    done
    
    # Display security score
    echo ""
    echo "  🎯 Security Score: ${security_score}/100 (Grade: $security_grade)"
    
    # Overall status
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🔒 Security health check PASSED"
    else
        log::error "⚠️  Security health check has warnings/failures"
    fi
    
    # Provide recommendations if needed
    if [[ $security_score -lt 100 ]]; then
        echo ""
        log::header "Security Recommendations"
        vault::security::provide_recommendations "$security_score"
    fi
    
    return $overall_status
}

#######################################
# Check encryption status
# Returns: 0 if encryption is active, 1 otherwise
#######################################
vault::security::check_encryption() {
    local response
    response=$(curl -sf "${VAULT_BASE_URL}/v1/sys/health" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        return 1
    fi
    
    # Check if response indicates encryption is active
    echo "$response" | jq -e '.initialized == true' >/dev/null 2>&1
}

#######################################
# Check seal status
# Returns: unsealed, sealed, or unknown
#######################################
vault::security::check_seal_status() {
    local response
    response=$(curl -sf "${VAULT_BASE_URL}/v1/sys/seal-status" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        echo "unknown"
        return 1
    fi
    
    local sealed
    sealed=$(echo "$response" | jq -r '.sealed' 2>/dev/null)
    
    if [[ "$sealed" == "false" ]]; then
        echo "unsealed"
        return 0
    elif [[ "$sealed" == "true" ]]; then
        echo "sealed"
        return 1
    else
        echo "unknown"
        return 1
    fi
}

#######################################
# Check if audit logging is enabled
# Returns: 0 if enabled, 1 otherwise
#######################################
vault::security::check_audit_enabled() {
    # In dev mode, audit is typically not enabled
    if [[ "$VAULT_MODE" == "dev" ]]; then
        return 1
    fi
    
    local response
    response=$(vault::api_request "GET" "/v1/sys/audit" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        return 1
    fi
    
    # Check if any audit devices are configured
    local audit_count
    audit_count=$(echo "$response" | jq 'length' 2>/dev/null || echo "0")
    
    [[ $audit_count -gt 0 ]]
}

#######################################
# Count configured policies
# Returns: number of policies
#######################################
vault::security::count_policies() {
    local response
    response=$(vault::api_request "LIST" "/v1/sys/policies/acl" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        echo "0"
        return 1
    fi
    
    local policies
    policies=$(echo "$response" | jq -r '.data.keys[]' 2>/dev/null | wc -l)
    echo "${policies:-0}"
}

#######################################
# Check token expiration
# Returns: 0 if token is valid, 1 otherwise
#######################################
vault::security::check_token_expiry() {
    local response
    response=$(vault::api_request "GET" "/v1/auth/token/lookup-self" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        return 1
    fi
    
    # Check if token has expiration
    local ttl
    ttl=$(echo "$response" | jq -r '.data.ttl' 2>/dev/null)
    
    if [[ -z "$ttl" ]] || [[ "$ttl" == "null" ]]; then
        # No expiration (root token in dev mode)
        return 0
    fi
    
    # Check if TTL is positive (not expired)
    [[ $ttl -gt 0 ]]
}

#######################################
# Check secret rotation status
# Returns: rotation status message
#######################################
vault::security::check_rotation_status() {
    # This would integrate with your rotation policies
    # For now, return a status based on mode
    if [[ "$VAULT_MODE" == "dev" ]]; then
        echo "Not configured (dev mode)"
    else
        echo "Manual rotation available"
    fi
}

#######################################
# Monitor security events in real-time
# Arguments:
#   $1 - duration in seconds (optional, default: continuous)
#######################################
vault::security::monitor() {
    local duration="${1:-0}"
    local start_time=$(date +%s)
    
    log::header "Vault Security Monitoring"
    log::info "Monitoring security events... (Press Ctrl+C to stop)"
    
    while true; do
        clear
        echo "═══════════════════════════════════════════════════════════"
        echo "          VAULT SECURITY MONITOR - $(date '+%Y-%m-%d %H:%M:%S')"
        echo "═══════════════════════════════════════════════════════════"
        echo ""
        
        # Display current security status
        local seal_status
        seal_status=$(vault::security::check_seal_status)
        echo "🔒 Seal Status: $seal_status"
        
        # Display token status
        if vault::security::check_token_expiry; then
            echo "🔑 Token Status: Valid"
        else
            echo "🔑 Token Status: Expired/Invalid"
        fi
        
        # Display audit status
        if vault::security::check_audit_enabled; then
            echo "📝 Audit Logging: Enabled"
        else
            echo "📝 Audit Logging: Disabled"
        fi
        
        # Display recent activity (if available)
        echo ""
        echo "Recent Activity:"
        echo "----------------"
        # This would show recent operations in production
        docker logs --tail 5 "$VAULT_CONTAINER_NAME" 2>/dev/null | grep -E "auth|secret|policy" || echo "No recent security events"
        
        # Check duration
        if [[ $duration -gt 0 ]]; then
            local current_time=$(date +%s)
            local elapsed=$((current_time - start_time))
            if [[ $elapsed -ge $duration ]]; then
                break
            fi
        fi
        
        sleep 5
    done
}

#######################################
# Generate security audit report
# Arguments:
#   $1 - output format (text/json)
#######################################
vault::security::audit_report() {
    local format="${1:-text}"
    local report_time=$(date -Iseconds)
    
    if [[ "$format" == "json" ]]; then
        # Generate JSON report
        local report
        report=$(jq -n \
            --arg time "$report_time" \
            --arg mode "$VAULT_MODE" \
            --arg seal "$(vault::security::check_seal_status)" \
            --arg encryption "$(vault::security::check_encryption && echo 'active' || echo 'inactive')" \
            --arg audit "$(vault::security::check_audit_enabled && echo 'enabled' || echo 'disabled')" \
            --arg policies "$(vault::security::count_policies)" \
            --arg tls "$([[ "$VAULT_TLS_DISABLE" == "1" ]] && echo 'disabled' || echo 'enabled')" \
            '{
                timestamp: $time,
                mode: $mode,
                security: {
                    seal_status: $seal,
                    encryption: $encryption,
                    audit_logging: $audit,
                    policy_count: $policies,
                    tls: $tls
                }
            }')
        echo "$report"
    else
        # Generate text report
        log::header "Vault Security Audit Report"
        echo "Generated: $report_time"
        echo "Mode: $VAULT_MODE"
        echo ""
        echo "Security Configuration:"
        echo "----------------------"
        echo "• Seal Status: $(vault::security::check_seal_status)"
        echo "• Encryption: $(vault::security::check_encryption && echo 'Active' || echo 'Inactive')"
        echo "• Audit Logging: $(vault::security::check_audit_enabled && echo 'Enabled' || echo 'Disabled')"
        echo "• Access Policies: $(vault::security::count_policies) configured"
        echo "• TLS: $([[ "$VAULT_TLS_DISABLE" == "1" ]] && echo 'Disabled' || echo 'Enabled')"
        echo ""
        echo "Recommendations:"
        echo "---------------"
        if [[ "$VAULT_MODE" == "dev" ]]; then
            echo "⚠️  Development mode detected - not suitable for production"
            echo "• Enable TLS for production use"
            echo "• Configure audit logging"
            echo "• Implement access policies"
        fi
        if ! vault::security::check_audit_enabled; then
            echo "• Enable audit logging for compliance"
        fi
        if [[ $(vault::security::count_policies) -eq 0 ]]; then
            echo "• Configure access control policies"
        fi
    fi
}

#######################################
# Provide security recommendations based on score
# Arguments:
#   $1 - security score
#######################################
vault::security::provide_recommendations() {
    local score="${1:-100}"
    
    # Check for specific issues and provide targeted recommendations
    if ! vault::security::check_audit_enabled; then
        echo "  • Enable audit logging: vault audit enable"
    fi
    
    if [[ $(vault::security::count_policies) -eq 0 ]]; then
        echo "  • Configure access policies: vault access setup-standard"
    fi
    
    if [[ "$VAULT_TLS_DISABLE" == "1" ]] && [[ "$VAULT_MODE" != "dev" ]]; then
        echo "  • Enable TLS for production use"
    fi
    
    if ! vault::security::check_token_expiry; then
        echo "  • Renew or rotate authentication tokens"
    fi
    
    local seal_status
    seal_status=$(vault::security::check_seal_status)
    if [[ "$seal_status" == "sealed" ]]; then
        echo "  • Unseal Vault: vault unseal"
    fi
    
    # General recommendations based on score
    if [[ $score -lt 80 ]]; then
        echo "  • Review and harden security configuration"
        echo "  • Implement regular secret rotation"
        echo "  • Enable comprehensive monitoring"
    fi
}

# Export functions for use by other scripts
export -f vault::security::health_check
export -f vault::security::monitor
export -f vault::security::audit_report
export -f vault::security::provide_recommendations