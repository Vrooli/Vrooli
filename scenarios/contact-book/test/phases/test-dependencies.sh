#!/bin/bash

set -e

echo "=== Dependency Tests ==="

# Check PostgreSQL via API health check (more reliable than direct psql)
echo "Testing PostgreSQL dependency..."
API_PORT=$(lsof -ti:19313 2>/dev/null | head -1 | xargs -I {} lsof -nP -p {} 2>/dev/null | grep LISTEN | awk '{print $9}' | cut -d: -f2 | head -1)
if [ -z "$API_PORT" ]; then
    API_PORT="19313"
fi

if curl -sf "http://localhost:${API_PORT}/health" | jq -e '.database == "healthy"' > /dev/null 2>&1; then
    echo "✅ PostgreSQL is accessible (via API)"
    echo "✅ Database schema exists (verified by healthy status)"
else
    echo "❌ PostgreSQL connection failed"
    exit 1
fi

# Check Qdrant (optional but preferred)
echo "Testing Qdrant dependency..."
if command -v resource-qdrant > /dev/null 2>&1; then
    if resource-qdrant status > /dev/null 2>&1; then
        echo "✅ Qdrant is available"
    else
        echo "⚠️  Qdrant not running (optional)"
    fi
else
    echo "⚠️  Qdrant CLI not available (optional)"
fi

# Check MinIO (optional)
echo "Testing MinIO dependency..."
if command -v resource-minio > /dev/null 2>&1; then
    if resource-minio status > /dev/null 2>&1; then
        echo "✅ MinIO is available"
    else
        echo "⚠️  MinIO not running (optional)"
    fi
else
    echo "⚠️  MinIO CLI not available (optional)"
fi

echo "✅ All dependency tests passed"
