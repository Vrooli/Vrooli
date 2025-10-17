#!/usr/bin/env bash
# Judge0 Security Module
# Handles security configuration and validation

#######################################
# Show current security configuration
#######################################
judge0::security::show_config() {
    log::info "$JUDGE0_MSG_SEC_LIMITS"
    echo
    
    printf "  $JUDGE0_MSG_SEC_CPU\n" "$JUDGE0_CPU_TIME_LIMIT"
    printf "  $JUDGE0_MSG_SEC_MEMORY\n" "$((JUDGE0_MEMORY_LIMIT / 1024))"
    echo "  $JUDGE0_MSG_SEC_NETWORK"
    echo "  $JUDGE0_MSG_SEC_SANDBOX"
    echo "  ⏰ Wall Time: ${JUDGE0_WALL_TIME_LIMIT}s"
    echo "  📁 Max File Size: $((JUDGE0_MAX_FILE_SIZE / 1024))MB"
    echo "  🔢 Max Processes: $JUDGE0_MAX_PROCESSES"
    echo "  📚 Stack Limit: $((JUDGE0_STACK_LIMIT / 1024))MB"
    echo
    
    if [[ "$JUDGE0_ENABLE_AUTHENTICATION" == "true" ]]; then
        log::success "  🔐 API Authentication: Enabled"
    else
        log::warning "  🔓 API Authentication: Disabled"
    fi
    echo
}

