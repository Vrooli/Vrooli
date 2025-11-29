# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

**Purpose**: Landing Manager is a meta-scenario that **only** registers templates, generates landing-page scenarios, and orchestrates agent handoff. It is **not** a landing page and does **not** ship admin portals, A/B testing, analytics, or Stripe runtime; those live in the landing-page template payload that Landing Manager copies into generated scenarios. Legacy landing/admin UI code in this repo is reference-only; runtime ships from the template payload.

**Primary users/verticals**:
- SaaS founders launching products (Vrooli Pro, future bundles)
- Entrepreneurs validating product-market fit with A/B testing
- Marketing teams building conversion-optimized landing pages
- Developers deploying monetized Vrooli scenarios

**Deployment surfaces**:
- CLI: `landing-manager` commands for template listing/showing/generation and agent-handoff triggers
- API: REST endpoints for programmatic template inspection and scenario generation
- UI: Factory dashboard only (template links + generation guidance). No landing/admin runtime here.
- Agent Integration: Triggers an agent to customize **generated** landing pages using template-safe APIs/files

> Scope note: All landing runtime features (admin portal, A/B testing, metrics, Stripe, subscription verification) belong to the template at `scripts/scenarios/templates/landing-page-react-vite` and the scenarios generated from it. The factory's responsibility stops at generating and orchestrating.

**Value promise**: Reduces landing page setup from weeks to minutes by shipping a factory + template bundle. Factory handles metadata and generation; templates provide the landing runtime once generated.

## 🎯 Operational Targets
> Runtime landing/admin targets live in `scripts/scenarios/templates/landing-page-react-vite/PRD.md` and are validated in generated scenarios.

### 🔴 P0 – Must ship for viability
- [x] **OT-P0-001: Template Registry & Discovery** — Expose templates via CLI/API with metadata for listing, inspection, and selection (availability, metadata quality, multi-template support)
- [x] **OT-P0-002: Scenario Generation Pipeline** — Generate runnable landing-page scenarios with single command, proper output structure, provenance stamping, and validation
- [x] **OT-P0-003: Agent Integration & Customization** — Trigger agent-based customization with structured briefs and predefined personas

### 🟠 P1 – Should have post-launch
- [x] **OT-P1-001: Generation Workflow Enhancements** — Add preview links, dry-run planning, generation diagnostics, UI-based lifecycle management (start/stop/logs/access generated scenarios without terminal), and UI-based promotion from staging to production

### 🟢 P2 – Future / expansion
- [ ] **OT-P2-001: Ecosystem Expansion** — Template marketplace, migration tooling, and factory-level analytics

## 🧱 Tech Direction Snapshot

**Preferred stacks**:
- **Landing page template**: React + TypeScript + Vite + TailwindCSS + shadcn + Lucide icons (lives in template payload)
- **Factory scenario API**: Go (template registry, generation orchestration)
- **Generated landing page APIs**: Go (Gin framework for REST, direct PostgreSQL) – shipped inside template payload

**Data + storage expectations**:
- **postgres**: Required for each generated landing page (content, A/B variants, metrics, subscriptions)
- **redis** (optional): Session caching for high-traffic landing pages
- **qdrant** (optional): Semantic search for template recommendations

**Integration strategy**:
- **Vrooli CLI**: Primary interface for template listing and scenario generation
- **Resource CLI** (claude-code): Agent customization via structured prompts
- **Direct API**: Programmatic landing page creation for advanced users

**Non-goals**:
- Not a general-purpose website builder (focused on landing pages)
- Not a CMS (content is API-driven, not file-based)
- Not a design tool (relies on templates + agent customization)
- Not a hosting platform (users deploy to their infrastructure)

## 🤝 Dependencies & Launch Plan

**Required resources**:
- **postgres**: Essential for all generated landing pages (content, variants, metrics, subscriptions)
- **app-issue-tracker**: Agent-based customization orchestration (issue filing + investigation trigger)

**Optional resources**:
- **redis**: Session caching for high-traffic scenarios
- **qdrant**: Template recommendations based on user intent
- **browserless**: Screenshot generation for template previews

**Scenario dependencies**:
- **scenario-authenticator** (future): Multi-tenant admin portal support
- **funnel-builder** (future): Embed conversion funnels in landing pages
- **referral-program-generator** (future): Add referral programs to landing pages

**Operational risks**:
- **Template maintenance**: Templates may become outdated; mitigate with versioning and migration guides
- **Agent quality**: AI-generated customizations may be poor; mitigate with preview + rollback
- **Stripe API changes**: Payment flows may break; mitigate with versioned SDK and monitoring

**Launch sequencing**:
1. **Phase 1 (P0)**: Single template (`saas-landing-page`), basic A/B testing, Stripe integration
2. **Phase 2 (P1)**: Performance optimization, accessibility compliance, video sections
3. **Phase 3 (P2)**: Multi-template support, advanced analytics, agent-generated variants

## 🎨 UX & Branding

**Look & feel**:
- **Public landing pages**: Bold, distinctive design per the `<frontend_aesthetics>` guidelines. Avoid generic "AI slop" aesthetics (Inter font, flat colors, low contrast). Use custom typography, layered gradients, high-contrast color schemes, and asymmetric layouts.
- **Admin portal**: Clean, professional SaaS dashboard aesthetic. Light theme by default, dark theme optional. Focus on clarity and efficiency (metrics-first UI).

**Accessibility**:
- WCAG 2.1 Level AA compliance target (≥ 90 Lighthouse score)
- Keyboard navigation for all interactive elements
- High color contrast ratios (4.5:1 for text, 3:1 for UI components)
- Screen reader-friendly labels and ARIA attributes

**Voice & messaging**:
- **Public landing pages**: Confident, persuasive, user-focused. Tone varies per customization (e.g., professional for B2B, friendly for B2C).
- **Admin portal**: Concise, action-oriented, data-driven. Tone: "Here's what's working, here's what to optimize."

**Branding hooks**:
- **Logo**: Customizable per landing page (agent can replace with user-provided logo)
- **Colors**: CSS variables for primary, secondary, accent colors (agent can override)
- **Typography**: Custom font stack per landing page (not default web fonts)
- **Iconography**: Lucide icons by default (consistent with Vrooli ecosystem)
