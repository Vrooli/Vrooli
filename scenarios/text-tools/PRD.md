# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-19
> **Status**: Active Implementation
> **Template Source**: PRD Control Tower Canonical Template

## 🎯 Overview

**Purpose**: Text-tools provides a comprehensive, reusable text processing and analysis platform that all other scenarios can leverage for text manipulation, search, extraction, transformation, and intelligence operations. It serves as the central text processing hub, eliminating duplication across scenarios and providing consistent, high-quality text operations.

**Primary Users**:
- Developers building text-processing features
- Data analysts working with text data
- Content creators managing text workflows
- Other Vrooli scenarios requiring text capabilities

**Deployment Surfaces**:
- RESTful API for programmatic access
- CLI interface for command-line workflows
- Event-driven integration for reactive scenarios
- Shared n8n workflows for common operations

**Key Capabilities**:
- Diff and comparison (line, word, character, semantic)
- Search and pattern matching (grep, regex, fuzzy, semantic)
- Text transformation (case, encoding, format conversion)
- Multi-format extraction (PDF, HTML, DOCX, OCR)
- AI-powered analysis (summarization, entities, sentiment)

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] Diff and comparison tools (line-by-line, word-by-word, semantic) ✅ 2025-09-30
- [x] Search and pattern matching (grep, regex, fuzzy search) ✅ 2025-09-30
- [x] Text transformation (case, encoding, format conversion) ✅ 2025-09-30
- [x] Text extraction from multiple formats (PDF, HTML, DOCX, images via OCR) ✅ 2025-09-30
- [x] Basic statistics (word count, character count, language detection) ✅ 2025-09-30
- [x] RESTful API for all core operations ✅ 2025-09-30
- [x] CLI interface with full feature parity ✅ 2025-09-30

### 🟠 P1 – Should have post-launch

- [x] Semantic search using embeddings (via Ollama) ✅ 2025-09-30
- [x] Text summarization and condensation ✅ 2025-09-30
- [x] Entity extraction (names, dates, locations, emails, URLs) ✅ 2025-09-30
- [x] Sentiment and tone analysis ✅ 2025-09-30
- [ ] Multi-language support and translation
- [ ] Batch processing for multiple files
- [x] Text sanitization (PII removal, HTML cleaning) ✅ 2025-09-30

### 🟢 P2 – Future / expansion

- [ ] Template engine with variables and conditionals
- [ ] Advanced NLP operations (topic modeling, keyword extraction)
- [ ] Text generation (test data, lorem ipsum, mockups)
- [ ] Visual diff interface with side-by-side comparison
- [ ] Pipeline builder for chaining operations
- [ ] Version history and change tracking

## 🧱 Tech Direction Snapshot

**Architecture Philosophy**: Modular plugin system for extensibility with streaming processing for large files.

**API Stack**:
- Go-based RESTful API with net/http
- JSON request/response format
- Streaming support for large file operations
- Plugin architecture for custom transformations

**CLI Stack**:
- Go CLI with Cobra framework
- Full API parity with human-readable defaults
- JSON output mode for scripting
- Unix pipe-friendly design

**Data Storage**:
- PostgreSQL for text metadata and operation history
- MinIO for large text files and processing results
- Redis (optional) for caching frequently accessed content
- Qdrant/pgvector for semantic search embeddings

**AI Integration**:
- Ollama for semantic search, summarization, NLP
- Graceful degradation when AI unavailable

**Performance Targets**:
- < 100ms response time for basic operations
- < 500ms for NLP operations
- Support files up to 100MB
- 1000 operations/second throughput

**Non-Goals**:
- Real-time collaborative editing (out of scope)
- Full text editor implementation (use existing tools)
- Direct database queries (use API/CLI)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- PostgreSQL (text metadata, operation history, templates)
- MinIO (large file storage, processing results)

**Optional Local Resources**:
- Ollama (semantic search, summarization, NLP)
- Redis (caching for performance)
- Qdrant (vector storage, fallback to pgvector)

**Resource Integration Priority**:
1. Use resource CLI commands (resource-postgres, resource-minio)
2. Direct API only when necessary (streaming operations)

**Scenario Dependencies**: None (foundational service)

**Downstream Enablement**:
- document-version-control (versioning with diff visualization)
- code-review-assistant (semantic code analysis)
- contract-analyzer (legal document comparison)
- content-moderator (automated content filtering)
- translation-hub (multi-language translation)

**Launch Risks**:
- Large file processing OOM → Mitigation: Stream processing, MinIO offload
- Ollama unavailability → Mitigation: Graceful degradation to basic ops
- OCR accuracy issues → Mitigation: Multiple engines, confidence scores

