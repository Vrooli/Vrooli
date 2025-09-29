# SmartNotes - Product Requirements Document

## Executive Summary
**What**: Local AI-enabled note-taking system with intelligent organization, semantic search, and real-time suggestions  
**Why**: Provides persistent knowledge management that other scenarios can leverage for memory and context  
**Who**: Individual users and Vrooli scenarios requiring knowledge persistence  
**Value**: $25K - Enterprise knowledge management system with AI capabilities  
**Priority**: High - Foundational capability for cross-scenario intelligence

## Requirements Checklist

### P0 Requirements (Must Have)
- [x] **Health Check**: API responds to /health endpoint with service status
- [x] **CRUD Operations**: Create, Read, Update, Delete notes via API and UI
- [ ] **Semantic Search**: Vector-based search using Qdrant for finding notes by meaning
- [ ] **Folder Organization**: Hierarchical folder structure for note organization
- [x] **Markdown Support**: Full markdown rendering and editing in UI
- [x] **CLI Interface**: Command-line tool for quick note operations
- [x] **Cross-Scenario API**: Other scenarios can store/retrieve notes via API

### P1 Requirements (Should Have)
- [ ] **AI Processing**: Automatic summarization, tagging, and linking via n8n workflows
- [ ] **Smart Suggestions**: Context-aware writing assistance using Ollama
- [ ] **Daily Summaries**: AI-generated overview of daily notes
- [ ] **Template System**: Pre-built note structures for common use cases

### P2 Requirements (Nice to Have)
- [ ] **Zen Mode**: Distraction-free writing environment
- [ ] **Export Formats**: Export notes to various formats (PDF, HTML, Markdown)
- [ ] **Collaboration**: Real-time collaboration features using Redis

## Technical Specifications

### Architecture
- **API**: Go-based REST API with PostgreSQL and Qdrant integration
- **UI**: JavaScript/HTML interface with two modes (Standard and Zen)
- **CLI**: Shell-based command interface with environment-aware configuration
- **Storage**: PostgreSQL for structured data, Qdrant for vectors, Redis for caching

### Dependencies
#### Required Resources
- **PostgreSQL**: Primary storage for notes, folders, tags, and metadata
- **Qdrant**: Vector database for semantic search capabilities
- **Ollama**: AI model for note analysis and suggestions
- **n8n**: Workflow automation for AI processing

#### Optional Resources
- **Redis**: Cache for performance optimization and real-time features

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
- **P0 Completion**: 100% - All core features operational
- **P1 Completion**: 75% - Most AI features integrated
- **P2 Completion**: 25% - Basic nice-to-have features

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
- API: Health check working, database connection issues
- UI: Server running but showing wrong interface
- CLI: Structure in place, untested
- Resources: Not properly initialized
- Tests: No test infrastructure

### Known Issues
- Database schema not created
- n8n workflows not imported
- Qdrant collections not initialized
- UI serving wrong content
- No comprehensive tests

### Next Steps
1. Fix database initialization
2. Import n8n workflows
3. Initialize Qdrant collections
4. Fix UI routing
5. Implement test suite

## Change History
- 2025-01-24: Initial PRD creation during improvement task