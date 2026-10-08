# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Stable operational-target IDs; operator-authorized rehabilitation amendment on 2026-09-30. Checkboxes require current evidence for their stated scope.

## 🎯 Overview
- **Purpose**: Central UI for designing, previewing, editing, and tracking shared React UI components across Vrooli scenarios. Eliminates component duplication, accelerates UI development, enforces design consistency, and enables systematic component evolution via versioning and drift detection.
- **Primary users/verticals**: Vrooli scenario developers (humans and agents) building UIs; design-system maintainers curating the shared primitive set.
- **Deployment surfaces**: Go API (per-domain SQLite), Go CLI (`react-component-library`), React/Vite/Tailwind UI, BAS flows. Local-only single-user; not multi-tenant.
- **Value promise**: Every shared UI primitive is authored once, previewed live in real React execution across multiple viewports, examples, and visual filters, adopted into target scenarios with drift tracking, and evolved with version diffs. Compounds across the platform — each accepted component becomes a permanent capability the system can reuse forever, including its data-defined example states and portable experience claims.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Component registry & source-derived contracts | Disk-walking indexer parses `component.json`, version folders, `@libraryId` / `@version` / `@deps` header comments, and per-version `story.json`, then upserts SQLite-backed registry, version, dependency, source-derived styling-contract, and story projections; malformed structural headers reject with a structured, actionable error and non-fatal contract drift is reported as findings.
- [ ] OT-P0-002 | Monaco editor with safe content I/O | TSX editing, save, format-on-type; all reads/writes routed through `package:api-core/storage` with path-traversal rejection.
- [ ] OT-P0-003 | Live preview executes real React in an isolated iframe | Per-component harness renders the actual component (no placeholder HTML), supports versioned named examples and stable `libraryId` harness URLs, reports preview-ready/error to the host, and reloads on save through cache-busted iframe navigation.
- [x] OT-P0-004 | Multi-viewport emulator | Device presets (mobile/tablet/desktop and named devices), continuous zoom 10–200%, rotate, reset; state persists across sessions.
- [x] OT-P0-005 | Search and filter the registry | Name and description substring match plus tag/category facets; p95 query under 100ms on the test corpus.
- [x] OT-P0-006 | Safe adoption workflow with drift status | `adoption_records` track scenario, path, adopted version; refresh computes current/behind/modified/unknown, while apply, reapply, and reconverge refuse unsafe writes unless the caller supplies the explicitly validated confirmation or override.
- [ ] OT-P0-007 | CLI parity for headless workflows | `react-component-library {components,adoptions,versions} ...` covers list/search/get/index/create/refresh, atomic batch operations, and lifecycle operations; default human output, `--json` opt-in (per `cli-steer`).
- [ ] OT-P0-008 | Test coverage meets the template floor | Per-domain SQLite-backed repository tests, handler tests over mocks, UI component tests per page, token-complete templates and adopting scenarios, catalog TypeScript/ESLint gates, real-browser preview sweeps with explicit selected-core versus full-catalog scope, component experience reconciliation, and BAS flows for primary user journeys.
- [ ] OT-P0-009 | Executable page design and reconciliation | From discoverable normal navigation, the studio shall guide scenario/page selection, durable briefing, composition, viewport/style review and governed implementation. When a scenario page declares experience regions, the library shall design them from ranked catalog assets, persist the authored sketch beside the scenario, reconcile every region against observed source with a closed verdict vocabulary, and emit an ordered implementation brief whose closing gate verifies coverage and drift.


