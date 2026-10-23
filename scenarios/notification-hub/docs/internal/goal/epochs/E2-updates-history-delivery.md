# E2 — Make Updates, History and Delivery evidence truthful

- Outcome: owners can inspect informational notifications, bounded decision history and available delivery evidence without inferred read, reach, attempt-history or downstream-work claims.
- Kind: feature
- Started: 2026-10-20T15:54:00Z
- Estimate: 16 work units, revised above E1's 10-unit estimate after E1 consumed 22 units; reuse current source and measure before editing.
- Expected size: 900 runtime lines (feature ceiling 1.5x); justify additions and remove superseded paths.
- Metric: E2's applicable Updates/History/Delivery behavior has exact source-to-screen/state/test mapping and focused self-validation; no benchmark gate is independently accepted by this epoch.
- Targets: current governed managed lifecycle and authorized isolated fixtures only; preserve consumer/data behavior; no personal-data writes, real notifications, fake human answers, unapproved schema changes or unrelated effects.
- Workers: task `49013184-e31d-40c3-bfd7-86a5282e3948`, run `1dce87fd-9478-4d00-948a-210772070fbc` admitted at `2026-10-20T15:54:08Z` with verified active effort reference; pending at admission. Parent orchestrator run `503e3320-4190-4335-85e2-f28318a620dd`; existing `prompt-manager/delivery-epoch-worker` profile `4c19aef1-36a2-4cc5-8961-42648692883b`, Luna medium interactive goal run.
- E1 step-back: E1 remains parked and unaccepted at 0/10 gates after 22 work units versus estimate 10. E1 log names exact UI-health and isolated-workflow return conditions. Do not duplicate those gates or weaken them here.

## Modules and boundaries

- Updates: informational notification content whether or not a decision exists; title/source/time/urgency and sensitivity only as authorized. No unread dot, mark-read or generic acknowledgement. Show Open decision only for an actual relationship. Empty means none in returned scope, not all producers healthy.
- History: decision history, not notification-read archive. Show returned answered/defaulted/expired records, keep answerable defaults discoverable, respect current bounded result limits, and never imply completeness or total count the API does not prove. Opening a row does not mutate it; expired items do not gain a reopen action. Reconcile existing routes before writing.
- Delivery overview/detail: separate notification state from channel attempts and provider receipts. Preserve Pending, Held, Routed, Delivered, Failed, Unroutable and Suppressed with authoritative safe reasons. Generic Delivered means an adapter accepted the payload; a provider-specific acceptance claim requires its receipt. No receipt means reach is unknown. Current state is not transition history; never invent attempt history, human reading or downstream progress. Hide credentials, raw endpoints and secret-bearing URLs.
- No user-triggered Retry delivery action unless a current owner contract explicitly supports it. Channel edits never resend a failed notification; disabling a channel does not retract accepted payloads or cancel decisions.
- Keep analytics outside this slice unless already commissioned in the current queue; do not invent denominators or extend scope. Keep five destinations, R24 ordinary recipient-scoped reads without login/grant, human-only answer authority, and E3 NH-DEC-26/A critical-only implementation intact.

## Required source and coverage references

- Complete contract: `docs/internal/delivery-reliability-research/follow-through/BENCHMARK.md`.
- At admission read the linked design sources and current scenario `PRD.md`, requirements, DESIGN, routes/API/storage, team owner state, plus local `docs/internal/delivery-operations/2026-10-12/sources/notification-hub-dossier.md` and `notification-hub-atlas.md`; selected manifest `notification-hub-revision/selected-image-manifest.json`; originals under `docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/`.
- Apply dossier sections “History and informational updates”, “Delivery as evidence”, “History and delivery cases”, “Bounded history and defaults”, “Truthful delivery”, and “Scoped diagnostic detail”; preserve later owner authority and existing R24.
- Explicitly reconcile selected manifest denominator (240 items, 102 compositions, 71 state IDs) with the complete redesign mapping; retain exact item membership and applicable/queued/adaptation rationale. Do not claim E2 accounts for whole redesign or accept later scope.
- Primary cases: NH-AC-26 (bounded results; >limit, equal timestamps, loaded wording, partial next-page behavior only if an authorized page API exists); NH-AC-27 (receipt vs read, all transport states, absent receipt, unknown enum and no fabricated attempts); NH-AC-36 (recipient/privacy boundary, safe logs/screenshots); applicable NH-AC-37 (keyboard/responsive/appearance on E2 routes). Preserve applicable NH-AC-20/25/35/40 relationships without changing E1/E3 behavior.

## Deletion and consolidation intent

Inspect existing notifications/history/delivery owners, repeated status mapping, duplicate list/detail state and obsolete unread/acknowledgement/retry affordances. Remove or consolidate superseded implementation and callers only where unique behavior and tests remain protected. If none is safe, name candidates inspected and why; findings alone are insufficient where duplication exists. Preserve existing integration consumers and unique regression protection.

## Exit gate

