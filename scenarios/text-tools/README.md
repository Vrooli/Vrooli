# Text Tools

## Overview

Text Tools is a comprehensive, foundational text processing and analysis platform that provides essential text manipulation capabilities for all Vrooli scenarios. It serves as the central hub for text operations, eliminating duplication and providing consistent, high-quality text processing across the ecosystem.

## Core Capabilities

### 🔍 Text Comparison (Diff)
- Line-by-line, word-by-word, character-level, and semantic comparison
- Configurable ignore options (whitespace, case)
- Similarity scoring and change tracking
- Visual diff representation

### 🔎 Pattern Search
- Plain text, regex, fuzzy, and semantic search
- Case-sensitive and whole-word matching
- Multi-file search support
- Match highlighting and context display

### 🔄 Text Transformation
- Case conversions (upper, lower, title, camel, snake, kebab)
- Encoding/decoding (Base64, URL, HTML entities)
- Format conversions (JSON, XML, YAML)
- Text sanitization and normalization
- Transformation pipelines for complex operations

### 📄 Text Extraction
- Extract from multiple formats (PDF, HTML, Word, images)
- OCR support for image text extraction
- Metadata extraction
- Format auto-detection

### 📊 Text Analysis
- Named entity recognition (emails, URLs, dates, etc.)
- Sentiment analysis
- Keyword extraction
- Language detection
- Text summarization
- Statistical analysis

## Architecture

```
text-tools/
├── api/              # Go API server
├── cli/              # Command-line interface
├── ui/               # Web interface
├── plugins/          # Modular text processing plugins
├── initialization/   # Database schemas and workflows
└── tests/           # Test suites
```

## Quick Start

### Using the CLI

```bash
# Compare two files
text-tools diff file1.txt file2.txt --type line

# Search for patterns
text-tools search "TODO" *.js --regex

# Transform text
text-tools transform input.txt --upper --sanitize

# Extract from documents
text-tools extract document.pdf --ocr

# Analyze text
text-tools analyze article.txt --summary 200 --keywords
```

### Using the API

```bash
# Diff endpoint
curl -X POST http://localhost:14000/api/v1/text/diff \
  -H "Content-Type: application/json" \
  -d '{"text1": "hello", "text2": "world"}'

# Search endpoint
curl -X POST http://localhost:14000/api/v1/text/search \
  -H "Content-Type: application/json" \
  -d '{"text": "sample text", "pattern": "text"}'
```

### Using the Web UI

Access the web interface at `http://localhost:24000` for:
- Interactive diff viewer
- Real-time search with highlighting
- Visual transformation pipeline builder
- Drag-and-drop file extraction
- Comprehensive text analysis dashboard

## Integration with Other Scenarios

Text Tools is designed to be consumed by other scenarios:

```javascript
// From another scenario's code
const response = await fetch('http://localhost:14000/api/v1/text/diff', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
        text1: originalContent,
        text2: modifiedContent,
        options: { type: 'line' }
    })
});
const diffResult = await response.json();
```

## Resource Dependencies

### Required
- **PostgreSQL**: Stores text metadata, operations history, and templates
- **MinIO**: Handles large text files and processing results

### Optional
- **Ollama**: Enables AI-powered features (semantic search, summarization)
- **Redis**: Caches frequently accessed text and results
- **Qdrant**: Vector storage for semantic search
- **N8n**: Workflow automation for complex pipelines

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/text/diff` | POST | Compare two texts |
| `/api/v1/text/search` | POST | Search for patterns |
| `/api/v1/text/transform` | POST | Transform text |
| `/api/v1/text/extract` | POST | Extract from documents |
| `/api/v1/text/analyze` | POST | Perform NLP analysis |
| `/health` | GET | Service health check |

## Performance

- **Response Time**: < 100ms for basic operations, < 500ms for NLP
- **Throughput**: 1000 operations/second
- **File Size**: Supports files up to 100MB
- **Accuracy**: > 95% for entity extraction

## Use Cases

1. **Code Review**: Diff comparison, syntax highlighting
2. **Document Management**: Version tracking, content extraction
3. **Content Moderation**: Sentiment analysis, entity detection
4. **Research**: Summarization, keyword extraction
5. **Data Processing**: Format conversion, sanitization

## Development

```bash
# Install dependencies
cd api && go mod download
cd ../ui && npm install

# Run tests
go test ./...
npm test

# Start development
vrooli scenario run text-tools
```

## Configuration

Environment variables:
- `TEXT_TOOLS_PORT`: API port (default: 14000)
- `TEXT_TOOLS_UI_PORT`: UI port (default: 24000)
- `DATABASE_URL`: PostgreSQL connection string
- `MINIO_URL`: MinIO server URL
- `OLLAMA_URL`: Ollama API URL (optional)
- `REDIS_URL`: Redis connection string (optional)

## Why Text Tools Matters

Text Tools isn't just another utility - it's a **foundational capability** that:
- **Eliminates duplication**: No more reimplementing text processing in every scenario
- **Ensures consistency**: All scenarios use the same high-quality text operations
- **Enables composition**: Scenarios can chain text operations together
- **Grows smarter**: Each improvement benefits all scenarios that use it

## Future Enhancements

- [ ] Machine learning-based text classification
- [ ] Multi-language translation via Ollama
- [ ] Real-time collaborative text editing
- [ ] Advanced template engine with conditionals
- [ ] Text-to-speech and speech-to-text integration
- [ ] Custom plugin development API

## Support

For issues or questions, check:
- API Documentation: http://localhost:24000 (click "API Docs")
- CLI Help: `text-tools help`
- Logs: `docker logs text-tools-api`

---

**Text Tools** - The foundation of text intelligence in Vrooli 🚀