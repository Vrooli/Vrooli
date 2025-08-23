# Haystack Resource

End-to-end framework for question answering and semantic search.

## Overview

Haystack provides sophisticated question-answering capabilities over documents and data. It integrates with existing document processing and vector search resources to enable RAG (Retrieval-Augmented Generation) and knowledge base scenarios.

## Features

- **Document Indexing**: Index text documents with automatic chunking and embedding
- **Semantic Search**: Query documents using natural language
- **File Upload**: Direct file upload and indexing support
- **Python Script Execution**: Run custom Python scripts in the Haystack environment
- **REST API**: Full-featured API for integration with other services

## Usage

### Start the service
```bash
vrooli resource haystack start
# or
resource-haystack start
```

### Check status
```bash
vrooli resource haystack status
# JSON format
vrooli resource haystack status --format json
```

### Index documents
```bash
# Index JSON documents
resource-haystack inject documents.json

# Upload and index a text file
resource-haystack inject article.txt

# Execute a Python script
resource-haystack inject process_data.py
```

### Query documents
Use the REST API:
```bash
curl -X POST http://localhost:8075/query \
  -H "Content-Type: application/json" \
  -d '{"query": "What is machine learning?", "top_k": 5}'
```

### Clear data
```bash
resource-haystack clear
```

## API Endpoints

- `GET /health` - Health check
- `POST /index` - Index documents
- `POST /query` - Query documents
- `POST /upload` - Upload and index file
- `GET /stats` - Get statistics
- `DELETE /clear` - Clear all data

## Data Formats

### JSON Document Format
```json
{
  "documents": [
    {
      "content": "Document text content",
      "metadata": {
        "source": "file.txt",
        "author": "John Doe"
      }
    }
  ]
}
```

### Python Script Guidelines
Scripts have access to:
- Full Haystack framework
- Transformers and sentence-transformers
- PyTorch
- The document store via environment variables

Example script:
```python
from haystack import Document
from haystack.document_stores.in_memory import InMemoryDocumentStore

# Your custom processing logic here
store = InMemoryDocumentStore()
doc = Document(content="Custom content")
store.write_documents([doc])
```

## Integration with Vrooli

Haystack integrates seamlessly with other Vrooli resources:
- Use with **Ollama** for local language models
- Combine with **unstructured-io** for document parsing
- Store embeddings in **Qdrant** for scalable vector search
- Orchestrate with **n8n** or **Huginn** for automated workflows

## Port Configuration

Default port: 8075 (configured via port-registry)