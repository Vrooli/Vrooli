## Steer focus: Screaming Architecture Audit

Prioritize **making scenario structure express the product's capabilities and boundaries** in `scenarios/{{TARGET}}/`. Assess the architecture first, then move code, docs, and tests toward domain-owned implementation with clear shared infrastructure and seams.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.

Read first when present:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — durable mental model, surfaces, domain map, and intended structure.
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — boundary registry, wiring, fakes, and test substitution points.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — unresolved architecture drift and deferred refactors.

Optional context:
- `docs/scenario-qa/methods/audit/screaming-architecture-audit.md` — when this lens applies, when it backfires, and what QA should challenge.
- `prompt-manager skill read temporal-flow-audit` — use when a discovered domain has lifecycle states, illegal transitions, retries, cancellation, or stale-completion risk.

---

> **Template example domain — delete on generation.** The react-vite template ships a `notes` domain (API, UI feature, CLI commands) as a worked example so first-time readers can see every surface lit up at once. It is *not* a domain every scenario inherits — when generating from the template, delete `notes` and replace it with the scenario's real product capabilities. Any worked example in this skill uses placeholder identifiers (`<domain>`, `<Resource>Card`, etc.); substituting them is the moment to ask whether a leftover `notes` folder in your scenario is real product vocabulary or template residue that should be removed.

---

### 0. Programmatic Validation — run this first

Before the manual audit workflow below, take a **photograph** of the scenario's
architecture with one command. This is the L5 ("programmatic drift checks") rung
of the maturity model — let the substrate find the drift so your manual pass
focuses on judgment, not discovery:

```bash
test-genie execute {{TARGET}} --preset architecture-audit --json > audit.json
```

The `architecture-audit` preset runs the per-surface conformance battery
(`structure`, `contracts`, `ui-health`, `docs`, `standards`) **plus** the
`architecture` phase, which delegates to architecture-cartographer for structural
cohesion (import cycles, coupling, convergence drift, mislocated files). Every
finding is normalized into one `ArchitectureFinding` shape with a stable id.

**When the audit nudges: open a tracked migration.** When the finding load
exceeds what one pass can responsibly fix, the audit output appends a migration
recommendation. Do not whack-a-mole a large refactor by hand — that is exactly
the failure mode (the surface area outgrows what you can track). Hand it to the
tracker, which sequences the work and reconciles each re-audit by stable id:

```bash
# Ingest the photograph
architecture-cartographer migration create {{TARGET}} --from-audit audit.json

# Get the prioritized worklist (regressions first, then cycles, then severity —
# cycles block dependent moves, so they lead)
architecture-cartographer migration next <migration-id>

# Fix a finding, then mark it off (the agent fixes by hand; the tracker records it)
architecture-cartographer migration resolve <migration-id> --finding <afid> --note "what you did"

# Re-audit and reconcile: gone → validated, persists → open, (re)appeared → REGRESSION
test-genie execute {{TARGET}} --preset architecture-audit --json > audit-2.json
architecture-cartographer migration reaudit <migration-id> --from-audit audit-2.json

# Repeat next→fix→reaudit until clean, then close
architecture-cartographer migration status <migration-id>
architecture-cartographer migration close <migration-id>
```

A re-audit that flags a **regression** means your fix introduced a new problem
(or a "resolved" finding came back) — handle those first. The doctrine behind
this loop — the four validation responsibilities and the test-genie↔cartographer
seam — is in `docs/reference/architecture-validation-responsibilities.md`.

The manual workflow below is how you reason about and execute each finding's fix;
the audit + tracker are how you discover the work and never lose track of it.

---

### 1. Scope Boundaries

**In scope:**
- aligning file/module structure with scenario purpose and product vocabulary
- separating API domain packages/features from shared infrastructure/substrate packages
- clarifying UI feature ownership and CLI domain command ownership
- moving domain-specific schemas, repositories, services, fakes, tests, workflows, and UI pieces into the owning domain
- moving generic mechanics with no product vocabulary into shared infrastructure or test utilities
- updating `ARCHITECTURE.md`, `SEAMS.md`, and `PROBLEMS.md` so the next agent inherits the model

**Out of scope:**
- broad rewrites that change behavior without immediate structural payoff
- cosmetic renames or file shuffles that do not improve capability ownership or discoverability
- adding new product features under the banner of architecture cleanup
- forcing every domain to have the same files when behavior does not need them
- creating standalone architecture-audit reports as the default memory surface

---

### 2. Architecture Maturity Model

Assess each surface independently. A scenario may be Level 4 in API shape and Level 2 in CLI shape.

