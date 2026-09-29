# BAS salvage reassessment — 2026-09-28 UTC

## Decision

**Still not accepted.** This reassessment updates the evidence for the
2026-09-27 salvage disposition; it does not freeze the candidate, create a
qualification lease, enable the delivery team, or authorize a BAS pilot.

## What the current source already resolved

The original review's first three behavioral concerns are no longer open in the
current worktree:

- `packages/api-core/server` now gives cleanup a fresh bounded context after a
  drain deadline and force-closes active standard-server requests. Its timeout,
  cleanup-order and request-cancellation regressions pass.
- The UI preserves a live navigation identity as `observation_unavailable`,
  keeps stop admission available, and prevents a successor start until the
  server operation settles. Recovery, invalid-status, session-fencing and
  successor-admission tests pass.
- The API redaction boundary covers Playwright and Claude callbacks, bounded
  history, completion/status responses, recording callbacks, URLs and human
  intervention text. Synthetic secret tests cover both navigators and the UI
  display path.

These are current source and focused-test facts, not managed-runtime proof.

## Current verification

- BAS vision and navigation-handler Go tests pass.
- UI record-mode navigation/conversation tests pass: 60/60, including the
  observation-loss → terminal recovery → successor-admission regression and
  31 `BAS-RH-J24` cases.
- Focused cancellation/recovery Go owners pass under `-race`.
- Focused profile-validation/cancellation-qualification race tests pass.
- The two selected real-Chromium typed-action cancellation/uncertain-effect
  tests pass.
- The shared protobuf closure passes `protogen verify`.

The repository-wide `git diff --check` remains non-clean only because of
unrelated pre-existing trailing whitespace in
`scenarios/audio-tools/ui/index.html`; the reviewed BAS/API-core paths are
clean.

## Remaining acceptance gaps

1. The bounded managed J07 restart owner now demonstrates one held navigation
   effect becoming uncertain, resources returning 1→0, replay denial, and
   driver/browser cleanup on candidate `sha256:1362c26b…`; the required
   outstanding status-wait cleanup cohort is still missing.
2. The current three-file ownership surface is 1,119 lines
   (`useAINavigation.ts` 567, `navigationEvents.ts` 376, and the WebSocket
   event owner 176) versus
   the 631-line baseline. Shared envelope validation, observation fencing and
   handoff admission are now consolidated, and the boundaries and focused
   tests are clearer, but this is not measured net simplification; moving code
   alone cannot justify salvage acceptance. A parser/lifecycle
   simplification or a stronger structural metric is still required before
   the broader E01 semantic exit.
3. The managed driver/browser and deployed Connect `AIService.AnalyzeElements`
   synthetic privacy owners passed at capture on build
   `sha256:a67b3dc43ff22b99205cf0afbb1a7c3e8671c2e02cec3e701c9c9c3a73a76622`,
   retained at `privacy-live-owner-20260928-current.json` and
   `privacy-api-owner-20260928-current.json`. Passive recorder/export,
   recovered-status, legacy at-rest privacy, desktop/mobile/native
   preservation, and the full J07 retained receipt remain unqualified. Later
   navigation source edits make these prior-build receipts stale for candidate
   qualification; preserve them as diagnostic evidence until the coordinated
   refresh at E01 closure.
4. No accepted candidate identity or one-use worker → independent Sol review →
   qualification receipt exists.

The next implementation work must therefore remain inside the single
navigation-observation/managed-shutdown boundary, simplify the lifecycle owner
with measured before/after evidence, and then obtain one bounded managed runtime
cohort before any acceptance decision.

## Post-reassessment diagnostic

The documented J07 owner cohort was rerun on unchanged build
`sha256:0c62b8cf731ca1a68e82561d7b72114bc2eb67df4f7b3cefa81ab97899c46d46`.
It observed one effect, released resources 1→0, retained uncertainty and
denied retry, but measured 1,676 ms restart preparation, 1,697 ms input stop,
18 ms cleanup and 9.45 s recovery. Only the input-stop band failed. This is
diagnostic evidence, not a receipt; same-build repeatability remains open and
the candidate is still not accepted.

### Post-reassessment local simplification — 2026-09-28

Abort and resume command admission now share one `isCurrentCommand` generation
fence instead of repeating the same navigation-ID/generation predicate. The
focused record-mode owner passes 60/60 and TypeScript checking passes. This is
a local ownership improvement, not net-complexity proof: the current three-file
navigation surface is 1,119 lines (567 lifecycle, 376 parser, 176 WebSocket)
versus the 631-line baseline. A subsequent edit also centralized navigation
envelope validation, observation fencing and handoff admission; the WebSocket
owner now dispatches through one type-selected parser and returns after the one
admitted event shape instead of reparsing the same message three more times.
The focused record-mode owner remains 60/60,
TypeScript and lint remain green. The parser/lifecycle surface is still net
positive, so simplification remains an acceptance gate.

## Cross-check of later adversarial notes — 2026-09-28 UTC

A later read-only review repeated several earlier concerns using stale line
references. The current source and focused tests were checked before carrying
those notes forward:

- `packages/api-core/server.shutdownAndCleanup` now runs owner cleanup under a
  fresh cleanup context even when the graceful drain deadline expires, then
  force-closes the HTTP server; the drain-deadline cleanup and forced-close
  request-cancellation tests pass.
- `useAINavigation` treats status transport loss as
  `observation_unavailable`, retains the navigation identity for stop/recovery,
  and gates successor admission; the current hook is 567 lines, not the 948
  lines cited by the stale note.
- AI navigation callbacks pass through `redactNavigationAction` before status,
  history, WebSocket and recording paths, and the UI display utility redacts
  sensitive values/results; focused Go and Vitest privacy tests pass.

These checks do not accept the candidate. Managed runtime receipts, current
qualification, and measured whole-surface simplification remain open. Do not
reopen the stale findings as new defects without a reproducer against the
current candidate build.
