# SmartNotes - Product Requirements Document

## Executive Summary
**What**: Local AI-enabled note-taking system with intelligent organization, semantic search, and real-time suggestions
**Why**: Provides persistent knowledge management that other scenarios can leverage for memory and context
**Who**: Individual users and Vrooli scenarios requiring knowledge persistence
**Value**: $25K - Enterprise knowledge management system with AI capabilities
**Priority**: High - Foundational capability for cross-scenario intelligence

## 🎯 Capability Definition

### Core Capability
SmartNotes adds persistent, AI-enhanced knowledge management to Vrooli, enabling scenarios to store, retrieve, and intelligently search textual information using semantic understanding rather than just keyword matching.

### Intelligence Amplification
- **Memory Persistence**: Scenarios gain long-term memory, allowing complex workflows to reference historical context
- **Semantic Understanding**: Vector-based search enables finding related information even when exact keywords don't match
- **Cross-Scenario Knowledge**: Any scenario can query SmartNotes' knowledge base via API, creating a shared intelligence layer

### Recursive Value
Future scenarios enabled by SmartNotes:
1. **Research Assistant**: Can store findings and cross-reference related topics automatically
2. **Meeting Summarizer**: Stores meeting notes and links to related past discussions
3. **Personal Knowledge Graph**: Builds relationship maps between notes across all scenarios
4. **Context-Aware Automation**: Workflows can query relevant past notes before making decisions
5. **Learning System**: Tracks what worked/failed across sessions to improve future actions

## Requirements Checklist

### P0 Requirements (Must Have)
- [x] **Health Check**: API responds to /health endpoint with service status ✅ 2025-01-24
- [x] **CRUD Operations**: Create, Read, Update, Delete notes via API and UI ✅ 2025-01-24
- [x] **Semantic Search**: Vector-based search using Qdrant for finding notes by meaning ✅ 2025-01-24
- [x] **Folder Organization**: Hierarchical folder structure for note organization ✅ 2025-01-24
- [x] **Markdown Support**: Full markdown rendering and editing in UI ✅ 2025-01-24
- [x] **CLI Interface**: Command-line tool for quick note operations ✅ 2025-01-24
- [x] **Cross-Scenario API**: Other scenarios can store/retrieve notes via API ✅ 2025-01-24

### P1 Requirements (Should Have)
- [ ] **AI Processing**: Automatic summarization, tagging, and linking via n8n workflows
- [ ] **Smart Suggestions**: Context-aware writing assistance using Ollama
- [ ] **Daily Summaries**: AI-generated overview of daily notes
- [ ] **Template System**: Pre-built note structures for common use cases

### P2 Requirements (Nice to Have)
- [ ] **Zen Mode**: Distraction-free writing environment
- [ ] **Export Formats**: Export notes to various formats (PDF, HTML, Markdown)
- [ ] **Collaboration**: Real-time collaboration features using Redis

## 🏗️ Technical Architecture

### Architecture
- **API**: Go-based REST API with PostgreSQL and Qdrant integration
- **UI**: JavaScript/HTML interface with two modes (Standard and Zen)
- **CLI**: Shell-based command interface with environment-aware configuration
- **Storage**: PostgreSQL for structured data, Qdrant for vectors, Redis for caching

### Resource Dependencies
**Required:**
- **PostgreSQL**: Primary storage for notes, folders, tags, metadata (Direct SQL connection)
- **Qdrant**: Vector database for semantic search (Direct API + Ollama embeddings)
- **Ollama**: AI embeddings via nomic-embed-text model (Shared workflow: embedding-generator.json)
- **n8n**: Workflow automation for AI processing (Shared workflows in service.json)

**Optional:**
- **Redis**: Cache for performance optimization and real-time collaboration features

### Data Models
```yaml
primary_entities:
  - name: Note
    storage: postgres (metadata), qdrant (vectors)
    schema:
      id: UUID
      user_id: UUID
      title: string
      content: text (markdown)
      vector_embedding: float[]
      tags: string[]
      folder_id: UUID (nullable)
  - name: Folder
    storage: postgres
    schema:
      id: UUID
      name: string
      parent_id: UUID (nullable)
      position: int
  - name: Tag
    storage: postgres
    schema:
      id: UUID
      name: string
      color: string
      usage_count: int
```