#######################################
# Validate security settings
# Returns:
#   0 if secure, 1 if issues found
#######################################
judge0::security::validate() {
    local issues=0
    
    log::info "🔍 Validating security configuration..."
    echo
    
    # Check API authentication
    if [[ "$JUDGE0_ENABLE_AUTHENTICATION" != "true" ]]; then
        log::warning "⚠️  API authentication is disabled"
        ((issues++))
    else
        log::success "✅ API authentication enabled"
    fi
    
    # Check network isolation
    if [[ "$JUDGE0_ENABLE_NETWORK" == "true" ]]; then
        log::warning "⚠️  Network access is enabled in execution containers"
        ((issues++))
    else
        log::success "✅ Network isolated in execution containers"
    fi
    
    # Check resource limits
    if [[ $JUDGE0_CPU_TIME_LIMIT -gt 30 ]]; then
        log::warning "⚠️  CPU time limit is high (${JUDGE0_CPU_TIME_LIMIT}s)"
        ((issues++))
    else
        log::success "✅ CPU time limit reasonable (${JUDGE0_CPU_TIME_LIMIT}s)"
    fi
    
    if [[ $((JUDGE0_MEMORY_LIMIT / 1024)) -gt 512 ]]; then
        log::warning "⚠️  Memory limit is high ($((JUDGE0_MEMORY_LIMIT / 1024))MB)"
        ((issues++))
    else
        log::success "✅ Memory limit reasonable ($((JUDGE0_MEMORY_LIMIT / 1024))MB)"
    fi
    
    # Check API key strength
    local api_key=$(judge0::get_api_key)
    if [[ -n "$api_key" ]]; then
        if [[ ${#api_key} -lt 32 ]]; then
            log::warning "⚠️  API key is shorter than recommended (32 chars)"
            ((issues++))
        else
            log::success "✅ API key length sufficient"
        fi
    fi
    
    # Check file permissions
    if [[ -f "${JUDGE0_CONFIG_DIR}/api_key" ]]; then
        local perms=$(stat -c %a "${JUDGE0_CONFIG_DIR}/api_key" 2>/dev/null || stat -f %A "${JUDGE0_CONFIG_DIR}/api_key" 2>/dev/null)
        if [[ "$perms" != "600" ]]; then
            log::warning "⚠️  API key file has loose permissions ($perms)"
            ((issues++))
        else
            log::success "✅ API key file permissions secure"
        fi
    fi
    
    # Check Docker socket access
    if docker::is_socket_exposed "$JUDGE0_CONTAINER_NAME"; then
        log::warning "⚠️  Docker socket is exposed to Judge0 container"
        ((issues++))
    else
        log::success "✅ Docker socket not exposed"
    fi
    
    echo
    if [[ $issues -eq 0 ]]; then
        log::success "🛡️  All security checks passed!"
        return 0
    else
        log::warning "🔍 Found $issues security concern(s)"
        echo
        echo "Run '$0 --action security-harden' to apply recommended settings"
        return 1
    fi
}

#######################################
# Apply security hardening
#######################################
judge0::security::harden() {
    log::info "🛡️  Applying security hardening..."
    echo
    
    # Enable API authentication
    if [[ "$JUDGE0_ENABLE_AUTHENTICATION" != "true" ]]; then
        log::info "Enabling API authentication..."
        export JUDGE0_ENABLE_AUTHENTICATION="true"
        
        # Generate API key if missing
        if [[ -z "$(judge0::get_api_key)" ]]; then
            local new_key=$(judge0::generate_api_key)
            judge0::save_api_key "$new_key"
            export JUDGE0_API_KEY="$new_key"
        fi
    fi
    
    # Disable network in containers
    if [[ "$JUDGE0_ENABLE_NETWORK" == "true" ]]; then
        log::info "Disabling network access in execution containers..."
        export JUDGE0_ENABLE_NETWORK="false"
    fi
    
    # Apply conservative resource limits
    log::info "Applying conservative resource limits..."
    export JUDGE0_CPU_TIME_LIMIT="5"
    export JUDGE0_WALL_TIME_LIMIT="10"
    export JUDGE0_MEMORY_LIMIT="262144"  # 256MB
    export JUDGE0_MAX_FILE_SIZE="5120"    # 5MB
    
    # Fix file permissions
    if [[ -f "${JUDGE0_CONFIG_DIR}/api_key" ]]; then
        chmod 600 "${JUDGE0_CONFIG_DIR}/api_key"
    fi
    
    # Restart Judge0 with new settings
    if judge0::is_running; then
        log::info "Restarting Judge0 with hardened configuration..."
        judge0::install::create_compose_file
        judge0::docker::restart
    else
        log::info "Judge0 is not running. Settings will be applied on next start."
    fi
    
    echo
    log::success "✅ Security hardening applied!"
    echo
    judge0::security::show_config
}

#######################################
# Test security boundaries
#######################################
judge0::security::test_boundaries() {
    log::info "🧪 Testing security boundaries..."
    echo
    
    if ! judge0::is_running; then
        log::error "Judge0 is not running"
        return 1
    fi
    
    # Test 1: Fork bomb protection
    log::info "Test 1: Fork bomb protection"
    local fork_bomb='while true; do echo "test" & done'
    local result=$(judge0::api::submit "$fork_bomb" "bash" 2>&1 || echo "")
    
    if echo "$result" | grep -q "Time Limit Exceeded\|Runtime Error"; then
        log::success "✅ Fork bomb prevented"
    else
        log::warning "⚠️  Fork bomb protection may be insufficient"
    fi
    
    # Test 2: Memory exhaustion
    log::info "Test 2: Memory exhaustion protection"
    local memory_bomb='a = [0] * (10**9)'  # Try to allocate 1GB
    result=$(judge0::api::submit "$memory_bomb" "python" 2>&1 || echo "")
    
    if echo "$result" | grep -q "Memory Limit Exceeded\|Runtime Error"; then
        log::success "✅ Memory exhaustion prevented"
    else
        log::warning "⚠️  Memory limit may be insufficient"
    fi
    
    # Test 3: Network access
    if [[ "$JUDGE0_ENABLE_NETWORK" != "true" ]]; then
        log::info "Test 3: Network isolation"
        local network_test='import urllib.request; urllib.request.urlopen("http://google.com")'
        result=$(judge0::api::submit "$network_test" "python" 2>&1 || echo "")
        
        if echo "$result" | grep -q "Error\|Failed"; then
            log::success "✅ Network access blocked"
        else
            log::warning "⚠️  Network isolation may be compromised"
        fi
    fi
    
    # Test 4: File system access
    log::info "Test 4: File system isolation"
    local fs_test='import os; print(os.listdir("/etc"))'
    result=$(judge0::api::submit "$fs_test" "python" 2>&1 || echo "")
    
    # Should succeed but with limited view
    if echo "$result" | grep -q "passwd\|shadow"; then
        log::warning "⚠️  Sensitive files may be accessible"
    else
        log::success "✅ File system appears isolated"
    fi
    
    echo
    log::info "Security boundary testing complete"
}

#######################################
# Generate security report
#######################################
judge0::security::report() {
    local report_file="${JUDGE0_LOGS_DIR}/security-report-$(date +%Y%m%d-%H%M%S).json"
    
    log::info "📊 Generating security report..."
    
    # Collect security information
    local config=$(judge0::security::get_config_json)
    local validation=$(judge0::security::validate >/dev/null 2>&1 && echo "passed" || echo "failed")
    local api_key_exists=$(test -f "${JUDGE0_CONFIG_DIR}/api_key" && echo "true" || echo "false")
    
    # Create report
    cat > "$report_file" <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "version": "$JUDGE0_VERSION",
  "security_validation": "$validation",
  "configuration": $config,
  "api_authentication": {
    "enabled": $JUDGE0_ENABLE_AUTHENTICATION,
    "key_exists": $api_key_exists,
    "key_length": $(test -f "${JUDGE0_CONFIG_DIR}/api_key" && wc -c < "${JUDGE0_CONFIG_DIR}/api_key" || echo 0)
  },
  "resource_limits": {
    "cpu_time": $JUDGE0_CPU_TIME_LIMIT,
    "wall_time": $JUDGE0_WALL_TIME_LIMIT,
    "memory_mb": $((JUDGE0_MEMORY_LIMIT / 1024)),
    "max_processes": $JUDGE0_MAX_PROCESSES,
    "max_file_size_mb": $((JUDGE0_MAX_FILE_SIZE / 1024)),
    "stack_limit_mb": $((JUDGE0_STACK_LIMIT / 1024))
  },
  "isolation": {
    "network_disabled": $(test "$JUDGE0_ENABLE_NETWORK" == "false" && echo "true" || echo "false"),
    "callbacks_disabled": $(test "$JUDGE0_ENABLE_CALLBACKS" == "false" && echo "true" || echo "false")
  }
}
EOF
    
    log::success "Security report saved to: $report_file"
    echo
    echo "Key findings:"
    cat "$report_file" | jq -r '
        "  Security Validation: \(.security_validation)",
        "  API Authentication: \(if .api_authentication.enabled then "Enabled" else "Disabled" end)",
        "  Network Isolation: \(if .isolation.network_disabled then "Enabled" else "Disabled" end)",
        "  Resource Limits: CPU \(.resource_limits.cpu_time)s, Memory \(.resource_limits.memory_mb)MB"
    '
}

#######################################
# Get security configuration as JSON
#######################################
judge0::security::get_config_json() {
    cat <<EOF
{
  "cpu_time_limit": $JUDGE0_CPU_TIME_LIMIT,
  "wall_time_limit": $JUDGE0_WALL_TIME_LIMIT,
  "memory_limit": $JUDGE0_MEMORY_LIMIT,
  "max_processes": $JUDGE0_MAX_PROCESSES,
  "max_file_size": $JUDGE0_MAX_FILE_SIZE,
  "stack_limit": $JUDGE0_STACK_LIMIT,
  "enable_network": $JUDGE0_ENABLE_NETWORK,
  "enable_callbacks": $JUDGE0_ENABLE_CALLBACKS,
  "enable_authentication": $JUDGE0_ENABLE_AUTHENTICATION
}
EOF
}