# Decisions — Measures Health

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-08 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-06-08 | Two domains: `validation` (coverage report + behavioral probe + fleet rollup) and `index` (central measures index + the single search-hub measures provider). | The measures plan (Phase 4) gives this scenario two distinct jobs — enforce/grade adoption, and federate/answer analytical questions. | Both built (proto/api/cli/tests green). `index` = `SearchService.Search/Status` over `internal/measureindex` (harvest → Matcher → measures-go Engine). | Revisit if the index provider needs to share state with validation (it does not today — both are read-only over the same harvested measures). |
| 2026-06-08 | The `index` Matcher is a deterministic **lexical** index over the curated `questions[]` (token overlap), NOT the plan's aisearch-go qdrant hybrid index — *yet*. | No scenario declares a `measure` block until Phase 5 (template) / Phase 6 (swarm-manager), so the corpus is **empty today**; a qdrant index would index nothing and be 100% unexercised (dead speculative code the greenfield rule forbids). The lexical leg is a genuine, offline, fully-tested production matcher for a small curated corpus (the same role cli-health's text fallback plays). | Ships a working federated provider now; `measures.Matcher` is the seam (`measures-go` already carries `MeasureComposer` for the embedding key) so the aisearch hybrid index drops in — or layers over — the lexical leg once a corpus exists, with no other change. | **Follow-up (Phase 6+):** wire the aisearch-go hybrid index (`MeasureComposer`, qdrant) behind/over `LexicalMatcher` once ≥1 scenario declares a measure, so the live retrieval path can actually be validated. |
| 2026-06-08 | The `index` is a Connect `SearchService` (proto-first, gives the CLI verb + endpoints.json), and the `MeasureHit` carrier fields carry explicit snake_case `json_name`; the search-hub descriptor points the `result_mapping.measure_field` at the Connect endpoint. | search-hub's generic result adapter reads the measure carrier with **snake_case** keys (`measure_id`, `executed_query`, …) via plain `encoding/json`; connect-go's default JSON codec (protojson) emits **camelCase** unless `json_name` is set. Single-surface proto-first beats a second plain endpoint. | The wire JSON the Connect server emits matches search-hub's adapter verbatim. Guarded by `TestSearch_WireShapeMatchesSearchHubAdapter` (marshals via protojson, asserts the exact snake_case keys) so a future proto edit that drops `json_name` fails loudly. | Revisit only if connect-go changes its default codec or search-hub's adapter switches to protojson. |
| 2026-06-08 | `ValidationService` (not `MeasuresService`) is the service name; `measures-health validate scenario` / `validate coverage` are the CLI verbs. | api-steer: one `<Domain>Service` per domain; the plan's `MeasuresService.ValidateScenario` / bare `coverage` names were illustrative. | Satisfies REQ-P1-002 (the RPC that validates a scenario's measures) under the domain-consistent name; `coverage` lives under the `validate` group. | Revisit only if an external consumer hard-codes the illustrative names. |
| 2026-06-08 | `api/internal/measurescan` is a faithful local copy of `cli-health/internal/measurescan` (manifest measure-block harvest + tier grading over `measures-go`), not a shared import. | `cli-health/internal/...` cannot be imported; promoting it to a shared `packages/measures-go` sub-package would rewire cli-health (15 refs / 4 files), expanding Phase 4's blast radius beyond its allowlist. | Transient duplication across two scenarios; both kept byte-compatible so the eventual promotion is a mechanical de-dup. | **Follow-up:** promote both copies to `packages/measures-go/manifestscan` and migrate cli-health in a focused change (out of this phase's scope). |
| 2026-06-08 | EXPECTED stateful domains derive from `packages/proto/schemas/<scenario>/v1/domain/*.proto`; a minimal name-based stateless filter (`settings`/`config`/`preferences`) + the manifest `measures.domains[]` override are the only classification levers. | The plan's verified expectation model (§250) — a domain proto == a persisted entity type. Per-message statefulness inference was over-engineering for v1. | Lead-with-inference + explicit override + waivers cover misclassification. | Revisit if the name-based filter proves too coarse — upgrade to message-shape inference. |
| 2026-06-08 | The behavioral probe never executes `write`/`destructive` measures (would mutate the target); it reports them "not probed (validated statically)". | The auto-execution gate's safety rule applies to probing too — a mutating endpoint cannot be proven without side effects. | A hollow `write` measure can pass the probe; documented honestly. | Revisit if a dry-run measure execution contract is added. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
