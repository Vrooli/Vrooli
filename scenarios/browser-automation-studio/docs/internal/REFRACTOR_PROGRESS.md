# Browser rehabilitation progress

This file is the active resume checkpoint, not an execution transcript. Follow
[TESTING.md](TESTING.md). Search only the referenced entries in
[PROBLEMS.md](../PROBLEMS.md). Detailed prior chronology is compressed and is
not routine resume input.

## Active candidate epoch E01 — integrate the stopped rehabilitation candidate

Status: implementation in progress; sensor readiness is mixed; current-source
candidate qualification is unknown. The previous worker was stopped on
2026-09-26. Preserve its product changes and do not commit.

### Outcome and boundary

Deliver one coherent, independently shippable BAS browser/capture execution
boundary from the accumulated stopped-agent work: capture and export results,
session/input semantics, frame delivery, retention and their honest managed
qualification must agree without parallel legacy paths. This is an integration
epoch over existing work, not permission to add unrelated features or refresh
scores piecemeal.

Affected product journeys include capture evidence, ad-hoc execution results,
recording/export retention, managed browser sessions, interactive input and
viewer frame delivery. Start with BAS-RF-143–146 and follow references only when
a changed path requires them. Shared invalidation roots are API/driver/UI source
identity, `.vrooli/service.json`, lifecycle freshness, managed browser state and
the six rehabilitation owner receipts.

Hypothesis: the stopped changes contain several useful repairs, but they were
developed and qualified as small work records. Reviewing them as one ownership
boundary will expose incomplete conversions and avoid repeated build/receipt
churn; after coherence and focused regressions, one frozen-candidate cycle can
truthfully qualify the result.

### Semantic exit gate

Keep E01 open until all of the following are true:

- every changed production path in the boundary is mapped to an intended
  behavior and BAS-RF/decision, with unrelated edits separated rather than
  silently absorbed;
- callers use one owner for each behavior; obsolete paths, shims and runtime
  migrations in the touched boundary are deleted;
- focused failing-before/fixed-after tests cover the repaired user-visible
  behavior and preservation obligations;
- the production and test-only line/complexity/duplication deltas are reported
  separately, with no claim that moving code reduced debt;
- no known actionable defect remains in this same ownership boundary;
- source is frozen, the managed candidate is rebuilt once, only invalidated
  owners are refreshed, the exact evidence phase runs once, the governed
  setpoint runs once and one adversarial review is recorded.

Permitted split conditions are only those in TESTING.md. Context compaction,
elapsed time, one green package or a score change does not close E01.

### Current truth

- **Implementation:** multiple focused repairs exist in the dirty worktree.
  W273's bounded two-frame UI paint queue passed 34/34 focused viewer tests and
  the prior worker reported targeted TypeScript/ESLint success. Capture-handler
  cohesion, session semantics, workflow results, retention, export and
  qualification infrastructure also changed and still require boundary-level
  reconciliation before this epoch can close.
- **Sensor readiness:** current owner implementations exist for evidence
  completeness, profile durability, cancellation/recovery, passive fidelity,
  resource budget and motion. Interactive-feedback still has the BAS-RF-144
  remote-owner/sensor gap. Treat this state as mixed until focused owner tests
  are reconfirmed after reconciliation.
- **Candidate qualification:** unknown for current source. The last reported
  managed candidate `sha256:7edf0d16…` scored 4/17, but later source and manifest
  edits make that historical evidence, not a current baseline. Do not refresh
  receipts or quote a new score until E01 freezes.

No lifecycle restart, build, broad Test Genie suite, evidence phase or setpoint
was run during the documentation/evidence-control pass.

### Resume exactly here

1. Inspect the existing BAS diff by ownership boundary; do not reset, discard or
   rewrite stopped-agent changes.
2. Build a short table in this checkpoint mapping each changed production path
   to intended behavior, focused checks, old paths/callers to remove and its
   E01 relevance. Move genuinely unrelated work to a documented successor
   epoch; do not qualify it independently during E01.
3. Begin with the capture/workflow/export/retention chain because it crosses the
   largest related API delta, then reconcile driver/session/frame behavior.
4. Run only focused discriminating tests while implementing. Update this same
   checkpoint before compaction with completed units, failures, pending
   operations and the exact next source action.
5. Use the boundary qualification sequence only after the semantic exit gate is
   satisfied.

## Evidence and documentation maintenance — 2026-09-26

The runtime evidence root measured 610 files, about 38 MiB and 977,008 text
lines before cleanup. Superseded top-level attempts and the 304-file raw capture
cohort were moved to two recoverable archives. Archive listing and SHA-256 were
verified before originals were removed. The directly readable working set is
now 31 files, 5,608,203 bytes and 134,066 text lines, within TESTING.md's
128-file/16-MiB/250,000-line budget.

Manifest:
`.vrooli/runtime/rehabilitation-evidence/archive/epoch-protocol-cleanup-2026-09-26.manifest.json`.
The archive SHA-256 values are `ec093bd91efb78df90ed9c23260d060d8277b79bcbd5a9eb0bbd80b7d063996f`
and `979ea0d6dac51e85b9f24af3e07fbc0d6d770e8741530449de0964d58ff693bb`.
This cleanup changes evidence storage only; it does not promote or invalidate a
product result.

The complete pre-compaction feedback ledger is preserved as
[`operator-feedback-history-2026-09-26.md.gz`](evidence/rehabilitation/operator-feedback-history-2026-09-26.md.gz)
(SHA-256 `ae92b5350d625ee6ae3a0b576ad3354a7bfa1e87ec9b3435ea5298d3ea97e0db`).
The previous active progress is preserved as
[`refactor-progress-snapshot-2026-09-26.md.gz`](evidence/rehabilitation/refactor-progress-snapshot-2026-09-26.md.gz)
(SHA-256 `c70ec68f474376aa4ab60b1ea641e2ae084ced9191a3d55df136406ea5c1358c`).
Earlier W228–W266 history is preserved as
[`refactor-progress-history-2026-09-25.md.gz`](evidence/rehabilitation/refactor-progress-history-2026-09-25.md.gz)
(SHA-256 `35f2231f66373a3881b2a22096924f5258e5b26a04fb86d5f43a5fd1662b51b6`).
Retrieve any of these with `gzip -cd <path>` only when a current decision needs
the historical detail.

## Recent epoch summary

Before the protocol change, W267–W277 repeatedly refreshed build-bound evidence
around focused repairs. The useful durable results were: lifecycle freshness is
now read by the setpoint; duplicate closure-cache work and the root API response
deadline were repaired; capture construction was decomposed; the viewer's
delayed-paint frame loss gained a deterministic regression and bounded queue;
owner cohorts reached clean observations on prior candidates; and shared browser
sessions were not closed to force qualification. The repeated candidate resets,
score reads and receipt refreshes are historical evidence of the inefficient
cadence that E01 must not repeat.
