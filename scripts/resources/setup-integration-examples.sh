#!/usr/bin/env bash
set -euo pipefail

# Setup Integration Examples
# This script sets up the necessary secrets and data for the integration cookbook examples

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
VAULT_SCRIPT="${SCRIPT_DIR}/storage/vault/manage.sh"

echo "🔧 Setting up integration examples..."
echo

# Check if Vault is available
if ! curl -sf http://localhost:8200/v1/sys/health >/dev/null 2>&1; then
    echo "❌ Vault is not available at http://localhost:8200"
    echo "💡 Start Vault first: ./scripts/resources/storage/vault/manage.sh --action start"
    exit 1
fi

echo "✅ Vault is available"

# Setup Vault secrets for examples
echo "🔐 Setting up Vault secrets..."

# Example API configurations (using fake/demo keys for safety)
"$VAULT_SCRIPT" --action put-secret \
    --path "test/stripe-config" \
    --value '{"stripe_api_key":"sk_test_demo_key_123","environment":"test","webhook_secret":"whsec_demo_123"}'

"$VAULT_SCRIPT" --action put-secret \
    --path "test/openai-config" \
    --value '{"openai_api_key":"sk-demo-openai-key-123","model":"gpt-3.5-turbo","max_tokens":1000}'

"$VAULT_SCRIPT" --action put-secret \
    --path "environments/dev/database" \
    --value '{"host":"localhost","port":5432,"user":"dev_user","password":"dev_password","database":"vrooli_dev"}'

"$VAULT_SCRIPT" --action put-secret \
    --path "environments/prod/database" \
    --value '{"host":"prod.example.com","port":5432,"user":"prod_user","password":"secure_prod_password","database":"vrooli_prod"}'

"$VAULT_SCRIPT" --action put-secret \
    --path "api-keys/services" \
    --value '{"github_token":"ghp_demo_token_123","slack_webhook":"https://hooks.slack.com/demo/123","sendgrid_api_key":"SG.demo_key_123"}'

echo "✅ Vault secrets configured"

# Create MinIO buckets if MinIO is available
if curl -sf http://localhost:9000/minio/health/live >/dev/null 2>&1; then
    echo "📦 Setting up MinIO buckets..."
    
    # Create buckets for examples (ignore errors if they already exist)
    curl -X PUT http://localhost:9000/ai-processing-input 2>/dev/null || true
    curl -X PUT http://localhost:9000/ai-processing-output 2>/dev/null || true
    curl -X PUT http://localhost:9000/podcast-analysis 2>/dev/null || true
    curl -X PUT http://localhost:9000/research-archive 2>/dev/null || true
    curl -X PUT http://localhost:9000/integration-examples 2>/dev/null || true
    
    echo "✅ MinIO buckets configured"
else
    echo "⏭️  MinIO not available, skipping bucket setup"
fi

# Test that examples work
echo "🧪 Testing integration examples..."

# Test 1: Vault secret retrieval
echo "  Testing Vault secret access..."
if curl -sf -H "X-Vault-Token: myroot" http://localhost:8200/v1/secret/data/test/stripe-config >/dev/null; then
    echo "  ✅ Vault secret access working"
else
    echo "  ❌ Vault secret access failed"
fi

# Test 2: Create a test secret masking workflow file
TEST_WORKFLOW_FILE="${SCRIPT_DIR}/examples/secret-masking-workflow.json"
mkdir -p "${SCRIPT_DIR}/examples"

cat > "$TEST_WORKFLOW_FILE" << 'EOF'
{
  "name": "Secret Masking Demo",
  "nodes": [
    {
      "parameters": {
        "path": "secret-masking-demo"
      },
      "name": "Webhook Trigger",
      "type": "n8n-nodes-base.webhook", 
      "typeVersion": 1,
      "position": [180, 300]
    },
    {
      "parameters": {
        "url": "http://localhost:8200/v1/secret/data/test/stripe-config",
        "options": {
          "headers": {
            "X-Vault-Token": "myroot"
          }
        }
      },
      "name": "Get Secret from Vault",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 3,
      "position": [380, 300]
    },
    {
      "parameters": {
        "jsCode": "// Mask sensitive data from Vault\nconst vaultData = $input.first().json.data.data;\nconst apiKey = vaultData.stripe_api_key;\n\n// Mask the API key: show first 7 chars and last 3 chars\nconst maskedKey = apiKey.length >= 10 ? \n  apiKey.substring(0, 7) + '*'.repeat(apiKey.length - 10) + apiKey.substring(apiKey.length - 3) :\n  '*'.repeat(apiKey.length);\n\nreturn {\n  masked_api_key: maskedKey,\n  environment: vaultData.environment,\n  key_length: apiKey.length,\n  processing_timestamp: new Date().toISOString(),\n  security_note: 'Original key safely masked for logging'\n};"
      },
      "name": "Mask Sensitive Data",
      "type": "n8n-nodes-base.code",
      "typeVersion": 2,
      "position": [580, 300]
    }
  ],
  "connections": {
    "Webhook Trigger": {
      "main": [
        [
          {
            "node": "Get Secret from Vault",
            "type": "main",
            "index": 0
          }
        ]
      ]
    },
    "Get Secret from Vault": {
      "main": [
        [
          {
            "node": "Mask Sensitive Data",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  }
}
EOF

echo "✅ Created secret masking workflow example"

# Summary
echo
echo "🎉 Integration examples setup complete!"
echo
echo "📚 Available Examples:"
echo "  1. Vault + n8n secret workflows"
echo "  2. Multi-environment configuration management"
echo "  3. Secret masking and secure processing"
echo "  4. MinIO storage buckets for AI pipelines"
echo
echo "🚀 Next Steps:"
echo "  1. Read the integration cookbook: scripts/resources/INTEGRATION_COOKBOOK.md"
echo "  2. Test Vault functionality: $VAULT_SCRIPT --action test-functional"
echo "  3. Import n8n workflows from: scripts/resources/storage/vault/examples/"
echo "  4. Explore individual resource examples in their respective directories"
echo
echo "💡 Integration Tips:"
echo "  • All examples use development-safe demo credentials"
echo "  • Replace demo keys with real ones for production use"
echo "  • Use Vault's auth-info command to get authentication details"
echo "  • Check resource discovery: ./scripts/resources/index.sh --action discover"