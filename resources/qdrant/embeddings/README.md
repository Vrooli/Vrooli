# Qdrant Semantic Knowledge System

A powerful embedding system that indexes project knowledge into Qdrant vector collections, enabling semantic search and pattern discovery across all Vrooli apps.

## 🚀 Quick Start

```bash
# Initialize embeddings for your project
cd /path/to/your/app
source resources/qdrant/embeddings/manage.sh
qdrant::embeddings::init

# Refresh embeddings (auto-detects changes)
qdrant::embeddings::refresh

# Search your project
qdrant::embeddings::search "how to send emails"

# Search all apps
qdrant::search::all_apps "webhook processing" workflows
```

## 🎯 Features

### Core Capabilities
- **App Isolation**: Each app has namespaced collections (`{app-id}-{type}`)
- **Git-Triggered Refresh**: Automatically detects when re-indexing is needed
- **Comprehensive Extraction**: Captures workflows, scenarios, documentation, code, and resources
- **Semantic Search**: Natural language queries find relevant content
- **Cross-App Discovery**: Find patterns and solutions across entire ecosystem
- **Smart Validation**: Shows what's embeddable vs missing with recommendations

### Search Features
- **Single App Search**: Find content within specific app
- **Multi-App Search**: Discover patterns across all apps
- **Pattern Discovery**: Identify recurring solutions
- **Solution Finding**: Locate reusable implementations
- **Gap Analysis**: Find missing knowledge areas
- **Interactive Explorer**: Browse and discover interactively

## 📦 What Gets Embedded

### 1. N8n Workflows
- Workflow names and descriptions
- Node types and integrations
- Webhook paths and triggers
- Sticky note comments
- Purpose analysis based on nodes

### 2. Scenarios
- PRD content (executive summary, value proposition, features)
- Scenario configurations (.scenario.yaml)
- Target users and success metrics
- Revenue models and technical requirements

### 3. Documentation
- Architecture decisions and patterns
- Security principles and vulnerabilities
- Lessons learned (what worked/failed)
- Breaking changes history
- Performance optimizations
- Code patterns and best practices

### 4. Code
- Function signatures and descriptions
- API endpoints and routes
- CLI commands
- SQL queries
- Design patterns (Singleton, Factory, Observer, etc.)

### 5. Resources
- Resource capabilities and commands
- Configuration details
- Dependencies and integrations
- API documentation
- Docker configurations

## 🔧 Commands

### Management Commands

```bash
# Initialize app identity
qdrant::embeddings::init [app-id]

# Refresh all embeddings (with progress tracking)
qdrant::embeddings::refresh [app-id] [--force]

# Validate setup and coverage
qdrant::embeddings::validate [directory]

# Show status of all apps
qdrant::embeddings::status

# Garbage collect orphaned collections
qdrant::embeddings::gc [--force]
```

### Search Commands

```bash
# Search current app
qdrant::search::single_app "query" "app-id" [type] [limit] [min_score]

# Search all apps
qdrant::search::all_apps "query" [type] [limit_per_app] [min_score]

# Discover patterns
qdrant::search::discover_patterns "query" [threshold]

# Find solutions
qdrant::search::find_solutions "problem description" [preferred_type]

# Analyze knowledge gaps
qdrant::search::find_gaps "topic"

# Compare apps
qdrant::search::compare_apps "query" "app1" "app2"

# Interactive explorer
qdrant::search::explore
```

## 🏗️ Architecture

### Directory Structure
```
embeddings/
├── manage.sh              # Main orchestrator
├── extractors/           # Content extraction
│   ├── workflows.sh      # N8n workflows
│   ├── scenarios.sh      # PRDs and configs
│   ├── docs.sh          # Documentation
│   ├── code.sh          # Functions and APIs
│   └── resources.sh     # Resource capabilities
├── indexers/            # Identity management
│   └── identity.sh      # App identity tracking
├── search/              # Search functionality
│   ├── single.sh        # Single app search
│   └── multi.sh         # Cross-app search
└── config/
    ├── schema.yaml      # Knowledge definitions
    └── templates/       # Document templates

```

### App Identity System

Each app maintains `.vrooli/app-identity.json`:
```json
{
  "app_id": "study-buddy-v2",
  "type": "generated",
  "source_scenario": "study-buddy",
  "last_indexed": "2025-01-22T12:00:00Z",
  "index_commit": "abc123def",
  "embedding_config": {
    "model": "mxbai-embed-large",
    "dimensions": 1024,
    "collections": {
      "workflows": "study-buddy-v2-workflows",
      "scenarios": "study-buddy-v2-scenarios",
      "knowledge": "study-buddy-v2-knowledge",
      "code": "study-buddy-v2-code",
      "resources": "study-buddy-v2-resources"
    }
  },
  "stats": {
    "total_embeddings": 523,
    "last_refresh_duration_seconds": 45
  }
}
```

## 📚 Document Templates

Standard templates in `config/templates/` ensure consistent knowledge capture:

