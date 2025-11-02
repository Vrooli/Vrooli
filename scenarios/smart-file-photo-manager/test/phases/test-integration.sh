#!/bin/bash
# Integration checks for smart-file-photo-manager

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_BASE_URL=""

if API_BASE_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  testing::phase::check "API health endpoint" curl -fsS "$API_BASE_URL/health"
  testing::phase::check "File listing endpoint" curl -fsS "$API_BASE_URL/api/files?limit=1"
else
  testing::phase::add_error "Unable to determine API URL for $SCENARIO_NAME"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

UPLOAD_PAYLOAD='{
  "filename": "integration-test-file.txt",
  "mime_type": "text/plain",
  "size_bytes": 128,
  "file_hash": "integration-hash-123",
  "storage_path": "/integration/test-file.txt",
  "folder_path": "/integration",
  "metadata": {"source": "integration-test"}
}'

UPLOAD_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/files" \
  -H "Content-Type: application/json" \
  -d "$UPLOAD_PAYLOAD")

FILE_ID=""
if command -v jq >/dev/null 2>&1; then
  FILE_ID=$(echo "$UPLOAD_RESPONSE" | jq -r '.file_id // empty')
else
  if echo "$UPLOAD_RESPONSE" | grep -q '"file_id"'; then
    FILE_ID=$(echo "$UPLOAD_RESPONSE" | sed -n 's/.*"file_id"[[:space:]]*:[[:space:]]*"\([^"]\+\)".*/\1/p' | head -n1)
  fi
fi

if [ -n "$FILE_ID" ]; then
  testing::phase::add_test passed
  testing::phase::check "Processing status endpoint" curl -fsS "$API_BASE_URL/api/processing-status/$FILE_ID"
else
  testing::phase::add_error "File upload request failed: $UPLOAD_RESPONSE"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Integration validation completed"
