# Implementation Plan: Brand Manager – Full Branding Lifecycle for All Scenarios

## Purpose
A single scenario that manages the full branding lifecycle for all Vrooli scenarios — generating, storing, applying, and validating brand identity. Replaces the deleted Brand Manager and App Personalizer scenarios with a clean rewrite. Serves both human designers (via UI wizard) and autonomous agents (via CLI/API) equally.

## Required Reading
```bash
prompt-manager skill read cli-steer api-steer storage-steer unit-testing-architecture-steer seam-discovery-and-enforcement
```

## Problem Statement
Vrooli scenarios currently have no consistent branding. Each scenario handles its own logos, colors, favicons, and identity ad-hoc (if at all). There is no central place to create, version, assign, or validate brand identity across the scenario ecosystem. This makes scenarios look unprofessional and blocks monetization readiness.

## Scope
### In Scope
- Brand CRUD: create, read, update, version brands (identity, visuals, colors, typography, voice, notes)
- AI generation: Ollama-first with OpenRouter fallback for text/image generation; agent-manager API for agentic tasks
- Asset storage: generated files (SVG, PNG, ICO, etc.) in Brand Manager's data directory
- Assignment: link brands to scenarios, track what was applied and when
- Application (two-tier): programmatic for standard patterns, agent-assisted (via agent-manager) for complex integrations
- Validation: HTTP provider for Scenario Auditor with extensible framework-aware inline scanning using `brand-manager:` comment markers
- Discovery: scan existing scenario state to auto-populate draft brands (including LPBS .vrooli/branding.json format)
- Design language file: `docs/DESIGN_LANGUAGE.md` generated per-scenario during apply for agent consumption
- All three surfaces: UI (React), CLI, REST API
- Opt-out mechanism tracked internally by Brand Manager (no service.json changes)
- SQLite for metadata + filesystem for binary assets
- WCAG AA contrast validation for defined color pairings during generation, plus test-genie lighthouse audits for applied UI accessibility

### Out of Scope
- Digital twin / behavioral personalization
- Multi-tenant white-labeling
- N8n orchestration, ComfyUI
- A/B testing, analytics dashboards, ML optimization
- Auto-applying on brand update (push model)
- Database query optimization or backup/DR
- Custom font hosting (use standard web fonts)
- Exhaustive WCAG validation of all arbitrary color combinations in applied UI

## Current Technical Context

### Established Patterns (from codebase exploration)
- **Storage**: Vrooli storage hierarchy — resources declared in `service.json`, data at `~/.vrooli/brand-manager/`, schema initialization in `initialization/storage/` directory. Use `api-core/storage` for filesystem runtime state. Single SQLite DB at `~/.vrooli/brand-manager/brand-manager.db`.
- **SQLite init**: Pattern from system-monitor — schema as Go string constant, `CREATE TABLE IF NOT EXISTS`, WAL mode + pragmas, executed on repository init.
- **Scenario Auditor providers**: External HTTP provider pattern — implement `externalRuleProvider` interface with `ID()`, `Name()`, `Rules()`, `Run()`. Register via `init()` + `registerExternalProvider()`. Examples: test-genie (30s timeout), tidiness-manager (2min timeout). Provider file: `scenario-auditor/api/external_rules_brand_manager.go`.
- **Agent spawning**: agent-manager API manages lifecycle with sandboxing, workspace isolation, and cost tracking.
- **Theming**: CSS custom properties with RGB space-separated values (`--slate-50: 248 250 252`) and `data-resolved-theme` attribute switching. Tailwind config references via `rgb(var(--slate-XX) / <alpha-value>)`.
- **File storage**: Atomic write pattern (temp file + rename) for crash-safe writes, as seen in swarm-manager's `json_store.go`.
- **Existing branding**: LPBS has `.vrooli/branding.json` with schema validation — includes site_name, tagline, theme colors (hex), logo URLs, SEO fields. Brand Manager's data model should be a superset; discovery can auto-import this format.
- **AI provider chain**: web-console's `AIProviderChain` pattern — interface-based `AIProvider` with `Name()` and `Generate()`, tried sequentially. Ollama as primary (localhost:11434), OpenRouter as fallback (requires `OPENROUTER_API_KEY`). stream-of-consciousness-analyzer uses a two-tier priority system with `Fallback` flag. Brand Manager should follow the web-console `AIProviderChain` pattern.

