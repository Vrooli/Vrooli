# Haystack Resource

End-to-end framework for question answering and semantic search.

## Overview

Haystack provides sophisticated question-answering capabilities over documents and data. It integrates with existing document processing and vector search resources to enable RAG (Retrieval-Augmented Generation) and knowledge base scenarios.

## Features

- **Document Indexing**: Index text documents with automatic chunking and embedding
- **Batch Processing**: Efficiently index large document sets in parallel (4 workers)
- **Semantic Search**: Query documents using natural language and vector similarity
- **LLM-Enhanced Search**: Use Ollama models for query expansion and answer generation
- **Custom Pipelines**: Create and run custom Haystack processing pipelines dynamically
- **File Upload**: Direct file upload and indexing support for text files
- **Multi-Language Support**: Automatic language detection for 55+ languages
- **Vector Storage**: Qdrant integration with automatic fallback to in-memory store
- **Python Script Execution**: Run custom Python scripts in the Haystack environment
- **REST API**: Full-featured FastAPI service with comprehensive endpoints
- **Performance Metrics**: Built-in metrics tracking and monitoring via /stats endpoint

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
resource-haystack content add documents.json

# Upload and index a text file
resource-haystack content add article.txt

# Execute a Python script
resource-haystack content add process_data.py
```

### Query documents
Standard query:
```bash
curl -X POST http://localhost:8075/query \
  -H "Content-Type: application/json" \
  -d '{"query": "What is machine learning?", "top_k": 5}'
```

Enhanced query with LLM:
```bash
curl -X POST http://localhost:8075/enhanced_query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What is machine learning?",
    "use_llm": true,
    "generate_answer": true,
    "top_k": 3
  }'
```

### Batch indexing
Index multiple document sets efficiently:
```bash
curl -X POST http://localhost:8075/batch_index \
  -H "Content-Type: application/json" \
  -d '{
    "documents": [
      [{"content": "Doc 1", "metadata": {"batch": 1}}],
      [{"content": "Doc 2", "metadata": {"batch": 2}}]
    ],
    "batch_size": 10
  }'
```

### Custom pipelines
Create a custom processing pipeline:
```bash
curl -X POST http://localhost:8075/custom_pipeline \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my_pipeline",
    "components": [
      {"type": "cleaner", "name": "clean", "params": {}},
      {"type": "splitter", "name": "split", "params": {"split_length": 100}},
      {"type": "embedder", "name": "embed", "params": {}},
      {"type": "writer", "name": "write", "params": {}}
    ],
    "connections": [
      {"from": "clean", "to": "split"},
      {"from": "split", "to": "embed"},
      {"from": "embed", "to": "write"}
    ]
  }'
```

### Clear data
```bash
resource-haystack clear
```

## API Endpoints

- `GET /health` - Health check with feature status
- `POST /index` - Index documents
- `POST /batch_index` - Batch index multiple document sets
- `POST /query` - Query documents  
- `POST /enhanced_query` - Query with LLM enhancement
- `POST /upload` - Upload and index file
- `POST /custom_pipeline` - Create custom processing pipeline
- `POST /run_pipeline/{name}` - Execute custom pipeline
- `GET /pipelines` - List registered pipelines
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