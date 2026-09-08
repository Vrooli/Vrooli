# Progress — Search Hub

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/search-hub/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| 2026-08-14 | — | codex | Archived |

| 2026-08-12 | codex | Honest-signal implementation completed at the code and contract layer. | Archived |

| 2026-08-11 | codex | Trustworthy retrieval and honest readiness plan completed. | Archived |

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-07-07 | agent | Search maturity certification hardening | Archived |
| 2026-06-30 | agent (reranker chain) | Search Hub production rerank now uses the shared `ai-go/search` chain through `routing.SharedReranker`: TEI cross-encoder primary, Ollama `rerank.llm_fallback` fallback, then honest by-provider grouping through the existing router degradation path. | Archived |
| 2026-06-11 | agent (routing resilience) | Search Hub query now has server-side timeout resilience around unified reranking. | Archived |
| 2026-06-04 | agent (eval domain) | Search-quality baseline harness | Archived |
| 2026-06-03 | agent (Phase 9) | UI, validation, cleanup — **the final phase; plan COMPLETE**. | Archived |
| 2026-06-03 | agent (Phase 1) | Scaffold generated; PRD + 11 requirement modules + proto (registry/routing) landed; deps set (postgres/ollama required, 5 soft providers); green end-to-end. | Archived |
| 2026-06-03 | agent (Phase 2) | Orientation docs authored: real `DOMAINS.md` (registry/providers/routing/rerank/metrics, mapped to requirement modules + build phases); `DATA.md` (postgres registry+telemetry, no corpus tables); `ARCHITECTURE.md` (thin-router invariant + storage diagram + intentional deviations); `INTEGRATIONS.md` (dependency-decisions rationale). 6/8 orient gates green. | Archived |
| 2026-06-03 | agent (Phase 2) | deferred | Archived |
| 2026-06-03 | agent (Phase 4) | Router core: explicit-type fan-out. | Archived |
| 2026-06-03 | agent (Phase 5) | Classifier / automatic routing. | Archived |
| 2026-06-03 | agent (Phase 6) | Cross-encoder rerank / unified ranking. | Archived |
| 2026-06-03 | agent (Phase 8) | Register all live providers + stub the gaps (federation coverage). | Archived |
| 2026-06-03 | agent (Phase 7) | Metrics & measurement — the validation backbone. | Archived |
| 2026-06-03 | agent (Phase 3) | Provider registry + registration contract shipped. | Archived |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-08-12 — Honest-signal implementation completed at the code and contract layer.**: Live substrate certification and provider-owner dense→hybrid before/after evidence remain explicitly pending under concurrent Ollama/Qdrant restoration; the dense/hybrid owner boundary is documented in `PROBLEMS.md`.

- **2026-07-07 — Search maturity certification hardening**: **Live full scan: architecture-cartographer, business-health, cli-health fully certify** (fresh runs, live labels, recall 1.0/0.93≥0.8, p95 172–342ms); knowledge-observatory→`SEARCH_EVAL_LABELS_STALE` (needs evals run); measures-health/swarm-manager/web-search/workflow-health fail with real corpus gaps (domain corpus authoring filed as follow-up — not fabricated).

- **2026-06-11 — done**: `Router.Query` applies a 25s default total query budget below the CLI's 30s default, rerank defaults to 5s and is clipped to the remaining query budget, one-candidate result sets skip rerank without marking degraded, and repeated reranker failures open a generic routing-owned circuit breaker that skips rerank until a cooldown permits a half-open probe.

- **2026-06-04 — Search-quality baseline harness**: Live A/B (rerank-off vs cross-encoder vs llm-qwen3) is the attended follow-up.

- **2026-06-03 — done**: **Follow-up surfaced (§9):** this plan delivered **axis ① for currently-live providers only**;

- **2026-06-03 — deferred**: `--finalize` deferred to end of Phase 3: the two remaining gates (`example-domain-removed`, `scaffold-health`) require the first real domain (`registry`) to replace `notes`, per START-HERE ("build a real domain, prove green, then remove the example"). `scaffold-health` is also blocked by a scaffold-template **false-positive** standards critical (`api/internal/httpc/doer.go:34` — a compile-time `var _ Doer = (*http.Client)(nil)` assertion, not a real timeout-less client).
