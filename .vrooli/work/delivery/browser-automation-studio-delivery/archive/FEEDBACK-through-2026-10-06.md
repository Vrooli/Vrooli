# Browser rehabilitation operator feedback

The orchestrator owns this file: it records new operator feedback verbatim at the
top of Open directives, then translates it into epoch Directives or brief
amendments. Workers never read it directly. When an item is resolved, move it to
the resolved table with a one-line disposition.

Verbatim history through 2026-09-29 (BAS-FB-001–055) is archived in
`~/.vrooli/plan-artifacts/epoch-based-delivery-orchestration-and-bas-rehabilitation/evidence/archive/bas-rehabilitation/retired-docs-2026-09-29.tar.gz`
(member `docs/internal/goal/FEEDBACK.md`) and
`~/.vrooli/plan-artifacts/epoch-based-delivery-orchestration-and-bas-rehabilitation/evidence/archive/bas-rehabilitation/loose-evidence-2026-09-29.tar.gz`
(`operator-feedback-history-*.md.gz`). Read them only when a decision depends on
exact wording.

## Open directives

### BAS-SUP-AUDIT-002 — E22 mechanical acceptance mismatch (2026-10-05, supervisor-audit)

> Supervisor audit of the 2026-10-05T16:25Z step-back: `agent-manager effort epoch-check scenarios/browser-automation-studio/docs/internal/goal/epochs/E22.md --runs 62b11517-9fd8-4f6a-941a-c7bd1b1a2cdb --json` reports `open_directives:["D2"]` and rejects the 2026-10-03T03:03Z parent-review slice-log line as malformed. It parses 16/18 work units, +379/750 runtime lines and the historical `ACCEPTED` line, with no overrun/growth step-back; wall-clock is review-only. The final worker line acknowledges D1 only. The parent text "D2 records" and `ack=D1; parent reviewed` do not satisfy the worker acknowledgment gate. Treat E22 acceptance as failed under the recorded gate. Preserve the historical line and evidence; use the supported owner reconciliation to reopen E22, secure a genuine D2 acknowledgment from an authorized successor if required, correct the malformed parent review in an append-only way, then independently rerun the original remaining exit gates before any renewed acceptance. Keep R1 current-build/player proof and R2 generated-contract decision open. Do not fabricate a worker acknowledgment or weaken the checker.

Status: open; receiving owner is BAS orchestrator run `70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb`. Return with the epoch-check receipt, revised acceptance standing and exact remaining gates. This entry is supervisor-audit steering; the orchestrator owns epoch and queue changes.

Orchestrator response (2026-10-06): reopened E22 and admitted authorized Luna-medium successor `98268861-3a7f-4b04-ba7a-e66b714fe4c0` under task `0622e0dc-d779-4ec5-b993-9a98ca04d200`; its slice log genuinely acknowledges D2/D3. Appended a parser-valid parent review correction while preserving the malformed historical line and original ACCEPTED line. Parent independently reran the daily 12/12 Linux qualification on source-attributed shadow build `sha256:e16bbda1f5ed9d79d12ba591912b24b08962bc4c4fca8512238948b7fa304cbd` and measured 254,178 runtime lines (+379 vs admission, within +750). E22 was renewed as accepted with R1/R2 open: the intended-player comparison awaits a persisted workflow/project association for capture `8ed37203-2ff7-49d7-9e1f-f953e1f0edc2` or a QA-supported route; the generated movie-spec timestamp/edit-map decision awaits its owner.

### BAS-FB-062 — slices are not held for the destination gap (2026-10-02, operator)

> Yes, go ahead with all three. (Approving: rewrite FB-061 plainly; measured
> slices may run now; bring me live-feature retirement candidates.)

Status: open. For the orchestrator:
- BAS-SUP-001's gap does not gate individual slices. Admit any slice whose exact
  deletion list and estimate are measured; keep the 205k gap visible in the
  Forecast as a planning item.
- Order: (1) DIAG-REDACT (FB-061), (2) REC-1A, (3) REC-FIX (FB-061), (4) WF-1
  after its census, then the remaining ownership slices as they are measured.
- Write a short `## Retirement candidates` list at the top of `QUEUE.md`: live
  features whose removal would close the gap, each with runtime lines, the
  user-visible loss and its callers. Wake the supervisor once when it exists;
  the operator chooses from it. Do not lower the 205k destination meanwhile.

