#!/usr/bin/env bash
#
# Example: Process Document and Analyze with Ollama
# This integration shows how to extract text from documents and analyze with AI
#

set -euo pipefail

# Configuration
DOCUMENT="${1:-/tmp/test_document.txt}"
OLLAMA_MODEL="${2:-llama3.2:3b}"

echo "🚀 Document Analysis Pipeline"
echo "=============================="
echo "📄 Document: $DOCUMENT"
echo "🤖 Model: $OLLAMA_MODEL"
echo

# Step 1: Check services are running
echo "1️⃣ Checking services..."
if ! vrooli resource unstructured-io status --quiet 2>/dev/null; then
    echo "❌ Unstructured.io is not running. Starting..."
    vrooli resource unstructured-io develop
fi

if ! vrooli resource ollama status --quiet 2>/dev/null; then
    echo "❌ Ollama is not running. Starting..."
    vrooli resource ollama develop
fi

# Step 2: Process the document
echo "2️⃣ Processing document with Unstructured.io..."
EXTRACTED_TEXT=$(vrooli resource unstructured-io content process "$DOCUMENT" 2>/dev/null | jq -r '.[].text' | tr '\n' ' ')

if [ -z "$EXTRACTED_TEXT" ]; then
    echo "❌ Failed to extract text from document"
    exit 1
fi

echo "✅ Extracted $(echo "$EXTRACTED_TEXT" | wc -w) words"
echo

# Step 3: Analyze with Ollama
echo "3️⃣ Analyzing with Ollama..."
PROMPT="Please analyze the following text and provide a brief summary (2-3 sentences):

$EXTRACTED_TEXT"

ANALYSIS=$(echo "$PROMPT" | vrooli resource ollama content execute "$OLLAMA_MODEL" 2>/dev/null)

# Step 4: Display results
echo "📊 Analysis Results"
echo "==================="
echo "$ANALYSIS"
echo

echo "✅ Pipeline complete!"