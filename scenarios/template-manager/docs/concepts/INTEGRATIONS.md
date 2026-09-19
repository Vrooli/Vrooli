# Integrations — Template Manager

Template Manager is an ecosystem meta-capability. Its integrations are mostly Vrooli scenarios and platform contracts, not third-party services.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite via api-core resolver | embedded storage | yes | registry, validation, debt, monitor | WAL DSN under Template Manager data dir | API health fails; write surfaces return typed unavailable/internal errors. |
| Vrooli lifecycle | local platform | yes | API, CLI, UI | `make start|test|logs|stop`, `.vrooli/service.json` | Scenario is unhealthy if started outside lifecycle contract. |
| test-genie | scenario | yes | phase-provider, validation monitor | scenario-validation/v1 provider contract; durable run wait protocol | Templates phase degrades if provider unavailable; deep validation run records failure. |
| search-hub | scenario | yes after docs phase | docs, debt | `.vrooli/search.json` provider registration | Docs/debt remain available locally but are not federated. |
| measures-health | scenario | yes after measures phase | dashboard, measures | `/measures` registry plus typed MeasuresService | Measures phase reports uncovered/malformed declarations. |
| vrooli-autoheal | scenario | yes after monitor phase | operations | critical monitored scenario defaults | Existing installs need documented config refresh; auto-heal does not manage Template Manager until registered. |
| prompt-manager | scenario | advisory | docs, learning loop | skills/actions/records discovery and capture | Work can proceed manually; reusable wins may not be indexed. |
| quality-health maturity/autofix patterns | scenario/reference | advisory | phase-provider | Autofix registry pattern | Implementation falls back to local equivalent if reference changes. |
| Existing vrooli CLI engine code | repo-local source | temporary | engine | Move code and tests, then delete old command owners | Cutover blocks until parity/e2e proof is green. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| SQLite | embedded | Default durable store for Template Manager. | Always enabled through lifecycle. |
| External object/blob store | not planned | Template Manager stores finding summaries, not large artifacts. | Deep validation artifacts need durable binary retention. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| test-genie | required | Provider conformance, templates phase, deep validation orchestration. | `scenario-validation/v1`, `test-genie runs wait --json`, provider descriptor. |
| search-hub | planned | Index factory docs and debt entries. | `.vrooli/search.json` providers. |
| measures-health | planned | Federate template standing metrics. | `/measures/declarations`, `/measures/execute`, typed MeasuresService parity. |
| vrooli-autoheal | planned | Keep Template Manager available as critical platform capability. | Default monitoring config and live status checks. |
| prompt-manager | advisory | Recall/discover/capture loop and skill/action docs. | Indexed docs, skills, records. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None | not-applicable | Template Manager is local platform infrastructure. | Add only if future template registries sync with remote catalogs. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | Resolver/open/ping error | Health reports unhealthy; write APIs fail typed errors. | health and repository tests |
| test-genie deep validation | durable run fails or times out | Validation run records failed/timed_out with diagnostic; scheduler continues later. | fake-runner and live scheduler tests |
| test-genie provider probe | provider-contract diagnostics | Phase descriptor/provider work blocks until conformance is green. | provider-contract check |
| search-hub unavailable | query misses provider | Docs/debt API remains source of truth; indexing diagnostics are surfaced. | docs/search smoke |
| autoheal not refreshed | status omits template-manager | RUNBOOK documents config refresh for existing installs. | autoheal defaults unit test |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain for each integration
- [`DATA.md`](DATA.md) — storage and retention
- [`FLOWS.md`](FLOWS.md) — integration-heavy workflows
