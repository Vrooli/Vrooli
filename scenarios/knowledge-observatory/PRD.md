# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2026-01-26
> **Status**: Active
> **Template**: Canonical PRD v2.0.0

## 🎯 Overview

Knowledge Observatory provides real-time introspection and management of Vrooli's semantic memory system stored in Qdrant. It acts as a consciousness monitor for collective intelligence, showing health, drift, gaps, and relationships across knowledge collections.

**Purpose**: Enable operators and agents to query, assess, and evolve the knowledge base with confidence by surfacing semantic search, quality metrics, and graph relationships in one system.

**Primary Users**:
- Scenario operators validating knowledge freshness and coherence
- Agents planning tasks that depend on accurate knowledge coverage
- Engineers maintaining Vrooli's semantic memory infrastructure

**Deployment Surfaces**:
- Web UI dashboard (operator console)
- Go API for programmatic access
- CLI for terminal-first workflows

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Semantic search across Qdrant | Natural-language search returning ranked results across collections
- [x] OT-P0-002 | Knowledge quality metrics | Coherence, freshness, redundancy, and coverage metrics per collection
- [x] OT-P0-003 | Knowledge graph access | API-powered graph endpoint for semantic relationships
- [x] OT-P0-004 | API endpoints for knowledge queries | Stable REST endpoints for search, health, and graph queries
- [x] OT-P0-005 | CLI exploration commands | CLI workflows for search, health, and graph inspection
- [x] OT-P0-006 | Operator dashboard UI | Matrix-style monitoring UI with search, metrics, and health views

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Knowledge timeline visualization | Timeline view of knowledge ingestion and evolution
- [ ] OT-P1-002 | Bulk knowledge operations | Prune, merge, and export operations at collection scale
- [ ] OT-P1-003 | Scenario contribution tracking | Trace knowledge entries back to source scenarios
- [ ] OT-P1-004 | Semantic diffing | Compare concept evolution over time
- [ ] OT-P1-005 | Coverage gap analysis | Highlight missing topics by collection and scenario

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | 3D graph visualization | Clustered graph UI with 3D layouts
- [ ] OT-P2-002 | AI-driven recommendations | Suggest knowledge updates and curation actions
- [ ] OT-P2-003 | Knowledge export/import | Snapshot and restore knowledge bundles
- [ ] OT-P2-004 | Advanced metadata filtering | Filter by source, age, quality, and tags

## 📊 Success Metrics

### Performance Targets
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Search Response Time | < 500ms for 95% of queries | API monitoring |
| Graph Rendering | < 2s for up to 1000 nodes | UI performance tracking |
| Quality Calculation | < 1s per collection scan | Backend monitoring |
| Resource Usage | < 512MB memory, < 10% CPU | System monitoring |

### Quality Gates
- [x] All P0 targets implemented and tested
- [x] Integration tests pass with Qdrant and Postgres resources
- [x] Performance targets met under normal load
- [x] Documentation complete (README, API docs, CLI help)
- [x] Scenario callable via API/CLI from other agents

## 🧱 Tech Direction Snapshot

**Frontend Stack**:
- React + Vite UI for operator dashboard
- Tailwind CSS with Matrix-inspired design tokens
- React Query for API polling and caching
- Shared UI primitives for consistent layout and error handling

**Backend Stack**:
- Go API server with Gorilla Mux
- Qdrant vector store for embeddings
- PostgreSQL for metadata, metrics, and ingestion history
- Optional Ollama embedding service for semantic enrichment

**Data Flow**:
- UI ↔ API (REST)
- API ↔ Qdrant (vector search + graph)
- API ↔ Postgres (metrics, metadata, jobs)

**Non-Goals**:
- Not a full data lake or ETL system
- Not a replacement for upstream embedding pipelines
- Not a general-purpose BI dashboard

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- Qdrant (primary semantic storage)
- PostgreSQL (metadata, metrics, job state)

**Optional Resources**:
- Ollama (enhanced embeddings and semantic enrichment)

**Scenario Dependencies**:
- None (standalone capability)

**Launch Sequencing**:
1. P0 API endpoints and CLI workflows
2. Operator dashboard UI with health + search
3. Metrics and graph visualizations
4. P1 timeline and coverage analysis

**Risks**:
- Large collections can slow graph rendering without throttling
- Misaligned embeddings can skew quality metrics
- Resource outages reduce real-time telemetry accuracy

## 🎨 UX & Branding

**Visual Identity**:
- Matrix-inspired green-on-black aesthetic with subtle glow accents
- Monospace typography for telemetry metadata and CLI framing
- High-contrast status cards and alert panels

**Interaction Patterns**:
- Clear navigation between dashboard, search, graph, and metrics
- Immediate feedback for loading/error states
- Lightweight transitions that reinforce system status

**Accessibility**:
- WCAG AA contrast for core text
- Keyboard navigation for primary actions
- Aria labels for search and status elements

**Performance Expectations**:
- Initial UI load < 2s on local network
- Health status refresh every 5s without blocking interactions

## 📎 Appendix

**Capability Definition**: Knowledge Observatory turns semantic memory into an observable system. It exposes what knowledge exists, how fresh it is, and how it connects—allowing agents to avoid redundancy, fill gaps, and reason over concept relationships.

**Recursive Value**: Each insight into knowledge health becomes feedback for future scenarios, enabling automated curation, drift detection, and continuous intelligence growth.

