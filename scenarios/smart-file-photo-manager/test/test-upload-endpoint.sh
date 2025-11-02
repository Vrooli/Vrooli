#!/bin/bash
# Lightweight upload smoke test for business phase reuse

set -euo pipefail

API_BASE_URL="${SMART_FILE_MANAGER_API_URL:-http://localhost:${API_PORT:-16025}}"

response=$(curl -s -X POST "$API_BASE_URL/api/files" \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "business-phase-upload.txt",
    "mime_type": "text/plain",
    "size_bytes": 256,
    "file_hash": "business-phase-hash-123",
    "storage_path": "/tests/business-phase-upload.txt",
    "folder_path": "/tests/business"
  }' || echo "ERROR")

if echo "$response" | grep -q '"file_id"'; then
  exit 0
else
  echo "Upload endpoint test failed: $response" >&2
  exit 1
fi
