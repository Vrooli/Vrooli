# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Production-Ready Foundation
> **Template Compliance**: Canonical PRD Template v2.0.0

## 🎯 Overview

**Purpose**: Advanced multi-agent creative ideation platform with document intelligence, semantic search, and real-time AI collaboration. Provides sophisticated brainstorming capabilities where multiple specialized AI agents work together to generate, refine, and validate ideas using contextual document understanding.

**Primary Users**:
- Creative professionals seeking AI-powered brainstorming tools
- Product managers developing feature ideas with context
- Strategists and innovators requiring multi-perspective analysis
- Research teams building on existing documentation

**Deployment Surfaces**:
- **CLI**: `idea-generator` command for terminal-based ideation workflows
- **API**: RESTful endpoints at `/api/ideas/*`, `/api/campaigns/*`, `/api/documents/*`
- **UI**: Custom Node.js web interface with magic dice generation and campaign management
- **Events**: Pub/sub for idea lifecycle events (generation, refinement, validation)

**Intelligence Amplification**: This scenario provides multi-agent collaboration patterns, document intelligence workflows, iterative refinement processes, semantic idea connections, and specialized agent roles that enhance future agent analytical thinking capabilities.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Campaign-based idea organization with color coding | Organize ideas into colorful campaigns with full CRUD operations
- [x] OT-P0-002 | Interactive dice-roll interface for instant idea generation | Single-click AI-powered idea generation via custom web interface
- [ ] OT-P0-003 | Document intelligence with PDF/DOCX processing | Upload and extract context from research documents to inform idea generation
- [ ] OT-P0-004 | Semantic search across ideas and documents | Find related ideas and documents using vector similarity search
- [ ] OT-P0-005 | Real-time chat refinement with AI agents | WebSocket-based conversational refinement of ideas with specialized agents
- [ ] OT-P0-006 | Six specialized agent types | Revise, Research, Critique, Expand, Synthesize, and Validate agents providing distinct perspectives
- [ ] OT-P0-007 | Context-aware generation using uploaded documents | Generate ideas informed by extracted document context and semantic connections
- [ ] OT-P0-008 | Vector embeddings for semantic connections | Store and query idea embeddings in Qdrant for discovering related concepts

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Chat history persistence and searchability | Store and search conversation history across refinement sessions
- [ ] OT-P1-002 | Idea evolution tracking with version history | Track how ideas evolve through multiple refinement iterations
- [ ] OT-P1-003 | Drag-and-drop document upload interface | Intuitive file upload with visual progress and format validation
- [ ] OT-P1-004 | WebSocket-based real-time collaboration | Multiple users collaborating on campaigns simultaneously
- [ ] OT-P1-005 | Export capabilities for refined ideas | Export ideas with refinement history to markdown, JSON, or CSV formats

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Team collaboration with shared campaigns | Multi-user workspace with permissions and activity feeds
- [ ] OT-P2-002 | API integrations for external data sources | Connect to research databases, news feeds, and market data
- [ ] OT-P2-003 | Mobile-responsive design for on-the-go ideation | Optimized mobile experience with touch-friendly interactions
- [ ] OT-P2-004 | Custom agent creation toolkit | User-defined agent personalities and specializations
- [ ] OT-P2-005 | Advanced analytics and insights | Track ideation patterns, identify trends, and measure idea quality

## 🧱 Tech Direction Snapshot

**API Stack**: Go + PostgreSQL + Redis for campaigns, ideas, and chat sessions

**UI Approach**: Custom Node.js web interface (Express.js, HTML5, CSS3) with creative brainstorming UX featuring magic dice animations, colorful campaign tabs, and slide-out AI agent chat panel

**Data Storage**:
- PostgreSQL for structured data (campaigns, ideas, chat history)
- Qdrant for vector embeddings and semantic search
- MinIO for document storage (PDF, DOCX uploads)
- Redis for real-time chat sessions and caching

