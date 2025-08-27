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
resource-vault content add --path "example_api_key" --value "sk-1234567890abcdef"
echo "   ✅ Stored example_api_key"
echo

# Retrieve the secret
echo "3. Retrieving the API key..."
API_KEY=$(resource-vault content get --path "example_api_key")
echo "   Retrieved: ${API_KEY:0:10}..." # Show first 10 chars only
echo

# Store a JSON credential
echo "4. Storing database credentials as JSON..."
DB_CREDS='{"host":"localhost","port":5432,"user":"dbuser","password":"dbpass123"}'
resource-vault content add --path "db_credentials" --value "$DB_CREDS"
echo "   ✅ Stored database credentials"
echo

# Retrieve and parse JSON
echo "5. Retrieving database credentials..."
CREDS=$(resource-vault content get --path "db_credentials")
echo "   Retrieved credentials (JSON format)"
echo

# Store multiple related secrets
echo "6. Storing multiple OAuth tokens..."
resource-vault content add --path "oauth/github_token" --value "ghp_exampletoken123"
resource-vault content add --path "oauth/gitlab_token" --value "glpat-exampletoken456"
echo "   ✅ Stored GitHub and GitLab tokens"
echo

# List secrets (if supported)
echo "7. Listing stored secrets..."
resource-vault content list --path "oauth" 2>/dev/null || echo "   Note: List command may require specific permissions"
echo

# Clean up examples
echo "8. Cleaning up example secrets..."
resource-vault content remove --path "example_api_key" 2>/dev/null || true
resource-vault content remove --path "db_credentials" 2>/dev/null || true
resource-vault content remove --path "oauth/github_token" 2>/dev/null || true
resource-vault content remove --path "oauth/gitlab_token" 2>/dev/null || true
echo "   ✅ Cleaned up example secrets"
echo

echo "=== Example Complete ==="
echo "This example demonstrated:"
echo "  • Storing simple string secrets"
echo "  • Storing JSON-formatted credentials"
echo "  • Organizing secrets with paths (oauth/)"
echo "  • Retrieving and using secrets"
echo "  • Cleaning up test data"