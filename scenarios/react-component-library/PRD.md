# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Stable operational-target IDs; operator-authorized rehabilitation amendment on 2026-09-30. Checkboxes require current evidence for their stated scope.

> **Current workbench status:** use [WORKBENCH-PROGRAM.md](docs/internal/WORKBENCH-PROGRAM.md)
> and [PROBLEMS.md](docs/internal/PROBLEMS.md) for the 2026-11-20 implementation
> and remaining certification work. Older architecture and launch snapshots are
> retained as history where they conflict with that status.

## 🎯 Overview
- **Purpose**: Central UI for designing, previewing, editing, reviewing, and composing shared React UI components across Vrooli scenarios. Eliminates component duplication, accelerates UI development, enforces design consistency, and enables systematic component evolution via versioning and drift detection. The workbench visual language is **D · Nocturne**; its current product target is five destinations — Start, Library, Design, Review and Stats — with Start/Library/Stats replacing the historical Home/Components/Coverage/Activity/Capabilities split. The historical eleven-page implementation record is [WORKBENCH-REDESIGN.md](docs/concepts/WORKBENCH-REDESIGN.md); the superseding IA and workspace model are [WORKBENCH-MATURATION.md](docs/concepts/WORKBENCH-MATURATION.md) and the delivery program is [WORKBENCH-PROGRAM.md](docs/internal/WORKBENCH-PROGRAM.md).
- **Primary users/verticals**: Vrooli scenario developers (humans and agents) building UIs; design-system maintainers curating the shared primitive set.
- **Deployment surfaces**: Go API (per-domain SQLite), Go CLI (`react-component-library`), React/Vite/Tailwind UI, BAS flows. Local-only single-user; not multi-tenant.
- **Value promise**: Every shared UI primitive is authored once, previewed live in real React execution across multiple viewports, examples, and visual filters, adopted into target scenarios with drift tracking, and evolved with version diffs. Compounds across the platform — each accepted component becomes a permanent capability the system can reuse forever, including its data-defined example states and portable experience claims.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Component registry & source-derived contracts | Disk-walking indexer parses `component.json`, version folders, `@libraryId` / `@version` / `@deps` header comments, and per-version `story.json`, then upserts SQLite-backed registry, version, dependency, source-derived styling-contract, and story projections; malformed structural headers reject with a structured, actionable error and non-fatal contract drift is reported as findings.
- [ ] OT-P0-002 | Monaco editor with safe content I/O | TSX editing, save, format-on-type; all reads/writes routed through the component authoring service with path-traversal rejection. Evidence: the Asset data drawer Source tab mounts locally bundled Monaco against the active draft, saves with `expected_sha256`, runs `CheckComponentVersion`, and publishes only through `PublishComponentVersion`. Unit coverage: `ui/src/workbench/component/SourceWorkbench.test.tsx`.
- [ ] OT-P0-003 | Live preview executes real React in an isolated iframe | Per-component harness renders the actual component (no placeholder HTML), supports versioned named examples and stable `libraryId` harness URLs, reports preview-ready/error to the host, and reloads on save through cache-busted iframe navigation.
- [x] OT-P0-004 | Multi-viewport emulator | Device presets (mobile/tablet/desktop and named devices), continuous zoom 10–200%, rotate, reset; state persists across sessions.
- [x] OT-P0-005 | Search and filter the registry | Name and description substring match plus tag/category facets; p95 query under 100ms on the test corpus.
- [x] OT-P0-006 | Safe adoption workflow with drift status | `adoption_records` track scenario, path, adopted version; refresh computes current/behind/modified/unknown, while apply, reapply, and reconverge refuse unsafe writes unless the caller supplies the explicitly validated confirmation or override.
- [ ] OT-P0-007 | CLI parity for headless workflows | `react-component-library {components,adoptions,versions} ...` covers list/search/get/index/create/refresh, atomic batch operations, and lifecycle operations; default human output, `--json` opt-in (per `cli-steer`).
- [ ] OT-P0-008 | Test coverage meets the template floor | Per-domain SQLite-backed repository tests, handler tests over mocks, UI component tests per page, token-complete templates and adopting scenarios, catalog TypeScript/ESLint gates, real-browser preview sweeps with explicit selected-core versus full-catalog scope, component experience reconciliation, and BAS flows for primary user journeys.
- [ ] OT-P0-009 | Executable page design and reconciliation | From discoverable normal navigation, the studio shall guide page selection, durable briefing, composition, viewport/style review and governed implementation. Workspace aggregates own composition; scenario experience contracts own intent and claims, linked by explicit reconciliation/export. The studio shall reconcile declared regions against observed source with a closed verdict vocabulary and emit an ordered implementation brief whose closing gate verifies coverage and drift. This replaces the original sketch-as-composition storage wording.