### ARCHITECTURE.md
```markdown
## Design Decisions
<!-- EMBED:DECISION:START -->
### [Date] Decision Title
**Context:** What required this decision?
**Decision:** What was decided?
**Rationale:** Why this approach?
**Trade-offs:** What are the compromises?
<!-- EMBED:DECISION:END -->
```

### LESSONS_LEARNED.md
```markdown
## What Worked
<!-- EMBED:SUCCESS:START -->
### Success Title
**Context:** What was the situation?
**Implementation:** What did you do?
**Result:** What was the outcome?
**Reusable:** Can this be applied elsewhere?
<!-- EMBED:SUCCESS:END -->
```

## 🔍 Search Examples

### Finding Existing Solutions
```bash
# Agent needs to send emails
$ qdrant::search::all_apps "send automated emails with attachments" workflows

Results:
1. study-buddy-v2: "Email Report Workflow" (0.92 similarity)
   - Uses Gmail node with attachment handling
   
2. system-monitor: "Alert Email Workflow" (0.87 similarity)
   - Sends emails on system events
```

### Discovering Patterns
```bash
$ qdrant::search::discover_patterns "authentication"

Pattern Discovery:
By Type:
  code: 8 occurrences
  workflows: 3 occurrences
  knowledge: 2 occurrences

By App:
  vrooli-main: 5 matches
  user-portal-v1: 4 matches
  api-gateway: 4 matches
```

### Finding Knowledge Gaps
```bash
$ qdrant::search::find_gaps "rate limiting"

Knowledge Gap Analysis:
Current Coverage:
  • code: 2 items
  • knowledge: 1 item

Potential Gaps:
  ❌ No workflows found
  ❌ No scenarios found
  
Recommendation: Create rate limiting workflows
```

## 🎯 Use Cases

### For AI Agents
- **Avoid Reinventing**: Find existing solutions before building new ones
- **Learn Patterns**: Discover how similar problems were solved
- **Combine Solutions**: Mix patterns from different apps
- **Understand Context**: Access documentation and decisions

### For Developers
- **Code Discovery**: Find functions and APIs across projects
- **Pattern Mining**: Identify common architectural patterns
- **Knowledge Transfer**: Learn from other projects' experiences
- **Gap Identification**: See what's missing in documentation

### For Recursive Improvement
- **Knowledge Accumulation**: Every solution becomes permanent
- **Pattern Evolution**: Similar patterns get refined over time
- **Emergent Capabilities**: Combining patterns creates new solutions
- **Collective Intelligence**: All apps contribute to shared knowledge

## 🚀 Advanced Features

### Semantic Expansion
Queries automatically expand to related concepts:
- "email" → finds "mail", "notification", "Gmail", "SMTP"
- "database" → finds "postgres", "mysql", "sql", "query"

### Content-Addressed Storage
- Each embedding has unique ID from content hash
- Duplicate content automatically deduplicated
- Changes trigger new embeddings

### Batch Processing
- Extracts and embeds content in batches
- Progress tracking for large projects
- Parallel collection searches

## 📊 Performance

- **Single app search**: <100ms typical
- **Cross-app search** (10 apps): <500ms typical
- **Full refresh** (1000 items): ~60 seconds
- **Embedding generation**: ~50ms per item

## 🔧 Configuration

### Environment Variables
```bash
QDRANT_URL=http://localhost:6333  # Qdrant server URL
DEFAULT_MODEL=mxbai-embed-large   # Embedding model
DEFAULT_DIMENSIONS=1024            # Vector dimensions
BATCH_SIZE=10                      # Batch processing size
```

### Models
Supported embedding models:
- `mxbai-embed-large` (1024 dims) - Default, best quality
- `nomic-embed-text` (768 dims) - Faster, good quality
- `all-minilm` (384 dims) - Fastest, lower quality

## 🐛 Troubleshooting

### No Results Found
1. Check if embeddings are initialized: `qdrant::embeddings::status`
2. Refresh embeddings: `qdrant::embeddings::refresh --force`
3. Validate setup: `qdrant::embeddings::validate`

### Slow Search
1. Check Qdrant connection: `curl http://localhost:6333/collections`
2. Reduce search limit: Use smaller limit values
3. Filter by type: Search specific types instead of "all"

### Missing Content
1. Run validation: `qdrant::embeddings::validate`
2. Check extractors are finding content
3. Ensure files match expected patterns

## 🔮 Future Enhancements

- **Learning Search**: Track which results agents use
- **Query Expansion**: Automatic synonym generation
- **Incremental Updates**: Only embed changed files
- **Custom Extractors**: Plugin system for new content types
- **Search Analytics**: Track popular queries and gaps
- **Cross-Language**: Support for more programming languages

## 📝 Contributing

To add new content types:
1. Create extractor in `extractors/`
2. Add to schema.yaml
3. Update manage.sh to call extractor
4. Add collection to identity template

## 📄 License

Part of the Vrooli project. See main LICENSE file.

---

*Built for recursive improvement - where every problem solved makes the system permanently smarter.*