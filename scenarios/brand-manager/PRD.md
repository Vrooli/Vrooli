# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: One coherent scenario that owns the full branding lifecycle for every Vrooli
  scenario — generate, store, version, assign, apply, **and validate** brand identity across UI,
  CLI, and API surfaces. The scenario that authors branding is the same one that validates it.
- **Primary users/verticals**: Scenario authors and operators standardizing visual identity
  across the fleet; agents running comprehensive scenario tests that need branding to be a
  first-class, prioritized validation dimension.
- **Deployment surfaces**: Connect-RPC API, manifest-driven CLI, React UI (dashboard + wizard),
  and a **test-genie delegated `branding` phase** consumed by `vrooli scenario test <any>`.
- **Value promise**: Branding stops being an after-thought. Every scenario's branding is
  continuously validated and auto-fixed inside the standard test loop, climbing a maturity
  ladder, while authoring/generation/apply make producing a compliant brand cheap.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Brand CRUD + Versioning | Create, read, update, delete, and version brands with identity, visuals, colors, typography, voice, and notes via Connect-RPC API and CLI (optimistic concurrency + idempotency preserved)
- [ ] OT-P0-002 | SQLite Storage | Single SQLite DB (WAL) at `~/.vrooli/brand-manager/brand-manager.db` with tables for brands, brand_versions, assignments, and assets; asset files at `~/.vrooli/brand-manager/assets/{brand_id}/`
- [ ] OT-P0-003 | AI Generation | **Text** facets (palette, typography, voice) use the local LLM provider chain (Ollama-first, OpenRouter fallback). **Image** facets — logo/favicon generation, prompt-based logo editing, background removal, and derived icon variants — run through the **image-tools** scenario (the reusable image capability) via a narrow `ImageBackend` client seam: submit the AI op → wait once for the durable job → download the blob → store as a brand asset. image-tools owns model/backend/provider selection, local backends, and an opt-in BYOK cloud fallback; brand-manager owns brand semantics, prompt recipes, and asset/version policy.
- [ ] OT-P0-004 | Programmatic Application | Apply brand elements to scenarios via CSS custom properties with `/* brand-manager:<element> */` markers, `manifest.json` `_brand`/`icons`/`theme_color`/`background_color` keys, the derived favicon/apple-touch/maskable icon set, and atomic static-asset copy
- [ ] OT-P0-005 | DESIGN.md Export | Generate root-level `DESIGN.md` per scenario during apply from structured brand data + user notes
- [ ] OT-P0-006 | Branding Validation as a test-genie Phase | Branding validation is delivered as a first-class test-genie **delegated phase** via `ScenarioValidationService` (`ValidateScenario`/`PreviewFix`/`ApplyFix`). `vrooli scenario test <scenario>` surfaces severity-gated branding findings (has-display-name, has-logo, has-favicon, has-color-system, has-typography, wcag-aa-contrast, brand-markers-applied), returns a `MaturityAssessment` ladder, and offers deterministic auto-fixes. Findings flow on the `FINDING_SOURCE_BRANDING` channel.
- [ ] OT-P0-007 | Discovery Scanner | Scan existing scenario state (service.json, theme/token files, static assets, manifests, `DESIGN.md`) and auto-populate draft brands with confidence scores
- [ ] OT-P0-008 | Brand Assignment | Link brands to scenarios, track what was applied, when, and at what version; support partial application (individual elements)
- [ ] OT-P0-009 | CLI Surface (manifest-driven) | Commands: create, list, get, update, delete, versions, assign, unassign, scenario-status, generate, discover, apply, scan — all manifest-declared Connect-RPC bindings (no programmatic shell)
- [ ] OT-P0-010 | WCAG AA Contrast | Validate contrast for defined color pairings (primary-on-background, text-on-surface) during generation and validation; reject / flag non-compliant pairings
- [ ] OT-P0-011 | Logo Candidates | Explore N concepts × M variations in one request, import existing images, refine with lineage (instruction, masked object removal, background removal, vectorize), and pick/reject/restore. Each candidate stores prompt, concept, role, model, seed, origin, parent and status; a pick sets the brand's canonical mark. Linked requirements: CAND-001…CAND-006.
- [ ] OT-P0-012 | Mark versus Container | Generation requests a mark on a transparent background; container styles and product lines are records so a product line shares one tile, background, padding, glow and safe zone. Linked requirements: STYLE-001…STYLE-003.
- [ ] OT-P0-013 | Vector Source of Truth | A raster pick is vectorized through image-tools; every PNG, ICO and ICNS target is rasterized from the composed SVG so a vector and its rasters cannot drift. Linked requirements: RENDER-001…RENDER-004.
- [ ] OT-P0-014 | Declared Targets and Apply | The scenario declares its brand and target profiles in `.vrooli/service.json`; apply writes every declared target to the `/public/*` layout, the marked `index.html` block, a relative-src `site.webmanifest`, og/twitter meta, and the electron PNGs, `icon.ico` and `icon.icns`. Linked requirements: TARGET-001…TARGET-005.
- [ ] OT-P0-015 | Declared-Icon-Targets Validation | The branding validation provider checks each declared target's existence, format, exact dimensions, maskable safe-zone containment, wiring and known placeholder hashes. Linked requirements: TARGET-003 and the validation module.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | UI Dashboard | Scenario-centric view of branding status across all scenarios with brand-library browsing and search
- [ ] OT-P1-002 | UI Wizard | Brand creation/editing with live preview, generation-option selection, and application preview before applying
- [ ] OT-P1-003 | Dark/Light Theme Preview | Preview brand application in both theme modes before committing
- [ ] OT-P1-004 | Framework-Extensible Scanning | Scanner plugins per framework/language beyond CSS + JSON (e.g., Tailwind config, SCSS)
- [ ] OT-P1-005 | Logo Page | A repo-owned operator surface: candidate gallery grouped by concept with lineage, side-by-side compare, pick/reject/restore, refine by instruction or brush mask, and a live preview of every declared target at 16/32/64/180/512 px on light and dark grounds with the maskable circle overlay. Linked requirements: LOGOUI-001…LOGOUI-004.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Agent-Assisted Application | Spawn sandboxed agents via agent-manager for complex/non-standard scenario integrations with a mandatory inline-marker constraint (deferred; declare agent-manager only when built)
- [ ] OT-P2-002 | Fleet-Wide Brand Adoption | Apply + validate branding across the whole scenario fleet (tracked under the `brand-manager-readiness` initiative, out of scope for this rebuild)

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go (Connect-RPC), React + Vite + Tailwind, proto-first contracts.
- Data + storage expectations: SQLite (WAL) for metadata; filesystem for assets.
- Integration strategy: served `ScenarioValidationService` consumed by test-genie over Connect;
  shared workflows > resource CLI > direct API for everything else.