### Key Files
- `.vrooli/schemas/service.schema.json` — master service schema
- `scenarios/scenario-auditor/api/external_rules.go` — provider registration pattern
- `scenarios/scenario-auditor/api/external_rules_tidiness_manager.go` — example provider
- `scenarios/swarm-manager/api/internal/storage/json_store.go` — atomic file write pattern
- `scenarios/swarm-manager/ui/tailwind.config.ts` — CSS custom property theme pattern
- `scenarios/swarm-manager/ui/src/styles.css` — dark/light theme switching pattern
- `scenarios/landing-page-business-suite/.vrooli/branding.json` — existing branding data format
- `scenarios/landing-page-business-suite/.vrooli/schemas/branding.schema.json` — existing branding schema
- `scenarios/web-console/api/ai_provider.go` — AIProvider interface + chain pattern (reference implementation)
- `scenarios/stream-of-consciousness-analyzer/api/suggestion_service.go` — two-tier Ollama/OpenRouter priority

## Target End State

### Two-Layer Branding Architecture
1. **Design Language File** (`docs/DESIGN_LANGUAGE.md` in each branded scenario): A markdown document generated by Brand Manager during "apply" from structured brand data + user notes. Serves as a skill/reference for agents working on the scenario. Contains abstract design guidance — tone, visual metaphors, patterns to follow/avoid, personality. Human-readable and agent-consumable.

2. **Brand Manager DB + Asset Storage**: The structured source of truth for concrete branding — hex colors, font names, logo files, favicon files, assignment records, version history. Single SQLite DB with tables for brands, versions, assignments, and asset metadata. Asset files at `~/.vrooli/brand-manager/assets/{brand_id}/`.

### Validation via Inline Scanning
Brand Manager validates that branding is applied by scanning scenarios for inline evidence in actual code:
- **Inline markers**: CSS comment markers (`/* brand-manager:<element> */` before declarations), JSON `_brand` keys. Scanner greps for `brand-manager:` prefix.
- **Multi-source validation**: Some brand elements require checking multiple places (e.g., CSS + manifest.json), yielding partial validation status
- **Framework-extensible**: Scanner plugins per framework/language. Start with CSS custom properties + JSON manifests. Add more as scenarios expand to new stacks
- **Convention-based**: Branded declarations use markers so the scanner finds them reliably — this is ground truth against what the code actually does
- **Progressive improvement**: Over time, add better standards and validation approaches for specific branding decisions
- **Lighthouse integration**: test-genie lighthouse audits provide supplemental WCAG accessibility checking for applied branding in UI scenarios

### AI Provider Architecture
- **Text generation** (palette suggestions, typography pairing, tagline/copy, design language prose): Ollama-first with OpenRouter fallback, following web-console's `AIProviderChain` pattern
- **Image generation** (logos, favicons, app icons, og-images): Ollama-first with OpenRouter fallback, using configurable model IDs
- **Agentic tasks** (agent-assisted application for complex scenarios): agent-manager API for spawning sandboxed Claude Code agents
- **Configuration**: Model IDs stored in Brand Manager config, with sensible defaults. User/agent can override per-brand or globally. Environment variables: `OLLAMA_URL`, `OPENROUTER_API_KEY`, provider-specific model selectors

### System Behavior
- Brands live in Brand Manager's library, assignable to multiple scenarios
- Application is explicit (user/agent triggers it), partial application supported
- Scenario Auditor queries Brand Manager via HTTP provider pattern to check compliance
- Both UI wizard and CLI provide full generation/application/validation workflows
- 99% of scenarios are expected to have branding; test-only scenarios opt out (tracked internally by Brand Manager)
- Colors generated with WCAG AA contrast validation for defined pairings