**AI/ML Integration**:
- Ollama via shared workflow for LLM inference across all six agent types
- nomic-embed-text model for 768-dimensional vector embeddings
- Multi-agent orchestration via n8n workflows

**Integration Strategy**:
- Shared workflows via n8n for reusable AI patterns (ollama.json, embedding-generator.json, document-intelligence.json)
- Resource CLIs for infrastructure management (resource-minio)
- Direct APIs only for real-time WebSocket chat and streaming document uploads

**Non-Goals**:
- Not a generic document management system (focus is ideation, not archival)
- Not a project management tool (ideas are exploratory, not task-oriented)
- Not a replacement for deep research platforms (complements but doesn't replace literature review tools)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- `postgres` – Campaign and idea persistence with JSONB for flexible metadata
- `ollama` – Multi-agent AI processing with mistral model for generation
- `qdrant` – Vector database for semantic search and idea connections
- `minio` – S3-compatible object storage for document uploads
- `redis` – Pub/sub for real-time chat and session management
- `n8n` – Workflow orchestration for multi-agent coordination
- `unstructured-io` – Document processing pipeline for PDF/DOCX extraction

**Scenario Dependencies**: None (standalone capability, but enhances research-assistant and mind-maps when combined)

**Risks**:
- Multi-agent coordination complexity may cause response delays (mitigation: robust workflow error handling, agent fallbacks)
- Document processing bottlenecks with large files (mitigation: queue management, parallel processing)
- Qdrant initialization issues observed in testing (mitigation: collection pre-creation in schema, health checks)
- Real-time chat performance with concurrent users (mitigation: WebSocket optimization, message batching)

**Launch Sequencing**:
1. Core foundation complete (Campaign CRUD + basic idea generation) ✅
2. Document intelligence integration (unstructured-io pipeline)
3. Semantic search enablement (Qdrant collection setup)
4. Multi-agent chat implementation (WebSocket + six agents)
5. P1 features (history, exports, collaboration)

## 🎨 UX & Branding

**Visual Identity**: "Figma meets Notion" – creative collaboration with enterprise polish. Primary focus on central magic dice with rotating gradient border, colorful campaign tabs with dynamic indicators, and idea cards with hover animations.

**Color Palette**:
- Primary: Indigo (#6366F1) for magic effects and primary actions
- Secondary: Pink (#EC4899) for creative energy gradients
- Tertiary: Green (#10B981) for validation and success states
- Campaign colors: 8 vibrant options (Red, Orange, Yellow, Green, Cyan, Blue, Purple, Pink) for user organization

**Typography**: Inter font family with 700 weight headings, 400 weight body text, 500 weight UI elements, optimized for creative professional readability

**Interaction Design**:
- Generation flow: Visual focus on central magic dice → optional context input → single-click generation → idea card slide-up animation
- Refinement flow: Brain icon on card → chat panel slides from right → agent selection → interactive conversation → apply suggestions
- Campaign management: Plus button → modal with name/description/color → new tab appears → ideas auto-organized
- Document integration: Upload button or drag-drop → visual progress → extracted context available → semantic search enabled

**Accessibility**:
- WCAG 2.1 AA compliance with proper color contrast and semantic HTML
- Full keyboard navigation (Space=Generate, Cmd+K=Search, Esc=Close)
- ARIA labels and focus management throughout interface
- Screen reader optimized with descriptive announcements

**Animation Philosophy**: CSS-based 60fps animations using transform/opacity for smooth experience. Magic dice bounce (1s infinite), gradient rotation (10s), hover lifts (0.3s cubic-bezier), panel slides (0.3s ease)

**Responsive Strategy**: Desktop-first optimization for creative professionals on large screens, tablet adaptation with collapsible panels and touch interactions, essential mobile features accessible on smaller devices

**Brand Voice**: "Your AI-powered innovation lab" – creative and engaging while maintaining enterprise value proposition. Professional yet playful, intelligent collaboration without overwhelming complexity.
