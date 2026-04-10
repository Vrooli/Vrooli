# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Canonical Specification
> **Template**: PRD Control Tower

## 🎯 Overview

**News Aggregator & Bias Analysis** is a real-time news aggregation platform with AI-powered bias detection, fact-checking, and multi-perspective analysis. It empowers users to consume news more critically by surfacing bias patterns, fact-check results, and diverse viewpoints across sources.

**Primary Users:**
- Journalists seeking balanced reporting
- Media researchers analyzing coverage patterns
- News consumers wanting unbiased information
- Academic institutions studying media bias

**Deployment Surfaces:**
- **UI**: Web-based dashboard for news exploration and analysis
- **API**: RESTful endpoints for article ingestion, bias scoring, and retrieval
- **CLI**: Command-line tools for feed management and batch analysis

**Purpose:** Add permanent capability for autonomous news monitoring, bias detection, and perspective synthesis using local AI resources.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | RSS feed aggregation | Ingest and parse multiple RSS feeds continuously
- [ ] OT-P0-002 | Article storage and retrieval | Store articles with metadata in PostgreSQL, expose via API
- [ ] OT-P0-003 | AI bias detection | Analyze article text for political/emotional bias using Ollama
- [ ] OT-P0-004 | Bias visualization UI | Display bias scores and analysis in web interface
- [ ] OT-P0-005 | Source management | CRUD operations for managing news sources and feeds
- [ ] OT-P0-006 | Real-time updates | Cache and serve trending topics via Redis
- [ ] OT-P0-007 | Health monitoring | API/UI health endpoints for lifecycle management

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Fact-checking integration | Cross-reference claims against fact-check databases
- [ ] OT-P1-002 | Perspective aggregation | Group articles by topic, show coverage differences
- [ ] OT-P1-003 | Historical trend analysis | Track bias patterns over time per source
- [ ] OT-P1-004 | User preferences | Save custom feeds, bias sensitivity, topic filters
- [ ] OT-P1-005 | Export functionality | Generate reports of bias analysis for articles/topics
- [ ] OT-P1-006 | Vector search | Semantic article similarity using Qdrant embeddings

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Automated alerts | n8n workflows for bias spike notifications
- [ ] OT-P2-002 | Cross-publication analysis | Compare how different outlets cover same events
- [ ] OT-P2-003 | Citation tracking | Trace source attribution chains
- [ ] OT-P2-004 | Browser extension | Real-time bias scoring for any article

## 🧱 Tech Direction Snapshot

**Stack:**
- **API**: Go (net/http) with PostgreSQL storage and Redis caching
- **UI**: Node.js/Express static server, vanilla JS or lightweight framework
- **AI**: Ollama (llama3.2) for bias analysis and content enrichment
- **Storage**: PostgreSQL schemas for articles, feeds, sources, analysis results
- **Caching**: Redis for feed caching, trending topics, real-time data

**Architecture:**
- RESTful API design with standard CRUD patterns
- Background feed polling separate from API serving
- Isolated test database for phased testing
- Lifecycle v2.0 compliance (no lib/ folder, service.json driven)

**Integration Strategy:**
- Direct resource integration (Postgres, Redis, Ollama, Qdrant, n8n)
- No scenario dependencies; standalone capability
- Optional n8n automation for alert workflows

**Non-Goals:**
- Not a content management system or publishing platform
- Not a social media aggregator (RSS/web sources only)
- Not providing original journalism or fact-checking research

## 🤝 Dependencies & Launch Plan

**Required Resources:**
- **PostgreSQL** – Article, feed, and analysis storage
- **Redis** – Feed caching, real-time updates, trending topics
- **Ollama** (llama3.2) – AI-powered bias detection and analysis
- **Qdrant** – Vector embeddings for semantic search (P1)
- **n8n** – Automation workflows for alerts (P2)

**Launch Sequence:**
1. Validate resource availability (dependencies phase)
2. Initialize database schema (setup lifecycle)
3. Build API binary and UI bundle
4. Run phased test suite (70-80% coverage target)
5. Start API and UI servers via lifecycle
6. Monitor health endpoints for operational status

**Risks:**
- RSS feed parsing fragility (many formats, encodings)
- AI bias detection accuracy depends on model quality
- Performance at scale (100K+ articles requires indexing strategy)
- Fact-check API rate limits (if external services used)

## 🎨 UX & Branding

**Visual Palette:**
- Clean, neutral interface emphasizing readability
- Color-coded bias indicators (left/center/right spectrum)
- Minimalist typography for long-form article consumption

**Accessibility Commitments:**
- WCAG 2.1 AA compliance
- High contrast mode for bias visualizations
- Keyboard navigation throughout UI
- Screen reader support for analysis results

**Voice & Personality:**
- Objective, informative tone
- Transparent about methodology and limitations
- Educational framing (not judgmental)
- Respect for diverse viewpoints

**Experience Promise:**
- Users should feel empowered, not overwhelmed
- Bias scores present context, not absolute truth
- Comparisons across sources foster critical thinking
- Quick access to original articles preserves user agency
