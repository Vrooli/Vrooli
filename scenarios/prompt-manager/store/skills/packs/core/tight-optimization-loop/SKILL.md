---
name: "tight-optimization-loop"
description: "Run unattended, evidence-bound optimization loops that compare candidates, enforce rights and capacity gates, promote a winner, remove its predecessor, and retain a durable receipt."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["optimization", "evaluation", "promotion", "capacity"]
  icon: "gauge"
  status: "active"
  revision: 1
  createdAt: "2026-11-14T00:00:00Z"
  updatedAt: "2026-11-14T00:00:00Z"
  requires:
    scenarios: ["program-runtime", "model-eval", "test-genie", "storage-manager"]
    commands: ["prompt-manager skill read", "vrooli scenario test"]
  origin:
    kind: "authored"
---
## Practice focus: Tight Optimization Loop

Drive a measurable target to a maintained winner through a bounded, unattended workflow. Keep the baseline comparable, the inputs immutable, every gate owner-backed, and promotion recoverable after interruption.

This skill supports the operator's stated intent: “we should be pointing a large amount of this two helping us increase our odds of success in the different opportunities that we are trying to pursue”. It applies to optimization work that can be expressed as a stable target adapter and a governed execution program.

### Scope

Use this method for model-role, retrieval, modality, desktop startup, and other repeatable performance or quality comparisons. Keep product acceptance criteria with the target owner. Use `local-experimentation` when a choice is expensive to reverse and needs controlled comparison; this skill governs the reusable unattended acquisition-to-cleanup lifecycle. Do not use this method to justify paid inference, unapproved terms, source-data deletion, or work beyond the active authority.

The LM6 target is a durable `optimize` program in Program Runtime with target adapters. The first adapter covers model roles; the second covers desktop startup and DP1. LM8, LM10, and LM11 route through that same engine. The complete destination and showcase live in `/home/matthalloran8/.vrooli/research/topics/local-model-performance/MISSION.md` under LM6. This skill supplies judgment; it does not claim that the program or either adapter has executed.

### 1. Define the comparison

Entry: the target owner has named the optimization objective, acceptance rule, valid operating context, and rollback boundary.

1. Select the owner adapter and identify the actual caller path. Reuse the existing governed `optimize` program when available. Do not build a private scheduler or a second orchestration path.
2. Freeze the baseline and candidate inputs by immutable identity: source revision, model or artifact digest, runtime/build configuration, evaluation set revision, and relevant machine/capacity conditions.
3. Bind each candidate to the same cases, metric definitions, repetitions, hardware placement, warm/cold state, and competing load. Record baseline measurements before acquisition or mutation.
4. State the acceptance rule before observing candidate results. Include quality or correctness floors and latency, startup, resource, disk, or other target metrics. Define tie and regression behavior.

For the LM6 model-role showcase, compare at least three eligible candidates across at least 20 representative cases. For desktop startup, use DP1's paired matrix; if DP1 is absent, the adapter must first establish its minimum paired matrix. These are LM6 destination requirements, not evidence that a run has occurred.

If the baseline cannot be reproduced under comparable conditions, stop candidate ranking. Repair the measurement path or mark the comparison blocked. Do not call an unmeasured candidate a winner.

### 2. Resolve owners and admit work

Before a long unattended run, identify who owns candidate acquisition, host/device capacity, model or artifact rights, evaluation, serving configuration, and predecessor cleanup. Route each operation through its owning scenario, resource, or control-plane interface. Read the current contracts and available program inputs before invocation; do not invent commands or imply missing capability exists.

Apply this admission table before acquisition and again before promotion:

| Evidence state | Decision |
| --- | --- |
| Rights record identifies the artifact, allowed use, obligations, source/revision, and any business-context trigger; obligations are met | Continue |
| Rights evidence is absent, ambiguous, or incompatible with the current use | Refuse that candidate; continue eligible candidates or mark the target blocked if none remain |
| Capacity owner reports sufficient disk and runtime headroom for acquisition, evaluation, rollback, and cleanup | Continue |
| Disk usage, required staging space, or cleanup/rollback reserve is unknown or below the owner-defined floor | Refuse acquisition or promotion; retain existing serving state |
| Resource contention has an owner-backed queue/admission decision | Follow that decision and preserve the job identity |
| Capacity or ownership is unknown | Mark blocked and resolve the owner path; do not infer spare capacity |

For local-model work, honor the mission's 120 GiB `/home` free-space floor. Below that floor, do not acquire models, run inference, restart services, use the GPU, or launch disk-heavy tests. Report blocked with the observed free space and the capacity owner to contact. Do not delete unrelated data to satisfy admission.

### 3. Run and recover the experiment

Entry: comparison contract, immutable inputs, owners, rights evidence, and capacity admission are recorded.

Run the governed program without an attending agent when the program and adapter expose the required bounded inputs and gates. Preserve a durable run identity and checkpoint candidate acquisition, source/build state, evaluation, gate results, promotion, and cleanup. On restart, attach to the existing operation or resume from its owner checkpoint. Never replay a completed acquisition, evaluation, promotion, or deletion because its receipt was not delivered.

