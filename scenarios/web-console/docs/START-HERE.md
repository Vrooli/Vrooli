# Start Here — Web Console

This is the first document to read when picking up work on `web-console`.
The scenario was generated before the `react-vite` template was versioned;
it is being retrofitted into structural conformance with template v1.0.0
in small, independently-mergeable batches. Treat this file as the
initialization protocol: gates are checked off in order, and gates that
are still open describe the only remaining template-adoption work.

Run `make orient` from this directory whenever you want a
machine-readable progress check. It delegates to
`template-manager orient web-console`, which reads
`.vrooli/orientation.json`.

Unlike a freshly generated scenario, web-console is **past initial
scaffolding** — the product is real, the proto domains are real, the
notes example domain never existed here. Several template gates are
therefore checked off below with an evidence link. Open gates describe
work that genuinely remains.

## Initialization Protocol

### Gate 0 — Scaffold Health  [x]

Complete. The scenario boots through the standard Vrooli lifecycle:

- `make start` / `make stop` / `make restart` / `make status` all
  delegate to `vrooli scenario ...`.
- `make test` submits the Test Genie comprehensive preset through
  `vrooli scenario test web-console`; `.vrooli/testing.json` owns any
  scenario-specific test configuration.
- API health is exposed at `/health` and gated by the
  `health` Connect service in `api/internal/health/`.

### Gate 1 — Charter  [x]

Complete. `PRD.md` exists and is free of template placeholder text.
It captures the durable scenario charter — pane-based terminal
workspaces, PTY-backed session durability, AI command generation,
configurable launch shortcuts, and the mobile-first floating toolbar.

### Gate 2 — Requirements Registry  [x]

Complete. `requirements/index.json` imports 12 scenario-specific
modules (`01-pane-based-terminal-workspace` through
`12-operational-observability-coverage`); the template's foundation
starter is not used.

### Gate 3 — Domain Map  [x]

Complete. `docs/concepts/DOMAINS.md` records the live bounded contexts;
`docs/concepts/ARCHITECTURE.md` describes the system map. The proto
domains under `packages/proto/schemas/web-console/v1/` mirror the
domain inventory.

### Gate 4 — Dependency Decisions  [x]

Complete. `.vrooli/service.json::dependencies.resources` declares the
three real resource dependencies (`kokoro`, `whisper`, `ollama`) with
explicit `startup_policy` and `degraded_behavior`. Storage is SQLite
embedded in the API process (see `notes.storage_strategy`).

### Gate 5 — Design Language  [ ]

Open. Root `DESIGN.md` is not yet authored. The UI uses a design
adapter implicit in `ui/src/design-tokens.css` and Tailwind theme,
but the canonical design contract document is missing.

- [ ] Author root `DESIGN.md` (binding tokens, motion, status colors,
      a11y floors, responsive transformations).
- [ ] Confirm `ui/src/design-tokens.css`, `ui/tailwind.theme.json`,
      `ui/tailwind.config.ts`, and `ui/src/components/ui/` primitives
      align with `DESIGN.md`.
- [ ] Audit settings surface against the design contract.

### Gate 6 — First Real Vertical Slice  [x]

Complete. Every domain in the scenario is a real vertical slice —
there is no `notes` example domain. Connect-RPC + proto schemas drive
the API (`api/internal/<domain>/`), CLI (`cli/domains/<domain>/`),
and UI (`ui/src/api/<domain>.ts`, migrating to
`ui/src/features/<domain>/` in Batch 3 of the template-adoption sweep).

### Gate 7 — Example Domain Removed  [x]

N/A. The web-console scenario never carried the notes example domain.
The orientation glob checks pass trivially.

### Gate 8 — Progress Handoff  [ ]

Partially complete. `docs/internal/PROBLEMS.md` exists. The active
progress log remains at `docs/PROGRESS.md`.

- [ ] Consolidate the progress log into the internal documentation area
      during the documentation-health pass that follows the template-adoption
      batches.

## Template Adoption Status (2026-05-14)

Tracking work on top of the template v1.0.0 contract. See the plan at
`plan-manager plans render web-console-react-vite-template-adoption-batches-1-3`.

- [x] **Batch 1 — Foundation & metadata.** `.vrooli/service.json` has
      `generation.template`; `.vrooli/orientation.json` exists;
      `docs/START-HERE.md` (this file) exists; `Makefile` exposes
      `orient` and `temporal-models`; `docs/manifest.json` is upgraded
      to schema v2.0.0.
- [ ] **Batch 2 — Eliminate non-sanctioned REST in UI.** Convert
      `ui/src/api/health.ts` to a Connect `HealthService` client;
      document `ui/src/api/uploads.ts` as the sanctioned multipart
      REST exception in `docs/internal/SEAMS.md`; add a fetch-grep
      regression test.
- [ ] **Batch 3 — UI feature-folder reorg.** Migrate `ui/src/` into
      `ui/src/features/<domain>/` per template convention, one domain
      per commit; delete `ui/src/api/` and `ui/src/domains/`.
- [ ] **Batch 4 — API screaming-architecture sweep.** Verify each
      `api/internal/<domain>/` matches the template's notes-domain
      shape; complete `docs/internal/SEAMS.md` registry.
- [ ] **Batch 5 — Quint temporal flows.** Author Quint models for
      durable session lifecycle and voice streaming pipeline; wire
      `make temporal-models` into the test gate.
- [ ] **Batch 6 — Documentation health sweep.** Author missing docs
      registered as `maturity: "missing"` in the manifest (`DESIGN.md`,
      `docs/operations/*`, `docs/business/*`, `docs/internal/TESTING.md`,
      etc.); consolidate the progress log into the internal documentation
      area.

Each open batch is independently mergeable. Do not start batch N+1
until N is merged and green — this is the lesson from the
swarm-manager big-bang revert on 2026-05-13.

## Architecture Rules

- Business logic belongs in `api/internal/<domain>/`.
- Wire contracts belong in
  `packages/proto/schemas/web-console/v1/<domain>/`.
- UI and CLI are translation layers over the API; they do not own
  business rules.
- Generated files are regenerated, not hand-edited.
- The only REST endpoint in the UI is `ui/src/api/uploads.ts`
  (multipart binary upload); every other UI→API call goes through
  Connect-RPC.

Read `docs/concepts/ARCHITECTURE.md` before changing structure, and
read `docs/internal/SEAMS.md` before crossing a boundary.

## Replacing The Example Domain

N/A. There is no example domain to replace. When adding a new domain,
follow the template pattern:

1. Add proto messages and a service under
   `packages/proto/schemas/web-console/v1/<domain>/`, then run
   `make generate` from `packages/proto`.
2. Add `api/internal/<domain>/` with types, repository, service,
   schema, tests.
3. Add `api/handlers/<domain>/` with a thin generated Connect handler.
4. Register the schema/endpoints in `api/internal/modules/registry.go`
   and mount the module in `api/main.go`.
5. Add `cli/domains/<domain>/` and register in `cli/domains/domains.go`.
6. Add endpoint→command rows in
   `api/cmd/gen-endpoints/cli_commands_seed.json`, then
   run `make endpoints`.
7. Add `ui/src/features/<domain>/` with a Connect client wrapper,
   components, hooks, stores, types, and tests.
8. Run code generation and `make test`.

Opaque binary uploads keep bytes on a REST multipart edge
(`ui/src/api/uploads.ts`) and keep metadata proto-typed.
