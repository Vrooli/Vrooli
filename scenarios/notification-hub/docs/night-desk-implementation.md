# Decision desk — implementation record

What shipped from the Night Desk proposal, what is verified, and what is
deliberately left out. Written 24 November 2026.

## Shipped

### Contract

- `decision_family`, `decision_cause`, `decision_kind` persist on the ask and
  project on every read; the UI groups by family first.
- Additive readiness, relatedness and follow-through fields on `Ask` (proto
  66–72), with the tables in `api/internal/conversations/schema.sql` and the
  boot-safe `ensureAdditiveColumns` seam.

### Readiness

- `packages/api-core/decisionquality` — one pure model, one rule per owner. It
  does not repeat the Hub's cover checks; it merges the caller's findings with
  the operator-facing smells the Hub lacked (no recommendation, options with no
  effect, long or ID-bearing questions, jargon).
- The grade and report are stored with the ask, backfilled on every boot, and
  projected as `quality_grade` (list + detail), `quality_report` (detail) and
  `quality_summary` (list, one plain line).
- The steward projects `quality_grade`/`quality_summary`, prints a readiness
  line in `DECISIONS.md`, and raises a `decision-quality` risk with the
  `decision-desk` mitigation.
- `severityOf` in the UI understands the Hub's `blocking`/`advice` tokens;
  blocking findings are not shown as advisory.

### Relatedness

- Lexical scoring is deterministic and always available. It is scoped to the
  ask's recipient and to the reader's audience; it never crosses a conversation.
- Declared family and subject identity outrank similarity. A suggestion never
  merges anything.
- An optional semantic leg (`NOTIFICATION_HUB_SEMANTIC=1`) embeds decision text
  through `ai-gateway` (`embedding.default`), stores vectors in the Hub's own
  SQLite, and only raises a score when the model and dimension match. A down
  gateway leaves the lexical read intact.
- Verified against the live gateway: two paraphrases of the same decision
  cluster above 0.5 in one call (~0.33 s), and the fake-embedder tests cover
  paraphrase, fallback, dimension change and backfill.

### Follow-through

- `ask_followthrough` stores what the owner reported. `GetAsk` projects it; an
  answered decision with no report says exactly that. Progress is never inferred.
- `POST /api/v1/conversations/asks/{id}/followthrough` records it for the owner;
  state and timestamps are validated, and a report is refused before the answer.

### UI

- Night Signal tokens, bundled Inter, the crescent mark, the quiet rail and the
  current light/dark palettes.
- **Desktop decision room.** At 1200 px and wider, `/asks` is the agreed
  two-pane composition: a 22 rem queue column beside the decision being read.
  Choosing a row selects it in place (`?ask=<id>`) instead of navigating, so the
  operator moves through the backlog without losing the queue. The pane shows
  the chips, kicker, headline, situation, the requester's sandboxed HTML page
  when one is attached (its charts, prose and pre-fill forms), a declared-cause
  block that includes the current decision, the recommendation, the answer
  options, and a fixed answer bar (accept, send, answer later, open the full
  decision). The queue groups by
  attention with counts; search, sort and filters remain available in compact
  form. Below 1200 px the queue keeps the single-column composition, and a
  direct `/asks/:id` link still opens the full packet.
- The rail puts **By source** under the destinations and keeps the signed-in
  line at the foot, matching the proposal.
- Decision rows carry a readiness chip only when it needs attention; the full
  packet shows readiness findings (collapsed), related decisions, the
  follow-through report, and optional impact facts.
- `/focus` is the full focus review: one decision at a time with its
  **requester-supplied HTML page** (charts, prose and forms), its recommendation,
  the complete Hub answer form (questions, notes and attachments), and a context
  rail with source, age, close, readiness and related decisions. The page can
  pre-fill the Hub form; only the Hub sends the answer. The queue has a
  **Start review** action.
- i18n keys added in English, Japanese and Arabic.

## Verified

| Check | Result |
|---|---|
| `go test ./...` in `scenarios/notification-hub/api` | all packages pass |
| `pnpm exec vitest run src/features/asks src/features/packet src/layout` | 379 pass |
| UI suite (all files) | 88 pass, 1 **pre-existing** failure: the orphan `input.test.tsx`; 763 tests pass |
| `pnpm build` | succeeds |
| `pnpm exec tsc --noEmit` | one **pre-existing** error: `src/components/ui/input.test.tsx` imports a component that does not exist (unrelated; the app builds and the test suite does not load it) |
| steward targeted pytest | 65 pass, 1 pre-existing failure (`test_pace_research_and_model_breaches`); full suite 705 pass, 9 pre-existing failures |
| live gateway embedding | passes (`NOTIFICATION_HUB_SEMANTIC_TEST=1`) |
| rendered app | 18 captures in `docs/mockups/night-desk/impl/` at 1440/390/320, light/dark; fixture-backed, not a live owner session. The desktop captures now show the decision room. |
| live backfill | 30 open decisions `needs_work`, 5 `ready` |

