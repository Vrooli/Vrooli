# LM6 Program Runtime handoff for desktop-startup

## Purpose

This is the integration contract between Program Runtime's staged `optimization.Adapter`
and scenario-to-desktop's `desktopoptimize` owner API. The package now exposes
staged `Measure`, `Promote`, and `Cleanup` operations, while retaining `Execute`
as an all-in-one wrapper. The package still has no registered live evaluator or
artifact lifecycle binding; injected owners must be supplied by the owning
scenario. This handoff is not evidence of a live measurement or promotion.

## Engine request mapping

Program Runtime pins one `desktop-startup` request with one candidate and its
predecessor. The target adapter maps it to `desktopoptimize.Request` as follows:

| Program Runtime field | DP1 field | Rule |
|---|---|---|
| `run_id` | `run_id` | Preserve byte-for-byte as the idempotency and journal key. |
| `target` | `target` | Must equal `desktop-startup`. |
| `subject.scenario` | `subject.scenario` | Exact scenario identity. |
| `subject.artifact_digest` | `subject.artifact_digest` | Must identify the predecessor/control artifact. |
| subject host/display/deployment/profiler identity | matching `subject` fields | All four required and immutable for the run. |
| sole candidate `id`, `artifact_digest`, `predecessor_digest` | `candidate` | Candidate predecessor digest must equal the subject artifact digest. |
| acquire stage's verified receipt for the sole candidate | `acquisition` | Include receipt ID, candidate ID, actual artifact digest, predecessor digest and canonical subject digest. Reject if any identity differs; Program Runtime must first compare any pinned expected digest to the acquired digest. |
| `suite_id` | `baseline_ref` | Stable reference to the desktop baseline cohort, not a live sensor. |
| `case_ids`, `minimum_cases` | `minimum_pairs` | `minimum_cases` must encode the six-cell minimum of 30 pairs; `minimum_pairs` may not be below five per cold/warm class. Case identities remain in the engine request digest. |
| requested startup settings | `candidate.startup_profile` | Opaque target-owned settings; never treated as host command authority. |

The DP1 evaluator receives `MinimumPlan` and must return all 30 pairs: five in
each cold/warm × three-phase cell. All phase rows for a pair share the same
control and candidate launch IDs. Control and candidate runs use the same host,
display, deployment mode and profiler mode. Every pair includes evidence
references and nonnegative durations. Unsupported phases remain unavailable and
fail closed; they cannot be removed from the denominator.

## Stage ownership and transaction boundary

The Program Runtime engine's DP1 sequence is preflight rights/disk admission,
`Acquire`, `Evaluate`, fresh rights/disk rechecks, `Promote`, and `Cleanup`.
The original DP1 `Execute`
transaction performed measurement, eligibility, promotion and predecessor
removal under one durable journal. It must not be called from the engine's
`Promote` stage after `Evaluate`: that would measure twice and remove the
predecessor before the engine records its cleanup stage.

The package now provides the stage seam directly:

- `Measure` collects and validates the matrix, then persists `measurement-complete`.
- `Promote` requires that passing receipt and persists `promotion-complete`; it
  never removes the predecessor.
- `Cleanup` requires the promotion receipt, removes the predecessor, and invokes
  rollback through its owner if removal fails.
- `Execute` calls these three operations in order for existing all-in-one callers.

Every request carries an `AcquisitionProof`. `ValidateRequest` matches its
receipt, candidate, actual artifact digest and predecessor digest against the
immutable target request before measurement. `Promote` revalidates the same
request and acquisition proof, so a candidate whose acquired digest differs
from the mapped digest cannot cross the promotion boundary.

Owner callbacks receive stable operation IDs derived from the pinned run ID:
`<run_id>/measure`, `<run_id>/promote`, `<run_id>/remove-predecessor`, and
`<run_id>/rollback`. They must return the original receipt when retried with
the same operation ID. Each stage intent is flushed before its owner callback;
owner errors leave that intent available for a same-ID retry. The integration
must persist each returned receipt in Program Runtime before advancing.

`Measure` requires the acquired candidate identity and current
`preflight_rights`/`preflight_disk` proofs. `Promote` and `Cleanup` require the
fresh post-evaluation `rights`/`disk` proofs. The matching
`preflight_proof_binding_digest` and `proof_binding_digest` are carried
unchanged from Program Runtime; the former binds the admission before Acquire,
the latter binds the actual acquired digest cohort. The journal digest pins all
target and acquisition fields while excluding gate proof bodies and binding
digests, which are rechecked at each stage. Legacy `Execute` callers may omit
separate preflight fields and use `rights`/`disk` for both admission checkpoints.

The production integration then binds these methods to scenario-to-desktop's
owner APIs:

1. **Staged owner port:** provide idempotent owner calls for paired measurement,
   promotion, and predecessor removal/rollback. `Evaluate` calls `Measure` and persists the complete DP1 matrix receipt; `Promote` verifies
   that exact matrix against the pinned request and engine rights/disk receipts,
   then calls the promotion owner; `Cleanup` calls the predecessor-removal owner
   and returns its durable receipt. Each call uses `run_id` plus a stable stage
   identifier. Promotion remains forbidden until the complete matrix passes
   all p95 no-regression checks and improves median first paint in at least one
   cohort. Cleanup failure invokes the owner rollback path and returns both
   receipts.
2. **Aggregate owner transaction:** retain `Execute` as the sole DP1 transaction
   owner and have Program Runtime add a single target-level operation contract
   whose receipt advances its durable engine record only after `Execute` reaches
   a terminal receipt. This requires an explicit engine contract change; do not
   make a staged adapter pretend the aggregate transaction is one stage.

No integration may invoke build, launch, promotion, cleanup or rollback through
a private shell command. The implementation must bind callbacks to the
scenario-to-desktop owners that own artifact and launch state.

## Proof binding and recovery

Program Runtime's canonical proof digest covers its pinned target, subject,
candidate cohort (including verified actual acquisition digests), suite and
cases. DP1's default gate digest covers only its `Identity`. The request carries
`preflight_proof_binding_digest` for pre-acquisition rights/disk proofs and
`proof_binding_digest` for the fresh post-acquisition proofs. Each pair must
match the corresponding Program Runtime canonical `SubjectDigest` and is passed
through unchanged. Program Runtime itself must verify both proof sets against
their canonical digests before calling DP1. The adapter must not substitute the
narrower DP1 digest or silently translate one digest into the other. This
preserves the engine's stronger binding to the candidate, suite, cases, and
acquired artifact. LM5 and capacity remain the sources of the proof receipts;
the target adapter only verifies and carries their digests.

The proofs must be current through both promotion and cleanup;
the disk lease must cover the candidate and rollback headroom. A gate proof is
not inferred from adapter configuration or from a successful evaluator call.

Persist the engine `run_id`, request digest, and each owner receipt before
advancing the next stage. On retry, owners return the same receipt for the same
run/stage identity. If the DP1 journal reports an ambiguous nonterminal stage,
surface `recovery-required` and reconcile with the original owner operation;
never replay a possibly completed promotion or deletion. Preserve the complete
matrix and its evidence references in the terminal Program Runtime record.

## Acceptance evidence

Source tests may use synthetic matrices to check exact denominator, refusal,
identity binding and recovery. They cannot establish a performance result. A
showcase requires a live scenario-to-desktop owner evaluator, actual comparable
control/candidate launch evidence, current LM5 and disk receipts, promotion and
predecessor-removal owner receipts, and successful recovery behavior. The
historical `performance-baseline.json` is not a paired DP1 matrix or a live
budget.
