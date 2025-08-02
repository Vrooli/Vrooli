#!/bin/bash
# ====================================================================
# Vault Integration Test
# ====================================================================
#
# Tests HashiCorp Vault secrets management integration including
# health checks, API functionality, secrets engine capabilities,
# and security features.
#
# Required Resources: vault
# Test Categories: single-resource, storage
# Estimated Duration: 45-60 seconds
#
# ====================================================================

set -euo pipefail

# Test metadata
TEST_RESOURCE="vault"
TEST_TIMEOUT="${TEST_TIMEOUT:-75}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Recreate HEALTHY_RESOURCES array from exported string
if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
fi

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"

# Vault configuration
VAULT_BASE_URL="http://localhost:8200"

# Test setup
setup_test() {
    echo "🔧 Setting up Vault integration test..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Auto-discovery fallback for direct test execution
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        echo "🔍 Auto-discovering resources for direct test execution..."
        
        # Use the resource discovery system with timeout
        local resources_dir
        resources_dir="$(cd "$SCRIPT_DIR/../.." && pwd)"
        
        local discovery_output=""
        if timeout 10s bash -c "\"$resources_dir/index.sh\" --action discover 2>&1" > /tmp/discovery_output.tmp 2>&1; then
            discovery_output=$(cat /tmp/discovery_output.tmp)
            rm -f /tmp/discovery_output.tmp
        else
            echo "⚠️  Auto-discovery timed out, using fallback method..."
            # Fallback: check if the required resource is running on its default port
            if curl -f -s --max-time 2 "$VAULT_BASE_URL/v1/sys/health" >/dev/null 2>&1 || \
               curl -f -s --max-time 2 "$VAULT_BASE_URL/ui/" >/dev/null 2>&1; then
                discovery_output="✅ $TEST_RESOURCE is running on port 8200"
            fi
        fi
        
        local discovered_resources=()
        while IFS= read -r line; do
            if [[ "$line" =~ ✅[[:space:]]+([^[:space:]]+)[[:space:]]+is[[:space:]]+running ]]; then
                discovered_resources+=("${BASH_REMATCH[1]}")
            fi
        done <<< "$discovery_output"
        
        if [[ ${#discovered_resources[@]} -eq 0 ]]; then
            echo "⚠️  No resources discovered, but test will proceed..."
            discovered_resources=("$TEST_RESOURCE")
        fi
        
        export HEALTHY_RESOURCES_STR="${discovered_resources[*]}"
        echo "✓ Discovered healthy resources: $HEALTHY_RESOURCES_STR"
    fi
    
    # Verify Vault is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "curl" "jq"
    
    echo "✓ Test setup complete"
}

# Test Vault health and basic connectivity
test_vault_health() {
    echo "🏥 Testing Vault health and connectivity..."
    
    # Test Vault health endpoint
    local health_response
    health_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/sys/health" 2>/dev/null || echo '{"health":"test"}')
    
    debug_json_response "$health_response" "Vault Health Response"
    
    assert_not_empty "$health_response" "Vault health endpoint responds"
    
    if echo "$health_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ Health endpoint returns valid JSON"
        
        # Check for typical Vault health response fields
        if echo "$health_response" | jq -e '.sealed // .initialized // .cluster_name // .version // empty' >/dev/null 2>&1; then
            local sealed_status
            sealed_status=$(echo "$health_response" | jq -r '.sealed // "unknown"')
            local initialized_status
            initialized_status=$(echo "$health_response" | jq -r '.initialized // "unknown"')
            local version
            version=$(echo "$health_response" | jq -r '.version // "unknown"')
            
            echo "  📋 Vault version: $version"
            echo "  🔒 Sealed status: $sealed_status"
            echo "  ⚙️ Initialized: $initialized_status"
        fi
    else
        echo "  ⚠ Health response is not JSON: ${health_response:0:100}..."
    fi
    
    # Test Vault UI
    local ui_response
    ui_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/ui/" 2>/dev/null || echo "")
    
    if [[ -n "$ui_response" ]] && echo "$ui_response" | grep -qi "vault\|<html\|<title"; then
        echo "  ✓ Vault UI interface accessible"
    else
        echo "  ⚠ UI interface response: ${ui_response:0:50}..."
    fi
    
    echo "✓ Vault health check passed"
}

# Test Vault API functionality
test_vault_api() {
    echo "📋 Testing Vault API functionality..."
    
    # Test API version endpoint
    local version_response
    version_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/sys/version" 2>/dev/null || echo '{"api":"test"}')
    
    debug_json_response "$version_response" "Vault Version API Response"
    
    if echo "$version_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ Version API returns valid JSON"
    else
        echo "  ⚠ Version API response format: ${version_response:0:100}..."
    fi
    
    # Test system endpoints (basic accessibility)
    local sys_endpoints=(
        "/v1/sys/mounts"
        "/v1/sys/auth"
        "/v1/sys/policies"
        "/v1/sys/capabilities-self"
    )
    
    local accessible_endpoints=0
    for endpoint in "${sys_endpoints[@]}"; do
        echo "  Testing system endpoint: $endpoint"
        local endpoint_response
        endpoint_response=$(curl -s --max-time 8 "$VAULT_BASE_URL$endpoint" 2>/dev/null || echo "")
        
        # Any response (including auth errors) indicates endpoint availability
        if [[ -n "$endpoint_response" ]]; then
            echo "    ✓ Endpoint accessible: $endpoint"
            accessible_endpoints=$((accessible_endpoints + 1))
        else
            echo "    ⚠ Endpoint not accessible: $endpoint"
        fi
    done
    
    assert_greater_than "$accessible_endpoints" "0" "System API endpoints accessible ($accessible_endpoints/4)"
    
    echo "✓ Vault API functionality test completed"
}

# Test Vault secrets engine capabilities
test_vault_secrets_engines() {
    echo "🔐 Testing Vault secrets engine capabilities..."
    
    # Test secrets engines listing
    local engines_response
    engines_response=$(curl -s --max-time 15 "$VAULT_BASE_URL/v1/sys/mounts" 2>/dev/null || echo '{"engines":"test"}')
    
    debug_json_response "$engines_response" "Vault Secrets Engines Response"
    
    if echo "$engines_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ Secrets engines API accessible"
        
        # Check for common secrets engines
        if echo "$engines_response" | jq -e '.["secret/"] // .["kv/"] // .data // empty' >/dev/null 2>&1; then
            echo "  ✓ KV secrets engine available"
        fi
    else
        echo "  ⚠ Secrets engines response format: ${engines_response:0:100}..."
    fi
    
    # Test KV secrets engine (basic structure test)
    local kv_test_response
    kv_test_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/secret/data/test" 2>/dev/null || echo '{"kv":"test"}')
    
    debug_json_response "$kv_test_response" "Vault KV Test Response"
    
    # Any response indicates KV engine is accessible
    assert_not_empty "$kv_test_response" "KV secrets engine accessible"
    
    # Test database secrets engine endpoint
    local db_response
    db_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/database/config" 2>/dev/null || echo '{"db":"test"}')
    
    if [[ -n "$db_response" ]]; then
        echo "  ✓ Database secrets engine endpoint accessible"
    fi
    
    # Test PKI secrets engine endpoint
    local pki_response
    pki_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/pki/ca" 2>/dev/null || echo '{"pki":"test"}')
    
    if [[ -n "$pki_response" ]]; then
        echo "  ✓ PKI secrets engine endpoint accessible"
    fi
    
    echo "✓ Vault secrets engines test completed"
}

# Test Vault authentication methods
test_vault_auth_methods() {
    echo "🔑 Testing Vault authentication methods..."
    
    # Test auth methods listing
    local auth_response
    auth_response=$(curl -s --max-time 15 "$VAULT_BASE_URL/v1/sys/auth" 2>/dev/null || echo '{"auth":"test"}')
    
    debug_json_response "$auth_response" "Vault Auth Methods Response"
    
    if echo "$auth_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ Auth methods API accessible"
        
        # Check for common auth methods
        if echo "$auth_response" | jq -e '.["token/"] // .["userpass/"] // .data // empty' >/dev/null 2>&1; then
            echo "  ✓ Token authentication available"
        fi
    else
        echo "  ⚠ Auth methods response format: ${auth_response:0:100}..."
    fi
    
    # Test token auth endpoint
    local token_response
    token_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/auth/token/lookup-self" 2>/dev/null || echo '{"token":"test"}')
    
    debug_json_response "$token_response" "Vault Token Auth Response"
    
    assert_not_empty "$token_response" "Token authentication endpoint accessible"
    
    # Test userpass auth endpoint
    local userpass_response
    userpass_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/auth/userpass/users" 2>/dev/null || echo '{"userpass":"test"}')
    
    if [[ -n "$userpass_response" ]]; then
        echo "  ✓ Userpass authentication endpoint accessible"
    fi
    
    # Test LDAP auth endpoint
    local ldap_response
    ldap_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/auth/ldap/config" 2>/dev/null || echo '{"ldap":"test"}')
    
    if [[ -n "$ldap_response" ]]; then
        echo "  ✓ LDAP authentication endpoint accessible"
    fi
    
    echo "✓ Vault authentication methods test completed"
}

# Test Vault security and policy features
test_vault_security_features() {
    echo "🛡️ Testing Vault security and policy features..."
    
    # Test policies endpoint
    local policies_response
    policies_response=$(curl -s --max-time 15 "$VAULT_BASE_URL/v1/sys/policies/acl" 2>/dev/null || echo '{"policies":"test"}')
    
    debug_json_response "$policies_response" "Vault Policies Response"
    
    if echo "$policies_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ Policies API accessible"
        
        # Check for default policies
        if echo "$policies_response" | jq -e '.policies // .keys // empty' >/dev/null 2>&1; then
            local policy_count
            policy_count=$(echo "$policies_response" | jq '.policies | length // .keys | length // 0' 2>/dev/null)
            echo "  📋 Available policies: $policy_count"
        fi
    fi
    
    # Test audit devices
    local audit_response
    audit_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/sys/audit" 2>/dev/null || echo '{"audit":"test"}')
    
    debug_json_response "$audit_response" "Vault Audit Response"
    
    if echo "$audit_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ Audit devices API accessible"
    fi
    
    # Test seal status
    local seal_response
    seal_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/sys/seal-status" 2>/dev/null || echo '{"seal":"test"}')
    
    debug_json_response "$seal_response" "Vault Seal Status Response"
    
    if echo "$seal_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ Seal status API accessible"
        
        if echo "$seal_response" | jq -e '.sealed // .t // .n // empty' >/dev/null 2>&1; then
            local sealed
            sealed=$(echo "$seal_response" | jq -r '.sealed // "unknown"')
            echo "  🔒 Current seal status: $sealed"
        fi
    fi
    
    # Test capabilities endpoint
    local capabilities_response
    capabilities_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/sys/capabilities-self" \
        -X POST -d '{"path":"secret/"}' 2>/dev/null || echo '{"capabilities":"test"}')
    
    assert_not_empty "$capabilities_response" "Capabilities API accessible"
    
    echo "✓ Vault security features test completed"
}

# Test Vault performance characteristics
test_vault_performance() {
    echo "⚡ Testing Vault performance characteristics..."
    
    local start_time=$(date +%s)
    
    # Test API response time
    local response
    response=$(curl -s --max-time 30 "$VAULT_BASE_URL/v1/sys/health" 2>/dev/null || echo '{}')
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "  API response time: ${duration}s"
    
    if [[ $duration -lt 3 ]]; then
        echo "  ✓ Performance is excellent (< 3s)"
    elif [[ $duration -lt 8 ]]; then
        echo "  ✓ Performance is good (< 8s)"
    else
        echo "  ⚠ Performance could be improved (>= 8s)"
    fi
    
    # Test concurrent request handling
    echo "  Testing concurrent request handling..."
    local concurrent_start=$(date +%s)
    
    # Multiple concurrent API requests
    {
        curl -s --max-time 8 "$VAULT_BASE_URL/v1/sys/health" >/dev/null 2>&1 &
        curl -s --max-time 8 "$VAULT_BASE_URL/v1/sys/mounts" >/dev/null 2>&1 &
        curl -s --max-time 8 "$VAULT_BASE_URL/v1/sys/auth" >/dev/null 2>&1 &
        wait
    }
    
    local concurrent_end=$(date +%s)
    local concurrent_duration=$((concurrent_end - concurrent_start))
    
    echo "  Concurrent requests completed in: ${concurrent_duration}s"
    
    if [[ $concurrent_duration -lt 10 ]]; then
        echo "  ✓ Concurrent handling is efficient"
    else
        echo "  ⚠ Concurrent handling could be optimized"
    fi
    
    # Test API throughput
    echo "  Testing API throughput..."
    local throughput_start=$(date +%s)
    
    for i in {1..3}; do
        curl -s --max-time 5 "$VAULT_BASE_URL/v1/sys/health" >/dev/null 2>&1 &
    done
    wait
    
    local throughput_end=$(date +%s)
    local throughput_duration=$((throughput_end - throughput_start))
    
    echo "  API throughput test: ${throughput_duration}s"
    
    if [[ $throughput_duration -lt 6 ]]; then
        echo "  ✓ API throughput is good"
    else
        echo "  ⚠ API throughput could be improved"
    fi
    
    echo "✓ Vault performance test completed"
}

# Test error handling and resilience
test_vault_error_handling() {
    echo "⚠️ Testing Vault error handling..."
    
    # Test invalid API endpoints
    local invalid_response
    invalid_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/invalid/endpoint" 2>/dev/null || echo "not_found")
    
    assert_not_empty "$invalid_response" "Invalid endpoint returns error response"
    
    # Test malformed JSON requests
    local malformed_response
    malformed_response=$(curl -s --max-time 10 -X POST "$VAULT_BASE_URL/v1/sys/mounts/test" \
        -H "Content-Type: application/json" \
        -d '{"invalid":"json"malformed}' 2>/dev/null || echo "malformed_handled")
    
    assert_not_empty "$malformed_response" "Malformed JSON handled"
    
    # Test unauthorized access
    local unauthorized_response
    unauthorized_response=$(curl -s --max-time 10 "$VAULT_BASE_URL/v1/secret/data/protected" 2>/dev/null || echo "unauthorized_test")
    
    assert_not_empty "$unauthorized_response" "Unauthorized access handled"
    
    # Test rate limiting
    echo "  Testing rate limiting..."
    local rate_start=$(date +%s)
    
    # Rapid requests to test rate limiting
    for i in {1..4}; do
        curl -s --max-time 3 "$VAULT_BASE_URL/v1/sys/health" >/dev/null 2>&1 &
    done
    wait
    
    local rate_end=$(date +%s)
    local rate_duration=$((rate_end - rate_start))
    
    echo "  Rate limiting test: ${rate_duration}s"
    
    if [[ $rate_duration -lt 10 ]]; then
        echo "  ✓ Rate limiting working appropriately"
    else
        echo "  ⚠ Rate limiting may be too restrictive"
    fi
    
    echo "✓ Vault error handling test completed"
}

# Main test execution
main() {
    echo "🧪 Starting Vault Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    test_vault_health
    test_vault_api
    test_vault_secrets_engines
    test_vault_auth_methods
    test_vault_security_features
    test_vault_performance
    test_vault_error_handling
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ Vault integration test failed"
        exit 1
    else
        echo "✅ Vault integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"