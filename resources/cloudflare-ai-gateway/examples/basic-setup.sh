#!/bin/bash
# Example: Basic Cloudflare AI Gateway setup

set -euo pipefail

echo "=== Cloudflare AI Gateway Basic Setup Example ==="
echo ""

# Check if resource is installed
if ! command -v resource-cloudflare-ai-gateway &>/dev/null; then
    echo "Installing Cloudflare AI Gateway resource..."
    vrooli resource install cloudflare-ai-gateway
fi

# Check status
echo "Checking gateway status..."
resource-cloudflare-ai-gateway status

# Check for credentials
if ! resource-vault get cloudflare_account_id &>/dev/null 2>&1; then
    echo ""
    echo "⚠️  Cloudflare credentials not configured!"
    echo ""
    echo "Please set up your credentials:"
    echo "  1. Get your Account ID from: https://dash.cloudflare.com"
    echo "  2. Create an API token at: https://dash.cloudflare.com/profile/api-tokens"
    echo "  3. Store credentials:"
    echo "     resource-vault set cloudflare_account_id YOUR_ACCOUNT_ID"
    echo "     resource-vault set cloudflare_api_token YOUR_API_TOKEN"
    echo ""
    exit 1
fi

# Start the gateway
echo "Starting gateway..."
resource-cloudflare-ai-gateway start

# Configure basic settings
echo "Configuring gateway..."
resource-cloudflare-ai-gateway configure --provider openrouter --cache-ttl 3600

# Create a sample configuration
cat > /tmp/gateway-config.json <<EOF
{
  "rules": [
    {
      "pattern": "/chat/completions",
      "cache": {
        "enabled": true,
        "ttl": 3600,
        "key": ["model", "messages", "temperature"]
      },
      "rate_limit": {
        "enabled": true,
        "requests_per_minute": 100
      }
    }
  ],
  "providers": [
    {
      "name": "openrouter",
      "endpoint": "https://openrouter.ai/api/v1",
      "priority": 1
    }
  ]
}
EOF

# Add the configuration
echo "Adding sample configuration..."
resource-cloudflare-ai-gateway content add --file /tmp/gateway-config.json --name sample-config

# List configurations
echo ""
echo "Available configurations:"
resource-cloudflare-ai-gateway content list

# Check final status
echo ""
echo "Final gateway status:"
resource-cloudflare-ai-gateway status

echo ""
echo "✓ Setup complete!"
echo ""
echo "The gateway is now active and will:"
echo "  - Cache AI requests for 1 hour"
echo "  - Rate limit to 100 requests per minute"
echo "  - Route through OpenRouter by default"
echo ""
echo "To view analytics: resource-cloudflare-ai-gateway logs"