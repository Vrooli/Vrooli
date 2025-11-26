# Product Requirements Document (PRD)

> **Version**: 1.0.0
> **Last Updated**: 2025-11-18
> **Status**: Canonical Specification
> **Source of Truth**: PRD Control Tower

## 🎯 Overview

App Personalizer enables dynamic customization of generated Vrooli scenarios using AI-powered modifications, digital twin personas, and brand assets. It transforms generic applications into personalized, white-labeled, or tenant-specific versions while maintaining functionality through automated validation and testing.

**Purpose**: Add permanent capability for application-level personalization and white-labeling across all Vrooli scenarios. Enable agents and users to apply persona-driven customizations, brand themes, and behavioral modifications to any generated application without manual code intervention.

**Primary Users & Verticals**:
- **Internal**: Agents automating personalization for multi-tenant SaaS scenarios
- **Business**: Companies requiring white-labeled versions of Vrooli scenarios
- **End Users**: Individuals seeking persona-based customization of their applications
- **Scenarios**: Other scenarios needing tenant-specific versions or branding

**Deployment Surfaces**:
- **API**: REST endpoints for personalization orchestration
- **CLI**: Command-line interface for local operations
- **Library**: Importable Go package for other scenarios
- **Note**: Orchestration now relies on API/CLI flows; no external workflow engine is required.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Core personalization API | POST /api/personalize accepts app path, persona ID, and brand ID; returns personalized app version
- [ ] OT-P0-002 | Backup and versioning system | Creates timestamped backups before modifications; supports rollback to previous versions
- [ ] OT-P0-003 | Digital twin integration | Fetches persona data from personal-digital-twin scenario; applies preferences to app configuration
- [ ] OT-P0-004 | Brand asset application | Applies colors, fonts, logos from brand-manager; updates theme files and static assets
- [ ] OT-P0-005 | AI-powered code modification | Uses Claude Code or Ollama to intelligently modify source files based on customization intent
- [ ] OT-P0-006 | Validation and testing | Runs app tests post-modification; ensures critical functionality remains intact

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Multi-deployment mode support | Implements copy, patch, and multi-tenant deployment strategies
- [ ] OT-P1-002 | Personalization history tracking | Stores modification history in PostgreSQL; enables analytics and auditing
- [ ] OT-P1-003 | CLI orchestration commands | Provides app-personalizer personalize, brand, list-personalizations commands
- [ ] OT-P1-004 | Orchestration templates | Pre-built API/CLI pipelines for common personalization patterns
- [ ] OT-P1-005 | MinIO artifact storage | Stores app versions, backups, and modification artifacts in object storage

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | A/B testing framework | Compare personalized versions; measure engagement and preference metrics
- [ ] OT-P2-002 | Collaborative personalization | Multiple stakeholders review and approve customizations before deployment
- [ ] OT-P2-003 | Visual preview system | Screenshot-based diff comparison before applying modifications
- [ ] OT-P2-004 | Personalization marketplace | Share and monetize custom themes, personas, and modification templates

## 🧱 Tech Direction Snapshot

**Architecture**:
- **API**: Go-based REST service using Go Fiber framework
- **Storage**: PostgreSQL for metadata; MinIO for app versions and artifacts
- **AI Integration**: Claude Code primary; Ollama (llama3.2, codellama) fallback
- **Orchestration**: Direct API/CLI pipelines (no external workflow engine)
- **CLI**: Shell-based command interface wrapping API calls

**Personalization Layers**:
1. **UI Theme**: Colors, fonts, logos, component styles
2. **Content**: Text defaults, prompts, welcome messages, help documentation
3. **Behavior**: AI personality, interaction patterns, feature toggles
4. **Structure**: Menu items, navigation, module visibility

**Integration Strategy**:
- Consumes digital twin data via personal-digital-twin API
- Fetches brand assets via brand-manager API
- Provides personalization API for other scenarios to call
- Stores results accessible to deployment scenarios

**Non-Goals**:
- Runtime personalization (modifications happen at build/deploy time)
- User-level customization within apps (focus is app-level personalization)
- Source code generation from scratch (modify existing scenarios only)

## 🤝 Dependencies & Launch Plan

**Required Resources**:
- **Ollama**: AI model inference for code modifications (llama3.2, codellama)
- **PostgreSQL**: Metadata, history, and personalization configuration storage

**Optional Resources**:
- **Claude Code**: Preferred AI for intelligent code modifications (fallback to Ollama)
- **MinIO**: Object storage for app versions and large artifacts (fallback to filesystem)

**Scenario Dependencies**:
- **personal-digital-twin**: Provides persona data for personalization (graceful degradation if unavailable)
- **brand-manager**: Supplies brand assets and themes (optional, uses defaults if missing)

**Launch Sequence**:
1. Phase 1: Core API + digital twin integration + basic CLI
2. Phase 2: Brand manager integration + validation framework
3. Phase 3: MinIO storage + orchestration templates (API/CLI first)
4. Phase 4: Multi-deployment modes + history tracking

**Risks**:
- AI modifications may break functionality → Mitigate with comprehensive test validation
- Persona/brand data unavailable → Provide sensible defaults and graceful fallbacks
- Large app modification timeouts → Implement streaming progress updates and async processing

## 🎨 UX & Branding

**Visual Palette**: Professional business application aesthetic with clean, modern interface. Neutral color scheme (grays, blues) with accent colors from Vrooli brand. Focus on clarity and functionality over visual flair.

**Tone & Voice**: Technical and precise when describing modifications. Friendly and encouraging when guiding users through personalization workflows. Transparent about AI-driven changes and validation results.

**Accessibility Targets**:
- WCAG 2.1 Level AA compliance for CLI output formatting
- Clear error messages with actionable remediation steps
- Progress indicators for long-running personalization operations
- Support for screen readers in any future UI components

**Experience Promise**: Users should feel confident that personalizations are safe, reversible, and validated. The system communicates clearly about what will change, provides preview opportunities, and ensures backups exist before modifications.

## 📎 Appendix

**Reference Materials**:
- Digital Twin Specification: scenarios/personal-digital-twin/PRD.md
- Brand Manager API: scenarios/brand-manager/README.md
- Deployment Patterns: docs/deployment/README.md

**Personalization Type Examples**:
- **SaaS White-Label**: Rebrand entire app for enterprise customer with their logo, colors, domain
- **Persona Adaptation**: Adjust AI assistant personality based on user's communication preferences
- **Feature Localization**: Enable/disable modules based on regional requirements or user tier
- **Content Customization**: Replace default examples and help text with domain-specific content
