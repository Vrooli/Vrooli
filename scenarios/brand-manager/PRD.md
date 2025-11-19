# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Canonical
> **Template Compliance**: v2.0.0

## 🎯 Overview

Brand Manager provides AI-powered brand identity generation and automated application integration. It creates comprehensive brand packages including logos, color palettes, typography systems, slogans, and marketing copy, then automatically applies these assets to existing Vrooli applications through Claude Code integration.

**Purpose**: Add permanent brand intelligence capability to Vrooli that enables professional visual identity creation and automated brand consistency across all scenarios.

**Primary Users**:
- Scenario developers needing professional branding for applications
- Agencies and consultants delivering branded solutions to clients
- Product managers launching new SaaS offerings

**Deployment Surfaces**:
- REST API for programmatic brand generation
- Web UI for interactive brand creation and management
- CLI for automation and CI/CD integration
- N8n workflows for orchestrated brand pipelines

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Complete brand generation pipeline | Generate logos, color palettes, typography, slogans, and marketing copy through AI-powered workflows
- [ ] OT-P0-002 | Professional web UI for brand management | Provide interactive interface for creating, viewing, and managing brand assets
- [ ] OT-P0-003 | ComfyUI integration for visual assets | Generate logos, icons, and favicons using AI image generation workflows
- [ ] OT-P0-004 | Automated Claude Code app integration | Apply generated brands to existing Vrooli applications through automated agent workflows
- [ ] OT-P0-005 | Brand asset storage and export | Store brand data in PostgreSQL and assets in MinIO with export packaging system
- [ ] OT-P0-006 | N8n workflow orchestration | Coordinate brand generation and integration processes through automated workflows
- [ ] OT-P0-007 | API endpoints for brand operations | Provide REST API for creating, retrieving, and integrating brand identities
- [ ] OT-P0-008 | Database schema for brand storage | Store brand metadata, color systems, typography, and integration tracking
- [ ] OT-P0-009 | Asset management system | Handle multiple asset formats (PNG, SVG, ICO) with version tracking
- [ ] OT-P0-010 | Ollama integration for copy generation | Generate brand copy, slogans, and marketing text using local LLM

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Brand template library | Provide industry-specific templates for common verticals
- [ ] OT-P1-002 | Color palette analyzer | Analyze color harmony and accessibility compliance
- [ ] OT-P1-003 | Brand guideline generation | Auto-generate comprehensive brand guideline documents
- [ ] OT-P1-004 | Multi-format asset variants | Generate logos in multiple sizes and formats automatically
- [ ] OT-P1-005 | Brand asset versioning | Track brand evolution with version history and rollback capability
- [ ] OT-P1-006 | Integration backup system | Create app backups before applying brand changes
- [ ] OT-P1-007 | Real-time preview system | Show live brand asset previews during generation
- [ ] OT-P1-008 | Qdrant semantic search | Find similar brands using vector embeddings for differentiation
- [ ] OT-P1-009 | Vault integration for secrets | Securely manage API keys and integration tokens
- [ ] OT-P1-010 | Brand export packaging | Package complete brand assets with guidelines for distribution

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Brand consistency checker | Validate brand application across multiple applications
- [ ] OT-P2-002 | A/B testing for variants | Test multiple brand versions with performance tracking
- [ ] OT-P2-003 | Brand analytics dashboard | Track brand usage and performance metrics
- [ ] OT-P2-004 | External design tool integration | Connect with Figma, Adobe tools for advanced workflows
- [ ] OT-P2-005 | ML-based brand optimization | Learn from feedback to improve future brand generation
- [ ] OT-P2-006 | Multi-brand management | Handle multiple brand identities for enterprise scenarios
- [ ] OT-P2-007 | Automated brand evolution | Adapt brands based on market feedback and trends
- [ ] OT-P2-008 | Real-time consistency monitoring | Alert on brand guideline violations across deployments

## 🧱 Tech Direction Snapshot

**Preferred UI Stack**: Vanilla JavaScript with modern CSS (glass morphism effects, gradient backgrounds), HTML5, responsive design with mobile-first approach

