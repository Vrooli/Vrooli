# Unstructured.io Testing Guide

This guide provides comprehensive information for testing the Unstructured.io document processing service.

## 🔍 API Overview

The Unstructured.io service provides a REST API for document processing at `http://localhost:11450`.

### Main Endpoints

1. **Health Check**
   - **Endpoint**: `GET /healthcheck`
   - **Purpose**: Verify service is running
   - **Response**: 200 OK if healthy

2. **Document Processing**
   - **Endpoint**: `POST /general/v0/general`
   - **Purpose**: Process documents and extract structured data
   - **Content-Type**: `multipart/form-data`
   - **Parameters**:
     - `files`: The document file (required)
     - `strategy`: Processing strategy - `fast`, `hi_res`, or `auto` (default: `hi_res`)
     - `languages`: Comma-separated language codes for OCR (default: `eng`)
     - `include_page_breaks`: Include page break elements (default: `true`)
     - `pdf_infer_table_structure`: Enable table detection in PDFs (default: `true`)
     - `encoding`: Text encoding (default: `utf-8`)

## 📁 Supported Formats

### Documents
- **PDF** (.pdf) - With text extraction and OCR
- **Word** (.docx, .doc) - Microsoft Word documents
- **Text** (.txt) - Plain text files
- **RTF** (.rtf) - Rich Text Format
- **OpenDocument** (.odt) - OpenDocument Text
- **Markdown** (.md) - Markdown documents
- **ReStructuredText** (.rst) - RST documentation
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

## 🧪 Testing Methods

### 1. Shell Script Testing

```bash
# Run the comprehensive test suite
./scripts/resources/ai/unstructured-io/test-api.sh

# Test specific functionality
./scripts/resources/ai/unstructured-io/manage.sh --action test
```

### 2. Python Testing

```bash
# Run Python test suite
python3 ./scripts/resources/ai/unstructured-io/test_api.py

# Or with explicit Python path
/usr/bin/python3 ./scripts/resources/ai/unstructured-io/test_api.py
```

### 3. Jest Integration Tests

```bash
# Run from server package directory
cd packages/server
pnpm test UnstructuredIoResource.test.ts

# Run with coverage
pnpm test --coverage UnstructuredIoResource.test.ts
```

### 4. Manual cURL Testing

```bash
# Health check
curl http://localhost:11450/healthcheck

# Process a text file
curl -X POST http://localhost:11450/general/v0/general \
  -F "files=@test.txt" \
  -F "strategy=fast"

# Process with specific options
curl -X POST http://localhost:11450/general/v0/general \
  -F "files=@document.pdf" \
  -F "strategy=hi_res" \
  -F "languages=eng,fra" \
  -F "include_page_breaks=true" \
  -F "pdf_infer_table_structure=true"
```

## 📊 Response Format

The API returns a JSON array of elements, each with:

```json
{
  "type": "Title|Header|NarrativeText|ListItem|Table|PageBreak|...",
  "text": "The extracted text content",
  "metadata": {
    "page_number": 1,
    "filename": "document.pdf",
    "languages": ["eng"],
    // Additional metadata...
  }
}
```

### Element Types

- **Title**: Main document title
- **Header**: Section headers
- **NarrativeText**: Regular paragraph text
- **ListItem**: Bulleted or numbered list items
- **Table**: Table data (preserved structure)
- **PageBreak**: Page boundary markers
- **Footer**: Footer text
- **Image**: Image placeholders
- **Formula**: Mathematical formulas
- **FigureCaption**: Figure/image captions

## 🔧 Test Scenarios

### Basic Functionality
1. Service health check
2. Simple text file processing
3. Multi-page document processing
4. Different output format conversions

### Format-Specific Tests
1. **PDF Tests**
   - Text extraction
   - Table detection
   - OCR for scanned pages
   - Multi-page handling

2. **Office Documents**
   - DOCX with formatting
   - XLSX with multiple sheets
   - PPTX slide extraction

3. **Web Content**
   - HTML with tables
   - Nested HTML structures
   - CSS-styled content

4. **Structured Data**
   - JSON parsing
   - CSV table extraction
   - XML structure preservation

