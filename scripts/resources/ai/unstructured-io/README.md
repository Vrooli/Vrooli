# Unstructured.io Document Processing Resource

Transform any document into AI-ready structured data. Unstructured.io provides enterprise-grade document processing that extracts text, tables, images, and metadata from PDFs, Word docs, PowerPoints, spreadsheets, emails, and more.

## 🚀 Quick Start

```bash
# Install Unstructured.io
./scripts/resources/ai/unstructured-io/manage.sh --action install

# Process a document
./scripts/resources/ai/unstructured-io/manage.sh --action process --file document.pdf

# Check service status
./scripts/resources/ai/unstructured-io/manage.sh --action status
```

## 📋 Overview

Unstructured.io is a comprehensive document processing platform that converts complex documents into structured data optimized for AI consumption. It combines layout analysis, OCR, table extraction, and intelligent chunking to make any document accessible to LLMs and vector databases.

### Key Features
- **Universal Format Support**: 20+ document formats including PDF, DOCX, PPTX, XLSX, images, emails
- **Advanced Processing**: Layout detection, table preservation, image analysis, metadata extraction
- **AI-Ready Output**: Structured JSON, Markdown, or plain text optimized for LLM processing
- **Processing Strategies**: Fast mode for quick extraction, Hi-Res mode for complex documents
- **Local Deployment**: Process sensitive documents without cloud dependencies

## 🛠️ Installation

### Prerequisites
- Docker installed and running
- 4GB RAM minimum (8GB recommended)
- 10GB disk space for Docker image and processing

### Install Command
```bash
# Basic installation
./scripts/resources/ai/unstructured-io/manage.sh --action install

# Force reinstall
./scripts/resources/ai/unstructured-io/manage.sh --action install --force yes
```

The installation will:
1. Pull the Unstructured.io Docker image
2. Create a container with proper resource limits
3. Start the service on port 11450
4. Verify API health

## 💡 Usage

### Basic Document Processing

```bash
# Process a PDF with default settings
./manage.sh --action process --file report.pdf

# Process with specific strategy
./manage.sh --action process --file report.pdf --strategy hi_res

# Get markdown output
./manage.sh --action process --file report.pdf --output markdown

# Process with specific OCR languages
./manage.sh --action process --file document.pdf --languages eng,fra,deu
```

### Processing Strategies

1. **fast** - Quick processing with basic extraction
   ```bash
   ./manage.sh --action process --file doc.pdf --strategy fast
   ```

2. **hi_res** - High resolution processing with advanced features (default)
   ```bash
   ./manage.sh --action process --file doc.pdf --strategy hi_res
   ```

3. **auto** - Automatically select best strategy
   ```bash
   ./manage.sh --action process --file doc.pdf --strategy auto
   ```

### Output Formats

1. **JSON** - Structured elements with metadata (default)
   ```bash
   ./manage.sh --action process --file doc.pdf --output json
   ```

2. **Markdown** - Formatted for LLM consumption
   ```bash
   ./manage.sh --action process --file doc.pdf --output markdown
   ```

3. **Text** - Plain text extraction
   ```bash
   ./manage.sh --action process --file doc.pdf --output text
   ```

4. **Elements** - Detailed element breakdown
   ```bash
   ./manage.sh --action process --file doc.pdf --output elements
   ```

### Batch Processing

```bash
# Process multiple files
./manage.sh --action process --file "file1.pdf,file2.docx,file3.pptx" --batch yes

# Process with options
./manage.sh --action process --file "*.pdf" --batch yes --strategy fast --output markdown
```

## 📁 Supported Formats

### Documents
- **PDF** (.pdf) - Portable Document Format
- **Word** (.docx, .doc) - Microsoft Word documents
- **Text** (.txt) - Plain text files
- **RTF** (.rtf) - Rich Text Format
- **OpenDocument** (.odt) - OpenDocument Text
- **Markdown** (.md) - Markdown documents
- **HTML** (.html) - Web pages
- **XML** (.xml) - XML documents
- **EPUB** (.epub) - E-book format

### Spreadsheets
- **Excel** (.xlsx, .xls) - Microsoft Excel

### Presentations
- **PowerPoint** (.pptx, .ppt) - Microsoft PowerPoint

### Images (with OCR)
- **PNG** (.png)
- **JPEG** (.jpg, .jpeg)
- **TIFF** (.tiff)
- **BMP** (.bmp)
- **HEIC** (.heic) - Apple image format

### Email
- **EML** (.eml) - Email messages
- **MSG** (.msg) - Outlook messages

## 🔧 Management Commands

### Service Lifecycle

```bash
# Start service
./manage.sh --action start

# Stop service
./manage.sh --action stop

# Restart service
./manage.sh --action restart

# Uninstall service
./manage.sh --action uninstall
```

### Monitoring

```bash
# Check status
./manage.sh --action status

# View service info
./manage.sh --action info

# View logs
./manage.sh --action logs

# Follow logs in real-time
./manage.sh --action logs --follow yes
```

## 🔗 Integration Examples

### With Ollama (Document Q&A)