### API Endpoints
- `GET /health` - Service health check
- `GET /api/notes` - List all notes
- `POST /api/notes` - Create new note
- `GET /api/notes/:id` - Get specific note
- `PUT /api/notes/:id` - Update note
- `DELETE /api/notes/:id` - Delete note
- `GET /api/notes/search` - Semantic search
- `GET /api/folders` - List folders
- `POST /api/folders` - Create folder
- `GET /api/tags` - List tags
- `POST /api/notes/:id/summarize` - Generate AI summary
- `GET /api/templates` - List templates

### CLI Commands
- `notes list` - List all notes
- `notes new <title>` - Create new note
- `notes view <id>` - View note content
- `notes edit <id>` - Edit note
- `notes delete <id>` - Delete note
- `notes search <query>` - Search notes
- `notes folders` - List folders
- `notes tags` - List tags
- `notes templates` - List templates
- `notes summary` - Get daily summary

### n8n Workflows
- `note-processor.json` - Main note processing pipeline
- `smart-tagging.json` - Automatic tag generation
- `note-search.json` - Enhanced search with context
- `daily-summary.json` - Generate daily summaries
- `smart-suggestions.json` - Real-time writing suggestions
- `intelligent-note-analyzer.json` - Deep note analysis
- `intelligent-note-linker.json` - Discover note relationships
- `note-to-mindmap.json` - Generate visual mindmaps

## Success Metrics

### Completion Targets
- **P0 Completion**: 100% ✅ - All core features operational
- **P1 Completion**: 0% - AI features pending n8n workflow setup
- **P2 Completion**: 0% - Nice-to-have features not started

### Quality Metrics
- **API Response Time**: < 500ms for standard operations
- **Search Accuracy**: > 85% relevance for semantic search
- **UI Load Time**: < 2 seconds initial load
- **Test Coverage**: > 80% for core functionality

### Performance Benchmarks
- **Concurrent Users**: Support 10+ simultaneous users
- **Note Capacity**: Handle 10,000+ notes efficiently
- **Search Speed**: Return results in < 1 second
- **Workflow Processing**: Complete AI analysis in < 5 seconds

## Business Justification

### Revenue Potential
- **Direct Sales**: $25K for enterprise deployment
- **SaaS Model**: $50/user/month for cloud version
- **API Access**: $500/month for external integrations

### Strategic Value
- **Foundation Service**: Critical for Vrooli's memory persistence
- **Cross-Scenario Synergy**: Enables knowledge sharing between scenarios
- **User Retention**: Sticky feature that increases platform value

## Implementation Progress

### Current Status
- API: ✅ Fully operational with all CRUD operations
- UI: ✅ Server running and accessible
- CLI: ✅ Functional for basic operations
- Database: ✅ PostgreSQL with full schema
- Semantic Search: ✅ Qdrant integration implemented
- Tests: ✅ Smoke and integration tests added

### Improvements Made (2025-01-24)
- ✅ Implemented semantic search with Qdrant vector database
- ✅ Added vector embeddings using Ollama's nomic-embed-text model
- ✅ Created indexing pipeline for automatic note vectorization
- ✅ Verified all API endpoints (notes, folders, tags, templates)
- ✅ Added comprehensive test infrastructure (smoke, integration tests)
- ✅ Validated folder hierarchy functionality
- ✅ Ensured cross-scenario API accessibility

### Known Limitations
- n8n workflows not yet imported (affects P1 AI features)
- Semantic search indexing happens asynchronously (slight delay)
- Template system basic (no advanced customization)
- No revision history tracking yet

### Next Steps for Future Improvements
1. Import and configure n8n workflows for AI processing
2. Add Ollama integration for smart suggestions
3. Implement daily AI summaries
4. Add real-time collaboration with Redis
5. Enhance UI with Zen mode

## 🖥️ CLI Interface Contract

### Command Structure
```bash
# Primary CLI executable
notes --help                  # Show all commands
notes list                    # List all notes
notes new <title>             # Create new note
notes view <id>               # View note content
notes edit <id>               # Edit note
notes delete <id>             # Delete note
notes search <query>          # Search notes
notes folders                 # List folders
notes tags                    # List tags
notes templates               # List templates
notes summary                 # Get daily summary
```

### Installation
```bash
./cli/install.sh             # Install CLI to ~/.local/bin
```

## 🔄 Integration Requirements

