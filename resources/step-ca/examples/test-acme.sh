#!/bin/bash
# Test ACME endpoint availability

set -euo pipefail

ACME_DIR="https://localhost:9010/acme/acme/directory"

echo "🧪 Testing Step-CA ACME endpoint..."

# Test ACME directory
echo -n "  Checking ACME directory... "
if timeout 5 curl -sk "$ACME_DIR" | grep -q "newNonce"; then
    echo "✅"
else
    echo "❌"
    exit 1
fi

# Test new-nonce endpoint
echo -n "  Checking new-nonce endpoint... "
NONCE_URL=$(curl -sk "$ACME_DIR" | jq -r '.newNonce')
if timeout 5 curl -sk -I "$NONCE_URL" | grep -q "200"; then
    echo "✅"
else
    echo "❌"
    exit 1
fi

echo "✅ ACME endpoint is functional"
echo ""
echo "📋 ACME Client Examples:"
echo ""
echo "# Using certbot (if installed):"
echo "certbot certonly \\"
echo "  --server $ACME_DIR \\"
echo "  --standalone \\"
echo "  -d example.local"
echo ""
echo "# Using acme.sh (if installed):"
echo "acme.sh --issue \\"
echo "  --server $ACME_DIR \\"
echo "  -d example.local \\"
echo "  --standalone"