- Record current checkout/source/build identity, lifecycle, exact write set and shared dirty-tree attribution. Before first product write, capture immutable exact-path prewrite snapshot and explicit absences; never overwrite existing E1 checkpoints.
- Map each changed screen/state/API field to current authoritative source and a desired-behavior regression. Distinguish missing fields/contracts from UI defects; keep unsupported projections unavailable and do not widen authorization or invent schema fields.
- Validate bounded list semantics, actual status/receipt rendering and safe empty/loading/partial/stale/failed states with focused checks. Run only scoped current-tree Test Genie phases selected from `docs/TESTING.md` and current scenario catalog after substantive changes; use server-owned wait once, preserve artifacts/build identity, and do not rerun unchanged failures.
- Any browser-backed check must prove active authorized lease, test-pool request and zero primary-during-test writes before observing recipient data; no personal recipient data, real send, fake answer or manual devrouting. If no supported safe fixture exists, state exact owner and return condition, leave affected gates open, and continue any disjoint safe work.
- Verify managed runtime only through the governed lifecycle. Record selected desktop/phone dark/light visual evidence where safe; implementation captures are not independent acceptance.
- Include changed/verified/remaining/unverified, traceability, exact commands/results/skips/receipts, additions/deletions/callers, debt, source comparison and attribution limits. Append evidence here, then run `agent-manager effort epoch-check <this file> --runs <worker run> --wake-key 503e3320-4190-4335-85e2-f28318a620dd`.
- Never self-accept. Independent code/architecture and visual review remain separate; this handoff is not E2 acceptance or whole-redesign closure.

## Non-goals

No analytics expansion, pagination invention, credential or integration editing, retry delivery, new read/answer authority, policy changes, real sends, test personal data, destructive migration or edits to E1's files. E1/E3/whole-app acceptance obligations stay open.

## Directives

- OWNER-QUALITY-001: consolidate duplicate state/status mechanisms and remove obsolete paths/callers where behavior protection remains; show concrete deletion and independent review.
- STEP-BACK-E1-20261020: E1's 22-unit overrun resulted in park/switch. Keep its safe workflow and UI-health return conditions open; do not repeat unsafe runtime checks.
- BENCHMARK: original clock `2026-10-20T03:48:04Z` to `2026-10-21T03:48:04Z`; no reset. Record waits, intervention and attribution honestly.

## Directives acknowledged

(worker records IDs in its first applicable slice-log entry)

## Worker evidence paths

- `../evidence/E2-attempt1/` — task-local immutable prewrite snapshot, exact allowed-path source comparison, and validation/build receipts for run `1dce87fd-9478-4d00-948a-210772070fbc`.

## Slice log

(worker appends timestamped work-unit lines)

2026-10-20T16:10:26Z | Implemented Updates, bounded History, delivery overview/detail and canonical seven-state mapping with scoped regressions; retained exact-path byte comparison, 240-item design reconciliation and Test Genie producer artifacts. Focused Vitest 5 files/19 tests, type-check, strings/selector freshness and temp build pass. Test Genie unit run `20261020-160522-476778ec` fails UI functions coverage 84.61% < 85%; scoped ESLint has 40 fixture-query errors. Exact-current-source managed identity, browser fixture/visual proof, independent review and acceptance remain open. First work slice acknowledges all applicable directives. | exit metric=0/10 benchmark gates independently accepted; E2 implementation handed off, required validation/visual/independent gates open | net runtime lines=+490 | net test lines=+184 | ack=OWNER-QUALITY-001,STEP-BACK-E1-20261020,BENCHMARK
2026-10-20T16:12:00Z | Corrected the allowed-path comparison to include the separately snapshotted HealthCard path; exact union is 20 product/generated paths. Final SequenceMatcher totals (not Git): runtime +628/-143 (net +485), tests +214/-30 (net +184). Evidence: `evidence/E2-attempt1/source-comparison.json`; original/checkpoint manifests preserved. | exit metric=0/10 benchmark gates independently accepted; E2 implementation handed off, required validation/visual/independent gates open | net runtime lines=0 | net test lines=0 | ack=OWNER-QUALITY-001,STEP-BACK-E1-20261020,BENCHMARK
2026-10-20T16:12:01Z | Measurement correction only: HealthCard checkpoint contributes +1/-6 lines previously absent from the first accounting, correcting cumulative runtime net by -5; no product file changed in this slice. | exit metric=0/10 benchmark gates independently accepted; E2 implementation handed off, required validation/visual/independent gates open | net runtime lines=-5 | net test lines=0 | ack=OWNER-QUALITY-001,STEP-BACK-E1-20261020,BENCHMARK
2026-10-20T16:13:00Z | Replaced fixture-text Testing Library queries with stable page-selector assertions; scoped ESLint is now 0 errors/1 warning, focused Vitest remains 5 files/19 passing tests, and type-check passes. Reconciled managed status to healthy current-source operation `startop-e42e5562edd21f0a08bc8a3b6cb450f5`; deployed dist asset hashes match the temp build except the lifecycle freshness manifest. Coverage Test Genie failure and browser/visual/independent gates remain open; no coverage retry. | exit metric=0/10 benchmark gates independently accepted; E2 implementation handed off, required validation/visual/independent gates open | net runtime lines=0 | net test lines=+5 | ack=OWNER-QUALITY-001,STEP-BACK-E1-20261020,BENCHMARK