### BAS-FB-061 — capture/review reliability work (2026-10-02, operator)

> The operator approved both briefs below, diagnostics first ("I1 A"), through
> the existing BAS team and its normal orchestrator/supervisor workflow.

Status: open; supersedes the earlier long form of this entry (kept in
`~/Documents/Codex/2026-10-02/task-9/`). Two queue items, run in order:

- **DIAG-REDACT** — brief 3, BAS-OBSERVABILITY-REDACTION-01:
  https://docs.google.com/document/d/1EqXUqhJEPJ5DrhEEMWKXcGvcTVbir1lJOWoLI5ckH_k/edit?tab=t.zabm70ptmjv7
  (redaction section). Redact the recovery credential at the producer in both
  `all_options.internal` and `modified_options`. Synthetic-only regressions
  inspect full serialized standard and deep responses, cache-hit branches and
  the API proxy/transformation, for set, unset and modified cases, keeping
  ordinary metadata and the set/unset indication. ≤50 net runtime lines; no
  generated-code edits, live secrets, dependency, security or access changes.
  Report version-skew and deployment limits rather than claiming universal
  coverage. A small epoch by operator choice; the ~3 h minimum does not apply.
- **REC-FIX** — brief 5, RECORDING-FIX-01:
  https://docs.google.com/document/d/1P8U8weXPXJTY788NnOYKj2Dx0nthSYHJazo0z3QJ07M/edit
  Starts after DIAG-REDACT is accepted. Outcome: stable native video geometry,
  truthful screenshot extent, observed/derived/missing pointer provenance,
  explicit source choice, useful intended-surface playback, originals and
  failures preserved. Feature budget: expected ≤500 net runtime lines, hard
  maximum the lesser of 750 or 1.5× the admitted estimate. Allowed: the brief's
  three narrow metadata corrections and restarting the `bas-goal` shadow; not
  the live BAS or other services. Read the full acceptance in the brief.

The requester reads progress from `epochs/` and `QUEUE.md`; nothing is pushed
to Drive and no outside release is needed.

### BAS-SUP-001 — forecast gap after E18 (2026-10-01, supervisor)

Status: no longer a hold on individual slices (BAS-FB-062); the gap stays a Forecast item.

The E18 Forecast measures 258,853 runtime lines against the 205,000 destination. Its remaining queued slices forecast about 22,752 lines at the recent actual-to-estimate ratio, leaving about 31,101 lines unaccounted for. Before admitting a later ownership slice beyond E19, measure the exact owner/file scopes behind this gap and add deletion-first slices whose forecast can close it. Keep E19's accepted brief and exit gates intact. This is a planning finding, not an epoch acceptance decision.

The due E17 sample confirms the 11 retired converter/type names have no API/UI source hits in the current tree. Its full exit-gate replay remains pending because E19 is actively changing the shared tree; compare against an isolated E17 revision or an unmodified E17 gate receipt before calling the sample complete. E17's recorded unit-delta sum also differs from direct inventory by 2,408 runtime and 393 support lines; preserve direct inventory as the measured outcome and reconcile the log accounting when the sample is completed.

### BAS-FB-060 — plan to the destination, not to the end of the queue (2026-10-01, operator)

> Go ahead with all that. (Approving: the orchestrator owns slice planning and
> raises estimate misses; park as a backstop, not a poll; the epoch write grant.)

Status: resolved 2026-10-01: Forecast and measured planning slices are recorded at
the top of `QUEUE.md`; E18 was admitted and dispatched to the Luna-medium worker.
The path-to-budget mapping gaps remain explicit in the Forecast and individual
slice admission conditions.

For the orchestrator:
- Your member contract now lists the goal home (including `epochs/`) as a write
  surface; the earlier omission was a contract defect, not a limit. Admit E18.
- `large-effort-orchestration` revision 22 changed: park with `--timeout 1h`
  (§2 step 4, revision 23): Agent Manager now compacts a session parked for
  20 minutes, so wakes are cheap but each costs a re-orientation; the timer is
  only a hung-worker backstop; keep a `## Forecast` at the top of `QUEUE.md` and plan new slices
  when the queue cannot close the gap (§2.1); close only when the destination is
  met (§6). Reread §2–§2.1 now.
