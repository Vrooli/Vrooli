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

### BAS-FB-056 — missing tools never hold an epoch (2026-09-30, operator)

> Missing tools never hold an epoch: record Gremlins as unverified, decide E2 on the passing coverage gates, and continue with the next slice.

Status: open. For the orchestrator: amend E2's mutation gate to unverified with
this entry as the recorded reason, decide E2 on its passing `go test`/coverage
gates, requeue further Go test maturation as a later slice, and admit S2.

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
| BAS-FB-055 | Translated into E0 directive D3 (two more worker runs, journey-ID slice-log prefixes); E0 accepted 2026-09-29. |
| BAS-FB-045/047/050/051 | Resolved by the epoch-orchestration plan: Luna-medium workers, infrequent Sol supervision, one orchestrator per goal, old supervision and qualification machinery retired (C1, C2). |
| BAS-FB-039–044, 046, 048, 049, 052–054 | Resolved before 2026-09-29; see the archive. |
