# Progress — Web Search

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-09 | idea-workshop session | done | Scenario scaffolded from `react-vite` (1.0.1) template + `vrooli-default` design kit. Charter + requirements + concept/ops docs formalized; implementation NOT started. See below. |

### 2026-06-09 — Scenario formalized (docs/PRD/requirements only; no implementation)

What landed:
- **Generated** the `web-search` scenario (template `react-vite` 1.0.1, design `vrooli-default`). Post-hooks (proto codegen, deps) intentionally NOT run — this pass is documentation only.
- **PRD** authored via `prd-control-tower` from a full workshop context file and published to `PRD.md`: 8 P0 + 7 P1 + 3 P2 operational targets capturing the L0–L3 ladder, the two-provider scope split, the findings/briefs store, research-and-reconcile, trust/freshness, and rate-safety.
- **Requirements** generated via `prd-control-tower` — 18 modules, one per operational target (`requirements/01-...` through `18-...`), all 18 linked (`requirements validate` healthy). Starter `01-foundation` removed.
- **Concept docs** rewritten scenario-specific: `DOMAINS.md` (livesearch/findings/research/federation/curation[P2]/health), `DATA.md` (finding/brief schema + own SQLite + Qdrant index, no KO writes), `FLOWS.md` (scope-aware query, L3 research-and-reconcile, finding status machine, budget governor — all L1 inventory), `INTEGRATIONS.md` (SearXNG/Qdrant/Ollama/reranker/browserless[P1]/search-hub/agent-manager[P1] contracts).
- **Dependencies** declared in `.vrooli/service.json` (resources + scenarios with degraded_behavior).
- **Operations + business + internal + reference docs** filled scenario-specific (DEPLOYMENT/RUNBOOK/OBSERVABILITY, MONETIZATION/GO-TO-MARKET, SECURITY/PERFORMANCE, ARCHITECTURE/UI-ARCHITECTURE intros, reference surface docs).
- **DECISIONS.md** records the durable workshop decisions (incl. the rejected "promote to KO" path).

Orientation gate status after this pass (`vrooli scenario orient web-search`):
- PASS: charter, requirements-registry, domain-map, design-language, first-real-vertical-slice (optional), progress-handoff.
- OPEN (implementation — out of scope for this pass): scaffold-health (`make test`), example-domain-removed (delete `notes`).

Next (implementation): start with `livesearch` (L0) per Gate 6 — proto-first under `packages/proto/schemas/web-search/v1/livesearch/`, build the SearXNG client + normalization, register `web-search.live` + `web-search.learnings`, then `findings` store, then remove the `notes` example (Gate 7). **Verify the SearXNG resource is healthy on this host before starting P0.**

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
