# Monetization Team

## Mission
Own the canonical monetization plan for Vrooli: catalog, tiers, channels, funnel, revenue lines, and financial model.

## Scope
Maintains monetization strategy, evidence, and operator-routed work for Vrooli's path to default-alive.

## Shared team corpus
Durable context lives in the `team:monetization` source-ledger scope. Use `source-ledger recall` and `source-ledger journal note`; file executable monetization work once through swarm-manager.

Does not own marketing execution, telemetry implementation, scenario code quality, or hardware-tier planning unless the operator explicitly initiates that work.

## Team-Specific Principles
- Active SKUs, active bundles, and active delivery tiers come first.
- Candidate surfaces need concrete revisit or activation triggers.
- Metrics and readiness claims carry honesty flags.
- Channels explain acquisition; revenue lines explain how money flows.
- Services lines must be deliberate, bounded, and productization-oriented.
- Telemetry gaps stay pending-telemetry and route to implementation elsewhere.

## Knowledge Topic Taxonomy

The team uses its source-ledger scope as the single source for opportunity intake, opportunity judgment, and benchmark evidence. Entries are prose with a typed topic prefix; no local append-only corpus files are maintained. Current catalog, promotion, and financial-posture state is read from the live Offer Desk instrument.

| Topic prefix | Owner | Purpose |
|---|---|---|
| `opportunity-inbox/<signal-type>/<slug>` | any (operator, vision-walk, peer agents) | Untriaged opportunity intake. Signal types and dispatch live in `docs/monetization/taxonomies/monetization-opportunity/README.md`; the `opportunity-scout` member drains it (classifier: `signal-classifier`). The inbox view is the unrouted set. |
| `monetization/opportunity/<slug>` | opportunity-scout | The opportunity pool — SKU-shaped bets with required front-matter (`kind`, `catalog.proposed_sku`, `catalog.parent_bundle`, `revisit_trigger`, `acquisition_hypothesis`, `retention_hypothesis`, `capability_reuse`, `tam`, `effort`, `status`). Maintained by `opportunity-pool-hygiene`. |
| `validation-inbox/<request-type>/<slug>` | any (operator, vision-walk, opportunity-scout conversion, catalog-strategist, financial-tracker, staleness sweep) | Untriaged validation requests. Request types and dispatch live in `docs/monetization/taxonomies/monetization-validation/README.md`; the `market-validator` member drains it (classifier: `signal-classifier`). The inbox view is the unrouted set. |
| `monetization-benchmark-adjacent-record/<slug>` | marketing-crew | Benchmark-adjacent evidence awaiting monetization validation. |
| `monetization-benchmark-record/<slug>` | market-validator | Validated, dated, applicable benchmark evidence used by catalog and financial judgment. |
| `scout-scan/YYYY-MM-DD` | opportunity-scout | Scout heartbeat summary and evidence of the opportunity scan. |
| `topic[old]:vision-walk/alpha/<topic>` (legacy) | vision-walk fallback | Generic alpha when no typed topic fits. Prefer the typed `opportunity-inbox/*` or `validation-inbox/*` forms. |

Other judgment and operator guidance remain authored documents under this team's config tree; executable changes are routed through swarm-manager.
- Money Ledger `/adapters` — operator-supplied financial input surface. It is not team-owned canon; leave missing values absent rather than fabricating them.
- Source Ledger `team:monetization` — durable team context and handoff history.

## Inbox & Pool Invariants

- **Unrouted-set invariant.** No entry remains under any inbox/queue topic after triage. Every entry exits via the actions in its taxonomy's `actionSelection` set (drop / observe / promote-to-canon / file-work / capability work item). Generated heartbeat `# Inbox Flow` sections render the procedure from `topics.json` + the relevant taxonomy. Deviations are a process bug. Validation requests become benchmark records only after the validator resolves them; a stale or unavailable source remains explicitly unresolved.
- **Front-matter discipline.** Pool entries (`monetization/opportunity/*`), scout scans, adjacent benchmark evidence, and benchmark records must carry their source/date/applicability fields. Hygiene flags missing fields for repair; agents must not silently fix them.
- **No file hand-writes.** Agents use `source-ledger journal note` and `source-ledger recall`; the ledger owns concurrency, provenance, wake, and compaction.
