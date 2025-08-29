# Qdrant Embeddings - Semantic Knowledge System

A powerful subsystem that indexes Vrooli app knowledge into vector collections, enabling semantic search, pattern discovery, and intelligent code reuse across your entire ecosystem.

## 🚀 Quick Start

```bash
# Initialize embeddings for your app
cd /path/to/your/app
resource-qdrant embeddings init

# Index all content
resource-qdrant embeddings refresh

# Search for knowledge
resource-qdrant embeddings search "email notification"

# Discover patterns across all apps
resource-qdrant embeddings search-all "authentication" --type code
```

## 🎯 Key Features

- **Semantic Search**: Natural language queries across code, docs, and workflows
- **Pattern Discovery**: Identify recurring solutions and best practices
- **App Isolation**: Namespaced collections maintain data separation
- **Smart Refresh**: Git-based change detection for efficient updates
- **Multi-Type Support**: Index workflows, code, documentation, and resources

## 📚 Documentation

- **[Usage Guide](docs/USAGE.md)** - Commands and workflows
- **[Architecture](docs/ARCHITECTURE.md)** - System design and data flow
- **[Capabilities](docs/CAPABILITIES.md)** - Complete feature list
- **[Patterns](docs/PATTERNS.md)** - Best practices and examples
- **[Integration](docs/INTEGRATION.md)** - Connect with other resources
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Performance](docs/PERFORMANCE.md)** - Optimization guide
- **[Contributing](docs/CONTRIBUTING.md)** - Development guidelines

## 🔧 Core Commands

```bash
# Management
resource-qdrant embeddings init       # Initialize app
resource-qdrant embeddings refresh    # Update embeddings
resource-qdrant embeddings status     # Check system status

# Search
resource-qdrant embeddings search "query"      # Search current app
resource-qdrant embeddings search-all "query"  # Search all apps
resource-qdrant embeddings patterns "topic"    # Discover patterns
resource-qdrant embeddings gaps "area"         # Find knowledge gaps
```

## 🏗️ Architecture Overview

```
Your App → Extractors → Ollama (embed) → Qdrant Collections → Search Results
            ↓              ↓                ↓                    ↓
         Content      Vectors[1536]    Namespaced         Semantic Results
```

## 📦 What Gets Indexed

- **Workflows**: N8n automation flows and integrations
- **Code**: Functions, APIs, and implementations
- **Documentation**: Markdown, patterns, and decisions  
- **Resources**: Configurations and capabilities
- **Scenarios**: PRDs and business requirements

## 🚀 Use Cases

1. **Before building**: Search for existing solutions
2. **During development**: Find patterns and examples
3. **Code review**: Discover similar implementations
4. **Knowledge sharing**: Learn from other teams
5. **AI agents**: RAG context for better responses

## 🔗 Requirements

- Qdrant vector database running
- Ollama with `mxbai-embed-large` model
- Git for change tracking
- 2GB+ RAM recommended

## 📈 Status

Production ready, actively used across 30+ Vrooli apps for intelligent knowledge management.

---

*Part of the Vrooli Resource Ecosystem - Making systems smarter through shared knowledge*