```bash
# 1. Process document to markdown
CONTENT=$(./manage.sh --action process --file report.pdf --output markdown)

# 2. Send to Ollama for analysis
echo "$CONTENT" | ollama run llama3.1:8b "Summarize this document"
```

### With n8n (Automated Workflows)

Create an n8n workflow that:
1. Monitors a folder for new documents
2. Sends documents to Unstructured.io for processing
3. Stores structured data in a database
4. Triggers notifications or further processing

### With MinIO (Document Archive)

```bash
# Process and store in MinIO
./manage.sh --action process --file document.pdf --output json > processed.json
mc cp processed.json minio/documents/
```

### Python Integration

```python
import requests

# Process document
with open('document.pdf', 'rb') as f:
    response = requests.post(
        'http://localhost:11450/general/v0/general',
        files={'files': f},
        data={
            'strategy': 'hi_res',
            'languages': 'eng'
        }
    )

# Get structured data
elements = response.json()
```

### TypeScript Integration

```typescript
// Use with Vrooli's resource system
const unstructured = await resourceRegistry.get('unstructured-io');
const result = await unstructured.processDocument(fileBuffer, {
    strategy: 'hi_res',
    outputFormat: 'markdown'
});
```

## ⚙️ Configuration

### Environment Variables

The service can be configured through environment variables:

```bash
# API configuration
UNSTRUCTURED_API_LOG_LEVEL=INFO
UNSTRUCTURED_API_MAX_REQUEST_SIZE=52428800  # 50MB
UNSTRUCTURED_API_WORKERS=1

# Processing limits
UNSTRUCTURED_IO_MAX_FILE_SIZE=50MB
UNSTRUCTURED_IO_TIMEOUT_SECONDS=300
```

### Resource Configuration

Edit `~/.vrooli/resources.local.json`:

```json
{
  "services": {
    "ai": {
      "unstructured-io": {
        "enabled": true,
        "baseUrl": "http://localhost:11450",
        "processing": {
          "defaultStrategy": "hi_res",
          "ocrLanguages": ["eng", "fra", "deu"],
          "chunkingEnabled": true,
          "maxChunkSize": 2000
        },
        "healthCheck": {
          "endpoint": "/healthcheck",
          "intervalMs": 60000,
          "timeoutMs": 5000
        }
      }
    }
  }
}
```

## 🎯 Use Cases

### 1. Invoice Processing
Extract structured data from PDF invoices:
```bash
./manage.sh --action process --file invoice.pdf --output json | \
  jq '.[] | select(.type == "Table")'
```

### 2. Contract Analysis
Convert legal documents to markdown for LLM analysis:
```bash
./manage.sh --action process --file contract.pdf --output markdown > contract.md
ollama run llama3.1:8b "Identify key terms and obligations in this contract" < contract.md
```

### 3. Email Attachment Processing
Process email attachments automatically:
```bash
# Extract and process all PDFs from email
for file in *.pdf; do
  ./manage.sh --action process --file "$file" --output json > "${file%.pdf}.json"
done
```

### 4. Knowledge Base Creation
Convert company documents to vector-ready chunks:
```bash
./manage.sh --action process --file handbook.pdf --output json | \
  jq -r '.[] | {text: .text, metadata: .metadata}'
```

## 🚨 Troubleshooting

### Common Issues

1. **Container won't start**
   ```bash
   # Check Docker is running
   docker info
   
   # Check port availability
   sudo lsof -i :11450
   ```

2. **Processing timeout**
   - Increase timeout for large files
   - Use `fast` strategy for quicker processing
   - Split large documents if possible

3. **Out of memory**
   ```bash
   # Check container resources
   docker stats vrooli-unstructured-io
   
   # Increase memory limit if needed
   docker update --memory="8g" vrooli-unstructured-io
   ```

4. **OCR not working**
   - Ensure image files are not corrupted
   - Check language codes are correct
   - Use hi_res strategy for better OCR

### Debug Commands

```bash
# Check container logs
docker logs vrooli-unstructured-io --tail 100

# Test with simple file
echo "Test document" > test.txt
./manage.sh --action process --file test.txt

# Check API directly
curl http://localhost:11450/healthcheck
```

## 📊 Performance Tips

1. **Strategy Selection**
   - Use `fast` for simple text extraction
   - Use `hi_res` for complex layouts, tables, or images
   - Use `auto` when unsure

2. **Batch Processing**
   - Process multiple files together for efficiency
   - Use consistent strategies within batches

3. **Resource Optimization**
   - Monitor memory usage with `docker stats`
   - Adjust container limits based on workload
   - Process large documents during off-peak times

## 🔒 Security Considerations

- All processing happens locally - no data sent to cloud
- Container runs with limited resources and no network access to external services
- Temporary files are automatically cleaned up
- Supports air-gapped environments

## 🆘 Getting Help

- **Logs**: `./manage.sh --action logs`
- **Status**: `./manage.sh --action status`
- **Info**: `./manage.sh --action info`
- **Unstructured.io Docs**: https://docs.unstructured.io/

---

**Note**: Unstructured.io is a powerful document processing engine that enables Vrooli's AI tiers to understand and process any document format, making it essential for business automation workflows.