# Browser rehabilitation progress

This is the single active resume checkpoint, not an execution transcript. Read
[TESTING.md](TESTING.md), run the campaign guard, and search only the referenced
entries in [PROBLEMS.md](../PROBLEMS.md). Do not read the histories during
routine resume.

Full chronology through the stopped 2026-09-26 run is preserved at
[`evidence/rehabilitation/refactor-progress-history-2026-09-26-agent-run.md.gz`](evidence/rehabilitation/refactor-progress-history-2026-09-26-agent-run.md.gz),
SHA-256 `b18da33354046fb9a36205810e3d8b7ae2fe3f8fa4128e49e1905df9c45cf74b`.
Retrieve only when a present decision depends on it with `gzip -cd <path>`.

## Active candidate epoch E04 — lifecycle and passive-recording ownership convergence

Candidate state: **implementation**

Qualification cycle: **unused**

### Outcome and independently shippable boundary

Make the browser/session lifecycle one coherent ownership boundary from
profile-backed session admission through recording, input, navigation,
replacement and teardown. A user can reuse or reset a persistent browser,
record actions passively, interrupt work and recover without duplicated
effects, stale frames/callbacks, leaked input state, lost acknowledged events or
competing cleanup paths. Finish the replacement: remove obsolete callers,
fixtures and compatibility paths, and reduce test/production complexity rather
than moving it.

This epoch includes the related session manager/reset/teardown, recording
pipeline and event routes, input/gesture/keyboard recovery, AI-navigation
cancellation, frame/WebSocket replacement, profile durability,
cancellation/recovery and passive-fidelity ownership already changed in the
dirty worktree. It does not split each defect, test file, receipt or setpoint
row into another epoch.

Affected contract surfaces: persistent sessions, passive recording fidelity,
interactive browser feedback, cancellation/recovery, artifact evidence and
the associated preservation journeys. Active issues: BAS-FB-020–033 cadence,
BAS-FB-036 test quality and BAS-RF entries found while completing this boundary.

### Falsifiable hypothesis and baseline

Hypothesis: the accumulated lifecycle repairs can be reduced to one explicit
owner per transition, with focused tests proving ordering, cancellation,
cleanup and recovery and with less duplication than the pre-epoch tree. It is
false if any transition has two mutation/cleanup owners, acknowledged input or
recording can be lost/replayed, a replaced page/session can still publish, a
focused test process does not terminate, or measured source/test complexity is
not improved.

Baseline from the stopped run: 42 purported epoch summaries, 17 managed
restarts, 29 exact evidence phase identities, 29 setpoint identities, repeated
stale-receipt `0/17` presentations, and one Jest process hung for about six
hours. The last comparable frozen candidate reached 8/17 readable; current
qualification is unknown, not a product regression. The campaign guard now
enforces one active epoch, a 96 KiB packet, a 16 KiB progress file and bounded
live evidence before more work.

### Semantic exit gate

Remain in `implementation` across compactions until all of the following are
true:

- the complete dirty lifecycle/passive-recording diff is understood and every
  changed owner has focused expected-behavior coverage;
- obsolete callers, parallel cleanup paths, compatibility shims and duplicate
  fixtures in this boundary are removed or explicitly proven necessary;
- session/profile/input/recording/navigation/frame teardown and replacement
  pass focused ordering, fault, cancellation and recovery checks;
- every invoked focused test exits and leaves no owned child process; the
  stopped run's orphaned Jest is either explained and prevented or recorded as
  a reproducible defect with an owned repair;
- production and test debt deltas are measured against the pre-epoch diff and
  demonstrate real simplification, not helper proliferation;
- `campaign_guard.py --stage resume` passes and no implementation work remains
  behind a qualification-only task.

Only then change the state to `frozen`. Before generating receipts run the
producer guard, create only the invalidated current-candidate cohorts, and run
one closure cycle: focused regressions, at most one needed managed
rebuild/restart, one exact rehabilitation-evidence phase, one setpoint read and
one adversarial review. A product failure reopens this same epoch; a stale
receipt does not erase implementation evidence or create another epoch.

### Completed work units retained from the stopped run

- Repaired session reuse artifact ownership, active-tab reset semantics and
  external-target handling; closed stale bitmap publication in the UI frame
  decoder.
- Added immediate HTTP fallback when a WebSocket closes during send admission;
  local and emulated-remote cohorts reported 1,000/1,000 correlated actions.
- Repaired video flush, live-input drain, modifier/pointer recovery,
  AI-navigation cancellation, page callbacks, frame replacement and
  session/pipeline teardown across the affected owners.
- Consolidated receipt identity/hash logic and fixed passive-fidelity selection
  so a lexically newer stale receipt cannot hide a source-valid receipt.
- Focused profile durability, passive fidelity and evidence-completeness owners
  passed individually on the prior managed candidate. Their later aggregate
  failures named other stale receipts; they did not disprove those behaviors.
- SessionManager owner cleanup reported a 79-line reduction and zero lint
  findings. Broader net debt reduction remains unproven.

These statements are inherited evidence, not permission to skip inspection or
rerun qualification. Preserve the user's uncommitted changes; the agent never
commits.

### Exact next actions

1. Run `python3 scenarios/browser-automation-studio/docs/internal/campaign_guard.py --stage resume`.
2. Inspect the current dirty diff by lifecycle owner and reconcile it with the
   focused tests already changed. Do not start with Test Genie, a restart,
   receipt generation or the setpoint.
3. Run the smallest focused suites for related completed work units, with a
   bounded timeout and post-run child-process check. Diagnose any non-exiting
   Jest owner before expanding validation.
4. Continue the same ownership boundary: finish callers/removals, repair actual
   failures, simplify fixtures and measure production/test debt deltas.
5. Freeze and qualify only when the semantic exit gate—not context length,
   elapsed time or a green individual test—is satisfied.

### Current evidence and interruption state

The live runtime evidence root was intentionally emptied before resume. Its 387
files, 12,574,234 bytes and 285,793 text lines were recoverably archived in
`.vrooli/runtime/rehabilitation-evidence/archive/superseded-working-set-before-campaign-gate-2026-09-26.tar.gz`
with manifest and verified SHA-256
`64dc4e20a79cd3df5f8619e957cf2703cb02717a4d4bd70c53de6aefdb4265c1`.
Do not restore it wholesale. It represents mixed candidates and failed/stale
attempts; retrieve a named artifact only for diagnosis.

The six-hour orphaned Jest child from the stopped agent was terminated by PID
after its exact command and ownership were verified. BAS's managed API was not
stopped. No product qualification was performed during campaign cleanup.

Changed: campaign state was compacted, runtime evidence was archived, and a
mechanical guard was added. Verified: archive listing/hash and stopped child
termination. Remaining: complete this epoch's implementation and focused
checks. Unverified: current candidate qualification, native/device/desktop
behavior and full complexity reduction.

## Epoch summary — prior stable owner-boundary candidate — 2026-09-26

The last stable candidate before later source edits reached 8/17 readable rows:
seven owner-backed capabilities plus capture. Interactive transport continuity,
owner receipt identity and several lifecycle defects were materially repaired.
That score is historical comparison evidence only; later source edits
invalidated candidate-bound receipts without proving product regressions.

## Epoch summary — campaign-control reset — 2026-09-26

The stopped run delivered useful code but violated epoch, active-packet,
evidence-retention and process-hygiene rules. Its narrative and mixed runtime
cohorts were archived recoverably. Resume under the executable guard and this
single large epoch; do not reconstruct the 42 micro-epochs in active state.
