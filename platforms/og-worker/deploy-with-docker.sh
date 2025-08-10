#!/bin/bash
# Deploy script using Docker with Node 20

echo "🚀 Deploying OG Worker using Docker..."

# Check if required environment variables are set
if [ -z "$CLOUDFLARE_API_TOKEN" ]; then
    echo "❌ Error: CLOUDFLARE_API_TOKEN environment variable is not set"
    echo ""
    echo "To deploy, you need to:"
    echo "1. Go to https://dash.cloudflare.com/profile/api-tokens"
    echo "2. Create a token with 'Edit Cloudflare Workers' permissions"
    echo "3. Set the token: export CLOUDFLARE_API_TOKEN='your-token-here'"
    echo ""
    exit 1
fi

if [ -z "$CLOUDFLARE_ACCOUNT_ID" ]; then
    echo "❌ Error: CLOUDFLARE_ACCOUNT_ID environment variable is not set"
    echo ""
    echo "To find your account ID:"
    echo "1. Go to https://dash.cloudflare.com/"
    echo "2. Select your domain"
    echo "3. On the right sidebar, find 'Account ID'"
    echo "4. Set it: export CLOUDFLARE_ACCOUNT_ID='your-account-id'"
    echo ""
    exit 1
fi

# Show partial values for debugging (first 8 chars only)
echo "✅ CLOUDFLARE_API_TOKEN is set: ${CLOUDFLARE_API_TOKEN:0:8}..."
echo "✅ CLOUDFLARE_ACCOUNT_ID is set: ${CLOUDFLARE_ACCOUNT_ID:0:8}..."
echo ""

# Build and run deployment in Docker
echo "📦 Installing dependencies and deploying..."
docker run --rm \
  -v $(pwd):/app \
  -w /app \
  -e CLOUDFLARE_API_TOKEN="$CLOUDFLARE_API_TOKEN" \
  -e CLOUDFLARE_ACCOUNT_ID="$CLOUDFLARE_ACCOUNT_ID" \
  node:20-alpine \
  sh -c "npm install -g pnpm && pnpm install && npx wrangler deploy"

echo ""
echo "✅ Deployment script completed!"