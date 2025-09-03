#!/bin/bash
# Universal health check script with proper retry logic
# Usage: wait-for-service.sh <url> [max_attempts] [delay]

set -euo pipefail

URL="${1:-}"
MAX_ATTEMPTS="${2:-15}"
DELAY="${3:-2}"

if [[ -z "$URL" ]]; then
    echo "Error: URL is required"
    exit 1
fi

echo "Waiting for service at $URL (max $MAX_ATTEMPTS attempts, ${DELAY}s delay)..."

attempt=1
while [[ $attempt -le $MAX_ATTEMPTS ]]; do
    if curl -sf --max-time 5 --connect-timeout 3 "$URL" > /dev/null 2>&1; then
        echo "✓ Service is ready at $URL (attempt $attempt/$MAX_ATTEMPTS)"
        exit 0
    fi
    
    echo "  Attempt $attempt/$MAX_ATTEMPTS failed, waiting ${DELAY}s..."
    sleep "$DELAY"
    attempt=$((attempt + 1))
done

echo "✗ Service at $URL failed to respond after $MAX_ATTEMPTS attempts"
exit 1