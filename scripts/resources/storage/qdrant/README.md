# Qdrant - High-Performance Vector Database

Qdrant is a vector similarity search engine with extended filtering support, designed for high-performance semantic search and AI applications. This resource provides automated installation, configuration, and management of Qdrant with comprehensive collection management for the Vrooli project.

## 🎯 Quick Reference

- **Category**: Storage
- **Ports**: 6333 (REST API), 6334 (gRPC API)
- **Container**: qdrant
- **API Docs**: [Complete API Reference](docs/API.md)
- **Status**: Production Ready

## 🚀 Quick Start

### Prerequisites
- Docker installed and running
- 2GB+ RAM available for optimal performance
- Ports 6333 and 6334 available
- 5GB+ disk space for vector storage

### Installation
```bash
# Install with default settings
./manage.sh --action install

# Install with custom ports
QDRANT_CUSTOM_PORT=7333 QDRANT_CUSTOM_GRPC_PORT=7334 ./manage.sh --action install

# Install with API key authentication (recommended for production)
QDRANT_CUSTOM_API_KEY=your-secure-key ./manage.sh --action install

# Force reinstall with custom settings
./manage.sh --action install --force yes
```

### Basic Usage
```bash
# Check service status with comprehensive information  
./manage.sh --action status

# Test all functionality including vector operations
./manage.sh --action test

# List all collections
./manage.sh --action list-collections

# Create a new collection for AI embeddings
./manage.sh --action create-collection --collection ai_embeddings --vector-size 1536 --distance Cosine

# Get collection information
./manage.sh --action collection-info --collection ai_embeddings

# Create backup of all collections
./manage.sh --action backup --snapshot-name daily-backup

# View service logs
./manage.sh --action logs
```

### Verify Installation
```bash
# Check service health and functionality
./manage.sh --action status

# Test vector operations
./manage.sh --action test

# Access web interface: http://localhost:6333/dashboard
# REST API base URL: http://localhost:6333/
# gRPC endpoint: grpc://localhost:6334/
```

## 🔧 Core Features

- **⚡ Vector Similarity Search**: Sub-millisecond semantic search with HNSW indexing
- **🎯 Semantic Search**: Find similar content using AI embeddings (OpenAI, HuggingFace, etc.)
- **🔍 Payload Filtering**: Filter search results by metadata without performance loss
- **📊 Real-time Operations**: Insert, update, delete vectors without service interruption
- **🗂️ Collection Management**: Organize vectors into separate searchable collections
- **💾 Persistent Storage**: Durable vector storage with backup and recovery
- **🌐 Dual API**: Both REST and gRPC APIs for different integration needs
- **📈 Performance Optimization**: Quantization and indexing for memory efficiency

## 📖 Documentation

- **[API Reference](docs/API.md)** - REST API, gRPC endpoints, and vector operations  
- **[Setup & Troubleshooting](docs/SETUP_AND_TROUBLESHOOTING.md)** - Complete configuration and troubleshooting guide

## 🎯 When to Use Qdrant

### Use Qdrant When:
- Building semantic search engines and similarity matching
- Creating AI-powered recommendation systems
- Implementing RAG (Retrieval Augmented Generation) pipelines
- Need real-time vector similarity search
- Building content discovery and clustering systems
- Require sub-millisecond query performance

### Consider Alternatives When:
- Need traditional relational data → [PostgreSQL](../postgres/)
- Want document-oriented storage → [MongoDB](../mongodb/)
- Building simple key-value cache → [Redis](../redis/)
- Need graph database relationships → [Neo4j](../neo4j/)

## 🔗 Integration Examples

### With AI Models (Ollama)
```bash
# Create collection for AI agent memory
./manage.sh --action create-collection --collection agent_memory --vector-size 1536 --distance Cosine

# Use with Ollama for semantic search
# 1. Generate embeddings with Ollama
# 2. Store vectors in Qdrant
# 3. Search for similar content
```

### With Document Processing (Unstructured.io)
```bash
# Create collection for document chunks
./manage.sh --action create-collection --collection document_chunks --vector-size 1536 --distance Cosine

# Process documents with Unstructured.io and store embeddings in Qdrant
# Enable semantic document search and RAG systems
```

### With Workflow Automation (n8n)
```bash
# Create collections for automated data processing
./manage.sh --action create-collection --collection user_profiles --vector-size 768 --distance Dot

# Use n8n workflows to:
# - Process incoming data
# - Generate embeddings
# - Store in Qdrant
# - Trigger similarity searches
```

## 🏗️ Architecture

### Vector Storage Model
```
Collection (e.g., "documents")
├── Vectors (dense numerical arrays)
├── Payloads (metadata as JSON)
├── Indexes (HNSW for fast search)
└── Configuration (distance metric, optimization settings)
```