- Write the first Forecast now. Runtime is 259,186 against the ~205k destination
  (gap ≈ 54k). S1 and S5–S10 each came in well under their estimates (E10 −671
  vs −3.5k, E12 −299 vs −1.5k, E14 −8); that is a planning finding. S12 and S13
  (≈ −4.5k) cannot close the gap.
- Before or alongside E18, measure each module against the QUEUE budgets and
  queue deletion-first slices for the largest gaps (Workflow, Recording,
  Workspace UI, Export, AI, Diagnostics), each with a deletion list and a measured
  estimate. Then wake the supervisor once with the planning finding recorded.

### BAS-FB-059 — widen the journey net before S1 (2026-09-30, operator)

> Before S1 (the session broker), add an epoch that widens the journey suite against the local fixture site: J03 (same selector across tabs/frames), J04 (redirect, SPA, popup, shadow DOM, service worker), J13 (shortcuts, drag, scroll), J15 (session capacity and reuse), J17 (loops and retries) and J18 (cancel failure evidence). S1 and S5 rewrite the session and recording core, so they need this net first.

Status: resolved 2026-09-30. E9 added the six requested journeys and the complete
shadow suite passes 11/11; the queue now places S1 after E9. E8 and E9 acceptance
records include each epoch's currently reported weighted non-cache tokens and
explicitly preserve Agent Manager's `usage not final` receipt status.

### BAS-FB-058 — S3 targets every existing V1 workflow (2026-09-30, operator)

> S3's intent is every V1 workflow that exists, not a count of 14. The scan result is authoritative: migrate every V1 workflow found (done), leave the Test Genie HTTP cases alone because they aren't workflows, and accept E6 once the V1 code path is deleted and its gates pass. Migrating the ~700 short-form playbooks stays a separate later slice.

Status: resolved 2026-09-30. E6's S3 gate now follows the scan: all six existing
V1 workflows migrated, BAS has zero V1 arrays, and GCT HTTP cases stay unchanged;
E6 accepted with BAS workflow-health recorded unverified after its queued-provider
timeout.

### BAS-FB-057 — S2 is not done (2026-09-30, operator)

> S2 is not done: the hand-written action vocabulary files (`actionBuilder.ts`, `actionParams.ts`, `nodeUtils.ts`, `execution_params.go`, `driver_convert.go`, `typeconv`, driver `params.ts` and `action-type-utils.ts`) still exist. Requeue the remainder at the top of the queue with a deletion list naming each file's end state, and hold future slices to their full queue scope.

Status: resolved 2026-09-30. S2 remainder was requeued with eight named paths and
accepted as E7 only after all eight were verified absent and its full gates passed.

### BAS-FB-036 — test-code quality

Professional test infrastructure with less duplication, drift and code volume.
Treat test debt inside an ownership-boundary epoch; no score-neutral helper
extraction.

### BAS-FB-020–038 (cadence group)

Produce meaningful product progress faster, use focused checks, stop repeating
broad evidence cycles, and make qualification serve delivery.

## Resolved

| IDs | Disposition |
| --- | --- |
| BAS-FB-056 | Gremlins recorded unverified under operator direction; E2 accepted on passing Go/coverage gates, further test maturation continued as E4. |
| BAS-FB-055 | Translated into E0 directive D3 (two more worker runs, journey-ID slice-log prefixes); E0 accepted 2026-09-29. |
| BAS-FB-045/047/050/051 | Resolved by the epoch-orchestration plan: Luna-medium workers, infrequent Sol supervision, one orchestrator per goal, old supervision and qualification machinery retired (C1, C2). |
| BAS-FB-039–044, 046, 048, 049, 052–054 | Resolved before 2026-09-29; see the archive. |

### BAS-RECOVERY-20261004 — approved build repair; consumer recovery still open

Owner authority: Sentinel_2c11fb2cac308191821616e3c8713ed7, “Yes, do all of that properly please,” responding to Sentinel_7a959ef8396481919a7284e5b7a48799. Coordinated recovery uses the existing team, native profile, budget and safeguards. This does not authorize retiring Record/Export or broaden any product decision.

The shared protobuf owner verifier (`cd packages/proto && make verify-committed-gen`) passed without drift. The BAS UI build failure was reproduced as Vite mapping the valid public `@bufbuild/protobuf/wire` import into `dist/esm/index.js/wire` (ENOTDIR). One explicit public wire-export alias was added beside the existing package aliases in ui/vite.config.ts. The same configured production build then passed (4148 modules); generated output and dependencies were not edited. Before/after bytes and hashes are in the task-14 coordinated-recovery receipt directory.

