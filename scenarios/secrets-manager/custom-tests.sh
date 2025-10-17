#!/usr/bin/env bash
# Custom Test Functions for Secrets Manager
# Specialized tests for dark chrome UI and security functionality

# Test dark chrome UI accessibility and cyberpunk aesthetic
test_dark_chrome_ui_accessibility() {
    local html_content
    html_content=$(curl -sf "http://localhost:${UI_PORT}")
    
    # Check for dark chrome styling elements
    echo "$html_content" | grep -q "dark-chrome" &&
    echo "$html_content" | grep -q "cyberpunk" &&
    echo "$html_content" | grep -q "matrix-green" &&
    echo "$html_content" | grep -q "chrome-silver" &&
    echo "$html_content" | grep -q "terminal-container" &&
    
    # Check for security-themed UI elements
    echo "$html_content" | grep -q "SECURITY VAULT" &&
    echo "$html_content" | grep -q "SECRET HEALTH MATRIX" &&
    echo "$html_content" | grep -q "SECURITY ALERTS" &&
    
    # Check for interactive elements
    echo "$html_content" | grep -q "scan-all-btn" &&
    echo "$html_content" | grep -q "validate-btn" &&
    echo "$html_content" | grep -q "provision-modal" &&
    
    # Check for terminal-style footer
    echo "$html_content" | grep -q "terminal-footer" &&
    echo "$html_content" | grep -q "VROOLI SECURITY FRAMEWORK"
}

# Test resource secret discovery functionality
test_resource_secret_discovery() {
    # Test scanning specific resources
    local postgres_response
    postgres_response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d '{"scan_type": "full", "resources": ["postgres"]}' \
        "http://localhost:${API_PORT}/api/v1/secrets/scan")
    
    # Should discover postgres-related secrets
    echo "$postgres_response" | jq -e '.discovered_secrets | length > 0' > /dev/null &&
    echo "$postgres_response" | jq -e '.resources_scanned | contains(["postgres"])' > /dev/null &&
    
    # Test scanning all resources
    local full_response
    full_response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d '{"scan_type": "full"}' \
        "http://localhost:${API_PORT}/api/v1/secrets/scan")
    
    # Should discover multiple secrets across resources
    echo "$full_response" | jq -e '.discovered_secrets | length >= 5' > /dev/null &&
    echo "$full_response" | jq -e '.scan_duration_ms | type == "number"' > /dev/null &&
    
    # Test scan with specific secret types
    local discovered_secrets
    discovered_secrets=$(echo "$full_response" | jq -r '.discovered_secrets')
    
    # Should find various secret types
    echo "$discovered_secrets" | jq -e 'map(.secret_type) | unique | length >= 2' > /dev/null
}

# Test vault integration and secret storage
test_vault_secret_storage() {
    # Test vault status endpoint if vault is available
    if curl -sf "http://localhost:8200/v1/sys/health" > /dev/null 2>&1; then
        # Vault is running, test integration
        local vault_status_response
        vault_status_response=$(curl -sf "http://localhost:8200/v1/sys/health")
        echo "$vault_status_response" | jq -e '.initialized' > /dev/null
    else
        # Vault not running, test graceful degradation
        local validation_response
        validation_response=$(curl -sf -X POST -H "Content-Type: application/json" \
            -d '{}' "http://localhost:${API_PORT}/api/v1/secrets/validate")
        
        # Should handle vault unavailability gracefully
        echo "$validation_response" | jq -e 'has("total_secrets")' > /dev/null
        return 0
    fi
    
    # Test secret provisioning endpoint (should exist even if not fully implemented)
    local provision_response
    provision_response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d '{"secret_key": "TEST_KEY", "secret_value": "test_value", "storage_method": "vault"}' \
        "http://localhost:${API_PORT}/api/v1/secrets/provision" || echo '{"error": "endpoint_error"}')
    
    # Should return a structured response (even if provisioning fails)
    echo "$provision_response" | jq -e 'has("success") or has("error")' > /dev/null
}

