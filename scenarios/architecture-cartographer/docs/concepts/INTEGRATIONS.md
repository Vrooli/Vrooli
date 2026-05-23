# Integrations — Architecture Cartographer

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API (all cartographer domains) | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| `go-code-graph` (scenario) | scenario, runtime | yes — for any target scenario with Go code | `graph`, `apply` domains | Connect-RPC; service `GoCodeGraphService` with methods `Extract`, `Rewrite` (planned) | Graph extraction for Go scenarios fails; cartographer reports degraded; agent prompted to start dependency. |
| `typescript-code-graph` (scenario) | scenario, runtime | yes — for any target scenario with TS code | `graph`, `apply` domains | Connect-RPC; service `TypeScriptCodeGraphService` with methods `Extract`, `Rewrite` (planned) | Graph extraction for TS scenarios fails; cartographer reports degraded; agent prompted to start dependency. |
| `git` (system binary) | local tool | yes | `signals` (git co-edit), `apply` (commits) | Shell-out via `os/exec` with non-interactive flags | Co-edit signal disabled; apply refuses to commit. |
| Go toolchain (`go build`) | local tool | yes — for Go target scenarios | `apply` (build-green guard) | Shell-out via `os/exec`; uses target scenario's Go module | Build-green guard cannot validate; refuses apply unless `--force --note`. |
| TypeScript toolchain (`tsc`, `pnpm`) | local tool | yes — for TS target scenarios | `apply` (build-green guard) | Shell-out via `os/exec`; uses target scenario's `tsconfig.json` | Build-green guard cannot validate; refuses apply unless `--force --note`. |

## Vrooli Resources

Cartographer does not require any external Vrooli resource in v1.
This is a deliberate choice — see Intentional Deviations in
[`ARCHITECTURE.md`](ARCHITECTURE.md).

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| Ollama | not-applicable in v1 | Embeddings deferred to P2 (OT-P2-001) as a suggestion-only ranker. Deterministic signals cover the v1 ladder. | Add only if v1 evidence shows high residual conflicts the deterministic ladder cannot answer. |
| Qdrant | not-applicable in v1 | Same as Ollama — required only if/when the embedding ranker ships. | Same as Ollama. |
| PostgreSQL / shared DB | not-applicable | Cartographer state is per-scenario and local; SQLite suffices. | Add only if a multi-tenant or shared-history use case emerges. |

## Scenario Dependencies

### `go-code-graph`

- **Status**: REQUIRED — scenario was initialized 2026-05-23 at
  `scenarios/go-code-graph/` (PRD, requirements, docs in place) but
  is not yet implemented. Cartographer's `graph` and `apply` domains
  block on its `Extract` and `Rewrite` services going live.
  Documented as Intentional Deviation in
  [`ARCHITECTURE.md`](ARCHITECTURE.md) and as the load-bearing
  problem in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).
- **Why this scenario, not in-process parsing**: Cartographer must
  never parse source code directly (see Architecture Rules in
  [`../START-HERE.md`](../START-HERE.md) and the
  `wrap-not-use` principle in user memory). Multiple consumers will
  reuse this scenario (cartographer + future Go static-analysis
  tools), so it earns standalone existence.
- **Contract** (planned proto, owned by go-code-graph):
  - `Extract(ExtractRequest{scenario_path: string}) returns (Graph)` —
    returns nodes (files, packages, symbols) and edges (imports,
    intra-package symbol references).
  - `Rewrite(RewriteRequest{operations: []FileMove|ImportRewrite})
    returns (RewriteResponse)` — executes mechanical refactors;
    delegated by cartographer's `apply` domain.
  - Wraps `golang.org/x/tools/go/packages` (canonical Go AST/import
    library).
- **Cartographer integration adapter**: `api/internal/graph/gocodegraph/`
  (planned) — handles client construction, retries on
  `CodeUnavailable`, URL resolution via shared scenario-discovery
  pattern (same pattern as ui-health's react-component-library
  client).
- **Failure behavior**: if scenario is unreachable when cartographer
  needs Go extraction, return a typed `IntegrationError{kind:
  scenario_unreachable, scenario: "go-code-graph"}`. CLI surfaces:
  "Cartographer needs `go-code-graph` running. Start it with
  `vrooli scenario start go-code-graph`."