### Default Collections
Qdrant automatically creates these collections for Vrooli:
- **agent_memory** (1536D, Cosine) - AI agent conversation memory
- **code_embeddings** (768D, Dot) - Source code semantic embeddings  
- **document_chunks** (1536D, Cosine) - Document chunks for RAG
- **conversation_history** (1536D, Cosine) - Chat history storage

### Performance Characteristics
- **Search Speed**: Sub-millisecond queries (p50 < 1ms)
- **Throughput**: 10K+ queries/second, 5K+ inserts/second
- **Memory Efficiency**: 4x-16x compression with quantization
- **Accuracy**: >95% recall@100, >98% precision

## 🔒 Security & Authentication

### API Key Authentication
```bash
# Set API key during installation
QDRANT_CUSTOM_API_KEY=your-secure-key ./manage.sh --action install

# All API requests require the key in headers:
# api-key: your-secure-key
```

### Data Protection
- Persistent storage in `~/.qdrant/data`
- Automated backup and recovery system
- Configurable data retention policies

## 🚀 Advanced Usage

### Collection Management
```bash
# Create specialized collections
./manage.sh --action create-collection --collection product_embeddings --vector-size 512 --distance Euclid

# Get detailed collection statistics
./manage.sh --action collection-info --collection product_embeddings

# Monitor index performance
./manage.sh --action index-stats --collection product_embeddings
```

### Backup & Recovery
```bash
# Create named backup
./manage.sh --action backup --snapshot-name pre-migration-2024

# List available backups
./manage.sh --action list-backups

# Get backup information
./manage.sh --action backup-info --snapshot-name pre-migration-2024

# Restore from backup
./manage.sh --action restore --snapshot-name pre-migration-2024
```

### Monitoring & Diagnostics
```bash
# Continuous health monitoring
./manage.sh --action monitor --interval 10

# Comprehensive diagnostics
./manage.sh --action diagnose

# Performance statistics for all collections
./manage.sh --action index-stats
```

## 💡 Common Use Cases

### 1. Semantic Search Engine
```bash
# Setup
./manage.sh --action create-collection --collection search_corpus --vector-size 1536

# Use case: Users search using natural language
# - Convert user query to embeddings
# - Search Qdrant for similar content
# - Return ranked results with metadata
```

### 2. Recommendation System
```bash
# Setup
./manage.sh --action create-collection --collection user_preferences --vector-size 768

# Use case: Content recommendations
# - Store user behavior as vectors
# - Find similar users or content
# - Generate personalized recommendations
```

### 3. RAG (Retrieval Augmented Generation)
```bash
# Setup
./manage.sh --action create-collection --collection knowledge_base --vector-size 1536

# Use case: AI chatbot with knowledge base
# - Store document chunks as vectors
# - Retrieve relevant context for queries
# - Augment LLM responses with retrieved data
```

## 🔧 Configuration

For detailed configuration options, performance tuning, and troubleshooting, see [Setup & Troubleshooting Guide](docs/SETUP_AND_TROUBLESHOOTING.md).

## 🔄 Integration with Vrooli Scenarios

Qdrant enhances these Vrooli scenarios:
- **AI Research Assistant**: Semantic search over research papers
- **Document Analysis Platform**: Find similar documents and content
- **Knowledge Base System**: Natural language querying
- **Content Recommendation**: Discover related content automatically
- **Multi-modal Search**: Combine text, image, and audio embeddings

## 📊 API Quick Reference

### Collections
```bash
# REST API examples
GET    /collections                    # List collections
POST   /collections/{name}             # Create collection
GET    /collections/{name}             # Get collection info
DELETE /collections/{name}             # Delete collection
```

### Vector Operations
```bash
# Insert vectors
PUT    /collections/{name}/points      # Upsert vectors

# Search vectors
POST   /collections/{name}/points/search    # Similarity search
POST   /collections/{name}/points/recommend # Recommendations

# Manage vectors
GET    /collections/{name}/points/{id}      # Get vector by ID
DELETE /collections/{name}/points/{id}      # Delete vector
```

For complete API documentation, see [docs/API.md](docs/API.md).

## 🔍 Testing & Examples

### Run Tests
```bash
# Comprehensive functionality test
./manage.sh --action test

# Test specific collection operations
./test/integration-test.sh
```

### Example Collection Setup
```bash
# AI/ML workflow setup
./manage.sh --action create-collection --collection embeddings --vector-size 1536 --distance Cosine
./manage.sh --action create-collection --collection features --vector-size 512 --distance Dot

# Verify setup
./manage.sh --action list-collections
./manage.sh --action collection-info --collection embeddings
```

---

**🎯 Ready to build powerful semantic search and AI applications with Qdrant!**

For advanced configuration, troubleshooting, and detailed API usage, see [Setup & Troubleshooting](docs/SETUP_AND_TROUBLESHOOTING.md) and [API Reference](docs/API.md).