- [ ] OT-P0-010 | Selected core production readiness | The selected core and necessary dependency closure shall meet the contracts, state, accessibility, template-fit and real-consumer acceptance gates in docs/internal/goal/TARGETS.md; the full desired catalog is excluded from this rehabilitation certification.
- [ ] OT-P0-011 | Honest scoped readiness and reusable adoption | Discovery and readiness shall distinguish verified behavior, failed behavior and missing/stale evidence for exact versions and scope; shared package adoption shall build in two real consumers and preserve intentional local composition/ejection provenance.
- [ ] OT-P0-012 | Maintainable ownership and cleanup | Each rehabilitated concept shall have one authoritative owner; refactor epochs shall complete concrete replacement/deletion lists and net-negative runtime budgets, while feature growth is justified and behavior preserved.
- [ ] OT-P0-013 | Governed generic repair and consumer evolution | A fresh agent shall repair an asset through a governed draft/release workflow and safely upgrade two consumers with impact, compatibility, local-change protection and recovery evidence.

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | DevTools-style visual filters | Color-scheme toggle (system/light/dark) and a vision-filter dropdown (blur 0–10px, grayscale, protanopia, deuteranopia, tritanopia) applied to the preview iframe.
- [ ] OT-P1-002 | Element selection via `@vrooli/iframe-bridge` | Hover overlay rect, ancestor breadcrumb, element screenshot, selector capture; selection feeds the AI chat panel context.
- [ ] OT-P1-003 | Adoption-drift backlog integration | When refresh detects "behind"/"modified", file a `fix` backlog item via `swarm-manager`'s CLI; never raw HTTP; dedupe via the recorded `drift_backlog_ref`.
- [x] OT-P1-004 | Dependency compatibility check on adopt | Component declares `@deps` (JSON in header); on adopt, validate against the target scenario's `package.json`; warn on missing/mismatch, block on incompatible-major.
- [x] OT-P1-005 | Version tracking with a real diff viewer | Each save records a new version; UI renders a side-by-side unified diff between any two versions and between the library version and an adopted copy.
- [x] OT-P1-006 | Theme-preview switcher | Pick from built-in themes or load a target scenario's `DESIGN.md`-derived theme; tokens mount as CSS custom properties on the harness `:root` before render. Resolver is a server endpoint, never client-derived.
- [ ] OT-P1-007 | Design-style fit validation | Components declare advisory `designStyles[]` affinities with rationale; adoption workflows validate the target scenario's `.vrooli/service.json` `generation.design.id` and surface native/compatible/discouraged/stale-style findings.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | AI-powered editing (single refactor preset) | Chat panel sends current component plus selected-element context to `resource-openrouter`; returns a unified-diff patch suggestion; user must explicitly accept; no auto-apply.
- [ ] OT-P2-002 | Importing components from external sources | Deferred; registry schema must not preclude later support for npm/external imports.
- [ ] OT-P2-003 | Storybook-style auto-generated docs from prop types | Deferred; not implemented in v0.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API; React 18 + Vite + Tailwind UI; Go CLI; Monaco for the code editor; `@vrooli/iframe-bridge` for host/child postmessage protocol.
- Data + storage expectations: SQLite via `modernc.org/sqlite` (CGO-free, pure-Go). Per-domain schema ownership (`api/internal/<dom>/schema.sql` + `schema.go`). No Postgres anywhere. No `migrations/` folder — greenfield, idempotent `CREATE TABLE IF NOT EXISTS` describes the desired clean state. Filesystem writes go through `package:api-core/storage` `NewResolver`.
- Integration strategy: Wrap external tools in scenario CLIs (per `feedback_skills_use_cli_never_api.md`). `app-issue-tracker` and `business-health` are invoked through their scenario CLIs; AI calls go through `resource-openrouter`; iframe bridge is consumed via the shared `@vrooli/iframe-bridge` package.
- Non-goals / guardrails: No multi-tenant cloud registry. No npm/external imports in v0. No auto-applied AI edits. No `brand-manager` coupling. No Postgres. No schema migrations folder. No central schema file. No SQL in handlers (handlers call services; services call per-domain repositories). No hard cross-domain FK constraints (soft FKs only). No PROGRESS.md "complete" claim without matching green tests.

