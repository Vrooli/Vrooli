# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: Central command center to create, validate, and publish PRDs across all Vrooli scenarios and resources
Target users: Vrooli operators, scenario owners, and AI agents performing generation/improvement
Deployment surfaces: Web UI (catalog + editor), REST API (catalog/drafts/targets/requirements), CLI tool
Value proposition: Standardize PRD creation while reducing authoring overhead and enabling reliable scenario generation through template compliance

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Ecosystem PRD catalog | List scenarios/resources with PRD/draft/violation status
- [ ] OT-P0-002 | Draft lifecycle + publish | Create/edit/save/validate/publish PRD drafts safely with atomic writes
- [ ] OT-P0-003 | Requirements + targets coverage | Parse requirements registries and show operational-target linkage gaps

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Backlog intake + convert | Capture freeform ideas and convert them into PRD drafts
- [ ] OT-P1-002 | AI assistance | Generate/rewrite PRD content with model choice and safe apply via diff

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Cached validation results | Reuse scenario-auditor validation results until content changes

## 🧱 Tech Direction Snapshot
Preferred stacks: Go-based API backend, React + TypeScript frontend using Vite, Go CLI implementation
Preferred storage: PostgreSQL for metadata/validation cache, filesystem for PRD drafts and published content
Integration strategy: API-first design with thin UI/CLI clients, scenario-auditor for validation, optional OpenRouter AI integration
Non-goals: Direct scenario execution (must use lifecycle), automated PRD changes without review, custom UI theming in v1

## 🤝 Dependencies & Launch Plan
Required resources: PostgreSQL instance for metadata storage
Scenario dependencies: scenario-auditor for validation, optional resource-openrouter/OpenRouter for AI features
Operational risks: Draft corruption during publish, validation service availability, AI response latency and costs
Launch sequencing: Template compliance validation → requirement linkage tracking → draft lifecycle workflows → AI assistance

## 🎨 UX & Branding
User experience: Dashboard-style interface optimized for quick status scanning and draft management with clear next actions
Visual design: Leverage existing Vrooli UI components and styling patterns, maintain consistent look and feel
Accessibility: Full keyboard navigation support, WCAG 2.1 compliant contrast ratios, clear focus indicators, confirmation dialogs for destructive actions