# Secure Document Processing

## Overview

Enterprise-grade secure document processing pipeline with encryption, compliance, and semantic search capabilities.

## Key Features
- Secure document intake and encryption using Vault
- Automated processing workflows handled in the Go API
- Semantic search powered by Qdrant and Ollama
- Compliance auditing and metadata management with PostgreSQL
- Object storage with MinIO for encrypted documents
- Node.js UI for document management and monitoring
- Go API for coordination and integration

## Architecture
- **API**: Go-based coordination server
- **UI**: Node.js web interface
- **Automation**: In-API orchestration for intake, processing, and indexing
- **AI**: Ollama for embeddings and semantic analysis
- **Storage**: PostgreSQL (metadata), MinIO (files), Qdrant (vectors)
- **Security**: Vault for key management and encryption

## Running the Scenario
```bash
make run  # or vrooli scenario run secure-document-processing
```

Access:
- UI: http://localhost:${UI_PORT}
- API: http://localhost:${API_PORT}

## Testing
```bash
make test  # Runs comprehensive test suite
```

## Development
```bash
make dev   # Start in development mode with hot reload
```

For detailed setup and configuration, see PRD.md and IMPLEMENTATION_PLAN.md.
