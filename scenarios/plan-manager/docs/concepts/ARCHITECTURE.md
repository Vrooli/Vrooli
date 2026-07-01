# Architecture — Plan Manager

## Purpose Of This Document

The system map for plan-manager: its surfaces, where business logic lives, the
contracts between layers, the shared infrastructure it stands on, and the rules
for extending it. Read this before changing structure. The product capabilities
themselves are mapped in [`DOMAINS.md`](DOMAINS.md); the data they persist is in
[`DATA.md`](DATA.md); the workflows are in [`FLOWS.md`](FLOWS.md).

The one architectural idea that shapes everything: plan-manager is a **planning
runtime and plan-logic SSOT**. It re-homes plan logic that is otherwise scattered
(swarm-manager `phased-plan-drain`, the prose authoring skill, project hygiene,
and legacy root plan storage) so that authoring and executing plans is cheap enough
for a local model. It does that by moving judgment into deterministic code (the
wizard + validators) and by injecting context just-in-time during execution.

## Scenario Shape

Standard Vrooli scenario shape: one API process, a CLI, and a UI, all over typed
proto contracts.

- **API** (`api/`): Go, Connect-RPC handlers; business logic in `api/internal/<domain>/`.
- **CLI** (`cli/`): Go, cli-core `ScenarioApp`; thin translation over the API.
- **UI** (`ui/`): React + Vite + Tailwind; thin translation over the API.
- **Contracts** (`packages/proto/schemas/plan-manager/v1/<domain>/`): proto authored first; codegen produces Go + TS + Python.
- **Domains**: `plans` (the structured-record SSOT), `authoring` (Composer), `execution` (Runner), `validation` (Ledger), `log` (the execution-log ledger), plus scaffold `health`.

## Scenario Shape Diagram

```
            authoring (Composer)        execution (Runner)        validation (Ledger)
                   │                          │                          │
                   └──────────── operate on ──┴──────────── delegate ────┘
                                          ▼
                                    plans (SSOT)
                                          ▼
                          ~/.vrooli durable home store (SQLite/file)
                                          ▲
                    Plan Manager CLI/UI/API read the structured record
```

## System Boundaries

- **Business rules live in `api/internal/<domain>/` only.** CLI and UI never own plan logic; they call the API.
- **Domain isolation.** `authoring`, `execution`, `validation`, and `log` operate *on* plans but do not own the plan record — they delegate persistence to `plans`. `execution` surfaces validation but delegates the computation to `validation`, and reads the typed work-product ledger from `log` (it does not own it).
- **Composition boundary.** Substrate plan-manager does not own — code-facts, git/git-control-tower baseline, the scenario-artifact freshness engine, test-genie/scenario-validation, and the scenario `.vrooli/service.json` maturity read behind the **work-posture seam** (`api/internal/plans/posture.go`) — is reached through seams (see [`../internal/SEAMS.md`](../internal/SEAMS.md)) and always degrades gracefully (a missing/unreadable maturity defaults the posture to Greenfield).
- **Hard exclusions.** plan-manager does **not** read agent transcripts, spawn agents, *auto*-promote candidate findings to bugs (promotion is explicit, agent/operator-driven via `log promote`), or own project-level validation. Those are explicitly other owners' jobs (see [`INTEGRATIONS.md`](INTEGRATIONS.md)).

## Contracts And Data Flow

1. A proto `service` per domain defines the wire contract; handlers implement the generated `*ServiceHandler`.
2. The CLI binds each command to a `connect-rpc` `service`/`method` via `cliapp.LoadFromManifest`; the UI calls generated Connect clients.
3. Plan + phase records persist to the durable `~/.vrooli` home store (see [`DATA.md`](DATA.md)); Plan Manager owns the structured record and rendered mirrors.
4. Authoring writes a structured plan; execution reads/advances it (and reads the `log` ledger summary for just-in-time context and the handoff); the `log` domain captures the agent's typed work products (decisions/findings/bugs/records/notes) in-flow; validation resolves references, computes staleness, and runs baselines; the rendered markdown view is always derived from the structured record, never parsed back.

The structured shape of what flows here is defined once in [`PLAN-MODEL.md`](PLAN-MODEL.md).

## Shared Infrastructure

- `api/internal/server/` — HTTP composition.
- `api/internal/modules/registry.go` — proto/endpoint/schema registry for boot + codegen parity.
- `api/internal/database/` — home-store-rooted SQLite plumbing (api-core/storage).
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/`, `ui/src/test-utils/` — shared UI primitives + test support.
- i18n, accessibility selectors, and design tokens from the template/design kit are durable seams to preserve.

## Extension Rules

- Start in proto: author the `service` block first, run `make generate`, then write handlers/CLI/UI against generated constants and clients. Never hand-write `Path:` literals in an `EndpointDescriptor`.
- Register every new proto file in `api/internal/modules/registry.go`; the global `TestProtoConnectParity` then covers it.
- Mirror the slice across proto → API domain → API handler module → CLI domain → UI feature → i18n strings → selectors → tests.
- Generated files are regenerated, not hand-edited.
- New cross-scenario reads go through a seam and must degrade gracefully.

## Architecture Maturity

Implemented. All product domains — `plans`, `authoring`, `execution`,
`validation`, and `log` — ship as real vertical slices (proto → `api/internal/`
→ Connect handlers → CLI → UI), backed by the durable `~/.vrooli` home store; the
scaffold `notes` example has been removed. The PRD, requirements registry, domain
map, this document, the [`PLAN-MODEL.md`](PLAN-MODEL.md) keystone,
[`DATA.md`](DATA.md), [`FLOWS.md`](FLOWS.md), and
[`INTEGRATIONS.md`](INTEGRATIONS.md) track that implemented contract. The remaining
maturity step is adoption: inverting Swarm Manager phased execution and wiring
the deferred downstream forwarding adapters. Root `vrooli plans` has been retired
in favor of direct Plan Manager CLI usage.

## Intentional Deviations

- **Storage is rooted at the shared `~/.vrooli` home store, not a scenario-private DB.** Deliberate: plan data is durable outside the scenario process and plan-manager owns the schema/logic; the home store is the persistence substrate. See [`DATA.md`](DATA.md) and `docs/internal/DECISIONS.md`.
- **The prose handoff is intentionally not owned here.** It requires reading transcripts, which violates this scenario's boundary; the orchestration layer owns it.

## Documentation Architecture

- `docs/concepts/` — durable product/system concepts (this doc, DOMAINS, PLAN-MODEL, DATA, FLOWS, INTEGRATIONS, UI-ARCHITECTURE).
- `docs/reference/` — stable lookup (API, CLI, configuration, UI manifest).
- `docs/internal/` — developer/agent memory (SEAMS, TESTING, DECISIONS, PROBLEMS, PROGRESS, SECURITY, PERFORMANCE, ERROR-HANDLING).
- `docs/operations/` and `docs/business/` — runtime and commercial framing.
- `docs/manifest.json` is the machine-readable contract; new durable docs are registered there.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and ownership
- [`PLAN-MODEL.md`](PLAN-MODEL.md) — the structured plan + phase schema
- [`FLOWS.md`](FLOWS.md) — workflows and state machines
- [`DATA.md`](DATA.md) — storage and data ownership
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
