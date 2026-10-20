# Independent E1 judgment — 2026-10-19

## Verdict

**NOT ACCEPTED — original E1 / Q1 acceptance is not demonstrated by the retained evidence.** This is an independent criterion review receipt only. It is not a native review approval/rejection, does not accept E1 or Q1, and does not change any existing review or release state.

## Scope and method

Read the original `CONTRACT.md`, `ACCEPTANCE.md`, `epochs/E1.md`, the current `GOAL.md`, `QUEUE.md`, `TARGETS.md`, `FEEDBACK.md`, the E1 visual receipt/comparison and its DPR follow-up, and the available retained source evidence. Reviewed the recorded results for:

- seven-phase run `20261019-070609-a2a47270` (pre-fix only);
- post-fix unit run `20261019-071151-de6b7922` and the focused 21-fixture close/reopen regression;
- ordinary UI run `120cfffc-a761-458a-aabd-9e203cd99f2e`;
- isolated seed apply/cleanup `46cdbec3-73f2-463f-bf0e-c41f186a680e`;
- visual runs `e73c0ca8-e1b5-44ba-8d6c-0ee57ecec4b9` and `2d152248-1463-4b27-9ab0-c3970fca8630`, their comparison and retained captures;
- existing E1 lifecycle, Connect/parity and isolation evidence and the preserved limits in the E1 ledger.

No tests, builds, generators, lifecycle operations, seed operations, UI cases, native review actions, or product/evidence mutations were performed. The only write in this review was this new receipt.

## Original E1 criteria

| Original criterion | Judgment | Evidence and limit |
|---|---|---|
| Committed Proto generation verification before consumer validation | **PASS** | The October 19 ledger records `make verify-committed-gen` passing before the seven-phase admission and again before later routed work. This criterion alone does not establish the post-fix mechanical gate. |
| Required seven-phase Test Genie gate for the final admitted source | **NOT MET** | `20261019-070609-a2a47270` passed 7/7 for the source admitted to that run, but it predates the completion-history correction and the 21-fixture lifecycle extension. `20261019-071151-de6b7922` is a post-fix unit-only run (1/1). The listed evidence contains no post-fix seven-phase result. Do not extend the earlier PASS to changed source. |
| Ordinary routed calendar journey, durable identity/revision and recovery | **PARTIAL PASS** | Ordinary UI run `120cfffc-a761-458a-aabd-9e203cd99f2e` passed the registered desktop and mobile keyboard cases on the named shadow build and lease; timelines record multiday and DST-fold create/edit/reopen revisions and the DST instant. Separate seed run `46cdbec3-73f2-463f-bf0e-c41f186a680e` reports supported CLI create/idempotency/get/revision-checked update/list and independent API reads for all-day and DST-fold events, then exact-token cleanup. Earlier retained routed journeys cover duplicate submission, stale conflict, and lost-response recovery; the later F6 aggregate is recorded as passing. These do not establish one current ordinary browser + supported CLI + API journey with the same exact durable IDs/readbacks: seed IDs are not exposed, and the ordinary UI run did not execute seed apply. |
| All 21 calendar fixture identities and required civil-time/provider semantics | **PASS for the focused fixture criterion** | The post-fix isolated SQLite regression created, edited, closed/reopened, and read all 21 accepted identities by durable ID, checking revision, provider identity, civil date, subject and availability preservation. The recorded assertions cover the fixture-based calendar semantics. This focused result is not the full mechanical gate or a live-provider test. |
| Isolated storage and disposable backup/restore before product-data writes | **PASS for the recorded prerequisite** | E1 history records an active SQLite backup and a disposable scratch restore with `RESTORE_STATUS_VERIFIED`; later event work used routed synthetic storage and reports no product/primary writes. The filesystem readiness caveat remains in history; it is not evidence that the verified DB restore failed. |
| UI / supported CLI / Connect owner, profile, endpoint, source and event-revision parity | **UNVERIFIED as an integrated current-runtime criterion** | Existing focused routed integration evidence demonstrates REST-shaped UI, an actual CLI process and generated Connect calls sharing a synthetic routed database, event/profile identity and revisions, with a synthetic authenticated subject. Current ordinary UI and seed CLI/API runs are separate and do not expose common event IDs or establish a single current serving-runtime readback across those surfaces. Actor-to-workspace ownership also remains unproven. Do not claim stronger parity than these receipts show. |
| Synthetic-only routing and numeric per-execution isolation evidence | **PARTIAL; retained limits remain** | Runs `120cfffc`, `e73c0ca8`, and `2d152248` report run-level routed lease counters with zero primary requests/writes and zero test-root writes. Seed cleanup's fail-closed path supplies its available zero-primary evidence, but the seed CLI does not expose numeric counters. Earlier D2/D6 per-case LeaseStats are explicitly UNVERIFIED. Aggregate/run counters do not fill that per-case retention gap. |
| F1 Plan/Month visual references and provenance | **PASS for the bounded original F1 gate** | The worker's six-original access acknowledgement is in the October 3 E1 ledger. The October 19 comparison records desktop/narrow Day/Night captures against the applicable six plates and describes scene, controls, calendar readability and responsive composition. The DPR follow-up records actual browser `devicePixelRatio: 2` for Day and Night at 1280×800 and 390×844 CSS viewports, separately from 2× output dimensions; run/build/routes/state, empty lease state, artifact identities and hashes are retained. The current captures were inspected. No original criterion requires a seeded visual lease, clean PNG, or recovery of the October 3 artifacts; those are not added as gates here. |
| Original accepted integrated planning journey and known correctness findings in `ACCEPTANCE.md` / `CONTRACT.md` | **NOT MET / NOT DEMONSTRATED** | The post-fix focused unit result addresses the reproduced completion case where completing 60 original / 30 remaining with no recorded actual invented 30 minutes. The listed evidence does not show the required full goal → project/milestone/work/deadline → proposal/acceptance → actual/remaining/completion → forecast → review/reopen journey with exact durable readbacks, nor current retests establishing PP-UX01–08 and the known Countdown, placement, populated milestone listing and reflection behaviors. The other preserved reproduced findings also remain without qualifying resolution here: split actuals reported as −50% / two samples; an unfinished Nov 30 goal reported 58 days early; and a 600-minute workload against 840 occupied minutes reported zero availability together with on-track/Nov 2 finishes. Do not infer those cases fixed from the completion-only unit run or the seven-phase pre-fix run. |

