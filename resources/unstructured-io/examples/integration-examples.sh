#!/usr/bin/env bash
# Integration examples for Unstructured.io with other Vrooli resources
# These examples demonstrate how to combine document processing with other services

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
MANAGE_SCRIPT="${SCRIPT_DIR}/../manage.sh"

echo "=== Unstructured.io Integration Examples ==="
echo
echo "These examples show how to integrate Unstructured.io with other Vrooli resources"
echo "for powerful document processing workflows."
echo

#######################################
# Example 1: Document Q&A with Ollama
#######################################
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1. Document Q&A Pipeline with Ollama"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
echo "Process a document and ask questions about it using Ollama:"
echo
cat << 'EXAMPLE1'
#!/bin/bash
# doc-qa.sh - Document Q&A script

DOCUMENT="$1"
QUESTION="${2:-What are the key points in this document?}"

# Process document to markdown
echo "Processing document..."
CONTENT=$($MANAGE_SCRIPT --action process --file "$DOCUMENT" --output markdown --quiet yes)

# Ask Ollama about the document
echo "Asking Ollama: $QUESTION"
echo "$CONTENT" | ollama run llama3.1:8b "$QUESTION"
EXAMPLE1
echo
echo "Usage: ./doc-qa.sh report.pdf \"What are the main findings?\""
echo

#######################################
# Example 2: Document Storage with MinIO
#######################################
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2. Document Archival Pipeline with MinIO"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
echo "Process documents and store structured data in MinIO:"
echo
cat << 'EXAMPLE2'
#!/bin/bash
# doc-archive.sh - Archive documents with metadata

DOCUMENT="$1"
BUCKET="${2:-documents}"
BASENAME=$(basename "$DOCUMENT" | sed 's/\.[^.]*$//')

# Process document
echo "Processing document..."
JSON=$($MANAGE_SCRIPT --action process --file "$DOCUMENT" --output json --quiet yes)

# Extract metadata
METADATA=$(echo "$JSON" | jq -r '.[0].metadata // {}')

# Save structured data
echo "$JSON" > "/tmp/${BASENAME}.json"

# Upload to MinIO with metadata
mc cp "/tmp/${BASENAME}.json" "minio/$BUCKET/" \
  --attr "original_file=$DOCUMENT" \
  --attr "processed_date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --attr "elements_count=$(echo "$JSON" | jq 'length')"

# Also upload original document
mc cp "$DOCUMENT" "minio/$BUCKET/originals/"

echo "✅ Archived to MinIO bucket: $BUCKET"
EXAMPLE2
echo
echo "Usage: ./doc-archive.sh contract.pdf legal-documents"
echo

#######################################
# Example 3: Document Search with Qdrant
#######################################
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3. Semantic Search Pipeline with Qdrant"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
echo "Build a searchable knowledge base using Qdrant:"
echo
cat << 'EXAMPLE3'
#!/bin/bash
# doc-index.sh - Index documents for semantic search

DOCUMENT="$1"
COLLECTION="${2:-knowledge-base}"

# Process document
echo "Processing document..."
CONTENT=$($MANAGE_SCRIPT --action process --file "$DOCUMENT" --output markdown --quiet yes)

# Generate embeddings using Ollama
echo "Generating embeddings..."
EMBEDDING=$(echo "$CONTENT" | ollama embeddings llama3.1:8b)

# Store in Qdrant
curl -X PUT "http://localhost:6333/collections/$COLLECTION/points" \
  -H "Content-Type: application/json" \
  -d "{
    \"points\": [{
      \"id\": \"$(uuidgen)\",
      \"vector\": $EMBEDDING,
      \"payload\": {
        \"file\": \"$DOCUMENT\",
        \"content\": \"$CONTENT\",
        \"indexed_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
      }
    }]
  }"

echo "✅ Indexed in Qdrant collection: $COLLECTION"
EXAMPLE3
echo
echo "Usage: ./doc-index.sh handbook.pdf employee-docs"
echo

#######################################
# Example 4: Multi-Resource Workflow
#######################################
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4. Complete Document Intelligence Pipeline"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
echo "Process, analyze, store, and monitor documents:"
echo
cat << 'EXAMPLE4'
#!/bin/bash
# doc-pipeline.sh - Complete document processing pipeline