## Implementation Strategy

### Phase 1: Foundation (SQLite schema + storage + CRUD API + CLI basics)
**Deliverables:**
- SQLite schema: brands, brand_versions, assignments, assets tables
- Repository layer with CRUD operations (in-memory SQLite for tests)
- REST API: `/api/v1/brands`, `/api/v1/brands/{id}/versions`, `/api/v1/assignments`, `/api/v1/assets/{id}`
- CLI commands: `brand-manager create`, `brand-manager list`, `brand-manager get`, `brand-manager update`
- Asset file storage at `~/.vrooli/brand-manager/assets/{brand_id}/`
- Opt-out tracking (internal list of excluded scenario names)
- `service.json` with SQLite resource declaration
- AI provider chain: `AIProvider` interface with `OllamaProvider` and `OpenRouterProvider` implementations

**Key files:**
- `scenarios/brand-manager/api/` — Go API server
- `scenarios/brand-manager/cli/` — Go CLI
- `scenarios/brand-manager/initialization/storage/` — SQLite schema

### Phase 2: Discovery + Validation (Scenario Auditor provider)
**Deliverables:**
- Discovery scanner: scan scenario directories for existing branding state (service.json, theme files, static assets, manifests, .vrooli/branding.json)
- Auto-populate draft brand from discovered state
- Scenario Auditor external rules provider (`external_rules_brand_manager.go`)
- `/api/v1/standards` endpoint for auditor queries
- `/api/v1/scenarios/{name}/status` endpoint for per-scenario branding status
- Inline marker scanner: grep for `brand-manager:` prefixed comments in CSS, `_brand` keys in JSON
- Validation rules: has-logo, has-favicon, has-color-system, has-display-name, has-typography
- CLI: `brand-manager discover {scenario}`, `brand-manager status {scenario}`

### Phase 3: Generation (Ollama + OpenRouter integration)
**Deliverables:**
- AI provider chain with Ollama-first, OpenRouter fallback (following web-console pattern)
- Configurable model IDs in Brand Manager config with sensible defaults
- Logo generation: prompt-based concepts → user selection → rasterize to multiple sizes
- Palette generation: primary/secondary/accent + semantic colors + dark/light variants
- WCAG AA contrast validation for defined color pairings during generation (primary-on-background, text-on-surface, etc.)
- Typography pairing suggestions
- Tagline/copy generation incorporating user notes
- CLI: `brand-manager generate {brand_id}` with interactive option selection
- Generation options stored as drafts until user confirms

### Phase 4: Application (programmatic + agent-assisted)
**Deliverables:**
- Programmatic application tier: CSS custom properties with `/* brand-manager:<element> */` markers, manifest.json with `_brand` keys, favicon paths, static asset dirs
- Design language file generation: `docs/DESIGN_LANGUAGE.md` from brand data + notes (LLM-generated prose)
- Agent-assisted application: spawn agent via agent-manager API for complex/non-standard scenarios
- Agent instructions include requirement that all changes must use inline markers for programmatic validation
- Partial application: apply individual brand elements independently
- CLI: `brand-manager apply {brand_id} --scenario {name}` with `--elements` filter

### Phase 5: UI Wizard
**Deliverables:**
- React dashboard: scenario-centric view of branding status across all scenarios
- Brand creation/editing wizard with live preview
- Generation UI: present options, allow refinement, iterate
- Application UI: preview before applying, element-level selection
- Brand library: browse, search, assign brands to scenarios
- Dark/light theme preview

## Contract Decisions

### Settled (Round 1)
| Decision | Choice | Rationale |
|----------|--------|-----------|
| AI provider | OpenRouter only | Single dependency, covers both text and image generation |
| Agent spawning | agent-manager API | Consistent with existing lifecycle management, sandboxing, cost tracking |
| Asset storage | Filesystem (`~/.vrooli/brand-manager/assets/`) | Using `api-core/storage` module, storage-steer conventions |
| Branding data location | Hybrid: design language file + Brand Manager DB + inline scanning | Agents get context via docs, structured data in DB, validation via inline code evidence |
| MVP scope | Full spec (all surfaces, all tiers) | Priority 2 but complete implementation needed |
| Scenario Auditor integration | HTTP provider pattern | Consistent with test-genie, tidiness-manager patterns |
| Primary user | Both human and agent equally | UI wizard and CLI are both critical paths |