### Strategy Tests
1. **Fast Strategy**
   - Quick text extraction
   - Basic structure detection
   - Suitable for simple documents

2. **Hi-Res Strategy**
   - Advanced layout analysis
   - Table structure inference
   - Better OCR accuracy
   - Slower but more accurate

3. **Auto Strategy**
   - Automatic strategy selection
   - Based on document complexity

### Error Handling
1. Invalid file formats
2. Corrupted files
3. Empty files
4. Oversized files
5. Network timeouts
6. Invalid parameters

### Performance Tests
1. Large file processing (>10MB)
2. Batch processing multiple files
3. Concurrent request handling
4. Memory usage monitoring

## 🐛 Debugging

### Check Service Status

```bash
# Using manage script
./scripts/resources/ai/unstructured-io/manage.sh --action status

# Check Docker container
docker ps | grep unstructured
docker logs vrooli-unstructured-io

# Check port availability
lsof -i :11450
```

### Common Issues

1. **Service Not Available**
   ```bash
   # Start the service
   ./scripts/resources/ai/unstructured-io/manage.sh --action start
   ```

2. **Processing Timeout**
   - Large files may take several minutes
   - Increase timeout in API calls
   - Use `fast` strategy for quicker results

3. **Memory Issues**
   ```bash
   # Check container resources
   docker stats vrooli-unstructured-io
   
   # Increase memory limit
   docker update --memory="8g" vrooli-unstructured-io
   ```

4. **OCR Not Working**
   - Ensure image quality is sufficient
   - Check language codes are correct
   - Use `hi_res` strategy for better OCR

## 📈 Performance Benchmarks

Typical processing times (on standard hardware):

| Document Type | Size | Fast Strategy | Hi-Res Strategy |
|--------------|------|---------------|-----------------|
| Text file | 10KB | <1s | 1-2s |
| PDF (text) | 1MB | 2-5s | 5-15s |
| PDF (scanned) | 5MB | 10-20s | 30-60s |
| DOCX | 500KB | 1-3s | 3-8s |
| Image (OCR) | 2MB | 5-10s | 15-30s |

## 🔗 Integration Examples

### TypeScript/Node.js

```typescript
import axios from 'axios';
import FormData from 'form-data';
import fs from 'fs';

async function processDocument(filePath: string) {
    const form = new FormData();
    form.append('files', fs.createReadStream(filePath));
    form.append('strategy', 'hi_res');
    
    const response = await axios.post(
        'http://localhost:11450/general/v0/general',
        form,
        {
            headers: form.getHeaders(),
            timeout: 60000
        }
    );
    
    return response.data;
}
```

### Python

```python
import requests

def process_document(file_path, strategy='hi_res'):
    with open(file_path, 'rb') as f:
        files = {'files': f}
        data = {
            'strategy': strategy,
            'languages': 'eng',
            'include_page_breaks': 'true'
        }
        
        response = requests.post(
            'http://localhost:11450/general/v0/general',
            files=files,
            data=data,
            timeout=60
        )
        
    return response.json()
```

### Using Vrooli's Resource System

```typescript
// Get the resource
const unstructured = await resourceRegistry.get('unstructured-io');

// Process a document
const result = await unstructured.processDocument(fileBuffer, {
    strategy: 'hi_res',
    outputFormat: 'markdown',
    languages: ['eng', 'fra']
});
```

## 📚 Additional Resources

- [Unstructured.io Documentation](https://docs.unstructured.io/)
- [API Reference](https://api.unstructured.io/general/docs)
- [Supported File Types](https://docs.unstructured.io/api-reference/api-services/supported-file-types)
- [Processing Strategies](https://docs.unstructured.io/api-reference/api-services/processing-strategies)

## 🎯 Test Checklist

- [ ] Service health check passes
- [ ] Basic text file processing works
- [ ] PDF processing extracts text correctly
- [ ] Table extraction works for structured documents
- [ ] OCR works for image files
- [ ] Different strategies produce expected results
- [ ] Error handling works for invalid inputs
- [ ] Performance is acceptable for typical documents
- [ ] Integration with Vrooli resource system works
- [ ] Memory usage stays within limits