### `typescript-code-graph`

- **Status**: REQUIRED — scenario was initialized 2026-05-23 at
  `scenarios/typescript-code-graph/` (PRD, requirements, docs in
  place) but is not yet implemented. Adds a Node sidecar process
  hosting `ts-morph` (REQ-P0-009 in that scenario) that must ship
  before the `graph` and `rewrite` domains can land. Same
  Intentional Deviation entry as above.
- **Why this scenario, not in-process parsing**: Identical reasoning
  to `go-code-graph`. Additionally, `react-component-library` is
  scheduled to migrate from its current regex-based parsing onto
  `typescript-code-graph` so its scan logic stops being brittle.
- **Contract** (planned proto, owned by typescript-code-graph):
  - `Extract(ExtractRequest{scenario_path: string}) returns (Graph)` —
    same shape as Go's contract; node types include components,
    hooks, types, modules.
  - `Rewrite(RewriteRequest{operations: []FileMove|ImportRewrite})
    returns (RewriteResponse)` — executes mechanical refactors via
    `ts-morph`.
  - Wraps `ts-morph` (canonical TS compiler-API wrapper) for parsing
    and refactoring.
- **Cartographer integration adapter**: `api/internal/graph/tscodegraph/`
  (planned).
- **Failure behavior**: same as `go-code-graph`.

### `react-component-library` (no runtime dependency from cartographer)

- **Status**: not a runtime dependency of architecture-cartographer.
- **Relationship**: react-component-library will become a *consumer*
  of `typescript-code-graph`, not of cartographer. ui-health continues
  to depend on react-component-library for React-specific semantics
  (widgets, slots, template versions). Cartographer reads only the
  raw TS graph from typescript-code-graph; React-specific data is not
  cartographer's concern.

### `knowledge-observatory` (validation pattern, not runtime)

- **Status**: pattern dependency only — no runtime call.
- **Relationship**: cartographer's docs surface is validated by
  knowledge-observatory's existing doc-health command (the same
  command used to validate this manifest now). The cartographer does
  not call knowledge-observatory at runtime; the integration is
  one-way (`knowledge-observatory docs health architecture-cartographer`).
- OT-P1-005 explicitly enumerates this as a P1 surface — cartographer
  findings could later be exposed to ui-health and knowledge-observatory
  via Connect-RPC, but that is not a v1 dependency.

### `swarm-manager` (future, P2)

- **Status**: P2 (OT-P2-003) — not in v1 scope.
- **Relationship**: detected conflicts and migration plans could be
  filed as initiative items in swarm-manager for human triage on
  scenarios the agent cannot complete autonomously.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None | not-applicable | Cartographer is fully local; no third-party APIs, webhooks, auth providers, or data feeds. | Add only if a future capability genuinely requires one. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests (planned) |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| `go-code-graph` unreachable | Connect-RPC `CodeUnavailable` after retries | `IntegrationError`; CLI prints actionable hint to start the scenario. | integration test with adapter pointed at non-running endpoint |
| `typescript-code-graph` unreachable | Same as above | Same as above | Same as above |
| `git` not installed | `exec.ErrNotFound` | Co-edit signal disabled; apply commits refused. | unit test substituting a fake `os/exec` runner |
| Go toolchain missing for target | `go build` exits non-zero with "go: command not found" | Build-green guard reports "cannot validate"; apply refuses unless `--force --note`. | integration test with broken `PATH` |
| TS toolchain missing for target | `tsc --noEmit` not invocable | Same as Go toolchain missing. | Same. |
| Manifest invalid (schema violation) | Parse error from manifest domain | Migration refuses to start; CLI prints schema violation with line numbers. | unit test against intentionally malformed manifest |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries and intentional deviations
- [`DOMAINS.md`](DOMAINS.md) — which domain uses each dependency
- [`DATA.md`](DATA.md) — storage ownership
- [`SIGNAL_LADDER.md`](SIGNAL_LADDER.md) — signals that depend on `git`
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — handling integration outages
