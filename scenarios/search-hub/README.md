# Search Hub

Search Hub is Vrooli's federated retrieval control plane. It registers
scenario-owned providers, classifies queries, fans out to eligible corpora,
reranks results, and records telemetry and evaluation evidence for useful or
degraded retrieval.

## What You Get

- Descriptor-driven provider registration and explicit or automatic routing.
- Circuit state, graded-empty demotion, decay/probation recovery, and partial
  results when a federated query reaches its deadline.
- Immutable eval suites and runs with provider-error outcomes, freshness and
  quality gates, and junk-leak withholding for automatic routing.
- Operator surfaces for status, federation, providers, insights, evals, and
  search maturity validation.
- Go API, CLI, and UI surfaces connected through generated Connect contracts.

Search Hub owns routing policy, registry metadata, telemetry, and eval mirrors.
Provider scenarios own their corpora, indexing, descriptors, and search
implementations. A provider is never added by teaching Search Hub its name.

## Running Search Hub

Use the scenario lifecycle so ports, health checks, and process ownership stay
consistent:

```bash
make setup
make start
make test
make stop
```

For an operator check:

```bash
search-hub status
search-hub federation status
search-hub providers list
search-hub insights --window 7
search-hub maturity scan --json
```

See [`docs/QUICKSTART.md`](docs/QUICKSTART.md) and
[`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md).

## Search Quality and Failure Semantics

An eval transport failure is recorded as `error` with the provider's message;
it is not treated as an empty expectation. Such a run is degraded, and a run
with zero graded cases cannot certify freshness. A fresh non-degraded run can
still be withheld from automatic routing when a gibberish-tagged case scores
at or above the strongest real-case score; the run id is reported as evidence.

Transport failures feed the circuit breaker. Only successful, non-degraded
zero-hit responses feed graded-empty demotion. Demotion decays into probation
and can recover through a successful probe. Providers marked `fixture` or
`experimental` remain available to explicit selectors but are excluded from
automatic routing.

Address resolution uses a short TTL cache and invalidates a failed entry.
Federated queries enforce a concurrency/timeout budget and return completed
groups as `partial` when their deadline expires.

## Documentation Map

| Need | Start here |
|---|---|
| First local run | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| System architecture | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Routing and eval workflows | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md) |
| CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |
| Configuration | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| Runtime recovery | [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md) |
| Known problems | [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md) |

## Customize Safely

Provider owners should update their own `.vrooli/search.json`, eval corpus,
and search implementation. Keep generic policy in Search Hub and add tests at
the injected seam before changing behavior. Run focused Go tests while
resources are unavailable, then validate the scenario through its lifecycle.

Do not hand-edit generated protobuf bindings, approved dependency manifests,
or Search Hub policy to special-case a provider. Regenerate contracts from
`packages/proto/schemas/search-hub/` and use the Scenario Dependency Analyzer
for dependency changes.