| Level | Name | What exists | Main drift risk |
|---|---|---|---|
| 0 | Opaque structure | Code is grouped by history or technical buckets; product capabilities are hard to find. | Agents add more code wherever it fits locally. |
| 1 | Documented mental model | `ARCHITECTURE.md` names surfaces, capabilities, flows, and boundaries. | Docs may describe intent while code remains scattered. |
| 2 | Capability map | Each product capability has an owner domain/feature/command group and source paths. | Boundaries are known but not enforced by file shape. |
| 3 | Domain-owned implementation | Domain logic, schema, service, handlers, UI feature, CLI commands, tests, and fakes live with the owning capability where appropriate. | Shared infrastructure and domain code can still blur. |
| 4 | Boundary and seam contract | Cross-cutting infrastructure is business-vocabulary-free; seams are registered in `SEAMS.md`; tests align with the new boundaries. | Manual registration and docs can drift. |
| 5 | Programmatic drift checks | Validation catches stale manifests, misplaced docs, forbidden generic buckets, production imports from test utilities, stale generated artifacts, or missing registrations. | Remaining drift requires new validator capability. |

Do not treat the level as a score to inflate. Use it to identify the next concrete move: document, map, relocate, register seams, or add validation.

---

### 3. The Core Model: Surfaces, Domains, Infrastructure

Start by separating three concepts.

**Surfaces** are how the product is exposed:
- API: core business logic, durable state, integrations, persistence, workflows.
- UI: feature-shaped presentation and browser interaction.
- CLI: thin operator/agent wrapper over API behavior.
- Contracts/proto/schema: canonical wire shape when the scenario uses generated contracts.

**Domains/features** are product vocabulary:
- If deleting the capability should delete the code, the code belongs with that domain/feature.
- If a helper exists only for one domain's tests, keep it domain-local.
- If a file name would make sense to a user or operator of this scenario, it is probably domain vocabulary.

**Shared infrastructure/substrate** is generic mechanics:
- It has no product vocabulary.
- It is used by unrelated domains.
- Examples: clocks, middleware, server composition, outbound HTTP clients, logging, generic test harnesses, generated-code registries.

For Go API code, avoid confusing "internal" with "infrastructure." Domain packages often live under `api/internal/<domain>/`; infrastructure packages live under `api/internal/<substrate>/`.

---

### 4. Domain Archetype Decision Model

Choose the shape by behavior, not by folder uniformity. First choose a primary archetype, then add secondary traits.

| Archetype | Use when | Typical traits |
|---|---|---|
| CRUD / entity | One concept is created, read, updated, deleted, listed | repository, schema, service, handler, UI list/card, CLI list/create/get |
| Temporal workflow | Correctness depends on allowed state changes over time | workflow, matrix tests, trace tests, invariants, spec status |
| Binary / blob | Opaque files, streams, imports, exports, attachments | blob store seam, multipart/stream edge, metadata persistence |
| Integration / client | External service or another scenario is wrapped | client seam, retry/idempotency, webhook handler, contract tests |
| Orchestration | Multiple domains/resources must be coordinated | planner/orchestrator, policy seams, checkpoints, progress state |
| Reporting / query | Read-heavy aggregation, dashboards, exports | query repository, views, report DTOs, cached summaries |
| Policy / rules | Decision logic is the core capability | pure rule functions, decision tables, table tests |
| Configuration / settings | User/operator-configurable behavior | defaults, validation, versioned shape, migration path |

Decision rule:

```text
Does the folder name describe a product capability?
  -> domain package / UI feature / CLI domain group
Does the code disappear when that capability is deleted?
  -> keep it domain-owned
Is it generic mechanics used by unrelated capabilities?
  -> shared infrastructure, no product vocabulary
Does correctness depend on state over time?
  -> pair with temporal-flow-audit inside the owning domain
Does it wrap an external system?
  -> integration/client trait with an explicit seam
```

