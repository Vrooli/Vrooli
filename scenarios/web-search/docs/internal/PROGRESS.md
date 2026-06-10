# Progress — Web Search

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-09 | idea-workshop session | done | Scenario scaffolded from `react-vite` (1.0.1) template + `vrooli-default` design kit. Charter + requirements + concept/ops docs formalized; implementation NOT started. See below. |
| 2026-06-09 | two-agent build | done | Full implementation landed: all 8 P0 + 6 of 7 P1 targets shipped and live-validated (SearXNG, scenario, both search-hub providers healthy). OT-P1-003 PARTIAL (bounded sweep / answer-first ordering enforced only by the L3 prompt, not the API). P2 (0/3) unbuilt. See verified-state entry below. |
| 2026-06-09 | completion plan | done | OT-P1-003 hardened (API-enforced bounded gather + answer-first ordering); OT-P2-001/002/003 implemented; live L3 run validated end-to-end against agent-manager. See completion entry below. |

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

### 2026-06-09 — Verified build state (post two-agent build, pre-completion)

The "documentation only" status above was superseded by a full two-agent build that the
build session never logged here. Verified by 3 independent code audits + live runtime:

- **P0 (8/8 DONE, live-validated):** L0 SearXNG search + L1 snippet synthesis
  (`api/internal/livesearch/`), dual provider registration (`api/searchreg.go` +
  `.vrooli/search.json`: `web-search.live` SCOPE_EXTERNAL + `web-search.learnings`
  SCOPE_PROJECT), scope-aware blending, findings store (SQLite + aisearch-go,
  `api/internal/findings/` + `api/internal/findingindex/`), findings CLI, TTL cache +
  token-bucket governor, core UI. SearXNG healthy; both providers `active` in search-hub;
  a federated `--all` query reaches both.
- **P1 (6/7 DONE, 1 PARTIAL):** L2 fetch/extract/synthesize, L3 agent-manager
  spawn/poll (`api/internal/research/`), auto-capture, contradiction + audit trail,
  trust/freshness (age-decay half-life 180d), dispute UI — all real and tested.
  **PARTIAL: OT-P1-003** — Gather (bounded sweep) → answer-first → Reconcile ordering
  lived only in the L3 prompt string; the API exposed `Reconcile()` (0.75 gate) but did
  not enforce/verify the bounded sweep or answer-first ordering.
- **P2 (0/3 MISSING):** OT-P2-001/002/003 unbuilt.

### 2026-06-09 — Completion pass (OT-P1-003 hardening + P2 + live L3)

What landed (web-search-scenario-completion plan):
- **Live L3 validated end-to-end** against a running agent-manager: a real CLI
  `research l3` run spawned an agent-manager run, completed, emitted a cited brief, and
  auto-captured 2 findings (confidence 0.95/0.90, source `l3`). First non-fake L3 evidence.
- **OT-P1-003 hardened** — see completion record / Phase 1 below for the API-enforced
  bounded gather + answer-first ordering.
- **OT-P2-001/002/003** implemented and wired (effectiveness ledger; search-hub
  classifier auto-routing default-OFF; periodic full-store GC).
- All 18 PRD operational targets now checked. api/cli/ui/search-hub builds + the
  `tests` phase (unit/integration/smoke) green; `golangci-lint`/`gofumpt`/`tsc`/`eslint`
  clean. Regression: GCT baseline diff `web-search` = preexisting/exit 0, `search-hub` =
  clean/exit 0.
- KNOWN: the `standards` test phase stays RED on **pre-existing scanner false-positives**
  (GCT-confirmed inherited from baseline, not this work) — see PROBLEMS.md and bug
  `knw-1781048636072856962`.

Per-phase detail + the substantive decisions are in execute record `rec-9f2610c3a99e6c33`.

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
