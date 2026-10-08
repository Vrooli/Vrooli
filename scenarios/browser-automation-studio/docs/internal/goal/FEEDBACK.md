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

Full text of every entry through 2026-10-06 is in
`archive/FEEDBACK-through-2026-10-06.md`.

## Open directives

### BAS-FB-064 — accept E26, re-aim at quality (2026-10-07, operator)

> Those recommendations sound good for addressing immediate issues.

The operator approved these points:
- **E26 gate amendment.** Generated `ui/dist` identity and pre-write material
  baselines are not exit gates. Accept E26 on its brief's source deletion
  list, focused API/UI checks, 12/12 journeys and inventory (−801). Do not
  reconstruct generated output.
- **The line-count phase is over.** GOAL.md now states the quality
  destination: 24 journeys, E22 R1/R2, recording reliability, macOS. Rewrite the
  Forecast for it. Plan journey slices first, because the 12 missing journeys
  are measurable and admissible now.
- **Gates.** Exit gates list only tests, journeys, inventory bounds and named
  reviews. Do not add identity, provenance or baseline-capture gates to briefs.
- **Handoff.** Keep one short `## Handoff` that you replace each time.
  Move the current long handoff's census notes to `archive/`.

Status: open. For the orchestrator: accept E26, rewrite the Forecast and
admit the first journey slice.

### BAS-FB-063 — redesign slices close the gap (2026-10-06, operator)

> Let's go with your recommendations.

The operator accepted these recommendations:
- Do not retire Record mode or Export. Remove the Retirement candidates list.
- Close the 205k gap with redesign slices: one simpler owner replaces several
  existing owners for the same behavior, with the replaced owners as the
  deletion list (`large-effort-orchestration` §2.1). Start with Workflow (RD-WF),
  then Recording UI and Export.
- Track E22 R1/R2, DET-SIGNIN-R1 and the live-BAS start under `## Waiting on
  others` in `QUEUE.md`. They never hold acceptance or other slices.
- Use one replaced `## Handoff`. A timer wake that finds no change writes
  nothing. When every slice truly waits on the operator, write `## Needs
  operator`, wake the supervisor once and park for 72h.
- The live-BAS outage (BAS-SUP-LIVE-001) is resolved; see Resolved.

Status: open. For the orchestrator: admit RD-WF as E23 now.

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
| BAS-SUP-LIVE-001 | 2026-10-06, operator session. The control plane now builds engaged live from a serving tree with the repo layout (`internal/baselinefloor/servingtree.go`). The frozen Sep 30 BAS no longer compiled against the current proto, so the operator approved promoting `bas-goal` to live (healthy, smoke passed) and a new `bas-goal` shadow was started right away. Live now runs the E22 code. |
| BAS-SUP-AUDIT-002 | E22 reopened, D2 acknowledged by successor 98268861, review corrected append-only, renewed acceptance 2026-10-06 with R1/R2 moved to Waiting on others. |
| BAS-FB-062 | Ordered DIAG-REDACT, REC-1A, REC-FIX (all accepted). Retirement candidates were listed; FB-063 declined retiring Record/Export. |
| BAS-FB-061 | DIAG-REDACT accepted as E20 and REC-FIX as E22. |
| BAS-FB-060, BAS-SUP-001 | Forecast kept at the top of QUEUE. The gap is now addressed by FB-063 redesign slices. |
| BAS-FB-057–059 | S2 remainder (E7), S3 at its scanned scope (E6), journey widening (E9). |
| BAS-FB-056 | Gremlins recorded unverified under operator direction; E2 accepted on passing Go/coverage gates, further test maturation continued as E4. |
| BAS-FB-055 | Translated into E0 directive D3 (two more worker runs, journey-ID slice-log prefixes); E0 accepted 2026-09-29. |
| BAS-FB-045/047/050/051 | Resolved by the epoch-orchestration plan: Luna-medium workers, infrequent Sol supervision, one orchestrator per goal, old supervision and qualification machinery retired (C1, C2). |
| BAS-FB-039–044, 046, 048, 049, 052–054 | Resolved before 2026-09-29; see the archive. |
