#!/bin/bash
# Direct deployment script (requires Node 20+)

echo "🚀 Deploying OG Worker..."

# Check Node version
NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 20 ]; then
    echo "❌ Error: Node.js version 20 or higher is required (you have v$NODE_VERSION)"
    echo ""
    echo "Options:"
    echo "1. Use ./deploy-with-docker.sh (uses Docker with Node 20)"
    echo "2. Install Node 20+ with nvm:"
    echo "   nvm install 20 && nvm use 20"
    echo ""
    exit 1
fi

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

# Show partial values for debugging
echo "✅ Node version: $(node -v)"
echo "✅ CLOUDFLARE_API_TOKEN is set: ${CLOUDFLARE_API_TOKEN:0:8}..."
echo "✅ CLOUDFLARE_ACCOUNT_ID is set: ${CLOUDFLARE_ACCOUNT_ID:0:8}..."
echo ""

# Deploy
echo "📦 Deploying to Cloudflare Workers..."
pnpm run deploy

echo ""
echo "✅ Deployment completed!"