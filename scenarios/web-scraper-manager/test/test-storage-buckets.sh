#!/usr/bin/env bash
# Test MinIO bucket accessibility

set -euo pipefail

echo "🔍 Testing MinIO storage buckets..."

# Find MinIO port - MinIO uses direct port mapping
MINIO_PORT=9000
if [ -z "$MINIO_PORT" ]; then
    echo "⚠️  MinIO container not found or port not mapped; skipping MinIO test"
    exit 0
fi

# Test MinIO is ready
echo "Testing MinIO connection on port $MINIO_PORT..."
if ! curl -sf "http://localhost:${MINIO_PORT}/minio/health/ready" > /dev/null; then
    echo "❌ MinIO is not ready"
    exit 1
fi

# Check if mc (MinIO client) is available
if ! command -v mc >/dev/null 2>&1; then
    echo "⚠️  MinIO client (mc) not available; skipping bucket tests"
    echo "✅ MinIO is running and healthy"
    exit 0
fi

# Configure MinIO client
mc alias set test-local "http://localhost:${MINIO_PORT}" minioadmin minioadmin 2>/dev/null || true

# Test required buckets exist using HTTP API
echo "Testing required buckets exist..."
REQUIRED_BUCKETS=("scraper-assets" "screenshots" "exports")

for bucket in "${REQUIRED_BUCKETS[@]}"; do
    # Check if bucket exists by trying to access it via HTTP
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -I "http://localhost:${MINIO_PORT}/${bucket}/" 2>/dev/null || echo "000")
    if [[ "$HTTP_STATUS" != "403" && "$HTTP_STATUS" != "200" ]]; then
        echo "❌ Required bucket '$bucket' does not exist or is not accessible (HTTP status: $HTTP_STATUS)"
        echo "   Hint: Run 'bash scripts/lib/setup-minio-buckets.sh' to create buckets"
        exit 1
    fi
    echo "✅ Bucket '$bucket' exists"
done

echo "✅ MinIO storage bucket tests passed"