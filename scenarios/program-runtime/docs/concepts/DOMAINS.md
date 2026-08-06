# Domains — Program Runtime

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

Five product domains are mapped below, plus the scaffold's `health`. None
are implemented yet — this map is the Gate 3 design artifact that decides
build order before code exists. The scaffold also ships one clearly fenced
worked example domain (never product scope) as a copyable reference;
`template-manager detemplate program-runtime` removes every fenced example
once the real domains are green.

**Build order is decided by the read graph, not by surface.** `bindings`
reads no other domain and is therefore first. `sessions` reads `bindings`
to know what a session may call. `programs` reads both. `telemetry`
observes `programs`. `actspace` reads `bindings` only, so it can proceed
in parallel with `sessions` once `bindings` is green.

```
bindings ──┬── sessions ── programs ── telemetry
           └── actspace
```

No two domains read each other, so no shared-data ownership question is
open. Within a domain, follow the `ARCHITECTURE.md` extension order:
proto, API, transport, CLI, then UI. Finish a domain before starting the
next; do not build every API, then every CLI, then every UI.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/program-runtime/v1/shared/health.proto` |
| bindings | Project the callable surface from the proto descriptor and CLI manifests, validate arguments, and refuse ungoverned calls. | An agent can only invoke what is typed and governed; this domain decides what exists to call. | Binding-resolution snapshots and refusal reasons. | registry | query, policy | Binding, Callable, Grant, Effect | `api/internal/bindings/`, `kernel/bindings/` |
| sessions | Own kernel session lifecycle: creation, grants, persistence of kernel state, reclamation. | Programs need somewhere durable to bind variables between submissions. | Sessions, grants, reclamation reasons. | lifecycle | service | Session, Grant, Reclamation | `api/internal/sessions/`, `kernel/host/` |
| programs | Accept a program, execute it in its session, and return only what it materializes. | This is the product: one submission replaces many tool calls. | Program submissions, results, failure detail. | execution | service | Program, Handle, Materialization | `api/internal/programs/`, `kernel/host/` |
| telemetry | Emit typed platform events for submissions, invocations, and failures. | Program failures are the highest-quality friction evidence in the system; they must leave the scenario. | Nothing durable beyond an outbox. | reporting | integration | ProgramEvent, FailureShape | `api/internal/telemetry/` |
| actspace | Serve the Act denominator and report the live binding numerator. | meta-optimization-manager cannot measure the acting surface without an owner that answers for it. | Nothing; the denominator is a doc, the numerator is computed live. | reporting | query | ActCell, OperationClass | `api/internal/actspace/`, `docs/spaces/act-space.md` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/program-runtime/v1/notes/` |

- Purpose: demonstrate the expected vertical slice for a real domain.
- Primary archetype: CRUD / entity.
- Secondary traits: binary/blob attachment upload, upload workflow.
- Owns: note records, attachment metadata, note validation, note
  service/repository seams, UI note interactions, CLI notes commands.
- Does not own: product scope for a generated scenario.
- API: `api/internal/notes/`, `api/handlers/notes/`.
- CLI: `cli/domains/notes/`.
- UI: `ui/src/features/notes/`, `ui/src/api/notes.ts`.
- Storage: domain-owned SQLite schema in `api/internal/notes/schema.sql`.
- Requirements: template starter only; replace with PRD-specific
  requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and
  workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).
<!-- EXAMPLE-DOMAIN:notes END -->

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### bindings

- Purpose: decide what an agent can call, and refuse everything else.
- Primary archetype: registry.
- Secondary traits: query, policy enforcement.
- Owns: ingestion of `packages/proto/gen/descriptor/image.binpb` and the
  scenario `cli/manifest.json` files; projection of manifest-bound Connect
  methods into a callable namespace; pre-flight argument validation against
  the bound request message; refusal of `run_eligible: false` commands; the
  grant check for `effect: destructive`.
- Does not own: the governance vocabulary itself. `effect`, `run_eligible`,
  `requires_confirmation`, and `permissions` are read from
  `.vrooli/schemas/cli-manifest.schema.json`. This scenario enforces that
  contract and never declares a second one.
- Does not own: transport. Outbound calls use ConnectRPC through the
  standard client; this domain decides whether a call is permitted and
  well-formed, not how bytes move.
- API: `api/internal/bindings/`.
- Kernel: `kernel/bindings/` — the callable projection the program sees.
- Storage: binding-resolution snapshots and refusal reasons only.
- Requirements: `PRT-P0-001`, `PRT-P0-002`, `PRT-P0-005`, `PRT-P1-001`.
- Tests: registry projection, argument validation, governance refusal,
  discovery degradation.

