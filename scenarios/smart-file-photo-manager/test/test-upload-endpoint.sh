#\!/bin/bash
# Test file upload endpoint

API_BASE_URL="${SMART_FILE_MANAGER_API_URL:-http://localhost:8090}"

echo "Testing file upload endpoint..."

# Test with minimal JSON payload (actual file upload would require multipart)
response=$(curl -s -X POST "$API_BASE_URL/api/files" \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "test-file.txt",
    "mime_type": "text/plain",
    "size_bytes": 100,
    "file_hash": "test-hash-123",
    "storage_path": "/test/path",
    "folder_path": "/test"
  }' || echo "ERROR")

if echo "$response" | grep -q "file_id"; then
  echo "✅ Upload endpoint responded successfully"
  exit 0
else
  echo "❌ Upload endpoint test failed: $response"
  exit 1
fi
