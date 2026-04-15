#!/usr/bin/env bash
# Document Q&A with Ollama integration
# Process a document and ask questions about it

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
MANAGE_SCRIPT="${RESOURCE_DIR}/manage.sh"

# Parse arguments
DOCUMENT="${1:-}"
QUESTION="${2:-What are the key points in this document?}"

if [[ -z "$DOCUMENT" ]]; then
    echo "Usage: $0 <document_file> [question]"
    echo "Example: $0 report.pdf \"What are the main findings?\""
    exit 1
fi

if [[ ! -f "$DOCUMENT" ]]; then
    echo "Error: File not found: $DOCUMENT"
    exit 1
fi

# Check if Ollama is available
if ! command -v ollama &> /dev/null; then
    echo "Error: Ollama is not installed or not in PATH"
    echo "Please ensure Ollama is running on port 11434"
    exit 1
fi

echo "📄 Processing document: $(basename "$DOCUMENT")"
echo "❓ Question: $QUESTION"
echo

# Process document to markdown
CONTENT=$("$MANAGE_SCRIPT" --action process --file "$DOCUMENT" --output markdown --quiet yes)

if [[ -z "$CONTENT" ]]; then
    echo "Error: Failed to process document"
    exit 1
fi

# Ask Ollama about the document
echo "🤖 Ollama's Answer:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "$CONTENT" | ollama run llama3.1:8b "$QUESTION"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