**Preferred API Stack**: Go with Gin framework, PostgreSQL for structured data, MinIO S3-compatible storage for assets

**Integration Strategy**:
- N8n workflows for orchestration (brand-pipeline.json, claude-spawner.json)
- ComfyUI custom workflows for visual asset generation (logo-generator.json, icon-creator.json)
- Direct database connections for complex queries
- REST API integration with Claude Code for automated app updates
- Ollama for text generation via N8n workflow nodes

**Data Storage**:
- PostgreSQL: Brand metadata, color systems, typography, integration tracking
- MinIO: Logo files, icon variants, favicon sets, export packages
- Qdrant (optional): Brand embeddings for semantic similarity search

**Non-goals**:
- External API dependencies (all processing local)
- Real-time collaborative editing (single-user workflow initially)
- Video or motion graphics generation (static assets only in v1.0)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- n8n (workflow orchestration for brand generation and Claude Code integration)
- comfyui (AI-powered logo and visual asset generation)
- ollama (brand copy generation and style analysis)
- postgres (brand data and integration tracking storage)
- minio (asset storage with S3-compatible API)
- claude-code (automated app integration agent)

**Optional Local Resources**:
- vault (secure API key storage, fallback to environment variables)
- qdrant (brand similarity search, fallback to database matching)

**Launch Sequencing**:
1. Core brand generation pipeline with ComfyUI and Ollama
2. Web UI for brand creation and asset viewing
3. Claude Code integration for automated app updates
4. Template library and advanced features (P1 targets)

**Known Risks**:
- ComfyUI workflow stability under load
- Claude Code integration complexity for diverse app structures
- Asset generation quality consistency
- MinIO storage scaling for large asset libraries

## 🎨 UX & Branding

**Visual Palette**: Deep purple primary (#2D1B69 for creativity), vibrant orange secondary (#FF6B35 for energy), teal accent (#4ECDC4 for balance), gradient backgrounds (667eea to 764ba2)

**Typography Tone**: Inter for UI (modern, clean), JetBrains Mono for technical content, Poppins for headings and brand names

**Accessibility Commitments**: WCAG AA compliance (4.5:1 color contrast minimum), full keyboard navigation support, proper ARIA labels, descriptive alt text for all brand assets, clear focus indicators

**Voice/Personality**: Creative professional tone, inspiring and confident mood, empowering users to create beautiful brands without design expertise

**Motion Language**: Smooth micro-interactions (color palette animations on hover, asset generation progress indicators, brand preview real-time updates), purposeful animations that guide attention

**Experience Promise**: "Modern design tool meets professional brand management" - combining the creativity of Figma/Canva with enterprise-grade brand asset management

## 📎 Appendix

**Design Philosophy**: Brand Manager bridges the gap between AI-powered creativity and professional brand deployment. Every generated brand becomes a permanent capability that other scenarios can leverage for cohesive visual identities.

**Recursive Intelligence Value**: Enables any scenario building user-facing applications to ensure consistent branding. Brand asset generation patterns become reusable knowledge for future app development. Unlocks downstream scenarios: marketing campaign generator, e-commerce store builder, SaaS product generator, website theme generator.

**Revenue Model**: Direct deployment value $10K-$50K per agency/consultant installation. Cost savings: Replaces $5K-$25K traditional brand design projects. Reusability: Every user-facing scenario benefits from professional branding.

**Cross-Scenario Integration**: Provides to marketing-automation (brand asset library), e-commerce-builder (complete brand identity), website-builder (brand-consistent themes). Consumes from competitor-monitor (competitor brand analysis), market-research (industry trend data).

**Performance Targets**: Brand generation time < 90 seconds for complete identity package. Asset quality > 95% approval rate for generated assets. Integration speed < 5 minutes to apply brand to existing app. Resource usage < 4GB memory, < 50% CPU during generation.

**Reference Documentation**: README.md (user-facing overview and quick start), docs/api.md (complete API specification with examples), docs/ui.md (UI component documentation and styling guide), initialization/automation/n8n/ (N8n workflow definitions), initialization/configuration/comfyui-workflows/ (ComfyUI asset generation workflows).