The experiment must preserve candidate isolation and one-factor interpretation. Reject or rerun a result when an input digest changes, the baseline drifts, evaluation coverage is incomplete, a required owner is unavailable, or concurrent load breaks comparability. Keep a candidate's failed result; do not silently remove a case or alter acceptance after seeing scores.

After evaluation, classify the result:

| Result | Action |
| --- | --- |
| Candidate passes every predeclared acceptance gate and beats or ties the baseline under the declared rule | Prepare promotion |
| Candidate fails a required quality/correctness floor or regresses a protected metric | Reject candidate; preserve evidence and continue remaining eligible candidates |
| Measurement is incomplete, incomparable, or interrupted without a recoverable checkpoint | Mark unknown/blocked; repair or resume from the checkpoint, then evaluate again |
| No candidate passes | Retain incumbent, report no winner, and state which falsifiable gate failed |

Unknown is never a pass. A blocked run is distinct from a measured candidate failure and from an executed successful optimization.

### 4. Promote, remove predecessor, and receipt

Promotion is a recoverable state transition, not a final file copy. Recheck rights, capacity, acceptance, input digests, and serving/consumer references immediately before changing defaults. Use the target owner's promotion and rollback mechanism. Record the incumbent identity and a rollback window before switching.

Once the successor is verified on the real owner path and the rollback window has passed, remove the superseded weights, build, selector, or configuration through its owning cleanup path. Confirm that no consumers or rollback references still require it. Keep unique evaluation evidence and source content. If promotion fails, restore or retain the incumbent and leave the candidate available for diagnosis; do not delete either artifact.

Write one durable receipt beside the program's owner record. It must link the target and adapter, baseline and candidate identities, case-set/matrix revision, raw result locations, acceptance rule, rights evidence, disk/capacity decisions, run/checkpoint identity, promotion and live verification, rollback window, predecessor deletion evidence, and unresolved limits. The receipt must distinguish `ready`, `blocked`, `candidate rejected`, and `executed/promoted`; readiness or a passing dry gate is not execution evidence.

For LM6, one unattended command must eventually promote a `chat.default` winner from at least three candidates and 20 cases, delete the losing weights, and produce this receipt. The desktop adapter must do the same for DP1's paired startup matrix and predecessor. Until live evidence exists, report those outcomes as remaining, not completed.

### 5. Retain and reuse

Keep immutable inputs, raw measurements, gate decisions, and the receipt with their producer. Add a reusable case or repair to the owning corpus or adapter when evidence shows a stable gap. Capture only novel operational lessons in the appropriate memory/work log; do not create a parallel ledger.

When LM8, LM10, or LM11 needs a sweep, pass its target definition and owner-specific acceptance to the same `optimize` program. Add target-specific adapter logic only where the contract differs. Keep acquisition, scheduling, checkpoint/recovery, common gates, promotion, cleanup, and receipts in the shared engine. A repeated adapter workaround is evidence to extend the engine or the true owner, not to preserve duplicate orchestration.

### Troubleshooting & Edge Cases

| Condition | Response |
| --- | --- |
| `/home` is below 120 GiB free | Stop before acquisition, inference, restart, GPU use, or disk-heavy validation. Record blocked capacity evidence. |
| Candidate has no rights record or needs terms accepted on the operator's behalf | Refuse that candidate. Continue compliant candidates; do not pause the whole comparison if a valid path remains. |
| Capacity, disk reserve, consumer references, or checkpoint state is unknown | Preserve incumbent and candidate artifacts. Mark blocked until the owning record is available. |
| Process or agent restarts during a phase | Resume from the stable run identity and last owner checkpoint. Inspect receipts before repeating effects. |
| Candidate wins metrics but misses a required quality floor | Reject it. Acceptance is fixed before the run. |
| Promotion succeeded but predecessor cleanup is not proven | Report partial execution. Preserve receipt and rollback evidence; do not claim completion. |
| Target adapter is missing or does not expose paired immutable inputs | Build the minimum adapter contract in the engine's owner, then rerun through the governed program. |
| A common workflow step appears in another adapter or cohort | Reuse the same program step; do not add private scheduling or gate logic. |

### Anti-patterns

| Anti-Pattern | Why It Fails | Better Approach |
| --- | --- | --- |
| Tune from a moving or unmatched baseline | Scores cannot identify a candidate effect | Freeze paired inputs and operating conditions |
| Treat missing rights or disk evidence as permission | Acquisition can violate obligations or strand the host | Refuse at the owner-backed gate |
| Let an agent supervise every candidate manually | Human attention becomes the scheduler and runs do not recover cleanly | Use one bounded governed program and durable checkpoints |
| Report a ready adapter as a successful optimization | Planned interfaces do not prove live promotion or deletion | Separate readiness, blockage, and execution receipts |
| Delete the incumbent before successor verification and rollback expiry | A failed promotion can take down the target without recovery | Promote, verify, wait through rollback, then delete with receipt |
| Duplicate orchestration per model cohort | Recovery and safety behavior diverge | Route each cohort through the same `optimize` engine |

### Output expectations

Return the target, baseline comparability, immutable candidate inputs, acceptance rule, owner paths, rights and capacity decisions, execution state, winner or rejection evidence, promotion/deletion receipts, and remaining blocked or unverified obligations. Do not claim an LM6 showcase from skill registration, contract readiness, or synthetic evidence.
