# Open feedback

No unacknowledged entries. Seq56 is archived after its same-task re-admission action and current reviewer verdict were reconciled in QUEUE.md and E1.md. Historical feedback and completed dispositions are preserved in [archive/FEEDBACK-history-through-20261020.md](archive/FEEDBACK-history-through-20261020.md).

### PP-SUP-AUDIT-LEGACY-MAPPING-ANSWER-SYNC-20261021 — Sync answered legacy mapping decision to current Planner queue (2026-10-21T00:04:25Z, supervisor)

> Supervisor bounded correction 2026-10-21 00:04Z. Native agent-manager effort decision list for scenarios/personal-planner/docs/internal/goal confirms PP-I18-LEGACY-WORKSPACE-MAPPING-20261020 state=answered, answer=separate_until_mapped, answeredVia=phone, answeredAt=2026-10-20T23:52:23.213527561Z, eventId de56f65daa246ea7713706200171e6520ed3f29e75b43ec360e383987d2941bc. QUEUE line40 incorrectly treats an earlier root pre-answer pending snapshot as overriding this later actual human answer and says not to sync signal57. Correct that stale statement on your same-parent check-in: use the native answered choice, keep ownerless/inaccessible legacy records separate until explicitly mapped, and do not fabricate mapping, expose them, or infer general E1/Q1 acceptance. Sole existing new-events worker task cbad6395-7d01-4c2f-8e2b-ad41de6a471f/run a7df133f-03ca-4217-b451-aa3dac6c8445 is RUNNING under your exact parent; do not cancel, duplicate, or direct it outside your own normal route. Your run 5d07e651-eaa6-48ec-89b4-5dc359059193 is PARKED(children) on key equal to your exact run ID. This accepted-effort signal permits neutral same-owner check-in; after reconciling the decision, repark the exact parent/key through the normal owner route if the child remains active. Do not use a guessed foreign continue/wake, fake child result, or new grant. Keep the Connect bounded result and other E1/Q1 gates distinct; record actual owner receipt and effect separately.

Recorded by `agent-manager effort feedback add`.

Status: reconciled — native decision was already answered and recorded; QUEUE/E1 updated, legacy rows remain unassigned.

### PP-I18-LEGACY-WORKSPACE-MAPPING-20261020 — operator decision (2026-10-20T23:52:23Z, operator — answered via phone)

Question: How should existing local Planner records with no verified owner key be treated before authenticated workspace access is enabled?

> separate_until_mapped — Keep legacy records separate until they are explicitly mapped

Offered: explicit_mapping | configured_owner | separate_until_mapped (recommended). Ask 5e4f2882-80dd-44ed-ba3d-23994bc07241. Recorded by `agent-manager effort decision sync`.

Status: reconciled — native phone answer `separate_until_mapped` recorded at 2026-10-21T00:07:23Z; no legacy rows assigned or exposed.

### PP-SDA-MODULE-FRESHNESS-OWNER-ROUTE-20261021 — Use the existing SDA module freshness repair for API build drift (2026-10-21T00:45:18Z, supervisor)

> Current infrastructure diagnosis: both worker9f84e827 and reviewerdb5a664a have returned; the reviewed sandbox lacked local replacement targets, while the current workspace has packages/api-core and packages/proto. Supported read-only scenario-dependency-analyzer freshness --module scenarios/personal-planner/api --json reports this exact module stale, diff_paths only scenarios/personal-planner/api/go.mod, next_action.code=apply_go_tidy and fixability=automatic, without downloads or new third-party dependency decision. Existing owning repair is scenario-dependency-analyzer freshness --module scenarios/personal-planner/api --apply. Use that supported owner route for this already-approved readiness repair, then validate API lifecycle build before the retained seven-phase gate. Do not run raw go mod tidy/get or invent another dependency consent. The printed next-action command currently drops the module value; root repairs that small SDA projection defect separately. Preserve other dirty source, actual dependency governance, native ownership and all full E1/Q1 gates; no product acceptance follows from tidy alone.

Recorded by `agent-manager effort feedback add`.

Status: reconciled — the current supported read-only check `scenario-dependency-analyzer freshness --module scenarios/personal-planner/api --json` reports the API module clean (1 checked, 1 clean, no stale/errors/downloads). Its current source hash differs from the older unchanged-hash receipt, so no further `--apply` is warranted. The admitted focused Test Genie `api` phase will verify current lifecycle readiness alongside the affected CLI contract/unit phases.

### PP-COHERENT-JOURNEY-REVIEW-BATCHING-20261021 — Finish user outcomes and batch dependent corrections (2026-10-21T01:34:45Z, supervisor)

> Owner priority: remove process overhead that displaces actual delivery. The queue records separate worker/reviewer/acceptance cycles for dependent target-date history/revision-guard work (+16 runtime/+48 test for the guard), while original E1 remains open. Canonical LEO4.1 requires coherent user-visible outcomes or whole replace-and-delete work; its review row says review material changes and reuse valid reviews, not a fresh reviewer task and epoch acceptance for every small handoff. Finish the active CLI correction without duplicate work. At the next normal return, group dependent remaining corrections and their affected validation into a meaningful end-to-end E1 outcome; retain author self-validation, source attribution, actual authority/isolation, independent material-change and full-epoch acceptance. Avoid treating a bounded task return as a new epoch or scheduling separate reviewer/acceptance ceremonies for each tiny correction. Preserve all original remaining gates, scope and honest limitations; no quality waiver or broad unchanged rerun. State the next user outcome and actual blocking fact in the current queue, keeping resolved slice history out of the active handoff.

Recorded by `agent-manager effort feedback add`.

Status: reconciled — the CLI manifest correction is attributed to its exact before/after GCT snapshots. The focused Test Genie run completed with `contracts` PASS and `unit` FAIL on the stale generated primitive-evidence artifact. Its terminal report and exact diagnosis are retained in QUEUE/E1 receipts; it will not be replayed. The admitted single-path artifact refresh is followed by only affected `contracts,api,unit` validation, one combined independent review reusing the existing actor-profile verdict, and one parent mechanical check. No unchanged phases are rerun, and full E1/Q1 acceptance remains open.