### Settled (Round 2)
| Decision | Choice | Rationale |
|----------|--------|-----------|
| SQLite schema | Single database with tables: brands, brand_versions, assignments, assets | Simple, queryable, one file to back up, all cross-brand queries natively supported |
| Design language file | Brand Manager generates `docs/DESIGN_LANGUAGE.md` during apply | Stays in sync with brand; LLM-generated prose from structured data + user notes |
| Validation markers | Inline evidence in actual code (ground truth approach) | Code markers are ground truth — config files can drift. Extensible per framework. Partial validation status for multi-source checks |
| REST API structure | Resource-centric: /brands, /brands/{id}/versions, /assignments, /assets/{id}, /scenarios/{name}/status, /standards | Clean REST semantics, each resource independently addressable |
| Implementation phasing | 5 phases: Foundation → Discovery+Validation → Generation → Application → UI | Validation early creates pull for generation/application. Each phase independently testable |
| Opt-out mechanism | Brand Manager tracks opt-outs internally by scenario name | No service.json changes required. Brand Manager must be running for auditor to check |

### Settled (Round 3)
| Decision | Choice | Rationale |
|----------|--------|-----------|
| Inline marker syntax | CSS `/* brand-manager:<element> */` comments, JSON `_brand` keys. Scanner greps for `brand-manager:` prefix | Simple, unique prefix, survives most minification, framework-agnostic |
| Testing strategy | Unit tests (in-memory SQLite) + HTTP handler tests (httptest) + interface-based test doubles for OpenRouter/agent-manager + integration test for full create→apply→validate flow | Comprehensive coverage, fast execution, follows existing patterns |
| Acceptance scope | `acceptance_allow`: `scenarios/brand-manager/**`, `scenarios/scenario-auditor/api/external_rules_brand_manager.go`, `.vrooli/schemas/service.schema.json` | Covers new scenario + auditor provider + service schema |
| AI provider strategy | Ollama-first with OpenRouter fallback (web-console AIProviderChain pattern). Configurable model IDs. Agent-manager API for agentic tasks | Local-first saves cost, OpenRouter enables monetization via AI credits subscription |
| WCAG validation | Validate contrast for defined pairings during generation (primary-on-background, text-on-surface) + test-genie lighthouse audits for applied accessibility | Practical palette-level validation at generation time; lighthouse covers runtime a11y |

## Testing Plan

### Unit Tests
- **SQLite repository layer**: In-memory SQLite for all CRUD operations on brands, versions, assignments, assets tables. Test schema creation, migrations, constraint enforcement.
- **HTTP handlers**: `httptest.NewRecorder` + gorilla/mux `SetURLVars` for route param injection. Test each endpoint independently with mocked repository.
- **AI provider chain**: Interface-based test doubles for `AIProvider`. Test Ollama→OpenRouter fallback sequence, timeout handling, error propagation.
- **Inline marker scanner**: Test against fixture files with known marker patterns. Verify detection of CSS `brand-manager:` comments, JSON `_brand` keys, partial validation states.
- **WCAG contrast calculator**: Pure-function tests for contrast ratio computation. Verify AA thresholds for defined pairings.
- **Design language generator**: Test markdown output structure given structured brand input + user notes.

### Integration Tests
- **Full lifecycle test**: Create brand → generate assets (mocked AI) → apply to temp scenario directory → validate via scanner → verify inline markers present. Uses `t.TempDir()` for isolated scenario directories.
- **Discovery test**: Populate temp scenario with existing branding artifacts (LPBS .vrooli/branding.json format, CSS custom properties, favicon files) → run discovery → verify draft brand correctly imports.
- **Scenario Auditor provider test**: Start Brand Manager test server → register external provider → invoke auditor rules → verify pass/fail based on scenario branding state.

