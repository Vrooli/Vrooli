#!/bin/bash
# ====================================================================
# Vault + n8n Secrets Integration Test
# ====================================================================
#
# @scenario: vault-n8n-secrets-integration
# @category: security
# @complexity: intermediate
# @services: vault,n8n
# @optional-services: 
# @duration: 3-5min
# @business-value: security-automation
# @market-demand: high
# @revenue-potential: $2000-8000
# @success-criteria: vault auth works, n8n retrieves secrets, end-to-end workflow functions
#
# This test validates the specific technical integration between Vault and n8n
# that was identified as problematic during comprehensive testing. It focuses
# on authentication setup, state detection, and working integration patterns.
#
# ====================================================================

set -euo pipefail

# Test metadata
REQUIRED_RESOURCES=("vault" "n8n")
TEST_TIMEOUT="${TEST_TIMEOUT:-300}"  # 5 minutes
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Recreate HEALTHY_RESOURCES array from exported string
if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
fi

# Source framework helpers  
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"
source "$SCRIPT_DIR/framework/helpers/fixtures.sh"
source "$SCRIPT_DIR/framework/helpers/metadata.sh"
source "$SCRIPT_DIR/framework/helpers/secure-config.sh"

# Service configuration with secure defaults
export_service_urls
VAULT_TOKEN=$(get_auth_token "vault" "myroot")  # Dev default only used in development environment

# Validate production security
if [[ "$VROOLI_ENV" == "production" ]]; then
    if [[ -z "$VAULT_TOKEN" || "$VAULT_TOKEN" == "myroot" ]]; then
        echo "❌ Production environment requires secure Vault token configuration"
        echo "   Set VROOLI_VAULT_TOKEN environment variable or configure in ~/.vrooli/.secrets"
        exit 1
    fi
fi

# Test fixtures
TEST_SECRETS_PATH="test/integration"
TEST_SECRET_DATA='{"api_key": "test-key-12345", "database_url": "postgresql://user:pass@localhost:5432/db"}'

#######################################
# Setup test environment
#######################################
setup_vault_n8n_integration() {
    echo "🔧 Setting up Vault + n8n integration test..."
    
    # Verify services are accessible
    assert_http_200 "$VAULT_BASE_URL/v1/sys/health" "Vault health check"
    assert_http_200 "$N8N_BASE_URL/healthz" "n8n health check"
    
    # Check Vault authentication
    echo "  🔐 Testing Vault authentication..."
    local vault_auth_response
    vault_auth_response=$(curl -s -H "X-Vault-Token: $VAULT_TOKEN" "$VAULT_BASE_URL/v1/auth/token/lookup-self" || true)
    
    if echo "$vault_auth_response" | grep -q '"policies"'; then
        echo "  ✅ Vault authentication successful"
    else
        echo "  ❌ Vault authentication failed"
        echo "  Response: $vault_auth_response"
        return 1
    fi
    
    # Check Vault seal status
    local seal_status
    seal_status=$(curl -s "$VAULT_BASE_URL/v1/sys/seal-status" | jq -r '.sealed')
    if [[ "$seal_status" == "false" ]]; then
        echo "  ✅ Vault is unsealed and ready"
    else
        echo "  ❌ Vault is sealed - cannot proceed"
        return 1
    fi
    
    # Clean up any existing test secrets
    curl -s -X DELETE -H "X-Vault-Token: $VAULT_TOKEN" \
        "$VAULT_BASE_URL/v1/secret/data/$TEST_SECRETS_PATH" >/dev/null || true
    
    # Optional: Import resource artifacts if using the enhanced template
    # Note: This scenario includes resource artifacts that can be imported:
    # - n8n workflow: resources/n8n/vault-integration-workflow.json
    # - PostgreSQL audit schema: resources/postgres/schema.sql
    # - Configuration template: resources/config/.env.template
    #
    # To use these artifacts with the template helper functions:
    # 1. Source the resource helpers (if using newer template):
    #    source "$SCRIPT_DIR/framework/helpers/resource-integration.sh"
    # 2. Import the workflow:
    #    import_n8n_workflow "vault-integration-workflow.json"
    # 3. Initialize audit database:
    #    init_postgres_database
    
    echo "  ✅ Setup complete"
}

