#!/usr/bin/env bash
# Store processed documents in MinIO with metadata
# Archives both original and processed versions

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/unstructured-io/integrations"
# Source trash module for safe cleanup
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
MANAGE_SCRIPT="${SCRIPT_DIR}/../manage.sh"

# Parse arguments
DOCUMENT="${1:-}"
BUCKET="${2:-documents}"

if [[ -z "$DOCUMENT" ]]; then
    echo "Usage: $0 <document_file> [bucket_name]"
    echo "Example: $0 contract.pdf legal-documents"
    exit 1
fi

if [[ ! -f "$DOCUMENT" ]]; then
    echo "Error: File not found: $DOCUMENT"
    exit 1
fi

# Check if MinIO client is available
if ! command -v mc &> /dev/null; then
    echo "Error: MinIO client (mc) is not installed"
    echo "Install with: wget https://dl.min.io/client/mc/release/linux-amd64/mc && chmod +x mc"
    exit 1
fi

# Check MinIO connection
if ! mc ls minio/ &> /dev/null; then
    echo "Error: MinIO is not configured or not accessible"
    echo "Configure with: mc alias set minio http://localhost:9000 minioadmin minioadmin"
    exit 1
fi

BASENAME=$(basename "$DOCUMENT" | sed 's/\.[^.]*$//')
EXTENSION="${DOCUMENT##*.}"
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "📄 Processing document: $(basename "$DOCUMENT")"

# Process document to JSON
JSON=$("$MANAGE_SCRIPT" --action process --file "$DOCUMENT" --output json --quiet yes)

if [[ -z "$JSON" ]]; then
    echo "Error: Failed to process document"
    exit 1
fi

# Extract metadata
ELEMENT_COUNT=$(echo "$JSON" | jq 'length')
FIRST_METADATA=$(echo "$JSON" | jq -r '.[0].metadata // {}')
FILE_SIZE=$(stat -f%z "$DOCUMENT" 2>/dev/null || stat -c%s "$DOCUMENT" 2>/dev/null || echo "0")

# Create enriched JSON with metadata
ENRICHED_JSON=$(jq -n \
    --arg filename "$(basename "$DOCUMENT")" \
    --arg original_path "$DOCUMENT" \
    --arg processed_at "$TIMESTAMP" \
    --arg file_size "$FILE_SIZE" \
    --arg element_count "$ELEMENT_COUNT" \
    --argjson elements "$JSON" \
    --argjson metadata "$FIRST_METADATA" \
    '{
        file_info: {
            name: $filename,
            original_path: $original_path,
            size_bytes: ($file_size | tonumber),
            processed_at: $processed_at
        },
        processing_info: {
            element_count: ($element_count | tonumber),
            strategy: "hi_res",
            processor: "unstructured-io"
        },
        document_metadata: $metadata,
        elements: $elements
    }')

# Save enriched JSON
TEMP_JSON="/tmp/${BASENAME}_processed.json"
echo "$ENRICHED_JSON" > "$TEMP_JSON"

# Ensure bucket exists
mc mb -p "minio/$BUCKET" 2>/dev/null || true
mc mb -p "minio/$BUCKET/originals" 2>/dev/null || true
mc mb -p "minio/$BUCKET/processed" 2>/dev/null || true

# Upload processed JSON with metadata
echo "⬆️  Uploading processed data to MinIO..."
mc cp "$TEMP_JSON" "minio/$BUCKET/processed/${BASENAME}.json" \
    --attr "Content-Type=application/json" \
    --attr "X-Amz-Meta-Original-File=$DOCUMENT" \
    --attr "X-Amz-Meta-Processed-Date=$TIMESTAMP" \
    --attr "X-Amz-Meta-Elements-Count=$ELEMENT_COUNT" \
    --attr "X-Amz-Meta-File-Size=$FILE_SIZE"

# Upload original document
echo "⬆️  Uploading original document..."
mc cp "$DOCUMENT" "minio/$BUCKET/originals/${BASENAME}.${EXTENSION}" \
    --attr "X-Amz-Meta-Processed=true" \
    --attr "X-Amz-Meta-Processed-Date=$TIMESTAMP" \
    --attr "X-Amz-Meta-Processed-Location=$BUCKET/processed/${BASENAME}.json"

# Clean up
trash::safe_remove "$TEMP_JSON" --temp

echo "✅ Successfully archived to MinIO"
echo "   Bucket: $BUCKET"
echo "   Original: minio/$BUCKET/originals/${BASENAME}.${EXTENSION}"
echo "   Processed: minio/$BUCKET/processed/${BASENAME}.json"
echo "   Elements: $ELEMENT_COUNT"