### sessions

- Purpose: give programs a durable place to keep state between submissions.
- Primary archetype: lifecycle.
- Secondary traits: service.
- Owns: session creation and identity; the grant set a session holds;
  kernel child-process supervision; state persistence across submissions;
  idle reclamation and wall-clock and memory ceilings; optional binding of
  a session to a workspace-sandbox workspace.
- Does not own: what a program may call — that is `bindings`. A session
  holds grants; it does not interpret them.
- Does not own: durable persistence of kernel variables. Kernel state is
  process memory by deliberate decision; see `DATA.md`.
- API: `api/internal/sessions/`.
- Kernel: `kernel/host/` — the session loop inside the sidecar.
- Storage: sessions, grants, reclamation reasons.
- Requirements: `PRT-P0-004`, `PRT-P1-004`, `PRT-P1-005`, `PRT-P2-003`.
- Tests: state survival across submissions, cross-session isolation,
  reclamation with a stated reason, ceiling enforcement.

### programs

- Purpose: run a submitted program and return only what it materializes.
- Primary archetype: execution.
- Secondary traits: service.
- Owns: program submission and admission; dispatch into the session
  kernel; the handle type and its bounded representation; explicit
  materialization; program history and failure detail as a queryable
  corpus; the in-program inference and delegation callables.
- Does not own: inference. `classify`, `extract`, and `judge` resolve
  through ai-gateway. This domain exposes them as callables and never
  contacts a provider directly.
- Does not own: agent runs. Delegation spawns an agent-manager run and
  collects its evidence; it does not reimplement run orchestration.
- API: `api/internal/programs/`.
- Kernel: `kernel/host/` — handle implementation and materialization.
- Storage: submissions, results, failure detail.
- Requirements: `PRT-P0-003`, `PRT-P1-002`, `PRT-P1-003`, `PRT-P1-006`,
  `PRT-P2-002`.
- Tests: bounded repr, explicit materialization, in-kernel aggregation,
  the context-bytes-per-query budget, inference routing, delegation.

### telemetry

- Purpose: get program evidence out of this scenario and onto the
  platform bus, where existing analysis already lives.
- Primary archetype: reporting.
- Secondary traits: integration.
- Owns: typed event emission for every submission, binding invocation,
  and failure, through `packages/api-core/eventbus`; the failure-shape
  classification carried on a failure event.
- Does not own: analysis, aggregation, or ranking. Friction analysis
  belongs to agent-manager; the ranked board belongs to
  meta-optimization-manager. Building a scenario-local analysis stack
  here would duplicate both.
- API: `api/internal/telemetry/`.
- Storage: an outbox only; events are not a durable local store.
- Requirements: `PRT-P0-006`.
- Tests: event emission per action class, failure locator presence, and
  an architecture test asserting no local aggregation.
- Known gap: agent identity does not currently reach events raised for
  in-program inference. See `INTEGRATIONS.md` §Identity propagation.

### actspace

- Purpose: answer for the Act projection, on both sides.
- Primary archetype: reporting.
- Secondary traits: query.
- Owns: the Act denominator at `docs/spaces/act-space.md`, served through
  the shared `space --projection act --json` verb via `api-core/spacecli`;
  the binding-registry RPC that reports, per operation class, whether every
  operation it names resolves to a governed binding.
- Does not own: coverage arithmetic, ranking, or gap derivation. Those
  belong to meta-optimization-manager, which computes the numerator live
  and never stores it.
- Does not own: the projection model. That is canon in
  `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md`.
- API: `api/internal/actspace/`.
- Storage: none. The denominator is a document; the numerator is computed.
- Requirements: `PRT-P0-007`, `PRT-P0-008`.
- Tests: denominator parse and spacedoc conformance, per-cell resolution,
  partially-bound cells reporting in-reach, unresolvable cells keeping
  authored status.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `mining` | Proposing skill and action candidates from recurring program shapes needs a corpus that does not exist until programs have run. Splitting it from `programs` before then would be a boundary drawn on a guess. | `PRT-P1-006` is green and the corpus holds enough programs for a recurring shape to be observable. |
| `refine` | Session-local harness refinement is a real capability, but its evidence is the run trajectory, which agent-manager owns, and its durable write targets are prompt-manager and vrooli-memory. Decided 2026-08-06 — not a domain here; see the durable-decision note in `docs/internal/DECISIONS.md`. | A refinement need appears whose evidence is the program corpus rather than the run trajectory. |
| `adapters` | Non-Python kernels (`PRT-P2-004`) would justify an adapter domain. One kernel does not. | A second kernel language has a measured capability argument. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
