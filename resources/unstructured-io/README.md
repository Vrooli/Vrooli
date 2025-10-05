# Unstructured.io Document Processing Resource

Transform any document into AI-ready structured data. Unstructured.io provides enterprise-grade document processing that extracts text, tables, images, and metadata from PDFs, Word docs, PowerPoints, spreadsheets, emails, and more.

## 🚀 Quick Start

```bash
# Install Unstructured.io
vrooli resource unstructured-io manage install

# Process a document
vrooli resource unstructured-io content process document.pdf

# Check service status
vrooli resource unstructured-io status
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
vrooli resource unstructured-io manage install

# Force reinstall
vrooli resource unstructured-io manage install --force
```

### Docker Compose Deployment
For standardized deployments, you can use the provided docker-compose.yml:
```bash
# Start service with docker-compose
cd resources/unstructured-io
docker compose up -d

# Stop service
docker compose down

# View logs
docker compose logs -f unstructured-io
```

The deployment includes:
- Resource limits (4GB memory, 2.0 CPU)
- Health checks with automatic restart
- Proper network configuration
- Persistent service management
- Pinned Docker image version (0.0.78) for reproducible deployments

### Installation Process
The installation will:
1. Pull the Unstructured.io Docker image
2. Create a container with proper resource limits
3. Start the service on port 11450
4. Verify API health

## 💡 Usage

### Basic Document Processing

```bash
# Process a PDF with default settings
vrooli resource unstructured-io content process report.pdf

# Process with specific strategy
vrooli resource unstructured-io content process report.pdf --strategy hi_res

# Get markdown output
vrooli resource unstructured-io content process report.pdf --output markdown

# Process with specific OCR languages
vrooli resource unstructured-io content process document.pdf --languages eng,fra,deu
```

### Caching

Unstructured.io automatically caches processed documents to improve performance:

```bash
# Clear all cache
vrooli resource unstructured-io content clear-cache
```

Cache features:
- Automatic caching of processed documents (1 hour TTL by default)
- File-based cache stored in `~/.vrooli/cache/unstructured-io/`
- Cache key includes file hash, strategy, output format, and languages
- Cached results show "📦 Using cached result" message
- Set `UNSTRUCTURED_IO_CACHE_ENABLED=no` to disable caching

### Processing Strategies

1. **fast** - Quick processing with basic extraction
   ```bash
   vrooli resource unstructured-io content process doc.pdf --strategy fast
   ```

2. **hi_res** - High resolution processing with advanced features (default)
   ```bash
   vrooli resource unstructured-io content process doc.pdf --strategy hi_res
   ```

3. **auto** - Automatically select best strategy
   ```bash
   vrooli resource unstructured-io content process doc.pdf --strategy auto
   ```

### Output Formats

1. **JSON** - Structured elements with metadata (default)
   ```bash
   vrooli resource unstructured-io content process doc.pdf --output json
   ```

2. **Markdown** - Formatted for LLM consumption
   ```bash
   vrooli resource unstructured-io content process doc.pdf --output markdown
   ```

3. **Text** - Plain text extraction
   ```bash
   vrooli resource unstructured-io content process doc.pdf --output text
   ```

4. **Elements** - Detailed element breakdown
   ```bash
   vrooli resource unstructured-io content process doc.pdf --output elements
   ```

### Batch Processing

```bash
# Process directory of files
vrooli resource unstructured-io content process-directory ./documents --strategy fast --output markdown
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
vrooli resource unstructured-io manage start

# Stop service
vrooli resource unstructured-io manage stop

# Restart service
vrooli resource unstructured-io manage restart

# Uninstall service
vrooli resource unstructured-io manage uninstall
```

### Monitoring

```bash
# Check status
vrooli resource unstructured-io status

# View service info
vrooli resource unstructured-io info

# View logs
vrooli resource unstructured-io logs

# Follow logs in real-time
vrooli resource unstructured-io logs --follow
```

## 🔗 Integration Examples

### Quick Start with Integration Scripts

Ready-to-use integration scripts are available in the `integrations/` directory:

```bash
# Document Q&A with Ollama
./integrations/doc-qa.sh report.pdf "What are the key findings?"

# Archive to MinIO with metadata
./integrations/doc-to-minio.sh contract.pdf legal-documents

# Batch process multiple files
./integrations/batch-process.sh -o markdown -d ./summaries /path/to/documents
```

See `examples/integration-examples.sh` for more detailed examples and workflows.

### With Ollama (Document Q&A)

```bash
# 1. Process document to markdown
CONTENT=$(vrooli resource unstructured-io content process report.pdf --output markdown)

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
vrooli resource unstructured-io content process document.pdf --output json > processed.json
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

Edit `~/.vrooli/service.json`:

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
vrooli resource unstructured-io content process invoice.pdf --output json | \
  jq '.[] | select(.type == "Table")'
```

### 2. Contract Analysis
Convert legal documents to markdown for LLM analysis:
```bash
vrooli resource unstructured-io content process contract.pdf --output markdown > contract.md
ollama run llama3.1:8b "Identify key terms and obligations in this contract" < contract.md
```

### 3. Email Attachment Processing
Process email attachments automatically:
```bash
# Extract and process all PDFs from email
for file in *.pdf; do
  vrooli resource unstructured-io content process "$file" --output json > "${file%.pdf}.json"
done
```

### 4. Knowledge Base Creation
Convert company documents to vector-ready chunks:
```bash
vrooli resource unstructured-io content process handbook.pdf --output json | \
  jq -r '.[] | {text: .text, metadata: .metadata}'
```

## 🚨 Troubleshooting

For comprehensive troubleshooting, see the **[Troubleshooting Guide](TROUBLESHOOTING.md)**.

### Quick Fixes

1. **Check service health**
   ```bash
   vrooli resource unstructured-io status
   vrooli resource unstructured-io logs --tail 50
   ```

2. **Common errors**
   - `[ERROR:CONNECTION]` - Service not running, use `vrooli resource unstructured-io manage start`
   - `[ERROR:TIMEOUT]` - Use `--strategy fast` or process smaller files
   - `[ERROR:FILE_TOO_LARGE]` - Split file or increase limits
   - `[ERROR:UNSUPPORTED_TYPE]` - Check supported formats with `vrooli resource unstructured-io info`

3. **Reset and reinstall**
   ```bash
   vrooli resource unstructured-io manage uninstall
   vrooli resource unstructured-io manage install --force
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

- **Logs**: `vrooli resource unstructured-io logs`
- **Status**: `vrooli resource unstructured-io status`
- **Info**: `vrooli resource unstructured-io info`
- **Help**: `vrooli resource unstructured-io help`
- **Run Tests**: `vrooli resource unstructured-io test all`
- **Unstructured.io Docs**: https://docs.unstructured.io/

---

**Note**: Unstructured.io is a powerful document processing engine that enables Vrooli's AI tiers to understand and process any document format, making it essential for business automation workflows.