- Non-goals / guardrails: **No** scenario-auditor integration (auditor is being retired — branding
  validation is a test-genie phase, not an auditor `external_rules` provider). **No** REST/gorilla
  compatibility surface. **No** splitting authoring and validation into separate scenarios. **No**
  Lighthouse integration. No fleet-wide brand application in this rebuild.

## 🤝 Dependencies & Launch Plan
- Required resources: SQLite (declared in `service.json`). Text generation reaches Ollama /
  OpenRouter via `OLLAMA_URL` / `OPENROUTER_API_KEY` env (degrades gracefully when absent).
- Scenario dependencies: **image-tools** (the image capability) for all logo/favicon/icon image
  work, reached over its public HTTP/Connect surface via api-core service discovery — brand-manager
  never imports image-tools internals or shells its CLI. Image readiness degrades gracefully:
  `GetImageBackendStatus` reports per-operation readiness, and image RPCs return actionable
  dependency errors (unavailable / model-not-installed / BYOK-key-missing). Consumed BY test-genie
  (which depends on this scenario as the `branding` phase provider). agent-manager is a future P2
  dependency only.
- Operational risks: text AI provider availability for facets (degrade gracefully); image-tools
  reachability + model/backend readiness for images (surfaced as actionable errors, not crashes);
  deterministic auto-fix and icon derivation must be idempotent (re-apply = byte-identical).
- Launch sequencing: regen scaffold → docs → domain port (brands→…→design) → validation phase →
  detemplate/finalize. GA = `make test` green + `branding` phase contributing findings to other
  scenarios' comprehensive runs.

## 🎨 UX & Branding
- Look & feel: Operational console (vrooli-default design kit), light/dark themes.
- Accessibility: WCAG AA contrast targets (the scenario validates this for others, so it must
  model it itself).
- Voice & messaging: Clear, system-level, first-principles; "branding as a continuously-validated
  capability," not a one-off cosmetic pass.
- Branding hooks: This scenario authors logos, favicons, color systems, and typography for the
  fleet; it dog-foods its own brand markers.

## 📎 Appendix
- Rebuild plan: `~/.vrooli/plans/brand-manager-regenerate-validation-as-test-genie-phase.md`.
- Validation provider contract: `packages/proto/schemas/scenario-validation/v1/validation.proto`.
- Reference provider impl: `scenarios/tidiness-manager/api/validation_connect.go`.
- Key reusable algorithms ported from the prior REST build: AIProviderChain (`aigen/`), WCAG
  contrast (`contrast/`), SQLite repositories (`repository/`), apply engine + discovery scanner.
