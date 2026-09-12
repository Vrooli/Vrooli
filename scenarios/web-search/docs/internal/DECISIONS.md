# Decisions — Web Search

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
| 2026-06-09 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-06-09 | **Web search is a new scenario, not folded into search-hub.** | search-hub is a thin router with a hard invariant: zero corpus, zero provider-specific code, zero external calls (architectural tests enforce it). Web search needs SearXNG-calling, caching, governing, and a corpus. | web-search is a provider; search-hub stays thin. The federation contract is proven to generalize beyond pre-indexed Qdrant corpora to a live external passthrough. | Only if the thin-router invariant is intentionally abandoned. |
| 2026-06-09 | **Two providers split by scope:** `web-search.live` = `SCOPE_EXTERNAL`, `web-search.learnings` = `SCOPE_PROJECT`. | Need web reachable but rate-safe; need internalized knowledge always available; need both blended. | The existing scope-aware routing makes a default federated query hit learnings (project) and NOT live web (external) — resolving rate-safety AND blending AND "smarter over time" in one mechanism, no special code. | If search-hub's scope semantics change. |
| 2026-06-09 | **Own a self-curating findings store; do NOT write into knowledge-observatory.** | Considered "promote to KO docs"; rejected. Unvetted external content must not pollute the curated trusted corpus, and doc files are the wrong substrate. | web-search keeps findings in its own SQLite (api-core storage) + aisearch-go index (cli-health pattern). It is a peer corpus, clearly labeled external, never KO documentation. | If a vetted graduation path into KO is ever designed deliberately. |
| 2026-06-09 | **Depth ladder L0–L3, each a distinct mechanism.** L0 raw, L1 snippet-synth, L2 fetch+synth, L3 iterative. L0/L1 = P0; L2/L3 = P1. | "Depth" is not one dial — each level adds a mechanism; L2→L3 crosses from pipeline to agent. | Clear prioritization; L0/L1 ship first. | — |
| 2026-06-09 | **L3 = an agent-manager run, not hand-rolled loop plumbing.** | Vrooli already has agentic loop infra (agent-manager). | L3 reuses proven infrastructure; web-search supplies L2 tools + findings commands the agent calls. | If agent-manager cannot support the run shape. |
| 2026-06-09 | **The L3 agent is also a librarian (research-and-reconcile).** Reads existing findings first, then reconciles (supersede/flag) as a bounded post-step. | Avoids re-deriving known knowledge; keeps the store consistent as a side effect of use. | Curation rides along on research; no separate background curation needed at P0/P1. | — |
| 2026-06-09 | **Terminology: `finding` (atomic cited claim, the indexed unit) + `brief` (one research run container).** | Chosen over "insight"/"learning" for precision. | Used across schema, API, CLI, UI, docs. | — |
| 2026-06-09 | **Findings are never hard-deleted — supersede/archive only; every mutation audited.** | Protects against an agent (or human) destroying good knowledge; gives undo + audit trail. | `superseded` rows kept in place with `superseded_by`; `finding_audit` append-only. | — |
| 2026-06-09 | **Disputed findings are surfaced WITH a warning + both sources; never silently resolved.** Superseded excluded from default search, retrievable via `--include-archived`. | Honesty over false confidence; archived knowledge stays recoverable but doesn't leak into normal results. | Default search filter `status != superseded`; disputed shown with `dispute_note`. | — |
| 2026-06-09 | **Auto-capture policy: L3 on by default, L2 opt-in by flag, L0/L1 never persist.** | L3 is deliberate/expensive (worth persisting); L0/L1 are shallow/ephemeral. | Distillation pass emits findings only at L3 (default) / L2 (flag). | — |
| 2026-06-09 | **Rate-safety via TTL cache + token-bucket budget governor; default query never fires a live web call.** | External engines are rate-limited; SearXNG must not be hammered. | Governor returns graceful "rate-limited, try later"; cache dampens repeats. | — |
| 2026-06-09 | **Defer to P2: usage-telemetry curation, classifier auto-routing to live web, periodic full-store GC.** | Per-query reconcile + age-decay are expected to suffice initially; classifier auto-routing is the highest rate-limit risk. | These are planned requirements (OT-P2-001..003), not built; not corner-painting. | When the store reaches volume / reconcile proves insufficient. |
| 2026-06-10 | **Findings export / cross-instance sharing stays DEFERRED (decision only, no implementation).** | Close-out review weighed implementing an export path now. No consumer exists yet, the store is days old, and an export format chosen before a real cross-instance use case would be speculative. The data already has a natural seam: `LoadIndexable` (repository read of all non-superseded findings) plus the audit trail. | No export surface ships with 1.0. The seam is recorded in [`../concepts/DATA.md`](../concepts/DATA.md) Import/Export so a future P2 can add a typed export (JSONL of finding+citations+audit) without restructuring. | A second Vrooli instance (or external consumer) actually needs the learnings corpus; or the fleet designs a vetted KO graduation path. |
| 2026-06-10 | **L2 fetch substrate = HTTP-first with per-URL escalation to browser-automation-studio; the browserless resource coupling is deleted (greenfield, owner-mandated).** | Live assessment found every production L2 fetch 404ing: the old `BrowserlessFetcher` POSTed `/chrome/content` while the deployed browserless served `/content` — and no seam-injected test could catch a substrate route bug. Owner rejected the one-line route patch: direct browser-resource coupling was the wrong shape. | `internal/research/fetch/` owns the substrate: `HTTPFetcher` (most pages are server-rendered; no browser cost) → `EscalatingFetcher` → `BASFetcher` (BAS `CaptureService.Capture` Connect-RPC with the new `inline_dom` field, so rendered HTML returns inline with zero filesystem coupling; BAS gains a fleet-wide rendered-fetch capability). Four new `WEB_SEARCH_*` levers; a NON-hermetic live smoke test pins the transport; browserless removed from service.json in favor of a `browser-automation-studio` scenario dep. | If BAS Capture's contract changes, or a dedicated fleet fetch-service scenario emerges. |
| 2026-06-10 | **Tuning control surface = six boot-time `WEB_SEARCH_*` env levers; compiled defaults stay the SSOT in their owning packages.** | Close-out surfaced the baked-in tunables (confidence gate 0.75, decay half-life 180d, gather cap 20, L3 loop budget 10, governor 60/min, cache TTL 5m). Options were config file vs env vs leaving hard-coded. Env matches the scenario's existing pattern (`WEB_SEARCH_GC_INTERVAL`) and the lifecycle's environment injection. | `api/tuning.go` parses the levers (malformed/out-of-range → compiled default, boot never blocked); documented in [`../reference/configuration.md`](../reference/configuration.md). Governor window (1m) and decay usage-curve constants deliberately NOT exposed — they shape invariants, not tradeoffs. The L3 loop budget is a task-contract directive; the hard run bound stays with agent-manager. | If operators need per-request (not boot-time) tuning, or the lever set keeps growing — re-evaluate config-file consolidation then. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| 2026-06-09 | "Promote valuable web results into knowledge-observatory." | Self-owned findings store in web-search (peer corpus, never KO docs). | Reframed during the idea-workshop; KO promotion rejected as a pollution/trust risk. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