## 🤝 Dependencies & Launch Plan
- Required resources: None at runtime for P0 (SQLite is in-process). `resource-openrouter` required only for P2 AI editing.
- Scenario dependencies: `app-issue-tracker` (P1 adoption issues); `flow-verifier` (build-time temporal-model lint); `business-health` (build-time PRD and requirements validation).
- Operational risks: Per-component React bundling now uses the Go esbuild path documented in `docs/RESEARCH.md`, with same-origin runtime imports and browser gates over the catalog. Remaining risk is coverage breadth: the preview/example path is gated, while broader adoption/editing journeys still need full BAS coverage. SQLite concurrent-writer pressure is a soft risk mitigated by WAL mode; the per-domain repository interface keeps a Postgres impl available later if needed.
- Historical launch sequencing (superseded for rehabilitation by docs/internal/goal/QUEUE.md): P0 vertical slices in order — (1) registry + header parsing, (2) Monaco + save, (3) live preview iframe execution, (4) multi-viewport + search, (5) adoption workflow. Then all P1 features in parallel. Usable-milestone demo: list → edit → preview in 3+ viewports with a vision filter → select element → adopt into target → bump version → status flips to "behind" → diff viewer renders. Onboarding-doc shout-out lands only after the usable milestone passes.

## 🎨 UX & Branding
- Look & feel: A clear reuse-first workspace with an intuitive studio entry and guided brief → compose → review → implement journey. Technical editing remains available within context; a dense three-panel IDE is not the mandatory home experience. Tailwind plus design tokens from `DESIGN.md`. Preserve backdrop-blur on graph nodes and overlays.
- Accessibility: WCAG AA floor for the library's own chrome. Keyboard navigation across registry list, editor, and preview controls is required. Arbitrary third-party preview code is outside blanket certification; selected core previews, studio controls and proving-consumer journeys must meet the rehabilitation accessibility gates.
- Voice & messaging: Terse, developer-facing. No marketing copy. Errors are structured and actionable (for example: "header parse failed at /path:line — missing `@version`").
- Branding hooks: Inherits Vrooli operational-console tokens (`vrooli-default` design kit, `react-vite-tailwind` adapter). The P1 theme-preview switcher applies *only* to preview content, never to the library's chrome.

## 📎 Appendix

**Deferred: Manifest postApply Actions**

The `scenario-ui-manifest/v1` schema reserves room (via `additionalProperties: true` on slot objects) for declarative `postApply` actions an adoption could trigger after writing the source file. Three actions are queued for a v2 schema bump:

1. **barrel-export** — append the new component to a slot-owned `index.ts` barrel so consumers can `import { X } from "@/layout"` without per-file imports.
2. **route-register** — append a new route entry to `ui/src/app/routes.tsx` when the adopted component lands in the `page` slot.
3. **i18n-merge** — merge supplied locale fragments into each `ui/src/i18n/locales/<locale>.json` for components that ship their own strings.

**Why deferred:** the four-source path resolver (explicit / template-manifest / heuristic / fallback) is the highest-value change. Adding action runners doubles the surface area we need to test and gives users a new way to corrupt their tree if a runner is buggy. Ship the path resolver first; promote one action at a time to v2 once we have evidence the manifest contract holds up in practice.

**Tracking:** when v2 starts, register a runner interface in `internal/adoptions/postapply/`, gate the per-action runner behind a manifest opt-in (`slot.postApply: ["barrel-export"]`), and back-fill behaviour with golden-file fixtures of the affected files. Schema migration is additive — slot objects keep `additionalProperties: true`, so v1 manifests upgrade silently when the runner ships.

**Linked contract:** `templates/scenarios/react-vite/ui/manifest.json`, `.vrooli/schemas/scenario-ui-manifest.schema.json`, `docs/concepts/UI-ARCHITECTURE.md` (template scenario).

**References**
- Reference scenarios: `app-monitor` (iframe-bridge integration, device-emulation hooks, inspector overlay), `flow-verifier` (newest template alignment, DESIGN.md shape), `reference-react-vite` (golden-reference layout).
- Substrate packages: `@vrooli/iframe-bridge`, `package:api-core/storage`, `package:api-core/database/schemas`.
- Stashed prior implementation: `/tmp/react-component-library-pre-rewrite-2026-05-12/` (reference only; no files are copied into this scenario).