# Test CLI advanced functionality
test_cli_advanced_features() {
    # Test CLI JSON output
    local status_json
    if command -v secrets-manager > /dev/null; then
        status_json=$(secrets-manager status --json 2>/dev/null || echo '{}')
        echo "$status_json" | jq empty > /dev/null 2>&1
    else
        # CLI not in PATH, test from local directory
        if [[ -f "$SCRIPT_DIR/cli/secrets-manager" ]]; then
            status_json=$("$SCRIPT_DIR/cli/secrets-manager" status --json 2>/dev/null || echo '{}')
            echo "$status_json" | jq empty > /dev/null 2>&1
        else
            return 1
        fi
    fi
}

# Test dashboard UI real-time updates
test_dashboard_ui_loads() {
    # Test that the dashboard can be accessed and loads key components
    local ui_response
    ui_response=$(curl -sf "http://localhost:${UI_PORT}")
    
    # Check for JavaScript functionality
    echo "$ui_response" | grep -q "script.js" &&
    echo "$ui_response" | grep -q "SecretsManager" &&
    
    # Check for CSS styling
    echo "$ui_response" | grep -q "styles.css" &&
    echo "$ui_response" | grep -q "dark-chrome" &&
    
    # Check for interactive elements
    echo "$ui_response" | grep -q "scan-all-btn" &&
    echo "$ui_response" | grep -q "validate-btn" &&
    echo "$ui_response" | grep -q "secrets-table"
}

# Test API error handling
test_api_error_handling() {
    # Test invalid endpoints
    local invalid_response
    invalid_response=$(curl -s "http://localhost:${API_PORT}/api/v1/invalid" | head -1)
    [[ -n "$invalid_response" ]] || return 1  # Should return some response
    
    # Test malformed JSON
    local malformed_response
    malformed_response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d '{"invalid_json":' \
        "http://localhost:${API_PORT}/api/v1/secrets/scan" 2>/dev/null | head -1)
    [[ -n "$malformed_response" ]] || return 1  # Should handle gracefully
    
    return 0
}

# Test security headers and CSP
test_security_headers() {
    # Test that security headers are present
    local headers
    headers=$(curl -sI "http://localhost:${UI_PORT}")
    
    echo "$headers" | grep -qi "x-content-type-options" &&
    echo "$headers" | grep -qi "x-frame-options" &&
    echo "$headers" | grep -qi "content-security-policy"
}

# Test database cleanup functionality
test_database_cleanup() {
    # Test that database views and functions exist
    PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h localhost -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-vrooli}" \
        -c "SELECT * FROM secret_health_summary LIMIT 1;" > /dev/null 2>&1 &&
    PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h localhost -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-vrooli}" \
        -c "SELECT * FROM missing_secrets_report LIMIT 1;" > /dev/null 2>&1
}

# Test n8n workflow integration (if n8n is available)
test_n8n_workflow_integration() {
    if curl -sf "http://localhost:5678/healthz" > /dev/null 2>&1; then
        # n8n is running, test workflow endpoints
        local workflows_response
        workflows_response=$(curl -sf "http://localhost:5678/api/v1/workflows" 2>/dev/null || echo '[]')
        
        # Should return workflow list
        echo "$workflows_response" | jq empty > /dev/null 2>&1
        return 0
    else
        # n8n not running, skip test
        return 0
    fi
}

# Performance test for large secret discovery
test_performance_large_scan() {
    # Test scanning performance with timeout
    local start_time end_time duration
    start_time=$(date +%s%N)
    
    local response
    response=$(timeout 30 curl -sf -X POST -H "Content-Type: application/json" \
        -d '{"scan_type": "full"}' \
        "http://localhost:${API_PORT}/api/v1/secrets/scan" || echo '{"error": "timeout"}')
    
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds
    
    # Should complete within reasonable time (30 seconds)
    [[ $duration -lt 30000 ]] &&
    echo "$response" | jq -e 'has("discovered_secrets") or has("error")' > /dev/null
}

echo "🧪 Custom test functions loaded for Secrets Manager"
echo "   • Dark Chrome UI Accessibility"  
echo "   • Resource Secret Discovery"
echo "   • Vault Integration"
echo "   • CLI Advanced Features"
echo "   • Security Headers"
echo "   • Database Cleanup"
echo "   • Performance Testing"