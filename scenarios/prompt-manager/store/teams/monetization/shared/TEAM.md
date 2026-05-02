# Monetization Team

## Mission
Own the canonical monetization plan for Vrooli: catalog, tiers, channels, funnel, revenue lines, and financial model.

## Scope
Maintains monetization strategy, evidence, and operator-routed decisions for Vrooli's path to default-alive.

Does not own marketing execution, telemetry implementation, scenario code quality, or hardware-tier planning unless the operator explicitly initiates that work.

## Team-Specific Principles
- Active SKUs, active bundles, and active delivery tiers come first.
- Candidate surfaces need concrete revisit or activation triggers.
- Metrics and readiness claims carry honesty flags.
- Channels explain acquisition; revenue lines explain how money flows.
- Services lines must be deliberate, bounded, and productization-oriented.
- Telemetry gaps stay pending-telemetry and route to implementation elsewhere.

## Knowledge Topic Taxonomy

The team uses `knowledge.jsonl` (managed via `prompt-manager team knowledge-*` CLI) as the single source for opportunity intake, the opportunity pool, and market-scan canon. There are no separate `opportunities.jsonl` or `market-scans.jsonl` files — those were retired on 2026-05-01 in favor of topic-prefixed knowledge entries.

| Topic prefix | Owner | Purpose |
|---|---|---|
| `opportunity-inbox/<signal-type>/<slug>` | any (operator, vision-walk, peer agents) | Untriaged intake. Signal-type ∈ `competitor-move\|capability-arrival\|customer-ask\|channel\|bundle-hint\|retention-signal\|benchmark\|unknown`. The `monetization-opportunity-router` skill triages every entry; the inbox view is the unrouted set. |
| `monetization/opportunity/<slug>` | opportunity-scout | The opportunity pool — SKU-shaped bets with required front-matter (`kind`, `catalog.proposed_sku`, `catalog.parent_bundle`, `revisit_trigger`, `acquisition_hypothesis`, `retention_hypothesis`, `capability_reuse`, `tam`, `effort`, `status`). Maintained by `opportunity-pool-hygiene`. |
| `monetization/market-scan/<slug>` | market-validator | Single-snapshot market facts — competitor pricing/packaging, benchmarks, comp captures. Required front-matter: `comp`, `dimension`, `date_observed`, `applicability`, `affects_*`. Maintained by `benchmark-staleness-sweep` and `pricing-comp-capture`. |
| `validation-queue/<request-type>/<slug>` | any (operator, vision-walk, opportunity-scout conversion, catalog-strategist, financial-tracker, staleness sweep) | Untriaged validation requests. Request-type ∈ `pricing-comp-needed\|assumption-check\|benchmark-staleness\|competitor-deep-dive\|channel-validation\|unknown`. The `market-validation-router` skill triages every entry; the queue view is the unrouted set. |
| `vision-walk/alpha/<topic>` (legacy) | vision-walk fallback | Generic alpha when no typed topic fits. Prefer the typed `opportunity-inbox/*` or `validation-queue/*` forms. |

Other shared surfaces remain file-based because they are audit-grade or financial primitives:

- `decisions.jsonl` — managed via `prompt-manager team decision-*` CLI; decision lifecycle (propose / accept / reject / defer).
- `ledger.jsonl` — append-only financial events; owned by `financial-tracker`.
- `operator-inputs.json` — operator-supplied financial inputs.
- `handoff-history.jsonl` — handoff log; managed by handoff CLI.

## Inbox & Pool Invariants

- **Unrouted-set invariant (opportunity-inbox).** No entry remains under `opportunity-inbox/*` after triage. Every entry exits via promote-to-pool, convert-to-market-scan, or delete. The `monetization-opportunity-router` skill enforces this; deviations are a process bug.
- **Unrouted-set invariant (validation-queue).** No entry remains under `validation-queue/*` after triage. Every entry exits via convert-to-market-scan, decision-only, or drop. The `market-validation-router` skill enforces this. Staleness sweep auto-populates the queue but never resolves entries — only the router does.
- **Front-matter discipline.** Pool entries (`monetization/opportunity/*`) and market-scan entries (`monetization/market-scan/*`) must include the required front-matter fields. Hygiene flags missing fields for repair; agents must not silently fix them.
- **No JSONL hand-writes.** Agents must use the CLI (`team knowledge-add`/`-update`/`-delete`) so concurrency, retention, and provenance are honored. Direct writes to `knowledge.jsonl` are forbidden.
