#!/usr/bin/env bash
# Basic document processing examples for Unstructured.io

# Get the directory where this script is located
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../.." && builtin pwd)}"
SCRIPT_DIR="$APP_ROOT/resources/unstructured-io/examples"
MANAGE_SCRIPT="$APP_ROOT/resources/unstructured-io/manage.sh"

echo "=== Unstructured.io Basic Processing Examples ==="
echo

# Example 1: Simple PDF processing
echo "1. Processing a PDF document"
echo "Command: $MANAGE_SCRIPT --action process --file document.pdf"
echo "This extracts all text and structure from the PDF"
echo

# Example 2: Get markdown output
echo "2. Converting to Markdown for LLM consumption"
echo "Command: $MANAGE_SCRIPT --action process --file report.pdf --output markdown"
echo "Perfect for feeding documents to Ollama or other LLMs"
echo

# Example 3: Extract tables only
echo "3. Extracting tables from a document"
echo 'Command: $MANAGE_SCRIPT --action process --file data.pdf --output json | jq '"'"'.[] | select(.type == "Table")'"'"''
echo "Isolates table data for further processing"
echo

# Example 4: Fast processing for simple documents
echo "4. Fast processing mode"
echo "Command: $MANAGE_SCRIPT --action process --file simple.txt --strategy fast"
echo "Quick extraction for basic text documents"
echo

# Example 5: Multi-language OCR
echo "5. Processing documents with multiple languages"
echo "Command: $MANAGE_SCRIPT --action process --file multilingual.pdf --languages eng,fra,deu"
echo "Supports OCR in multiple languages simultaneously"
echo

# Example 6: Create a test document and process it
echo "6. Live example - creating and processing a test document"
echo

# Create a test HTML document
TEST_FILE="/tmp/test_document.html"
cat > "$TEST_FILE" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Test Document</title>
</head>
<body>
    <h1>Unstructured.io Test Document</h1>
    <p>This is a test document to demonstrate processing capabilities.</p>
    
    <h2>Features</h2>
    <ul>
        <li>Document parsing</li>
        <li>Structure extraction</li>
        <li>Table detection</li>
    </ul>
    
    <h2>Sample Data</h2>
    <table>
        <tr>
            <th>Format</th>
            <th>Supported</th>
            <th>OCR Required</th>
        </tr>
        <tr>
            <td>PDF</td>
            <td>Yes</td>
            <td>Sometimes</td>
        </tr>
        <tr>
            <td>DOCX</td>
            <td>Yes</td>
            <td>No</td>
        </tr>
        <tr>
            <td>Images</td>
            <td>Yes</td>
            <td>Yes</td>
        </tr>
    </table>
    
    <p>This demonstrates how Unstructured.io preserves document structure.</p>
</body>
</html>
EOF

echo "Created test document: $TEST_FILE"
echo
echo "Processing with JSON output:"
echo "$MANAGE_SCRIPT --action process --file $TEST_FILE --output json"
echo
echo "Processing with Markdown output:"
echo "$MANAGE_SCRIPT --action process --file $TEST_FILE --output markdown"
echo

# Example 7: Pipeline example
echo "7. Document processing pipeline"
echo "This example shows how to chain Unstructured.io with other tools:"
echo
cat << 'EOF'
# Step 1: Process document
CONTENT=$($MANAGE_SCRIPT --action process --file report.pdf --output markdown)

# Step 2: Send to Ollama for analysis
echo "$CONTENT" | ollama run llama3.1:8b "Summarize the key points"

# Step 3: Save structured data
$MANAGE_SCRIPT --action process --file report.pdf --output json > report_structured.json

# Step 4: Extract specific elements
cat report_structured.json | jq '.[] | select(.type == "Title" or .type == "Header")'
EOF

echo
echo "=== Additional Resources ==="
echo "- Run '$MANAGE_SCRIPT --help' for all options"
echo "- Check '$SCRIPT_DIR/../README.md' for detailed documentation"
echo "- View logs with '$MANAGE_SCRIPT --action logs'"
echo