- [ ] OT-P0-010 | Selected core production readiness | The selected core and necessary dependency closure shall meet the contracts, state, accessibility, template-fit and real-consumer acceptance gates in docs/internal/goal/TARGETS.md; the full desired catalog is excluded from this rehabilitation certification.
- [ ] OT-P0-011 | Honest scoped readiness and reusable adoption | Discovery and readiness shall distinguish verified behavior, failed behavior and missing/stale evidence for exact versions and scope; shared package adoption shall build in two real consumers and preserve intentional local composition/ejection provenance.
- [ ] OT-P0-012 | Maintainable ownership and cleanup | Each rehabilitated concept shall have one authoritative owner; refactor epochs shall complete concrete replacement/deletion lists and net-negative runtime budgets, while feature growth is justified and behavior preserved.
- [ ] OT-P0-013 | Governed generic repair and consumer evolution | A fresh agent shall repair an asset through a governed draft/release workflow and safely upgrade two consumers with impact, compatibility, local-change protection and recovery evidence.
- [ ] OT-P0-014 | Nocturne workbench redesign | **Historical scope, superseded by the five-destination target.** The operator-selected **D · Nocturne** design replaced the workbench chrome across eleven pages: Home, Components, Component, Design, Review, Coverage, Capabilities, Activity, Settings, Create and Preview; the current target consolidates those into Start, Library, Design, Review and Stats plus the Component, Preview, New and Settings object routes. The original acceptance text below is retained as historical scope: fixture canvas and in-context page mode; preserve catalog kinds through the Asset data drawer; first-class Review/Activity/Create; route preservation; shared RCL controls; honest unavailable, stale, unmeasured and permission-denied states. **Implementation evidence below is a 2026-11-19 snapshot, not current status:** the shell and eleven-route layout described there have since been replaced by the five-destination workspace. Current completion and evidence are in [WORKBENCH-PROGRAM.md](docs/internal/WORKBENCH-PROGRAM.md) and [PROBLEMS.md](docs/internal/PROBLEMS.md). Design history: [docs/concepts/WORKBENCH-REDESIGN.md](docs/concepts/WORKBENCH-REDESIGN.md); current IA: [docs/concepts/WORKBENCH-MATURATION.md](docs/concepts/WORKBENCH-MATURATION.md); original requirements: [requirements/26-workbench-redesign](requirements/26-workbench-redesign/module.json).

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | DevTools-style visual filters | Color-scheme toggle (system/light/dark) and a vision-filter dropdown (blur 0–10px, grayscale, protanopia, deuteranopia, tritanopia) applied to the preview iframe.
- [ ] OT-P1-002 | Element selection via `@vrooli/iframe-bridge` | Hover overlay rect, ancestor breadcrumb, element screenshot, selector capture; selection feeds the AI chat panel context.
- [x] OT-P1-003 | Adoption-drift backlog integration | When refresh detects "behind"/"modified", file a `fix` backlog item via `swarm-manager`'s CLI; never raw HTTP; dedupe via the recorded `drift_backlog_ref`.
- [x] OT-P1-004 | Dependency compatibility check on adopt | Component declares `@deps` (JSON in header); on adopt, validate against the target scenario's `package.json`; warn on missing/mismatch, block on incompatible-major.
- [x] OT-P1-005 | Version tracking with a real diff viewer | Each save records a new version; UI renders a side-by-side unified diff between any two versions and between the library version and an adopted copy.
- [x] OT-P1-006 | Theme-preview switcher | Pick from built-in themes or load a target scenario's `DESIGN.md`-derived theme; tokens mount as CSS custom properties on the harness `:root` before render. Resolver is a server endpoint, never client-derived.
- [ ] OT-P1-007 | Design-style fit validation | Components declare advisory `designStyles[]` affinities with rationale; adoption workflows validate the target scenario's `.vrooli/service.json` `generation.design.id` and surface native/compatible/discouraged/stale-style findings.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | AI-powered editing (single refactor preset) | Chat panel sends current component plus selected-element context to `resource-openrouter`; returns a unified-diff patch suggestion; user must explicitly accept; no auto-apply.
- [ ] OT-P2-002 | Importing components from external sources | Deferred; registry schema must not preclude later support for npm/external imports.
- [ ] OT-P2-003 | Storybook-style auto-generated docs from prop types | Deferred; not implemented in v0.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API; React 18 + Vite + Tailwind UI; Go CLI; Monaco for the code editor; `@vrooli/iframe-bridge` for host/child postmessage protocol. The Nocturne workbench uses current RCL assets for shared controls/surfaces and a scoped chrome token layer; no synthetic component markup and no replacement of BaseStyles.
- Data + storage expectations (architecture snapshot): SQLite via `modernc.org/sqlite` (CGO-free, pure-Go), per-domain schema ownership, and filesystem writes through `package:api-core/storage` `NewResolver`. The former “no migrations folder” rule is historical; current migration status and ownership are described in [WORKBENCH-PROGRAM.md](docs/internal/WORKBENCH-PROGRAM.md).
- Integration strategy: Wrap external tools in scenario CLIs (per `feedback_skills_use_cli_never_api.md`). `app-issue-tracker` and `business-health` are invoked through their scenario CLIs; workbench AI uses AI Gateway, and semantic search uses the shared `packages/ai-go/search` adapter; iframe bridge is consumed via the shared `@vrooli/iframe-bridge` package.
- Non-goals / guardrails: No multi-tenant cloud registry. No npm/external imports in v0. No auto-applied AI edits. No `brand-manager` coupling. No Postgres. No central schema file. No SQL in handlers (handlers call services; services call per-domain repositories). No hard cross-domain FK constraints (soft FKs only). No PROGRESS.md "complete" claim without matching green tests.