## Exact unmet original gates

1. A seven-phase mechanical result for the source after the completion-history correction and fixture extension.
2. The original ordinary, durable cross-surface journey/readback criterion is not fully evidenced across one current UI + supported CLI/API execution with exact shared IDs/revisions; existing isolated seed IDs are unavailable in retained output.
3. Current integrated UI/CLI/Connect owner/profile/endpoint/source/event revision parity and actor-to-workspace ownership remain unverified at the required scope.
4. The accepted integrated planning journey and required retained PP-UX/domain correctness outcomes have not been demonstrated. The completion-without-actual defect has focused post-fix unit evidence only; the other three reproduced contract defects and listed UI behaviors have no qualifying passing evidence in this packet.
5. The previously amended per-case numeric LeaseStats subcheck remains explicitly unverified where receipts lack per-case counters; run-level counters must not be substituted for it.

F1 is not included among the unmet gates: the actual-DPR follow-up supplies the only missing provenance field identified in the prior F1 comparison. No extra PNG-quality, seeded-visual, historical-artifact-recovery, device, provider-write, or P1 performance gate is introduced.

## Preserved independent states

- Current-source run `dbd705d7-e563-42c1-90d2-a1567d0557d7` remains `NEEDS_REVIEW` / `APPROVAL_PENDING`; this receipt takes no action on it.
- Original F8 remains `NEEDS_REWORK`; the historical 78-file attribution remains missing. Current-source inventories are not historical attribution.
- No E1/Q1 acceptance, native review disposition, or F8 resolution is claimed.
- The October 3 visual artifact not-found remains a historical limit; current October 19 captures and comparison stand on their own.