Example classification (using placeholder identifiers — substitute with your scenario's actual domain; see the template-example callout at the top of this skill):

```text
<domain>
  primary: CRUD / entity
  secondary: binary/blob for attachments, temporal workflow for upload states
  API: service, repository, schema, attachment workflow
  UI: <Resource>Card, AttachmentUploadWorkflow
  CLI: <domain> commands
  shared infrastructure: BlobStore seam, modeltest helpers
```

---

### 5. Canonical File-Shape Guidance

Use scenario-local conventions first. For React-Vite-style scenarios, the healthy target is:

```text
api/
  handlers/<domain>/          # transport edge and module constructor
  internal/<domain>/          # domain types, service, repository, schema, workflows, mocks
  internal/<substrate>/       # infrastructure with no product vocabulary
ui/src/
  features/<domain>/          # domain UI and feature-local state/workflows/mocks
  api/<domain>.ts             # per-domain API client wrapper
  components/                 # cross-feature components
  test-utils/                 # cross-feature test helpers only
cli/
  domains/<domain>/           # thin commands for the domain
```

Equivalent shapes are fine for other stacks. Preserve these invariants:
- API owns business logic; UI and CLI do not duplicate it.
- Domain-specific fakes and fixtures live with the domain.
- Cross-domain test helpers live in shared test utilities.
- Shared infrastructure must not absorb product vocabulary.
- Central registries should be thin registration points, not logic owners.

---

### 6. Audit Workflow

1. **Read the docs.** Start with `ARCHITECTURE.md`, `SEAMS.md`, and `PROBLEMS.md`. Treat docs as claims to verify against code.
2. **Build or validate the mental model.** Identify product capabilities, main flows, surfaces, data ownership, and integration boundaries.
3. **Map current physical structure.** Find entrypoints, handlers/routes/jobs, domain packages/features, infrastructure packages, schema files, test fakes, and registration files.
4. **Assign maturity levels.** Score API, UI, CLI, and docs separately using the maturity model.
5. **Classify domains by archetype.** Name primary archetype and secondary traits for each meaningful capability.
6. **Find drift.** Look for generic buckets, god files, scattered capability logic, misplaced fakes, docs that promise domains that code does not show, and tests that assert through the wrong layer.
7. **Improve incrementally.** Prefer one high-confidence local move over a broad restructure.
8. **Update docs.** Put durable model changes in `ARCHITECTURE.md`, seam changes in `SEAMS.md`, and unresolved drift in `PROBLEMS.md`.

Discovery prompts:
- What are the first five folders a new agent sees, and do they explain the product?
- If I delete this capability, which files should disappear?
- Which files have product vocabulary but live under generic infrastructure?
- Which files have generic infrastructure names but contain domain rules?
- Are UI and CLI thin surfaces, or do they recreate API decisions?
- Are fakes/test helpers domain-local when only one domain uses them?
- Does each seam have one production wiring point and a documented test fake?

---

### 7. Safe Refactoring Guidelines

You may:
- rename modules, functions, components, and types to better match product vocabulary
- move code into more appropriate domain, feature, CLI-domain, or infrastructure folders
- split god files into cohesive units with clear ownership
- move one-domain fakes from shared testutil into the domain
- move cross-domain helpers out of domains into shared infrastructure/testutil
- add or adjust tests to match the clarified boundary
- add domain archetype notes and maturity status to architecture docs

You must:
- preserve observable behavior and public contracts unless the user explicitly approves a contract change
- keep refactors small enough to validate in this loop
- update imports, tests, docs, and registration points consistently
- avoid weakening tests or deleting meaningful coverage to make moves pass
- record broad/high-risk redesigns in `PROBLEMS.md` rather than half-implementing them

Challenge yourself before making a move:
- Does this make the product capability easier to find?
- Does it make ownership clearer?
- Does it reduce future coordinated edits?
- Would a second agent naturally make the same decision from the docs?

---

### 8. Documentation

Use `knowledge-observatory-tools` to read and update stable docs.

`ARCHITECTURE.md` is the durable mental model. It should include:
- the scenario's surfaces and their responsibilities
- domain map with source paths
- domain archetypes and secondary traits
- shared infrastructure map
- architecture maturity by surface
- major deviations from the scenario/template norm

`SEAMS.md` is the boundary registry. It should include:
- interfaces/modules that production wires once and tests substitute
- production wiring point
- test fake or harness
- why the seam exists
- architecture alignment notes for boundary decisions

`PROBLEMS.md` is unresolved drift. It should include:
- architecture smells not fixed now
- broad refactors deferred for risk
- documentation/code divergence that requires a later pass
- stale standalone audit reports that should be folded into stable docs

Avoid creating `SCREAMING_ARCHITECTURE_AUDIT.md` by default. A one-off audit report is acceptable only for a migration or handoff that cannot fit the stable docs yet; it should have a clear retirement path into `ARCHITECTURE.md`, `SEAMS.md`, or `PROBLEMS.md`.

Recommended architecture sections:

```markdown
## Domain Map

| Domain | Surface(s) | Primary Archetype | Secondary Traits | Source Paths | Notes |
|---|---|---|---|---|---|

## Shared Infrastructure

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|

## Architecture Maturity

| Surface | Level | Evidence | Remaining Drift |
|---|---|---|---|
```

Recommended seams addition:

```markdown
## Architecture Alignment Notes

| Area | Drift | Decision | Follow-up |
|---|---|---|---|
```

---

### 9. Output Expectations

By the end of this loop, the scenario should:
- have a clearer product mental model in docs and code
- make product capabilities easier to find and delete
- separate domain-owned implementation from shared infrastructure
- keep API business logic out of UI and CLI surfaces
- have seams and tests aligned with the clarified boundaries
- record unresolved architecture drift in stable documentation

Avoid superficial changes. The goal is not a prettier tree; it is a codebase where future agents can predict where a change belongs.