## 🤝 Dependencies & Launch Plan
- Required resources: None at runtime for P0 (SQLite is in-process). `resource-openrouter` required only for P2 AI editing.
- Scenario dependencies: `app-issue-tracker` (P1 adoption issues); `flow-verifier` (build-time temporal-model lint); `business-health` (build-time PRD and requirements validation).
- Operational risks (older launch snapshot): per-component React bundling uses the Go esbuild path documented in `docs/RESEARCH.md`, with same-origin runtime imports and browser gates over the catalog. Current workbench verification and remaining journey/search evidence are tracked in [WORKBENCH-PROGRAM.md](docs/internal/WORKBENCH-PROGRAM.md); do not treat this dated launch text as current readiness evidence. SQLite concurrent-writer pressure remains a soft risk mitigated by WAL mode.
- Historical launch sequencing (superseded for rehabilitation by docs/internal/goal/QUEUE.md): P0 vertical slices in order — (1) registry + header parsing, (2) Monaco + save, (3) live preview iframe execution, (4) multi-viewport + search, (5) adoption workflow. Then all P1 features in parallel. Usable-milestone demo: list → edit → preview in 3+ viewports with a vision filter → select element → adopt into target → bump version → status flips to "behind" → diff viewer renders. Onboarding-doc shout-out lands only after the usable milestone passes.

## 🎨 UX & Branding
- Look & feel: **D · Nocturne** established the visual direction: deep warm charcoal/plum surfaces, amber signal and selection, ice-blue secondary emphasis, serif display headings, sans working UI and mono metadata. The original eleven-route map is historical; the current destination model is Start, Library, Design, Review and Stats, with Component/Preview, New and Settings as object/secondary routes. The Component page has Fixtures and In context modes plus an Asset data drawer. Evidence is ambient, not a seven-tab maze. The Design studio keeps the brief → compose → review → adopt spine and adds compare and present. The original page map is in [docs/concepts/WORKBENCH-REDESIGN.md](docs/concepts/WORKBENCH-REDESIGN.md); the current map is [WORKBENCH-MATURATION.md](docs/concepts/WORKBENCH-MATURATION.md).
- Accessibility: WCAG AA floor for the library's own chrome, including Nocturne contrast. Keyboard navigation across registry list, fixture canvas, inspector, design studio and preview controls is required. Arbitrary third-party preview code is outside blanket certification; selected core previews, studio controls and proving-consumer journeys must meet the rehabilitation accessibility gates. Phone compositions at 390 and 320 px are required before production readiness.
- Voice & messaging: Terse, developer-facing. No marketing copy. Errors are structured and actionable (for example: "header parse failed at /path:line — missing `@version`"). Create, Extract and Adopt must say what they do, what they need and what changes.
- Branding hooks: Inherits Vrooli operational-console tokens (`vrooli-default` design kit, `react-vite-tailwind` adapter). Nocturne is workbench chrome, not a replacement theme for published components; shared component values remain owned by BaseStyles. The P1 theme-preview switcher applies *only* to preview content, never to the workbench chrome.

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