**Launch Sequence**:
1. Core P0 API implementation ✅ Complete
2. CLI with feature parity ✅ Complete
3. PostgreSQL/MinIO integration ✅ Complete
4. Ollama integration for P1 features ✅ Complete
5. Performance optimization (ongoing)
6. Cross-scenario integration validation (ongoing)

## 🎨 UX & Branding

**Visual Style**:
- Dark theme for developer-friendly experience
- Monospace fonts for code/text display
- System fonts for UI elements
- Dense layouts for maximum information density
- Subtle animations only for state transitions

**Personality & Tone**:
- Technical and focused
- Powerful and precise
- No unnecessary verbosity
- Professional developer tool aesthetic

**CLI Experience**:
- Intuitive verb-noun command patterns
- Human-readable output by default
- --json flag for machine parsing
- Unix pipe-friendly (stdin/stdout)
- Color-coded diff output
- Progress indicators for long operations

**API Experience**:
- Clear, consistent endpoint naming
- Comprehensive error messages
- API key authentication
- Rate limiting with clear feedback
- Detailed documentation with examples

**Accessibility**:
- WCAG AA compliance for UI components
- Keyboard navigation support
- High contrast color schemes
- Clear visual diff indicators
- Screen reader compatible outputs

**Target Feeling**: Users should feel like they have a Swiss Army knife for text processing—every tool they need, nothing extraneous, with the power and precision of a developer-grade solution.

## 📎 Appendix

### Related Documentation
- README.md - Quick start guide and basic usage
- docs/api.md - Complete API reference
- docs/cli.md - CLI usage examples and patterns
- docs/plugins.md - Plugin development guide
- requirements/index.json - Detailed requirements registry

### Related PRDs
- scenarios/image-tools/PRD.md - Sister service for image processing
- scenarios/document-manager/PRD.md - Document management (consumes text-tools)

### Intelligence Amplification
Text-tools amplifies agent intelligence by:
- Providing semantic search for conceptual relationship understanding
- Offering diff tools for learning from changes and iterations
- Supporting pattern extraction for identifying important information
- Enabling text transformation pipelines for continuous optimization
- Creating shared knowledge base of text processing patterns

### Recursive Value
New scenarios enabled after text-tools exists:
1. code-review-assistant (diff, syntax highlighting, semantic analysis)
2. document-version-control (diff, merge, transformation)
3. contract-analyzer (entity extraction, version comparison)
4. content-moderator (sentiment, entity extraction, pattern matching)
5. translation-hub (extraction, transformation infrastructure)

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 100ms for basic ops, < 500ms for NLP | API monitoring |
| Throughput | 1000 operations/second | Load testing |
| Accuracy | > 95% for entity extraction | Validation suite |
| Resource Usage | < 2GB memory, < 30% CPU | System monitoring |
| File Size Support | Up to 100MB per file | Integration tests |

### Quality Gates
- [x] All P0 requirements implemented and tested ✅ 2025-09-30
- [x] Integration tests pass with Ollama, PostgreSQL, MinIO ✅ 2025-09-30
- [ ] Performance targets met under load
- [x] Documentation complete (README, API docs, CLI help) ✅ 2025-09-30
- [x] Scenario can be invoked by other agents via API/CLI ✅ 2025-09-30
- [ ] At least 3 other scenarios successfully integrated

### Data Models
Primary entities stored in PostgreSQL:
- TextDocument (id, name, content_hash, format, size, language, minio_path, metadata)
- TextOperation (id, document_id, operation_type, parameters, result_summary, duration_ms)
- TextTemplate (id, name, content, variables, description, usage_count)

### Event Interface
Published events:
- text.document.created (document_id, format, size)
- text.analysis.completed (document_id, analysis_type, results)
- text.diff.detected (doc1_id, doc2_id, change_count)

Consumed events:
- document.uploaded → Automatically extract and index text
- translation.requested → Process translation via Ollama

### Security Considerations
- Optional encryption at rest in MinIO
- API key authentication for all endpoints
- Audit trail for all operations with user context
- PII detection and optional redaction
- Rate limiting to prevent abuse

### Known Limitations
- OCR accuracy ~90% on handwritten text (workaround: confidence scores, manual review)
- Large file processing limited to 100MB per file
- Semantic search requires Ollama availability

---

**Last Updated**: 2025-11-19
**Status**: Implementation Complete (P0: 100%, P1: 71%)
**Owner**: AI Agent
**Review Cycle**: Weekly validation against implementation
