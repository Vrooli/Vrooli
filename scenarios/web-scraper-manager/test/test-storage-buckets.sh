#!/usr/bin/env bash
# Test MinIO bucket accessibility

set -euo pipefail

echo "🔍 Testing MinIO storage buckets..."

# Test MinIO is ready
echo "Testing MinIO connection..."
if ! curl -sf "http://localhost:${RESOURCE_PORTS[minio]}/minio/health/ready" > /dev/null; then
    echo "❌ MinIO is not ready"
    exit 1
fi

# Configure MinIO client
mc alias set test-local "http://localhost:${RESOURCE_PORTS[minio]}" minioadmin minioadmin 2>/dev/null || true

# Test required buckets exist
echo "Testing required buckets exist..."
REQUIRED_BUCKETS=("scraper-assets" "screenshots" "exports")

for bucket in "${REQUIRED_BUCKETS[@]}"; do
    if ! mc ls test-local/"$bucket" > /dev/null 2>&1; then
        echo "❌ Required bucket '$bucket' does not exist or is not accessible"
        exit 1
    fi
done

# Test write/read operations
echo "Testing bucket write/read operations..."
TEST_FILE="test-$(date +%s).txt"
TEST_CONTENT="Web Scraper Manager test file"

# Write test file
echo "$TEST_CONTENT" | mc pipe test-local/scraper-assets/"$TEST_FILE" 2>/dev/null || {
    echo "❌ Failed to write test file to scraper-assets bucket"
    exit 1
}

# Read test file
RETRIEVED_CONTENT=$(mc cat test-local/scraper-assets/"$TEST_FILE" 2>/dev/null || echo "FAILED")
if [[ "$RETRIEVED_CONTENT" != "$TEST_CONTENT" ]]; then
    echo "❌ Failed to read test file from scraper-assets bucket"
    exit 1
fi

# Clean up test file
mc rm test-local/scraper-assets/"$TEST_FILE" 2>/dev/null || true

echo "✅ MinIO storage bucket tests passed"