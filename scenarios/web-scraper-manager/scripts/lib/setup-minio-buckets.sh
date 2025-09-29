#!/usr/bin/env bash
# Setup MinIO buckets for Web Scraper Manager

set -euo pipefail

echo "🗄️ Setting up MinIO storage buckets..."

# Wait for MinIO to be ready
echo "Waiting for MinIO to be ready..."
timeout 60 bash -c 'until curl -sf http://localhost:9000/minio/health/ready > /dev/null 2>&1; do sleep 1; done'

# Configure MinIO client
mc alias set local http://localhost:9000 minioadmin minioadmin 2>/dev/null || true

# Create buckets
echo "Creating storage buckets..."
mc mb local/scraper-assets 2>/dev/null || echo "Bucket scraper-assets already exists"
mc mb local/screenshots 2>/dev/null || echo "Bucket screenshots already exists" 
mc mb local/exports 2>/dev/null || echo "Bucket exports already exists"

# Set bucket policies (public read for assets)
mc anonymous set download local/scraper-assets 2>/dev/null || true
mc anonymous set download local/screenshots 2>/dev/null || true

echo "✅ MinIO buckets configured successfully"