### Test Seams
- `AIProvider` interface: test doubles for Ollama and OpenRouter
- `AgentSpawner` interface: test double for agent-manager API calls
- `AssetStore` interface: test double for filesystem operations (or real filesystem with `t.TempDir()`)
- Repository interface: in-memory SQLite for fast, isolated tests

## Rollout / Validation Checklist
1. Phase 1 gate: `brand-manager create/list/get/update` CLI commands work, API returns correct JSON, SQLite stores and retrieves brands
2. Phase 2 gate: `brand-manager discover <scenario>` correctly imports LPBS branding.json format, Scenario Auditor provider reports branding status
3. Phase 3 gate: `brand-manager generate <brand_id>` produces logo concepts, palette, typography suggestions via Ollama (or OpenRouter fallback). WCAG contrast validation rejects non-AA pairings
4. Phase 4 gate: `brand-manager apply <brand_id> --scenario <name>` writes inline markers, generates `docs/DESIGN_LANGUAGE.md`, agent-assisted application spawns via agent-manager API
5. Phase 5 gate: UI wizard allows brand creation, preview, and application with live feedback
6. End-to-end: fresh scenario → discover (empty) → generate brand → apply → validate via Scenario Auditor → all rules pass

## Risks + Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| OpenRouter image generation quality insufficient for professional logos | High — logos are the most visible brand element | Medium | Configurable models allow switching to better providers as they become available. Ship with best current option. Support user-uploaded logos as override. |
| Agent-assisted application produces invalid/unvalidatable changes | Medium — defeats the validation guarantee | Medium | Hard constraint: agent instructions mandate that all changes use inline markers. If an agent can't meet this, the application fails rather than silently degrading. |
| OpenRouter API changes break generation workflow | Medium — blocks new brand creation | Low | Interface-based OpenRouter client allows swapping implementations. Brands already created still work. |
| Inline markers accidentally deleted during scenario development | Medium — validation reports false negatives | Medium | Scanner detects missing markers as "validation unknown" not "validation passed". Re-apply restores markers. |
| Opt-out tracking requires Brand Manager to be running for auditor checks | Low — only affects test scenarios | Low | Auditor can cache opt-out list. Fallback: treat unknown scenarios as needing branding (safe default). |
| Ollama not available on all deployment targets | Medium — blocks generation on machines without local AI | Low | OpenRouter fallback ensures generation always works when API key configured. Graceful error when neither is available. |
| CSS marker comments stripped by aggressive minification | Medium — validation breaks for minified builds | Low | Document supported minifier configs. Most CSS minifiers (PostCSS, cssnano) preserve comments with `!` prefix — use `/*! brand-manager:<element> */` if needed. |

## Non-goals / Prohibited Patterns
- No auto-push of brand updates to scenarios (pull model only)
- No ComfyUI or local image generation pipelines
- No direct writes to scenario branding state from outside Brand Manager
- No MinIO/object storage dependency
- No service.json schema modifications for branding

## Definition of Done
1. All 5 phases implemented and passing their respective gate checks
2. `go test ./...` passes with ≥80% coverage on repository and handler layers
3. Integration test exercises full create→apply→validate lifecycle
4. Scenario Auditor external provider registered and reporting branding rules
5. CLI, API, and UI all support brand creation, generation, application, and validation
6. At least one real scenario (e.g., LPBS) has a brand discovered, applied, and validated
7. `docs/DESIGN_LANGUAGE.md` generated for applied scenario is coherent and useful for agent consumption
8. WCAG AA contrast validation rejects non-compliant pairings during generation
9. No service.json schema changes committed (opt-out is internal)
10. All changes within `acceptance_allow` patterns: `scenarios/brand-manager/**`, `scenarios/scenario-auditor/api/external_rules_brand_manager.go`, `.vrooli/schemas/service.schema.json`