process_document() {
    local FILE="$1"
    local FILENAME=$(basename "$FILE")
    
    echo "📄 Processing: $FILENAME"
    
    # 1. Extract content with Unstructured.io
    JSON=$($MANAGE_SCRIPT --action process --file "$FILE" --output json --quiet yes)
    MARKDOWN=$($MANAGE_SCRIPT --action process --file "$FILE" --output markdown --quiet yes)
    
    # 2. Analyze with Ollama
    SUMMARY=$(echo "$MARKDOWN" | ollama run llama3.1:8b "Summarize in 3 bullet points")
    ENTITIES=$(echo "$MARKDOWN" | ollama run llama3.1:8b "Extract all names, dates, and organizations")
    
    # 3. Extract tables if present
    TABLES=$($MANAGE_SCRIPT --action extract-tables --file "$FILE" --quiet yes)
    
    # 4. Store in MinIO
    RESULT=$(jq -n \
        --arg file "$FILENAME" \
        --argjson content "$JSON" \
        --arg summary "$SUMMARY" \
        --arg entities "$ENTITIES" \
        --arg tables "$TABLES" \
        '{
            file: $file,
            processed_at: now | todate,
            content: $content,
            analysis: {
                summary: $summary,
                entities: $entities,
                tables: $tables
            }
        }')
    
    echo "$RESULT" | mc pipe "minio/processed/${FILENAME%.pdf}.json"
    
    # 5. Send metrics to QuestDB
    local ELEMENT_COUNT=$(echo "$JSON" | jq 'length')
    echo "document_processing,file=$FILENAME elements=$ELEMENT_COUNT $(date +%s)000000000" | \
        nc localhost 9011
    
    echo "✅ Processed: $FILENAME"
}

# Process all PDFs in a directory
for file in "$1"/*.pdf; do
    [ -f "$file" ] && process_document "$file"
done
EXAMPLE4
echo
echo "Usage: ./doc-pipeline.sh /path/to/documents"
echo

#######################################
# Example 5: Real-time Monitoring
#######################################
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "5. Document Processing Monitor with Node-RED"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
echo "Monitor document processing in real-time:"
echo
cat << 'EXAMPLE5'
#!/bin/bash
# doc-monitor.sh - Send processing events to Node-RED

DOCUMENT="$1"
WEBHOOK_URL="http://localhost:1880/document-processed"

# Process document
START_TIME=$(date +%s)
RESULT=$($MANAGE_SCRIPT --action process --file "$DOCUMENT" --output json --quiet yes)
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Send event to Node-RED
curl -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d "{
    \"file\": \"$DOCUMENT\",
    \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"duration_seconds\": $DURATION,
    \"elements_count\": $(echo "$RESULT" | jq 'length'),
    \"status\": \"success\"
  }"

echo "✅ Event sent to Node-RED"
EXAMPLE5
echo
echo "Usage: ./doc-monitor.sh invoice.pdf"
echo

#######################################
# Example 6: Batch Processing with n8n
#######################################
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "6. Trigger n8n Workflow for Batch Processing"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
echo "Process multiple documents through n8n workflow:"
echo
cat << 'EXAMPLE6'
#!/bin/bash
# doc-batch-n8n.sh - Batch process through n8n

DIRECTORY="$1"
WORKFLOW_WEBHOOK="http://localhost:5678/webhook/process-documents"

# Create file list
FILES=()
for file in "$DIRECTORY"/*.{pdf,docx,xlsx}; do
    [ -f "$file" ] && FILES+=("$file")
done

# Send to n8n workflow
curl -X POST "$WORKFLOW_WEBHOOK" \
  -H "Content-Type: application/json" \
  -d "{
    \"files\": $(printf '%s\n' "${FILES[@]}" | jq -R . | jq -s .),
    \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"options\": {
      \"strategy\": \"hi_res\",
      \"output\": \"json\",
      \"store_in_minio\": true,
      \"generate_summary\": true
    }
  }"

echo "✅ Batch job sent to n8n (${#FILES[@]} files)"
EXAMPLE6
echo
echo "Usage: ./doc-batch-n8n.sh /path/to/documents"
echo

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Additional Integration Ideas"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
echo "• Use SearXNG to find documents online, then process with Unstructured.io"
echo "• Combine with Whisper to transcribe audio → convert to text → process"
echo "• Use Agent-S2 to download documents from websites → process automatically"
echo "• Store processing history in Redis for deduplication"
echo "• Use Vault to manage API keys for document sources"
echo "• Create ComfyUI workflows that include document text in image generation"
echo
echo "For more examples, see the workflow-integration.json file in this directory."
echo