#!/usr/bin/env bash
# Example: Basic secret storage and retrieval with HashiCorp Vault

set -e

echo "=== Vault Basic Secrets Example ==="
echo

# Check vault is running
echo "1. Checking Vault status..."
resource-vault status
echo

# Store a simple secret
echo "2. Storing a simple API key..."
resource-vault inject example_api_key "sk-1234567890abcdef"
echo "   ✅ Stored example_api_key"
echo

# Retrieve the secret
echo "3. Retrieving the API key..."
API_KEY=$(resource-vault get-secret example_api_key)
echo "   Retrieved: ${API_KEY:0:10}..." # Show first 10 chars only
echo

# Store a JSON credential
echo "4. Storing database credentials as JSON..."
DB_CREDS='{"host":"localhost","port":5432,"user":"dbuser","password":"dbpass123"}'
resource-vault inject db_credentials "$DB_CREDS"
echo "   ✅ Stored database credentials"
echo

# Retrieve and parse JSON
echo "5. Retrieving database credentials..."
CREDS=$(resource-vault get-secret db_credentials)
echo "   Retrieved credentials (JSON format)"
echo

# Store multiple related secrets
echo "6. Storing multiple OAuth tokens..."
resource-vault inject oauth/github_token "ghp_exampletoken123"
resource-vault inject oauth/gitlab_token "glpat-exampletoken456"
echo "   ✅ Stored GitHub and GitLab tokens"
echo

# List secrets (if supported)
echo "7. Listing stored secrets..."
resource-vault list-secrets / 2>/dev/null || echo "   Note: List command may require specific permissions"
echo

# Clean up examples
echo "8. Cleaning up example secrets..."
resource-vault delete-secret example_api_key --force 2>/dev/null || true
resource-vault delete-secret db_credentials --force 2>/dev/null || true
resource-vault delete-secret oauth/github_token --force 2>/dev/null || true
resource-vault delete-secret oauth/gitlab_token --force 2>/dev/null || true
echo "   ✅ Cleaned up example secrets"
echo

echo "=== Example Complete ==="
echo "This example demonstrated:"
echo "  • Storing simple string secrets"
echo "  • Storing JSON-formatted credentials"
echo "  • Organizing secrets with paths (oauth/)"
echo "  • Retrieving and using secrets"
echo "  • Cleaning up test data"