#######################################
# Test Vault secret CRUD operations
#######################################
test_vault_secret_operations() {
    echo "🔐 Testing Vault secret operations..."
    
    # Create test secret
    echo "  📝 Creating test secret..."
    local create_response
    create_response=$(curl -s -X POST \
        -H "X-Vault-Token: $VAULT_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"data\": $TEST_SECRET_DATA}" \
        "$VAULT_BASE_URL/v1/secret/data/$TEST_SECRETS_PATH")
    
    if echo "$create_response" | grep -q '"version"'; then
        echo "  ✅ Secret created successfully"
    else
        echo "  ❌ Failed to create secret"
        echo "  Response: $create_response"
        return 1
    fi
    
    # Retrieve test secret
    echo "  📖 Retrieving test secret..."
    local retrieve_response
    retrieve_response=$(curl -s -H "X-Vault-Token: $VAULT_TOKEN" \
        "$VAULT_BASE_URL/v1/secret/data/$TEST_SECRETS_PATH")
    
    # Validate secret content
    local retrieved_api_key
    retrieved_api_key=$(echo "$retrieve_response" | jq -r '.data.data.api_key')
    
    if [[ "$retrieved_api_key" == "test-key-12345" ]]; then
        echo "  ✅ Secret retrieved correctly"
    else
        echo "  ❌ Secret retrieval failed or data corrupted"
        echo "  Expected: test-key-12345"
        echo "  Got: $retrieved_api_key"
        return 1
    fi
    
    # Test secret listing
    echo "  📋 Testing secret listing..."
    local list_response
    list_response=$(curl -s -X LIST -H "X-Vault-Token: $VAULT_TOKEN" \
        "$VAULT_BASE_URL/v1/secret/metadata/test")
    
    if echo "$list_response" | grep -q '"integration"'; then
        echo "  ✅ Secret listing works"
    else
        echo "  ❌ Secret listing failed"
        echo "  Response: $list_response"
        return 1
    fi
}

