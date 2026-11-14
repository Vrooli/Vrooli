# Integrations — Architecture Cartographer

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API (all cartographer domains) | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| `code-facts` | scenario, runtime | yes | `graph`, surface and endpoint analysis | `CodeFactsService.DescribeCodeFacts` with `include_graphs`; source identity, parse units, cached language graphs and extraction limits | Unavailable producers yield explicit partial evidence; cancellation, invalid targets and internal errors remain errors. |
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

### `code-facts`

Code Facts owns source discovery, parse-unit admission, source/configuration identity,
and the cache shared by facts and language graphs. Cartographer requests unpaged,
bounded graph envelopes through `api/internal/graph/codefacts/client.go`, preserves
extraction profiles and omissions, and namespaces node identities by parse unit.
It owns architectural interpretation and immutable snapshots, including historical
partial results. Snapshot identity includes source and skipped-provider evidence so
provider recovery cannot return a stale partial result.

Go and TypeScript parser scenarios remain transitive producers behind Code Facts.
Cartographer has no private project resolver, source fingerprint scanner, or direct
parser extraction clients. Requests for future rewrite operations must use their
own owner contract; graph extraction does not imply mutation authority.

### `react-component-library` (no runtime dependency from cartographer)

- **Status**: not a runtime dependency of architecture-cartographer.
- **Relationship**: react-component-library will become a *consumer*
  of `typescript-code-graph`, not of cartographer. ui-health continues
  to depend on react-component-library for React-specific semantics
  (widgets, slots, template versions). Cartographer reads only the
  cached TS graph from Code Facts; React-specific data is not
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
| Code Facts or language producer unavailable | Typed transport error or graph-envelope warning | Record skipped extraction as partial; preserve successful parse units. Retry on the next request through the shared owner. | `internal/graph/codefacts/client_test.go`, `internal/graph/service_test.go` |
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