Source-applied/build-verified is not consumer acceptance. Fresh control-plane status still reports the live instance unhealthy (API_PORT unresolved, UI HTTP 503); shadow status retains the Oct3 failed replacement build and no tracked processes. The original owning team must reconcile actual active executions, preserve the qualified quiet window, use the supported scenario lifecycle to retry the original shadow operation, and return its health/build identity plus the blocked consumer's own receipt. Do not infer completion from the old deployed build or launch another leader. Source repair independent review, runtime retry, receiver acknowledgement and E22-R1 acceptance remain OPEN. E22-R2 generated movie-spec ownership and the runtime budget gap remain explicit; deleting features still requires the actual owner choice.

Shared observability/provenance changes were prepared and applied to the existing canonical supervision skill, worker reference and await banner. Focused await UI regressions (3) and type-check pass; native Agent Manager unit run 20261004-015342-4cba1984 fails with two COMPANION_REIMPLEMENTED findings and two TEST_EXECUTION_FAILURE findings (API go test and UI test:coverage) and is not a green certification. Independent native review/admission requires existing caller proof, which this execution context does not have; no anonymous dispatch, credential exchange or new grant was attempted. Preserve this handoff open until the receiving owner acknowledges it and the original operation passes.


#### Recovery validation follow-up — 2026-10-04

Complete failed-gate attribution found one defect caused by the await wording repair: the existing RunDetailParts component suite asserted the old “Parked — waiting, not hung” heading. That one assertion was corrected; the existing component suite plus new colocated await regressions now pass all 10 tests. Three contemporary RunsPage investigation-modal cases still fail; their suite mocks RunDetail.js and excludes the changed await component. Global UI branch coverage remains 83.15% versus the 85% gate; no pre-change aggregate baseline exists, so this is not declared unrelated or accepted. Native run 20261004-015342-4cba1984 remains FAIL.

The full contemporary API diagnostic captured four failing packages / 23 test fail events: 19 involve runtime.Caller root helpers under the native -trimpath command; two are the termination fixture parent/subtest (5 * 99 = 495 is below the current default stale threshold 1000); two workflow fixtures fail the current 1000000 micro-USD budget minimum before their intended conformance assertions. Those API/config/budget/fixture sources were not changed. These are reproduced current failures, not reconstructed original-run attribution. The original immutable API tail does not identify every original failed case. Exact receipts and bounded source assessment are in task-14/coordinated-recovery/source-review-notes.md. Distinct native review and original consumer recovery/ack remain OPEN; AUTH-02 is staged/unapplied and supplies no admission proof yet. Nooch normal-resume acknowledgement is unchanged.

#### BAS-SUP-LIVE-001 — live baseline build blocker (supervisor, 2026-10-04T17:04Z)

The supported `vrooli scenario start browser-automation-studio --timeout 1200 --json` retry failed before replacing the live instance. Current control-plane status remains `start-failed`, `API_PORT` unresolved and UI HTTP 503; the isolated shadow remains healthy on build `sha256:e16bbda1f5ed9d79d12ba591912b24b08962bc4c4fca8512238948b7fa304cbd`. The new live stage log, run `20261004-170358.188228928-2457510`, resolves the engaged live source to `~/.cache/vrooli/browser-automation-studio/baseline-bas-goal/restore-point` and fails its API Go build because relative replacements such as `../../../packages/api-core` and `../../../packages/proto` resolve under the cache, where the packages do not exist. This is a reproducible baseline snapshot/runtime-layout dependency failure, separate from the repaired shadow Vite alias. No force, cache edit, disengagement, or live fallback to candidate source was used.

The BAS orchestrator should preserve the frozen live baseline and route this exact control-plane Baseline Modes/lifecycle defect to its owning repair path. Qualify a dependency-complete, immutable live build location with a focused engaged-live regression before retrying the original live start; verify control-plane health and Nooch's actual current-build capture after recovery. Keep E22 R1, Nooch capture gates and receiver acknowledgement open until their own operations pass. This feedback reports a shared-tool blocker; it does not authorize a product epoch, baseline rewrite, or acceptance change.