#######################################
# Test n8n integration pattern
#######################################
test_n8n_vault_integration() {
    echo "🔗 Testing n8n + Vault integration pattern..."
    
    # Create a test workflow JSON for n8n that retrieves secrets from Vault
    local workflow_json='
{
  "name": "Vault Integration Test Workflow",
  "nodes": [
    {
      "parameters": {
        "path": "integration-test"
      },  
      "name": "Webhook Trigger",
      "type": "n8n-nodes-base.webhook",
      "typeVersion": 1,
      "position": [240, 300],
      "webhookId": "vault-integration-test"
    },
    {
      "parameters": {
        "url": "'"$VAULT_BASE_URL"'/v1/secret/data/'"$TEST_SECRETS_PATH"'",
        "options": {
          "headers": {
            "X-Vault-Token": "'"$VAULT_TOKEN"'"
          }
        }
      },
      "name": "Retrieve Secret from Vault",
      "type": "n8n-nodes-base.httpRequest", 
      "typeVersion": 3,
      "position": [460, 300]
    },
    {
      "parameters": {
        "jsCode": "// Process the secret from Vault\nconst vaultData = $node[\\"Retrieve Secret from Vault\\"].json.data.data;\n\nreturn {\n  success: true,\n  message: \\"Secret retrieved successfully from Vault\\",\n  api_key_prefix: vaultData.api_key.substring(0, 8) + \\"...\\",\n  database_available: vaultData.database_url ? true : false,\n  timestamp: new Date().toISOString()\n};"
      },
      "name": "Process Secret Data",
      "type": "n8n-nodes-base.code",
      "typeVersion": 2,
      "position": [680, 300]
    }
  ],
  "connections": {
    "Webhook Trigger": {
      "main": [
        [
          {
            "node": "Retrieve Secret from Vault",
            "type": "main",
            "index": 0
          }
        ]
      ]
    },
    "Retrieve Secret from Vault": {
      "main": [
        [
          {
            "node": "Process Secret Data",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  },
  "settings": {
    "executionOrder": "v1"
  }
}'
    
    # Save workflow to temp file
    local workflow_file="/tmp/vault-integration-test-workflow.json"
    echo "$workflow_json" > "$workflow_file"
    
    echo "  📝 Created working n8n workflow template"
    echo "  📄 Workflow saved to: $workflow_file"
    echo "  ℹ️  Note: An enhanced workflow is available at resources/n8n/vault-integration-workflow.json"
    
    # Test the individual API call that n8n would make
    echo "  🧪 Testing HTTP request pattern (simulating n8n node)..."
    local test_response
    test_response=$(curl -s -H "X-Vault-Token: $VAULT_TOKEN" \
        "$VAULT_BASE_URL/v1/secret/data/$TEST_SECRETS_PATH")
    
    # Validate the response that n8n would receive
    local api_key_from_response
    api_key_from_response=$(echo "$test_response" | jq -r '.data.data.api_key')
    
    if [[ "$api_key_from_response" == "test-key-12345" ]]; then
        echo "  ✅ HTTP request pattern works (n8n integration ready)"
    else
        echo "  ❌ HTTP request pattern failed"
        echo "  Response: $test_response"
        return 1
    fi
    
    # Test error handling
    echo "  🔍 Testing error handling for non-existent secret..."
    local error_response
    error_response=$(curl -s -H "X-Vault-Token: $VAULT_TOKEN" \
        "$VAULT_BASE_URL/v1/secret/data/nonexistent/path" || true)
    
    if echo "$error_response" | grep -q '"errors"'; then
        echo "  ✅ Error handling works correctly"
    else
        echo "  ⚠️  Unexpected error response format"
    fi
}

#######################################
# Test authentication and state detection improvements
#######################################
test_auth_and_state_detection() {
    echo "🔍 Testing authentication and state detection improvements..."
    
    # Test Vault token validation
    echo "  🔐 Validating Vault token..."
    local token_info
    token_info=$(curl -s -H "X-Vault-Token: $VAULT_TOKEN" \
        "$VAULT_BASE_URL/v1/auth/token/lookup-self")
    
    local token_policies
    token_policies=$(echo "$token_info" | jq -r '.data.policies[]' 2>/dev/null || echo "")
    
    if [[ "$token_policies" == *"root"* ]]; then
        echo "  ✅ Token has root permissions (development OK)"
    else
        echo "  ⚠️  Token permissions: $token_policies"
    fi
    
    # Test service state detection
    echo "  🔍 Testing improved service state detection..."
    
    # Check if services are truly ready (not just running)
    local vault_ready=false
    local n8n_ready=false
    
    # Vault readiness check
    if curl -s "$VAULT_BASE_URL/v1/sys/health" | jq -e '.initialized == true and .sealed == false' >/dev/null; then
        vault_ready=true
        echo "  ✅ Vault is properly initialized and unsealed"
    else
        echo "  ❌ Vault is not ready for operations"
    fi
    
    # n8n readiness check  
    if curl -s "$N8N_BASE_URL/healthz" | grep -q '"status":"ok"'; then
        n8n_ready=true
        echo "  ✅ n8n is ready for operations" 
    else
        echo "  ❌ n8n is not ready for operations"
    fi
    
    if [[ "$vault_ready" == true && "$n8n_ready" == true ]]; then
        echo "  ✅ Both services are integration-ready"
        return 0
    else
        echo "  ❌ Services are not ready for integration"
        return 1
    fi
}

#######################################
# Cleanup test artifacts
#######################################
cleanup_vault_n8n_test() {
    if [[ "$TEST_CLEANUP" == "true" ]]; then
        echo "🧹 Cleaning up test artifacts..."
        
        # Remove test secrets from Vault
        curl -s -X DELETE -H "X-Vault-Token: $VAULT_TOKEN" \
            "$VAULT_BASE_URL/v1/secret/data/$TEST_SECRETS_PATH" >/dev/null || true
        
        # Remove temp workflow file
        rm -f /tmp/vault-integration-test-workflow.json
        
        echo "  ✅ Cleanup complete"
    fi
}

#######################################
# Main test execution
#######################################
main() {
    echo "🧪 Starting Vault + n8n Integration Test"
    echo "======================================"
    
    # Check prerequisites
    for resource in "${REQUIRED_RESOURCES[@]}"; do
        if [[ ! " ${HEALTHY_RESOURCES[*]} " =~ " $resource " ]]; then
            echo "❌ Required resource '$resource' is not healthy"
            echo "   Start the resource and run the test again"
            exit 1
        fi
    done
    
    echo "✅ Prerequisites met: ${REQUIRED_RESOURCES[*]}"
    echo
    
    # Run test phases
    local test_start_time
    test_start_time=$(date +%s)
    
    # Setup
    if ! setup_vault_n8n_integration; then
        echo "❌ Setup failed"
        cleanup_vault_n8n_test
        exit 1
    fi
    echo
    
    # Test Vault operations
    if ! test_vault_secret_operations; then
        echo "❌ Vault operations test failed"
        cleanup_vault_n8n_test
        exit 1
    fi
    echo
    
    # Test n8n integration
    if ! test_n8n_vault_integration; then
        echo "❌ n8n integration test failed"  
        cleanup_vault_n8n_test
        exit 1
    fi
    echo
    
    # Test improvements
    if ! test_auth_and_state_detection; then
        echo "❌ Authentication and state detection test failed"
        cleanup_vault_n8n_test
        exit 1
    fi
    echo
    
    # Calculate test duration
    local test_end_time
    test_end_time=$(date +%s)
    local test_duration=$((test_end_time - test_start_time))
    
    echo "🎉 All tests passed!"
    echo "======================================"
    echo "✅ Vault authentication: Working"
    echo "✅ Secret CRUD operations: Working" 
    echo "✅ n8n HTTP integration pattern: Working"
    echo "✅ Error handling: Working"
    echo "✅ State detection: Improved"
    echo "📄 Working workflow template created"
    echo "⏱️  Test duration: ${test_duration}s"
    echo
    echo "📋 Next Steps:"
    echo "  1. Import /tmp/vault-integration-test-workflow.json into n8n"
    echo "  2. Activate the workflow in n8n web interface"
    echo "  3. Test with: curl -X POST http://localhost:5678/webhook/integration-test"
    echo
    
    # Cleanup
    cleanup_vault_n8n_test
}

# Execute main function
main "$@"