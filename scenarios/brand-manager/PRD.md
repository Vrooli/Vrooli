# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Manages the full branding lifecycle for all Vrooli scenarios — generating, storing, applying, and validating brand identity. Replaces the deleted Brand Manager and App Personalizer scenarios with a clean rewrite.
- **Primary users/verticals**: Human designers (via UI wizard) and autonomous agents (via CLI/API) equally. Any Vrooli scenario owner needing consistent, professional branding.
- **Deployment surfaces**: CLI, API, UI (React dashboard + wizard)
- **Value promise**: Eliminates ad-hoc branding across scenarios, ensures monetization readiness with professional identity, and provides a reusable brand library that compounds across the ecosystem.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Brand CRUD | Create, read, update, and version brands with identity, visuals, colors, typography, voice, and notes via API and CLI
- [ ] OT-P0-002 | SQLite Storage | Single SQLite DB with tables for brands, brand_versions, assignments, and assets. Asset files stored at `~/.vrooli/brand-manager/assets/{brand_id}/`
- [ ] OT-P0-003 | AI Generation | Ollama-first with OpenRouter fallback for text (palette, typography, copy) and image (logo, favicon, icon) generation using AIProviderChain pattern
- [ ] OT-P0-004 | Programmatic Application | Apply brand elements to scenarios via CSS custom properties with `/* brand-manager:<element> */` markers, manifest.json with `_brand` keys, favicon paths, static assets
- [ ] OT-P0-005 | Design Language File | Generate `docs/DESIGN_LANGUAGE.md` per-scenario during apply from structured brand data + user notes
- [ ] OT-P0-006 | Scenario Auditor Integration | HTTP provider (`external_rules_brand_manager.go`) reporting branding compliance rules: has-logo, has-favicon, has-color-system, has-display-name, has-typography
- [ ] OT-P0-007 | Inline Validation Scanner | Grep for `brand-manager:` prefixed CSS comments and `_brand` JSON keys to validate branding is actually applied in code
- [ ] OT-P0-008 | Discovery Scanner | Scan existing scenario state (service.json, theme files, static assets, manifests, .vrooli/branding.json) and auto-populate draft brands with confidence scores
- [ ] OT-P0-009 | Brand Assignment | Link brands to scenarios, track what was applied, when, and what version. Support partial application (individual elements)
- [ ] OT-P0-010 | CLI Surface | Commands: create, list, get, update, generate, discover, apply, status
- [ ] OT-P0-011 | REST API | Resource-centric endpoints: /brands, /brands/{id}/versions, /assignments, /assets/{id}, /scenarios/{name}/status, /standards
- [ ] OT-P0-012 | WCAG AA Contrast | Validate contrast for defined color pairings (primary-on-background, text-on-surface) during generation, reject non-compliant pairings

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Agent-Assisted Application | Spawn sandboxed agents via agent-manager API for complex/non-standard scenario integrations with mandatory inline marker constraint
- [ ] OT-P1-002 | UI Dashboard | Scenario-centric view of branding status across all scenarios with brand library browsing and search
- [ ] OT-P1-003 | UI Wizard | Brand creation/editing with live preview, generation option selection, and application preview before applying
- [ ] OT-P1-004 | Dark/Light Theme Preview | Preview brand application in both theme modes before committing

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Framework-Extensible Scanning | Scanner plugins per framework/language beyond CSS + JSON (e.g., Tailwind config, SCSS)
- [ ] OT-P2-002 | Lighthouse Integration | test-genie lighthouse audits for supplemental WCAG accessibility checking on applied branding

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go (API + CLI), React + Vite (UI), SQLite (metadata), filesystem (assets)
- Data + storage expectations: Single SQLite DB at `~/.vrooli/brand-manager/brand-manager.db`, asset files at `~/.vrooli/brand-manager/assets/{brand_id}/`, atomic writes (temp file + rename)
- Integration strategy: Scenario Auditor HTTP provider pattern (like test-genie, tidiness-manager). AIProviderChain pattern from web-console. Agent-manager API for agentic tasks. Storage via `api-core/storage` module.
- Non-goals / guardrails: No auto-push of brand updates (pull model only). No ComfyUI or local image pipelines. No direct writes to scenario branding from outside Brand Manager. No MinIO. No service.json schema modifications for branding. No digital twin/behavioral personalization. No multi-tenant white-labeling. No A/B testing or analytics.

## 🤝 Dependencies & Launch Plan
- Required resources: SQLite (declared in service.json), Ollama (OLLAMA_URL), OpenRouter (OPENROUTER_API_KEY)
- Scenario dependencies: scenario-auditor (for validation provider registration), agent-manager (for agent-assisted application), web-console (reference for AIProviderChain pattern)
- Operational risks: OpenRouter image quality for logos (mitigated by configurable models + user upload override). Agent-assisted changes must use inline markers (hard constraint, fail rather than degrade). CSS markers could be stripped by aggressive minification (document supported configs).
- Launch sequencing: Phase 1 (Foundation) → Phase 2 (Discovery + Validation) → Phase 3 (Generation) → Phase 4 (Application) → Phase 5 (UI Wizard). Each phase has gate checks.

## 🎨 UX & Branding
- Look & feel: Follows Vrooli theming conventions — CSS custom properties with RGB space-separated values, `data-resolved-theme` attribute switching, dark/light mode support
- Accessibility: WCAG AA contrast validation for defined color pairings during brand generation. Lighthouse audits via test-genie for applied scenarios
- Voice & messaging: Professional, practical, tool-oriented. Serves both designers and agents equally
- Branding hooks: Brand Manager is itself a brandable scenario. Logo, favicon, color system, typography all manageable through its own system

## 📎 Appendix
- Existing branding reference: LPBS `.vrooli/branding.json` format (site_name, tagline, theme colors, logo URLs, SEO)
- Key reference files: web-console `ai_provider.go` (AIProviderChain), scenario-auditor `external_rules.go` (provider registration), swarm-manager `json_store.go` (atomic writes)
- Workshop rounds: 4 rounds covering AI provider strategy, validation markers, SQLite schema, application workflow, testing strategy, acceptance scope