### Cross-Scenario Usage
Other scenarios can integrate with SmartNotes via:
1. **API Endpoints**: RESTful HTTP API for CRUD operations
2. **CLI Commands**: Shell-based interface for quick operations
3. **Shared Workflows**: n8n workflows for AI-enhanced operations

### Integration Patterns
```yaml
api_integration:
  - POST /api/notes: Store scenario output as notes
  - GET /api/search/semantic: Find related context
  - GET /api/notes?folder_id=X: Retrieve scenario-specific notes

cli_integration:
  - notes new "$(scenario output)": Pipe output to notes
  - notes search "query": Find relevant context

workflow_integration:
  - note-processor.json: AI-enhanced note processing
  - semantic-search.json: Vector-based search
```

## 🎨 Style and Branding Requirements

### UI Style
- **Standard Mode**: Clean, modern interface with productivity focus
  - Collapsible sidebar for folders/tags
  - Markdown preview with live rendering
  - Keyboard shortcuts for power users

- **Zen Mode**: Distraction-free writing environment
  - Minimal UI elements
  - Focus mode with ambient background
  - Automatic saving

### Visual Design
- **Colors**: Blue (#6366f1) for primary actions, Green (#10b981) for tags
- **Typography**: Monospace for code blocks, Sans-serif for UI
- **Icons**: Emoji-based folder icons for quick visual identification

## 💰 Value Proposition

### Direct Revenue
- **Enterprise License**: $25,000 one-time for on-premise deployment
- **SaaS Model**: $50/user/month for cloud-hosted version
- **API Access**: $500/month for external system integrations

### Strategic Value
- **Foundation Service**: Enables 20+ future scenarios requiring persistent memory
- **Competitive Moat**: AI-enhanced semantic search differentiates from basic note tools
- **User Retention**: Sticky feature that locks in users (hard to migrate notes away)

### Market Positioning
- **Roam Research**: $15/month → SmartNotes offers similar features locally for free
- **Notion**: $10/month → SmartNotes provides better AI integration
- **Obsidian**: Free locally → SmartNotes adds semantic search + cross-scenario integration

## 🧬 Evolution Path

### Phase 1 (Current - P0 Complete)
- ✅ Basic CRUD operations
- ✅ Semantic search with Qdrant
- ✅ Folder organization
- ✅ Cross-scenario API access

### Phase 2 (Next - P1 Implementation)
- AI-powered summarization via n8n workflows
- Smart tagging and auto-linking
- Daily AI-generated summaries
- Template system expansion

### Phase 3 (Future - P2 Enhancements)
- Real-time collaboration (Redis)
- Export to multiple formats
- Revision history tracking
- Knowledge graph visualization

### Long-term Vision
- Become the "memory layer" for all Vrooli scenarios
- Enable AI agents to learn from historical context
- Build relationship graphs across all notes automatically

## 🔄 Scenario Lifecycle Integration

### Setup Phase
```bash
vrooli scenario setup notes
# 1. Builds Go API binary
# 2. Installs npm dependencies
# 3. Creates PostgreSQL database and schema
# 4. Initializes Qdrant collections
# 5. Generates code embeddings
# 6. Installs CLI to ~/.local/bin
```

### Development Phase
```bash
vrooli scenario start notes
# Starts both API (Go) and UI (Node.js) servers
# Ports allocated via service.json configuration
```

### Testing Phase
```bash
vrooli scenario test notes
# Runs phased test suite:
# - Structure validation
# - Dependency checks
# - Business logic tests
# - Performance benchmarks
```

### Lifecycle Hooks
- **Pre-setup**: Check resource dependencies (postgres, qdrant, ollama, n8n)
- **Post-setup**: Populate sample notes if database empty
- **Health checks**: API /health endpoint every 30s
- **Graceful shutdown**: Save pending changes, close DB connections

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Qdrant vector DB unavailable | High - No semantic search | Fallback to PostgreSQL full-text search |
| PostgreSQL connection lost | Critical - No data access | Exponential backoff retry with 10 attempts |
| Ollama embedding slow | Medium - Async indexing lag | Queue embeddings, process in background |
| n8n workflows fail | Low - P1 features affected | Graceful degradation, log errors |

### Data Risks
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Note data loss | Critical | PostgreSQL backups, WAL archiving |
| Vector index corruption | Medium | Re-index from source notes |
| Concurrent edit conflicts | Low | Last-write-wins with timestamp |

### Operational Risks
| Risk | Impact | Mitigation |
|------|--------|-----------|
| Port conflicts | Medium | Dynamic port allocation via service.json |
| Memory leaks (Go) | Low | Connection pooling, regular health monitoring |
| UI build failures | Low | Pre-built static files, fallback mode |

## ✅ Validation Criteria

### Functional Validation
- [ ] All P0 CRUD endpoints return 2xx for valid requests
- [ ] Semantic search returns relevant results (>85% accuracy on test queries)
- [ ] CLI commands execute successfully and output valid JSON/text
- [ ] Folder hierarchy maintains parent-child relationships correctly
- [ ] Tags auto-complete and track usage counts accurately

### Performance Validation
- [ ] Health check responds in <500ms
- [ ] List notes endpoint responds in <500ms for 1000 notes
- [ ] Create note completes in <1000ms including DB write
- [ ] Semantic search returns results in <1000ms
- [ ] Concurrent requests (10 users) handled without timeouts

### Integration Validation
- [ ] PostgreSQL schema creates successfully on first setup
- [ ] Qdrant collections initialize with correct vector dimensions
- [ ] Ollama embeddings generate without errors
- [ ] n8n workflows import successfully (when implemented)
- [ ] Other scenarios can call API endpoints and receive valid responses

### Security Validation
- [ ] No SQL injection vulnerabilities in query parameters
- [ ] Environment variables required (no default passwords)
- [ ] Sensitive data (passwords) not logged to stdout/stderr
- [ ] CORS configured to prevent unauthorized cross-origin requests

## 📝 Implementation Notes

### Code Organization
- **api/main.go**: Core API server with all handlers
- **api/semantic_search.go**: Qdrant integration and vector operations
- **ui/index.html**: Standard note-taking interface
- **ui/zen-index.html**: Distraction-free writing mode
- **cli/notes**: Shell script for CLI operations
- **test/phases/**: Phased testing scripts (structure, dependencies, business, performance)

### Development Patterns
- **Error Handling**: Log errors, return user-friendly messages
- **Database Pooling**: Max 25 open connections, 5 idle, 5min lifetime
- **Async Operations**: Background goroutines for embedding generation
- **Health Monitoring**: Exponential backoff for DB reconnection (10 retries, 30s max delay)

### Testing Strategy
- **Unit Tests**: Go tests for individual handlers (api/*_test.go)
- **Integration Tests**: Bash scripts testing full workflows (test/phases/test-integration.sh)
- **Business Tests**: End-to-end user scenarios (test/phases/test-business.sh)
- **Performance Tests**: Load testing and response time validation (test/phases/test-performance.sh)

### Deployment Considerations
- **Port Configuration**: Uses API_PORT and UI_PORT from environment
- **Database Setup**: Requires postgres resource running first
- **Vector Storage**: Qdrant must be populated before semantic search works
- **CLI Installation**: Run cli/install.sh to add to PATH

## 🔗 References

### External Documentation
- [Qdrant Vector Database](https://qdrant.tech/documentation/)
- [Ollama Embeddings API](https://ollama.ai/library/nomic-embed-text)
- [n8n Workflow Automation](https://docs.n8n.io/)
- [PostgreSQL Full-Text Search](https://www.postgresql.org/docs/current/textsearch.html)

### Internal References
- `/docs/scenarios/PHASED_TESTING_ARCHITECTURE.md` - Testing standards
- `/scripts/resources/contracts/v2.0/universal.yaml` - Service contract spec
- `initialization/postgres/schema.sql` - Database schema definition
- `initialization/qdrant/collections.json` - Vector collection config

### Related Scenarios
- **research-assistant**: Uses SmartNotes API for storing research findings
- **meeting-summarizer**: Stores meeting notes and links to related discussions
- **personal-knowledge-graph**: Builds on SmartNotes to create relationship maps

### Shared Workflows
- `initialization/automation/n8n/ollama.json` - AI text generation
- `initialization/automation/n8n/embedding-generator.json` - Vector embeddings
- `initialization/automation/n8n/semantic-search.json` - Vector similarity search

## Change History
- 2025-10-20: Updated PRD to match v2.0 standards (added all required sections)
- 2025-01-24: Implemented comprehensive test suite
- 2025-01-24: Added semantic search capability with Qdrant
- 2025-01-24: Completed all P0 requirements
- 2025-01-24: Initial PRD creation during improvement task