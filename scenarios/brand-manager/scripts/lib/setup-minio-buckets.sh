#!/bin/bash
# Setup MinIO buckets for Brand Manager
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/brand-manager/scripts/lib"
SCENARIO_DIR="${APP_ROOT}/scenarios/brand-manager"

# Source shared resource utilities
# shellcheck disable=SC1091
source "$SCRIPT_DIR/resource-utils.sh"

# MinIO Configuration
MINIO_PORT=$(get_resource_port "minio")
MINIO_ENDPOINT="${MINIO_ENDPOINT:-localhost:$MINIO_PORT}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"
MINIO_ALIAS="brand-manager-minio"

# Required buckets
BUCKETS=(
    "brand-logos"
    "brand-icons"
    "brand-exports"
    "brand-templates"
    "app-backups"
)

echo "🪣 Setting up MinIO buckets for Brand Manager..."

# Setup MinIO client alias
echo "📡 Configuring MinIO client..."
mc alias set "$MINIO_ALIAS" "http://$MINIO_ENDPOINT" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" || {
    echo "❌ Failed to setup MinIO alias" >&2
    exit 1
}

# Create buckets
for bucket in "${BUCKETS[@]}"; do
    echo "🪣 Checking bucket: $bucket"
    if ! mc ls "$MINIO_ALIAS/$bucket" >/dev/null 2>&1; then
        echo "📦 Creating bucket: $bucket"
        mc mb "$MINIO_ALIAS/$bucket" || {
            echo "❌ Failed to create bucket: $bucket" >&2
            exit 1
        }
        echo "✅ Created bucket: $bucket"
    else
        echo "✅ Bucket exists: $bucket"
    fi
done

echo "🎉 MinIO buckets setup completed successfully!"
echo "📍 Access MinIO console at: http://$MINIO_ENDPOINT"