## Adversarial rounds and fixes

Two review passes were run (one steward integration review, one backend/UI
self-review after the host restarted and lost the automated reviewers). The
findings that were real and fixed:

- `Related` could cross the recipient boundary: the SQL now requires the ask's
  own recipient before scoring (and still applies the reader's audience).
- Follow-through could be reported before the decision was answered: the service
  now requires `answered`/`defaulted`.
- The quality model re-checked the big picture, packet and recommendation
  reason that the Hub already reports: one rule, one owner; the duplicates are
  gone and the Hub's codes are preserved.
- The UI mapped `severity: "blocking"` to “advisory”: fixed.
- `FocusPage` called `flow.submit(draft)` where the hook takes no argument: fixed.
- The focus route had no entry point: the queue now has **Start review**.
- `quality_summary` repeated its grade in the steward line: the summary is the
  findings only.
- The boot backfill treated every named sender as an agent: it now uses the same
  `run:`/`owner:` rule as the send path.
- The embedding cache retained answered asks: closed asks are pruned on backfill.
- `Object.hasOwn` exceeded the project's TypeScript lib target: replaced with
  `Object.prototype.hasOwnProperty.call`.
- Status updates were being graded as decisions (no options, no recommendation →
  false `needs_work`): grading now skips `kind=status`.
- The semantic leg could stall a detail read for the gateway's full timeout and
  loaded vectors even with no embedder: it now runs only when an embedder is
  configured, under a 750 ms child context, and a failure is logged once a
  minute.
- A boot backfill could re-embed unchanged questions and keep closed vectors:
  the text hash is checked and closed asks are pruned.
- Jargon matched substrings (“mockup”, “stubborn”) and the id rule flagged model
  numbers (“F150”, “S-400”): jargon is word-bounded and the id rule is gone
  (the Hub/page check still owns it).
- The follow-through handler leaked raw errors and accepted non-JSON: it now
  maps 404/400/500, requires JSON, and defaults the reporter to the verified
  owner.
- `askVisibility` could emit an empty `AND ()` for an audience with only a
  subject: it now reports “sees nothing”.
- Subject-key reads now get the same related/follow-through decoration as
  id reads.
- The backfill now uses the handler's own refusable rule (`source == ""`), so a
  boot cannot silently regrade an ask.

Second UI round (reviewed against the rendered app):

- Light-theme primary and muted text were below AA (4.28/4.48:1): moved to
  `#0a6d81` / `#54687e`, and the rail's hover/active surfaces use the muted
  surface instead of white-on-white.
- A green “Ready” chip appeared on every assessed row: the chip now appears
  only for `needs_work` / `returned`, so the queue stays quiet.
- An empty follow-through `result_url` linked to the app root: guarded before
  `safeContextUrl`; “cost of waiting” now shows only while the ask is awaiting.
- The packet dropped a known grade when its report was missing: the grade stays
  and the panel says the report is not included.
- Focus review cached its batch forever and reported progress from the loaded
  page: `staleTime: 0`, `totalAwaiting` drives progress and the done message,
  and the phone tab bar is hidden on `/focus`.
- Focus review now moves focus to each new decision, announces progress, shows
  the new evidence, and offers “Check answer status” plus a packet link on a
  failed or changed read.
- Related links carry the current packet as their return origin; unknown basis
  tokens are labelled neutrally instead of “similar wording 0.00”.
- Lint errors in every touched UI file are gone.
- `asks.kind` labels missing from Japanese and Arabic were added; the locale
  parity test now passes.
- The desktop queue was still a centred single column after the first UI round.
  It now uses the approved two-pane decision room, with the rail's source list
  under the destinations.
- The focus review rendered only the list row's answer controls: it never
  fetched the full ask, so the requester's HTML page, its charts and forms, the
  readiness report and related decisions were all missing. Focus now reads the
  full detail, embeds the sandboxed explainer page, passes its pre-fill and
  answer-change messages into the same draft, and renders the complete answer
  form in a two-column room.

## Deliberately not shipped

- **Impact facts** (`unblocks`, `cost_of_waiting`) are on the contract and the
  UI renders them, but no producer flag/CLI writes them yet.
- **Readiness-aware count.** The operator count still includes unready decisions.
  Excluding them without a first-class “returned to producer” surface would hide
  work; the flag is intentionally not switched on.
- **Page backfill.** The writer needs the operator-issued Agent Manager grant.
  The pages are unchanged on disk and 12 of 16 open research pages still fail
  the Hub's own check; the audit is in `docs/decision-page-audit-2026-11-24.md`.
- **Steward `page_lint` removal.** It measures a different object (the steward's
  page file) from the Hub's ask record. Consolidation needs a shared page
  artifact first; the two are deliberately not merged.

## Enable semantic relatedness

```bash
# the scenario reads this at boot
export NOTIFICATION_HUB_SEMANTIC=1
vrooli scenario restart notification-hub
```

With the flag off, the desk is lexical only and needs no resource, which is the
scenario's stated posture.
