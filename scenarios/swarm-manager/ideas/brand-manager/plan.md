# Implementation Plan: Brand Manager – Full Branding Lifecycle for All Scenarios

## Purpose
A single scenario that manages the full branding lifecycle for all Vrooli scenarios — generating, storing, applying, and validating brand identity. Replaces the deleted Brand Manager and App Personalizer scenarios with a clean rewrite. Serves both human designers (via UI wizard) and autonomous agents (via CLI/API) equally.

## Problem Statement
Vrooli scenarios currently have no consistent branding. Each scenario handles its own logos, colors, favicons, and identity ad-hoc (if at all). There is no central place to create, version, assign, or validate brand identity across the scenario ecosystem. This makes scenarios look unprofessional and blocks monetization readiness.

## Scope
### In Scope
- Brand CRUD: create, read, update, version brands (identity, visuals, colors, typography, voice, notes)
- AI generation: use OpenRouter for logo concepts, palette generation, typography pairing, copy
- Asset storage: generated files (SVG, PNG, ICO, etc.) in Brand Manager's data directory
- Assignment: link brands to scenarios, track what was applied and when
- Application (two-tier): programmatic for standard patterns, agent-assisted (via agent-manager) for complex integrations
- Validation: HTTP provider for Scenario Auditor with extensible framework-aware scanning
- Discovery: scan existing scenario state to auto-populate draft brands
- Design language file: markdown document generated per-scenario for agent consumption
- All three surfaces: UI (React), CLI, REST API
- Opt-out mechanism for test-only scenarios
- SQLite for metadata + filesystem for binary assets

### Out of Scope
- Digital twin / behavioral personalization
- Multi-tenant white-labeling
- N8n orchestration, ComfyUI
- A/B testing, analytics dashboards, ML optimization
- Auto-applying on brand update (push model)
- Database query optimization or backup/DR
- Custom font hosting (use standard web fonts)

## Current Technical Context

### Established Patterns (from codebase exploration)
- **Storage**: Vrooli storage hierarchy — resources declared in `service.json`, data at `~/.vrooli/brand-manager/`, schema initialization in `initialization/storage/` directory. Use `api-core/storage` for filesystem runtime state.
- **Scenario Auditor providers**: External HTTP provider pattern — implement `externalRuleProvider` interface with `ID()`, `Name()`, `Rules()`, `Run()`. Register via `init()` + `registerExternalProvider()`. Examples: test-genie (30s timeout), tidiness-manager (2min timeout).
- **Agent spawning**: agent-manager API manages lifecycle with sandboxing, workspace isolation, and cost tracking.
- **Theming**: CSS custom properties pattern (e.g., swarm-manager's `rgb(var(--slate-50) / <alpha-value>)` in Tailwind config). Runtime switching via data attributes on document root.
- **File storage**: Atomic write pattern (temp file + rename) for crash-safe writes, as seen in swarm-manager's `json_store.go`.

### Key Files
- `.vrooli/schemas/service.schema.json` — master service schema (no branding fields currently)
- `scenarios/scenario-auditor/api/external_rules.go` — provider registration pattern
- `scenarios/swarm-manager/api/internal/storage/json_store.go` — atomic file write pattern
- `scenarios/swarm-manager/ui/tailwind.config.ts` — CSS custom property theme pattern

## Target End State

### Two-Layer Branding Architecture
1. **Design Language File** (`docs/DESIGN_LANGUAGE.md` in each branded scenario): A markdown document that serves as a skill/reference for agents working on the scenario. Contains abstract design guidance — tone, visual metaphors, patterns to follow/avoid, personality. This is human-readable and agent-consumable context, not structured config.

2. **Brand Manager DB + Asset Storage**: The structured source of truth for concrete branding — hex colors, font names, logo files, favicon files, assignment records, version history. Brand Manager owns this data and serves it via API/CLI.

### Validation via Scanning
Brand Manager validates that branding is applied by scanning scenarios for evidence:
- **Standard markers**: Known file paths (e.g., `public/favicon.ico`, `public/logo.svg`), CSS custom property declarations, manifest.json fields, service.json metadata fields
- **Framework-extensible**: Scanner plugins per framework/language so future scenarios in different stacks can be validated
- **Convention-based**: Branded theme declarations use a marker comment or live in a known file (e.g., `src/theme/brand.css`) so the scanner can find them reliably

### System Behavior
- Brands live in Brand Manager's library, assignable to multiple scenarios
- Application is explicit (user/agent triggers it), partial application supported
- Scenario Auditor queries Brand Manager via HTTP provider pattern to check compliance
- Both UI wizard and CLI provide full generation/application/validation workflows
- 99% of scenarios are expected to have branding; test-only scenarios opt out via service.json field

## Implementation Strategy
<!-- TBD — pending round 2 decisions on data model, API design, and phasing -->

## Contract Decisions

### Settled (Round 1)
| Decision | Choice | Rationale |
|----------|--------|-----------|
| AI provider | OpenRouter only | Single dependency, covers both text and image generation |
| Agent spawning | agent-manager API | Consistent with existing lifecycle management, sandboxing, cost tracking |
| Asset storage | Filesystem (storage-steer conventions) | `~/.vrooli/brand-manager/assets/`, using `api-core/storage` module |
| Branding data location | Hybrid: design language file + Brand Manager DB + scanning | Agents get context via docs, structured data in DB, validation via scanning |
| MVP scope | Full spec (all surfaces, all tiers) | Priority 2 but complete implementation needed |
| Scenario Auditor integration | HTTP provider pattern | Consistent with test-genie, tidiness-manager patterns |
| Primary user | Both human and agent equally | UI wizard and CLI are both critical paths |

## Testing Plan
<!-- TBD -->

## Rollout / Validation Checklist
<!-- TBD -->

## Risks + Mitigations
<!-- TBD -->

## Non-goals / Prohibited Patterns
- No auto-push of brand updates to scenarios (pull model only)
- No ComfyUI or local image generation pipelines
- No direct writes to service.json branding fields from outside Brand Manager
- No MinIO/object storage dependency

## Definition of Done
<!-- TBD -->
